package git

import (
	"fmt"
	"path/filepath"
)

// RepoState represents the state of a checked out git repository
//
//	Paths are the paths to relevant directories on the filesystem
//	CommitSha is the currently checked out revision
//	ChangedFiles is a list containing the names of files that changed in the currently checked out
//	commit
//	TargetFiles can be used to store user specified files to target within the repository
type RepoState struct {
	Paths        *RepoPaths
	CommitSha    string
	ChangedFiles []string
	TargetFiles  []string
}

// makePaths is a convenience function for constructing valid filepath names within a git repo
func makePaths(files []string, repoPath string) []string {
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, filepath.Join(repoPath, f))
	}
	return paths
}

// AllChangedFilePaths returns full filesystem paths by joining state.Paths.RepoPath()
// with each entry in state.ChangedFiles. Only .yaml files are included (filtered in GetChangedFiles).
func AllChangedFilePaths(state *RepoState) []string {
	if len(state.ChangedFiles) == 0 {
		return []string{}
	}
	paths := makePaths(state.ChangedFiles, state.Paths.RepoPath())
	return paths
}

// ALlTargetFilePaths returns full filesystem paths by joining state.Paths.RepoPath()
// with each entry in state.TargetFiles. Only .yaml files are included.
func AllTargetFilePaths(state *RepoState) []string {
	if len(state.TargetFiles) == 0 {
		return []string{}
	}
	paths := makePaths(state.TargetFiles, state.Paths.RepoPath())
	return paths
}

// HasChangedFiles returns true if state.ChangedFiles is non-empty.
func HasChangedFiles(state *RepoState) bool {
	return len(state.ChangedFiles) > 0
}

// ChangedFilesCount returns len(state.ChangedFiles).
func ChangedFilesCount(state *RepoState) int {
	return len(state.ChangedFiles)
}

// FirstChangedFile returns the first full path. Error: "git shows no changed files" if slice is empty.
func FirstChangedFile(state *RepoState) (string, error) {
	if !HasChangedFiles(state) {
		return "", fmt.Errorf("git shows no changed files")
	}
	return filepath.Join(state.Paths.RepoPath(), state.ChangedFiles[0]), nil
}
