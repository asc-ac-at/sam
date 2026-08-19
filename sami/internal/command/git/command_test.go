package git

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
)

func TestNewRunner(t *testing.T) {
	r := command.NewRunner()
	if r == nil {
		t.Fatal("NewRunner returned nil")
	}
	if r.Timeout != 0 {
		t.Errorf("expected zero timeout, got %v", r.Timeout)
	}
}

func TestNewRunner_WithTimeout(t *testing.T) {
	d := 30 * time.Second
	r := command.NewRunner(command.WithTimeout(d))
	if r.Timeout != d {
		t.Errorf("expected timeout %v, got %v", d, r.Timeout)
	}
}

func TestNewRunner_DefaultTimeout(t *testing.T) {
	r1 := command.NewRunner()
	r2 := command.NewRunner()
	if r1.Timeout != r2.Timeout {
		t.Errorf("default timeouts differ: %v vs %v", r1.Timeout, r2.Timeout)
	}
}

func TestRunnerOptionChained(t *testing.T) {
	d := 5 * time.Minute
	r := command.NewRunner(command.WithTimeout(d))
	if r.Timeout != d {
		t.Errorf("expected %v, got %v", d, r.Timeout)
	}
}

func TestCmdRunner_New(t *testing.T) {
	r := command.NewRunner(command.WithTimeout(1 * time.Minute))
	cfg := r.New("echo", "hello")
	if cfg == nil {
		t.Fatal("New returned nil")
	}
	if cfg.Timeout != 1*time.Minute {
		t.Errorf("expected timeout %v, got %v", 1*time.Minute, cfg.Timeout)
	}
}

func TestCmdConfig_Builder(t *testing.T) {
	r := command.NewRunner(command.WithTimeout(5 * time.Second))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env := []string{"FOO=bar"}

	cfg := r.New("echo", "test").
		WithStdout(stdout).
		WithStderr(stderr).
		WithEnv(env)

	// Verify the config was built correctly
	if cfg.Args[0] != "echo" {
		t.Errorf("expected args[0] = 'echo', got %q", cfg.Args[0])
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("expected timeout %v, got %v", 5*time.Second, cfg.Timeout)
	}
}

func TestCmdConfig_WithTimeoutOverrides(t *testing.T) {
	r := command.NewRunner(command.WithTimeout(10 * time.Second))
	cfg := r.New("echo", "test").
		WithTimeout(30 * time.Second)

	if cfg.Timeout != 30*time.Second {
		t.Errorf("expected timeout override to 30s, got %v", cfg.Timeout)
	}
}

func TestCmdRunner_Quote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "'hello'"},
		{"with spaces", "'with spaces'"},
		{"", "''"},
		{"it's", "'it's'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := command.Quote(tt.input)
			if got != tt.want {
				t.Errorf("Quote(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRunShell_QuoteWraps(t *testing.T) {
	got := command.Quote("echo hello")
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("Quote should wrap in single quotes: got %q", got)
	}
}

func TestCmdConfig_WithStdio(t *testing.T) {
	r := command.NewRunner()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	stdin := strings.NewReader("input")

	cfg := r.New("cmd").
		WithStdout(stdout).
		WithStderr(stderr).
		WithStdin(stdin)

	// We can't easily verify the internal state of CmdConfig without
	// reflection, but we can verify the builder pattern returns non-nil
	if cfg == nil {
		t.Fatal("builder returned nil")
	}
	if cfg.Stdout == nil {
		t.Error("WithStdout should set stdout")
	}
	if cfg.Stderr == nil {
		t.Error("WithStderr should set stderr")
	}
	if cfg.Stdin == nil {
		t.Error("WithStdin should set stdin")
	}
}
