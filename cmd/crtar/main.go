// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/asc-ac-at/sam/internal/crtar"
)

// defaults
var defaultSWSVersion = "2025.06"
var defaultArchSubdir = "x86_64/amd/zen4"
var defaultName = "unnamed"
var defaultRepo = "software.asc.ac.at"

// config holds the resolved command-line options. archSubdir is read from
// --arch-subdir (or the deprecated -cpuArchSubdir alias). accelSubdir is
// accepted via --accel-subdir but is not yet applied to the search path.
type config struct {
	eessiVersion string
	archSubdir   string
	accelSubdir  string
	name         string
	outputDir    string
	repo         string
	version      bool
	verbose      bool
}

// parseFlags parses crtar's command line. The deprecated -cpuArchSubdir
// alias shares the archSubdir variable with --arch-subdir, so the last
// flag given on the command line wins. out receives flag usage text.
func parseFlags(args []string, out io.Writer) (*config, error) {
	fs := flag.NewFlagSet("crtar", flag.ContinueOnError)
	fs.SetOutput(out)

	var c config
	fs.StringVar(&c.eessiVersion, "EESSI-version", defaultSWSVersion, "Version of the (EEESI based) software stack")
	fs.StringVar(&c.archSubdir, "arch-subdir", defaultArchSubdir, "Architecture subdirectory to search (e.g. x86_64/amd/zen4)")
	fs.StringVar(&c.archSubdir, "cpuArchSubdir", defaultArchSubdir, "Deprecated alias for --arch-subdir")
	fs.StringVar(&c.accelSubdir, "accel-subdir", "", "Accelerator subdirectory (e.g. accel/nvidia/cc90); accepted but not yet applied to the search path")
	fs.StringVar(&c.name, "name", defaultName, "Name of the tarball being created")
	fs.StringVar(&c.outputDir, "outputDir", "/opt/adm/sam-archives", "Output directory to save tarball")
	fs.StringVar(&c.repo, "repo", defaultRepo, "CVMFS repository for which the software was built")
	fs.BoolVar(&c.version, "version", false, "print version info")
	fs.BoolVar(&c.verbose, "verbose", false, "enable debug logging")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return &c, nil
}

var Version = "unknown"

func printVersion() {
	fmt.Printf("crtar version: %s\n", Version)
}

func main() {
	cfg, err := parseFlags(os.Args[1:], os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(2)
	}
	if cfg.version {
		printVersion()
		return
	}
	if cfg.verbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}
	if cfg.accelSubdir != "" {
		slog.Warn("note: --accel-subdir is not yet applied to the search path", "value", cfg.accelSubdir)
	}

	listFile, err := crtar.MakeListFile(cfg.repo, cfg.eessiVersion, cfg.archSubdir)
	if err != nil {
		slog.Error("making list file", "error", err)
		os.Exit(1)
	}

	if err := crtar.ExecTar(cfg.repo, cfg.archSubdir, cfg.name, cfg.outputDir, listFile); err != nil {
		slog.Error("execTar failed", "error", err)
		os.Exit(1)
	}
}
