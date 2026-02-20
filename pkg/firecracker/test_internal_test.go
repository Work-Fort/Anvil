// anvil/pkg/firecracker/test_internal_test.go
package firecracker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Work-Fort/Anvil/pkg/config"
)

// TestGetFirecrackerBinary_WithVersionedSymlink tests parsing version from symlink target
func TestGetFirecrackerBinary_WithVersionedSymlink(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	firecrackerDir := filepath.Join(tmpDir, "firecracker")
	versionDir := filepath.Join(firecrackerDir, "1.14.1")

	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	// Create the firecracker binary
	binaryPath := filepath.Join(versionDir, "firecracker")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

	// Create symlink: default -> 1.14.1/firecracker (full path)
	defaultLink := filepath.Join(firecrackerDir, "default")
	if err := os.Symlink(binaryPath, defaultLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Override global config for test
	oldDataDir := config.GlobalPaths.DataDir
	config.GlobalPaths.DataDir = tmpDir
	defer func() { config.GlobalPaths.DataDir = oldDataDir }()

	// Call getFirecrackerBinary
	result, err := getFirecrackerBinary()
	if err != nil {
		t.Fatalf("getFirecrackerBinary() failed: %v", err)
	}

	// Verify the path is correct
	expectedPath := filepath.Join(tmpDir, "firecracker", "1.14.1", "firecracker")
	if result != expectedPath {
		t.Errorf("getFirecrackerBinary() = %q, want %q", result, expectedPath)
	}

	// Verify the file exists at the returned path
	if _, err := os.Stat(result); err != nil {
		t.Errorf("returned path does not exist: %v", err)
	}
}

// TestGetFirecrackerBinary_NoSymlink tests error when symlink doesn't exist
func TestGetFirecrackerBinary_NoSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Override global config for test
	oldDataDir := config.GlobalPaths.DataDir
	config.GlobalPaths.DataDir = tmpDir
	defer func() { config.GlobalPaths.DataDir = oldDataDir }()

	// Call getFirecrackerBinary with no symlink
	_, err := getFirecrackerBinary()
	if err == nil {
		t.Error("getFirecrackerBinary() should fail when symlink doesn't exist")
	}
}

// TestGetFirecrackerBinary_BrokenSymlink tests error when binary doesn't exist
func TestGetFirecrackerBinary_BrokenSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	firecrackerDir := filepath.Join(tmpDir, "firecracker")

	if err := os.MkdirAll(firecrackerDir, 0755); err != nil {
		t.Fatalf("failed to create firecracker dir: %v", err)
	}

	// Create symlink to non-existent path
	defaultLink := filepath.Join(firecrackerDir, "default")
	nonExistentPath := filepath.Join(tmpDir, "firecracker", "1.14.1", "firecracker")
	if err := os.Symlink(nonExistentPath, defaultLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// Override global config for test
	oldDataDir := config.GlobalPaths.DataDir
	config.GlobalPaths.DataDir = tmpDir
	defer func() { config.GlobalPaths.DataDir = oldDataDir }()

	// Call getFirecrackerBinary
	_, err := getFirecrackerBinary()
	if err == nil {
		t.Error("getFirecrackerBinary() should fail when binary doesn't exist")
	}
}
