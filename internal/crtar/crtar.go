// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package crtar

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/asc-ac-at/sam/pkg/subproc"
)

// ExecTar constructs a "tar" command and
// tar --exclude=.cvmfscatalog --exclude=*.wh.* -C ${TOPDIR} -czf ${TARBALL} --files-from=${FILES_LIST}
// TOPDIR=workingDir
// TARBALL=tarballName
// FILES_LIST=listFile
// Change to the workingDir and create a tarball named tarballName using the
// files in the listFile. Exclude anything mathching the two regular expressions
// at the front of the args slice. Returns the absolute path of the tarball.
// accelSubdir may be empty (CPU-only build) or an EESSI-style accelerator
// subdir relative to the arch dir (e.g. accel/nvidia/cc100); when set it is
// included in the tarball name.
func ExecTar(repo, archSubdir, accelSubdir, name, outdir string, listFile *os.File, owner, group string) (string, error) {
	// the list file is single-use: consume it here and clean it up, it
	// carries no value after the tarball exists
	defer os.Remove(listFile.Name())
	var args []string
	// second exclude is redundant because of the filter below
	args = append(args, "tar", "--exclude=.cvmfscatalog", "--exclude=*.wh.*")
	workingDir := versionsDir(repo)
	args = append(args, "-C", workingDir)
	tarball := tarballPath(archSubdir, accelSubdir, name, outdir)
	args = append(args, "-czf", tarball)
	filesFrom := fmt.Sprintf("--files-from=%s", listFile.Name())
	args = append(args, filesFrom)

	// optionally change ownership of files being packed into tarball
	if (owner != "") || (group != "") {
		args = append(args, "--numeric-owner")
	}
	if owner != "" {
		u := fmt.Sprintf("--owner=%s", owner)
		args = append(args, u)
	}
	if group != "" {
		g := fmt.Sprintf("--group=%s", group)
		args = append(args, g)
	}

	lockFile, lferr := acquireLockfile(tarball)
	if lferr != nil {
		return "", fmt.Errorf("could not acquire lockfile: %w", lferr)
	}

	var stderr bytes.Buffer
	cfg := subproc.New(args)
	cfg.Stdout = io.Discard
	cfg.Stderr = &stderr

	if err := cfg.Run(); err != nil {
		return "", fmt.Errorf("creating tarball %s failed %w: %s", tarball, err, strings.TrimSpace(stderr.String()))
	}
	slog.Info("created tarball", "path", tarball)
	removeLockfile(lockFile)
	return tarball, nil
}

// tarballPath constructs a filepath to the tarball that will be subsequently created.
// returns a string containing the absolute path
// The target triplet in the name is <arch>-<accel> with slashes normalized to
// dashes, e.g. x86_64-amd-zen5-accel-nvidia-cc100 (mirrors EESSI's tarball
// naming in bot/build.sh); for CPU-only builds the accel part is omitted.
func tarballPath(archSubdir, accelSubdir, name, outdir string) string {
	target := strings.ReplaceAll(archSubdir, "/", "-")
	if accelSubdir != "" {
		target += "-" + strings.ReplaceAll(accelSubdir, "/", "-")
	}
	t := time.Now()
	ts := t.Format("20060102150405")
	result := fmt.Sprintf("%s/%s-%s-%s.tar.gz", outdir, name, target, ts)
	slog.Debug("resolved tarball path", "path", result)
	return result
}

// get the working directory for tarball creation
// Assume that we are working in a container with a fusemount writeable overlay
// that is bind mounted for a particular CVMFS repo at
// /tmp/cvmfs/<repo>/operlay-upper/versions
func overlayUpperDir(repo string) string {
	// trailing slash is important!
	repoDir := fmt.Sprintf("/tmp/%s/overlay-upper/", repo)
	return path.Dir(repoDir)
}

// versionsDir constructs a path in the overlayfs
func versionsDir(repo string) string {
	return path.Join(overlayUpperDir(repo), "/versions")
}

