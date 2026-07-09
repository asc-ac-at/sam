package cvmfs

import (
	"fmt"
	"html/template"
	"os"
	"time"
)

type CvmfsBuildCmdData struct {
	CpuMarch   string
	GpuMarch   string
	SwsVariant string
	GitBranch  string
	GitRepo    string
	Easystack  string
	Publish    bool
	LmodInit   string
	CvmfsRepo  string
	Template   string
	Name       string
	Timestamp  string
}

func timestamp() string {
	t := time.Now()
	return fmt.Sprint(t.Format("20060102150405"))
}

func NewCvmfsBuildCmdData() *CvmfsBuildCmdData {
	cmdData := &CvmfsBuildCmdData{
		Publish:   false,
		GitRepo:   "https://gitlab.tuwien.ac.at/vsc/software-stacks/asc-software-layer",
		CvmfsRepo: "/cvmfs/software.asc.ac.at",
		Template:  buildCmdTmpl,
		Name:      "my-software",
		Timestamp: timestamp(),
	}
	return cmdData
}

const buildCmdTmpl = `#!/usr/bin/env bash

stack_file="{{ .Easystack }}"
if [ ! -f ${stack_file} ]; then
    printf "ERR - file not found ${stack_file}"
    exit 1
fi

source {{ .LmodInit }}
export EESSI_PROJECT_INSTALL={{ .CvmfsRepo }}

ml --force purge
ml load "EESSI/{{ .SwsVariant }}" "ASC/{{ .SwsVariant }}" \
    && ml load EESSI-extend || printf "ERR - module not found EESSI/{{ .SwsVariant }} ASC/{{ .SwsVariant }}\n"

{{if .Publish}}
eb -r --easystack ${stack_file}
if [[ "$?" -eq 0 ]]; then
    crtar -EESSI-version {{ .SwsVariant }} -name "{{ .Name }}-{{ .Timestamp }}"
else
    cp -a /tmp/ ${LOGDIR}/ctr-tmp
fi
{{- else}}
eb -r --easystack ${stack_file}
{{end}}
`

func renderBuildCmd(tmpl string, data *CvmfsBuildCmdData) error {
	t := template.Must(template.New("CvmfsBuildCmd").Parse(tmpl))
	err := t.Execute(os.Stdout, data)
	return err
}
