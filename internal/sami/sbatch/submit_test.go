package sbatch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeSbatch writes an executable stand-in for the sbatch binary into a
// tempdir and returns its path. Tests install it as Cfg.Args[0].
func fakeSbatch(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "sbatch")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNewSbatchSubmitter_DefaultArgs(t *testing.T) {
	sub := NewSbatchSubmitter(nil)

	want := []string{"sbatch", "--parsable"}
	if strings.Join(sub.Cfg.Args, " ") != strings.Join(want, " ") {
		t.Errorf("args: got %v, want %v", sub.Cfg.Args, want)
	}
	if sub.Cfg.Timeout != defaultSubmitTimeout {
		t.Errorf("timeout: got %v, want %v", sub.Cfg.Timeout, defaultSubmitTimeout)
	}
}

func TestNewSbatchSubmitter_AppendsFlagsInOrder(t *testing.T) {
	flags := []string{"--dependency=afterok:123", "--hold"}
	sub := NewSbatchSubmitter(flags)

	want := []string{"sbatch", "--parsable", "--dependency=afterok:123", "--hold"}
	if strings.Join(sub.Cfg.Args, " ") != strings.Join(want, " ") {
		t.Errorf("args: got %v, want %v", sub.Cfg.Args, want)
	}
	// the constructor must not mutate the caller's slice
	if len(flags) != 2 || flags[0] != "--dependency=afterok:123" || flags[1] != "--hold" {
		t.Errorf("input slice mutated: %v", flags)
	}
}

func TestNewSbatchSubmitter_EmptyFlagsSameAsNil(t *testing.T) {
	nilSub := NewSbatchSubmitter(nil)
	emptySub := NewSbatchSubmitter([]string{})
	if strings.Join(nilSub.Cfg.Args, " ") != strings.Join(emptySub.Cfg.Args, " ") {
		t.Errorf("nil vs empty flags differ: %v vs %v", nilSub.Cfg.Args, emptySub.Cfg.Args)
	}
}

func TestSubmit_ReturnsParsableJobID(t *testing.T) {
	capFile := filepath.Join(t.TempDir(), "stdin")
	t.Setenv("SBATCH_TEST_STDIN_CAPTURE", capFile)

	sub := NewSbatchSubmitter([]string{"--dependency=afterok:123"})
	sub.Cfg.Args[0] = fakeSbatch(t, `cat > "$SBATCH_TEST_STDIN_CAPTURE"; echo 4242`)

	payload := []byte("#!/usr/bin/env bash\necho build\n")
	jobID, err := sub.Submit(payload)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if jobID != "4242" {
		t.Errorf("jobID: got %q, want %q", jobID, "4242")
	}

	got, err := os.ReadFile(capFile)
	if err != nil {
		t.Fatalf("reading captured stdin: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("sbatch stdin: got %q, want %q", got, payload)
	}
}

func TestSubmit_TrimsWhitespaceAroundJobID(t *testing.T) {
	sub := NewSbatchSubmitter(nil)
	sub.Cfg.Args[0] = fakeSbatch(t, `cat >/dev/null; echo "  4242  "`)

	jobID, err := sub.Submit([]byte("script"))
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if jobID != "4242" {
		t.Errorf("jobID: got %q, want %q", jobID, "4242")
	}
}

func TestSubmit_EmptyJobIDErrors(t *testing.T) {
	sub := NewSbatchSubmitter(nil)
	sub.Cfg.Args[0] = fakeSbatch(t, `cat >/dev/null`)

	if _, err := sub.Submit([]byte("script")); err == nil {
		t.Fatal("expected error when sbatch prints no job id")
	} else if !strings.Contains(err.Error(), "no job id") {
		t.Errorf("error should mention missing job id, got: %v", err)
	}
}

func TestSubmit_FailureIncludesStderr(t *testing.T) {
	sub := NewSbatchSubmitter(nil)
	sub.Cfg.Args[0] = fakeSbatch(t, `echo "sbatch: error: invalid dependency" >&2; exit 1`)

	_, err := sub.Submit([]byte("script"))
	if err == nil {
		t.Fatal("expected error for failing sbatch")
	}
	if !strings.Contains(err.Error(), "invalid dependency") {
		t.Errorf("error should surface sbatch stderr, got: %v", err)
	}
}

func TestSubmit_Timeout(t *testing.T) {
	sub := NewSbatchSubmitter(nil)
	// exec keeps the sleeper in the shell's own process; a plain `sleep`
	// child would outlive the kill and hold the stdout pipe open, blocking
	// cmd.Wait until it exits.
	sub.Cfg.Args[0] = fakeSbatch(t, `exec sleep 5`)
	sub.Cfg.Timeout = 50 * time.Millisecond

	start := time.Now()
	if _, err := sub.Submit([]byte("script")); err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("timeout not enforced, Submit blocked for %v", elapsed)
	}
}

// Submit must surface the underlying exit error, not only stderr text.
func TestSubmit_FailureWrapsExitError(t *testing.T) {
	sub := NewSbatchSubmitter(nil)
	sub.Cfg.Args[0] = fakeSbatch(t, `exit 3`)

	_, err := sub.Submit([]byte("script"))
	if err == nil {
		t.Fatal("expected error for exit status 3")
	}
	if !strings.Contains(fmt.Sprint(err), "exit status 3") {
		t.Errorf("error should wrap the exit status, got: %v", err)
	}
}
