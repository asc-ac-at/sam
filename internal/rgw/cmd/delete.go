// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

// newDeleteBucketCmd represents the bucket delete command
func newDeleteBucketCmd(opts *Options, c *client.Client) *cobra.Command {
	var deleteCmd = &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a bucket from the store.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.DeleteBucket(cmd.Context(), args[0])
		},
	}
	return deleteCmd
}

// newDeleteObjectCmd represents the object delete command
func newDeleteObjectCmd(opts *Options, c *client.Client) *cobra.Command {
	var deleteCmd = &cobra.Command{
		Use:   "delete <bucket> <key>",
		Short: "Delete an object from a bucket.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.DeleteObject(cmd.Context(), args[0], args[1])
		},
	}
	return deleteCmd
}
