// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/download"
	"github.com/Work-Fort/Anvil/pkg/util"
)

// BuildPhase represents a phase in the kernel build process
type BuildPhase int

const (
	PhaseDownload BuildPhase = iota
	PhaseVerify
	PhaseExtract
	PhaseConfigure
	PhaseCompile
	PhasePackage
)

// BuildOptions contains options for building a kernel
type BuildOptions struct {
	Version           string
	Arch              string
	VerificationLevel string
	ConfigFile        string
	Interactive       bool             // Whether to show TUI
	Writer            io.Writer        // Optional: custom writer for build output (for TUI streaming)
	ProgressCallback  func(float64)    // Optional: callback for download progress (0.0 to 1.0)
	PhaseCallback     func(BuildPhase) // Optional: callback for phase transitions
	StatsCallback     func(BuildStats) // Optional: callback for final build statistics
	Context           context.Context  // Optional: context for cancellation
}

// BuildStats contains statistics about a completed build
type BuildStats struct {
	TotalDuration     time.Duration
	DownloadDuration  time.Duration
	ExtractDuration   time.Duration
	ConfigureDuration time.Duration
	CompileDuration   time.Duration
	PackageDuration   time.Duration
	UncompressedSize  int64
	CompressedSize    int64
	UncompressedHash  string
	CompressedHash    string
	KernelVersion     string
	OutputPath        string
	CompressedPath    string
	BuildTimestamp    time.Time // Timestamp when build completed
}

// Kernel.org autosigner key (signs sha256sums.asc)
const (
	autosignerKeyID          = "632D3A06589DA6B1"
	autosignerKeyFingerprint = "B8868C80BA62A1FFFAF5FDA9632D3A06589DA6B1"
)

// kernelOrgRelease represents a kernel.org API release response
type kernelOrgRelease struct {
	LatestStable struct {
		Version string `json:"version"`
	} `json:"latest_stable"`
}

// buildLogger wraps a writer to emit structured log messages for TUI
type buildLogger struct {
	writer io.Writer
}

func (bl *buildLogger) Info(msg string) {
	bl.writer.Write([]byte(fmt.Sprintf("[INFO] %s\n", msg)))
}

func (bl *buildLogger) Warn(msg string) {
	bl.writer.Write([]byte(fmt.Sprintf("[WARN] %s\n", msg)))
}

func (bl *buildLogger) Error(msg string) {
	bl.writer.Write([]byte(fmt.Sprintf("[ERROR] %s\n", msg)))
}

func (bl *buildLogger) Debug(msg string) {
	bl.writer.Write([]byte(fmt.Sprintf("[DEBUG] %s\n", msg)))
}

// Build builds a kernel from source
func Build(opts BuildOptions) error {
	// Default to host architecture if not specified
	if opts.Arch == "" {
		opts.Arch = runtime.GOARCH
		// Convert Go arch names to kernel arch names
		if opts.Arch == "amd64" {
			opts.Arch = "x86_64"
		} else if opts.Arch == "arm64" {
			opts.Arch = "aarch64"
		}
	}

	// Default verification level to high
	if opts.VerificationLevel == "" {
		opts.VerificationLevel = "high"
	}

	// Validate architecture
	if opts.Arch != "x86_64" && opts.Arch != "aarch64" && opts.Arch != "all" {
		return fmt.Errorf("unsupported architecture: %s (supported: x86_64, aarch64, all)", opts.Arch)
	}

	// Validate verification level
	if opts.VerificationLevel != "high" && opts.VerificationLevel != "medium" && opts.VerificationLevel != "disabled" {
		return fmt.Errorf("invalid verification level: %s (must be: high, medium, disabled)", opts.VerificationLevel)
	}

	// Determine output writer (custom writer for TUI, or stdout for CLI)
	writer := opts.Writer
	if writer == nil {
		writer = os.Stdout
	}

	// Use context or background
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Handle "all" architecture - build for both x86_64 and aarch64
	if opts.Arch == "all" {
		architectures := []string{"x86_64", "aarch64"}
		for _, arch := range architectures {
			archOpts := opts
			archOpts.Arch = arch

			logger := &buildLogger{writer: writer}
			if err := runBuild(archOpts, logger, opts.ProgressCallback, opts.PhaseCallback, ctx); err != nil {
				return fmt.Errorf("failed to build for %s: %w", arch, err)
			}
		}
		return nil
	}

	// Single architecture build
	logger := &buildLogger{writer: writer}
	if err := runBuild(opts, logger, opts.ProgressCallback, opts.PhaseCallback, ctx); err != nil {
		return err
	}

	return nil
}

