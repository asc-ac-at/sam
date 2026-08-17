package cvmfs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

func TestNewCvmfsBuildCmdData(t *testing.T) {
	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)

	if data == nil {
		t.Fatal("NewCvmfsBuildCmdData returned nil")
	}

	if data.CPU != opts.CPU {
		t.Errorf("CPU = %q, want %q", data.CPU, opts.CPU)
	}
	if data.GPU != opts.GPU {
		t.Errorf("GPU = %q, want %q", data.GPU, opts.GPU)
	}
	if data.SWSVariant != opts.SWSVariant {
		t.Errorf("SWSVariant = %q, want %q", data.SWSVariant, opts.SWSVariant)
	}
	if data.Publish != false {
		t.Error("expected Publish default false")
	}
	if data.CvmfsRepo != "/cvmfs/software.asc.ac.at" {
		t.Errorf("unexpected CvmfsRepo: %q", data.CvmfsRepo)
	}
	if data.Name != "my-software" {
		t.Errorf("unexpected Name: %q", data.Name)
	}
	if data.Template != buildCmdTmpl {
		t.Error("Template should be buildCmdTmpl")
	}
}

func TestCvmfsBuildCmdData_Publish(t *testing.T) {
	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)

	data.Publish = true
	if !data.Publish {
		t.Error("Publish should be true after setting")
	}

	data.Publish = false
	if data.Publish {
		t.Error("Publish should be false after setting")
	}
}

func TestNewCvmfsBuildCmdData_DifferentOpts(t *testing.T) {
	tests := []struct {
		name    string
		cpu     string
		gpu     string
		sws     string
		wantCPU string
		wantGPU string
		wantSWS string
	}{
		{"zen4/h100", "zen4", "h100", "2025.06", "zen4", "h100", "2025.06"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := optsForTest()
			opts.CPU = tt.cpu
			opts.GPU = tt.gpu
			opts.SWSVariant = tt.sws

			data := NewCvmfsBuildCmdData(opts)

			if data.CPU != tt.wantCPU {
				t.Errorf("CPU = %q, want %q", data.CPU, tt.wantCPU)
			}
			if data.GPU != tt.wantGPU {
				t.Errorf("GPU = %q, want %q", data.GPU, tt.wantGPU)
			}
			if data.SWSVariant != tt.wantSWS {
				t.Errorf("SWSVariant = %q, want %q", data.SWSVariant, tt.wantSWS)
			}
		})
	}
}

func TestNewCvmfsBuildCmdData_NameHardcoded(t *testing.T) {
	opts := optsForTest()
	opts.Name = "custom-build"

	data := NewCvmfsBuildCmdData(opts)

	// Name is hardcoded in NewCvmfsBuildCmdData
	if data.Name != "my-software" {
		t.Errorf("Name is hardcoded to %q", data.Name)
	}
}

func TestRenderBuildCmd_WritesFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)
	data.Publish = true

	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	err = renderBuildCmd(buildCmdTmpl, data, outFile)
	if err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read rendered file: %v", err)
	}

	// Should contain template placeholders substituted
	got := string(content)
	if !strings.Contains(got, data.CPU) {
		t.Errorf("rendered output should contain CPU %q", data.CPU)
	}
	if !strings.Contains(got, data.GPU) {
		t.Errorf("rendered output should contain GPU %q", data.GPU)
	}
	if !strings.Contains(got, data.SWSVariant) {
		t.Errorf("rendered output should contain SWSVariant %q", data.SWSVariant)
	}
	if !strings.Contains(got, "eb -r --easystack") {
		t.Error("rendered output should contain eb command")
	}
}

func TestRenderBuildCmd_PublishTrue(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)
	data.Publish = true

	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	err = renderBuildCmd(buildCmdTmpl, data, outFile)
	if err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}

	content, _ := os.ReadFile(outFile)
	got := string(content)
	if !strings.Contains(got, "crtar") {
		t.Error("rendered output should contain crtar when Publish is true")
	}
}

func TestRenderBuildCmd_PublishFalse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)
	data.Publish = false

	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	err = renderBuildCmd(buildCmdTmpl, data, outFile)
	if err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}

	content, _ := os.ReadFile(outFile)
	got := string(content)
	if strings.Contains(got, "crtar") {
		t.Error("rendered output should not contain crtar when Publish is false")
	}
}

func TestRenderBuildCmd_InvalidPath(t *testing.T) {
	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)

	// Use a path where parent doesn't exist
	err := renderBuildCmd(buildCmdTmpl, data, "/nonexistent/dir/build_cmd.sh")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRenderBuildCmd_NonZeroSWS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	opts.SWSVariant = "2026.01"
	data := NewCvmfsBuildCmdData(opts)

	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	err = renderBuildCmd(buildCmdTmpl, data, outFile)
	if err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}

	content, _ := os.ReadFile(outFile)
	if !strings.Contains(string(content), "2026.01") {
		t.Error("rendered output should contain SWSVariant")
	}
}

func optsForTest() *shared.Options {
	return &shared.Options{
		CPU:        "zen4",
		GPU:        "h100",
		GitRepo:    "https://gitlab.tuwien.ac.at/vsc/software-stacks/asc-software-layer",
		SWSVariant: "2025.06",
		Name:       "test-build",
	}
}
