// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

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

// objectInfo is the JSON row emitted by `object list --format json`.
// RFC3339 LastModified keeps the payload parseable by shelling tools
// (e.g. the auto-ingest pull cycle, sami TODO-2469a100).
type objectInfo struct {
	Key          string    `json:"Key"`
	Size         int64     `json:"Size"`
	LastModified time.Time `json:"LastModified"`
}

// newListObjectsCmd represents the object list command
func newListObjectsCmd(opts *Options, c *client.Client) *cobra.Command {
	var format string
	var listCmd = &cobra.Command{
		Use:   "list <bucket>",
		Short: "List objects in a bucket.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			objects, err := c.ListObjects(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if format == "json" {
				rows := []objectInfo{}
				for _, o := range objects {
					rows = append(rows, objectInfo{
						Key:          *o.Key,
						Size:         *o.Size,
						LastModified: *o.LastModified,
					})
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(rows)
			}
			for _, o := range objects {
				fmt.Println(*o.Key)
			}
			return nil
		},
	}
	listCmd.Flags().StringVar(&format, "format", "", "Output format: json")
	return listCmd
}
