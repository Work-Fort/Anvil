// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/github"
	"github.com/Work-Fort/Anvil/pkg/util"
)

// Download downloads a Firecracker binary
func Download(version string) error {
	return DownloadWithProgress(version, nil, nil)
}

// DownloadWithProgress downloads a Firecracker binary with progress and status tracking
func DownloadWithProgress(version string, progressCallback func(float64), statusCallback func(string)) error {
	arch, err := config.GetArch()
	if err != nil {
		return err
	}

	// Create GitHub client for API and downloads
	client := github.NewClient()

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

	outputDir := filepath.Join(config.GlobalPaths.FirecrackerDir, version)
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
	tempFile := filepath.Join(config.GlobalPaths.CacheDir, filename)

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
	if err := util.ExtractTarGz(tempFile, config.GlobalPaths.CacheDir); err != nil {
		return fmt.Errorf("failed to extract Firecracker: %w", err)
	}

	// Move binary to final location
	if statusCallback != nil {
		statusCallback("Installing binary...")
	}
	extractedDir := filepath.Join(config.GlobalPaths.CacheDir, fmt.Sprintf("release-v%s-%s", version, fcArch))
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

	fmt.Printf("✓ Firecracker installed: %s\n", outputFile)
	fmt.Println()
	fmt.Println("To use:")
	fmt.Printf("  %s --version\n", outputFile)
	fmt.Println()
	fmt.Println("To set as default:")
	fmt.Printf("  anvil set firecracker %s\n", version)

	return nil
}

// List lists installed Firecracker versions
func List() error {
	theme := config.CurrentTheme
	titleStyle := theme.InfoStyle().Bold(true)
	markerStyle := theme.SuccessStyle()
	versionStyle := theme.InfoStyle()
	subtleStyle := theme.SubtleStyle()

	symlinkPath := filepath.Join(config.GlobalPaths.BinDir, "firecracker")
	defaultVersion := ""

	// Check if there's a default version set
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

	fmt.Println()
	fmt.Println(titleStyle.Render("Installed Firecracker versions"))
	fmt.Println()

	entries, err := os.ReadDir(config.GlobalPaths.FirecrackerDir)
	if err != nil || len(entries) == 0 {
		fmt.Println(subtleStyle.Render("  No Firecracker versions installed"))
		fmt.Println()
		fmt.Println(subtleStyle.Render("Download Firecracker with:"))
		fmt.Println(subtleStyle.Render("  anvil download firecracker <version>"))
		return nil
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		fcFile := filepath.Join(config.GlobalPaths.FirecrackerDir, version, "firecracker")

		if _, err := os.Stat(fcFile); err == nil {
			found = true
			if version == defaultVersion {
				fmt.Printf("  %s %s %s\n",
					markerStyle.Render("●"),
					versionStyle.Render(version),
					subtleStyle.Render("(default)"))
			} else {
				fmt.Printf("    %s\n", versionStyle.Render(version))
			}
		}
	}

	if !found {
		fmt.Println(subtleStyle.Render("  No Firecracker versions installed"))
		fmt.Println()
		fmt.Println(subtleStyle.Render("Download Firecracker with:"))
		fmt.Println(subtleStyle.Render("  anvil download firecracker <version>"))
	}

	fmt.Println()
	fmt.Println(subtleStyle.Render("Set default with:"))
	fmt.Println(subtleStyle.Render("  anvil set firecracker <version>"))

	return nil
}

// Set sets a Firecracker version as default by creating a symlink
func Set(version string) error {
	sourceFile := filepath.Join(config.GlobalPaths.FirecrackerDir, version, "firecracker")
	symlinkPath := filepath.Join(config.GlobalPaths.BinDir, "firecracker")

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

	fmt.Printf("✓ Firecracker %s set as default\n\n", version)
	fmt.Println("Run 'firecracker --version' to verify")

	return nil
}

// ShowVersions shows available Firecracker versions from GitHub
func ShowVersions() error {
	theme := config.CurrentTheme
	titleStyle := theme.InfoStyle().Bold(true)
	defaultMarkerStyle := theme.SuccessStyle()
	installedMarkerStyle := theme.InfoStyle()
	versionStyle := theme.InfoStyle()
	subtleStyle := theme.SubtleStyle()

	log.Debug("Fetching available Firecracker versions from GitHub")

	client := github.NewClient()
	parts := strings.Split(config.FirecrackerRepo, "/")
	releases, err := client.GetReleases(parts[0], parts[1], 10)
	if err != nil {
		return fmt.Errorf("failed to fetch Firecracker versions: %w", err)
	}

	// Sort releases by semantic version (newest first)
	releases = github.SortReleasesBySemver(releases)

	symlinkPath := filepath.Join(config.GlobalPaths.BinDir, "firecracker")
	defaultVersion := ""

	// Check if there's a default version set
	if target, err := os.Readlink(symlinkPath); err == nil {
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "firecracker" && i+1 < len(parts) {
				defaultVersion = parts[i+1]
				break
			}
		}
	}

	fmt.Println()
	fmt.Printf("%s %s\n", titleStyle.Render("Available Firecracker versions"), subtleStyle.Render("(latest 10)"))
	fmt.Println()

	if len(releases) == 0 {
		fmt.Println(subtleStyle.Render("  No Firecracker releases found"))
		fmt.Println()
		fmt.Println(subtleStyle.Render("This is unexpected - check your network connection or GitHub API access."))
		return nil
	}

	for _, release := range releases {
		version := github.StripVersionPrefix(release.TagName)
		fcFile := filepath.Join(config.GlobalPaths.FirecrackerDir, version, "firecracker")

		if version == defaultVersion {
			fmt.Printf("  %s %s\n",
				defaultMarkerStyle.Render("·"),
				versionStyle.Render(version))
		} else if _, err := os.Stat(fcFile); err == nil {
			fmt.Printf("  %s %s\n",
				installedMarkerStyle.Render("-"),
				versionStyle.Render(version))
		} else {
			fmt.Printf("    %s\n", versionStyle.Render(version))
		}
	}

	fmt.Println()
	fmt.Println(subtleStyle.Render("Legend: · default  - installed"))
	fmt.Println()
	fmt.Println(subtleStyle.Render("Download with:"))
	fmt.Println(subtleStyle.Render("  anvil download firecracker <version>"))

	return nil
}
