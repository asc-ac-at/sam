// ideas from this package come from the fantastic lazygit
// "github.com/jesseduffield/lazygit"
package git

import "strings"

// convenience struct for building git commands. Especially useful when
// including conditional args
type GitCommandBuilder struct {
	// command string
	Args []string
}

func NewGitCmd(command string) *GitCommandBuilder {
	return &GitCommandBuilder{Args: []string{command}}
}

func (gcb *GitCommandBuilder) Arg(args ...string) *GitCommandBuilder {
	gcb.Args = append(gcb.Args, args...)

	return gcb
}

func (gcb *GitCommandBuilder) ToArgv() []string {
	return append([]string{"git"}, gcb.Args...)
}

func (gcb *GitCommandBuilder) ToString() string {
	return strings.Join(gcb.ToArgv(), " ")
}

// the -C arg will make git do a `cd` to the directory before doing anything else
func (gcb *GitCommandBuilder) Dir(path string) *GitCommandBuilder {
	// repo path comes before the command
	gcb.Args = append([]string{"-C", path}, gcb.Args...)

	return gcb
}
