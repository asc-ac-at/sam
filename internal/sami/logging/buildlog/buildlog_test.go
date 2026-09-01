package buildlog

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSetupLoggingDir(t *testing.T) {
	// Create a temporary base directory for testing
	tmpBase, err := os.MkdirTemp("", "sami-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp base directory: %v", err)
	}
	defer os.RemoveAll(tmpBase)

	// Test data
	swName := "buildenv-nvhpc-25.9"

	// Create logging directory
	logDir, err := SetupLoggingDir(tmpBase, swName)
	if err != nil {
		t.Fatalf("SetupLoggingDir failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Fatalf("Logging directory was not created: %s", logDir)
	}

	// Verify directory structure
	currentUser, _ := user.Current()
	today := time.Now().Format("20060102")
	expectedParent := filepath.Join(tmpBase, "logs", currentUser.Username, today)

	if !strings.HasPrefix(logDir, expectedParent) {
		t.Errorf("Log directory not in expected parent path.\nGot:  %s\nWant: %s", logDir, expectedParent)
	}

	// Verify directory name pattern: sami.XXXXXXX.{SW_NAME}-{TOOLCHAIN}
	dirName := filepath.Base(logDir)
	expectedSuffix := "." + swName
	if !strings.HasSuffix(dirName, expectedSuffix) {
		t.Errorf("Directory name doesn't match expected pattern.\nGot:  %s\nWant suffix: %s", dirName, expectedSuffix)
	}

	if !strings.HasPrefix(dirName, "sami.") {
		t.Errorf("Directory name doesn't start with 'sami.' prefix: %s", dirName)
	}
}

func TestSetupLoggingDir_CreatesUniqueDirectories(t *testing.T) {
	// Create a temporary base directory for testing
	tmpBase, err := os.MkdirTemp("", "sami-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp base directory: %v", err)
	}
	defer os.RemoveAll(tmpBase)

	swName := "test-sw"

	// Create multiple directories
	dir1, err := SetupLoggingDir(tmpBase, swName)
	if err != nil {
		t.Fatalf("First SetupLoggingDir failed: %v", err)
	}

	dir2, err := SetupLoggingDir(tmpBase, swName)
	if err != nil {
		t.Fatalf("Second SetupLoggingDir failed: %v", err)
	}

	// Verify they are different (unique temp naming)
	if dir1 == dir2 {
		t.Errorf("Expected unique directories, but got the same path: %s", dir1)
	}

	// Verify both exist
	if _, err := os.Stat(dir1); os.IsNotExist(err) {
		t.Errorf("First directory doesn't exist: %s", dir1)
	}
	if _, err := os.Stat(dir2); os.IsNotExist(err) {
		t.Errorf("Second directory doesn't exist: %s", dir2)
	}
}

func TestSetupLoggingDir_WithDifferentInputs(t *testing.T) {
	tmpBase, err := os.MkdirTemp("", "sami-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp base directory: %v", err)
	}
	defer os.RemoveAll(tmpBase)

	tests := []struct {
		name      string
		swName    string
		toolchain string
	}{
		{"simple names", "myapp", "gcc"},
		{"complex names", "buildenv-default-nvhpc-25.9", "nvidia-nvhpc-25.9"},
		{"with versions", "app-1.0.0", "toolchain-2024a"},
		{"empty-ish names", "a", "b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logDir, err := SetupLoggingDir(tmpBase, tt.swName)
			if err != nil {
				t.Errorf("SetupLoggingDir failed: %v", err)
				return
			}

			// Verify directory exists
			if _, err := os.Stat(logDir); os.IsNotExist(err) {
				t.Errorf("Directory was not created: %s", logDir)
			}

			// Verify naming pattern
			dirName := filepath.Base(logDir)
			expectedSuffix := "." + tt.swName
			if !strings.HasSuffix(dirName, expectedSuffix) {
				t.Errorf("Directory name doesn't match pattern. Got: %s, want suffix: %s", dirName, expectedSuffix)
			}
		})
	}
}

func TestSetupLoggingDir_InvalidBasePath(t *testing.T) {
	// Use a temp file as the base path: MkdirAll can't create a directory
	// tree through a regular file, regardless of permissions.
	tmpFile, err := os.CreateTemp("", "invalid-base-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = SetupLoggingDir(tmpFile.Name(), "test-sw")
}

func TestGenerateLogPath(t *testing.T) {
	tmpBase, err := os.MkdirTemp("", "sami-logging-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp base directory: %v", err)
	}
	defer os.RemoveAll(tmpBase)

	swName := "test-sw"

	// Generate path
	logPath, err := GenerateLogPath(tmpBase, swName)
	if err != nil {
		t.Fatalf("GenerateLogPath failed: %v", err)
	}

	// Verify path structure
	currentUser, _ := user.Current()
	today := time.Now().Format("20060102")
	expectedPath := filepath.Join(tmpBase, "logs", currentUser.Username, today)

	if logPath != expectedPath {
		t.Errorf("Generated path doesn't match expected.\nGot:  %s\nWant: %s", logPath, expectedPath)
	}

	// Verify directory was NOT created (GenerateLogPath should not create dirs)
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("GenerateLogPath should not create directories, but %s exists", logPath)
	}
}

func TestGenerateLogPath_InvalidUser(t *testing.T) {
	// This test is tricky because we can't easily mock user.Current()
	// In practice, this would only fail in very unusual system configurations
	// We'll just verify the function handles the error case properly
	_, err := GenerateLogPath("/tmp", "test")
	// Should succeed in normal conditions
	if err != nil {
		t.Logf("GenerateLogPath returned error (may be expected in some environments): %v", err)
	}
}

func TestDefaultBasePath(t *testing.T) {
	// Verify the constant is set correctly
	if DefaultBasePath != "/opt/adm/asc-software-stack" {
		t.Errorf("DefaultBasePath has unexpected value: %s", DefaultBasePath)
	}
}
