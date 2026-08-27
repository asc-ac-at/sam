// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package crtar

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
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
// at the front of the args slice.
func ExecTar(repo, cpuArchSubdir, name, outdir string, listFile *os.File) ([]string, error) {
	var args []string
	// second exclude is redundant because of the filter below
	args = append(args, "--exclude=.cvmfscatalog", "--exclude=*.wh.*")
	workingDir := versionsDir(repo)
	args = append(args, "-C", workingDir)
	tarball := tarballPath(cpuArchSubdir, name, outdir)
	args = append(args, "-czf", tarball)
	filesFrom := fmt.Sprintf("--files-from=%s", listFile.Name())
	args = append(args, filesFrom)

	lockFile, lferr := acquireLockfile(tarball)
	if lferr != nil {
		return nil, fmt.Errorf("could not acquire lockfile: %w", lferr)
	}
	stdout, err := runCmd("tar", args)
	if err != nil {
		return nil, fmt.Errorf("creating tarball %s failed %w", tarball, err)
	}
	log.Printf("tarball %s created", tarball)
	removeLockfile(lockFile)
	return stdout, nil
}

// tarballPath constructs a filepath to the tarball that will be subsequently created.
// returns a string containing the absolute path
func tarballPath(cpuArchSubdir, name, outdir string) string {
	normalizedArchDir := strings.ReplaceAll(cpuArchSubdir, "/", "-")
	t := time.Now()
	ts := t.Format("20060102150405")
	result := fmt.Sprintf("%s/%s-%s-%s.tar.gz", outdir, name, normalizedArchDir, ts)
	log.Printf("tarballPath -> %s", result)
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
func archDir(repo string, version string, cpuArchSubdir string) string {
	versionsDir := versionsDir(repo)
	return path.Join(versionsDir, version, "software", "linux", cpuArchSubdir)
}

// Check for the presence of a lockfile
// Lockfiles are created in order to prevent race conditions whereby the
// ingestion service tries to read a partially written tarball
func acquireLockfile(tarballPath string) (*os.File, error) {
	name := strings.TrimRight(tarballPath, ".tar.gz")
	lf := fmt.Sprintf("%s.lock", name)
	lockFilePath := filepath.Clean(lf)
	log.Printf("acquireLockfile find or create -> %s", lockFilePath)

	if _, err := os.Stat(lockFilePath); err == nil { // lockfile found!
		return nil, fmt.Errorf("lockfile %s already present", lockFilePath)
	} else {
		result, err := os.Create(lockFilePath)
		if err != nil {
			return nil, fmt.Errorf("acquireLockfile failed to create %s: %w", lockFilePath, err)
		}
		log.Printf("aquireLockfile created -> %s\n", result.Name())
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

// runCmd is a wrapper around os/exec
// execute a find command in a subprocess
func runCmd(cmdName string, args []string) ([]string, error) {
	cmd := exec.Command(cmdName, args...)
	var spStdOut bytes.Buffer
	var spStdErr bytes.Buffer
	cmd.Stdout = &spStdOut
	cmd.Stderr = &spStdErr
	err := cmd.Run()
	if err != nil {
		log.Println("subprocess stderr: ", spStdErr.String())
		log.Println("error executing command: ", err)
		return []string{}, err
	}
	lines := strings.Split(spStdOut.String(), "\n")
	return lines, nil
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
	result = append(result, strings.TrimSpace(stdout.String()))
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

// newListFile creates a files.list.txt in workdir
func newListFile(workdir string) (*os.File, error) {
	file, err := os.CreateTemp(workdir, "files.list.txt")

	if err != nil {
		log.Println("error creating ListFile")
		return nil, err
	}
	return file, nil
}

// MakeListFile collects any visible paths along the "software" and
// "modules" subdirectories of the overlayfs. A visible path in this
// context will equate to a software/module combination, or set of
// combinations that exist after an easybuild command has succeeded in
// building software into the overlay filesystem.
func MakeListFile(repo, version, cpuArchSubdir string) (*os.File, error) {

	archDir := archDir(repo, version, cpuArchSubdir)

	// file list for the tarball
	var fileList []string

	modules, err := findModules(archDir)
	if err != nil {
		log.Println("Error finding modules: ", err)
		log.Println("exiting")
		os.Exit(1)
	}
	fileList = append(fileList, modules...)

	software, err := findSoftware(archDir)
	if err != nil {
		log.Println("Error finding software: ", err)
		log.Println("exiting")
		os.Exit(1)
	}

	fileList = append(fileList, software...)

	//for i := range(fileList) {
	//	fmt.Println(fileList[i])
	//}

	workdir := versionsDir(repo)
	tmpfile, err := newListFile(workdir)
	if err != nil {
		log.Fatalf("creating tmpfile in %s failed", workdir)
	}
	// write any files we've found
	writer := bufio.NewWriter(tmpfile)
	for _, s := range fileList {
		s = strings.TrimSpace(s)
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