// runBuild executes the actual build process
func runBuild(opts BuildOptions, logger *buildLogger, progressCallback func(float64), phaseCallback func(BuildPhase), ctx context.Context) error {
	// Track build timing
	buildStartTime := time.Now()
	var downloadStart, extractStart, configureStart, compileStart, packageStart time.Time
	var downloadDuration, extractDuration, configureDuration, compileDuration, packageDuration time.Duration

	// Check context at start
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
	buildDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "build")
	artifactsDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "artifacts")

	// Create directories
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("failed to create build directory: %w", err)
	}
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return fmt.Errorf("failed to create artifacts directory: %w", err)
	}

	// Determine kernel version
	version := opts.Version
	if version == "" {
		logger.Info("Fetching latest stable kernel version from kernel.org...")
		var err error
		version, err = getLatestKernelVersion()
		if err != nil {
			return fmt.Errorf("failed to fetch latest kernel version: %w", err)
		}
		logger.Info(fmt.Sprintf("Latest stable kernel version: %s", version))
	} else {
		logger.Info(fmt.Sprintf("Using provided kernel version: %s", version))
	}

	// Determine output paths
	var kernelFilename, kernelImage string
	if opts.Arch == "x86_64" {
		kernelFilename = fmt.Sprintf("vmlinux-%s-%s", version, opts.Arch)
		kernelImage = "vmlinux"
	} else {
		kernelFilename = fmt.Sprintf("Image-%s-%s", version, opts.Arch)
		kernelImage = "arch/arm64/boot/Image"
	}
	kernelPath := filepath.Join(artifactsDir, kernelFilename)

	// Check if kernel already exists
	if _, err := os.Stat(kernelPath); err == nil {
		logger.Info(fmt.Sprintf("Kernel already exists: %s", kernelPath))

		// Load build stats from cached build and send to callback
		statsFile := filepath.Join(artifactsDir, "build-stats.json")
		if stats, err := ReadBuildStats(statsFile); err == nil {
			if opts.StatsCallback != nil {
				opts.StatsCallback(stats)
			}
		} else {
			logger.Warn(fmt.Sprintf("Failed to load cached build stats: %v", err))
		}

		return nil
	}
	if _, err := os.Stat(kernelPath + ".xz"); err == nil {
		logger.Info(fmt.Sprintf("Compressed kernel already exists: %s.xz", kernelPath))

		// Load build stats from cached build and send to callback
		statsFile := filepath.Join(artifactsDir, "build-stats.json")
		if stats, err := ReadBuildStats(statsFile); err == nil {
			if opts.StatsCallback != nil {
				opts.StatsCallback(stats)
			}
		} else {
			logger.Warn(fmt.Sprintf("Failed to load cached build stats: %v", err))
		}

		return nil
	}

	logger.Info(fmt.Sprintf("Building kernel from source for architecture: %s", opts.Arch))

	// Check for required build tools
	logger.Info("Checking for required build tools...")
	if err := checkBuildTools(opts.Arch); err != nil {
		return err
	}

	// Extract major version for download URL
	majorVersion := strings.Split(version, ".")[0]

	// Download and verify kernel source
	kernelURL := fmt.Sprintf("https://cdn.kernel.org/pub/linux/kernel/v%s.x/linux-%s.tar.xz", majorVersion, version)
	kernelTarball := filepath.Join(buildDir, fmt.Sprintf("linux-%s.tar.xz", version))
	kernelSrcDir := filepath.Join(buildDir, fmt.Sprintf("linux-%s", version))

	// Delete cached source when verification is enabled (security: always use fresh sources)
	if opts.VerificationLevel != "disabled" {
		if _, err := os.Stat(kernelTarball); err == nil {
			logger.Info("Deleting cached source (verification enabled - using fresh sources)")
			os.Remove(kernelTarball)
		}
		if _, err := os.Stat(kernelSrcDir); err == nil {
			os.RemoveAll(kernelSrcDir)
		}
	}

	// Download kernel source if not already present
	if _, err := os.Stat(kernelTarball); os.IsNotExist(err) {
		if phaseCallback != nil {
			phaseCallback(PhaseDownload)
		}
		downloadStart = time.Now()
		logger.Info(fmt.Sprintf("Downloading kernel source from %s...", kernelURL))
		if err := download.File(kernelURL, kernelTarball, progressCallback); err != nil {
			return fmt.Errorf("failed to download kernel source: %w", err)
		}
		downloadDuration = time.Since(downloadStart)
		logger.Info("Kernel source downloaded successfully")
	} else {
		logger.Info("Kernel source already downloaded")
	}

	// Verify kernel source
	if phaseCallback != nil {
		phaseCallback(PhaseVerify)
	}
	if err := verifyKernelSource(logger, opts.VerificationLevel, majorVersion, version, kernelTarball, buildDir); err != nil {
		return err
	}

	// Extract kernel source
	if _, err := os.Stat(kernelSrcDir); os.IsNotExist(err) {
		if phaseCallback != nil {
			phaseCallback(PhaseExtract)
		}
		extractStart = time.Now()
		logger.Info("Extracting kernel source...")
		if err := util.ExtractTarXzWithProgress(kernelTarball, buildDir, progressCallback); err != nil {
			return fmt.Errorf("failed to extract kernel source: %w", err)
		}
		extractDuration = time.Since(extractStart)
		logger.Info("Kernel source extracted successfully")
	} else {
		logger.Info("Kernel source already extracted, skipping...")
	}

	// Apply kernel configuration
	if phaseCallback != nil {
		phaseCallback(PhaseConfigure)
	}
	configureStart = time.Now()
	if err := applyKernelConfig(logger, opts, kernelSrcDir, ctx); err != nil {
		return err
	}
	configureDuration = time.Since(configureStart)

	// Build the kernel
	if phaseCallback != nil {
		phaseCallback(PhaseCompile)
	}
	compileStart = time.Now()
	if err := buildKernelImage(logger, opts, kernelSrcDir, kernelImage, ctx); err != nil {
		return err
	}
	compileDuration = time.Since(compileStart)

	// Package artifacts
	if phaseCallback != nil {
		phaseCallback(PhasePackage)
	}
	packageStart = time.Now()
	if err := packageArtifacts(logger, opts, version, kernelSrcDir, kernelImage, artifactsDir, kernelFilename, ctx); err != nil {
		return err
	}
	packageDuration = time.Since(packageStart)

	logger.Info("Build completed successfully!")

	// Collect build stats
	stats := collectBuildStats(
		version,
		kernelPath,
		time.Since(buildStartTime),
		downloadDuration,
		extractDuration,
		configureDuration,
		compileDuration,
		packageDuration,
	)

	// Write build stats to JSON file in artifacts directory
	statsFile := filepath.Join(artifactsDir, "build-stats.json")
	if err := writeBuildStats(statsFile, stats); err != nil {
		logger.Warn(fmt.Sprintf("Failed to write build stats: %v", err))
	}

	// Call stats callback if provided
	if opts.StatsCallback != nil {
		opts.StatsCallback(stats)
	}

	return nil
}

