// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	isamctr "github.com/asc-ac-at/sam/internal/samctr"
	"github.com/spf13/cobra"
)

var (
	execExample = `
	# run a program inside the container
	samctr exec -- ls /cvmfs

	# run a program inside an existing container (resume from tmpdir)
	samctr exec --resume /path/to/previous/build/ctr-tmp -- find /cvmfs -name "my-software"

	# run an arbitrary "build_cmd" script
	cat >build_cmd.sh <<EOF
	#!/bin/sh

	export EESSI_SITE_INSTALL_PREFIX=/cvmfs/software.asc.ac.at
	source /cvmfs/software.eessi.io/versions/2025.06/init/lmod/sh
	module load EESSI-extend
	eb -r <some-easyconfig>.eb
	EOF
	samctr exec -- /bin/sh <build_cmd.sh
`
)

func ApptainerExecArg(rs *RuntimeState) string {

	fusemounts := ""
	for _, fm := range rs.Storage.FuseMounts {
		fusemounts = fusemounts + fmt.Sprintf("--fusemount %s ", fm.FuseCmdForExec())
	}

	bindmounts := isamctr.BindMountsApptainerFmt(rs.AllBindMounts)
	extraOpts := strings.Join(rs.ApptainerCmdOpts, " ")

	prg := strings.Join(rs.ArgsAfterDash, " ")
	arg := fmt.Sprintf(`apptainer exec %s %s %s %s %s`, fusemounts, bindmounts, extraOpts, rs.ContainerSif, prg)

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
`,
	Example: execExample,
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
		slog.Debug("parsed args after dash", "count", len(argsAfterDash))
		Runtime.ArgsAfterDash = argsAfterDash

		execArg := ApptainerExecArg(Runtime)
		if ToStdout {
			fmt.Printf("/bin/sh -c %s\n", execArg)
			return
		} else {
			cfg := newSystemShell(Runtime, execArg)
			if err := cfg.Run(); err != nil {
				log.Fatalf(`/bin/sh -c %s failed: %q`, execArg, err)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(execCmd)
}
