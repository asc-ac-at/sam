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
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging/buildlog"
)

var (
	ctrTool string
	publish bool
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
			data := NewCvmfsBuildCmdData(opts)
			publish, _ := cmd.Flags().GetBool("publish")
			data.Publish = publish

			// 1. setup logging
			bldPath, err := buildlog.NewBuildLogPaths(opts.BuildLogBasePath, opts.Name)
			if err != nil {
				return err
			}

			// 2. git stuff (technical term)
			state, err := git.SetupGit(opts, bldPath, logger)
			if err != nil {
				return err
			}
			state, err = git.GetChangedFiles(state, logger)
			if err != nil {
				return err
			}

			// 3. render build cmd
			data.Easystacks = git.AllChangedFilePaths(state)
			if err = renderBuildCmd(buildCmdTmpl, data, bldPath.BuildCmd); err != nil {
				return err
			}
			logger.Debug(fmt.Sprintf("rendered build command to: %s", bldPath.BuildCmd))
			// 4. select build runner
			// 5. run build (hand off to runner)

			return nil
		},
	}

	cmd.Flags().StringVarP(&ctrTool, "tool", "t", "samctr", "Container tool used to run the build environment")
	cmd.Flags().BoolVarP(&publish, "publish", "p", false, "Publish the archive by sending to stratum0 for ingestion")

	shared.RegisterFlags(cmd, opts)

	return cmd
}
