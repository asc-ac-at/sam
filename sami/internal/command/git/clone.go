package git

import "fmt"

func mkGitCloneCmd(repo, dir string) *GitCommandBuilder {
	cmd := fmt.Sprintf("git clone -- %s %s", repo, dir)
	res := NewGitCmd(cmd)
	return res
}
