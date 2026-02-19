// SPDX-License-Identifier: Apache-2.0
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetConfigValue_ValidatesScope(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	GlobalPaths = &Paths{
		ConfigDir: filepath.Join(tmpDir, "config"),
	}
	os.MkdirAll(GlobalPaths.ConfigDir, 0755)

	// All keys can now be set in any scope (precedence handles conflicts)

	// User-recommended key in repo scope (should succeed - repo can override)
	err := SetConfigValue("use-tui", "true", ScopeRepo)
	if err != nil {
		t.Errorf("SetConfigValue should allow user-recommended key in repo scope: %v", err)
	}

	// Repo-recommended key in user scope (should succeed - user can set defaults)
	err = SetConfigValue("signing.key.name", "Test", ScopeUser)
	if err != nil {
		t.Errorf("SetConfigValue should allow repo-recommended key in user scope: %v", err)
	}
}

func TestSetConfigValue_ValidatesValue(t *testing.T) {
	tmpDir := t.TempDir()
	GlobalPaths = &Paths{
		ConfigDir: filepath.Join(tmpDir, "config"),
	}
	os.MkdirAll(GlobalPaths.ConfigDir, 0755)

	// Try to set invalid enum value (should fail)
	err := SetConfigValue("log-level", "invalid-level", ScopeUser)
	if err == nil {
		t.Error("SetConfigValue should reject invalid enum value")
	}

	// Try to set valid enum value (should succeed)
	err = SetConfigValue("log-level", "info", ScopeUser)
	if err != nil {
		t.Errorf("SetConfigValue should accept valid enum: %v", err)
	}
}

func TestFlattenKeys(t *testing.T) {
	nested := map[string]interface{}{
		"top": "value",
		"signing": map[string]interface{}{
			"key": map[string]interface{}{
				"name": "Test",
			},
		},
	}

	keys := flattenKeys(nested, "")
	expectedKeys := []string{"top", "signing.key.name"}

	if len(keys) != len(expectedKeys) {
		t.Errorf("flattenKeys returned %d keys, want %d", len(keys), len(expectedKeys))
	}

	for _, expected := range expectedKeys {
		found := false
		for _, key := range keys {
			if key == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("flattenKeys missing key: %s", expected)
		}
	}
}

func TestWarnMisplacedKeys(t *testing.T) {
	// This test verifies warnMisplacedKeys doesn't crash
	// Actual warnings go to log, which we won't capture in tests

	tmpDir := t.TempDir()
	GlobalPaths = &Paths{
		ConfigDir: filepath.Join(tmpDir, "config"),
	}
	os.MkdirAll(GlobalPaths.ConfigDir, 0755)

	// Create a config file with misplaced key
	configPath := filepath.Join(GlobalPaths.ConfigDir, "config.yaml")
	content := "signing.key.name: Test\n" // Repo key in user config (wrong)
	os.WriteFile(configPath, []byte(content), 0644)

	// Call warnMisplacedKeys - should not panic
	warnMisplacedKeys(GlobalPaths.ConfigDir, "user")
}

func TestSetConfigValue_ForbiddenKeyInRepoScope(t *testing.T) {
	tmpDir := t.TempDir()
	GlobalPaths = &Paths{
		ConfigDir: filepath.Join(tmpDir, "config"),
	}
	os.MkdirAll(GlobalPaths.ConfigDir, 0755)

	// Try to set forbidden key in repo scope
	err := SetConfigValue("github-token", "ghp_1234567890abcdefABCDEF", ScopeRepo)
	if err == nil {
		t.Error("SetConfigValue should reject forbidden key in repo scope")
	}
	if !strings.Contains(err.Error(), "cannot be set in repo config") {
		t.Errorf("Error should mention repo restriction: %v", err)
	}
	if !strings.Contains(err.Error(), "sensitive") {
		t.Errorf("Error should mention sensitive nature: %v", err)
	}
}

func TestSetConfigValue_ForbiddenKeyInUserScope(t *testing.T) {
	tmpDir := t.TempDir()
	GlobalPaths = &Paths{
		ConfigDir: filepath.Join(tmpDir, "config"),
	}
	os.MkdirAll(GlobalPaths.ConfigDir, 0755)

	// Set github-token in user scope
	err := SetConfigValue("github-token", "ghp_1234567890abcdefABCDEF", ScopeUser)
	if err != nil {
		t.Errorf("SetConfigValue should allow github-token in user scope: %v", err)
	}

	// Verify it was written
	configPath := filepath.Join(GlobalPaths.ConfigDir, "config.yaml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	if !strings.Contains(string(content), "github-token") {
		t.Error("github-token should be written to user config")
	}
}
