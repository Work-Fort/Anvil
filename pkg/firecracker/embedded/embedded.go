// SPDX-License-Identifier: Apache-2.0
package embedded

import (
	_ "embed"
	"fmt"
	"os"
)

// VsockServerBinary contains the embedded vsock-server-standalone binary
// This is embedded at build time from vsock-server-standalone in this directory
// The binary is copied here during the build process
//
//go:embed vsock-server-standalone
var VsockServerBinary []byte

// ExtractVsockServer extracts the embedded vsock-server binary to a temporary file
// Returns the path to the extracted binary and a cleanup function
func ExtractVsockServer() (path string, cleanup func(), err error) {
	if len(VsockServerBinary) == 0 {
		return "", nil, fmt.Errorf("vsock-server binary not embedded (build with: task go:build)")
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "vsock-server-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	tmpPath := tmpFile.Name()

	// Write embedded binary
	if _, err := tmpFile.Write(VsockServerBinary); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("failed to write binary: %w", err)
	}

	// Close file
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return "", nil, fmt.Errorf("failed to make binary executable: %w", err)
	}

	// Return path and cleanup function
	cleanup = func() {
		os.Remove(tmpPath)
	}

	return tmpPath, cleanup, nil
}

// GetVsockServerSize returns the size of the embedded vsock-server binary in bytes
func GetVsockServerSize() int {
	return len(VsockServerBinary)
}

// IsVsockServerEmbedded returns true if the vsock-server binary is embedded
func IsVsockServerEmbedded() bool {
	return len(VsockServerBinary) > 0
}
