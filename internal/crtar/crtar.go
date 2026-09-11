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

// ExecTar constructs the tarball in three passes:
//   1. tar --exclude=*.wh.* -C ${TOPDIR} -cf ${TARBALL} --files-from=${ROOTS}
//   2. tar -rf ${TARBALL} -C ${TOPDIR} --no-recursion --files-from=${ANCESTORS}
//   3. gzip -9f ${TARBALL}
//
// Pass 1 carries the collected build roots exactly as crtar always has.
// Pass 2 appends their ancestor dirs as non-recursive members so the
// publisher never synthesizes them with sentinel uid/gid ((uid_t)-1 /
// nobody:nogroup — EOVERFLOW poison for all later write-path ops; see the
// withAncestors doc). Appending to a gzip stream is impossible, hence the
// explicit third pass.
//
// NOTE on exclusions: --exclude=.cvmfscatalog was dropped on 2026-09-11 after
// e2 (debug/repro-eoverflow/) proved a .cvmfscatalog member survives ingest
// and lands correctly. Gate markers now travel deliberately: the build
// recipe places the marker in-tree; crtar transports. The whiteout
// exclude stays (overlayfs guts never belong in a repo).
func ExecTar(repo, archSubdir, accelSubdir, name, outdir string, roots, ancestors *os.File, owner, group string) (string, error) {
	// the list files are single-use: consume them here and clean up, they
	// carry no value after the tarball exists
	defer os.Remove(roots.Name())
	defer os.Remove(ancestors.Name())

	workingDir := versionsDir(repo)
	tarball := tarballPath(archSubdir, accelSubdir, name, outdir)
	// uncompressed intermediate; gzip joins at the end (appending to a gzip
	// stream is impossible, hence pass 2 runs against the plain tar)
	plain := strings.TrimSuffix(tarball, ".tar.gz") + ".tar"

	ownArgs := []string{}
	// optionally change ownership of files being packed into tarball
	if (owner != "") || (group != "") {
		ownArgs = append(ownArgs, "--numeric-owner")
	}
	if owner != "" {
		ownArgs = append(ownArgs, fmt.Sprintf("--owner=%s", owner))
	}
	if group != "" {
		ownArgs = append(ownArgs, fmt.Sprintf("--group=%s", group))
	}

	lockFile, lferr := acquireLockfile(tarball)
	if lferr != nil {
		return "", fmt.Errorf("could not acquire lockfile: %w", lferr)
	}
	defer removeLockfile(lockFile)

	run := func(args []string) error {
		var stderr bytes.Buffer
		cfg := subproc.New(args)
		cfg.Stdout = io.Discard
		cfg.Stderr = &stderr
		if err := cfg.Run(); err != nil {
			return fmt.Errorf("%s failed %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return nil
	}

	pass1 := append([]string{"tar", "--exclude=*.wh.*", "-C", workingDir,
		"-cf", plain, "--files-from=" + roots.Name()}, ownArgs...)
	if err := run(pass1); err != nil {
		return "", fmt.Errorf("creating tarball %s failed: %w", plain, err)
	}

	// ancestors pass may be legitimately empty (e.g. synthetic fixtures);
	// appending zero members is fine but wastes a process, so skip it
	if info, err := ancestors.Stat(); err == nil && info.Size() > 0 {
		pass2 := append([]string{"tar", "-rf", plain, "-C", workingDir,
			"--no-recursion", "--files-from=" + ancestors.Name()}, ownArgs...)
		if err := run(pass2); err != nil {
			return "", fmt.Errorf("appending ancestor members to %s failed: %w", plain, err)
		}
	}

	if err := run([]string{"gzip", "-9f", plain}); err != nil {
		return "", fmt.Errorf("compressing %s failed: %w", plain, err)
	}

	// guard: gzip -f renames in place; fail loudly if the expected final
	// artifact is missing (never ship a silent half-built tarball)
	if _, err := os.Stat(tarball); err != nil {
		return "", fmt.Errorf("expected final tarball %s missing after gzip: %w", tarball, err)
	}

	slog.Info("created tarball", "path", tarball)
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

// withAncestors expands the collected paths with every ancestor directory
// between the tar working dir and each entry, deduped, parents first.
// Published catalog surgery (2026-09-11, debug/repro-eoverflow/, variants
// r1 vs r2) proved: members missing from the tar get synthesized by
// cvmfs_server ingest with uid/gid = (uid_t)-1 (rendered nobody:nogroup),
// and such rows poison every later write-path op through the publisher
// transaction (EOVERFLOW). Ancestors are emitted via a separate
// --no-recursion --files-from pass, never in the same list as the roots:
// single-list emission would recurse into the ancestors and duplicate
// every child member, which trips AddEntry's primary-key constraint
// (catalog_rw.cc:165 assert).
func withAncestors(workdir string, paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		seen[p] = struct{}{}
	}
	var chain []string
	for _, p := range paths {
		for d := path.Dir(p); d != workdir && d != "." && strings.HasPrefix(d, workdir); d = path.Dir(d) {
			if _, dup := seen[d]; dup {
				continue
			}
			seen[d] = struct{}{}
			chain = append(chain, d)
		}
	}
	return chain
}

// writeListFile renders collected paths (absolute at call time) relative to
// the tar working dir and writes them, one per line, into a fresh temp file.
// Members must be RELATIVE to workdir (the tar -C dir): py-auto-ingest
// plants the tarball under <repo>/versions via cvmfs_server ingest
// -b versions, and re-roots anything else under that base, burying the
// payload. Error hard on any path that escapes the working dir.
func writeMemberList(workdir string, paths []string) (*os.File, error) {
	tmpfile, err := newListFile()
	if err != nil {
		return nil, err
	}
	wdPrefix := workdir + string(os.PathSeparator)
	writer := bufio.NewWriter(tmpfile)
	for _, s := range paths {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !strings.HasPrefix(s, wdPrefix) {
			tmpfile.Close()
			return nil, fmt.Errorf("collected path %q escapes tar working dir %s", s, workdir)
		}
		rel := filepath.ToSlash(strings.TrimPrefix(s, wdPrefix))
		if _, err := writer.WriteString(rel + "\n"); err != nil {
			tmpfile.Close()
			return nil, fmt.Errorf("writing to temp file %s: %w", tmpfile.Name(), err)
		}
	}
	if err := writer.Flush(); err != nil {
		tmpfile.Close()
		return nil, fmt.Errorf("flushing temp file %s: %w", tmpfile.Name(), err)
	}
	if err := tmpfile.Sync(); err != nil {
		tmpfile.Close()
		return nil, fmt.Errorf("syncing temp file %s: %w", tmpfile.Name(), err)
	}
	return tmpfile, nil
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
//
// Returns two files: the roots list (the historical crtar behavior) and
// the ancestors list (their complete ancestor chains, for the separate
// --no-recursion pass in ExecTar — see withAncestors).
func MakeListFile(repo, version, archSubdir, accelSubdir string) (*os.File, *os.File, error) {

	roots := searchRoots(repo, version, archSubdir, accelSubdir)

	// file list for the tarball
	var fileList []string

	for _, root := range roots {
		if _, err := os.Stat(path.Join(root, "modules")); err == nil {
			modules, err := findModules(root)
			if err != nil {
				return nil, nil, fmt.Errorf("finding modules under %s: %w", root, err)
			}
			fileList = append(fileList, modules...)
		} else {
			slog.Debug("no modules subtree, skipping", "dir", root)
		}

		if _, err := os.Stat(path.Join(root, "software")); err == nil {
			software, err := findSoftware(root)
			if err != nil {
				return nil, nil, fmt.Errorf("finding software under %s: %w", root, err)
			}
			fileList = append(fileList, software...)
		} else {
			slog.Debug("no software subtree, skipping", "dir", root)
		}
	}

	if len(fileList) == 0 {
		return nil, nil, fmt.Errorf("nothing to pack: no built modules or software found under %s", strings.Join(roots, ", "))
	}

	workdir := versionsDir(repo)

	rootsFile, err := writeMemberList(workdir, fileList)
	if err != nil {
		return nil, nil, err
	}
	ancFile, err := writeMemberList(workdir, withAncestors(workdir, fileList))
	if err != nil {
		rootsFile.Close()
		os.Remove(rootsFile.Name())
		return nil, nil, err
	}
	return rootsFile, ancFile, nil
}
