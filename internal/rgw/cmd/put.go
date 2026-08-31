// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

// newPutCmd represents the object put command
func newPutCmd(opts *Options, c *client.Client) *cobra.Command {
	var putCmd = &cobra.Command{
		Use:   "put <bucket> <key> <file>",
		Short: "Upload a local file to an object.",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.UploadFile(cmd.Context(), args[0], args[1], args[2])
		},
	}
	return putCmd
}
