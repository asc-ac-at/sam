package cvmfs

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/config"
)

// CvmfsBuildCmdData holds all required data to create a build_cmd.sh script
// In particular, the script will run using _one_ Easystack file at a time.
type CvmfsBuildCmdData struct {
	ArchSubdir  string
	AccelSubdir string
	SWSVariant  string
	Easystacks  []string
	Publish     bool
	LmodInit    string
	CvmfsRepo   string
	Template    string
	Name        string
	Timestamp   string
}

func timestamp() string {
	t := time.Now()
	return fmt.Sprint(t.Format("20060102150405"))
}

// NewCvmfsBuildCmdData creates a structure with
func NewCvmfsBuildCmdData(opts *shared.Options) *CvmfsBuildCmdData {
	cmdData := &CvmfsBuildCmdData{
		SWSVariant: opts.SWSVariant,
		Publish:    false,
		LmodInit:   filepath.Join("/cvmfs/software.eessi.io/versions", opts.SWSVariant, "init/lmod/sh"),
		CvmfsRepo:  "/cvmfs/software.asc.ac.at",
		Template:   buildCmdTmpl,
		Name:       "my-software",
		Timestamp:  timestamp(),
	}
	// user supplied target files take precedence over changed files in the repo
	if len(opts.Files) > 0 {
		cmdData.Easystacks = opts.Files
	}
	return cmdData
}

// resolveSubdirs maps the --arch / --accel / --generic inputs to EESSI
// subdirectories for crtar. generic targets x86_64/generic directly and
// needs no config lookup; arch/accel names are resolved through the
// mapping tables in the sami config. Returns (archSubdir, accelSubdir).
// accelSubdir is empty for CPU-only builds.
func resolveSubdirs(cfg *config.File, arch, accel string, generic bool) (string, string, error) {
	if generic && arch != "" {
		return "", "", errors.New("--generic and --arch are mutually exclusive")
	}
	if !generic && arch == "" {
		return "", "", errors.New("--arch (or --generic) is required when publishing")
	}
	var archSubdir string
	if generic {
		archSubdir = config.GenericArchSubdir
	} else {
		var err error
		archSubdir, err = cfg.ArchSubdir(arch)
		if err != nil {
			return "", "", err
		}
	}
	if accel == "" {
		return archSubdir, "", nil
	}
	accelSubdir, err := cfg.AccelSubdir(accel)
	if err != nil {
		return "", "", err
	}
	return archSubdir, accelSubdir, nil
}

//go:embed buildcmd.tmpl
var buildCmdTmpl string

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
