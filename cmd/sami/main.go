/*
Copyright © 2026 Adam McCartney
// SPDX-License-Identifier: MIT
*/
package main

import (
	"log/slog"
	"os"

	"github.com/asc-ac-at/sam/internal/sami/cli/shared"
	"github.com/asc-ac-at/sam/internal/sami/logging"
	"github.com/asc-ac-at/sam/pkg/cmd/sami"
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
	sami.Execute(opts, logger)
}
