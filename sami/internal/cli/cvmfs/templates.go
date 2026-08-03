package cvmfs

import (
	"fmt"
	"html/template"
	"os"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

// CvmfsBuildCmdData holds all required data to create a build_cmd.sh script
// In particular, the script will run using _one_ Easystack file at a time.
type CvmfsBuildCmdData struct {
	CPU        string
	GPU        string
	Arch       string
	SWSVariant string
	Easystacks []string
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

// NewCvmfsBuildCmdData creates a structure with
func NewCvmfsBuildCmdData(opts *shared.Options) *CvmfsBuildCmdData {
	cmdData := &CvmfsBuildCmdData{
		CPU:        opts.CPU,
		GPU:        opts.GPU,
		SWSVariant: opts.SWSVariant,
		Publish:    false,
		CvmfsRepo:  "/cvmfs/software.asc.ac.at",
		Template:   buildCmdTmpl,
		Name:       "my-software",
		Timestamp:  timestamp(),
	}
	if opts.GPU != "" {
		cmdData.Arch = opts.CPU + "-" + opts.GPU
	} else {
		cmdData.Arch = opts.CPU
	}
	return cmdData
}

const buildCmdTmpl = `#!/usr/bin/env bash

# General setup
source {{ .LmodInit }}
export EESSI_PROJECT_INSTALL={{ .CvmfsRepo }}

ml --force purge
ml load "EESSI/{{ .SWSVariant }}" "ASC/{{ .SWSVariant }}" \
    && ml load EESSI-extend || printf "ERR - module not found EESSI/{{ .SWSVariant }} ASC/{{ .SWSVariant }}"

{{ range .Easystacks }}
echo "Building easystack: {{ . }}"
stack_file="{{ . }}"
if [ ! -f ${stack_file} ]; then
    printf "ERR - file not found ${stack_file}"
    exit 1
fi

eb -r --easystack {{ . }}
if [[ "$?" -ne 0 ]]; then
    printf "ERR - easybuild failed for ${stack_file}"
	exit 1
fi
{{end}}

{{if .Publish}}
if [[ "$?" -eq 0 ]]; then
    crtar -EESSI-version {{ .SWSVariant }} -name "{{ .Name }}-{{ .Arch }}-{{ .Timestamp }}"
else
    cp -a /tmp/ ${LOGDIR}/ctr-tmp
fi
{{end}}

`

func renderBuildCmd(tmpl string, data *CvmfsBuildCmdData, buildCmd string) error {
	t := template.Must(template.New("CvmfsBuildCmd").Parse(tmpl))
	bcmd, err := os.Create(buildCmd)
	if err != nil {
		return err
	}
	if err = t.Execute(bcmd, data); err != nil {
		return err
	}
	return nil
}
