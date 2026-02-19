// SPDX-License-Identifier: Apache-2.0
package init

import (
	"os"
	"testing"
)

func TestGenerateRepoFiles_Success(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "anvil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create settings
	settings := InitSettings{
		KeyName:       "Test Key",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	// Generate files
	files, err := GenerateRepoFiles(settings)
	if err != nil {
		t.Fatalf("GenerateRepoFiles failed: %v", err)
	}

	// Verify files were returned
	if len(files) == 0 {
		t.Error("Expected files to be created, got empty list")
	}

	// Verify all files exist
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", file)
		}
	}
}

func TestGenerateRepoFiles_AllExpectedFiles(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "anvil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create settings
	settings := InitSettings{
		KeyName:       "Test Key",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	// Generate files
	_, err = GenerateRepoFiles(settings)
	if err != nil {
		t.Fatalf("GenerateRepoFiles failed: %v", err)
	}

	// Check expected files
	expectedFiles := []string{
		"anvil.yaml",
		".gitignore",
		"configs/kernel-x86_64.config",
		"configs/kernel-aarch64.config",
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file not created: %s", file)
		}
	}
}

func TestGenerateRepoFiles_DirectoryStructure(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "anvil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create settings
	settings := InitSettings{
		KeyName:       "Test Key",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	// Generate files
	_, err = GenerateRepoFiles(settings)
	if err != nil {
		t.Fatalf("GenerateRepoFiles failed: %v", err)
	}

	// Check expected directories
	expectedDirs := []string{
		"configs",
		"keys",
		"keys/history",
	}

	for _, dir := range expectedDirs {
		info, err := os.Stat(dir)
		if os.IsNotExist(err) {
			t.Errorf("Expected directory not created: %s", dir)
		} else if !info.IsDir() {
			t.Errorf("Expected %s to be a directory", dir)
		}
	}
}

func TestGenerateRepoFiles_TemplateRendering(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "anvil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create settings with specific values
	settings := InitSettings{
		KeyName:       "Test Key Name",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "2y",
		KeyFormat:     "binary",
		HistoryFormat: "binary",
	}

	// Generate files
	_, err = GenerateRepoFiles(settings)
	if err != nil {
		t.Fatalf("GenerateRepoFiles failed: %v", err)
	}

	// Read the generated config file
	content, err := os.ReadFile("anvil.yaml")
	if err != nil {
		t.Fatalf("Failed to read anvil.yaml: %v", err)
	}

	contentStr := string(content)

	// Verify template values were rendered
	expectedStrings := []string{
		`name: "Test Key Name"`,
		`email: "test@example.com"`,
		`expiry: "2y"`,
		`format: "binary"`,
		`format: "binary"`, // history format
	}

	for _, expected := range expectedStrings {
		if !containsString(contentStr, expected) {
			t.Errorf("Expected config to contain %q, but it didn't.\nContent:\n%s", expected, contentStr)
		}
	}
}

func TestGenerateRepoFiles_RollbackOnError(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "anvil-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create a file that will conflict with directory creation
	if err := os.WriteFile("configs", []byte("conflict"), 0644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}

	// Create settings
	settings := InitSettings{
		KeyName:       "Test Key",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	// Generate files - should fail
	_, err = GenerateRepoFiles(settings)
	if err == nil {
		t.Fatal("Expected GenerateRepoFiles to fail due to conflict, but it succeeded")
	}

	// Verify rollback - check that files that might have been created are cleaned up
	// The .gitignore and anvil.yaml might have been created before the error
	if _, err := os.Stat("anvil.yaml"); err == nil {
		t.Error("Expected anvil.yaml to be rolled back, but it still exists")
	}

	if _, err := os.Stat(".gitignore"); err == nil {
		t.Error("Expected .gitignore to be rolled back, but it still exists")
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr ||
		(len(s) > len(substr) && containsString(s[1:], substr)))))
}
