// SPDX-License-Identifier: Apache-2.0
package update

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/github"
	"github.com/Work-Fort/Anvil/pkg/util"
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates the update command
func NewUpdateCmd(version, disableUpdate string) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update anvil to the latest version",
		Long: `Update the anvil binary to the latest version from GitHub releases.

This command:
  1. Checks for the latest release on GitHub
  2. Downloads the appropriate binary for your architecture
  3. Verifies the PGP signature
  4. Verifies the SHA256 checksum
  5. Replaces the current binary

Security:
  - All downloads are verified with PGP signatures
  - Checksums are validated before installation
  - Uses the official signing key from releases`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateSelf(version, disableUpdate)
		},
	}
}

func updateSelf(version, disableUpdate string) error {
	theme := config.CurrentTheme

	// Check if updates are disabled (set by package managers)
	if disableUpdate == "true" {
		fmt.Println()
		fmt.Printf("%s Updates are disabled for this installation\n", theme.WarningIndicator())
		fmt.Println()
		fmt.Println("This version was installed by a package manager.")
		fmt.Println("Use your package manager to update:")
		fmt.Println()
		fmt.Printf("  • Arch Linux (AUR):  %s\n", theme.InfoStyle().Render("yay -Syu anvil"))
		fmt.Printf("  • Debian/Ubuntu:     %s\n", theme.InfoStyle().Render("sudo apt update && sudo apt upgrade anvil"))
		fmt.Printf("  • Fedora/RHEL:       %s\n", theme.InfoStyle().Render("sudo dnf update anvil"))
		fmt.Printf("  • Generic:           Check your package manager's documentation\n")
		fmt.Println()
		return nil
	}

	log.Info("Checking for anvil updates...")

	// Get latest CLI release (tagged with cli-v prefix)
	client := github.NewClient()
	parts := strings.Split(config.GitHubRepo, "/")

	// Fetch recent releases and find the latest CLI release
	releases, err := client.GetReleases(parts[0], parts[1], 20)
	if err != nil {
		return fmt.Errorf("failed to fetch releases: %w", err)
	}

	// Find the latest release with cli-v prefix
	var release *github.Release
	for i := range releases {
		if strings.HasPrefix(releases[i].TagName, "cli-v") {
			release = &releases[i]
			break
		}
	}

	if release == nil {
		return fmt.Errorf("no CLI releases found (looking for cli-v* tags)")
	}

	// Extract version from tag (cli-v1.0.0 -> 1.0.0)
	latestVersion := strings.TrimPrefix(release.TagName, "cli-v")

	// Compare versions (handle both cli-v and v prefixes for current version)
	currentVersion := strings.TrimPrefix(strings.TrimPrefix(version, "cli-v"), "v")
	if latestVersion == currentVersion {
		fmt.Printf("%s Already on latest version: %s\n", theme.CompleteIndicator(), currentVersion)
		return nil
	}

	fmt.Printf("%s New version available: %s (current: %s)\n", theme.InfoStyle().Render("→"), latestVersion, currentVersion)

	// Detect architecture
	arch := runtime.GOARCH
	if arch == "amd64" {
		// Keep as is
	} else if arch == "arm64" {
		// Keep as is
	} else {
		return fmt.Errorf("unsupported architecture: %s (supported: amd64, arm64)", arch)
	}

	binaryName := fmt.Sprintf("anvil-linux-%s", arch)
	compressedBinaryName := binaryName + ".xz"

	// Find required assets
	var binaryURL, checksumsURL, signatureURL, publicKeyURL string
	for _, asset := range release.Assets {
		switch asset.Name {
		case compressedBinaryName:
			binaryURL = asset.BrowserDownloadURL
		case "SHA256SUMS":
			checksumsURL = asset.BrowserDownloadURL
		case "SHA256SUMS.asc":
			signatureURL = asset.BrowserDownloadURL
		case "signing-key.asc":
			publicKeyURL = asset.BrowserDownloadURL
		}
	}

	if binaryURL == "" {
		return fmt.Errorf("could not find binary asset '%s' in release %s", compressedBinaryName, release.TagName)
	}
	if checksumsURL == "" {
		return fmt.Errorf("could not find SHA256SUMS in release %s", release.TagName)
	}
	if signatureURL == "" {
		return fmt.Errorf("could not find SHA256SUMS.asc in release %s", release.TagName)
	}
	if publicKeyURL == "" {
		return fmt.Errorf("could not find signing-key.asc in release %s", release.TagName)
	}

	// Create temp directory for downloads
	tempDir := filepath.Join(config.GlobalPaths.CacheDir, "update-"+latestVersion)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	compressedBinaryPath := filepath.Join(tempDir, compressedBinaryName)
	binaryPath := filepath.Join(tempDir, binaryName)
	checksumsPath := filepath.Join(tempDir, "SHA256SUMS")
	signaturePath := filepath.Join(tempDir, "SHA256SUMS.asc")
	publicKeyPath := filepath.Join(tempDir, "signing-key.asc")

	// Download all files
	log.Info("Downloading update files...")

	if err := client.DownloadFile(binaryURL, compressedBinaryPath, nil); err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	fmt.Printf("  %s Downloaded %s\n", theme.CompleteIndicator(), compressedBinaryName)

	if err := client.DownloadFile(checksumsURL, checksumsPath, nil); err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	fmt.Printf("  %s Downloaded SHA256SUMS\n", theme.CompleteIndicator())

	if err := client.DownloadFile(signatureURL, signaturePath, nil); err != nil {
		return fmt.Errorf("failed to download signature: %w", err)
	}
	fmt.Printf("  %s Downloaded SHA256SUMS.asc\n", theme.CompleteIndicator())

	if err := client.DownloadFile(publicKeyURL, publicKeyPath, nil); err != nil {
		return fmt.Errorf("failed to download public key: %w", err)
	}
	fmt.Printf("  %s Downloaded signing-key.asc\n", theme.CompleteIndicator())

	// Verify signature
	log.Info("Verifying signature...")
	if err := verifySignature(checksumsPath, signaturePath, publicKeyPath); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	fmt.Printf("  %s Signature verified\n", theme.CompleteIndicator())

	// Verify checksum of compressed file
	log.Info("Verifying checksum...")
	if err := verifyChecksum(compressedBinaryPath, checksumsPath, compressedBinaryName); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	fmt.Printf("  %s Checksum verified\n", theme.CompleteIndicator())

	// Decompress binary
	log.Info("Decompressing binary...")
	if err := util.DecompressXZ(compressedBinaryPath, binaryPath); err != nil {
		return fmt.Errorf("failed to decompress binary: %w", err)
	}
	fmt.Printf("  %s Decompressed binary\n", theme.CompleteIndicator())

	// Make executable
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %w", err)
	}

	// Get current binary path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks if any
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		realPath = exePath
	}

	// Replace current binary
	log.Info("Installing update...")
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to read new binary: %w", err)
	}

	if err := os.WriteFile(realPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	fmt.Println()
	fmt.Printf("%s Updated to version %s\n", theme.CompleteIndicator(), latestVersion)
	fmt.Println()
	fmt.Println(theme.SubtleStyle().Render("Run 'anvil version' to verify"))

	return nil
}

