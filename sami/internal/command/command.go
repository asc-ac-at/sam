package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// CmdRunner controls _when_ a shell command is run
type CmdRunner interface {
	Run(args ...string) error
	RunShell(arg string) error
	New(args ...string) *CmdConfig
}

// cmdRunner is for minimal CmdConfig situations, where all we want to set is the timeout
type cmdRunner struct {
	Timeout time.Duration
}

// RunnerOption is a way to dynamically set options on a runner
// Note that WithTimout is the only option that can currently be set, so it's somehow a bit of a
// fancy implementation for this one thing. Also we're mixing implmentations in this file.
// See git.checkoutCommit or git.initializeRepo for examples of the other implementation shape
type RunnerOption func(*cmdRunner)

// NewRunner is a bit too clever for my taste, it dynamically sets the
// options on a cmdRunner it's unclear if we are restricted to only
// every setting functions that are implemented to return a RunnerOption
// function type, note we would probably need to expand the cmdRunner
// struct in order to handle the other types.
func NewRunner(opts ...RunnerOption) *cmdRunner {
	r := &cmdRunner{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WithTimeout can be used to create a new runner with a timeout
// example:
//
//	runner := NewRunner(WithTimeout(10 * time.Minute))
func WithTimeout(t time.Duration) RunnerOption {
	return func(r *cmdRunner) { r.Timeout = t }
}

// Run runs the shell command ARG within context command controlled by a TIMEOUT
func (r *cmdRunner) Run(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// Quote simply wraps a string in single quotes in preperation of passing to shell
func Quote(str string) string {
	return fmt.Sprintf(`'%s'`, str)
}

// Run shell is a preconfigured cmdRunner that executes the arg in a shell
func (r *cmdRunner) RunShell(arg string) error {
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", Quote(arg))
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func (r *cmdRunner) New(args ...string) *CmdConfig {
	return &CmdConfig{
		Args:    args,
		Timeout: r.Timeout,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Stdin:   os.Stdin,
		Env:     os.Environ(),
	}
}

// CmdConfig configures and runs a command with custom stdio,
// environment, and other exec.Cmd options.
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
		Timeout: 30 * time.Second,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Stdin:   os.Stdin,
	}
	return cfg
}

// WithStdout sets the writer for command stdout.
func (c *CmdConfig) WithStdout(w io.Writer) *CmdConfig {
	c.Stdout = w
	return c
}

// WithStderr sets the writer for command stderr.
func (c *CmdConfig) WithStderr(w io.Writer) *CmdConfig {
	c.Stderr = w
	return c
}

// WithStdin sets the reader for command stdin.
func (c *CmdConfig) WithStdin(r io.Reader) *CmdConfig {
	c.Stdin = r
	return c
}

// WithEnv sets the environment variables for the command.
func (c *CmdConfig) WithEnv(env []string) *CmdConfig {
	c.Env = env
	return c
}

// WithTimeout overrides the runner's default timeout for this command.
func (c *CmdConfig) WithTimeout(t time.Duration) *CmdConfig {
	c.Timeout = t
	return c
}

// Run executes the configured command.
func (c *CmdConfig) Run() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Args[0], c.Args[1:]...)
	cmd.Env = c.Env
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	cmd.Stdin = c.Stdin

	return cmd.Run()
}

// Output runs the command and returns its stdout only (stderr is discarded).
func (c *CmdConfig) Output() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Args[0], c.Args[1:]...)
	cmd.Env = c.Env
	cmd.Stdin = c.Stdin

	return cmd.Output()
}

// CombinedOutput runs the command and returns its combined stdout and stderr.
func (c *CmdConfig) CombinedOutput() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.Args[0], c.Args[1:]...)
	cmd.Env = c.Env
	cmd.Stdin = c.Stdin

	return cmd.CombinedOutput()
}
