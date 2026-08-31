// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2026 Adam McCartney <adam@mur.at>
*/
package crtar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecTar_IncludesFullPackageTree reproduces the production list shape:
// findSoftware emits package *directories* (e.g. software/Go/1.25.7) and
// findModules emits module files + symlinks. Asserts the tarball recurses
// the whole directory (bin payload included) and preserves the symlink.
func TestExecTar_IncludesFullPackageTree(t *testing.T) {
	repo := uniqueRepo()
	t.Cleanup(func() { os.RemoveAll(filepath.Join("/tmp", repo)) })

	pkgDir := filepath.Join(
		versionsDir(repo), "2025.06", "software", "linux", "x86_64",
		"amd", "zen4", "software", "Go", "1.25.7")

	binFile := filepath.Join(pkgDir, "bin", "go")
	if err := os.MkdirAll(filepath.Dir(binFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binFile, []byte("fake go toolchain"), 0o755); err != nil {
		t.Fatal(err)
	}
	ebDir := filepath.Join(pkgDir, "easybuild")
	if err := os.MkdirAll(ebDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ebDir, "Go-1.25.7.eb"), []byte("name = 'Go'"), 0o644); err != nil {
		t.Fatal(err)
	}

	modDir := filepath.Join(
		versionsDir(repo), "2025.06", "software", "linux", "x86_64",
		"amd", "zen4", "modules", "all", "Go")
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	modFile := filepath.Join(modDir, "1.25.7.lua")
	if err := os.WriteFile(modFile, []byte("module perms"), 0o644); err != nil {
		t.Fatal(err)
	}
	// absolute symlink, as EESSI compiler-precedence links appear in the overlay
	compilerDir := filepath.Join(modDir, "..", "..", "compiler", "Go")
	if err := os.MkdirAll(compilerDir, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(compilerDir, "1.25.7.lua")
	if err := os.Symlink("/cvmfs/software.asc.ac.at/versions/2025.06/software/linux/x86_64/amd/zen4/modules/all/Go/1.25.7.lua", link); err != nil {
		t.Fatal(err)
	}

	// the list mirrors production: the package directory, one module file, one symlink
	outdir := t.TempDir()
	listFile := writeListFile(t, outdir, pkgDir, modFile, link)

	tb, err := ExecTar(repo, "amd/zen4", "sami", outdir, listFile)
	if err != nil {
		t.Fatalf("ExecTar: %v", err)
	}

	names := readTarNames(t, tb)
	wantSubstrings := []string{
		"software/Go/1.25.7/bin/go",
		"software/Go/1.25.7/easybuild/Go-1.25.7.eb",
		"modules/all/Go/1.25.7.lua",
		"modules/compiler/Go/1.25.7.lua",
	}
	for _, want := range wantSubstrings {
		var found bool
		for _, n := range names {
			if strings.Contains(n, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("tarball members %v missing %q", names, want)
		}
	}
}
