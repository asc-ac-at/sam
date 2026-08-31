package build

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/config"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
)

func TestNewCvmfsBuildCmdData(t *testing.T) {
	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)

	if data == nil {
		t.Fatal("NewCvmfsBuildCmdData returned nil")
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
	if data.Name != opts.Name {
		t.Errorf("Name = %q, want %q (from opts)", data.Name, opts.Name)
	}
	if data.Template != buildCmdTmpl {
		t.Error("Template should be buildCmdTmpl")
	}
	if data.Logdir != opts.BuildLogBasePath {
		t.Errorf("Logdir = %q, want %q", data.Logdir, opts.BuildLogBasePath)
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
	opts := optsForTest()
	opts.SWSVariant = "2026.01"

	data := NewCvmfsBuildCmdData(opts)

	if data.SWSVariant != "2026.01" {
		t.Errorf("SWSVariant = %q, want %q", data.SWSVariant, "2026.01")
	}
}

func TestRenderBuildCmd_WritesFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	opts.Files = []string{"asc_eb_5.2.1-system-CUDA-12.9.1.yaml", "asc_eb_5.3.0-system.yaml"}
	data := NewCvmfsBuildCmdData(opts)
	data.Publish = true
	data.ArchSubdir = "x86_64/amd/zen4"
	data.AccelSubdir = "accel/nvidia/cc90"

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
	if !strings.Contains(got, "--arch-subdir "+data.ArchSubdir) {
		t.Errorf("rendered output should contain arch subdir %q, got: %q", data.ArchSubdir, got)
	}
	if !strings.Contains(got, "--accel-subdir "+data.AccelSubdir) {
		t.Errorf("rendered output should contain accel subdir %q, got: %q", data.AccelSubdir, got)
	}
	if !strings.Contains(got, data.SWSVariant) {
		t.Errorf("rendered output should contain SWSVariant %q", data.SWSVariant)
	}
	if !strings.Contains(got, "eb -r --easystack") {
		t.Errorf("rendered output should contain eb command, got: %q", got)
	}
}

// TestRenderBuildCmd_HermeticUserNamespace asserts the rendered build script
// unsets EESSI_USER_INSTALL so EasyBuild never resolves dependencies against
// the builder's personal user install ($HOME/eessi) - see sami TODO-dd222a56.
func TestRenderBuildCmd_HermeticUserNamespace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	data := NewCvmfsBuildCmdData(optsForTest())
	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	if err := renderBuildCmd(buildCmdTmpl, data, outFile); err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read rendered file: %v", err)
	}
	if !strings.Contains(string(content), "unset EESSI_USER_INSTALL") {
		t.Errorf("rendered output should unset EESSI_USER_INSTALL, got:\n%s", string(content))
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

func TestRenderBuildCmd_PublishCPUOnly(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)
	data.Publish = true
	data.ArchSubdir = "x86_64/amd/zen4"
	data.AccelSubdir = ""

	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	err = renderBuildCmd(buildCmdTmpl, data, outFile)
	if err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}

	content, _ := os.ReadFile(outFile)
	got := string(content)
	if !strings.Contains(got, "--arch-subdir "+data.ArchSubdir) {
		t.Errorf("rendered output should contain arch subdir %q, got: %q", data.ArchSubdir, got)
	}
	if strings.Contains(got, "--accel-subdir") {
		t.Errorf("rendered output should not contain --accel-subdir for CPU-only build, got: %q", got)
	}
}

func TestRenderBuildCmd_PublishLogdir(t *testing.T) {
	render := func(t *testing.T, data *CvmfsBuildCmdData) string {
		t.Helper()
		outFile := filepath.Join(t.TempDir(), "build_cmd.sh")
		if err := renderBuildCmd(buildCmdTmpl, data, outFile); err != nil {
			t.Fatalf("renderBuildCmd failed: %v", err)
		}
		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("Failed to read rendered file: %v", err)
		}
		return string(content)
	}

	t.Run("concrete logdir in failure branch", func(t *testing.T) {
		opts := optsForTest()
		data := NewCvmfsBuildCmdData(opts)
		data.Publish = true
		data.ArchSubdir = "x86_64/amd/zen4"

		got := render(t, data)

		want := "cp -a /tmp/ " + data.Logdir + "/ctr-tmp"
		if !strings.Contains(got, want) {
			t.Errorf("rendered output should contain %q, got: %q", want, got)
		}
		if strings.Contains(got, "${LOGDIR}") {
			t.Errorf("rendered output should not rely on the ${LOGDIR} env var, got: %q", got)
		}
		if strings.Contains(got, "{{") {
			t.Errorf("rendered output should not contain unresolved template placeholders, got: %q", got)
		}
	})

	// Pins the current render for an empty --log-basepath: the failure
	// branch produces the degenerate absolute path /ctr-tmp. If flag
	// validation starts rejecting an empty base path (TODO-2d836b56
	// item 3), update or drop this subtest.
	t.Run("empty basepath renders degenerate path", func(t *testing.T) {
		opts := optsForTest()
		opts.BuildLogBasePath = ""
		data := NewCvmfsBuildCmdData(opts)
		data.Publish = true
		data.ArchSubdir = "x86_64/amd/zen4"

		got := render(t, data)

		if data.Logdir != "" {
			t.Fatalf("expected empty Logdir, got %q", data.Logdir)
		}
		if !strings.Contains(got, "cp -a /tmp/ /ctr-tmp") {
			t.Errorf("expected degenerate render %q, got: %q", "cp -a /tmp/ /ctr-tmp", got)
		}
	})
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

