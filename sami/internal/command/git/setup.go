package git

import (
	"fmt"
	"log/slog"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/cli/shared"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/logging/buildlog"
)

func SetupGit(opts *shared.Options, bldPath *buildlog.BuildLogPaths, logger *slog.Logger) error {

	if err := initializeRepo(opts, bldPath, logger); err != nil {
		return err
	}

	repoPaths, err := GetRepoPathsForDir(bldPath.GitRepoPath, logger)
	if err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("SetupGit set repoPaths %s", repoPaths))
	commitSha, err := getCommitSha(opts, repoPaths, logger)
	if err != nil {
		logger.Error("SetupGit failed to fetch commitSha")
		return err
	}
	logger.Info(fmt.Sprintf("SetupGit found user requested commit %s", commitSha))
	return nil
}

func initializeRepo(opts *shared.Options, bldPath *buildlog.BuildLogPaths, logger *slog.Logger) error {
	gitClone := NewGitCmd("clone").Arg(opts.GitRepo, bldPath.GitRepoPath).ToArgv()
	gitRunner := command.NewRunner(command.WithTimeout(3 * time.Minute))
	if err := gitRunner.Run(gitClone...); err != nil {
		return err
	}
	logger.Debug(fmt.Sprintf("ran %s", gitClone))
	logger.Info("will now check remote for changed files")
	return nil
}

// getCommitSha determines the commit sha of the commit requested by the user
// the commit request is specified via options to the command line
func getCommitSha(opts *shared.Options, repoPaths *RepoPaths, logger *slog.Logger) (string, error) {
	if opts.GitCommit != "" {
		return opts.GitCommit, nil
	}
	if opts.GitBranch != "" {
		res, err := getCommitShaFromBranchName(opts.GitBranch, repoPaths, logger)
		if err != nil {
			logger.Error(fmt.Sprintf("getCommitSha failed to determine commit from branch %s", opts.GitBranch))
			return "", err
		}
		return res, nil
	}
	//if opts.GitMergeReqId != 0 {
	//	res, err := getCommitShaFromMergeReqId(opts.GitMergeReqId, logger)
	//	if err != nil {
	//		return "", err
	//	}
	//	return res, nil
	//}
	return "", fmt.Errorf("getCommitSha error unkown")
}
