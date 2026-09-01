package git

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
)

type RepoPaths struct {
	repoPath       string
	repoGitDirPath string
}

// Path of the repo. If we're in a the main worktree, this will be the same as WorktreePath()
// If we're in a bare repo, it will be the parent folder of the bare repo
func (rp *RepoPaths) RepoPath() string {
	return rp.repoPath
}

// path of the git-dir for the repo.
// If this is a bare repo, it will be the location of the bare repo
// If this is a non-bare repo, it will be the location of the .git dir in
// the main worktree.
func (rp *RepoPaths) RepoGitDirPath() string {
	return rp.repoGitDirPath
}

// GetRepoPathsForDir goes to the dir and checks the values for the paths
// Returns a RepoPaths struct or error. If RepoPaths is returned, the repo
// exists and is checkout out on the default branch at the repoPath.
func GetRepoPathsForDir(dir string, logger *slog.Logger) (*RepoPaths, error) {
	//TODO: we need to call this _after_ git clone
	gitCmd := NewGitCmd("rev-parse").Arg("--show-toplevel", "--absolute-git-dir")
	gitCmd.Dir(dir)

	cfg := command.NewCmdConfig(gitCmd.ToArgv())

	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr
	cfg.Timeout = 1 * time.Minute

	if err := cfg.Run(); err != nil {
		return nil, fmt.Errorf("git rev-parse (%s): %w: %s", dir, err, strings.TrimSpace(stderr.String()))
	}

	lines := strings.Split(stdout.String(), "\n")
	if len(lines) != 3 {
		return nil, fmt.Errorf("GetRepoPathsForDir unexpected output with %d lines (expected 3)", len(lines))
	}
	rp := &RepoPaths{
		repoPath:       lines[0],
		repoGitDirPath: lines[1],
	}
	logger.Debug(fmt.Sprintf("GetRepoPathsForDir set - repoPath: %s, repoGitDirPath %s", rp.RepoPath(), rp.RepoGitDirPath()))
	return rp, nil
}
