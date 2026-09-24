// SPDX-License-Identifier: GPL-2.0
package samctr

import (
	"strings"
	"testing"
	"time"
)

// Regression guard: runtime Environ additions (notably
// APPTAINERENV_LD_LIBRARY_PATH, the --nv host-libs guard) must reach the
// spawned /bin/sh env on every subproc-based entry path. subproc.Config
// with Env=nil inherits os.Environ only, which is how the guard was
// silently lost and `source .../init/lmod/sh` re-broke on GPU hosts
// (lua5.1 resolving host /lib64/libm.so.6, GLIBC_2.38 not found).
func TestNewSystemShell_CarriesRuntimeEnv(t *testing.T) {
	rs := &RuntimeState{
		Environ: []string{"APPTAINERENV_LD_LIBRARY_PATH=/.singularity.d/libs"},
	}
	cfg := newSystemShell(rs, "apptainer shell foo.sif")

	found := false
	for _, e := range cfg.Env {
		if e == "APPTAINERENV_LD_LIBRARY_PATH=/.singularity.d/libs" {
			found = true
		}
	}
	if !found {
		t.Fatalf("runtime Environ addition missing from subproc env: %v", cfg.Env)
	}
	if cfg.Timeout != 72*time.Hour {
		t.Errorf("shell/exec must keep the 72h watchdog timeout, got %v", cfg.Timeout)
	}
	if strings.Join(cfg.Args, " ") != "/bin/sh -c apptainer shell foo.sif" {
		t.Errorf("unexpected args: %v", cfg.Args)
	}
}
