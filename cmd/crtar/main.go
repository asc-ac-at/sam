// SPDX-License-Identifier: GPL-2.0
/*
    (c) 2025 Adam McCartney <adam@mur.at>
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/asc-ac-at/sam/internal/crtar"
)

// args
var defaultSWSVersion = "2023.06"
var eessiVersionPtr = flag.String("EESSI-version", defaultSWSVersion, "Version of the (EEESI based) software stack")
var defaultCpuArchSubdir = "x86_64/amd/zen4"
var cpuArchSubdirPtr = flag.String("cpuArchSubdir", defaultCpuArchSubdir, "CPU Arch subdirectory to search")
var defaultName = "unnamed"
var namePtr = flag.String("name", defaultName, "Name of the tarball being created")
var outputDirPtr = flag.String("outputDir", "/opt/adm/sw-archives", "Output directory to save tarball")
var defaultRepo = "software.asc.ac.at"
var repoPtr = flag.String("repo", defaultRepo, "CVMFS repository for which the software was built")
var versionFlag = flag.Bool("version", false, "print version info")

var Version = "unknown"

func printVersion() {
	fmt.Printf("crtar version: %s\n", Version)
}

func main() {
	flag.Parse()
	if *versionFlag {
		printVersion()
		return
	}
	listFile, err := crtar.MakeListFile(*repoPtr, *eessiVersionPtr, *cpuArchSubdirPtr)
	if err != nil {
		log.Printf("error making listfile: %s, exiting", err)
		os.Exit(1)
	}

	_, execErr := crtar.ExecTar(*repoPtr, *cpuArchSubdirPtr, *namePtr, *outputDirPtr, listFile)
	if execErr != nil {
		log.Fatalf("execTar failed %s\n", execErr)
	}

}
