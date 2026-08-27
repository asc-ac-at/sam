// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package subproc

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNew_Defaults(t *testing.T) {
	cfg := New([]string{"echo", "hello"})

	if len(cfg.Args) != 2 || cfg.Args[0] != "echo" || cfg.Args[1] != "hello" {
		t.Errorf("Args = %v, want [echo hello]", cfg.Args)
	}
	if cfg.Timeout != 0 {
		t.Errorf("Timeout = %v, want 0 default (disabled; callers set one explicitly)", cfg.Timeout)
	}
	if cfg.Stdout != os.Stdout {
		t.Error("Stdout should default to os.Stdout")
	}
	if cfg.Stderr != os.Stderr {
		t.Error("Stderr should default to os.Stderr")
	}
	if cfg.Stdin != os.Stdin {
		t.Error("Stdin should default to os.Stdin")
	}
	if cfg.Env != nil {
		t.Errorf("Env = %v, want nil (exec.Cmd inherits os.Environ on nil)", cfg.Env)
	}
}

func TestConfig_RunEmptyArgs(t *testing.T) {
	cfg := &Config{}
	err := cfg.Run()
	if err == nil {
		t.Fatal("expected error for empty Args")
	}
	if !strings.Contains(err.Error(), "empty Args") {
		t.Errorf("error should name the cause, got: %v", err)
	}
}

func TestConfig_RunCapturesStdout(t *testing.T) {
	cfg := New([]string{"echo", "hello"})

	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr

	if err := cfg.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "hello" {
		t.Errorf("stdout = %q, want %q", got, "hello")
	}
}

func TestConfig_RunStderrVisibleOnFailure(t *testing.T) {
	cfg := New([]string{"sh", "-c", "echo boom >&2; exit 3"})

	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr

	err := cfg.Run()
	if err == nil {
		t.Fatal("expected non-zero exit error")
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Errorf("stderr buffer should carry the failure diagnostics, got: %q", stderr.String())
	}
	// the canon: nothing on stdout for a failing, silent command
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty, got: %q", stdout.String())
	}
}

func TestConfig_RunZeroTimeoutDisablesTimeout(t *testing.T) {
	cfg := New([]string{"true"})
	cfg.Timeout = 0

	if err := cfg.Run(); err != nil {
		t.Fatalf("zero timeout must not expire the command: %v", err)
	}
}

func TestConfig_RunTimeoutExpires(t *testing.T) {
	cfg := New([]string{"sleep", "30"})
	cfg.Timeout = 50 * time.Millisecond

	var stderr bytes.Buffer
	cfg.Stderr = &stderr

	err := cfg.Run()
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "signal: killed") {
		t.Errorf("timeout kills the process, got: %v", err)
	}
}
