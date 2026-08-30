// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

// newObjectCmd represents the object command
func newObjectCmd(opts *Options, c *client.Client) *cobra.Command {
	var objectCmd = &cobra.Command{
		Use:   "object",
		Short: "Work with objects in the store.",
	}

	objectCmd.AddCommand(newPutCmd(opts, c))
	objectCmd.AddCommand(newGetCmd(opts, c))
	objectCmd.AddCommand(newDeleteObjectCmd(opts, c))
	objectCmd.AddCommand(newListObjectsCmd(opts, c))
	return objectCmd
}
