// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
        "os"

        "github.com/spf13/cobra"
)



// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
        Use:   "samctr",
        Short: "A containerized build environment for CVMFS software repositories.",
        Long: `A containerized build environment for CVMFS software repositories.

This program wraps the Apptainer container runtime, exposing two
commands that facilitate running a container with a set of default
options. The core functionality of the commands is preserved - "shell"
launches an interactive shell session in the container, "exec" executes
a command in the container. The main focus of this program is to make it
easier to run these commands with the required confiration parameters
needed to mount a CVMFS repository in a writeable way using fusemounts.

Examples:
	samctr shell 
	samctr exec ls /etc`,
        // Uncomment the following line if your bare application
        // has an action associated with it:
        // Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
        err := RootCmd.Execute()
        if err != nil {
                os.Exit(1)
        }
}


func init() {
		registerFlags(RootCmd)
		cobra.OnInitialize(initConfig)
}