func TestRenderBuildCmd_LmodInit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)

	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	err = renderBuildCmd(buildCmdTmpl, data, outFile)
	if err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read rendered file: %v", err)
	}

	want := "/cvmfs/software.eessi.io/versions/" + opts.SWSVariant + "/init/lmod/sh"
	if !strings.Contains(string(content), "source "+want) {
		t.Errorf("rendered output should source lmod init %q\n got: %q", want, string(content))
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
		GitRepo:          "https://gitlab.tuwien.ac.at/vsc/software-stacks/asc-software-layer",
		SWSVariant:       "2025.06",
		Name:             "test-build",
		BuildLogBasePath: "/tmp/sami-test-logs",
	}
}

func TestRenderBuildCmd_PublishOutputDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)
	data.Publish = true
	data.ArchSubdir = "x86_64/amd/zen4"
	data.OutputDir = "/opt/adm/sam-archives"

	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	if err := renderBuildCmd(buildCmdTmpl, data, outFile); err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read rendered file: %v", err)
	}
	got := string(content)
	if !strings.Contains(got, "--outputDir /opt/adm/sam-archives") {
		t.Errorf("rendered output should pass the output dir to crtar, got: %q", got)
	}
	if !strings.Contains(got, "grep '^TARBALL='") {
		t.Errorf("rendered output should capture crtar's TARBALL= contract line, got: %q", got)
	}
	if strings.Contains(got, "rgw object put") {
		t.Errorf("rendered output should not contain the rgw upload when RGW is false, got: %q", got)
	}
}

func TestRenderBuildCmd_PublishRGW(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sami-render-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	opts := optsForTest()
	data := NewCvmfsBuildCmdData(opts)
	data.Publish = true
	data.ArchSubdir = "x86_64/amd/zen4"
	data.OutputDir = "/log/run/sami.xyz"
	data.RGW = true
	data.RGWBucket = "sam-archives"
	data.RGWEndpoint = "https://rgw.example.org"

	outFile := filepath.Join(tmpDir, "build_cmd.sh")
	if err := renderBuildCmd(buildCmdTmpl, data, outFile); err != nil {
		t.Fatalf("renderBuildCmd failed: %v", err)
	}

	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Failed to read rendered file: %v", err)
	}
	got := string(content)
	if !strings.Contains(got, `rgw object put sam-archives "$(basename "$tb")" "$tb"`) {
		t.Errorf("rendered output should contain the rgw upload line, got: %q", got)
	}
	if !strings.Contains(got, "export AWS_ENDPOINT_URL=https://rgw.example.org") {
		t.Errorf("rendered output should export the configured endpoint, got: %q", got)
	}
}

func TestNewCommand_RGWRequiresPublish(t *testing.T) {
	opts := optsForTest()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cmd := NewCommand(opts, logger)
	if err := cmd.Flags().Set("rgw", "true"); err != nil {
		t.Fatalf("setting --rgw: %v", err)
	}
	if err := cmd.PreRunE(cmd, nil); err == nil || !strings.Contains(err.Error(), "--rgw requires --publish") {
		t.Errorf("PreRunE with --rgw only = %v, want --rgw requires --publish", err)
	}

	cmd = NewCommand(opts, logger)
	if err := cmd.Flags().Set("rgw", "true"); err != nil {
		t.Fatalf("setting --rgw: %v", err)
	}
	if err := cmd.Flags().Set("publish", "true"); err != nil {
		t.Fatalf("setting --publish: %v", err)
	}
	if err := cmd.PreRunE(cmd, nil); err != nil {
		t.Errorf("PreRunE with --rgw --publish = %v, want nil", err)
	}
}

func TestResolveSubdirs(t *testing.T) {
	cfg := &config.File{
		ArchMapping:  map[string]string{"zen4": "x86_64/amd/zen4", "zen5": "x86_64/amd/zen5", "generic": "x86_64/generic"},
		AccelMapping: map[string]string{"cc80": "accel/nvidia/cc80", "cc90": "accel/nvidia/cc90"},
	}

	t.Run("arch hit", func(t *testing.T) {
		arch, accel, err := resolveSubdirs(cfg, "zen4", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if arch != "x86_64/amd/zen4" {
			t.Errorf("archSubdir = %q, want x86_64/amd/zen4", arch)
		}
		if accel != "" {
			t.Errorf("accelSubdir = %q, want empty (CPU-only build)", accel)
		}
	})

	t.Run("arch and accel hit", func(t *testing.T) {
		arch, accel, err := resolveSubdirs(cfg, "zen5", "cc90")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if arch != "x86_64/amd/zen5" {
			t.Errorf("archSubdir = %q, want x86_64/amd/zen5", arch)
		}
		if accel != "accel/nvidia/cc90" {
			t.Errorf("accelSubdir = %q, want accel/nvidia/cc90", accel)
		}
	})

	t.Run("generic arch via mapping", func(t *testing.T) {
		arch, accel, err := resolveSubdirs(cfg, "generic", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if arch != "x86_64/generic" {
			t.Errorf("archSubdir = %q, want x86_64/generic", arch)
		}
		if accel != "" {
			t.Errorf("accelSubdir = %q, want empty (generic is CPU-only)", accel)
		}
	})

	t.Run("unknown arch errors and lists valid", func(t *testing.T) {
		_, _, err := resolveSubdirs(cfg, "skylake", "")
		if err == nil {
			t.Error("expected error for unknown arch")
		}
		for _, want := range []string{"zen4", "zen5"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error should list valid arch %q, got: %v", want, err)
			}
		}
	})

	t.Run("unknown accel errors and lists valid", func(t *testing.T) {
		_, _, err := resolveSubdirs(cfg, "zen4", "gfx90a")
		if err == nil {
			t.Error("expected error for unknown accel")
		}
		for _, want := range []string{"cc80", "cc90"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error should list valid accel %q, got: %v", want, err)
			}
		}
	})
}
