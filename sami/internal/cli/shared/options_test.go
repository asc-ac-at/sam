package shared

import (
	"strings"
	"testing"
)

func TestNewOptions_Defaults(t *testing.T) {
	opts := NewOptions()

	if opts.CPU != "zen4" {
		t.Errorf("expected CPU default 'zen4', got %q", opts.CPU)
	}
	if opts.GPU != "h100" {
		t.Errorf("expected GPU default 'h100', got %q", opts.GPU)
	}
	if opts.GitRepo != "https://gitlab.tuwien.ac.at/vsc/software-stacks/asc-software-layer" {
		t.Errorf("unexpected GitRepo: %q", opts.GitRepo)
	}
	if opts.SWSVariant != "2025.06" {
		t.Errorf("expected SWSVariant default '2025.06', got %q", opts.SWSVariant)
	}
	if opts.BuildLogBasePath != "/opt/adm/asc-software-stack" {
		t.Errorf("unexpected BuildLogBasePath: %q", opts.BuildLogBasePath)
	}
	if opts.GitBranch != "" {
		t.Errorf("expected empty GitBranch, got %q", opts.GitBranch)
	}
	if opts.GitCommit != "" {
		t.Errorf("expected empty GitCommit, got %q", opts.GitCommit)
	}
	if opts.GitMergeReqId != 0 {
		t.Errorf("expected GitMergeReqId 0, got %d", opts.GitMergeReqId)
	}
	if opts.Name != "" {
		t.Errorf("expected empty Name, got %q", opts.Name)
	}
	if opts.Verbose {
		t.Error("expected Verbose false")
	}
}

func TestNewOptions_ReturnsNonNil(t *testing.T) {
	opts := NewOptions()
	if opts == nil {
		t.Fatal("NewOptions returned nil")
	}
}

func TestNewOptions_PointerReturned(t *testing.T) {
	opts1 := NewOptions()
	opts2 := NewOptions()
	// Each call should return a distinct pointer
	if opts1 == opts2 {
		t.Error("NewOptions should return distinct pointers on each call")
	}
}

func TestOptions_AllFields(t *testing.T) {
	opts := NewOptions()

	// Modify all fields
	opts.CPU = "amd"
	opts.GPU = "a100"
	opts.GitBranch = "develop"
	opts.GitCommit = "abc123"
	opts.GitRepo = "https://example.com/repo.git"
	opts.GitMergeReqId = 42
	opts.SWSVariant = "2026.01"
	opts.Name = "my-build"
	opts.BuildLogBasePath = "/tmp/logs"
	opts.Verbose = true

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"CPU", opts.CPU, "amd"},
		{"GPU", opts.GPU, "a100"},
		{"GitBranch", opts.GitBranch, "develop"},
		{"GitCommit", opts.GitCommit, "abc123"},
		{"GitRepo", opts.GitRepo, "https://example.com/repo.git"},
		{"GitMergeReqId", opts.GitMergeReqId, 42},
		{"SWSVariant", opts.SWSVariant, "2026.01"},
		{"Name", opts.Name, "my-build"},
		{"BuildLogBasePath", opts.BuildLogBasePath, "/tmp/logs"},
		{"Verbose", opts.Verbose, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s: got %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestNewOptions_BuildBackendDefault(t *testing.T) {
	opts := NewOptions()
	if opts.BuildBackend != string(BackendLocal) {
		t.Errorf("expected BuildBackend default 'local', got %q", opts.BuildBackend)
	}
}

func TestParseBackend(t *testing.T) {
	cases := []struct {
		in      string
		want    Backend
		wantErr bool
	}{
		{"slurm", BackendSlurm, false},
		{"local", BackendLocal, false},
		{"bogus", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := ParseBackend(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseBackend(%q): expected error, got nil", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseBackend(%q): unexpected error %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseBackend(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if err := func() error { _, err := ParseBackend("bogus"); return err }(); err != nil && !strings.Contains(err.Error(), "slurm") {
		t.Errorf("unknown backend error should list valid values, got: %v", err)
	}
}