// archDir is the subdirectory representing a microarchitecture
func archDir(repo string, version string, archSubdir string) string {
	versionsDir := versionsDir(repo)
	return path.Join(versionsDir, version, "software", "linux", archSubdir)
}

// Check for the presence of a lockfile
// Lockfiles are created in order to prevent race conditions whereby the
// ingestion service tries to read a partially written tarball
func acquireLockfile(tarballPath string) (*os.File, error) {
	name := strings.TrimSuffix(tarballPath, ".tar.gz")
	lf := fmt.Sprintf("%s.lock", name)
	lockFilePath := filepath.Clean(lf)
	slog.Debug("found or creating lockfile", "path", lockFilePath)

	if _, err := os.Stat(lockFilePath); err == nil { // lockfile found!
		return nil, fmt.Errorf("lockfile %s already present", lockFilePath)
	} else {
		result, err := os.Create(lockFilePath)
		if err != nil {
			return nil, fmt.Errorf("acquireLockfile failed to create %s: %w", lockFilePath, err)
		}
		slog.Debug("created lockfile", "path", result.Name())
		return result, nil
	}
}

// removeLockFile removes the temporary lockFile
func removeLockfile(lockFile *os.File) error {
	err := os.Remove(lockFile.Name())
	if err != nil {
		return err
	}
	return nil
}

/*
 * find commands
 *      configuration of subproc calls to the "find" executable
 */

// findCmd configures a subprocess for running find.
func findCmd(args []string) ([]string, error) {
	var result []string
	prg := []string{"find"}
	prg = append(prg, args...)
	cfg := subproc.New(prg)
	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr
	if err := cfg.Run(); err != nil {
		return []string{}, fmt.Errorf("%s: %w, %s", cfg.Args, err, strings.TrimSpace(stderr.String()))
	}
	if out := strings.TrimSpace(stdout.String()); out != "" {
		result = strings.Split(out, "\n")
	}
	return result, nil
}

// findModules wraps the "find" command in a transparent os/exec wrapper
// and configures the args in order to find the
func findModules(searchPath string) ([]string, error) {
	var result []string

	modulePath := path.Join(searchPath, "modules")

	// files
	files, ferr := findCmd([]string{modulePath, "-type", "f"})
	if ferr != nil {
		return []string{}, ferr
	}
	result = append(result, files...)

	// symlinks
	lns, lnerr := findCmd([]string{modulePath, "-type", "l"})
	if lnerr != nil {
		return []string{}, lnerr
	}
	result = append(result, lns...)

	return result, nil
}

// findSoftware wraps the "find" command in a transparent os/exec
// wrapper and uses it to find software subdirectories along the
// searchPath. Returns an array of the found paths as strings and error.
func findSoftware(searchPath string) ([]string, error) {
	var result []string

	pattern := path.Join(searchPath, "software", "*", "*")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob error for %q: %w", pattern, err)
	}
	if len(matches) == 0 {
		return result, nil
	}

	// build the find args
	// find <match1> ... <matchN> -maxdepth 1 -name easybuild -type d
	// easybuild dirs
	args := append(matches, "-maxdepth", "1", "-name", "easybuild", "-type", "d")

	dirs, err := findCmd(args)
	if err != nil {
		return []string{}, err
	}
	for _, easyBuildDir := range dirs {
		p := filepath.Dir(filepath.Clean(easyBuildDir))
		result = append(result, p)
	}
	return result, nil
}

// newListFile creates a temporary files.list.txt in the OS temp dir.
// Deliberately NOT in the tar working dir: a list file left next to the
// work tree can be captured by tar (and pushed through py-auto-ingest),
// which is how dozens of files.list.txt<N> files ended up on the stratum.
func newListFile() (*os.File, error) {
	file, err := os.CreateTemp("", "files.list.txt")
	if err != nil {
		return nil, fmt.Errorf("creating list file: %w", err)
	}
	return file, nil
}