// writeBuildStats writes build statistics to a JSON file
func writeBuildStats(path string, stats BuildStats) error {
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal build stats: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write build stats file: %w", err)
	}

	return nil
}

// ReadBuildStats reads build statistics from a JSON file
func ReadBuildStats(path string) (BuildStats, error) {
	var stats BuildStats

	data, err := os.ReadFile(path)
	if err != nil {
		return stats, fmt.Errorf("failed to read build stats file: %w", err)
	}

	if err := json.Unmarshal(data, &stats); err != nil {
		return stats, fmt.Errorf("failed to unmarshal build stats: %w", err)
	}

	return stats, nil
}

// CheckKernelInstalled checks if a kernel with the given build stats is installed
// Returns (isInstalled, timestampedVersion, error)
func CheckKernelInstalled(stats BuildStats) (bool, string, error) {
	// Generate the timestamped version name
	timestamp := stats.BuildTimestamp.Format("20060102T150405")
	versionWithTimestamp := fmt.Sprintf("%s-%s", stats.KernelVersion, timestamp)

	// Check if the directory exists in kernels dir
	kernelDir := filepath.Join(config.GlobalPaths.KernelsDir, versionWithTimestamp)
	if _, err := os.Stat(kernelDir); os.IsNotExist(err) {
		return false, "", nil
	} else if err != nil {
		return false, "", fmt.Errorf("failed to check kernel directory: %w", err)
	}

	// Directory exists - kernel is installed
	return true, versionWithTimestamp, nil
}

// CheckCachedBuild checks if a completed build exists for the given version
func CheckCachedBuild(version string) (bool, string, error) {
	artifactsDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "artifacts")
	statsFile := filepath.Join(artifactsDir, "build-stats.json")

	// Check if build-stats.json exists
	if _, err := os.Stat(statsFile); os.IsNotExist(err) {
		return false, "", nil
	}

	// Try to read the stats to verify it's valid
	stats, err := ReadBuildStats(statsFile)
	if err != nil {
		return false, "", err
	}

	// Check if the version matches (empty version means "any cached build")
	if version != "" && stats.KernelVersion != version {
		log.Debugf("Cached build version mismatch: cached=%s, requested=%s", stats.KernelVersion, version)
		return false, "", nil
	}

	// Check if output files exist
	if _, err := os.Stat(stats.OutputPath); os.IsNotExist(err) {
		log.Debugf("Cached build output missing: %s", stats.OutputPath)
		return false, "", nil
	}

	if _, err := os.Stat(stats.CompressedPath); os.IsNotExist(err) {
		log.Debugf("Cached build compressed output missing: %s", stats.CompressedPath)
		return false, "", nil
	}

	return true, statsFile, nil
}

