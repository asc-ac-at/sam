// Package sbatch renders sbatch submit scripts from slurm configuration
// (internal/config) and caller-supplied build payloads. It is intentionally
// general: any command that wants to run a payload under sbatch renders its
// own payload, renders headers here, and composes them with RenderScript.
package sbatch

import (
	"embed"
	"fmt"
	"io"
	"sync"
	"text/template"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/config"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging/buildlog"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// parsedTemplate caches a lazily parsed template. Parsing happens exactly once
// on first use; template.Template is safe for concurrent Execute after Parse,
// so the single copy is shared across all callers.
type parsedTemplate struct {
	name string
	once sync.Once
	tmpl *template.Template
	err  error
}

func (p *parsedTemplate) get() (*template.Template, error) {
	p.once.Do(func() {
		p.tmpl, p.err = template.ParseFS(templateFS, p.name)
	})
	return p.tmpl, p.err
}

var (
	headersTmpl = &parsedTemplate{name: "templates/sbatch-headers.tmpl"}
	scriptTmpl  = &parsedTemplate{name: "templates/sbatch-script.tmpl"}
)

// ScriptData carries the pre-rendered pieces of a submit script.
type ScriptData struct {
	// Headers is the rendered output of RenderHeaders.
	Headers string
	// BuildCmdPath is the path to the rendered build-command script (e.g.
	// $LOGDIR/build_cmd.sh), redirect-fed into the container's shell.
	BuildCmdPath string
}

// RenderHeaders renders the #SBATCH directive block for partition. The whole
// block must precede any executable line in the final submit script, which
// is what RenderScript guarantees.
func RenderHeaders(cfg *config.SbatchConfig, part string, blPath *buildlog.BuildLogPaths, w io.Writer) error {
	tmpl, err := headersTmpl.get()
	if err != nil {
		return fmt.Errorf("parsing sbatch headers template: %w", err)
	}
	data, err := cfg.SbatchHeaders(part, blPath)
	if err != nil {
		return fmt.Errorf("fetching partition config for %q: %w", part, err)
	}
	if err := tmpl.ExecuteTemplate(w, "sbatch-headers.tmpl", data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	return nil
}

// RenderScript composes a submit script from pre-rendered sbatch headers
// followed by a pre-rendered build-command payload. The ordering matters:
// sbatch only reads #SBATCH directives that precede the first executable
// line.
func RenderScript(data ScriptData, w io.Writer) error {
	tmpl, err := scriptTmpl.get()
	if err != nil {
		return fmt.Errorf("parsing sbatch script template: %w", err)
	}
	if err := tmpl.ExecuteTemplate(w, "sbatch-script.tmpl", data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	return nil
}
