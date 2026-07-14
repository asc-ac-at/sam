/*
Copyright © 2026 Adam McCartney <adam.mccartney@tuwien.ac.at>
*/
package cvmfs

import (
	"errors"
	"log/slog"

	"github.com/spf13/cobra"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
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
			if opts.GitBranch != "" && opts.GitMergeReqID != 0 {
				return errors.New("--gitBranch and --gitMergeRequestId are mutually exclusive")
			}
			if opts.GitCommit != "" && opts.GitMergeReqID != 0 {
				return errors.New("--gitCommit and --gitMergeRequestId are mutually exclusive")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			data := NewCvmfsBuildCmdData(opts)
			publish, _ := cmd.Flags().GetBool("publish")
			data.Publish = publish

			// 1. setup logging
			_, err := buildlog.NewBuildLogPaths(opts.BuildLogBasePath, opts.Name)
			if err != nil {
				return err
			}

			// 2. setup git
			// 3. render build cmd
			err = renderBuildCmd(buildCmdTmpl, data)
			if err != nil {
				return err
			}
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
