package git

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"gitlab.tuwien.ac.at/vsc/software-stacks/sami.git/internal/command"
)

//func GetCommitFromMergeRequestNum(num int, repo GitRepo) (string, error) {
//
//}

// getCommitShaFromMergeReqId gets the commit sha from a merge id supplied by the caller
func getCommitShaFromMergeReqId(mrid int, rp *RepoPaths, logger *slog.Logger) (string, error) {
	gitCmd := NewGitCmd("ls-remote").Arg("origin", fmt.Sprintf("refs/merge-requests/%d/head", mrid))
	gitCmd.Dir(rp.RepoPath())
	runner := command.NewRunner(command.WithTimeout(10 * time.Second))
	cmdCfg := runner.New(gitCmd.ToArgv()...)
	buf := bytes.Buffer{}
	cmdCfg.WithStdout(&buf).WithStderr(os.Stderr)
	logger.Info("getCommitShaFromMergeReqId returned ", "merge-req-id", mrid)
	return "", nil
}

func getCommitShaFromBranchName(name string, rp *RepoPaths, logger *slog.Logger) (string, error) {
	gitCmd := NewGitCmd("show-ref").Arg(name)
	gitCmd.Dir(rp.RepoPath())
	logger.Debug("getCommitShaFromBranchName preparing to run ", "cmd", gitCmd.ToString())
	runner := command.NewRunner(command.WithTimeout(10 * time.Second))
	cmdCfg := runner.New(gitCmd.ToArgv()...)
	buf := bytes.Buffer{}
	cmdCfg.WithStdout(&buf).WithStderr(os.Stderr)
	out, err := cmdCfg.Output()
	logger.Info("return", "out", out)
	if err != nil {
		return "", err
	}
	commitId := strings.Split(string(out), " ")
	return commitId[0], nil
}

// get <full length SHA-1> corresponding to MergeRequest number (gitlab only)
// mr_head=$(git ls-remote origin "refs/merge-requests/${MR_NUM}/head" | awk '{print $1}')

// get files changed in an MR
// git diff-tree --name-only --no-commit-id ${mr_head} -r

// fetch a specific commit
// git fetch PATH-TO-REPO-GIT-DIR <full length SHA-1>

// get a
