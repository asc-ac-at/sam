package git

import (
	"strings"
	"testing"
)

func TestNewGitCmd(t *testing.T) {
	gcb := NewGitCmd("clone")
	if gcb == nil {
		t.Fatal("NewGitCmd returned nil")
	}
	if len(gcb.args) != 1 || gcb.args[0] != "clone" {
		t.Errorf("unexpected initial args: %v", gcb.args)
	}
}

func TestGitCommandBuilder_Arg(t *testing.T) {
	gcb := NewGitCmd("clone").Arg("origin", "dir")
	want := []string{"clone", "origin", "dir"}
	if len(gcb.args) != len(want) {
		t.Fatalf("expected %d args, got %d", len(want), len(gcb.args))
	}
	for i, w := range want {
		if gcb.args[i] != w {
			t.Errorf("args[%d] = %q, want %q", i, gcb.args[i], w)
		}
	}
}

func TestGitCommandBuilder_ToArgv(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		addl []string
		want []string
	}{
		{
			name: "simple",
			cmd:  "clone",
			want: []string{"git", "clone"},
		},
		{
			name: "with args",
			cmd:  "clone",
			addl: []string{"origin", "dir"},
			want: []string{"git", "clone", "origin", "dir"},
		},
		{
			name: "complex command",
			cmd:  "show-ref",
			addl: []string{"--heads", "refs/heads/main"},
			want: []string{"git", "show-ref", "--heads", "refs/heads/main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gcb := NewGitCmd(tt.cmd)
			if len(tt.addl) > 0 {
				gcb.Arg(tt.addl...)
			}
			got := gcb.ToArgv()
			if len(got) != len(tt.want) {
				t.Fatalf("ToArgv() = %v, want %v", got, tt.want)
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("ToArgv()[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestGitCommandBuilder_ToString(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		addl []string
		want string
	}{
		{
			name: "simple",
			cmd:  "clone",
			want: "git clone",
		},
		{
			name: "with args",
			cmd:  "clone",
			addl: []string{"origin", "dir"},
			want: "git clone origin dir",
		},
		{
			name: "with flags",
			cmd:  "clone",
			addl: []string{"--bare", "--mirror"},
			want: "git clone --bare --mirror",
		},
		{
			name: "complex",
			cmd:  "rev-parse",
			addl: []string{"--show-toplevel", "--absolute-git-dir"},
			want: "git rev-parse --show-toplevel --absolute-git-dir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gcb := NewGitCmd(tt.cmd)
			if len(tt.addl) > 0 {
				gcb.Arg(tt.addl...)
			}
			got := gcb.ToString()
			if got != tt.want {
				t.Errorf("ToString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGitCommandBuilder_Dir(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		addl []string
		dir  string
		want []string
	}{
		{
			name: "simple with dir",
			cmd:  "clone",
			addl: []string{"origin", "dir"},
			dir:  "/path/to/work",
			want: []string{"git", "-C", "/path/to/work", "clone", "origin", "dir"},
		},
		{
			name: "no extra args with dir",
			cmd:  "status",
			dir:  "/repo",
			want: []string{"git", "-C", "/repo", "status"},
		},
		{
			name: "complex with dir",
			cmd:  "show-ref",
			addl: []string{"main"},
			dir:  "/some/absolute/path",
			want: []string{"git", "-C", "/some/absolute/path", "show-ref", "main"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gcb := NewGitCmd(tt.cmd)
			if len(tt.addl) > 0 {
				gcb.Arg(tt.addl...)
			}
			gcb.Dir(tt.dir)
			got := gcb.ToArgv()
			if len(got) != len(tt.want) {
				t.Fatalf("ToArgv() = %v, want %v", got, tt.want)
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("ToArgv()[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestGitCommandBuilder_DirPrecedesCommand(t *testing.T) {
	// Verify that -C path is inserted before the command, not appended
	gcb := NewGitCmd("push").Arg("origin", "main")
	gcb.Dir("/work/repo")
	argv := gcb.ToArgv()

	// argv should be: git -C /work/repo push origin main
	// The -C must come before "push"
	cIdx := -1
	pshIdx := -1
	for i, a := range argv {
		if a == "-C" {
			cIdx = i
		}
		if a == "push" {
			pshIdx = i
		}
	}
	if cIdx < 0 {
		t.Error("missing -C flag in argv")
	}
	if pshIdx < 0 {
		t.Error("missing 'push' command in argv")
	}
	if cIdx >= pshIdx {
		t.Errorf("-C flag (%d) should precede command 'push' (%d)", cIdx, pshIdx)
	}
}

func TestGitCommandBuilder_Chained(t *testing.T) {
	// Verify method chaining works correctly
	gcb := NewGitCmd("clone").
		Arg("origin", "local-dir").
		Dir("/work").
		Arg("--depth", "1")

	got := gcb.ToArgv()
	want := []string{"git", "-C", "/work", "clone", "origin", "local-dir", "--depth", "1"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestGitCommandBuilder_StringContainsGitPrefix(t *testing.T) {
	gcb := NewGitCmd("status")
	s := gcb.ToString()
	if !strings.HasPrefix(s, "git ") {
		t.Errorf("ToString() = %q, expected to start with 'git '", s)
	}
}
