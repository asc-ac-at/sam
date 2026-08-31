package build

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/config"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging/buildlog"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

// testLogPaths mirrors what RunE builds before calling runBackend; only
// BuildLog (sbatch --output) and BuildCmd (payload path) matter here.
func testLogPaths() *buildlog.BuildLogPaths {
	return &buildlog.BuildLogPaths{
		BuildLog: "/tmp/x",
		BuildCmd: "/tmp/x/build_cmd.sh",
	}
}

type fakeSubmitter struct {
	scripts [][]byte
	jobID   string
	err     error
}

func (f *fakeSubmitter) Submit(script []byte) (string, error) {
	f.scripts = append(f.scripts, script)
	if f.err != nil {
		return "", f.err
	}
	return f.jobID, nil
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
}

// Only the backend dispatch path is covered here — git setup etc. happens
// before this point in RunE.
func TestRunBackend_UnknownBackendErrors(t *testing.T) {
	opts := shared.NewOptions()
	opts.BuildBackend = "bogus"
	sub := &fakeSubmitter{}

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub); err == nil {
		t.Fatal("expected error for unknown backend")
	}
	if len(sub.scripts) != 0 {
		t.Errorf("fakeSubmitter should not be called on unknown backend, got %d calls", len(sub.scripts))
	}
}

func TestRunBackend_LocalSkipsSbatch(t *testing.T) {
	opts := shared.NewOptions()
	opts.BuildBackend = string(shared.BackendLocal)
	sub := &fakeSubmitter{}

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub); err != nil {
		t.Fatalf("local backend: %v", err)
	}
	if len(sub.scripts) != 0 {
		t.Errorf("local backend must not submit via sbatch, got %d submissions", len(sub.scripts))
	}
}

func TestRunBackend_Slurm_NoConfigRendersOnly(t *testing.T) {
	// ensure neither search location exists: point the user config dir and
	// the system path at empty tempdirs.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withMissingSystemConfig(t)

	opts := shared.NewOptions()
	opts.BuildBackend = string(shared.BackendSlurm)
	opts.Partition = "zen4_cpu"
	sub := &fakeSubmitter{}

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub); err != nil {
		t.Fatalf("missing config must be render-only, not an error: %v", err)
	}
	if len(sub.scripts) != 0 {
		t.Errorf("no config found: expected no submission, got %d", len(sub.scripts))
	}
}

func TestRunBackend_Slurm_SubmitsComposedScript(t *testing.T) {
	// write a config into the (tempdir-backed) user config location
	userCfg := filepath.Join(t.TempDir(), "sami", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(userCfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userCfg, []byte(testSbatchConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(filepath.Dir(userCfg)))
	withMissingSystemConfig(t)

	opts := shared.NewOptions()
	opts.BuildBackend = string(shared.BackendSlurm)
	opts.Partition = "testpart"
	sub := &fakeSubmitter{jobID: "4242"}

	bl := testLogPaths()
	if err := runBackend(opts, bl, newTestLogger(), sub); err != nil {
		t.Fatalf("slurm backend: %v", err)
	}
	if len(sub.scripts) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(sub.scripts))
	}
	s := string(sub.scripts[0])
	for _, want := range []string{
		"#!/usr/bin/env bash",
		"#SBATCH -p testpart",
		// job-name and output are rendered from code, not config; these
		// literal lines are the contract fixed by the header refactor.
		"#SBATCH --job-name=sami",
		"#SBATCH --output=/tmp/x/slurm-%j.out",
		"samctr exec \\\n    -- /bin/sh <" + bl.BuildCmd,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("submitted script missing %q\n got: %q", want, s)
		}
	}
}

// minimal config sufficient for the render path
const testSbatchConfig = `
sbatch-config:
  shared:
    header: "#sami --vanilla"
  partitions:
    testpart:
      qos: testqos
      ntasks: 8
      mem: 32G
      cpus-per-task: 1
      threads-per-core: 1
`

// withMissingSystemConfig redirects the exported config.SystemConfigPath
// var into an empty tempdir so tests don't depend on whether /etc/sami
// exists on the machine running them.
func withMissingSystemConfig(t *testing.T) {
	t.Helper()
	orig := config.SystemConfigPath
	config.SystemConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	t.Cleanup(func() { config.SystemConfigPath = orig })
}
