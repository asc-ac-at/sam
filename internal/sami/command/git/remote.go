package git

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/asc-ac-at/sam/internal/sami/command"
)

// fetchHead fetches gitRef from origin into the local clone and returns the
// sha FETCH_HEAD points at. Working through refs (branch heads as well as
// GitLab's refs/merge-requests/*/head) sidesteps uploadpack policy for
// fetching arbitrary shas, and ensures the object is present in what may be
// a --depth=1, default-branch-only clone.
func fetchHead(ref string, state *RepoState, logger *slog.Logger) (string, error) {
	// depth must cover the commit AND its parent: downstream uses
	// 'git diff-tree <sha>' to compute changed files, and a depth-1 fetch
	// leaves the parent grafted away, which makes diff-tree see a root
	// commit and return an empty file list
	fetch := NewGitCmd("fetch").Arg("--depth=2", "origin", ref)
	fetch.Dir(state.Paths.RepoPath())

	cfg := command.NewCmdConfig(fetch.ToArgv())

	var stderr bytes.Buffer
	cfg.Stdout = io.Discard
	cfg.Stderr = &stderr
	cfg.Timeout = 3 * time.Minute

	logger.Debug("fetchHead running", "ref", ref)
	if err := cfg.Run(); err != nil {
		return "", fmt.Errorf("git fetch (%s): %w: %s", ref, err, strings.TrimSpace(stderr.String()))
	}

	revParse := NewGitCmd("rev-parse").Arg("FETCH_HEAD")
	revParse.Dir(state.Paths.RepoPath())

	revCfg := command.NewCmdConfig(revParse.ToArgv())

	var stdout bytes.Buffer
	stderr.Reset()
	revCfg.Stdout = &stdout
	revCfg.Stderr = &stderr
	revCfg.Timeout = 10 * time.Second

	if err := revCfg.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse (FETCH_HEAD): %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	sha := strings.TrimSpace(stdout.String())
	if sha == "" {
		return "", fmt.Errorf("git rev-parse (FETCH_HEAD) returned empty sha for ref %s", ref)
	}
	return sha, nil
}

// getCommitShaFromMergeReqId gets the commit sha from a merge id supplied by the caller
// get <full length SHA-1> corresponding to MergeRequest number (gitlab only)
// internal logic is based on the shell pipeline:
//
//	git ls-remote origin "refs/merge-requests/${MR_NUM}/head" | awk '{print $1}'
//
// returns the full length SHA-1 if the merge request was found, an empty string if the MR does not
// exist
func getCommitShaFromMergeReqId(mrid int, state *RepoState, logger *slog.Logger) (*RepoState, error) {
	ref := fmt.Sprintf("refs/merge-requests/%d/head", mrid)
	sha, err := fetchHead(ref, state, logger)
	if err != nil {
		return state, err
	}
	state.CommitSha = sha
	return state, nil
}

// getCommitShaFromBranchName
func getCommitShaFromBranchName(name string, state *RepoState, logger *slog.Logger) (*RepoState, error) {
	sha, err := fetchHead(name, state, logger)
	if err != nil {
		return state, err
	}
	state.CommitSha = sha
	return state, nil
}

var fullShaPattern = regexp.MustCompile("^[0-9a-fA-F]{40}$")

// getCommitShaFromSha resolves a user-supplied commit sha by fetching it
// directly. The ASC GitLab's uploadpack policy permits sha fetches (verified
// live 2026-09-16). Only full 40-character shas are accepted: shorthand
// expansion would require additional resolver fetches, so the caller gets an
// actionable error instead.
func getCommitShaFromSha(sha string, state *RepoState, logger *slog.Logger) (*RepoState, error) {
	sha = strings.TrimSpace(sha)
	if !fullShaPattern.MatchString(sha) {
		return state, fmt.Errorf("--git-commit requires the full 40-character SHA (got %q); expand shorthand with `git rev-parse <sha>`", sha)
	}
	got, err := fetchHead(sha, state, logger)
	if err != nil {
		return state, err
	}
	if !strings.EqualFold(got, sha) {
		return state, fmt.Errorf("fetch of %q resolved to unexpected sha %q", sha, got)
	}
	state.CommitSha = got
	return state, nil
}

// get files changed in an MR
// git diff-tree --name-only --no-commit-id ${mr_head} -r

// fetch a specific commit
// git fetch PATH-TO-REPO-GIT-DIR <full length SHA-1>

// get a
