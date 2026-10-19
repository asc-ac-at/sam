package build

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asc-ac-at/sam/internal/sami/cli/shared"
	"github.com/asc-ac-at/sam/internal/sami/config"
	"github.com/asc-ac-at/sam/internal/sami/logging/buildlog"
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

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub, false); err == nil {
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

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub, false); err != nil {
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

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub, false); err != nil {
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
	if err := runBackend(opts, bl, newTestLogger(), sub, false); err != nil {
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
		"samctr exec \\\n    --cleanenv \\\n    -- /bin/sh <" + bl.BuildCmd,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("submitted script missing %q\n got: %q", want, s)
		}
	}
}

// testKeksConfig extends the minimal config with the keks lane selected.
const testKeksConfig = testSbatchConfig + `
rgw:
  bucket: sam-archives
  credentials:
    backend: keks
`

// writeUserConfig stages a user-level sami config and points the loader at it
// (mirrors the render-path setup above; shared by the keks-lane tests).
func writeUserConfig(t *testing.T, content string) {
	t.Helper()
	userCfg := filepath.Join(t.TempDir(), "sami", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(userCfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userCfg, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(filepath.Dir(userCfg)))
}

// stubKeks installs a fake keks binary on PATH (sh script).
func stubKeks(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "keks")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestRunSlurmBackend_KeksLaneResolvesAtSubmit(t *testing.T) {
	writeUserConfig(t, testKeksConfig)
	withMissingSystemConfig(t)
	stubKeks(t, `echo "VALUE-$2"`)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	opts := shared.NewOptions()
	opts.BuildBackend = string(shared.BackendSlurm)
	opts.Partition = "testpart"
	sub := &fakeSubmitter{jobID: "4242"}

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub, true); err != nil {
		t.Fatalf("keks lane submit: %v", err)
	}
	assertEnv := func(name, frag string) {
		t.Helper()
		v := os.Getenv(name)
		if !strings.Contains(v, frag) {
			t.Errorf("%s = %q, want value resolved from the pair ref (%q)", name, v, frag)
		}
	}
	assertEnv("AWS_ACCESS_KEY_ID", "access_key")
	assertEnv("AWS_SECRET_ACCESS_KEY", "secret_key")
	if len(sub.scripts) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(sub.scripts))
	}
	// the script must still carry the template's presence gate + file fallback
	s := string(sub.scripts[0])
	if !strings.Contains(s, "elif [ -f \"${rgw_creds}\" ]") {
		t.Errorf("script lost the env-lane fallback\ngot:\n%s", s)
	}
}

func TestRunSlurmBackend_KeksLaneEnvWins(t *testing.T) {
	writeUserConfig(t, testKeksConfig)
	withMissingSystemConfig(t)
	stubKeks(t, `exit 1`) // would fail if consulted: prepared env must win
	t.Setenv("AWS_ACCESS_KEY_ID", "prepared-ak")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "prepared-sk")

	opts := shared.NewOptions()
	opts.BuildBackend = string(shared.BackendSlurm)
	opts.Partition = "testpart"
	sub := &fakeSubmitter{jobID: "4242"}

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub, true); err != nil {
		t.Fatalf("keks lane submit with prepared env: %v", err)
	}
	if got := os.Getenv("AWS_ACCESS_KEY_ID"); got != "prepared-ak" {
		t.Errorf("AWS_ACCESS_KEY_ID = %q, want the prepared value untouched", got)
	}
	if len(sub.scripts) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(sub.scripts))
	}
}

func TestRunSlurmBackend_KeksLaneDeadSessionRefusesEarly(t *testing.T) {
	writeUserConfig(t, testKeksConfig)
	withMissingSystemConfig(t)
	stubKeks(t, `echo "no vault token (VAULT_TOKEN / ~/.vault-token) - run \"papiere bitte\"" >&2
exit 1`)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	opts := shared.NewOptions()
	opts.BuildBackend = string(shared.BackendSlurm)
	opts.Partition = "testpart"
	sub := &fakeSubmitter{jobID: "4242"}

	err := runBackend(opts, testLogPaths(), newTestLogger(), sub, true)
	if err == nil {
		t.Fatal("expected keks-lane refusal, got nil")
	}
	if !strings.Contains(err.Error(), `run "papiere bitte"`) {
		t.Errorf("refusal lost keks' login hint: %v", err)
	}
	if len(sub.scripts) != 0 {
		t.Error("submit must not happen once the keks lane refused")
	}
}

func TestRunSlurmBackend_EnvLaneNeverConsultsKeks(t *testing.T) {
	writeUserConfig(t, testSbatchConfig) // no rgw section: default env lane
	withMissingSystemConfig(t)
	stubKeks(t, `exit 1`)

	opts := shared.NewOptions()
	opts.BuildBackend = string(shared.BackendSlurm)
	opts.Partition = "testpart"
	sub := &fakeSubmitter{jobID: "4242"}

	if err := runBackend(opts, testLogPaths(), newTestLogger(), sub, true); err != nil {
		t.Fatalf("env lane: %v", err)
	}
	if len(sub.scripts) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(sub.scripts))
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