// collectBuildStats collects statistics about the completed build
func collectBuildStats(version, kernelPath string, totalDuration, downloadDuration, extractDuration, configureDuration, compileDuration, packageDuration time.Duration) BuildStats {
	stats := BuildStats{
		KernelVersion:     version,
		OutputPath:        kernelPath,
		CompressedPath:    kernelPath + ".xz",
		TotalDuration:     totalDuration,
		DownloadDuration:  downloadDuration,
		ExtractDuration:   extractDuration,
		ConfigureDuration: configureDuration,
		CompileDuration:   compileDuration,
		PackageDuration:   packageDuration,
		BuildTimestamp:    time.Now(), // Record when build completed
	}

	// Get uncompressed kernel size and hash
	if info, err := os.Stat(kernelPath); err == nil {
		stats.UncompressedSize = info.Size()
	}
	if hash, err := util.CalculateSHA256(kernelPath); err == nil {
		stats.UncompressedHash = hash
	}

	// Get compressed kernel size and hash
	if info, err := os.Stat(kernelPath + ".xz"); err == nil {
		stats.CompressedSize = info.Size()
	}
	if hash, err := util.CalculateSHA256(kernelPath + ".xz"); err == nil {
		stats.CompressedHash = hash
	}

	return stats
}

// InstallBuiltKernel installs a built kernel to the kernels directory with a timestamped name
func InstallBuiltKernel(stats BuildStats, setAsDefault bool) (string, error) {
	arch, err := config.GetArch()
	if err != nil {
		return "", err
	}

	kernelName, err := config.GetKernelName()
	if err != nil {
		return "", err
	}

	// Create timestamped version name: version-YYYYMMDDTHHmmss
	// Use the build timestamp from stats to ensure consistency
	timestamp := stats.BuildTimestamp.Format("20060102T150405")
	versionWithTimestamp := fmt.Sprintf("%s-%s", stats.KernelVersion, timestamp)

	// Create destination directory
	destDir := filepath.Join(config.GlobalPaths.KernelsDir, versionWithTimestamp)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create kernel directory: %w", err)
	}

	// Destination file names with timestamp
	destKernel := filepath.Join(destDir, fmt.Sprintf("%s-%s-%s", kernelName, versionWithTimestamp, arch))
	destKernelXz := destKernel + ".xz"

	// Copy uncompressed kernel
	if err := copyFile(stats.OutputPath, destKernel); err != nil {
		return "", fmt.Errorf("failed to copy kernel: %w", err)
	}

	// Copy compressed kernel
	if err := copyFile(stats.CompressedPath, destKernelXz); err != nil {
		return "", fmt.Errorf("failed to copy compressed kernel: %w", err)
	}

	// Copy checksums if they exist
	if _, err := os.Stat(stats.OutputPath + ".sha256"); err == nil {
		destChecksum := destKernel + ".sha256"
		if err := copyFile(stats.OutputPath+".sha256", destChecksum); err != nil {
			return "", fmt.Errorf("failed to copy checksum: %w", err)
		}
	}

	if _, err := os.Stat(stats.CompressedPath + ".sha256"); err == nil {
		destChecksumXz := destKernelXz + ".sha256"
		if err := copyFile(stats.CompressedPath+".sha256", destChecksumXz); err != nil {
			return "", fmt.Errorf("failed to copy compressed checksum: %w", err)
		}
	}

	// Set as default if requested
	if setAsDefault {
		symlinkPath := filepath.Join(config.GlobalPaths.DataDir, kernelName)

		// Remove existing symlink
		os.Remove(symlinkPath)

		// Create new symlink
		if err := os.Symlink(destKernel, symlinkPath); err != nil {
			return "", fmt.Errorf("failed to create symlink: %w", err)
		}
	}

	return versionWithTimestamp, nil
}

