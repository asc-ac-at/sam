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
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/rand"
	"fmt"
	"io"
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

/*
 * ExecTar tests
 *
 * ExecTar builds its working directory from the fixed repo layout
 * /tmp/<repo>/versions, so these tests construct a fake overlay tree
 * under a uniquely-named /tmp/<repo> and remove it afterwards. The
 * layout mirrors an actual build: a modules lua file plus a software
 * easybuild subtree.
 */

// uniqueRepo returns a repo name unlikely to collide with a real overlay
// dir or with other test runs.
func uniqueRepo() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("crtar-test-%x", b)
}

// writeListFile writes each entry on its own line to a fresh temp file in
// outdir and returns the open *os.File, mirroring MakeListFile's output.
func writeListFile(t *testing.T, outdir string, entries ...string) *os.File {
	t.Helper()
	lf, err := os.CreateTemp(outdir, "files.list.txt")
	if err != nil {
		t.Fatal(err)
	}
	w := bufio.NewWriter(lf)
	for _, e := range entries {
		if _, err := w.WriteString(e + "\n"); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := lf.Sync(); err != nil {
		t.Fatal(err)
	}
	return lf
}

// readTarNames lists the member names of a .tar.gz archive.
func readTarNames(t *testing.T, archivePath string) []string {
	t.Helper()
	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
	return names
}

// TestExecTar_CreatesTarball builds a fake overlay holding one module file,
// lists it, and asserts ExecTar produces a tarball containing the file and
// leaves no lockfile behind.
func TestExecTar_CreatesTarball(t *testing.T) {
	repo := uniqueRepo()
	t.Cleanup(func() { os.RemoveAll(filepath.Join("/tmp", repo)) })

	modFile := filepath.Join(
		versionsDir(repo), "2025.06", "software", "linux", "x86_64",
		"amd", "zen4", "modules", "all", "Go", "1.25.7.lua")
	if err := os.MkdirAll(filepath.Dir(modFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modFile, []byte("module load Go"), 0o644); err != nil {
		t.Fatal(err)
	}

	outdir := t.TempDir()
	listFile := writeListFile(t, outdir, modFile)

	name, cpu := "sami", "amd/zen4"
	tb, err := ExecTar(repo, cpu, name, outdir, listFile)
	if err != nil {
		t.Fatalf("ExecTar: %v", err)
	}
	if !strings.HasPrefix(tb, outdir+string(os.PathSeparator)) || !strings.HasSuffix(tb, "tar.gz") {
		t.Errorf("returned tarball path %q: want <outdir>/<...>.tar.gz", tb)
	}

	matches, err := filepath.Glob(filepath.Join(outdir, name+"-*.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 tarball, got %d (%v)", len(matches), matches)
	}
	if matches[0] != tb {
		t.Errorf("returned path %q does not match the tarball on disk %q", tb, matches[0])
	}

	names := readTarNames(t, matches[0])
	var found bool
	for _, n := range names {
		if strings.HasSuffix(n, "1.25.7.lua") {
			found = true
		}
	}
	if !found {
		t.Errorf("tarball members %v missing module file", names)
	}

	// lockfile should have been removed on success
	if _, err := os.Stat(matches[0] + ".lock"); !os.IsNotExist(err) {
		t.Errorf("lockfile still present after successful ExecTar")
	}
}

// TestExecTar_FailsOnMissingFile lists a path that does not exist and
// asserts ExecTar returns an error naming the tarball step.
func TestExecTar_FailsOnMissingFile(t *testing.T) {
	repo := uniqueRepo()
	t.Cleanup(func() { os.RemoveAll(filepath.Join("/tmp", repo)) })
	// the -C working dir must exist for tar to start
	if err := os.MkdirAll(versionsDir(repo), 0o755); err != nil {
		t.Fatal(err)
	}

	outdir := t.TempDir()
	missing := filepath.Join(versionsDir(repo), "2025.06", "nope.lua")
	listFile := writeListFile(t, outdir, missing)

	tb, err := ExecTar(repo, "amd/zen4", "sami", outdir, listFile)
	if err == nil {
		t.Fatal("expected ExecTar to fail for a missing listed file")
	}
	if tb != "" {
		t.Errorf("failed ExecTar returned path %q, want empty", tb)
	}
	if !strings.Contains(err.Error(), "creating tarball") {
		t.Errorf("error should mention the tarball step, got: %v", err)
	}
}

// TestAcquireLockfileAlreadyPresent asserts that a pre-existing lockfile
// blocks acquisition with an error naming the lockfile.
func TestAcquireLockfileAlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "sami.tar.gz")
	// acquireLockfile strips the .tar.gz suffix before appending .lock
	lock := strings.TrimSuffix(tarball, ".tar.gz") + ".lock"
	if err := os.WriteFile(lock, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireLockfile(tarball); err == nil {
		t.Fatal("expected lockfile-present error")
	} else if !strings.Contains(err.Error(), "already present") {
		t.Errorf("error should mention the lockfile, got: %v", err)
	}
}
