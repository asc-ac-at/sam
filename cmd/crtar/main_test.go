// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package main

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

// discard is the usage sink for tests; flag usage must not pollute output.
var discard = io.Discard

func TestParseFlags_Defaults(t *testing.T) {
	c, err := parseFlags(nil, discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if c.archSubdir != defaultArchSubdir {
		t.Errorf("archSubdir = %q, want %q", c.archSubdir, defaultArchSubdir)
	}
	if c.accelSubdir != "" {
		t.Errorf("accelSubdir = %q, want empty for CPU-only build", c.accelSubdir)
	}
	if c.repo != defaultRepo || c.name != defaultName {
		t.Errorf("unexpected defaults: repo=%q name=%q", c.repo, c.name)
	}
	if c.version {
		t.Error("version should default to false")
	}
}

func TestParseFlags_ArchSubdir(t *testing.T) {
	c, err := parseFlags([]string{"--arch-subdir", "x86_64/amd/zen5"}, discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if c.archSubdir != "x86_64/amd/zen5" {
		t.Errorf("archSubdir = %q, want x86_64/amd/zen5", c.archSubdir)
	}
}

func TestParseFlags_DeprecatedAlias(t *testing.T) {
	c, err := parseFlags([]string{"-cpuArchSubdir", "x86_64/generic"}, discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if c.archSubdir != "x86_64/generic" {
		t.Errorf("archSubdir via deprecated alias = %q, want x86_64/generic", c.archSubdir)
	}
}

func TestParseFlags_LastFlagWins(t *testing.T) {
	// legacy callers may pass -cpuArchSubdir; the new flag must override it
	// when both appear and --arch-subdir comes later.
	c, err := parseFlags([]string{"-cpuArchSubdir", "x86_64/generic", "--arch-subdir", "x86_64/amd/zen4"}, discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if c.archSubdir != "x86_64/amd/zen4" {
		t.Errorf("archSubdir = %q, want x86_64/amd/zen4 (last flag wins)", c.archSubdir)
	}
}

func TestParseFlags_AccelSubdirAccepted(t *testing.T) {
	c, err := parseFlags([]string{"--accel-subdir", "accel/nvidia/cc90"}, discard)
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	// accepted for now; combination logic lands in a later step
	if c.accelSubdir != "accel/nvidia/cc90" {
		t.Errorf("accelSubdir = %q, want accel/nvidia/cc90", c.accelSubdir)
	}
}

func TestParseFlags_Help(t *testing.T) {
	_, err := parseFlags([]string{"-h"}, discard)
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("parseFlags(-h) = %v, want flag.ErrHelp", err)
	}
}

func TestParseFlags_UnknownFlag(t *testing.T) {
	_, err := parseFlags([]string{"--bogus"}, discard)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should name the unknown flag, got: %v", err)
	}
}

func TestPrintContract(t *testing.T) {
	var buf strings.Builder
	printContract(&buf, "/out/gh-2.86.0-x86_64-amd-zen4-20260830120000.tar.gz")
	if buf.String() != "TARBALL=/out/gh-2.86.0-x86_64-amd-zen4-20260830120000.tar.gz\n" {
		t.Errorf("contract line = %q", buf.String())
	}
}