// ArchiveInstalledKernel copies built kernel artifacts to the repo-local archive directory
// under arch/version subdirectories and maintains an index.json mapping.
//
// Structure:
//
//	archive/
//	├── x86_64/
//	│   └── {version}/
//	│       ├── vmlinux-{version}-x86_64
//	│       ├── vmlinux-{version}-x86_64.xz
//	│       ├── vmlinux-{version}-x86_64.sha256
//	│       ├── vmlinux-{version}-x86_64.xz.sha256
//	│       └── signing-key.asc
//	└── index.json  {"x86_64": {"6.18.9": "x86_64/6.18.9/vmlinux-6.18.9-x86_64.xz"}}
func ArchiveInstalledKernel(stats BuildStats, archiveDir string) error {
	// Derive arch from compressed filename: vmlinux-6.18.9-x86_64.xz → x86_64
	base := strings.TrimSuffix(filepath.Base(stats.CompressedPath), ".xz")
	parts := strings.Split(base, "-")
	arch := parts[len(parts)-1]

	// Create arch/version subdirectory
	versionDir := filepath.Join(archiveDir, arch, stats.KernelVersion)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Copy artifacts into the arch/version directory
	type srcDst struct{ src, dst string }
	copies := []srcDst{
		{stats.OutputPath, filepath.Join(versionDir, filepath.Base(stats.OutputPath))},
		{stats.CompressedPath, filepath.Join(versionDir, filepath.Base(stats.CompressedPath))},
	}
	for _, extra := range []string{stats.OutputPath + ".sha256", stats.CompressedPath + ".sha256"} {
		if _, err := os.Stat(extra); err == nil {
			copies = append(copies, srcDst{extra, filepath.Join(versionDir, filepath.Base(extra))})
		}
	}
	for _, c := range copies {
		if err := copyFile(c.src, c.dst); err != nil {
			return fmt.Errorf("failed to archive %s: %w", filepath.Base(c.src), err)
		}
	}

	// Generate SHA256SUMS by concatenating all individual .sha256 files.
	// SignArtifacts expects this file when signing the directory.
	if err := generateSHA256SUMS(versionDir); err != nil {
		return fmt.Errorf("failed to generate SHA256SUMS: %w", err)
	}

	// Update archive/index.json: path is relative to archiveDir
	kernelPath := filepath.Join(arch, stats.KernelVersion, filepath.Base(stats.CompressedPath))
	return updateArchiveIndex(archiveDir, arch, stats.KernelVersion, kernelPath)
}

// generateSHA256SUMS concatenates all *.sha256 files in dir into a single
// SHA256SUMS file in the standard sha256sum format, as expected by SignArtifacts.
func generateSHA256SUMS(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var combined []byte
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sha256") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		combined = append(combined, data...)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			combined = append(combined, '\n')
		}
	}

	return os.WriteFile(filepath.Join(dir, "SHA256SUMS"), combined, 0644)
}

// updateArchiveIndex reads (or initialises) archive/index.json and records
// the compressed kernel path for the given arch and version.
func updateArchiveIndex(archiveDir, arch, version, kernelPath string) error {
	indexPath := filepath.Join(archiveDir, "index.json")

	index := map[string]map[string]string{
		"x86_64":  {},
		"aarch64": {},
	}

	if data, err := os.ReadFile(indexPath); err == nil {
		// Ignore unmarshal errors — corrupt index is replaced cleanly
		_ = json.Unmarshal(data, &index)
	}

	if index[arch] == nil {
		index[arch] = map[string]string{}
	}
	index[arch][version] = kernelPath

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal archive index: %w", err)
	}
	return os.WriteFile(indexPath, data, 0644)
}

// getLatestKernelVersion fetches the latest stable kernel version from kernel.org
func getLatestKernelVersion() (string, error) {
	resp, err := http.Get("https://www.kernel.org/releases.json")
	if err != nil {
		return "", fmt.Errorf("failed to fetch kernel.org API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("kernel.org API returned status: %s", resp.Status)
	}

	var release kernelOrgRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse kernel.org API response: %w", err)
	}

	if release.LatestStable.Version == "" {
		return "", fmt.Errorf("no latest stable version found")
	}

	return release.LatestStable.Version, nil
}

// ValidateVersion checks if a kernel version exists in kernel.org releases
func ValidateVersion(version string) error {
	// Fetch releases from kernel.org
	resp, err := http.Get("https://www.kernel.org/releases.json")
	if err != nil {
		// If we can't reach kernel.org, allow the version (might be offline build)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// If API fails, allow the version
		return nil
	}

	// Parse the full releases structure to get all available versions
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		// If we can't parse, allow the version
		return nil
	}

	// Extract releases array
	releases, ok := data["releases"].([]interface{})
	if !ok {
		// If structure is unexpected, allow the version
		return nil
	}

	// Check if version exists in releases
	var availableVersions []string
	for _, release := range releases {
		releaseMap, ok := release.(map[string]interface{})
		if !ok {
			continue
		}
		if ver, ok := releaseMap["version"].(string); ok {
			availableVersions = append(availableVersions, ver)
			if ver == version {
				// Version found
				return nil
			}
		}
	}

	// Version not found - provide helpful error with available versions
	if len(availableVersions) > 10 {
		availableVersions = availableVersions[:10]
	}
	return fmt.Errorf("kernel version %s not found in kernel.org releases\n\nAvailable versions (latest 10):\n  %s\n\nUse --verification-level disabled if building a very new kernel",
		version,
		strings.Join(availableVersions, "\n  "))
}

