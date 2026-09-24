// SPDX-License-Identifier: GPL-2.0
package samctr

import (
	"os"
	"time"

	"github.com/asc-ac-at/sam/pkg/subproc"
)

// newSystemShell prepares the /bin/sh -c <arg> config used by both the
// shell and exec subcommands. It must carry the host OS env plus any
// runtime additions -- notably APPTAINERENV_LD_LIBRARY_PATH, the --nv
// guard against apptainer injecting host lib dirs (/lib:/lib64) ahead of
// EESSI's compat-layer libs. subproc.Config{Env: nil} inherits only
// os.Environ, which silently dropped this guard and re-broke the EESSI
// init (lua5.1 -> GLIBC_2.38 not found) on GPU hosts.
func newSystemShell(rs *RuntimeState, arg string) *subproc.Config {
	cfg := subproc.New([]string{"/bin/sh", "-c", arg})
	cfg.Timeout = 72 * time.Hour
	cfg.Env = append(os.Environ(), rs.Environ...)
	return cfg
}
