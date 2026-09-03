// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

// Simple runner for non-interactive tasks
// Note the timeout -> (processes running under this context will be killed when
// the timeout completes)
func Runner(prg string, argFmt func(rs *RuntimeState) []string, runtime *RuntimeState) error {
	runtime.SetApptainerBindPaths()
	args := argFmt(runtime)
	ctx, cancel := context.WithTimeout(context.Background(), 72*time.Hour)
	defer cancel()
	// Q: we're hardcoding "apptainer" below, maybe we should make this more
	// generic in the future?
	cmd := exec.CommandContext(ctx, prg, args...)
	cmd.Env = append(os.Environ(), runtime.Environ...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	slog.Debug("runner", "cmd", cmd)
	return cmd.Run()
}

// Set up a call to "/bin/sh"
// note that any job will be automatically terminated after the values set by
// WithTimeout
func RunSystemShell(runtime *RuntimeState, argFmt func(rs *RuntimeState) string) error {
	runtime.SetApptainerBindPaths()
	arg := argFmt(runtime)
	ctx, cancel := context.WithTimeout(context.Background(), 72*time.Hour)
	defer cancel()
	// pass the arg line as-is; needed quoting (e.g. fusemount "container:..."
	// specs) is already present in the returned string and sh parses it
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", arg)
	cmd.Env = append(os.Environ(), runtime.Environ...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	slog.Debug("shell runner", "cmd", cmd)
	return cmd.Run()
}

func PullRunner(runtime *RuntimeState) error {
	return Runner("apptainer", ApptainerPullArgs, runtime)
}

// Run a system command and get the output
func IoRunner(prg, arg string) (string, error) {
	slog.Debug("io runner", "prg", prg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, prg, arg)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("IoRunner %s: %w", prg, err)
	}
	return out.String(), nil
}
