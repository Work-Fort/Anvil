// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/github"
	"github.com/Work-Fort/Anvil/pkg/util"
)

// Get gets a kernel by trying to download pre-built version first, then building from source if needed
func Get(version string, buildOpts *BuildOptions) error {
	// Try to download pre-built kernel first
	if err := Download(version); err == nil {
		// Download successful
		return nil
	}

	// Download failed or not available, build from source
	fmt.Println()
	fmt.Println("[INFO] Pre-built kernel not available, building from source...")
	fmt.Println()

	// Use provided build options or create default ones
	opts := BuildOptions{
		Version:     version,
		Interactive: true, // Default to interactive for Get
	}
	if buildOpts != nil {
		opts = *buildOpts
	} else if opts.Version == "" {
		opts.Version = version
	}

	return Build(opts)
}

// Download downloads and verifies a kernel version with optional progress callback
func Download(version string) error {
	return DownloadWithProgress(version, nil, nil)
}

// DownloadWithProgress downloads and verifies a kernel version with progress and status tracking
func DownloadWithProgress(version string, progressCallback func(float64), statusCallback func(string)) error {
	arch, err := config.GetArch()
	if err != nil {
		return err
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return err
	}

	// Create GitHub client for API and downloads
	client := github.NewClient()

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
	outputDir := filepath.Join(config.GlobalPaths.KernelsDir, version)
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
	tempFile := filepath.Join(config.GlobalPaths.CacheDir, filename)

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
	checksumFile := filepath.Join(config.GlobalPaths.CacheDir, "SHA256SUMS")
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
	sigFile := filepath.Join(config.GlobalPaths.CacheDir, "SHA256SUMS.asc")
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
	log.Debug("Importing Cracker Barrel signing key")
	keyFile := filepath.Join(config.GlobalPaths.CacheDir, "signing-key.asc")
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

	fmt.Printf("✓ Kernel installed: %s\n", outputFile)
	fmt.Println()
	fmt.Println("To use with Firecracker:")
	fmt.Printf("  firecracker --config-file config.json (with \"kernel_image_path\": %q)\n", outputFile)
	fmt.Println()
	fmt.Println("To set as default:")
	fmt.Printf("  anvil set kernel %s\n", version)

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

// List lists installed kernel versions
func List() error {
	theme := config.CurrentTheme
	titleStyle := theme.InfoStyle().Bold(true)
	markerStyle := theme.SuccessStyle()
	versionStyle := theme.InfoStyle()
	subtleStyle := theme.SubtleStyle()

	arch, err := config.GetArch()
	if err != nil {
		return err
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return err
	}

	symlinkPath := filepath.Join(config.GlobalPaths.DataDir, kernelName)
	defaultVersion := ""

	// Check if there's a default version set
	if target, err := os.Readlink(symlinkPath); err == nil {
		// Extract version from path like: /path/to/kernels/6.18.9/vmlinux-6.18.9-x86_64
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "kernels" && i+1 < len(parts) {
				defaultVersion = parts[i+1]
				break
			}
		}
	}

	fmt.Println()
	fmt.Printf("%s %s\n", titleStyle.Render("Installed kernels"), subtleStyle.Render(fmt.Sprintf("(%s)", arch)))
	fmt.Println()

	entries, err := os.ReadDir(config.GlobalPaths.KernelsDir)
	if err != nil || len(entries) == 0 {
		fmt.Println(subtleStyle.Render("  No kernels installed"))
		fmt.Println()
		fmt.Println(subtleStyle.Render("Download a kernel with:"))
		fmt.Println(subtleStyle.Render("  anvil download kernel <version>"))
		return nil
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		kernelFile := filepath.Join(config.GlobalPaths.KernelsDir, version, fmt.Sprintf("%s-%s-%s", kernelName, version, arch))

		if _, err := os.Stat(kernelFile); err == nil {
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
		fmt.Println(subtleStyle.Render(fmt.Sprintf("  No kernels installed for %s", arch)))
		fmt.Println()
		fmt.Println(subtleStyle.Render("Download a kernel with:"))
		fmt.Println(subtleStyle.Render("  anvil download kernel <version>"))
	}

	fmt.Println()
	fmt.Println(subtleStyle.Render("Set default with:"))
	fmt.Println(subtleStyle.Render("  anvil set kernel <version>"))

	return nil
}

// Set sets a kernel version as default by creating a symlink
func Set(version string) error {
	arch, err := config.GetArch()
	if err != nil {
		return err
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return err
	}

	sourceFile := filepath.Join(config.GlobalPaths.KernelsDir, version, fmt.Sprintf("%s-%s-%s", kernelName, version, arch))
	symlinkPath := filepath.Join(config.GlobalPaths.DataDir, kernelName)

	// Check if version exists
	if _, err := os.Stat(sourceFile); err != nil {
		return fmt.Errorf("kernel %s not found. Download it first with: anvil download kernel %s", version, version)
	}

	log.Debugf("Setting kernel %s as default", version)

	// Remove existing symlink if present
	os.Remove(symlinkPath)

	// Create symlink
	if err := os.Symlink(sourceFile, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	fmt.Printf("✓ Kernel %s set as default\n\n", version)
	fmt.Printf("Symlink created: %s -> %s\n", symlinkPath, sourceFile)
	fmt.Println("Use with Firecracker config: \"kernel_image_path\":", symlinkPath)

	return nil
}

// ShowVersions shows available kernel versions from GitHub
func ShowVersions() error {
	theme := config.CurrentTheme
	titleStyle := theme.InfoStyle().Bold(true)
	defaultMarkerStyle := theme.SuccessStyle()
	installedMarkerStyle := theme.InfoStyle()
	versionStyle := theme.InfoStyle()
	subtleStyle := theme.SubtleStyle()

	log.Debug("Fetching available kernel versions from GitHub")

	client := github.NewClient()
	parts := strings.Split(config.GitHubRepo, "/")
	releases, err := client.GetReleases(parts[0], parts[1], 10)
	if err != nil {
		return fmt.Errorf("failed to fetch kernel versions: %w", err)
	}

	// Sort releases by semantic version (newest first)
	releases = github.SortReleasesBySemver(releases)

	arch, err := config.GetArch()
	if err != nil {
		return err
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return err
	}

	symlinkPath := filepath.Join(config.GlobalPaths.DataDir, kernelName)
	defaultVersion := ""

	// Check if there's a default version set
	if target, err := os.Readlink(symlinkPath); err == nil {
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "kernels" && i+1 < len(parts) {
				defaultVersion = parts[i+1]
				break
			}
		}
	}

	fmt.Println()
	fmt.Printf("%s %s\n", titleStyle.Render("Available kernel versions"), subtleStyle.Render("(latest 10)"))
	fmt.Println()

	if len(releases) == 0 {
		fmt.Println(subtleStyle.Render("  No kernel releases available yet"))
		fmt.Println()
		fmt.Println(subtleStyle.Render("Kernel releases are built automatically when new versions are released."))
		fmt.Println(subtleStyle.Render("You can also request a specific version by creating a build request issue."))
		return nil
	}

	for _, release := range releases {
		version := github.StripVersionPrefix(release.TagName)
		kernelFile := filepath.Join(config.GlobalPaths.KernelsDir, version, fmt.Sprintf("%s-%s-%s", kernelName, version, arch))

		if version == defaultVersion {
			fmt.Printf("  %s %s\n",
				defaultMarkerStyle.Render("·"),
				versionStyle.Render(version))
		} else if _, err := os.Stat(kernelFile); err == nil {
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
	fmt.Println(subtleStyle.Render("  anvil download kernel <version>"))

	return nil
}
