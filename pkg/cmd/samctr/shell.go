// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	isamctr "github.com/asc-ac-at/sam/internal/samctr"
	"github.com/asc-ac-at/sam/pkg/subproc"
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
	arg := fmt.Sprintf(`apptainer shell %s %s %s %s`, fusemounts, bindmounts, extraOpts, rs.ContainerSif)
	slog.Debug("apptainer shell arg", "arg", arg)
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

		shellArg := ApptainerShellArg(Runtime)
		if ToStdout {
			fmt.Printf("/bin/sh -c %s\n", shellArg)
			return
		} else {
			cfg := subproc.New([]string{"/bin/sh", "-c", shellArg})
			cfg.Timeout = 72 * time.Hour
			if err := cfg.Run(); err != nil {
				log.Fatalf(`/bin/sh -c %s failed, %q`, shellArg, err)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(shellCmd)
}