// checkBuildTools verifies that required build tools are installed
func checkBuildTools(arch string) error {
	// Check make
	if _, err := exec.LookPath("make"); err != nil {
		return fmt.Errorf("make not found. Please install build-essential")
	}

	// Check gcc
	if _, err := exec.LookPath("gcc"); err != nil {
		return fmt.Errorf("gcc not found. Please install build-essential")
	}

	// Check cross-compiler for ARM64
	if arch == "aarch64" {
		if _, err := exec.LookPath("aarch64-linux-gnu-gcc"); err != nil {
			return fmt.Errorf("aarch64-linux-gnu-gcc not found. Install with: sudo apt-get install gcc-aarch64-linux-gnu")
		}
	}

	return nil
}

// verifyKernelSource verifies the downloaded kernel source based on verification level
func verifyKernelSource(logger *buildLogger, verificationLevel, majorVersion, version, kernelTarball, buildDir string) error {
	if verificationLevel == "disabled" {
		logger.Warn("Verification disabled - proceeding without any security checks")
		logger.Warn("  The kernel source tarball has NOT been verified")
		logger.Warn("  Only use this for testing or when kernel is too new for checksums")
		return nil
	}

	// Download checksums file
	logger.Info("Downloading checksums file for verification...")
	checksumsURL := fmt.Sprintf("https://cdn.kernel.org/pub/linux/kernel/v%s.x/sha256sums.asc", majorVersion)
	checksumsFile := filepath.Join(buildDir, "sha256sums.asc")

	if err := download.File(checksumsURL, checksumsFile, nil); err != nil {
		return fmt.Errorf("could not download checksums file from kernel.org: %w\nUse --verification-level disabled to proceed anyway (not recommended)", err)
	}
	defer os.Remove(checksumsFile)

	// PGP verification (only for 'high' level)
	if verificationLevel == "high" {
		logger.Info("Verifying PGP signature on checksums file...")

		// Import autosigner key
		if err := importAutosignerKey(logger); err != nil {
			logger.Warn("Could not import autosigner key, skipping PGP verification")
		} else {
			// Verify the signature
			cmd := exec.Command("gpg", "--verify", checksumsFile)
			output, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "Good signature") {
				return fmt.Errorf("PGP signature verification failed\nThe checksums file may have been tampered with\n%s", string(output))
			}
			logger.Info("✓ PGP signature verification passed")
			logger.Info("  Signed by: Kernel.org checksum autosigner <autosigner@kernel.org>")
		}
	} else if verificationLevel == "medium" {
		logger.Info("Skipping PGP verification (verification-level: medium)")
		logger.Info("  Trusting HTTPS connection to kernel.org for checksums file")
	}

	// SHA256 checksum verification (for both 'high' and 'medium' levels)
	logger.Info("Verifying kernel source checksum...")

	// Read checksums file
	content, err := os.ReadFile(checksumsFile)
	if err != nil {
		return fmt.Errorf("failed to read checksums file: %w", err)
	}

	// Extract the checksum for our specific kernel version
	tarballName := fmt.Sprintf("linux-%s.tar.xz", version)
	var expectedHash string
	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, tarballName) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				expectedHash = parts[0]
				break
			}
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("checksum not found in sha256sums.asc for %s\nKernel may be too new and not yet in checksums file.\nUse --verification-level disabled to proceed anyway (not recommended)", tarballName)
	}

	// Verify the tarball checksum
	actualHash, err := util.CalculateSHA256(kernelTarball)
	if err != nil {
		return fmt.Errorf("failed to calculate tarball checksum: %w", err)
	}

	if !strings.EqualFold(expectedHash, actualHash) {
		return fmt.Errorf("checksum verification FAILED!\nExpected: %s\nActual:   %s\nThe tarball may be corrupted or tampered with.\nRemove %s and try again", expectedHash, actualHash, kernelTarball)
	}

	logger.Info("✓ Checksum verification passed")
	logger.Info(fmt.Sprintf("  Expected: %s", expectedHash))
	logger.Info(fmt.Sprintf("  Actual:   %s", actualHash))

	return nil
}

