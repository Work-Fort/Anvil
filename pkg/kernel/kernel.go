// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/github"
	"github.com/Work-Fort/Anvil/pkg/util"
	"github.com/charmbracelet/log"
)

// KernelInfo describes an installed kernel version
type KernelInfo struct {
	Version   string   `json:"version"`
	IsDefault bool     `json:"is_default"`
	Files     []string `json:"files"`
	Path      string   `json:"path"`
}

// AvailableVersion describes a kernel version available for download
type AvailableVersion struct {
	Version     string `json:"version"`
	IsInstalled bool   `json:"is_installed"`
	IsDefault   bool   `json:"is_default"`
}

// Get gets a kernel by trying to download pre-built version first, then building from source if needed
func Get(version string, client *github.Client, paths *config.Paths, buildOpts *BuildOptions) error {
	// Try to download pre-built kernel first
	if err := Download(version, client, paths); err == nil {
		// Download successful
		return nil
	}

	// Download failed or not available, build from source
	// Use provided build options or create default ones
	opts := BuildOptions{
		Version: version,
	}
	if buildOpts != nil {
		opts = *buildOpts
	} else if opts.Version == "" {
		opts.Version = version
	}

	return Build(opts, paths)
}

// Download downloads and verifies a kernel version with optional progress callback
func Download(version string, client *github.Client, paths *config.Paths) error {
	return DownloadWithProgress(version, client, paths, nil, nil)
}

