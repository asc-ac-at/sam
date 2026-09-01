/*
Copyright © 2026 Adam McCartney <adam@mur.at>
*/
package sami

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "dev"

func printVersion() {
	fmt.Printf("sami version: %s\n", version)
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Version info for sami",
	Long:  `Release version of the sami binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		printVersion()
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
