// SPDX-License-Identifier: GPL-2.0
/*
   (c) 2025 Adam McCartney <adam@mur.at>
*/
package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
)

// args

func initDefaults() map[string]string {
	defaults := make(map[string]string)
	defaults["stackver"] = "2025.06"
	defaults["name"] = ""
	defaults["toolchain"] = ""
	// only true as of v2025.06 in February 2026 ... this should be computed
	defaults["ebver"] = "5.2.0"
	defaults["gitrepo"] = ""
	defaults["ebopts"] = ""
	return defaults
}

func initOpts() map[string]*string {
	defaults := initDefaults()
	opts := make(map[string]*string)
	opts["stackver"] = flag.String("stackver", defaults["stackver"], "Version of the software stack release")
	opts["name"] = flag.String("name", defaults["name"], "Name of the software package being build")
	opts["toolchain"] = flag.String("toolchain", defaults["toolchain"], "easybuild toolchain being used to build the software")
	opts["ebver"] = flag.String("ebver", defaults["ebver"], "easybuild version being used to build")
	opts["gitrepo"] = flag.String("gitrepo", defaults["gitrepo"], "path to the checked out git repo containing easystack")
	opts["ebopts"] = flag.String("ebopts", defaults["ebopts"], "any extra options to pass to easybuild")
	flag.Parse()
	return opts
}

// eventually this would come from a file ?
func initConfig() map[string]string {
	config := make(map[string]string)
	config["lmodInit"] = "/opt/adm/asc-software-stack/asc-software-layer-scripts/init/lmod/bash"
	config["installDir"] = "/cvmfs/software.eessi.io"
	config["buildCmdTmpl"] = `
#!/usr/bin/env bash

stack_file="{{ .GitRepo }}/easystacks/{{ .StackVer }}/asc_eb_{{ .EbVer }}-{{ .Toolchain }}.yaml"
if [ ! -f ${stack_file} ]; then
    printf "ERR - file not found ${stack_file}"
    exit 1
fi

source {{ .LmodInit }}
export EESSI_PROJECT_INSTALL={{ .InstallDir }}
TS=$(date +%y%m%d%M%S)

ml --force purge
ml load "EESSI/{{ .StackVer }}" "ASC/{{ .StackVer }}" \
    && ml load EESSI-extend || printf "ERR - module not found EESSI/{{ .StackVer }} ASC/{{ .StackVer }}\n"

eb -r --easystack ${stack_file} "{{ .EbOpts }}" \
    && crtar -EESSI-version '{{ .StackVer }}' -name "{{ .Name }}-{{ .Toolchain }}-${TS}"
`
	return config
}

var Version = "unknown"
var versionFlag = flag.Bool("version", false, "print version info")

func printVersion() {
	fmt.Printf("samgx version: %s\n", Version)
}

type BuildCmdData struct {
	StackVer, Name, Toolchain, EbVer, GitRepo, EbOpts *string
	LmodInit, InstallDir                              string
}

func buildCmd(tmpl string, data BuildCmdData) error {
	t := template.Must(template.New("BuildCmd").Parse(tmpl))
	err := t.Execute(os.Stdout, data)
	return err
}

func main() {
	opts := initOpts()
	config := initConfig()
	if *versionFlag {
		printVersion()
		return
	}
	data := BuildCmdData{
		StackVer:   opts["stackver"],
		Name:       opts["name"],
		Toolchain:  opts["toolchain"],
		EbVer:      opts["ebver"],
		GitRepo:    opts["gitrepo"],
		EbOpts:     opts["ebopts"],
		LmodInit:   config["lmodInit"],
		InstallDir: config["installDir"],
	}
	err := buildCmd(config["buildCmdTmpl"], data)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
}
