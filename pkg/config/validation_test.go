// SPDX-License-Identifier: Apache-2.0
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRepoPath_ValidPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"simple directory", "keys"},
		{"nested directory", "keys/signing"},
		{"with subdirs", "config/keys/current"},
		{"single char", "k"},
		{"with hyphens", "signing-keys"},
		{"with underscores", "signing_keys"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepoPath(tt.path)
			if err != nil {
				t.Errorf("validateRepoPath(%q) should accept valid path: %v", tt.path, err)
			}
		})
	}
}

func TestValidateRepoPath_PathTraversal(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../keys"},
		{"nested parent", "../../keys"},
		{"mixed traversal", "keys/../../../other"},
		{"absolute unix", "/etc/keys"},
		{"absolute unix 2", "/home/user/keys"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepoPath(tt.path)
			if err == nil {
				t.Errorf("validateRepoPath(%q) should reject path traversal", tt.path)
			}
		})
	}
}

func TestValidateRepoPath_ExistingFile(t *testing.T) {
	// Create a temporary directory and file
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create a file
	testFile := "testfile.txt"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should reject existing file
	err = validateRepoPath(testFile)
	if err == nil {
		t.Error("validateRepoPath should reject path pointing to existing file")
	}
}

func TestValidateRepoPath_ExistingDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create a directory
	testDir := "testdir"
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Should accept existing directory
	err = validateRepoPath(testDir)
	if err != nil {
		t.Errorf("validateRepoPath should accept existing directory: %v", err)
	}
}

func TestValidateRepoPath_NonExistentPath(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Non-existent path should be accepted (will be created later)
	err = validateRepoPath("nonexistent/path/to/keys")
	if err != nil {
		t.Errorf("validateRepoPath should accept non-existent path: %v", err)
	}
}

func TestValidateRepoPath_DotPaths(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		shouldErr bool
	}{
		{"current dir", ".", false},
		{"current dir with subdir", "./keys", false},
		{"parent dir", "..", true},
		{"parent then child", "../keys", true},
		{"nested with dots", "./keys/./current", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepoPath(tt.path)
			if tt.shouldErr && err == nil {
				t.Errorf("validateRepoPath(%q) should error", tt.path)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("validateRepoPath(%q) should not error: %v", tt.path, err)
			}
		})
	}
}

func TestValidateValue_SigningKeyLocation_RepoScope(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Valid path (non-existent) - repo scope
	err = ValidateValue("signing.key.location", "keys", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept valid path in repo scope: %v", err)
	}

	// Invalid path (parent traversal) - repo scope
	err = ValidateValue("signing.key.location", "../keys", ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject parent directory traversal in repo scope")
	}

	// Create a file and verify it's rejected - repo scope
	testFile := "testfile.txt"
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err = ValidateValue("signing.key.location", testFile, ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject path pointing to file in repo scope")
	}

	// Create a directory and verify it's accepted - repo scope
	testDir := "testdir"
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	err = ValidateValue("signing.key.location", testDir, ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept existing directory in repo scope: %v", err)
	}

	// Absolute path should be rejected in repo scope
	err = ValidateValue("signing.key.location", "/absolute/path", ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject absolute path in repo scope")
	}
}

func TestValidateValue_SigningKeyLocation_UserScope(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Valid absolute path (non-existent) - user scope
	nonExistentPath := filepath.Join(tmpDir, "nonexistent", "keys")
	err := ValidateValue("signing.key.location", nonExistentPath, ScopeUser)
	if err != nil {
		t.Errorf("ValidateValue should accept non-existent absolute path in user scope: %v", err)
	}

	// Create a file and verify it's rejected - user scope
	testFile := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err = ValidateValue("signing.key.location", testFile, ScopeUser)
	if err == nil {
		t.Error("ValidateValue should reject path pointing to file in user scope")
	}

	// Create a directory and verify it's accepted - user scope
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	err = ValidateValue("signing.key.location", testDir, ScopeUser)
	if err != nil {
		t.Errorf("ValidateValue should accept existing directory in user scope: %v", err)
	}

	// Absolute path should be allowed in user scope (XDG paths)
	absolutePath := filepath.Join(tmpDir, "keys")
	err = ValidateValue("signing.key.location", absolutePath, ScopeUser)
	if err != nil {
		t.Errorf("ValidateValue should accept absolute path in user scope: %v", err)
	}
}

func TestValidateKeyLocationPath_ValidPaths(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name string
		path string
	}{
		{"absolute non-existent", filepath.Join(tmpDir, "nonexistent")},
		{"absolute with subdirs", filepath.Join(tmpDir, "path", "to", "keys")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeyLocationPath(tt.path)
			if err != nil {
				t.Errorf("validateKeyLocationPath(%q) should accept valid path: %v", tt.path, err)
			}
		})
	}
}

