// SPDX-License-Identifier: Apache-2.0

//go:build integration

package init

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNonInteractiveMode_CreatesExpectedFiles is an integration test that verifies
// the non-interactive init mode creates all expected repository files and directories.
//
// Run with: go test -tags=integration ./cmd/init/...
//
// Manual verification (playground/):
//
//	rm -rf playground/*
//	cd playground && ../build/anvil init \
//	  --key-name "Test Kernels" \
//	  --key-email "test@example.com"
//
// Expected output:
//
//	warning: not a git repository - consider running 'git init' first
//	✓ Repository initialized successfully
//	✓ anvil.yaml
//	✓ .gitignore
//	✓ configs/kernel-x86_64.config
//	✓ configs/kernel-aarch64.config
//
// Expected files:
//
//	anvil.yaml   - repo configuration with signing key info
//	.gitignore            - ignores build artifacts and local archives
//	configs/kernel-x86_64.config  - minimal x86_64 kernel config template
//	configs/kernel-aarch64.config - minimal aarch64 kernel config template
//	keys/                 - directory for signing keys
//	keys/history/         - directory for public key history
func TestNonInteractiveMode_CreatesExpectedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	flags := InitFlags{
		KeyName:       "Test Kernels",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	if err := runNonInteractiveWithFlags(flags); err != nil {
		t.Fatalf("runNonInteractiveWithFlags() failed: %v", err)
	}

	// Verify all expected files and directories exist
	expectedPaths := []struct {
		path  string
		isDir bool
	}{
		{"anvil.yaml", false},
		{".gitignore", false},
		{"configs/kernel-x86_64.config", false},
		{"configs/kernel-aarch64.config", false},
		{"keys", true},
		{"keys/history", true},
	}

	for _, ep := range expectedPaths {
		fullPath := filepath.Join(tmpDir, ep.path)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("expected %s to exist, got: %v", ep.path, err)
			continue
		}
		if ep.isDir && !info.IsDir() {
			t.Errorf("expected %s to be a directory", ep.path)
		}
		if !ep.isDir && info.IsDir() {
			t.Errorf("expected %s to be a file, not a directory", ep.path)
		}
	}
}

// TestNonInteractiveMode_RejectsReinit verifies that running init twice in the
// same directory fails with an appropriate error.
func TestNonInteractiveMode_RejectsReinit(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	flags := InitFlags{
		KeyName:       "Test Kernels",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	// First init should succeed
	if err := runNonInteractiveWithFlags(flags); err != nil {
		t.Fatalf("first runNonInteractiveWithFlags() failed: %v", err)
	}

	// Second init should fail with "already initialized" error
	// validatePreFlight is called by runInit (cobra handler) not runNonInteractiveWithFlags,
	// so we test validatePreFlight directly here.
	if err := validatePreFlight(); err == nil {
		t.Error("validatePreFlight() should return error when anvil.yaml already exists")
	}
}
