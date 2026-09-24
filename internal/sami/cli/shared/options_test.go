package shared

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewOptions_Defaults(t *testing.T) {
	opts := NewOptions()

	if opts.GitRepo != "git@gitlab.tuwien.ac.at:vsc/software-stacks/asc-software-layer" {
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
	if opts.Owner != "" {
		t.Errorf("expected empty Owner, got %q", opts.Owner)
	}
	if opts.Group != "" {
		t.Errorf("expected empty Group, got %q", opts.Group)
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
	opts.GitBranch = "develop"
	opts.GitCommit = "abc123"
	opts.GitRepo = "https://example.com/repo.git"
	opts.GitMergeReqId = 42
	opts.SWSVariant = "2026.01"
	opts.Name = "my-build"
	opts.BuildLogBasePath = "/tmp/logs"
	opts.Verbose = true
	opts.Owner = "90116"
	opts.Group = "200300"

	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{"GitBranch", opts.GitBranch, "develop"},
		{"GitCommit", opts.GitCommit, "abc123"},
		{"GitRepo", opts.GitRepo, "https://example.com/repo.git"},
		{"GitMergeReqId", opts.GitMergeReqId, 42},
		{"SWSVariant", opts.SWSVariant, "2026.01"},
		{"Name", opts.Name, "my-build"},
		{"BuildLogBasePath", opts.BuildLogBasePath, "/tmp/logs"},
		{"Verbose", opts.Verbose, true},
		{"Owner", opts.Owner, "90116"},
		{"Group", opts.Group, "200300"},
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

// parseWith runs the flag machinery of a throwaway command and returns the
// resulting options, so tests exercise the real cobra/pflag binding.
func parseWith(t *testing.T, args []string) *Options {
	t.Helper()
	opts := NewOptions()
	cmd := &cobra.Command{
		Use:  "test",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	RegisterFlags(cmd, opts)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("parsing %v: %v", args, err)
	}
	return opts
}

func TestRegisterFlags_SbatchFlagsRepeatable(t *testing.T) {
	opts := parseWith(t, []string{
		"--sbatch-flags=--dependency=afterok:123",
		"--sbatch-flags=--hold",
	})
	want := []string{"--dependency=afterok:123", "--hold"}
	if !reflect.DeepEqual(opts.SbatchFlags, want) {
		t.Errorf("SbatchFlags: got %v, want %v", opts.SbatchFlags, want)
	}
}

// pflag must take the next argument as the value even when it starts with
// dashes; that is how sbatch options are written on the sam command line.
func TestRegisterFlags_SbatchFlagsSpaceSeparatedValue(t *testing.T) {
	opts := parseWith(t, []string{"--sbatch-flags", "--dependency=afterok:9"})
	want := []string{"--dependency=afterok:9"}
	if !reflect.DeepEqual(opts.SbatchFlags, want) {
		t.Errorf("SbatchFlags: got %v, want %v", opts.SbatchFlags, want)
	}
}

// StringArray (unlike StringSlice) must not split on commas: values pass
// through verbatim. CSV splitting would break e.g. --dependency=afterok:1:2.
func TestRegisterFlags_SbatchFlagsNoCommaSplit(t *testing.T) {
	opts := parseWith(t, []string{"--sbatch-flags=--dependency=afterok:1:2,afterany:3"})
	want := []string{"--dependency=afterok:1:2,afterany:3"}
	if !reflect.DeepEqual(opts.SbatchFlags, want) {
		t.Errorf("SbatchFlags: got %v, want %v", opts.SbatchFlags, want)
	}
}

// Every `flag:"name"` struct tag must match the name RegisterFlags binds
// the field to. Fields without a tag (GitBranch, GitCommit) are skipped.
func TestRegisterFlags_TagsMatchFlagNames(t *testing.T) {
	opts := NewOptions()
	cmd := &cobra.Command{Use: "test"}
	RegisterFlags(cmd, opts)

	typ := reflect.TypeOf(*opts)
	for i := 0; i < typ.NumField(); i++ {
		tag, ok := typ.Field(i).Tag.Lookup("flag")
		if !ok {
			continue
		}
		// RegisterFlags binds on PersistentFlags; cmd.Flags() only sees them
		// after Execute merges the sets, so look them up at the source.
		if cmd.PersistentFlags().Lookup(tag) == nil {
			t.Errorf("field %s: tag %q matches no registered flag", typ.Field(i).Name, tag)
		}
	}
}

func TestRegisterFlags_SbatchFlagsDefaultEmpty(t *testing.T) {
	opts := parseWith(t, nil)
	if len(opts.SbatchFlags) != 0 {
		t.Errorf("expected no SbatchFlags by default, got %v", opts.SbatchFlags)
	}
}

func TestRegisterFlags_EbFlagsRepeatable(t *testing.T) {
	opts := parseWith(t, []string{
		"--eb-flags=--rebuild",
		"--eb-flags=--fetch-timeout=60",
	})
	want := []string{"--rebuild", "--fetch-timeout=60"}
	if !reflect.DeepEqual(opts.EasyBuildFlags, want) {
		t.Errorf("EasyBuildFlags: got %v, want %v", opts.EasyBuildFlags, want)
	}
}

// pflag must take the next argument as the value even when it starts with
// dashes; that is how easybuild options are written on the sam command line.
func TestRegisterFlags_EbFlagsSpaceSeparatedValue(t *testing.T) {
	opts := parseWith(t, []string{"--eb-flags", "--from-pr"})
	want := []string{"--from-pr"}
	if !reflect.DeepEqual(opts.EasyBuildFlags, want) {
		t.Errorf("EasyBuildFlags: got %v, want %v", opts.EasyBuildFlags, want)
	}
}

// StringArray (unlike StringSlice) must not split on commas: values pass
// through verbatim. CSV splitting would break e.g. --try-amend with a list.
func TestRegisterFlags_EbFlagsNoCommaSplit(t *testing.T) {
	opts := parseWith(t, []string{"--eb-flags=--try-amend=a=1,2"})
	want := []string{"--try-amend=a=1,2"}
	if !reflect.DeepEqual(opts.EasyBuildFlags, want) {
		t.Errorf("EasyBuildFlags: got %v, want %v", opts.EasyBuildFlags, want)
	}
}

func TestRegisterFlags_EbFlagsDefaultEmpty(t *testing.T) {
	opts := parseWith(t, nil)
	if len(opts.EasyBuildFlags) != 0 {
		t.Errorf("expected no EasyBuildFlags by default, got %v", opts.EasyBuildFlags)
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
