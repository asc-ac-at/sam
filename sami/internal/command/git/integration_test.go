package git

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
)

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// runGit runs git in dir using the verbose CmdConfig canon, failing the test
// on any error. Returns trimmed stdout.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	argv := append([]string{"git", "-C", dir}, args...)

	cfg := command.NewCmdConfig(argv)

	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr
	cfg.Timeout = 30 * time.Second

	if err := cfg.Run(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String())
}

// newTestRepo builds a two-commit local repo: the first commit adds a
// placeholder file, the second adds an easystack yaml — so diff-tree of HEAD
// reports exactly one .yaml file.
func newTestRepo(t *testing.T) (dir string, headSha string, branch string) {
	t.Helper()
	dir = t.TempDir()

	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "sami-test@example.com")
	runGit(t, dir, "config", "user.name", "Sami Test")

	if err := os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "seed")

	stackDir := filepath.Join(dir, "easystacks", "2025.06")
	if err := os.MkdirAll(stackDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stackFile := filepath.Join(stackDir, "asc_eb_5.3.0-test.yaml")
	if err := os.WriteFile(stackFile, []byte("easyconfigs: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add easystack")

	headSha = runGit(t, dir, "rev-parse", "HEAD")
	branch = runGit(t, dir, "branch", "--show-current")
	return dir, headSha, branch
}

func newTestState(dir, sha string) *RepoState {
	return &RepoState{
		Paths:     &RepoPaths{repoPath: dir, repoGitDirPath: filepath.Join(dir, ".git")},
		CommitSha: sha,
	}
}

func TestGetRepoPathsForDir_Integration(t *testing.T) {
	dir, _, _ := newTestRepo(t)

	rp, err := GetRepoPathsForDir(dir, newDiscardLogger())
	if err != nil {
		t.Fatalf("GetRepoPathsForDir: %v", err)
	}

	wantDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	gotRepo, err := filepath.EvalSymlinks(rp.RepoPath())
	if err != nil {
		t.Fatal(err)
	}
	if gotRepo != wantDir {
		t.Errorf("RepoPath = %q, want %q", gotRepo, wantDir)
	}
	if want := filepath.Join(wantDir, ".git"); rp.RepoGitDirPath() != want {
		t.Errorf("RepoGitDirPath = %q, want %q", rp.RepoGitDirPath(), want)
	}
}

func TestGetChangedFiles_Integration(t *testing.T) {
	dir, headSha, _ := newTestRepo(t)
	state := newTestState(dir, headSha)

	got, err := GetChangedFiles(state, newDiscardLogger())
	if err != nil {
		t.Fatalf("GetChangedFiles: %v", err)
	}
	want := "easystacks/2025.06/asc_eb_5.3.0-test.yaml"
	if len(got.ChangedFiles) != 1 || got.ChangedFiles[0] != want {
		t.Errorf("ChangedFiles = %v, want [%s]", got.ChangedFiles, want)
	}
}

func TestGetChangedFiles_BadShaReportsStderr(t *testing.T) {
	dir, _, _ := newTestRepo(t)
	state := newTestState(dir, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef")

	_, err := GetChangedFiles(state, newDiscardLogger())
	if err == nil {
		t.Fatal("expected error for unknown commit, got nil")
	}
	// per the canon: git's own diagnostics travel inside the error
	if !strings.Contains(err.Error(), "bad object") {
		t.Errorf("error should carry git's stderr, got: %v", err)
	}
}

func TestGetCommitShaFromBranchName_Integration(t *testing.T) {
	dir, headSha, branch := newTestRepo(t)
	state := newTestState(dir, "")

	got, err := getCommitShaFromBranchName(branch, state, newDiscardLogger())
	if err != nil {
		t.Fatalf("getCommitShaFromBranchName(%q): %v", branch, err)
	}
	if got.CommitSha != headSha {
		t.Errorf("CommitSha = %q, want %q", got.CommitSha, headSha)
	}
}