// verifySignature verifies the PGP signature of the checksums file
func verifySignature(checksumsPath, signaturePath, publicKeyPath string) error {
	// Import the public key into a temporary GPG keyring
	cmd := exec.Command("gpg", "--no-default-keyring", "--keyring", "trustedkeys.gpg", "--import", publicKeyPath)
	cmd.Dir = filepath.Dir(publicKeyPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to import public key: %w\nOutput: %s", err, output)
	}

	// Verify the signature
	cmd = exec.Command("gpg", "--no-default-keyring", "--keyring", "trustedkeys.gpg", "--verify", signaturePath, checksumsPath)
	cmd.Dir = filepath.Dir(checksumsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("GPG verification failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// verifyChecksum verifies the SHA256 checksum of the downloaded binary
func verifyChecksum(binaryPath, checksumsPath, binaryName string) error {
	// Calculate actual checksum
	f, err := os.Open(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to open binary: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))

	// Read expected checksum from SHA256SUMS
	checksumFile, err := os.Open(checksumsPath)
	if err != nil {
		return fmt.Errorf("failed to open checksums file: %w", err)
	}
	defer checksumFile.Close()

	var expectedChecksum string
	scanner := bufio.NewScanner(checksumFile)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == binaryName {
			expectedChecksum = parts[0]
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read checksums file: %w", err)
	}

	if expectedChecksum == "" {
		return fmt.Errorf("checksum not found for %s in SHA256SUMS", binaryName)
	}

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch:\n  expected: %s\n  actual:   %s", expectedChecksum, actualChecksum)
	}

	return nil
}
