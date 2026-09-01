package git

import (
	"bytes"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/asc-ac-at/sam/internal/sami/command"
)

// getChangedFiles runs a git command that determines what files changed at a particular commit
//
//	`git diff-tree --name-only --no-commit-id <commit sha1> -r`
//
// returns a list of the paths to the changed files within the git worktree
func GetChangedFiles(state *RepoState, logger *slog.Logger) (*RepoState, error) {
	gitDiffTreeCmd := NewGitCmd("diff-tree").Arg("--name-only", "--no-commit-id", state.CommitSha, "-r")
	gitDiffTreeCmd.Dir(state.Paths.RepoPath())

	cfg := command.NewCmdConfig(gitDiffTreeCmd.ToArgv())

	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr
	cfg.Timeout = 3 * time.Second

	if err := cfg.Run(); err != nil {
		return state, fmt.Errorf("git diff-tree (%s): %w: %s", state.CommitSha, err, strings.TrimSpace(stderr.String()))
	}

	out := stdout.String()
	logger.Debug("GetChangedFiles returns ", "out", out)
	lines := strings.Split(out, "\n")
	lines = slices.DeleteFunc(lines, func(s string) bool {
		return s == "" // delete the empty strings
	})
	lines = slices.DeleteFunc(lines, func(s string) bool {
		return !strings.HasSuffix(s, ".yaml") // filter for .yaml easystack files
	})
	state.ChangedFiles = lines
	return state, nil
}
