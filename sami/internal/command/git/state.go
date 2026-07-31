package git

import (
	"fmt"
	"path/filepath"
)

type RepoState struct {
	Paths        *RepoPaths
	CommitSha    string
	ChangedFiles []string
}

// AllChangedFilePaths returns full filesystem paths by joining state.Paths.RepoPath()
// with each entry in state.ChangedFiles. Only .yaml files are included (filtered in GetChangedFiles).
func AllChangedFilePaths(state *RepoState) []string {
	if len(state.ChangedFiles) == 0 {
		return []string{}
	}
	paths := make([]string, 0, len(state.ChangedFiles))
	for _, f := range state.ChangedFiles {
		paths = append(paths, filepath.Join(state.Paths.RepoPath(), f))
	}
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
