/*
Copyright © 2026 Adam McCartney <adam.mccartney@tuwien.ac.at>
*/
package cvmfs

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

var (
	ctrTool string
	publish bool
)

func NewCommand(opts *shared.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cvmfs",
		Short: "Build software to publish on cvmfs repository",
		Long: `Start a build container and build software inside.

Typically you run this command when you want to publish software to a
cvmfs repository. The configuration of the build environment is specified
by the container tool e.g: samctr.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.GitBranch != "" && opts.GitCommit != "" {
				return fmt.Errorf("--gitBranch and --gitCommit are mutually exclusive")
			}
			if opts.GitBranch != "" && opts.GitMergeReqID != 0 {
				return fmt.Errorf("--gitBranch and --gitMergeRequestId are mutually exclusive")
			}
			if opts.GitCommit != "" && opts.GitMergeReqID != 0 {
				return fmt.Errorf("--gitCommit and --gitMergeRequestId are mutually exclusive")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			shared.RegisterFlags(cmd, opts)
			data := NewCvmfsBuildCmdData(opts)
			publish, _ := cmd.Flags().GetBool("publish")
			data.Publish = publish

			err := renderBuildCmd(buildCmdTmpl, data)
			if err != nil {
				log.Println(fmt.Errorf("%w", err))
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&ctrTool, "tool", "t", "samctr", "Container tool used to run the build environment")
	cmd.Flags().BoolVarP(&publish, "publish", "p", false, "Publish the archive by sending to stratum0 for ingestion")

	return cmd
}
