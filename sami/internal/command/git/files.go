package git

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
)

// getChangedFiles runs a git command that determines what files changed at a particular commit
//
//	`git diff-tree --name-only --no-commit-id <commit sha1> -r`
//
// returns a list of the paths to the changed files within the git worktree
func GetChangedFiles(state *RepoState, logger *slog.Logger) (*RepoState, error) {
	gitDiffTreeCmd := NewGitCmd("diff-tree").Arg("--name-only", "--no-commit-id", state.CommitSha, "-r")
	gitDiffTreeCmd.Dir(state.Paths.RepoPath())
	runner := command.NewRunner(command.WithTimeout(3 * time.Second))
	cmdCfg := runner.New(gitDiffTreeCmd.ToArgv()...)
	buf := bytes.Buffer{}
	cmdCfg.WithStdout(&buf).WithStderr(os.Stderr)
	out, err := cmdCfg.Output()
	if err != nil {
		return state, err
	}
	logger.Info("GetChangedFiles returns ", "out", out)
	lines := strings.Split(string(out), "\n")
	for i := range lines {
		fmt.Println(lines[i])
	}
	lines = slices.DeleteFunc(lines, func(s string) bool {
		return s == "" // delete the empty strings
	})
	lines = slices.DeleteFunc(lines, func(s string) bool {
		return !strings.HasSuffix(s, ".yaml") // filter for .yaml easystack files
	})
	state.ChangedFiles = lines
	return state, nil
}
