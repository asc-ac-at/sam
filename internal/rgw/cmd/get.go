// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

// newGetCmd represents the object get command
func newGetCmd(opts *Options, c *client.Client) *cobra.Command {
	var getCmd = &cobra.Command{
		Use:   "get <bucket> <key> <file>",
		Short: "Download an object to a local file.",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.DownloadFile(cmd.Context(), args[0], args[1], args[2])
		},
	}
	return getCmd
}
