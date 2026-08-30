// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import "github.com/spf13/cobra"

// Options is a mutable object containing data collected from user specified
// command line arguments.
type Options struct {
	// Region is the AWS region passed to commands that create or query the store.
	Region string
}

func NewOptions() *Options {
	return &Options{}
}

// RegisterFlags binds command line flags to the supplied Options and returns it.
func RegisterFlags(cmd *cobra.Command, opts *Options) *Options {
	cmd.Flags().StringVar(&opts.Region, "region", "", "AWS region")
	return opts
}
