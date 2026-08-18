package config

import (
	"embed"
	"fmt"
	"io"
	"log"
	"text/template"
)

//go:embed templates/sbatch-headers.tmpl
var templateFS embed.FS

func SbatchTemplate() (*template.Template, error) {
	return template.ParseFS(templateFS, "templates/sbatch-headers.tmpl")
}

func RenderSbatchHeaders(cfg *SbatchConfig, part string, w io.Writer) error {

	tmpl, err := SbatchTemplate()
	if err != nil {
		log.Println("sbatch template not found")
		return err
	}

	data, err := cfg.SbatchHeaders(part)
	if err != nil {
		return fmt.Errorf("Error fetching partition config for part %s", part)
	}

	if err := tmpl.ExecuteTemplate(w, "sbatch-headers.tmpl", data); err != nil {
		return fmt.Errorf("Error executing template: %v", err)
	}
	return nil
}
