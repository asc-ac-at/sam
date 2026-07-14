/*
Copyright © 2026 Adam McCartney
// SPDX-License-Identifier: MIT
*/
package main

import (
	"log/slog"
	"os"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/cmd"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging"
)

func main() {
	opts := shared.NewOptions()
	// setup cli logging
	var logLevel slog.Level
	if opts.Verbose {
		logLevel = slog.LevelDebug
	} else {
		logLevel = slog.LevelInfo
	}
	cfg := logging.Config{
		Level: logLevel,
	}
	logger := logging.NewLogger(os.Stdout, cfg)
	cmd.Execute(logger)
}
