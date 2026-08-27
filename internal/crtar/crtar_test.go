// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package crtar

// Real test case for FindModules
// {EESSI 2023.06} Apptainer> find  /tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4/modules/ -type f
// /tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4/modules/all/Go/1.25.0.lua

// Test case for FindSoftware
// {EESSI 2023.06} Apptainer> export ARCHDIR=/tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4
// {EESSI 2023.06} Apptainer> find ${ARCHDIR}/software/*/* -maxdepth 1 -name easybuild -type d
// /tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4/software/Go/1.25.0/easybuild
// {EESSI 2023.06} Apptainer> find ${ARCHDIR}/software/*/* -maxdepth 1 -name easybuild -type d | xargs -r dirname
// /tmp/software.asc.ac.at/overlay-upper/versions/2023.06/software/linux/x86_64/amd/zen4/software/Go/1.25.0

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFindCmd runs the refactored subproc-backed find helper on a
// directory that holds a single file and asserts that file shows up.
func TestFindCmd(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := findCmd([]string{dir, "-type", "f"})
	if err != nil {
		t.Fatalf("findCmd: %v", err)
	}
	// findCmd returns a single element carrying the joined stdout.
	if len(got) != 1 {
		t.Fatalf("findCmd result sets = %d, want 1: %v", len(got), got)
	}
	if !strings.Contains(got[0], "a.txt") {
		t.Errorf("findCmd output = %q, want to contain a.txt", got[0])
	}
}

// TestFindCmdMissingDir asserts that findCmd surfaces an error when the
// search root does not exist, rather than silently succeeding.
func TestFindCmdMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	if _, err := findCmd([]string{missing, "-type", "f"}); err == nil {
		t.Fatal("expected error for missing search root")
	}
}

// TestFindModules builds a modules tree containing a regular file and a
// symlink and asserts that findModules reports both, mirroring the
// documented file + symlink use case from the EESSI image.
func TestFindModules(t *testing.T) {
	dir := t.TempDir()
	softDir := filepath.Join(dir, "modules", "all", "Go")
	if err := os.MkdirAll(softDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modFile := filepath.Join(softDir, "1.25.0.lua")
	if err := os.WriteFile(modFile, []byte("-- module"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(modFile, filepath.Join(softDir, "default")); err != nil {
		t.Fatal(err)
	}

	got, err := findModules(dir)
	if err != nil {
		t.Fatalf("findModules: %v", err)
	}
	joined := strings.Join(got, "\n")
	if !strings.Contains(joined, "1.25.0.lua") {
		t.Errorf("findModules output = %q, missing module file", joined)
	}
	if !strings.Contains(joined, "default") {
		t.Errorf("findModules output = %q, missing symlink", joined)
	}
}

// TestFindModulesMissingPath asserts that findModules reports an error
// when the modules subtree is absent.
func TestFindModulesMissingPath(t *testing.T) {
	if _, err := findModules(t.TempDir()); err == nil {
		t.Fatal("expected error for missing modules path")
	}
}

// TestFindSoftware builds a software/<name>/<version>/easybuild tree and
// asserts that findSoftware reduces each easybuild dir to its parent,
// i.e. software/<name>/<version>.
func TestFindSoftware(t *testing.T) {
	dir := t.TempDir()
	easybuild := filepath.Join(dir, "software", "Go", "1.25.0", "easybuild")
	if err := os.MkdirAll(easybuild, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := findSoftware(dir)
	if err != nil {
		t.Fatalf("findSoftware: %v", err)
	}
	want := filepath.Join(dir, "software", "Go", "1.25.0")
	var found bool
	for _, p := range got {
		if filepath.Clean(p) == want {
			found = true
		}
	}
	if !found {
		t.Errorf("findSoftware = %v, want to contain %q", got, want)
	}
}

// TestFindSoftwareNoMatches asserts that an empty software tree yields
// no results and no error.
func TestFindSoftwareNoMatches(t *testing.T) {
	got, err := findSoftware(t.TempDir())
	if err != nil {
		t.Fatalf("findSoftware: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("findSoftware = %v, want empty", got)
	}
}
