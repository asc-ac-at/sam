// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
/*
Copyright © 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"

	"github.com/spf13/cobra"
)

// this gets overwritten by the go build command
var version = "dev"

func printVersion() {
	fmt.Printf("samctr version: %s\n", version)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Version info for samctr",
	Long:  `Version info for samctr`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