func TestValidateKeyLocationPath_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "testfile.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	err := validateKeyLocationPath(testFile)
	if err == nil {
		t.Error("validateKeyLocationPath should reject existing file")
	}
}

func TestValidateKeyLocationPath_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	err := validateKeyLocationPath(testDir)
	if err != nil {
		t.Errorf("validateKeyLocationPath should accept existing directory: %v", err)
	}
}

func TestGetSigningKeyLocation_IgnoresEnvInRepoContext(t *testing.T) {
	// This test verifies that ENV variables are ignored when in a repo context
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Save and restore ENV
	oldEnv := os.Getenv("ANVIL_SIGNING_KEY_LOCATION")
	defer func() {
		if oldEnv != "" {
			os.Setenv("ANVIL_SIGNING_KEY_LOCATION", oldEnv)
		} else {
			os.Unsetenv("ANVIL_SIGNING_KEY_LOCATION")
		}
	}()

	// Set ENV variable
	os.Setenv("ANVIL_SIGNING_KEY_LOCATION", "/env/override")

	// Create repo config
	repoConfig := `signing:
  key:
    location: repo-keys`
	if err := os.WriteFile("anvil.yaml", []byte(repoConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// In repo context, ENV should be ignored
	location := GetSigningKeyLocation()
	if location != "repo-keys" {
		t.Errorf("GetSigningKeyLocation() in repo context should ignore ENV: got %q, want %q", location, "repo-keys")
	}

	// Remove repo config
	os.Remove("anvil.yaml")

	// Outside repo context, ENV should be respected
	// Note: This would need proper Viper initialization to work fully
	// For now, we just verify the repo context behavior
}

func TestValidateRepoFilePath_ValidPaths(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create test files
	testFiles := []string{
		"kernel.config",
		"configs/x86_64.config",
		"configs/arch/aarch64.config",
	}

	for _, file := range testFiles {
		dir := filepath.Dir(file)
		if dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(file, []byte("CONFIG=y"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range testFiles {
		t.Run(file, func(t *testing.T) {
			err := validateRepoFilePath(file)
			if err != nil {
				t.Errorf("validateRepoFilePath(%q) should accept valid file: %v", file, err)
			}
		})
	}
}

func TestValidateRepoFilePath_PathTraversal(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"parent directory", "../kernel.config"},
		{"nested parent", "../../configs/kernel.config"},
		{"mixed traversal", "configs/../../../other.config"},
		{"absolute unix", "/etc/kernel.config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRepoFilePath(tt.path)
			if err == nil {
				t.Errorf("validateRepoFilePath(%q) should reject path traversal", tt.path)
			}
			if !strings.Contains(err.Error(), "outside repository") && !strings.Contains(err.Error(), "relative to repository") {
				t.Errorf("Error should mention path restriction: %v", err)
			}
		})
	}
}

func TestValidateRepoFilePath_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = validateRepoFilePath("nonexistent.config")
	if err == nil {
		t.Error("validateRepoFilePath should reject non-existent file")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("Error should mention file doesn't exist: %v", err)
	}
}

func TestValidateRepoFilePath_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create a directory
	if err := os.Mkdir("configs", 0755); err != nil {
		t.Fatal(err)
	}

	err = validateRepoFilePath("configs")
	if err == nil {
		t.Error("validateRepoFilePath should reject directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("Error should mention directory: %v", err)
	}
}

func TestValidateValue_KernelConfigX86_64_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create test file
	if err := os.WriteFile("kernel-x86_64.config", []byte("CONFIG=y"), 0644); err != nil {
		t.Fatal(err)
	}

	err = ValidateValue("kernels.config.x86_64", "kernel-x86_64.config", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept valid kernel config file: %v", err)
	}
}

func TestValidateValue_KernelConfigX86_64_InvalidPaths(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
	}{
		{"non-existent", "nonexistent.config"},
		{"path traversal", "../kernel.config"},
		{"absolute path", "/etc/kernel.config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValue("kernels.config.x86_64", tt.path, ScopeRepo)
			if err == nil {
				t.Errorf("ValidateValue should reject invalid path: %q", tt.path)
			}
		})
	}
}

func TestValidateValue_KernelConfigAarch64_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create test file in subdirectory
	if err := os.MkdirAll("configs", 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("configs/aarch64.config", []byte("CONFIG=y"), 0644); err != nil {
		t.Fatal(err)
	}

	err = ValidateValue("kernels.config.aarch64", "configs/aarch64.config", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept valid kernel config file: %v", err)
	}
}

func TestValidateValue_KernelConfig_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create a directory
	if err := os.Mkdir("configs", 0755); err != nil {
		t.Fatal(err)
	}

	err = ValidateValue("kernels.config.x86_64", "configs", ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject directory for kernel config")
	}
}
