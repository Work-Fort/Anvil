// SPDX-License-Identifier: Apache-2.0
package cmdutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/Work-Fort/Anvil/pkg/github"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/Work-Fort/Anvil/pkg/ui"
	"golang.org/x/term"
)

// IsInteractive checks if stdin is connected to a terminal AND the user wants TUI mode
func IsInteractive() bool {
	// Check both terminal capability and user preference
	return term.IsTerminal(int(os.Stdin.Fd())) && config.GetUseTUI()
}

// IsVersionDownloaded checks if a specific version is already downloaded
func IsVersionDownloaded(target, version string) bool {
	switch target {
	case "kernel":
		arch, err := config.GetArch()
		if err != nil {
			return false
		}
		kernelName, err := config.GetKernelName()
		if err != nil {
			return false
		}
		kernelFile := filepath.Join(config.GlobalPaths.KernelsDir, version, fmt.Sprintf("%s-%s-%s", kernelName, version, arch))
		_, err = os.Stat(kernelFile)
		return err == nil

	case "firecracker":
		fcFile := filepath.Join(config.GlobalPaths.FirecrackerDir, version, "firecracker")
		_, err := os.Stat(fcFile)
		return err == nil

	default:
		return false
	}
}

// DeleteVersion deletes a downloaded version
func DeleteVersion(target, version string) error {
	log.Debugf("deleteVersion: Called with target=%s version=%s", target, version)

	var versionDir string

	switch target {
	case "kernel":
		versionDir = filepath.Join(config.GlobalPaths.KernelsDir, version)
	case "firecracker":
		versionDir = filepath.Join(config.GlobalPaths.FirecrackerDir, version)
	default:
		log.Debugf("deleteVersion: Unknown target: %s", target)
		return fmt.Errorf("unknown target: %s", target)
	}

	log.Debugf("deleteVersion: Version directory to delete: %s", versionDir)

	// Check if this is the default version
	var symlinkPath string
	switch target {
	case "kernel":
		kernelName, err := config.GetKernelName()
		if err != nil {
			log.Debugf("deleteVersion: GetKernelName failed: %v", err)
			return err
		}
		symlinkPath = filepath.Join(config.GlobalPaths.DataDir, kernelName)
	case "firecracker":
		symlinkPath = filepath.Join(config.GlobalPaths.BinDir, "firecracker")
	}

	// Check if this version is the default
	if target, err := os.Readlink(symlinkPath); err == nil {
		if strings.Contains(target, version) {
			log.Debugf("deleteVersion: Removing symlink: %s", symlinkPath)
			// Remove the symlink
			os.Remove(symlinkPath)
		}
	}

	// Remove the version directory
	log.Debugf("deleteVersion: Calling os.RemoveAll(%s)", versionDir)

	if err := os.RemoveAll(versionDir); err != nil {
		log.Debugf("deleteVersion: os.RemoveAll failed: %v", err)
		return fmt.Errorf("failed to delete version: %w", err)
	}

	log.Debugf("deleteVersion: Successfully deleted directory")

	theme := config.CurrentTheme
	fmt.Println()
	fmt.Println(theme.SuccessMessage(fmt.Sprintf("Deleted %s version %s", target, version)))

	return nil
}

// GetDefaultVersion returns the currently set default version for the target
func GetDefaultVersion(target string) string {
	var symlinkPath string

	switch target {
	case "kernel":
		kernelName, err := config.GetKernelName()
		if err != nil {
			return ""
		}
		symlinkPath = filepath.Join(config.GlobalPaths.DataDir, kernelName)
	case "firecracker":
		symlinkPath = filepath.Join(config.GlobalPaths.BinDir, "firecracker")
	default:
		return ""
	}

	// Read the symlink
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return ""
	}

	// Extract version from path
	parts := strings.Split(target, "/")
	for i, part := range parts {
		if (part == "kernels" || part == "firecracker") && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// ShowVersionSelector displays an interactive TUI to select a version
func ShowVersionSelector(target string) error {
	// Helper function to fetch and categorize versions
	fetchVersions := func() ([]string, []string, error) {
		client := github.NewClient()
		var releases []github.Release
		var err error

		switch target {
		case "kernel":
			parts := strings.Split(config.GitHubRepo, "/")
			releases, err = client.GetReleases(parts[0], parts[1], 10)
		case "firecracker":
			parts := strings.Split(config.FirecrackerRepo, "/")
			releases, err = client.GetReleases(parts[0], parts[1], 10)
		default:
			return nil, nil, fmt.Errorf("unknown target: %s", target)
		}

		if err != nil {
			return nil, nil, fmt.Errorf("failed to fetch versions: %w", err)
		}

		if len(releases) == 0 {
			return nil, nil, fmt.Errorf("no releases found")
		}

		// Sort releases by semantic version (newest first)
		releases = github.SortReleasesBySemver(releases)

		// Separate downloaded and available versions from GitHub releases
		var downloadedVersions []string
		var availableVersions []string
		downloadedSet := make(map[string]bool) // Track what we've already listed

		for _, release := range releases {
			version := github.StripVersionPrefix(release.TagName)

			if IsVersionDownloaded(target, version) {
				downloadedVersions = append(downloadedVersions, version)
				downloadedSet[version] = true
			} else {
				availableVersions = append(availableVersions, version)
			}
		}

		// Also scan local directory for versions not in GitHub releases (e.g., built kernels)
		var localDir string
		switch target {
		case "kernel":
			localDir = config.GlobalPaths.KernelsDir
		case "firecracker":
			localDir = config.GlobalPaths.FirecrackerDir
		}

		if entries, err := os.ReadDir(localDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				version := entry.Name()
				// Only add if not already in the list from GitHub releases
				if !downloadedSet[version] {
					downloadedVersions = append(downloadedVersions, version)
					downloadedSet[version] = true
				}
			}
		}

		if len(availableVersions) == 0 && len(downloadedVersions) == 0 {
			return nil, nil, fmt.Errorf("no versions found")
		}

		return downloadedVersions, availableVersions, nil
	}

	// Fetch initial versions
	downloadedVersions, availableVersions, err := fetchVersions()
	if err != nil {
		return err
	}

	// Create callback functions
	downloadFn := func(version string, progressCallback func(float64), statusCallback func(string)) error {
		switch target {
		case "kernel":
			return kernel.DownloadWithProgress(version, progressCallback, statusCallback)
		case "firecracker":
			return firecracker.DownloadWithProgress(version, progressCallback, statusCallback)
		default:
			return fmt.Errorf("unknown target: %s", target)
		}
	}

	setDefaultFn := func(version string) error {
		switch target {
		case "kernel":
			return kernel.Set(version)
		case "firecracker":
			return firecracker.Set(version)
		default:
			return fmt.Errorf("unknown target: %s", target)
		}
	}

	deleteFn := func(version string) error {
		return DeleteVersion(target, version)
	}

	getDefaultVerFn := func() string {
		return GetDefaultVersion(target)
	}

	// Run the interactive selector
	return ui.RunVersionSelector(target, downloadedVersions, availableVersions, downloadFn, setDefaultFn, deleteFn, fetchVersions, getDefaultVerFn)
}
