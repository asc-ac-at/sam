package host_injections

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

func TestNewCommand_LongDescription(t *testing.T) {
	opts := shared.NewOptions()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
	if !strings.Contains(cmd.Long, "host_injections") {
		t.Error("Long description should mention 'host_injections'")
	}
	if !strings.Contains(cmd.Long, "EESSI") {
		t.Error("Long description should mention 'EESSI'")
	}
}

func TestNewCommand_PRunE(t *testing.T) {
	opts := shared.NewOptions()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	if cmd.PreRunE == nil {
		t.Fatal("PreRunE should be set")
	}

	// PreRunE should require --name
	err := cmd.PreRunE(cmd, []string{})
	if err == nil {
		t.Error("expected error when --name is not set")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Errorf("error should mention '--name': %v", err)
	}
}

func TestNewCommand_PRunE_NameRequired(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "my-build"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error when --name is set: %v", err)
	}
}

func TestNewCommand_PRunE_MutualExclusivity_GitBranchAndGitCommit(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	opts.GitBranch = "main"
	opts.GitCommit = "abc123"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err == nil {
		t.Error("expected error for --gitBranch and --gitCommit both set")
	}
}

func TestNewCommand_PRunE_MutualExclusivity_GitBranchAndMR(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	opts.GitBranch = "main"
	opts.GitMergeReqId = 42
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err == nil {
		t.Error("expected error for --gitBranch and --git-mr-id both set")
	}
}

func TestNewCommand_PRunE_MutualExclusivity_GitCommitAndMR(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	opts.GitCommit = "abc123"
	opts.GitMergeReqId = 42
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err == nil {
		t.Error("expected error for --gitCommit and --git-mr-id both set")
	}
}

func TestNewCommand_PRunE_AllExclusiveSet(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	opts.GitBranch = "main"
	opts.GitCommit = "abc123"
	opts.GitMergeReqId = 42
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err == nil {
		t.Error("expected error for all three exclusive flags set")
	}
}

func TestNewCommand_PRunE_OnlyBranchOK(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	opts.GitBranch = "develop"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error with only --git-branch: %v", err)
	}
}

func TestNewCommand_PRunE_OnlyCommitOK(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	opts.GitCommit = "abc123"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error with only --git-commit: %v", err)
	}
}

func TestNewCommand_PRunE_OnlyMRIDOK(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	opts.GitMergeReqId = 42
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error with only --git-mr-id: %v", err)
	}
}

func TestNewCommand_PRunE_NoFlagsOK(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	err := cmd.PreRunE(cmd, []string{})
	if err != nil {
		t.Errorf("expected no error with only --name set: %v", err)
	}
}

func TestNewCommand_RunDoesNotPanic(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	// Should not panic when Run is called
	cmd.Run(cmd, []string{})
}

func TestNewCommand_RunPrintsMessage(t *testing.T) {
	opts := shared.NewOptions()
	opts.Name = "test"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cmd := NewCommand(opts, logger)

	// Capture stdout to verify the printed message
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.Run(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	got := string(buf[:n])
	if !strings.Contains(got, "hostInjections called") {
		t.Errorf("expected 'hostInjections called' in output, got: %q", got)
	}
}

func TestNewCommand_AddsToParent(t *testing.T) {
	parentCmd := &cobra.Command{Use: "parent"}
	opts := shared.NewOptions()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	subCmd := NewCommand(opts, logger)

	parentCmd.AddCommand(subCmd)

	if len(parentCmd.Commands()) != 1 {
		t.Errorf("expected 1 child command, got %d", len(parentCmd.Commands()))
	}

	found := parentCmd.Commands()[0]
	if found.Use != "hostInjections" {
		t.Errorf("expected child command Use 'hostInjections', got %q", found.Use)
	}
}
