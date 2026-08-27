// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/

// Package subproc provides configurable execution of external commands:
// custom stdio, environment, and timeouts on top of os/exec.
package subproc

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Config configures and runs a command with custom stdio,
// environment, and other exec.Cmd options.
// fields:
//
//	Args    - all arguments that get passed through to os/exec, with Args[0] being the program to call
//	Timeout - if not completed in this time, context will be canceled. Timeout: 0 -> disables the timeout
//	Stdout  - an io.Writer for Stdout
//	Stderr  - an io.Writer for Stderr
//	Stdin   - an io.Reader for Stdin
//	Env     - the program's execution environment, nil value will default to os.Environ
type Config struct {
	Args    []string
	Timeout time.Duration
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
	Env     []string
}

// New creates a bare bones Config with some sane defaults.
// The timeout defaults to 0 (disabled): each caller should
// explicitly set the timeout for the command being run.
func New(args []string) *Config {
	cfg := &Config{
		Args:    args,
		Timeout: 0,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Stdin:   os.Stdin,
	}
	return cfg
}

// Run executes the configured command.
func (c *Config) Run() error {
	if len(c.Args) == 0 {
		return fmt.Errorf("subproc.Run: empty Args")
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
