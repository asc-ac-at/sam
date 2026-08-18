package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// redirectSearchPaths points the config search at isolated tempdirs and
// returns the user and system candidate paths. The system path is a package
// var so tests can redirect it.
func redirectSearchPaths(t *testing.T) (userPath, systemPath string) {
	t.Helper()

	userDir := t.TempDir()
	// os.UserConfigDir honours XDG_CONFIG_HOME
	t.Setenv("XDG_CONFIG_HOME", userDir)
	userPath = filepath.Join(userDir, "sami", "config.yaml")

	systemPath = filepath.Join(t.TempDir(), "config.yaml")
	t.Cleanup(func(orig string) func() { // restore package var
		return func() { SystemConfigPath = orig }
	}(SystemConfigPath))
	SystemConfigPath = systemPath

	return userPath, systemPath
}

func TestFind_UserOverridesSystemWholeFile(t *testing.T) {
	userPath, systemPath := redirectSearchPaths(t)

	writeConfigAt(t, systemPath)
	writeConfigAt(t, userPath)

	got, err := Find()
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got != userPath {
		t.Errorf("Find = %q, want user path %q", got, userPath)
	}
}

func TestFind_FallsBackToSystem(t *testing.T) {
	_, systemPath := redirectSearchPaths(t)
	writeConfigAt(t, systemPath)

	got, err := Find()
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if got != systemPath {
		t.Errorf("Find = %q, want system path %q", got, systemPath)
	}
}

func TestFind_NotFound(t *testing.T) {
	redirectSearchPaths(t) // create neither file

	_, err := Find()
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Find error = %v, want ErrNotFound", err)
	}
}

func TestLoad_ErrNotFound_IsWrappedComparisonFriendly(t *testing.T) {
	redirectSearchPaths(t)

	_, err := Load()
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load error = %v, want ErrNotFound", err)
	}
}

func writeConfigAt(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(canonicalYAML), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}
