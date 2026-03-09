// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/github"
	"github.com/Work-Fort/Anvil/pkg/util"
	"github.com/charmbracelet/log"
)

// FirecrackerInfo describes an installed Firecracker version
type FirecrackerInfo struct {
	Version   string `json:"version"`
	IsDefault bool   `json:"is_default"`
	Path      string `json:"path"`
}

// AvailableFirecracker describes a Firecracker version available for download
type AvailableFirecracker struct {
	Version     string `json:"version"`
	IsInstalled bool   `json:"is_installed"`
	IsDefault   bool   `json:"is_default"`
}

// Download downloads a Firecracker binary
func Download(version string, paths *config.Paths) error {
	return DownloadWithProgress(version, paths, nil, nil)
}

// DownloadWithProgress downloads a Firecracker binary with progress and status tracking
func DownloadWithProgress(version string, paths *config.Paths, progressCallback func(float64), statusCallback func(string)) error {
	arch, err := config.GetArch()
	if err != nil {
		return err
	}

	// Create GitHub client for API and downloads
	client := github.NewClient(config.GetGitHubToken(), config.GitHubAPI)

	// If no version specified, get latest
	if version == "" {
		parts := strings.Split(config.FirecrackerRepo, "/")
		release, err := client.GetLatestRelease(parts[0], parts[1])
		if err != nil {
			return fmt.Errorf("failed to fetch latest Firecracker version: %w", err)
		}
		version = github.StripVersionPrefix(release.TagName)
		log.Debugf("Using latest Firecracker version: %s", version)
	}

	outputDir := filepath.Join(paths.FirecrackerDir, version)
	outputFile := filepath.Join(outputDir, "firecracker")

	// Check if already downloaded
	if _, err := os.Stat(outputFile); err == nil {
		log.Infof("Firecracker already exists: %s", outputFile)
		return nil
	}

	log.Debugf("Downloading Firecracker %s for %s", version, arch)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Firecracker uses the same arch naming for aarch64
	fcArch := arch

	releaseURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s", config.FirecrackerRepo, version)
	filename := fmt.Sprintf("firecracker-v%s-%s.tgz", version, fcArch)
	tempFile := filepath.Join(paths.CacheDir, filename)

	// Download with automatic GitHub token injection and progress tracking
	if statusCallback != nil {
		statusCallback("Downloading Firecracker...")
	}
	log.Debugf("Downloading from: %s/%s", releaseURL, filename)
	downloadURL := fmt.Sprintf("%s/%s", releaseURL, filename)
	if err := client.DownloadFile(downloadURL, tempFile, progressCallback); err != nil {
		return fmt.Errorf("failed to download Firecracker: %w", err)
	}

	// Extract
	if statusCallback != nil {
		statusCallback("Extracting archive...")
	}
	log.Debug("Extracting Firecracker")
	if err := util.ExtractTarGz(tempFile, paths.CacheDir); err != nil {
		return fmt.Errorf("failed to extract Firecracker: %w", err)
	}

	// Move binary to final location
	if statusCallback != nil {
		statusCallback("Installing binary...")
	}
	extractedDir := filepath.Join(paths.CacheDir, fmt.Sprintf("release-v%s-%s", version, fcArch))
	extractedBinary := filepath.Join(extractedDir, fmt.Sprintf("firecracker-v%s-%s", version, fcArch))

	data, err := os.ReadFile(extractedBinary)
	if err != nil {
		return fmt.Errorf("failed to read extracted binary: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0755); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}

	// Clean up
	if statusCallback != nil {
		statusCallback("Cleaning up temporary files...")
	}
	os.Remove(tempFile)
	os.RemoveAll(extractedDir)

	// Done
	if statusCallback != nil {
		statusCallback("Installation complete!")
	}

	return nil
}

// List returns installed Firecracker versions with their metadata.
func List(paths *config.Paths) ([]FirecrackerInfo, error) {
	// Determine default version from symlink
	defaultVersion := ""
	symlinkPath := filepath.Join(paths.BinDir, "firecracker")
	if target, err := os.Readlink(symlinkPath); err == nil {
		// Extract version from path like: /path/to/firecracker/1.11.1/firecracker
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "firecracker" && i+1 < len(parts) {
				defaultVersion = parts[i+1]
				break
			}
		}
	}

	entries, err := os.ReadDir(paths.FirecrackerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read firecracker directory: %w", err)
	}

	var versions []FirecrackerInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := entry.Name()
		fcFile := filepath.Join(paths.FirecrackerDir, version, "firecracker")

		if _, err := os.Stat(fcFile); err == nil {
			versions = append(versions, FirecrackerInfo{
				Version:   version,
				IsDefault: version == defaultVersion,
				Path:      fcFile,
			})
		}
	}

	return versions, nil
}

// Set sets a Firecracker version as default by creating a symlink
func Set(version string, paths *config.Paths) error {
	sourceFile := filepath.Join(paths.FirecrackerDir, version, "firecracker")
	symlinkPath := filepath.Join(paths.BinDir, "firecracker")

	// Check if version exists
	if _, err := os.Stat(sourceFile); err != nil {
		return fmt.Errorf("firecracker %s not found. Download it first with: anvil download firecracker %s", version, version)
	}

	log.Debugf("Setting Firecracker %s as default", version)

	// Remove existing symlink if present
	os.Remove(symlinkPath)

	// Create symlink
	if err := os.Symlink(sourceFile, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// Remove removes an installed Firecracker version
func Remove(version string, paths *config.Paths) error {
	versionDir := filepath.Join(paths.FirecrackerDir, version)
	if _, err := os.Stat(versionDir); err != nil {
		return fmt.Errorf("firecracker %s not found", version)
	}

	log.Debugf("Removing Firecracker %s", version)

	// Check if this version is the default and remove symlink if so
	symlinkPath := filepath.Join(paths.BinDir, "firecracker")
	if target, err := os.Readlink(symlinkPath); err == nil {
		if strings.Contains(target, version) {
			os.Remove(symlinkPath)
		}
	}

	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove firecracker %s: %w", version, err)
	}

	return nil
}

// ShowVersions returns available Firecracker versions from GitHub with install status.
func ShowVersions(paths *config.Paths) ([]AvailableFirecracker, error) {
	log.Debug("Fetching available Firecracker versions from GitHub")

	client := github.NewClient(config.GetGitHubToken(), config.GitHubAPI)
	parts := strings.Split(config.FirecrackerRepo, "/")
	releases, err := client.GetReleases(parts[0], parts[1], 10)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Firecracker versions: %w", err)
	}

	// Sort releases by semantic version (newest first)
	releases = github.SortReleasesBySemver(releases)

	// Determine default version from symlink
	defaultVersion := ""
	symlinkPath := filepath.Join(paths.BinDir, "firecracker")
	if target, err := os.Readlink(symlinkPath); err == nil {
		symParts := strings.Split(target, "/")
		for i, part := range symParts {
			if part == "firecracker" && i+1 < len(symParts) {
				defaultVersion = symParts[i+1]
				break
			}
		}
	}

	var versions []AvailableFirecracker
	for _, release := range releases {
		version := github.StripVersionPrefix(release.TagName)
		fcFile := filepath.Join(paths.FirecrackerDir, version, "firecracker")

		av := AvailableFirecracker{
			Version:   version,
			IsDefault: version == defaultVersion,
		}

		if _, err := os.Stat(fcFile); err == nil {
			av.IsInstalled = true
		}

		versions = append(versions, av)
	}

	return versions, nil
}
