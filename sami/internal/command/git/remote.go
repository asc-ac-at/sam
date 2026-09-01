package git

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
)

// getCommitShaFromMergeReqId gets the commit sha from a merge id supplied by the caller
// get <full length SHA-1> corresponding to MergeRequest number (gitlab only)
// internal logic is based on the shell pipeline:
//
//	git ls-remote origin "refs/merge-requests/${MR_NUM}/head" | awk '{print $1}'
//
// returns the full length SHA-1 if the merge request was found, an empty string if the MR does not
// exist
func getCommitShaFromMergeReqId(mrid int, state *RepoState, logger *slog.Logger) (*RepoState, error) {

	gitCmd := NewGitCmd("ls-remote").Arg("origin", fmt.Sprintf("refs/merge-requests/%d/head", mrid))
	gitCmd.Dir(state.Paths.RepoPath())

	cfg := command.NewCmdConfig(gitCmd.ToArgv())

	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr
	cfg.Timeout = 10 * time.Second

	if err := cfg.Run(); err != nil {
		return state, fmt.Errorf("git ls-remote (mr %d): %w: %s", mrid, err, strings.TrimSpace(stderr.String()))
	}

	out := stdout.String()
	logger.Debug("getCommitShaFromMergeReqId returned ", "out", out)
	parts := strings.Split(out, "\t")
	commitId := parts[0]
	if commitId == "" {
		return state, fmt.Errorf("getCommitShaFromMergeReqId found no sha for mr id %d", mrid)
	}
	state.CommitSha = commitId
	return state, nil
}

// getCommitShaFromBranchName
func getCommitShaFromBranchName(name string, state *RepoState, logger *slog.Logger) (*RepoState, error) {
	gitCmd := NewGitCmd("show-ref").Arg(name)
	gitCmd.Dir(state.Paths.RepoPath())
	logger.Debug("getCommitShaFromBranchName preparing to run ", "cmd", gitCmd.ToString())

	cfg := command.NewCmdConfig(gitCmd.ToArgv())

	var stdout, stderr bytes.Buffer
	cfg.Stdout = &stdout
	cfg.Stderr = &stderr
	cfg.Timeout = 10 * time.Second

	if err := cfg.Run(); err != nil {
		return state, fmt.Errorf("git show-ref (%s): %w: %s", name, err, strings.TrimSpace(stderr.String()))
	}

	out := stdout.String()
	logger.Debug("return", "out", out)
	commitId := strings.Split(out, " ")
	state.CommitSha = commitId[0]
	return state, nil
}

// get files changed in an MR
// git diff-tree --name-only --no-commit-id ${mr_head} -r

// fetch a specific commit
// git fetch PATH-TO-REPO-GIT-DIR <full length SHA-1>

// get a
