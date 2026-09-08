package build

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	easybuild "github.com/asc-ac-at/sam/internal/sami"
	"github.com/asc-ac-at/sam/internal/sami/cli/shared"
	"github.com/asc-ac-at/sam/internal/sami/config"
)

// CvmfsBuildCmdData holds all required data to create a build_cmd.sh script
// In particular, the script will run using _one_ Easystack file at a time.
type CvmfsBuildCmdData struct {
	ArchSubdir  string
	AccelSubdir string
	OutputDir   string
	RGW         bool
	RGWBucket   string
	RGWEndpoint string
	SWSVariant  string
	Easystacks  []*easybuild.Easystack
	Publish     bool
	LmodInit    string
	CvmfsRepo   string
	Template    string
	Name        string
	Logdir      string
	Owner       string
	Group       string
}

// NewCvmfsBuildCmdData creates a structure with
func NewCvmfsBuildCmdData(opts *shared.Options) (*CvmfsBuildCmdData, error) {
	cmdData := &CvmfsBuildCmdData{
		SWSVariant: opts.SWSVariant,
		Publish:    false,
		LmodInit:   filepath.Join("/cvmfs/software.eessi.io/versions", opts.SWSVariant, "init/lmod/sh"),
		CvmfsRepo:  "/cvmfs/software.asc.ac.at",
		Template:   buildCmdTmpl,
		Name:       opts.Name,
		Logdir:     opts.BuildLogBasePath,
		Owner:      opts.Owner,
		Group:      opts.Group,
	}
	// user supplied target files take precedence over changed files in the repo
	if len(opts.Files) > 0 {
		var estacks []*easybuild.Easystack
		for _, f := range opts.Files {
			es, err := easybuild.NewEasystack(f)
			if err != nil {
				return nil, fmt.Errorf(`easybuild.NewEasystack(%q) failed: %w`, f, err)
			}
			estacks = append(estacks, es)
		}
		cmdData.Easystacks = estacks
	}
	return cmdData, nil
}

// resolveSubdirs maps the --arch / --accel / --generic inputs to EESSI
// subdirectories for crtar. arch/accel names are resolved through the
// mapping tables in the sami config. Returns (archSubdir, accelSubdir).
// accelSubdir is empty for CPU-only builds.
func resolveSubdirs(cfg *config.File, arch, accel string) (string, string, error) {
	var archSubdir string
	var err error
	archSubdir, err = cfg.ArchSubdir(arch)
	if err != nil {
		return "", "", err
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
