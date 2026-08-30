// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

// newBucketCmd represents the bucket command
func newBucketCmd(opts *Options, c *client.Client) *cobra.Command {
	var bucketCmd = &cobra.Command{
		Use:   "bucket",
		Short: "Work with buckets in the store.",
	}

	bucketCmd.AddCommand(newCreateCmd(opts, c))
	bucketCmd.AddCommand(newDeleteBucketCmd(opts, c))
	bucketCmd.AddCommand(newListBucketsCmd(opts, c))
	return bucketCmd
}
