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

// Accel tree layout being tested (EESSI style, as produced by a GPU build
// into the overlay):
//
//	versions/<ver>/software/linux/x86_64/amd/zen5/modules/all/...        (CPU tree, possibly untouched)
//	versions/<ver>/software/linux/x86_64/amd/zen5/accel/nvidia/cc100/modules/all/NVHPC/25.9.lua
//	versions/<ver>/software/linux/x86_64/amd/zen5/accel/nvidia/cc100/software/NVHPC/25.9/easybuild/...

const (
	testArchSubdir  = "x86_64/amd/zen5"
	testAccelSubdir = "accel/nvidia/cc100"
)

// mkModuleAndSoftware plants one module file and one package dir with an
// easybuild metadata subtree under root, mirroring an EasyBuild install.
func mkModuleAndSoftware(t *testing.T, root, pkg, version string) {
	t.Helper()

	modDir := filepath.Join(root, "modules", "all", pkg)
	if err := os.MkdirAll(modDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modDir, version+".lua"), []byte("-- module"), 0o644); err != nil {
		t.Fatal(err)
	}

	ebDir := filepath.Join(root, "software", pkg, version, "easybuild")
	if err := os.MkdirAll(ebDir, 0o755); err != nil {
		t.Fatal(err)
	}
}

// readListFile returns the trimmed lines of a MakeListFile output file.
func readListFile(t *testing.T, f *os.File) []string {
	t.Helper()
	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// TestMakeListFile_ArchAndAccel plants content in both the CPU arch dir and
// the accel subdir below it, and asserts MakeListFile lists both.
func TestMakeListFile_ArchAndAccel(t *testing.T) {
	repo := uniqueRepo()
	t.Cleanup(func() { os.RemoveAll(filepath.Join("/tmp", repo)) })

	cpuRoot := archDir(repo, "2025.06", testArchSubdir)
	accelRoot := filepath.Join(cpuRoot, testAccelSubdir)
	mkModuleAndSoftware(t, cpuRoot, "Go", "1.25.7")
	mkModuleAndSoftware(t, accelRoot, "NVHPC", "25.9")

	lf, err := MakeListFile(repo, "2025.06", testArchSubdir, testAccelSubdir)
	if err != nil {
		t.Fatalf("MakeListFile: %v", err)
	}

	lines := readListFile(t, lf)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		filepath.Join(cpuRoot, "modules", "all", "Go", "1.25.7.lua"),
		filepath.Join(cpuRoot, "software", "Go", "1.25.7"),
		filepath.Join(accelRoot, "modules", "all", "NVHPC", "25.9.lua"),
		filepath.Join(accelRoot, "software", "NVHPC", "25.9"),
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("list file missing %q\ngot:\n%s", want, joined)
		}
	}
}

// TestMakeListFile_AccelOnly reproduces an accelerator build that touched
// only the accel tree: the CPU arch dir may not even exist in the overlay.
// MakeListFile must still succeed and list the accel content.
func TestMakeListFile_AccelOnly(t *testing.T) {
	repo := uniqueRepo()
	t.Cleanup(func() { os.RemoveAll(filepath.Join("/tmp", repo)) })

	accelRoot := filepath.Join(archDir(repo, "2025.06", testArchSubdir), testAccelSubdir)
	mkModuleAndSoftware(t, accelRoot, "NVHPC", "25.9")

	lf, err := MakeListFile(repo, "2025.06", testArchSubdir, testAccelSubdir)
	if err != nil {
		t.Fatalf("MakeListFile: %v", err)
	}

	lines := readListFile(t, lf)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		filepath.Join(accelRoot, "modules", "all", "NVHPC", "25.9.lua"),
		filepath.Join(accelRoot, "software", "NVHPC", "25.9"),
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("list file missing %q\ngot:\n%s", want, joined)
		}
	}
}

// TestMakeListFile_NothingFound asserts an empty overlay is an error, not a
// silently empty list file.
func TestMakeListFile_NothingFound(t *testing.T) {
	repo := uniqueRepo()
	t.Cleanup(func() { os.RemoveAll(filepath.Join("/tmp", repo)) })
	if err := os.MkdirAll(versionsDir(repo), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := MakeListFile(repo, "2025.06", testArchSubdir, testAccelSubdir); err == nil {
		t.Fatal("expected error when nothing was built into the overlay")
	} else if !strings.Contains(err.Error(), "nothing to pack") {
		t.Errorf("error should state nothing was found, got: %v", err)
	}
}

// TestTarballPath covers the name normalization for CPU-only and accel
// builds: slashes become dashes, and the accel target is appended when set.
func TestTarballPath(t *testing.T) {
	outdir := "/out"

	got := tarballPath("x86_64/amd/zen5", "", "sami", outdir)
	if !strings.Contains(got, "sami-x86_64-amd-zen5-") {
		t.Errorf("CPU-only tarball name %q should contain name and normalized arch", got)
	}
	if strings.Contains(got, "accel") {
		t.Errorf("CPU-only tarball name %q must not mention accel", got)
	}

	got = tarballPath("x86_64/amd/zen5", testAccelSubdir, "sami", outdir)
	want := filepath.Join(outdir, "sami-x86_64-amd-zen5-accel-nvidia-cc100-")
	if !strings.HasPrefix(got, want) {
		t.Errorf("accel tarball name = %q, want prefix %q", got, want)
	}
	if !strings.HasSuffix(got, ".tar.gz") {
		t.Errorf("tarball name %q should keep the .tar.gz suffix", got)
	}
}

// TestExecTar_AccelNameOnDisk runs a full arch+accel packaging cycle over a
// fake overlay and asserts the tarball on disk carries the accel target in
// its name and the accel module in its payload.
func TestExecTar_AccelNameOnDisk(t *testing.T) {
	repo := uniqueRepo()
	t.Cleanup(func() { os.RemoveAll(filepath.Join("/tmp", repo)) })

	accelRoot := filepath.Join(archDir(repo, "2025.06", testArchSubdir), testAccelSubdir)
	mkModuleAndSoftware(t, accelRoot, "NVHPC", "25.9")

	outdir := t.TempDir()
	lf, err := MakeListFile(repo, "2025.06", testArchSubdir, testAccelSubdir)
	if err != nil {
		t.Fatalf("MakeListFile: %v", err)
	}

	tb, err := ExecTar(repo, testArchSubdir, testAccelSubdir, "sami", outdir, lf)
	if err != nil {
		t.Fatalf("ExecTar: %v", err)
	}

	if !strings.Contains(filepath.Base(tb), "-accel-nvidia-cc100-") {
		t.Errorf("tarball name %q should carry the normalized accel target", tb)
	}

	names := readTarNames(t, tb)
	var found bool
	for _, n := range names {
		if strings.Contains(n, "accel/nvidia/cc100/modules/all/NVHPC/25.9.lua") {
			found = true
		}
	}
	if !found {
		t.Errorf("tarball members %v missing the accel module file", names)
	}
}
