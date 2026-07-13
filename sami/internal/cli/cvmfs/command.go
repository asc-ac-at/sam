/*
Copyright © 2026 Adam McCartney <adam.mccartney@tuwien.ac.at>
*/
package cvmfs

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging"
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
			if opts.Name == "" {
				return fmt.Errorf("--name is required")
			}
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
			data := NewCvmfsBuildCmdData(opts)
			publish, _ := cmd.Flags().GetBool("publish")
			data.Publish = publish

			// 1. setup logging
			blp := logging.NewBuildLogPaths(opts.BuildLogBasePath, opts.Name)
			log.Printf("git repo: %s\n", blp.GitRepo)
			log.Printf("build cmd: %s\n", blp.BuildCmd)

			// 2. setup git
			// 3. render build cmd
			err := renderBuildCmd(buildCmdTmpl, data)
			if err != nil {
				log.Println(fmt.Errorf("%w", err))
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
