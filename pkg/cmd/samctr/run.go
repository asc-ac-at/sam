// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package samctr

import (
	"bytes"
	"context"
	"fmt"
	"log"
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
	log.Printf("runner %s", cmd)
	return cmd.Run()
}

// Set up a call to "/bin/sh"
// note that any job will be automatically terminated after the values set by
// WithTimeout
func RunSystemShell(runtime *RuntimeState, argFmt func(rs *RuntimeState) string) error {
	log.Printf("=== Shell Runner Called ===")
	runtime.SetApptainerBindPaths()
	arg := argFmt(runtime)
	ctx, cancel := context.WithTimeout(context.Background(), 72*time.Hour)
	defer cancel()
	// be careful to quote the arg ... some opts may contain literal quotes
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", fmt.Sprintf(`'%s'`, arg))
	cmd.Env = append(os.Environ(), runtime.Environ...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	log.Printf("ShellRunner cmd: %s", cmd)
	return cmd.Run()
}

func PullRunner(runtime *RuntimeState) error {
	log.Printf("=== Pull Runner Called ===")
	return Runner("apptainer", ApptainerPullArgs, runtime)
}

// Run a system command and get the output
func IoRunner(prg, arg string) (string, error) {
	log.Printf("=== IoRunner -> %s Called ===", prg)
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
