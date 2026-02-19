// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpTimeout is the timeout for HTTP requests to kernel.org.
// Prevents indefinite hangs in CI if kernel.org is slow or unreachable.
const httpTimeout = 30 * time.Second

// VersionCheckResult holds the outcome of a version check.
type VersionCheckResult struct {
	Version        string `json:"version"`
	Available      bool   `json:"available"`
	ChecksumsReady bool   `json:"checksums_ready"`
	Buildable      bool   `json:"buildable"`
	Message        string `json:"message,omitempty"`
	ChecksumsURL   string `json:"checksums_url,omitempty"`
	SourceURL      string `json:"source_url,omitempty"`
}

// CheckVersion validates that a kernel version is available on kernel.org
// and has checksums ready for verified builds. If version is empty or "latest",
// the latest stable version is resolved automatically.
//
// Returns (*VersionCheckResult, nil) for all check outcomes (including "not buildable").
// Returns (nil, error) only for hard failures (e.g., cannot reach kernel.org for
// version resolution).
func CheckVersion(version string) (*VersionCheckResult, error) {
	// Resolve "latest" or empty to actual version
	if version == "" || version == "latest" {
		resolved, err := GetLatestKernelVersion()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve latest kernel version: %w", err)
		}
		version = resolved
	}

	majorVersion := strings.Split(version, ".")[0]
	checksumsURL := fmt.Sprintf("https://cdn.kernel.org/pub/linux/kernel/v%s.x/sha256sums.asc", majorVersion)
	sourceURL := fmt.Sprintf("https://cdn.kernel.org/pub/linux/kernel/v%s.x/linux-%s.tar.xz", majorVersion, version)

	result := &VersionCheckResult{
		Version:      version,
		ChecksumsURL: checksumsURL,
		SourceURL:    sourceURL,
	}

	// Step 1: Check if version exists in kernel.org releases
	if err := ValidateVersion(version); err != nil {
		result.Available = false
		result.Buildable = false
		result.Message = fmt.Sprintf("Version %s not found in kernel.org releases", version)
		return result, nil
	}
	result.Available = true

	// Step 2: Check if checksums file is accessible and contains this version
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(checksumsURL)
	if err != nil {
		result.ChecksumsReady = false
		result.Buildable = false
		result.Message = fmt.Sprintf("Cannot fetch checksums: %v", err)
		return result, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.ChecksumsReady = false
		result.Buildable = false
		result.Message = fmt.Sprintf("Checksums file returned HTTP %d", resp.StatusCode)
		return result, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.ChecksumsReady = false
		result.Buildable = false
		result.Message = fmt.Sprintf("Failed to read checksums file: %v", err)
		return result, nil
	}

	tarballName := fmt.Sprintf("linux-%s.tar.xz", version)
	if strings.Contains(string(body), tarballName) {
		result.ChecksumsReady = true
		result.Buildable = true
	} else {
		result.ChecksumsReady = false
		result.Buildable = false
		result.Message = "Checksums not yet available for this version (normal right after a kernel release)"
	}

	return result, nil
}
