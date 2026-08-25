package git

import (
	"fmt"
	"log/slog"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging/buildlog"
)

// SetupGit is a generic catch-all function that checks out a git repository at the commit specified
// by the user. The specific commit is handled using one of the available command line options.
func SetupGit(opts *shared.Options, blPath *buildlog.BuildLogPaths, logger *slog.Logger) (*RepoState, error) {

	if err := initializeRepo(opts, blPath, logger); err != nil {
		return nil, err
	}

	state := &RepoState{}
	repoPaths, err := GetRepoPathsForDir(blPath.GitRepoPath, logger)
	logger.Debug(fmt.Sprintf("SetupGit set repoPaths %s", repoPaths))
	if err != nil {
		return state, err
	}
	state.Paths = repoPaths

	state, err = getCommitSha(opts, state, logger)
	if err != nil {
		logger.Error("SetupGit failed to fetch commitSha")
		return state, err
	}
	if err := checkoutCommit(state, logger); err != nil {
		return state, err
	}
	state = setTargetFiles(opts, state)
	return state, nil
}

// initializeRepo will perform a git clone on the opts.GitRepo at the
// GitRepoPath in the build log directory tree.
// returns nil on success
func initializeRepo(opts *shared.Options, blPath *buildlog.BuildLogPaths, logger *slog.Logger) error {
	gitClone := NewGitCmd("clone").Arg(opts.GitRepo, blPath.GitRepoPath).ToArgv()
	gitRunner := command.NewRunner(command.WithTimeout(3 * time.Minute))
	if err := gitRunner.Run(gitClone...); err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("ran %s", gitClone))
	return nil
}

// getCommitSha determines the commit sha of the commit requested by the user
// the commit request is specified via options to the command line
// mutates state by updating the CommitSha field
func getCommitSha(opts *shared.Options, state *RepoState, logger *slog.Logger) (*RepoState, error) {
	if opts.GitCommit != "" {
		state.CommitSha = opts.GitCommit
	}
	var err error
	if opts.GitBranch != "" {
		state, err = getCommitShaFromBranchName(opts.GitBranch, state, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("getCommitSha failed to determine commit from branch %s", opts.GitBranch))
			return state, err
		}
		return state, nil
	}
	if opts.GitMergeReqId != 0 {
		state, err = getCommitShaFromMergeReqId(opts.GitMergeReqId, state, logger)
		if err != nil {
			return state, err
		}
		return state, nil
	}
	return state, fmt.Errorf("getCommitSha error unkown")
}

// checkoutCommit checks out the commit referred to by
// returns nil on success
func checkoutCommit(state *RepoState, logger *slog.Logger) error {
	gitCheckoutCmd := NewGitCmd("checkout").Arg(state.CommitSha)
	gitCheckoutCmd.Dir(state.Paths.RepoPath())
	gitCheckout := gitCheckoutCmd.ToArgv()
	runner := command.NewRunner(command.WithTimeout(3 * time.Second))
	if err := runner.Run(gitCheckout...); err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("ran %s", gitCheckout))
	return nil
}

// setTargetFiles will copy the array of user specified target files if any were supplied as
// options
func setTargetFiles(opts *shared.Options, state *RepoState) *RepoState {
	if len(opts.Files) == 0 {
		return state
	}
	for _, f := range opts.Files {
		state.TargetFiles = append(state.TargetFiles, f)
	}
	return state
}
