package easybuild

import "testing"

// TestEbVersionFromPath covers the anchor behavior of EbVersionFromPath.
// Conforming filenames follow the schema from
// asc-software-layer/easystacks/readme.md: <prefix>_eb_<version>-<toolchain>.yaml
// where the project prefix (asc, ...) is optional.
func TestEbVersionFromPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		// conforming names from asc-software-layer/easystacks/
		{"plain toolchain", "asc_eb_5.2.1-2023a.yaml", "5.2.1"},
		{"intel toolchain", "asc_eb_5.1.2-intel-2023b.yaml", "5.1.2"},
		{"cuda toolchain", "asc_eb_5.2.0-2024a-CUDA-12.6.0.yaml", "5.2.0"},
		{"nvhpc toolchain", "asc_eb_5.3.0-nvidia-nvhpc-25.9.yaml", "5.3.0"},
		{"system toolchain", "asc_eb_5.4.0-system.yaml", "5.4.0"},

		// the project prefix is not necessarily "asc"
		{"no prefix", "eb_5.4.0-system.yaml", "5.4.0"},
		{"underscore prefix", "myorg_eb_5.1.2-intel-2025a.yaml", "5.1.2"},
		{"dash prefix", "myorg-eb_5.1.2-intel-2025a.yaml", "5.1.2"},

		// callers may pass full paths; the filename governs the version
		{"absolute path", "/srv/asc-software-layer/easystacks/2025.06/asc_eb_5.4.0-2025a.yaml", "5.4.0"},
		{"relative path", "easystacks/2023.06/asc_eb_5.2.1-2023a.yaml", "5.2.1"},
		{"eb_ marker in a directory component", "/srv/releases/eb_4.9.4/asc_eb_5.2.1-2023a.yaml", "5.2.1"},
		{"versioned marker in a directory component", "/srv/eb_4.9.4-builds/asc_eb_5.2.1-2023a.yaml", "5.2.1"},

		// drifted / foreign names must not extract a version
		{"underscore separator drift", "asc_eb_5.1.0_0_system.yaml", ""},
		{"missing toolchain", "asc_eb_5.2.1.yaml", ""},
		{"two-component version", "asc_eb_5.2-2023a.yaml", ""},
		{"cuda_samples", "cuda_samples.yaml", ""},
		{"nvofbf", "nvofbf.yaml", ""},
		{"test.yaml", "test.yaml", ""},
		{"false marker rgb_", "rgb_5.2.1-2023a.yaml", ""},
		{"false marker web_", "web_5.2.1-2023a.yaml", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EbVersionFromPath(tt.path); got != tt.want {
				t.Errorf("EbVersionFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestEbModuleName(t *testing.T) {
	// the EESSI-extend module is spelled "EasyBuild"; module lookup is
	// case-sensitive, so this doubles as a guard against mis-casing.
	if got := EbModuleName("5.2.1"); got != "EasyBuild/5.2.1" {
		t.Errorf("EbModuleName(%q) = %q, want %q", "5.2.1", got, "EasyBuild/5.2.1")
	}
}

func TestNewEasystack(t *testing.T) {
	t.Run("conforming path", func(t *testing.T) {
		es, err := NewEasystack("asc_eb_5.2.1-2023a.yaml")
		if err != nil {
			t.Fatalf("NewEasystack returned unexpected error: %v", err)
		}
		if es.Path != "asc_eb_5.2.1-2023a.yaml" {
			t.Errorf("Path = %q, want %q", es.Path, "asc_eb_5.2.1-2023a.yaml")
		}
		if es.EbVersion != "EasyBuild/5.2.1" {
			t.Errorf("EbVersion = %q, want %q", es.EbVersion, "EasyBuild/5.2.1")
		}
	})

	t.Run("non-conforming path errors", func(t *testing.T) {
		for _, path := range []string{
			"asc_eb_5.1.0_0_system.yaml",
			"cuda_samples.yaml",
			"asc_eb_5.2.1.yaml",
		} {
			if es, err := NewEasystack(path); err == nil {
				t.Errorf("NewEasystack(%q) = %+v, nil; want error", path, es)
			}
		}
	})
}
