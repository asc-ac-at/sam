package config

import (
	"bytes"
	"strings"
	"testing"
)

func renderForTest(t *testing.T, partition string) (string, error) {
	t.Helper()
	f := loadFromYAML(t, canonicalYAML)
	var buf bytes.Buffer
	err := RenderSbatchHeaders(&f.Sbatch, partition, &buf)
	return buf.String(), err
}

func TestRenderSbatchHeaders_GPUPartition(t *testing.T) {
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
		"#SBATCH --job-name=\"${SW_NAME}-samctr\"",
		"#SBATCH --output=\"$LOGDIR\"/slurm-%j.out",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing %q\n got: %q", want, out)
		}
	}
}

func TestRenderSbatchHeaders_CPUPartitionOmitsGres(t *testing.T) {
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

func TestRenderSbatchHeaders_UnknownPartition(t *testing.T) {
	_, err := renderForTest(t, "nope")
	if err == nil {
		t.Fatal("expected error for unknown partition, got nil")
	}
	if !strings.Contains(err.Error(), "valid partitions") {
		t.Errorf("error should mention valid partitions, got: %v", err)
	}
}