// importAutosignerKey imports the kernel.org autosigner GPG key
func importAutosignerKey(logger *buildLogger) error {
	// Check if gpg is available
	if _, err := exec.LookPath("gpg"); err != nil {
		return fmt.Errorf("gpg not found")
	}

	// Check if key is already imported
	cmd := exec.Command("gpg", "--list-keys", autosignerKeyID)
	if err := cmd.Run(); err == nil {
		// Key already imported
		return nil
	}

	logger.Info("Importing kernel.org autosigner GPG key...")
	logger.Info(fmt.Sprintf("  Key ID: %s", autosignerKeyID))
	logger.Info(fmt.Sprintf("  Fingerprint: %s", autosignerKeyFingerprint))

	// Try multiple keyservers
	keyservers := []string{
		"hkps://keyserver.ubuntu.com",
		"hkps://keys.openpgp.org",
		"hkps://pgp.mit.edu",
	}

	for _, keyserver := range keyservers {
		logger.Info(fmt.Sprintf("  Trying keyserver: %s", keyserver))
		cmd := exec.Command("gpg", "--keyserver", keyserver, "--recv-keys", autosignerKeyID)
		if err := cmd.Run(); err == nil {
			logger.Info("✓ Autosigner key imported successfully")

			// Verify the fingerprint matches
			cmd = exec.Command("gpg", "--fingerprint", autosignerKeyID)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to verify fingerprint: %w", err)
			}

			// Simple check: output should contain the fingerprint
			if !strings.Contains(string(output), autosignerKeyFingerprint) {
				return fmt.Errorf("fingerprint mismatch - possible key substitution attack")
			}

			return nil
		}
	}

	return fmt.Errorf("failed to import autosigner key from any keyserver")
}

// applyKernelConfig applies the Firecracker kernel configuration
func applyKernelConfig(logger *buildLogger, opts BuildOptions, kernelSrcDir string, ctx context.Context) error {
	logger.Info(fmt.Sprintf("Applying Firecracker kernel configuration for %s...", opts.Arch))

	// Check context
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Determine config file
	configFile := opts.ConfigFile
	if configFile == "" {
		// Check if we're in repo mode (anvil.yaml exists)
		repoConfigPath := filepath.Join(".", config.LocalConfigFile+config.DefaultConfigExt)
		if _, err := os.Stat(repoConfigPath); err == nil {
			// Repo mode: get kernel config from repo config
			if opts.Arch == "x86_64" {
				configFile = config.GetKernelsConfigX86_64()
			} else if opts.Arch == "aarch64" {
				configFile = config.GetKernelsConfigAarch64()
			}

			if configFile == "" {
				return fmt.Errorf(
					"kernel config not found in repo config for %s\n\n"+
						"Add to anvil.yaml:\n"+
						"kernels:\n"+
						"  config:\n"+
						"    %s: path/to/kernel.config",
					opts.Arch,
					opts.Arch,
				)
			}
			logger.Info(fmt.Sprintf("Using kernel config from repo: %s", configFile))
		} else {
			// Not in repo mode: require --config flag
			return fmt.Errorf(
				"kernel config file required (not in repo mode)\n\n"+
					"Either:\n"+
					"  1. Use --config flag: anvil build-kernel --config path/to/kernel.config\n"+
					"  2. Create anvil.yaml in repo root with:\n"+
					"     kernels:\n"+
					"       config:\n"+
					"         x86_64: configs/kernel-x86_64.config\n"+
					"         aarch64: configs/kernel-aarch64.config",
			)
		}
	}

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	// Copy config file to kernel source
	destConfig := filepath.Join(kernelSrcDir, ".config")
	if err := copyFile(configFile, destConfig); err != nil {
		return fmt.Errorf("failed to copy kernel config: %w", err)
	}

	// Update config for new kernel version
	logger.Info("Running make olddefconfig to update config...")

	var cmd *exec.Cmd
	if opts.Arch == "aarch64" {
		cmd = exec.Command("make", "olddefconfig", "ARCH=arm64")
	} else {
		cmd = exec.Command("make", "olddefconfig")
	}
	cmd.Dir = kernelSrcDir
	// Route output through logger's writer (pipes to TUI properly)
	cmd.Stdout = logger.writer
	cmd.Stderr = logger.writer

	// Run with proper process group handling for cancellation
	if err := runCommandWithProcessGroup(ctx, cmd); err != nil {
		return fmt.Errorf("failed to update kernel config: %w", err)
	}

	return nil
}

