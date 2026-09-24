/*
Copyright © 2026 Adam McCartney <adam.mccartney@tuwien.ac.at>
*/
package build

import (
	"errors"
	"fmt"
	"log/slog"

	easybuild "github.com/asc-ac-at/sam/internal/sami"
	"github.com/asc-ac-at/sam/internal/sami/cli/shared"
	"github.com/asc-ac-at/sam/internal/sami/command/git"
	"github.com/asc-ac-at/sam/internal/sami/config"
	"github.com/asc-ac-at/sam/internal/sami/logging/buildlog"
	"github.com/asc-ac-at/sam/internal/sami/sbatch"
	"github.com/spf13/cobra"
)

var (
	buildExample = `
	# build for a zen4/h100 arch/accel combination using a specific slurm partition, using the most recently changed files in a gitlab merge request
	sami build --name <some-name> --arch zen4 --accel cc90 --build-backend slurm --partition <some-partition> --git-mr-id <N>

	# build the most recently changed file(s) from a git branch
	sami build --name <some-name> --arch zen4 --accel cc90 --build-backend slurm --partition <some-partition> --git-branch <remote-branch>

	# build using a specific file from a git commit
	sami build --name <some-name> --arch zen4 --accel cc90 --build-backend slurm --partition <some-partition> --git-commit <sha> --files easystack/2025.06/some-file.yaml

	# build and publish the tarball to rados-gateway
	sami build --name <some-name> --arch zen4 --accel cc90 --build-backend slurm --partition <some-partition> --git-mr-id <N> --publish

	# change the ownership of the files in the created tarball
	sami build --name <some-name> --arch zen4 --accel cc90 --build-backend slurm --partition <some-partition> --git-mr-id <N> --owner <some-user> --group <some-group>
	`
)

// validatePublish enforces the publish invariant: uploading the tarball to
// the radosgw bucket requires the bucket (and optionally endpoint) configured.
func validatePublish(cfg *config.File) error {
	if cfg.RGW.Bucket == "" {
		return errors.New("--publish requires an rgw.bucket entry in the sami config")
	}
	return nil
}

var (
	ctrTool   string
	publish   bool
	arch      string
	accel     string
	outputDir string
)

func NewCommand(opts *shared.Options, logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build software to publish on cvmfs repository",
		Long: `Start a build container and build software inside.

Typically you run this command when you want to publish software to a
cvmfs repository. The configuration of the build environment is specified
by the container tool e.g: samctr.`,
		Example: buildExample,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Name == "" {
				return errors.New("--name is required")
			}
			if opts.GitBranch != "" && opts.GitCommit != "" {
				return errors.New("--gitBranch and --gitCommit are mutually exclusive")
			}
			if opts.GitBranch != "" && opts.GitMergeReqId != 0 {
				return errors.New("--gitBranch and --gitMergeRequestId are mutually exclusive")
			}
			if opts.GitCommit != "" && opts.GitMergeReqId != 0 {
				return errors.New("--gitCommit and --gitMergeRequestId are mutually exclusive")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {

			// 0. possible coroutine implementation
			//    if by this point we can determine:
			//    user specified "--files"  OR git repo's "changed files",
			//    we could possible dispatch most of the following in the context of a coroutine.
			//    The result of that would be that we would not need to build each easystack file
			//    sequentially.
			//    + each easystack file gets it's own:
			//       - logdir
			//       - build_cmd.sh
			//       - slurm job
			//    Cost:
			//    + we need to determine if we rely on remote changes from git, if so, then we need
			//    to do a version of `git.GetChangedFiles(state, logger)` remotely
			//    The other (distinct) use case, is when we have multiple "accel" or
			//    "arch" values.

			// 1. setup logging
			blPath, err := buildlog.NewBuildLogPaths(opts.BuildLogBasePath, opts.Name)
			if err != nil {
				return err
			}

			// 2. git stuff (technical term)
			state, err := git.SetupGit(opts, blPath, logger)
			if err != nil {
				return err
			}

			state, err = git.GetChangedFiles(state, logger)
			if err != nil {
				return err
			}

			// 3.1 setup build cmd data
			data, err := NewCvmfsBuildCmdData(opts)
			if err != nil {
				return fmt.Errorf(`NewCmfsBuildCmdData(opts) failed with %w`, err)
			}
			publish, _ := cmd.Flags().GetBool("publish")
			data.Publish = publish

			// the per-run log dir is the default drop point for the tarball
			data.Logdir = blPath.BuildLog
			data.OutputDir = outputDir
			if data.OutputDir == "" {
				data.OutputDir = data.Logdir
			}

			// 3.1.1 configure subdirectories
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("publishing requires a sami config with arch-mapping: %w", err)
			}
			archSubdir, accelSubdir, err := resolveSubdirs(cfg, arch, accel)
			if err != nil {
				return err
			}
			data.ArchSubdir = archSubdir
			data.AccelSubdir = accelSubdir

			// when publishing, resolve the crtar subdirs now: the mapping
			// tables live in the sami config and are needed identically for
			// both the slurm and the local backend
			if data.Publish {
				if err := validatePublish(cfg); err != nil {
					return err
				}
				data.RGW = true
				data.RGWBucket = cfg.RGW.Bucket
				data.RGWEndpoint = cfg.RGW.Endpoint
			}

			// 3.2 render build cmd
			var fpaths []string
			if len(state.TargetFiles) > 0 {
				fpaths = git.AllTargetFilePaths(state)
			} else {
				fpaths = git.AllChangedFilePaths(state)
			}

			var estacks []*easybuild.Easystack
			for _, fpath := range fpaths {
				es, err := easybuild.NewEasystack(fpath)
				if err != nil {
					return fmt.Errorf(`easybuild.NewEasystack(%q) failed with %w`, fpath, err)
				}
				estacks = append(estacks, es)
			}
			data.Easystacks = estacks

			if err = renderBuildCmd(buildCmdTmpl, data, blPath.BuildCmd); err != nil {
				return err
			}
			logger.Debug(fmt.Sprintf("rendered build command to: %s", blPath.BuildCmd))

			// 4+5. select build backend and hand the rendered build to it
			return runBackend(opts, blPath, logger, sbatch.NewSbatchSubmitter(opts.SbatchFlags), publish)
		},
	}

	cmd.Flags().StringVarP(&ctrTool, "tool", "t", "samctr", "Container tool used to run the build environment")
	cmd.Flags().BoolVarP(&publish, "publish", "p", false, "Publish the archive by sending to stratum0 for ingestion")
	cmd.Flags().StringVar(&arch, "arch", "", "CPU architecture short name (e.g. zen4), resolved via arch-mapping in the sami config")
	cmd.Flags().StringVar(&accel, "accel", "", "Accelerator short name (e.g. cc90), resolved via accel-mapping in the sami config")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory where crtar writes the tarball (default: the per-run log directory)")

	shared.RegisterFlags(cmd, opts)

	return cmd
}
