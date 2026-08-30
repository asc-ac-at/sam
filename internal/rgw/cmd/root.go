// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package cmd

import (
	"context"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/asc-ac-at/sam/internal/rgw/client"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rgw",
	Short: "Rados Gateway command line tool.",
	Long:  `Provides a minimal set of operations above the aws s3 sdk.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once.
func Execute() {
	opts := NewOptions()
	c, err := client.New(context.Background())
	if err != nil {
		log.Printf("failed to create client: %v", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(newBucketCmd(opts, c))
	rootCmd.AddCommand(newObjectCmd(opts, c))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
