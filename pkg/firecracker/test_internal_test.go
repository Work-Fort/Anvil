// anvil/pkg/firecracker/test_internal_test.go
package firecracker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Work-Fort/Anvil/pkg/config"
)

// TestGetFirecrackerBinary_WithVersionedSymlink tests resolving binary via BinDir symlink
func TestGetFirecrackerBinary_WithVersionedSymlink(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	firecrackerDir := filepath.Join(tmpDir, "firecracker")
	versionDir := filepath.Join(firecrackerDir, "1.14.1")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("failed to create version dir: %v", err)
	}

	// Create the firecracker binary
	binaryPath := filepath.Join(versionDir, "firecracker")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("failed to create binary: %v", err)
	}

	// Create symlink: bin/firecracker -> full path to versioned binary
	defaultLink := filepath.Join(binDir, "firecracker")
	if err := os.Symlink(binaryPath, defaultLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	paths := &config.Paths{DataDir: tmpDir, BinDir: binDir}

	// Call getFirecrackerBinary
	result, err := getFirecrackerBinary(paths)
	if err != nil {
		t.Fatalf("getFirecrackerBinary() failed: %v", err)
	}

	// Verify the path is correct
	if result != binaryPath {
		t.Errorf("getFirecrackerBinary() = %q, want %q", result, binaryPath)
	}

	// Verify the file exists at the returned path
	if _, err := os.Stat(result); err != nil {
		t.Errorf("returned path does not exist: %v", err)
	}
}

// TestGetFirecrackerBinary_NoSymlink tests error when symlink doesn't exist
func TestGetFirecrackerBinary_NoSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	paths := &config.Paths{DataDir: tmpDir, BinDir: binDir}

	// Call getFirecrackerBinary with no symlink
	_, err := getFirecrackerBinary(paths)
	if err == nil {
		t.Error("getFirecrackerBinary() should fail when symlink doesn't exist")
	}
}

// TestGetFirecrackerBinary_BrokenSymlink tests error when binary doesn't exist
func TestGetFirecrackerBinary_BrokenSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	// Create symlink to non-existent path
	defaultLink := filepath.Join(binDir, "firecracker")
	nonExistentPath := filepath.Join(tmpDir, "firecracker", "1.14.1", "firecracker")
	if err := os.Symlink(nonExistentPath, defaultLink); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	paths := &config.Paths{DataDir: tmpDir, BinDir: binDir}

	// Call getFirecrackerBinary
	_, err := getFirecrackerBinary(paths)
	if err == nil {
		t.Error("getFirecrackerBinary() should fail when binary doesn't exist")
	}
}