// DownloadWithProgress downloads and verifies a kernel version with progress and status tracking
func DownloadWithProgress(version string, client *github.Client, paths *config.Paths, progressCallback func(float64), statusCallback func(string)) error {
	arch, err := config.GetArch()
	if err != nil {
		return err
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return err
	}

	// If no version specified, get latest
	if version == "" {
		parts := strings.Split(config.GitHubRepo, "/")
		release, err := client.GetLatestRelease(parts[0], parts[1])
		if err != nil {
			return fmt.Errorf("failed to fetch latest kernel version: %w", err)
		}
		version = github.StripVersionPrefix(release.TagName)
		log.Debugf("Using latest kernel version: %s", version)
	}

	filename := fmt.Sprintf("%s-%s-%s.xz", kernelName, version, arch)
	outputDir := filepath.Join(paths.KernelsDir, version)
	outputFile := filepath.Join(outputDir, fmt.Sprintf("%s-%s-%s", kernelName, version, arch))

	// Check if already downloaded
	if _, err := os.Stat(outputFile); err == nil {
		log.Infof("Kernel already exists: %s", outputFile)
		return nil
	}

	log.Debugf("Downloading kernel %s for %s", version, arch)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	releaseURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s", config.GitHubRepo, version)
	tempFile := filepath.Join(paths.CacheDir, filename)

	// Download compressed kernel
	if statusCallback != nil {
		statusCallback("Downloading kernel...")
	}
	if progressCallback != nil {
		progressCallback(0) // Reset to 0 for this step
	}
	log.Debugf("Downloading from: %s/%s", releaseURL, filename)
	if err := client.DownloadFile(fmt.Sprintf("%s/%s", releaseURL, filename), tempFile, progressCallback); err != nil {
		return fmt.Errorf("failed to download kernel: %w", err)
	}

	// Download checksums
	if statusCallback != nil {
		statusCallback("Downloading checksums...")
	}
	if progressCallback != nil {
		progressCallback(0) // Reset to 0 for this step
	}
	log.Debug("Downloading checksums")
	checksumFile := filepath.Join(paths.CacheDir, "SHA256SUMS")
	if err := client.DownloadFile(fmt.Sprintf("%s/SHA256SUMS", releaseURL), checksumFile, progressCallback); err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	// Download signature
	if statusCallback != nil {
		statusCallback("Downloading signature...")
	}
	if progressCallback != nil {
		progressCallback(0) // Reset to 0 for this step
	}
	log.Debug("Downloading PGP signature")
	sigFile := filepath.Join(paths.CacheDir, "SHA256SUMS.asc")
	if err := client.DownloadFile(fmt.Sprintf("%s/SHA256SUMS.asc", releaseURL), sigFile, progressCallback); err != nil {
		return fmt.Errorf("failed to download PGP signature: %w", err)
	}

	// Download signing key
	if statusCallback != nil {
		statusCallback("Downloading signing key...")
	}
	if progressCallback != nil {
		progressCallback(0) // Reset to 0 for this step
	}
	log.Debug("Importing Anvil signing key")
	keyFile := filepath.Join(paths.CacheDir, "signing-key.asc")
	if err := client.DownloadFile(fmt.Sprintf("%s/signing-key.asc", releaseURL), keyFile, progressCallback); err != nil {
		return fmt.Errorf("failed to download signing key: %w", err)
	}

	// Import GPG key
	if statusCallback != nil {
		statusCallback("Importing GPG key...")
	}
	if progressCallback != nil {
		progressCallback(0)
	}
	cmd := exec.Command("gpg", "--import", "--quiet", keyFile)
	if err := cmd.Run(); err != nil {
		// Ignore errors - key might already be imported
	}
	if progressCallback != nil {
		progressCallback(1.0)
	}

	// Verify PGP signature
	if statusCallback != nil {
		statusCallback("Verifying PGP signature...")
	}
	if progressCallback != nil {
		progressCallback(0)
	}
	log.Debug("Verifying PGP signature")
	cmd = exec.Command("gpg", "--verify", sigFile, checksumFile)
	output, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "Good signature") {
		return fmt.Errorf("PGP signature verification failed")
	}
	if progressCallback != nil {
		progressCallback(1.0)
	}

	// Verify compressed file checksum
	if statusCallback != nil {
		statusCallback("Verifying compressed checksum...")
	}
	if progressCallback != nil {
		progressCallback(0)
	}
	log.Debug("Verifying compressed kernel checksum")
	if err := util.VerifySHA256File(tempFile, checksumFile); err != nil {
		return fmt.Errorf("compressed kernel checksum verification failed: %w", err)
	}
	if progressCallback != nil {
		progressCallback(1.0)
	}

	// Decompress - this is the slowest operation
	if statusCallback != nil {
		statusCallback("Decompressing kernel...")
	}
	if progressCallback != nil {
		progressCallback(0) // Reset to 0 for this step
	}
	log.Debug("Decompressing kernel")
	// Note: DecompressXZWithProgress will report progress from 0-100% as it reads the compressed file
	if err := util.DecompressXZWithProgress(tempFile, outputFile, progressCallback); err != nil {
		return fmt.Errorf("failed to decompress kernel: %w", err)
	}

	// Verify decompressed kernel checksum
	if statusCallback != nil {
		statusCallback("Verifying decompressed checksum...")
	}
	if progressCallback != nil {
		progressCallback(0)
	}
	log.Debug("Verifying decompressed kernel checksum")
	if err := util.VerifySHA256File(outputFile, checksumFile); err != nil {
		os.Remove(outputFile)
		return fmt.Errorf("decompressed kernel checksum verification failed: %w", err)
	}
	if progressCallback != nil {
		progressCallback(1.0)
	}

	// Clean up
	if statusCallback != nil {
		statusCallback("Cleaning up...")
	}
	if progressCallback != nil {
		progressCallback(0)
	}
	os.Remove(tempFile)
	os.Remove(checksumFile)
	os.Remove(sigFile)
	os.Remove(keyFile)
	if progressCallback != nil {
		progressCallback(1.0)
	}

	// Done
	if statusCallback != nil {
		statusCallback("Installation complete!")
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// List returns installed kernel versions with their metadata.
func List(paths *config.Paths) ([]KernelInfo, string, error) {
	arch, err := config.GetArch()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get architecture: %w", err)
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get kernel name: %w", err)
	}

	// Determine default version from symlink
	defaultVersion := ""
	kernelSymlink := filepath.Join(paths.DataDir, kernelName)
	if target, err := os.Readlink(kernelSymlink); err == nil {
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "kernels" && i+1 < len(parts) {
				defaultVersion = parts[i+1]
				break
			}
		}
	}

	entries, err := os.ReadDir(paths.KernelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, arch, nil
		}
		return nil, arch, fmt.Errorf("failed to read kernels directory: %w", err)
	}

	var kernels []KernelInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := entry.Name()
		versionDir := filepath.Join(paths.KernelsDir, version)
		ki := KernelInfo{
			Version:   version,
			IsDefault: version == defaultVersion,
			Path:      versionDir,
		}

		// List files in version directory
		files, err := os.ReadDir(versionDir)
		if err == nil {
			for _, f := range files {
				ki.Files = append(ki.Files, f.Name())
			}
		}
		kernels = append(kernels, ki)
	}

	return kernels, arch, nil
}

// Set sets a kernel version as default by creating a symlink
func Set(version string, paths *config.Paths) error {
	arch, err := config.GetArch()
	if err != nil {
		return fmt.Errorf("failed to get architecture: %w", err)
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return fmt.Errorf("failed to get kernel name: %w", err)
	}

	sourceFile := filepath.Join(paths.KernelsDir, version, fmt.Sprintf("%s-%s-%s", kernelName, version, arch))
	symlinkPath := filepath.Join(paths.DataDir, kernelName)

	// Check if version exists
	if _, err := os.Stat(sourceFile); err != nil {
		return fmt.Errorf("kernel %s not found for %s", version, arch)
	}

	log.Debugf("Setting kernel %s as default", version)

	// Remove existing symlink if present
	os.Remove(symlinkPath)

	// Create symlink
	if err := os.Symlink(sourceFile, symlinkPath); err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	return nil
}

