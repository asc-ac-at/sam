/*
Copyright © 2026 Adam McCartney
// SPDX-License-Identifier: MIT
*/
package sami

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/build"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sami",
	Short: "Software And Modules Installer",
	Long: `A tool for managing the installation of software and modules on HPC systems.

Created for use at the Austrian Scientific Computing (ASC) Research Center.`,
	TraverseChildren: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(opts *shared.Options, logger *slog.Logger) {
	rootCmd.AddCommand(build.NewCommand(opts, logger))

	err := rootCmd.Execute()
	if err != nil {
		logger.Error("execution failed", "error", err)
		os.Exit(1)
	}
}

func init() {
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sami.git.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	//rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
