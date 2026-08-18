package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotFound is returned by Find when no sami config exists in any of the
// standard locations. Callers may compare with errors.Is and degrade
// gracefully (warn + render-only) rather than hard-fail.
var ErrNotFound = errors.New("no sami config found")

// SystemConfigPath is a var (not const) so tests can redirect it into a
// tempdir.
var SystemConfigPath = "/etc/sami/config.yaml"

// userConfigPath returns the per-user config location
// "$XDG_CONFIG_HOME/sami/config.yaml", honouring XDG_CONFIG_HOME via
// os.UserConfigDir.
func userConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving user config dir: %w", err)
	}
	return filepath.Join(dir, "sami", "config.yaml"), nil
}

// Find locates the config file: the per-user path overrides the system-wide
// path, whole-file (no per-key merging). Returns ErrNotFound when neither
// exists.
func Find() (string, error) {
	if up, err := userConfigPath(); err == nil {
		if _, err := os.Stat(up); err == nil {
			return up, nil
		}
	}
	if _, err := os.Stat(SystemConfigPath); err == nil {
		return SystemConfigPath, nil
	}
	return "", ErrNotFound
}

// Load finds and parses the config. Returns ErrNotFound (wrapped) when no
// config file exists in any standard location.
func Load() (*File, error) {
	path, err := Find()
	if err != nil {
		return nil, err
	}
	return LoadSbatchConfig(path)
}
