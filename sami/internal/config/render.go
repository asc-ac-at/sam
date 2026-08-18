package config

import (
	"embed"
	"fmt"
	"io"
	"log"
	"sync"
	"text/template"
)

//go:embed templates/sbatch-headers.tmpl
var templateFS embed.FS

var (
	sbatchTmplOnce sync.Once
	sbatchTmpl     *template.Template
	sbatchTmplErr  error
)

// sbatchTemplate parses the embedded template exactly once and reuses it for
// every render. template.Template is safe for concurrent Execute after Parse,
// so a single package-level copy is shared across all callers.
func sbatchTemplate() (*template.Template, error) {
	sbatchTmplOnce.Do(func() {
		sbatchTmpl, sbatchTmplErr = template.ParseFS(templateFS, "templates/sbatch-headers.tmpl")
	})
	return sbatchTmpl, sbatchTmplErr
}

func SbatchTemplate() (*template.Template, error) {
	return sbatchTemplate()
}

func RenderSbatchHeaders(cfg *SbatchConfig, part string, w io.Writer) error {

	tmpl, err := SbatchTemplate()
	if err != nil {
		log.Println("sbatch template not found")
		return err
	}

	data, err := cfg.SbatchHeaders(part)
	if err != nil {
		return fmt.Errorf("fetching partition config for %q: %w", part, err)
	}

	if err := tmpl.ExecuteTemplate(w, "sbatch-headers.tmpl", data); err != nil {
		return fmt.Errorf("Error executing template: %v", err)
	}
	return nil
}
