// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

// newListBucketsCmd represents the bucket list command
func newListBucketsCmd(opts *Options, c *client.Client) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List buckets in the store.",
		RunE: func(cmd *cobra.Command, args []string) error {
			buckets, err := c.ListBuckets(cmd.Context())
			if err != nil {
				return err
			}
			for _, b := range buckets {
				fmt.Println(*b.Name)
			}
			return nil
		},
	}
	return listCmd
}

// newListObjectsCmd represents the object list command
func newListObjectsCmd(opts *Options, c *client.Client) *cobra.Command {
	var listCmd = &cobra.Command{
		Use:   "list <bucket>",
		Short: "List objects in a bucket.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			objects, err := c.ListObjects(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			for _, o := range objects {
				fmt.Println(*o.Key)
			}
			return nil
		},
	}
	return listCmd
}