// runCommandWithProcessGroup runs a command and ensures all child processes are killed on cancellation
func runCommandWithProcessGroup(ctx context.Context, cmd *exec.Cmd) error {
	// Create a new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return err
	}

	// Monitor context cancellation
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled - kill the entire process group
		if cmd.Process != nil {
			// Send SIGKILL to the entire process group (negative PID)
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-done // Wait for the process to actually terminate
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// buildKernelImage builds the kernel image
func buildKernelImage(logger *buildLogger, opts BuildOptions, kernelSrcDir, kernelImage string, ctx context.Context) error {
	logger.Info("Building kernel (this may take a while)...")

	// Check context
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// ARM64 kernels >= 6.11 need make prepare to generate syscall headers (unistd_64.h)
	if opts.Arch == "aarch64" {
		prepCmd := exec.Command("make", "prepare", "ARCH=arm64")
		prepCmd.Dir = kernelSrcDir
		prepCmd.Stdout = logger.writer
		prepCmd.Stderr = logger.writer
		if err := runCommandWithProcessGroup(ctx, prepCmd); err != nil {
			if ctx != nil && ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("kernel prepare failed: %w", err)
		}
	}

	var cmd *exec.Cmd
	numCPU := runtime.NumCPU()

	if opts.Arch == "x86_64" {
		cmd = exec.Command("make", fmt.Sprintf("-j%d", numCPU), "vmlinux")
	} else {
		cmd = exec.Command("make", fmt.Sprintf("-j%d", numCPU), "Image", "ARCH=arm64")
	}
	cmd.Dir = kernelSrcDir
	// Route output through logger's writer (pipes to TUI properly)
	cmd.Stdout = logger.writer
	cmd.Stderr = logger.writer

	// Run with proper process group handling for cancellation
	if err := runCommandWithProcessGroup(ctx, cmd); err != nil {
		// Check if error is due to context cancellation
		if ctx != nil && ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("kernel build failed: %w", err)
	}

	logger.Info("Kernel built successfully")
	return nil
}

// packageArtifacts packages the built kernel and generates checksums
func packageArtifacts(logger *buildLogger, opts BuildOptions, version, kernelSrcDir, kernelImage, artifactsDir, outputName string, ctx context.Context) error {
	logger.Info("Preparing release artifacts...")

	// Check context
	if ctx != nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	// Determine kernel binary path
	kernelBinary := filepath.Join(kernelSrcDir, kernelImage)

	// Copy kernel binary to artifacts directory
	outputPath := filepath.Join(artifactsDir, outputName)
	if err := copyFile(kernelBinary, outputPath); err != nil {
		return fmt.Errorf("failed to copy kernel binary: %w", err)
	}

	// Generate SHA256 checksum of decompressed kernel
	logger.Info("Generating SHA256 checksum of decompressed kernel...")
	hash, err := util.CalculateSHA256(outputPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksumFile := outputPath + ".sha256"
	if err := os.WriteFile(checksumFile, []byte(fmt.Sprintf("%s  %s\n", hash, outputName)), 0644); err != nil {
		return fmt.Errorf("failed to write checksum file: %w", err)
	}

	// Compress kernel with xz (keep decompressed copy for signing)
	logger.Info("Compressing kernel with xz (this may take a while)...")
	compressedPath := outputPath + ".xz"
	if err := util.CompressXZ(outputPath, compressedPath); err != nil {
		return fmt.Errorf("failed to compress kernel: %w", err)
	}
	logger.Info("Kernel compressed successfully")

	// Generate SHA256 checksum of compressed kernel
	logger.Info("Generating SHA256 checksum of compressed kernel...")
	hashCompressed, err := util.CalculateSHA256(compressedPath)
	if err != nil {
		return fmt.Errorf("failed to calculate compressed checksum: %w", err)
	}
	checksumFileCompressed := compressedPath + ".sha256"
	if err := os.WriteFile(checksumFileCompressed, []byte(fmt.Sprintf("%s  %s.xz\n", hashCompressed, outputName)), 0644); err != nil {
		return fmt.Errorf("failed to write compressed checksum file: %w", err)
	}

	// Copy kernel config
	configSrc := filepath.Join(kernelSrcDir, ".config")
	configDst := filepath.Join(artifactsDir, fmt.Sprintf("config-%s-%s", version, opts.Arch))
	if err := copyFile(configSrc, configDst); err != nil {
		return fmt.Errorf("failed to copy kernel config: %w", err)
	}

	// List artifacts
	logger.Info("Artifacts created:")
	entries, err := os.ReadDir(artifactsDir)
	if err == nil {
		for _, entry := range entries {
			if strings.Contains(entry.Name(), version) && strings.Contains(entry.Name(), opts.Arch) {
				info, _ := entry.Info()
				logger.Info(fmt.Sprintf("  %s (%d bytes)", entry.Name(), info.Size()))
			}
		}
	}

	return nil
}
