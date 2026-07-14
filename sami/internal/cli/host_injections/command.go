/*
Copyright © 2026 Adam McCartney <adam.mccartney@tuwien.ac.at>
*/
package host_injections

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

func NewCommand(opts *shared.Options, logger *slog.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hostInjections",
		Short: "Build software for site specific install",
		Long: `This will build and optionally install software into 'host_injections' path.

host_injections is a special location defined during the configuration
of a client to use EESSI. Typically, you run this command on a compute
node of a HPC system to install software specific to that site.

More info: https://www.eessi.io/docs/site_specific_config/host_injections/#the-host_injections-variant-symlink
`,
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
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("hostInjections called")
		},
	}
	return cmd
}