// Remove removes an installed kernel version.
func Remove(version string, paths *config.Paths) error {
	kernelDir := filepath.Join(paths.KernelsDir, version)
	if _, err := os.Stat(kernelDir); err != nil {
		return fmt.Errorf("kernel version %s not found", version)
	}

	log.Debugf("Removing kernel %s", version)

	if err := os.RemoveAll(kernelDir); err != nil {
		return fmt.Errorf("failed to remove kernel: %w", err)
	}

	return nil
}

// Clean removes installed kernel versions. If keepDefault is true, the default
// version is preserved; otherwise all versions and the default symlink are removed.
// Returns the list of removed version strings.
func Clean(keepDefault bool, paths *config.Paths) ([]string, error) {
	if !keepDefault {
		// Remove entire kernels directory
		if err := os.RemoveAll(paths.KernelsDir); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove kernels: %w", err)
		}

		removed := []string{"All kernels"}

		// Remove kernel symlink
		kernelName, err := config.GetKernelName()
		if err == nil {
			symlinkPath := filepath.Join(paths.DataDir, kernelName)
			os.Remove(symlinkPath)
			removed = append(removed, "Kernel symlink")
		}

		return removed, nil
	}

	// Remove only non-default kernel versions
	defaultVersion := ""
	kernelName, err := config.GetKernelName()
	if err == nil {
		kernelSymlink := filepath.Join(paths.DataDir, kernelName)
		if target, linkErr := os.Readlink(kernelSymlink); linkErr == nil {
			parts := strings.Split(target, "/")
			for i, part := range parts {
				if part == "kernels" && i+1 < len(parts) {
					defaultVersion = parts[i+1]
					break
				}
			}
		}
	}

	entries, err := os.ReadDir(paths.KernelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read kernels directory: %w", err)
	}

	var removed []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "default" {
			continue
		}

		version := entry.Name()
		if version == defaultVersion {
			continue
		}

		path := filepath.Join(paths.KernelsDir, version)
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("failed to remove %s: %w", path, err)
		}
		removed = append(removed, version)
	}

	if removed == nil {
		removed = []string{}
	}

	return removed, nil
}

// CleanBuildCache removes the kernel build cache. If all is true, the entire
// build directory is removed; otherwise only the build subdirectory is removed
// (keeping the source cache). Returns a status string and the cleaned path.
func CleanBuildCache(all bool, paths *config.Paths) (status string, cleanedPath string, err error) {
	buildDir := paths.KernelBuildDir
	if _, err := os.Stat(buildDir); err != nil {
		if os.IsNotExist(err) {
			return "clean", buildDir, nil
		}
		return "", "", err
	}

	if all {
		if err := os.RemoveAll(buildDir); err != nil {
			return "", "", fmt.Errorf("failed to clean build cache: %w", err)
		}
		return "cleaned", buildDir, nil
	}

	// Remove only the build subdirectory, keep source cache
	buildSubdir := filepath.Join(buildDir, "build")
	if err := os.RemoveAll(buildSubdir); err != nil {
		return "", "", fmt.Errorf("failed to clean build output: %w", err)
	}

	return "cleaned", buildSubdir, nil
}

// ShowVersions returns available kernel versions from GitHub with install status.
func ShowVersions(client *github.Client, paths *config.Paths) ([]AvailableVersion, error) {
	log.Debug("Fetching available kernel versions from GitHub")
	parts := strings.Split(config.GitHubRepo, "/")
	releases, err := client.GetReleases(parts[0], parts[1], 10)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch kernel versions: %w", err)
	}

	// Sort releases by semantic version (newest first)
	releases = github.SortReleasesBySemver(releases)

	arch, err := config.GetArch()
	if err != nil {
		return nil, fmt.Errorf("failed to get architecture: %w", err)
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return nil, fmt.Errorf("failed to get kernel name: %w", err)
	}

	// Determine default version from symlink
	defaultVersion := ""
	symlinkPath := filepath.Join(paths.DataDir, kernelName)
	if target, err := os.Readlink(symlinkPath); err == nil {
		symParts := strings.Split(target, "/")
		for i, part := range symParts {
			if part == "kernels" && i+1 < len(symParts) {
				defaultVersion = symParts[i+1]
				break
			}
		}
	}

	var versions []AvailableVersion
	for _, release := range releases {
		version := github.StripVersionPrefix(release.TagName)
		kernelFile := filepath.Join(paths.KernelsDir, version, fmt.Sprintf("%s-%s-%s", kernelName, version, arch))

		av := AvailableVersion{
			Version:   version,
			IsDefault: version == defaultVersion,
		}

		if _, err := os.Stat(kernelFile); err == nil {
			av.IsInstalled = true
		}

		versions = append(versions, av)
	}

	return versions, nil
}
