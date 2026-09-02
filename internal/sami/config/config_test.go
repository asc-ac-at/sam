package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/asc-ac-at/sam/internal/sami/logging/buildlog"
)

// fixedLogPaths returns a stable BuildLogPaths fixture: SbatchHeaders uses
// only BuildLog, to derive the slurm --output path, so a fixed literal keeps
// assertions deterministic.
func fixedLogPaths() *buildlog.BuildLogPaths {
	return &buildlog.BuildLogPaths{BuildLog: "/logdir"}
}

const canonicalYAML = `
sbatch-config:
  shared:
    header: "#sami --vanilla"
  partitions:
    zen4_gpu:
      qos: standard
      gres: "gpu:1"
      ntasks: 32
      mem: 256G
      cpus-per-task: 1
      threads-per-core: 1
    zen4_cpu:
      qos: standard
      ntasks: 32
      mem: 256G
      cpus-per-task: 1
      threads-per-core: 2
    zen5_gpu:
      qos: standard
      gres: "gpu:1"
      ntasks: 32
      mem: 256G
      cpus-per-task: 1
      threads-per-core: 1

arch-mapping:
  zen4: x86_64/amd/zen4

rgw:
  bucket: sam-archives
  endpoint: https://rgw.example.org
`

// loadFromYAML writes yaml to a temp file and loads it through LoadSbatchConfig,
// so tests exercise the real loader (path reading + strict decoding).
func loadFromYAML(t *testing.T, yaml string) *File {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	f, err := LoadSbatchConfig(path)
	if err != nil {
		t.Fatalf("LoadSbatchConfig: %v", err)
	}
	return f
}

func TestLoadSbatchConfig_Canonical(t *testing.T) {
	f := loadFromYAML(t, canonicalYAML)

	if len(f.Sbatch.Partitions) != 3 {
		t.Fatalf("got %d partitions, want 3", len(f.Sbatch.Partitions))
	}
	if f.Sbatch.SharedConfig.Header != "#sami --vanilla" {
		t.Errorf("Header = %q", f.Sbatch.SharedConfig.Header)
	}
	p := f.Sbatch.Partitions["zen4_gpu"]
	if p.Qos != "standard" {
		t.Errorf("Qos = %q", p.Qos)
	}
	if p.Ntasks != "32" {
		t.Errorf("Ntasks = %q, want 32", p.Ntasks)
	}
	if p.Gres != "gpu:1" {
		t.Errorf("Gres = %q, want gpu:1", p.Gres)
	}
	if p.Mem != "256G" {
		t.Errorf("Mem = %q", p.Mem)
	}
	if p.CpusPerTask != 1 {
		t.Errorf("CpusPerTask = %d, want 1", p.CpusPerTask)
	}
	if p.ThreadsPerCore != 1 {
		t.Errorf("ThreadsPerCore = %d, want 1", p.ThreadsPerCore)
	}
	if f.RGW.Bucket != "sam-archives" {
		t.Errorf("RGW.Bucket = %q, want sam-archives", f.RGW.Bucket)
	}
	if f.RGW.Endpoint != "https://rgw.example.org" {
		t.Errorf("RGW.Endpoint = %q, want https://rgw.example.org", f.RGW.Endpoint)
	}
}

func TestLoadSbatchConfig_CanonicalFile(t *testing.T) {
	// the committed canonical fixture
	f, err := LoadSbatchConfig(filepath.Join("..", "test", "config.yaml"))
	if err != nil {
		t.Fatalf("LoadSbatchConfig(fixture): %v", err)
	}
	if len(f.Sbatch.Partitions) != 3 {
		t.Fatalf("got %d partitions, want 3", len(f.Sbatch.Partitions))
	}
	if f.RGW.Bucket != "sam-archives" {
		t.Errorf("fixture RGW.Bucket = %q, want sam-archives", f.RGW.Bucket)
	}
	if f.RGW.Endpoint != "" {
		t.Errorf("fixture RGW.Endpoint = %q, want empty (comment only)", f.RGW.Endpoint)
	}
}

func TestLoadSbatchConfig_PartitionNamePopulatedFromKey(t *testing.T) {
	f := loadFromYAML(t, canonicalYAML)

	for name, p := range f.Sbatch.Partitions {
		if p.Partition != name {
			t.Errorf("Partition = %q, want map key %q", p.Partition, name)
		}
	}
}

func TestLoadSbatchConfig_UnknownKeyRejected(t *testing.T) {
	bad := canonicalYAML + "  unknown_key:\n    anything: true\n"
	f, err := LoadSbatchConfig(writeTempConfig(t, bad))
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !strings.Contains(err.Error(), "unknown_key") {
		t.Errorf("error should mention the unknown key, got: %v", err)
	}
	if f != nil {
		t.Error("expected nil *File on error")
	}
}

