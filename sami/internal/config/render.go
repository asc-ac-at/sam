package config

import (
	"embed"
	"fmt"
	"io"
	"log"
	"sync"
	"text/template"
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
	sbatchHeadersTmpl = &parsedTemplate{name: "templates/sbatch-headers.tmpl"}
	sbatchScriptTmpl  = &parsedTemplate{name: "templates/sbatch-script.tmpl"}
)

// sbatchTemplate returns the cached sbatch-headers template.
func sbatchTemplate() (*template.Template, error) {
	return sbatchHeadersTmpl.get()
}

func SbatchTemplate() (*template.Template, error) {
	return sbatchTemplate()
}

func RenderSbatchHeaders(cfg *SbatchConfig, part string, w io.Writer) error {

	tmpl, err := sbatchTemplate()
	if err != nil {
		log.Println("sbatch template not found")
		return err
	}

	data, err := cfg.SbatchHeaders(part)
	if err != nil {
		return fmt.Errorf("fetching partition config for %q: %w", part, err)
	}

	if err := tmpl.ExecuteTemplate(w, "sbatch-headers.tmpl", data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	return nil
}

// SbatchScriptData carries the pre-rendered pieces of a submit script.
type SbatchScriptData struct {
	// SbatchHeaders is the rendered output of RenderSbatchHeaders.
	SbatchHeaders string
	// BuildCmd is the rendered build-command payload, package-local to the
	// caller (e.g. cvmfs buildcmd.tmpl or a host_injections template).
	BuildCmd string
}

// RenderSbatchScript composes a submit script from a set of pre-rendered
// sbatch headers followed by a pre-rendered build-command payload. The
// ordering matters: sbatch only reads #SBATCH directives that precede the
// first executable line.
func RenderSbatchScript(data SbatchScriptData, w io.Writer) error {
	tmpl, err := sbatchScriptTmpl.get()
	if err != nil {
		return fmt.Errorf("parsing sbatch script template: %w", err)
	}
	if err := tmpl.ExecuteTemplate(w, "sbatch-script.tmpl", data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}
	return nil
}
