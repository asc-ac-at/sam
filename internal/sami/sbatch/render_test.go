package sbatch

import (
	"bytes"
	"strings"
	"testing"

	"github.com/asc-ac-at/sam/internal/sami/config"
	"github.com/asc-ac-at/sam/internal/sami/logging/buildlog"
)

// the committed canonical fixture
const fixturePath = "../test/config.yaml"

// testLogPaths is a fixture whose BuildLog determines the rendered --output
// path; the literal keeps expectations stable.
var testLogPaths = &buildlog.BuildLogPaths{BuildLog: "/logdir"}

func renderForTest(t *testing.T, partition string) (string, error) {
	t.Helper()
	f, err := config.LoadSbatchConfig(fixturePath)
	if err != nil {
		t.Fatalf("LoadSbatchConfig(fixture): %v", err)
	}
	var buf bytes.Buffer
	err = RenderHeaders(&f.Sbatch, partition, testLogPaths, &buf)
	return buf.String(), err
}

func TestRenderHeaders_GPUPartition(t *testing.T) {
	out, err := renderForTest(t, "zen4_gpu")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{
		"#sami --vanilla",
		"#SBATCH -p zen4_gpu",
		"#SBATCH --qos standard",
		"#SBATCH --ntasks 32",
		"#SBATCH --gres gpu:1",
		"#SBATCH --mem 256G",
		"#SBATCH --cpus-per-task=1",
		"#SBATCH --threads-per-core=1",
		// job-name and output are rendered from code, not config; these
		// literal lines are the contract fixed by the header refactor.
		"#SBATCH --job-name=sami",
		"#SBATCH --output=/logdir/slurm-%j.out",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing %q\n got: %q", want, out)
		}
	}
}

func TestRenderHeaders_CPUPartitionOmitsGres(t *testing.T) {
	out, err := renderForTest(t, "zen4_cpu")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "--gres") {
		t.Errorf("CPU partition should not render a gres line, got: %q", out)
	}
	if !strings.Contains(out, "#SBATCH -p zen4_cpu") {
		t.Errorf("rendered output missing partition line, got: %q", out)
	}
}

func TestRenderHeaders_EmptyHeaderOmitted(t *testing.T) {
	cfg := &config.SbatchConfig{
		Partitions: map[string]config.PartitionConfig{
			"p1": {
				Partition:      "p1",
				Qos:            "q1",
				Ntasks:         "8",
				Mem:            "32G",
				CpusPerTask:    1,
				ThreadsPerCore: 1,
			},
		},
	}
	var buf bytes.Buffer
	if err := RenderHeaders(cfg, "p1", testLogPaths, &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := buf.String()
	first, _, _ := strings.Cut(out, "\n")
	if first != "#SBATCH -p p1" {
		t.Errorf("first line = %q, want partition line (empty header must be omitted)", first)
	}
	if !strings.Contains(out, "#SBATCH --job-name=sami") {
		t.Errorf("rendered output missing job-name line, got: %q", out)
	}
}

func TestRenderHeaders_UnknownPartition(t *testing.T) {
	_, err := renderForTest(t, "nope")
	if err == nil {
		t.Fatal("expected error for unknown partition, got nil")
	}
	if !strings.Contains(err.Error(), "valid partitions") {
		t.Errorf("error should mention valid partitions, got: %v", err)
	}
}

func TestRenderScript_ComposesHeadersThenBuildCmd(t *testing.T) {
	f, err := config.LoadSbatchConfig(fixturePath)
	if err != nil {
		t.Fatalf("LoadSbatchConfig(fixture): %v", err)
	}

	var hdrBuf bytes.Buffer
	if err := RenderHeaders(&f.Sbatch, "zen4_gpu", testLogPaths, &hdrBuf); err != nil {
		t.Fatalf("render headers: %v", err)
	}

	buildCmdPath := "/some/logdir/build_cmd.sh"
	var out bytes.Buffer
	err = RenderScript(ScriptData{
		Headers:      hdrBuf.String(),
		BuildCmdPath: buildCmdPath,
	}, &out)
	if err != nil {
		t.Fatalf("RenderScript: %v", err)
	}

	s := out.String()
	lines := strings.SplitN(s, "\n", 2)
	if lines[0] != "#!/usr/bin/env bash" {
		t.Errorf("first line = %q, want shebang", lines[0])
	}
	hi := strings.Index(s, "#SBATCH -p zen4_gpu")
	tail := "samctr exec \\\n    -- /bin/sh <" + buildCmdPath
	ti := strings.Index(s, tail)
	if hi < 0 {
		t.Error("headers missing from composed script")
	}
	if ti < 0 {
		t.Errorf("samctr exec tail missing, want %q\n got: %q", tail, s)
	}
	if hi >= 0 && ti >= 0 && hi > ti {
		t.Errorf("headers must precede the samctr exec tail\n got: %q", s)
	}
}