func TestLoadSbatchConfig_StrayPartitionKeyRejected(t *testing.T) {
	bad := strings.Replace(canonicalYAML,
		"zen4_cpu:\n      qos: standard",
		"zen4_cpu:\n      partition: zen4_cpu\n      qos: standard",
		1)
	if _, err := LoadSbatchConfig(writeTempConfig(t, bad)); err == nil {
		t.Fatal("expected error for stray partition: key, got nil")
	}
}

func TestLoadSbatchConfig_DuplicateKeyRejected(t *testing.T) {
	bad := strings.Replace(canonicalYAML,
		"    zen4_cpu:",
		"    zen4_cpu:\n      qos: doubled\n      ntasks: 1\n      mem: 64G\n      cpus-per-task: 1\n      threads-per-core: 1\n    zen4_cpu:",
		1)
	if _, err := LoadSbatchConfig(writeTempConfig(t, bad)); err == nil {
		t.Fatal("expected error for duplicate partition key, got nil")
	}
}

func TestSbatchHeaders_Hit(t *testing.T) {
	f := loadFromYAML(t, canonicalYAML)

	h, err := f.Sbatch.SbatchHeaders("zen4_gpu", fixedLogPaths())
	if err != nil {
		t.Fatalf("SbatchHeaders: %v", err)
	}
	if h.Header != "#sami --vanilla" {
		t.Errorf("Header = %q", h.Header)
	}
	if h.Partition.Partition != "zen4_gpu" {
		t.Errorf("Partition = %q", h.Partition.Partition)
	}
	if h.Partition.Gres != "gpu:1" {
		t.Errorf("Gres = %q", h.Partition.Gres)
	}
	// job-name and output are rendered from code, not config: assert the
	// exact values the template will emit.
	if h.JobName != "sami" {
		t.Errorf("JobName = %q, want %q", h.JobName, "sami")
	}
	if want := filepath.Join("/logdir", "slurm-%j.out"); h.Output != want {
		t.Errorf("Output = %q, want %q", h.Output, want)
	}
}

func TestSbatchHeaders_CPUPartitionHasNoGres(t *testing.T) {
	f := loadFromYAML(t, canonicalYAML)

	h, err := f.Sbatch.SbatchHeaders("zen4_cpu", fixedLogPaths())
	if err != nil {
		t.Fatalf("SbatchHeaders: %v", err)
	}
	if h.Partition.Gres != "" {
		t.Errorf("CPU partition should have no gres, got %q", h.Partition.Gres)
	}
	if h.Partition.ThreadsPerCore != 2 {
		t.Errorf("ThreadsPerCore = %d, want 2", h.Partition.ThreadsPerCore)
	}
}

func TestSbatchHeaders_MissListsValidPartitions(t *testing.T) {
	f := loadFromYAML(t, canonicalYAML)

	_, err := f.Sbatch.SbatchHeaders("nope", fixedLogPaths())
	if err == nil {
		t.Fatal("expected error for unknown partition, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, `"nope"`) {
		t.Errorf("error should name the requested partition, got: %v", err)
	}
	for _, name := range []string{"zen4_cpu", "zen4_gpu", "zen5_gpu"} {
		if !strings.Contains(msg, name) {
			t.Errorf("error should list valid partition %q, got: %v", name, err)
		}
	}
}

func writeTempConfig(t *testing.T, yaml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestArchSubdir_Mapping(t *testing.T) {
	f, err := LoadSbatchConfig(filepath.Join("..", "test", "config.yaml"))
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	t.Run("hit", func(t *testing.T) {
		sub, err := f.ArchSubdir("zen4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub != "x86_64/amd/zen4" {
			t.Errorf("got %q, want x86_64/amd/zen4", sub)
		}
	})

	t.Run("miss lists valid archs", func(t *testing.T) {
		_, err := f.ArchSubdir("skylake")
		if err == nil {
			t.Fatal("expected error for unknown arch")
		}
		for _, want := range []string{"zen4", "zen5"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error should list valid arch %q, got: %v", want, err)
			}
		}
	})
}

func TestAccelSubdir_Mapping(t *testing.T) {
	f, err := LoadSbatchConfig(filepath.Join("..", "test", "config.yaml"))
	if err != nil {
		t.Fatalf("failed to load fixture: %v", err)
	}

	t.Run("hit", func(t *testing.T) {
		sub, err := f.AccelSubdir("cc90")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub != "accel/nvidia/cc90" {
			t.Errorf("got %q, want accel/nvidia/cc90", sub)
		}
	})

	t.Run("miss lists valid accels", func(t *testing.T) {
		_, err := f.AccelSubdir("gfx90a")
		if err == nil {
			t.Fatal("expected error for unknown accel")
		}
		for _, want := range []string{"cc80", "cc90"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error should list valid accel %q, got: %v", want, err)
			}
		}
	})
}
