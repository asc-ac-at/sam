package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// CmdConfig configures and runs a command with custom stdio,
// environment, and other exec.Cmd options.
// fields:
//
//	Args    - all arguments that get passed through to os/exec, with Args[0] being the program to call
//	Timeout - if not completed in this time, context will be canceled. Timeout: 0 -> disables the timeout
//	Stdout  - an io.Writer for Stdout
//	Stderr  - an io.Writer for Stderr
//	Stdin   - an io.Reader for Stdin
//	Env     - the program's execution environment, nil value will default to os.Environ
type CmdConfig struct {
	Args    []string
	Timeout time.Duration
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
	Env     []string
}

// NewCmdConfig creates a bare bones CmdConfig with some sane defaults
func NewCmdConfig(args []string) *CmdConfig {
	cfg := &CmdConfig{
		Args:    args,
		Timeout: 3 * time.Second,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Stdin:   os.Stdin,
	}
	return cfg
}

// Run executes the configured command.
// each caller should explicitly set the timeout for the command being run
func (c *CmdConfig) Run() error {
	if len(c.Args) == 0 {
		return fmt.Errorf("command.Run: empty Args")
	}
	ctx := context.Background()
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, c.Args[0], c.Args[1:]...)
	cmd.Env = c.Env
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	cmd.Stdin = c.Stdin

	return cmd.Run()
}
