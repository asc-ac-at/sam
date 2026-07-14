/*
Copyright © 2026 Adam McCartney <adam.mccartney@tuwien.ac.at>
*/
package build

import (
	"log/slog"

	"github.com/spf13/cobra"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/cvmfs"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/host_injections"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

func NewCommand(logger *slog.Logger) *cobra.Command {
	opts := shared.NewOptions()

	cmd := &cobra.Command{
		Use:   "build",
		Short: "",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		TraverseChildren: true,
	}

	shared.RegisterFlags(cmd, opts)

	cmd.AddCommand(cvmfs.NewCommand(opts, logger))
	cmd.AddCommand(host_injections.NewCommand(opts, logger))

	return cmd
}