// searchRoots returns the list of <archDir>/.. roots to scan for built
// content: the CPU arch dir always, plus the accelerator subdir hanging off
// it when accelSubdir is set (EESSI layout: software/linux/<arch>/accel/
// <vendor>/<cc>/{modules,software}).
func searchRoots(repo, version, archSubdir, accelSubdir string) []string {
	arch := archDir(repo, version, archSubdir)
	roots := []string{arch}
	if accelSubdir != "" {
		roots = append(roots, path.Join(arch, accelSubdir))
	}
	return roots
}

// MakeListFile collects any visible paths along the "software" and
// "modules" subdirectories of the overlayfs. A visible path in this
// context will equate to a software/module combination, or set of
// combinations that exist after an easybuild command has succeeded in
// building software into the overlay filesystem.
//
// The scan covers the CPU arch dir and (when accelSubdir is set) the
// accelerator subdir below it. Each "modules"/"software" subtree is only
// scanned when present in the overlay: an accelerator build may touch only
// the accel tree and a CPU-only build only the CPU tree, so a missing
// subtree is skipped (as in EESSI's create_tarball.sh) rather than treated
// as an error. Finding nothing at all is an error.
func MakeListFile(repo, version, archSubdir, accelSubdir string) (*os.File, error) {

	roots := searchRoots(repo, version, archSubdir, accelSubdir)

	// file list for the tarball
	var fileList []string

	for _, root := range roots {
		if _, err := os.Stat(path.Join(root, "modules")); err == nil {
			modules, err := findModules(root)
			if err != nil {
				return nil, fmt.Errorf("finding modules under %s: %w", root, err)
			}
			fileList = append(fileList, modules...)
		} else {
			slog.Debug("no modules subtree, skipping", "dir", root)
		}

		if _, err := os.Stat(path.Join(root, "software")); err == nil {
			software, err := findSoftware(root)
			if err != nil {
				return nil, fmt.Errorf("finding software under %s: %w", root, err)
			}
			fileList = append(fileList, software...)
		} else {
			slog.Debug("no software subtree, skipping", "dir", root)
		}
	}

	if len(fileList) == 0 {
		return nil, fmt.Errorf("nothing to pack: no built modules or software found under %s", strings.Join(roots, ", "))
	}

	workdir := versionsDir(repo)
	tmpfile, err := newListFile()
	if err != nil {
		return nil, err
	}
	// members must be RELATIVE to workdir (the tar -C dir): py-auto-ingest
	// plants the tarball under <repo>/versions via cvmfs_server ingest
	// -b versions, and re-roots anything else under that base, burying the
	// payload. Error hard on any path that escapes the working dir.
	wdPrefix := workdir + string(os.PathSeparator)
	for i, s := range fileList {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !strings.HasPrefix(s, wdPrefix) {
			tmpfile.Close()
			return nil, fmt.Errorf("collected path %q escapes tar working dir %s", s, workdir)
		}
		fileList[i] = filepath.ToSlash(strings.TrimPrefix(s, wdPrefix))
	}

	// write any files we've found
	writer := bufio.NewWriter(tmpfile)
	for _, s := range fileList {
		if s == "" {
			continue
		}
		if _, err := writer.WriteString(s + "\n"); err != nil {
			tmpfile.Close()
			return nil, fmt.Errorf("writing to temp file %s: %w", tmpfile.Name(), err)
		}
	}
	// flush buffer
	if err := writer.Flush(); err != nil {
		tmpfile.Close()
		return nil, fmt.Errorf("flushing temp file %s: %w", tmpfile.Name(), err)
	}

	// ensure data on disk
	if err := tmpfile.Sync(); err != nil {
		tmpfile.Close()
		return nil, fmt.Errorf("syncing temp file %s: %w", tmpfile.Name(), err)
	}
	return tmpfile, nil
}
