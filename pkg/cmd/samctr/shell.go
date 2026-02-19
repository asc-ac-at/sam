// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
	"log"
	"strings"

	isamctr "github.com/asc-ac-at/sam/internal/samctr"
	"github.com/spf13/cobra"
)

func ApptainerShellArg(rs *RuntimeState) string {
	// We're going to construct a big 'ol string to pass to shell

	fusemounts := ""
	for _, fm := range rs.Storage.FuseMounts {
		fusemounts = fusemounts + fmt.Sprintf("--fusemount %s ", fm.FuseCmdForExec())
	}

	bindmounts := isamctr.BindMountsApptainerFmt(rs.AllBindMounts)
	extraOpts := strings.Join(rs.ApptainerCmdOpts, " ")
	arg := fmt.Sprintf(`'apptainer shell %s %s %s %s'`, fusemounts, bindmounts, extraOpts, rs.ContainerSif)
	log.Printf("AtpptainerShellArg: %s", arg)
	return arg
}

// shellCmd represents the shell command
var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Configure Apptainer shell",
	Long: `Configure Apptainer shell.

This will prepare a command to execute Apptainer shell with the desired configuration.`,
	PreRunE: PrepareContainerPreRun,
	Run: func(cmd *cobra.Command, args []string) {

		if ToStdout {
			fmt.Printf("/bin/sh -c %s\n", ApptainerShellArg(Runtime))
			return
		} else {
			RunSystemShell(Runtime, ApptainerShellArg)
		}
	},
}

func init() {
	RootCmd.AddCommand(shellCmd)
}
