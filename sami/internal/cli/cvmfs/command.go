/*
Copyright © 2026 Adam McCartney <adam.mccartney@tuwien.ac.at>
*/
package cvmfs

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command/git"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/config"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging/buildlog"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/sbatch"
)

var (
	ctrTool   string
	publish   bool
	arch      string
	accel     string
	outputDir string
	rgwUpload bool
)

func NewCommand(opts *shared.Options, logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cvmfs",
		Short: "Build software to publish on cvmfs repository",
		Long: `Start a build container and build software inside.

Typically you run this command when you want to publish software to a
cvmfs repository. The configuration of the build environment is specified
by the container tool e.g: samctr.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if rgwUpload && !publish {
				return errors.New("--rgw requires --publish")
			}
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
			data := NewCvmfsBuildCmdData(opts)
			publish, _ := cmd.Flags().GetBool("publish")
			data.Publish = publish

			// the per-run log dir is the default drop point for the tarball
			data.Logdir = blPath.BuildLog
			data.OutputDir = outputDir
			if data.OutputDir == "" {
				data.OutputDir = data.Logdir
			}

			// when publishing, resolve the crtar subdirs now: the mapping
			// tables live in the sami config and are needed identically for
			// both the slurm and the local backend
			if data.Publish {
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

				if rgwUpload {
					if cfg.RGW.Bucket == "" {
						return errors.New("--rgw requires an rgw.bucket entry in the sami config")
					}
					data.RGW = true
					data.RGWBucket = cfg.RGW.Bucket
					data.RGWEndpoint = cfg.RGW.Endpoint
				}
			}

			// 3.2 render build cmd
			if len(state.TargetFiles) > 0 {
				data.Easystacks = git.AllTargetFilePaths(state)
			} else {
				data.Easystacks = git.AllChangedFilePaths(state)
			}

			if err = renderBuildCmd(buildCmdTmpl, data, blPath.BuildCmd); err != nil {
				return err
			}
			logger.Debug(fmt.Sprintf("rendered build command to: %s", blPath.BuildCmd))

			// 4+5. select build backend and hand the rendered build to it
			return runBackend(opts, blPath, logger, sbatch.NewSbatchSubmitter())
		},
	}

	cmd.Flags().StringVarP(&ctrTool, "tool", "t", "samctr", "Container tool used to run the build environment")
	cmd.Flags().BoolVarP(&publish, "publish", "p", false, "Publish the archive by sending to stratum0 for ingestion")
	cmd.Flags().StringVar(&arch, "arch", "", "CPU architecture short name (e.g. zen4), resolved via arch-mapping in the sami config")
	cmd.Flags().StringVar(&accel, "accel", "", "Accelerator short name (e.g. cc90), resolved via accel-mapping in the sami config")
	cmd.Flags().BoolVar(&rgwUpload, "rgw", false, "Upload the tarball to the radosgw bucket configured in the sami config (requires --publish)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory where crtar writes the tarball (default: the per-run log directory)")

	shared.RegisterFlags(cmd, opts)

	return cmd
}
