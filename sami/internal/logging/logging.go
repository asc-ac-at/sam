/*
Package logging provides logging directory setup for build commands.

The logging directory structure follows the pattern from the sami PoC:

	{base_path}/logs/{USER}/{YYYYMMDD}/sami.XXXXXXX.{SW_NAME}-{TOOLCHAIN}/

Where:
  - base_path: Configurable base path (default: /opt/adm/asc-software-stack)
  - USER: Current system user
  - YYYYMMDD: Current date
  - XXXXXXX: Random characters for uniqueness
  - SW_NAME, TOOLCHAIN: Provided by caller
*/
package logging

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

// DefaultBasePath is the default base path for logging directories
const DefaultBasePath = "/opt/adm/asc-software-stack"

// SetupLoggingDir creates the logging directory structure for a build command.
//
// Parameters:
//   - basePath: Base path for logs (e.g., /opt/adm/asc-software-stack)
//   - swName: Software name being built (e.g., buildenv-nvhpc-25.9)
//   - toolchain: Toolchain identifier (e.g., nvidia-nvhpc-25.9)
//
// Returns:
//   - string: Full path to the created logging directory
//   - error: Any error encountered during directory creation
//
// Example:
//
//	logDir, err := SetupLoggingDir("/opt/adm/asc-software-stack", "buildenv-nvhpc-25.9", "nvidia-nvhpc-25.9")
//	// Returns: /opt/adm/asc-software-stack/logs/<user>/<date>/sami.XXXXXXX.buildenv-nvhpc-25.9-nvidia-nvhpc-25.9
func SetupLoggingDir(basePath, swName, toolchain string) (string, error) {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	username := currentUser.Username

	// Get current date in YYYYMMDD format
	today := time.Now().Format("20060102")

	// Build the parent directory path
	logParent := filepath.Join(basePath, "logs", username, today)

	// Create parent directory structure
	if err := os.MkdirAll(logParent, 0755); err != nil {
		return "", fmt.Errorf("failed to create parent directory %s: %w", logParent, err)
	}

	// Create unique temp directory with pattern: sami.XXXXXXX.{SW_NAME}-{TOOLCHAIN}
	logDir, err := os.MkdirTemp(logParent, fmt.Sprintf("sami.*.%s-%s", swName, toolchain))
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	return logDir, nil
}

// GenerateLogPath generates the logging directory path without creating it.
// Useful for testing or when you need to know the path before creation.
//
// Parameters:
//   - basePath: Base path for logs
//   - swName: Software name being built
//   - toolchain: Toolchain identifier
//
// Returns:
//   - string: The path where the log directory would be created
//   - error: Any error encountered (e.g., getting current user)
func GenerateLogPath(basePath, swName, toolchain string) (string, error) {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	username := currentUser.Username

	// Get current date in YYYYMMDD format
	today := time.Now().Format("20060102")

	// Build and return the parent directory path
	return filepath.Join(basePath, "logs", username, today), nil
}
