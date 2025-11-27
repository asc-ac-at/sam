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
)

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
		return nil, fmt.Errorf("could not acquire lockfile %s: %w", lockFile.Name(), lferr)
	}
	stdout, err := runCmd("tar", args)
	if err != nil {
		return nil, fmt.Errorf("creating tarball %s failed %w", tarball, err)
	}
	log.Printf("tarball %s created", tarball)
	removeLockfile(lockFile)
	return stdout, nil
}

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

func versionsDir(repo string) string {
	return path.Join(overlayUpperDir(repo), "/versions")
}

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
		return nil, fmt.Errorf("lockfile %s already present %w", lockFilePath, err)
	} else {
		result, err := os.Create(lockFilePath)
		log.Printf("aquireLockfile created -> %s\n", result.Name())
		if err != nil {
			return nil, fmt.Errorf("acquireLockfile failed to create %s: %w", lockFilePath, err)
		}
		return result, nil
	}
}

func removeLockfile(lockFile *os.File) error {
	err := os.Remove(lockFile.Name())
	if err != nil {
		return err
	}
	return nil
}

// Execute a find command in a subprocess
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

func findModules(searchPath string) ([]string, error) {
	var result []string

	modulePath := path.Join(searchPath, "modules")

	// files
	fArgs := []string{modulePath, "-type", "f"}
	fRes, err := runCmd("find", fArgs)
	if err != nil {
		log.Println("FindModules error: ", err)
		return result, err
	}
	result = append(result, fRes...)

	// symlinks
	sArgs := []string{modulePath, "-type", "l"}
	sRes, err := runCmd("find", sArgs)
	if err != nil {
		log.Println("FindModules symlink error: ", err)
		return result, err
	}
	result = append(result, sRes...)

	return result, nil
}

// Find
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

	var absPaths []string

	// build the find args
	// find <match1> ... <matchN> -maxdepth 1 -name easybuild -type d
	args := append([]string{}, absPaths...)
	// easybuild dirs
	args = append(args, "-maxdepth", "1", "-name", "easybuild", "-type", "d")

	sRes, err := runCmd("find", args)
	if err != nil {
		log.Println("FindSoftware error: ", err)
		return []string{}, err
	}

	for _, easyBuildDir := range sRes {
		p := filepath.Dir(filepath.Clean(easyBuildDir))

		result = append(result, p)
	}
	return result, nil
}

// create
func newListFile(workdir string) (*os.File, error) {
	file, err := os.CreateTemp(workdir, "files.list.txt")

	if err != nil {
		log.Println("error creating ListFile")
		return nil, err
	}
	return file, nil
}

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
