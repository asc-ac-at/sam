package command

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewCmdConfig_Defaults(t *testing.T) {
	cfg := NewCmdConfig([]string{"echo", "hello"})

	if len(cfg.Args) != 2 || cfg.Args[0] != "echo" || cfg.Args[1] != "hello" {
		t.Errorf("Args = %v, want [echo hello]", cfg.Args)
	}
	if cfg.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s default (callers override per command)", cfg.Timeout)
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

func TestCmdConfig_RunEmptyArgs(t *testing.T) {
	cfg := &CmdConfig{}
	err := cfg.Run()
	if err == nil {
		t.Fatal("expected error for empty Args")
	}
	if !strings.Contains(err.Error(), "empty Args") {
		t.Errorf("error should name the cause, got: %v", err)
	}
}

func TestCmdConfig_RunCapturesStdout(t *testing.T) {
	cfg := NewCmdConfig([]string{"echo", "hello"})

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

func TestCmdConfig_RunStderrVisibleOnFailure(t *testing.T) {
	cfg := NewCmdConfig([]string{"sh", "-c", "echo boom >&2; exit 3"})

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

func TestCmdConfig_RunZeroTimeoutDisablesTimeout(t *testing.T) {
	cfg := NewCmdConfig([]string{"true"})
	cfg.Timeout = 0

	if err := cfg.Run(); err != nil {
		t.Fatalf("zero timeout must not expire the command: %v", err)
	}
}

func TestCmdConfig_RunTimeoutExpires(t *testing.T) {
	cfg := NewCmdConfig([]string{"sleep", "30"})
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
