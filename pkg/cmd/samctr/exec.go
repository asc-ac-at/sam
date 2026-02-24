// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
	"log"
	"os"
	"strings"

	isamctr "github.com/asc-ac-at/sam/internal/samctr"
	"github.com/spf13/cobra"
)

func ApptainerExecArg(rs *RuntimeState) string {

	fusemounts := ""
	for _, fm := range rs.Storage.FuseMounts {
		fusemounts = fusemounts + fmt.Sprintf("--fusemount %s ", fm.FuseCmdForExec())
	}

	bindmounts := isamctr.BindMountsApptainerFmt(rs.AllBindMounts)
	extraOpts := strings.Join(rs.ApptainerCmdOpts, " ")

	prg := strings.Join(rs.ArgsAfterDash, " ")
	arg := fmt.Sprintf(`'apptainer exec %s %s %s %s %s'`, fusemounts, bindmounts, extraOpts, rs.ContainerSif, prg)

	return arg
}

// execCmd represents the exec command
var execCmd = &cobra.Command{
	Use:   "exec",
	Short: "Configure Apptainer exec command",
	Long: `Configure Apptainer exec command

This will set up a call to "apptainer exec" using values provided by the
config file and/or inputs using the cli flags. A typical use case for exec
is to run a software build from the context of at slurm job.

Examples:
	$ samctr exec -- ls /cvmfs
	
	$ cat hello_world.py | samctr exec -- python3

	$ samctr exec -- /bin/sh -<build_cmd.sh

	$ cat >build_go_1250.sh <<EOF
	#!/bin/sh

	export EESSI_PROJECT_INSTALL=/cvmfs/software.asc.ac.at
	source /cvmfs/software.eessi.io/versions/2023.06/init/lmod/bash
	source /cvmfs/software.eessi.io/versions/2023.06/init/bash
	module load EESSI-extend
	eb -r Go-1.25.0.eb
	EOF
	$ samctr exec -- /bin/sh <build_go_1250.sh


`,
	PreRunE: PrepareContainerPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		var argsAfterDash []string
		at := cmd.Flags().ArgsLenAtDash()

		if at > 0 {
			all := cmd.Flags().Args()
			// guard against out of range
			if at <= len(all) {
				argsAfterDash = all[at:]
			}
		}
		// Fallback: if the FlagSet approach didn't yield results, scan os.Args for "--".
		// This is robust across different parsing setups and when cobra/pflag behavior differs.
		if len(argsAfterDash) == 0 {
			for i, a := range os.Args {
				if a == "--" && i+1 <= len(os.Args) {
					argsAfterDash = os.Args[i+1:]
					break
				}
			}
		}
		log.Printf("args len at dash %d", len(argsAfterDash))
		Runtime.ArgsAfterDash = argsAfterDash

		if ToStdout {
			fmt.Printf("/bin/sh -c apptainer %s\n", ApptainerExecArg(Runtime))
			return
		} else {
			RunSystemShell(Runtime, ApptainerExecArg)
		}
	},
}

func init() {
	RootCmd.AddCommand(execCmd)
}
