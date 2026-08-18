package cvmfs

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"
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
	// user supplied target files take precedence over changed files in the repo
	if len(opts.Files) > 0 {
		cmdData.Easystacks = opts.Files
	}
	return cmdData
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
