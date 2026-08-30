// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

// newCreateCmd represents the bucket create command
func newCreateCmd(opts *Options, c *client.Client) *cobra.Command {
	var createCmd = &cobra.Command{
		Use:   "create <name>",
		Short: "Create a bucket in the store.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.CreateBucket(cmd.Context(), args[0], opts.Region)
		},
	}
	RegisterFlags(createCmd, opts)
	return createCmd
}
