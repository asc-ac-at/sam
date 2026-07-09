/*
Copyright © 2026 Adam McCartney <adam.mccartney@tuwien.ac.at>
*/
package cvmfs

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cvmfs",
		Short: "Build software to publish on cvmfs repository",
		Long: `Start a build container and build software inside.

Typically you run this command when you want to publish software to a
cvmfs repository. The configuration of the build environment is specified
by the container tool e.g: samctr.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			buildCmdData, err := registerFlags(cmd)
			if err != nil {
				log.Println("error preprocessing")
				return err
			}
			err = renderBuildCmd(buildCmdTmpl, buildCmdData)
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
