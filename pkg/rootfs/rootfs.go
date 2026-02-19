// SPDX-License-Identifier: Apache-2.0
package rootfs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Work-Fort/Anvil/pkg/config"
	"libguestfs.org/guestfs"
)

// CreatePhase represents a phase in the rootfs creation process
type CreatePhase int

const (
	PhaseDownload CreatePhase = iota
	PhaseCreate
	PhaseFormat
	PhasePopulate
	PhaseInjectBinary
	PhaseComplete
)

// CreateOptions contains options for creating a rootfs
type CreateOptions struct {
	OutputPath     string
	SizeMB         int
	AlpineVersion  string            // e.g., "3.23"
	AlpinePatch    string            // e.g., "3"
	Interactive    bool              // Whether to show TUI
	Writer         io.Writer         // Optional: custom writer for output (for TUI streaming)
	PhaseCallback  func(CreatePhase) // Optional: callback for phase transitions
	StatsCallback  func(CreateStats) // Optional: callback for final statistics
	Context        context.Context   // Optional: context for cancellation
	ForceOverwrite bool              // Overwrite existing file
	InjectBinary   bool              // Whether to inject binary into rootfs
	BinaryPath     string            // Path to binary to inject (default: current executable)
	BinaryDestPath string            // Destination path in rootfs (default: /usr/bin/anvil)
}

// CreateStats contains statistics about a completed rootfs creation
type CreateStats struct {
	TotalDuration  time.Duration
	OutputPath     string
	SizeMB         int
	CreateTime     time.Time
	AlpineVersion  string
	BinaryInjected bool
}

// rootfsLogger wraps a writer to emit structured log messages for TUI
type rootfsLogger struct {
	writer io.Writer
}

func (rl *rootfsLogger) Info(msg string) {
	rl.writer.Write([]byte(fmt.Sprintf("[INFO] %s\n", msg)))
}

func (rl *rootfsLogger) Warn(msg string) {
	rl.writer.Write([]byte(fmt.Sprintf("[WARN] %s\n", msg)))
}

func (rl *rootfsLogger) Error(msg string) {
	rl.writer.Write([]byte(fmt.Sprintf("[ERROR] %s\n", msg)))
}

func (rl *rootfsLogger) Debug(msg string) {
	rl.writer.Write([]byte(fmt.Sprintf("[DEBUG] %s\n", msg)))
}

// initScriptTemplate is the template for the init script
// It will be formatted with the binary path
const initScriptTemplate = `#!/bin/sh
# Init script for Firecracker VM

# Mount essential filesystems
mount -t proc none /proc
mount -t sysfs none /sys
mount -t devtmpfs none /dev

# Setup networking (loopback)
ip link set lo up

# Print boot info
echo "=========================================="
echo "Cracker Barrel Firecracker VM"
echo "Kernel version: $(uname -r)"
echo "Architecture: $(uname -m)"
echo "=========================================="

# Start vsock server if binary exists
if [ -x %s ]; then
    echo "Starting vsock server..."
    %s &
    AGENT_PID=$!
    echo "Server started with PID ${AGENT_PID}"
else
    echo "WARNING: Vsock server binary not found at %s"
fi

# Keep VM running (block forever)
echo "VM ready - vsock server running on port 8000"
while true; do
    sleep 1000
done
`

// Create creates an Alpine-based rootfs for Firecracker with optional anvil binary injection
func Create(opts CreateOptions) error {
	startTime := time.Now()

	// Set defaults
	if opts.OutputPath == "" {
		opts.OutputPath = filepath.Join(config.GlobalPaths.DataDir, "alpine-rootfs.ext4")
	}
	if opts.SizeMB == 0 {
		opts.SizeMB = 512 // 512MB like frontier
	}
	if opts.AlpineVersion == "" {
		opts.AlpineVersion = "3.23"
	}
	if opts.AlpinePatch == "" {
		opts.AlpinePatch = "3"
	}
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.InjectBinary {
		if opts.BinaryPath == "" {
			// Default to current executable
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get current executable path: %w", err)
			}
			opts.BinaryPath = exe
		}
		if opts.BinaryDestPath == "" {
			opts.BinaryDestPath = "/usr/bin/vsock-server"
		}
	}

	logger := &rootfsLogger{writer: opts.Writer}

	// Check if output file already exists
	if !opts.ForceOverwrite {
		if _, err := os.Stat(opts.OutputPath); err == nil {
			return fmt.Errorf("rootfs already exists: %s (use --force to overwrite)", opts.OutputPath)
		}
	}

	// Create output directory
	outputDir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Phase 1: Download Alpine tarball
	if opts.PhaseCallback != nil {
		opts.PhaseCallback(PhaseDownload)
	}

	alpineURL := fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/v%s/releases/x86_64/alpine-minirootfs-%s.%s-x86_64.tar.gz",
		opts.AlpineVersion, opts.AlpineVersion, opts.AlpinePatch)

	logger.Info(fmt.Sprintf("Downloading Alpine Linux %s.%s...", opts.AlpineVersion, opts.AlpinePatch))
	alpineTarball := filepath.Join(os.TempDir(), "alpine-minirootfs.tar.gz")
	defer os.Remove(alpineTarball)

	if err := downloadFile(alpineURL, alpineTarball); err != nil {
		return fmt.Errorf("failed to download Alpine tarball: %w", err)
	}

	// Phase 2: Create empty image
	if opts.PhaseCallback != nil {
		opts.PhaseCallback(PhaseCreate)
	}

	logger.Info(fmt.Sprintf("Creating %dMB empty image...", opts.SizeMB))
	if err := createEmptyImage(opts.OutputPath, opts.SizeMB); err != nil {
		return fmt.Errorf("failed to create empty image: %w", err)
	}

	// Phase 3: Format and populate with libguestfs
	if opts.PhaseCallback != nil {
		opts.PhaseCallback(PhaseFormat)
	}

	logger.Info("Formatting as ext4 and populating rootfs...")
	if err := formatAndPopulateRootfs(opts.OutputPath, alpineTarball, opts.BinaryDestPath, logger, opts.PhaseCallback); err != nil {
		return fmt.Errorf("failed to format and populate rootfs: %w", err)
	}

	// Phase 5: Inject binary if requested
	if opts.InjectBinary {
		if opts.PhaseCallback != nil {
			opts.PhaseCallback(PhaseInjectBinary)
		}

		logger.Info(fmt.Sprintf("Injecting vsock server binary to %s...", opts.BinaryDestPath))
		if err := injectBinaryWithLibguestfs(opts.OutputPath, opts.BinaryPath, opts.BinaryDestPath, logger); err != nil {
			return fmt.Errorf("failed to inject binary: %w", err)
		}
	}

	// Phase 6: Complete
	if opts.PhaseCallback != nil {
		opts.PhaseCallback(PhaseComplete)
	}

	// Call stats callback if provided
	if opts.StatsCallback != nil {
		opts.StatsCallback(CreateStats{
			TotalDuration:  time.Since(startTime),
			OutputPath:     opts.OutputPath,
			SizeMB:         opts.SizeMB,
			CreateTime:     time.Now(),
			AlpineVersion:  fmt.Sprintf("%s.%s", opts.AlpineVersion, opts.AlpinePatch),
			BinaryInjected: opts.InjectBinary,
		})
	}

	logger.Info(fmt.Sprintf("Alpine rootfs created successfully: %s", opts.OutputPath))
	return nil
}

// downloadFile downloads a file from a URL to a local path
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// createEmptyImage creates an empty file of the specified size in MB
func createEmptyImage(path string, sizeMB int) error {
	// Create the file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Truncate to the desired size (in bytes)
	sizeBytes := int64(sizeMB) * 1024 * 1024
	if err := f.Truncate(sizeBytes); err != nil {
		return fmt.Errorf("failed to truncate file: %w", err)
	}

	return nil
}

// formatAndPopulateRootfs formats the image as ext4 and populates it using libguestfs
func formatAndPopulateRootfs(imagePath, alpineTarball, binaryDestPath string, logger *rootfsLogger, phaseCallback func(CreatePhase)) error {
	// Create guestfs handle
	g, err := guestfs.Create()
	if err != nil {
		return fmt.Errorf("failed to create guestfs handle: %w", err)
	}
	defer g.Close()

	// Add the drive
	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := g.Add_drive(absPath, &guestfs.OptargsAdd_drive{
		Format_is_set:   true,
		Format:          "raw",
		Readonly_is_set: false,
	}); err != nil {
		return fmt.Errorf("failed to add drive: %w", err)
	}

	// Launch the appliance
	logger.Info("Launching libguestfs appliance...")
	if err := g.Launch(); err != nil {
		return fmt.Errorf("failed to launch guestfs: %w", err)
	}

	// Get devices
	devices, err := g.List_devices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("no devices found")
	}
	device := devices[0]

	// Format device as ext4
	logger.Info("Formatting device as ext4...")
	if err := g.Mkfs("ext4", device, nil); err != nil {
		return fmt.Errorf("failed to format device as ext4: %w", err)
	}

	// Trigger populate phase callback
	if phaseCallback != nil {
		phaseCallback(PhasePopulate)
	}

	// Mount the filesystem
	logger.Info("Mounting filesystem...")
	if err := g.Mount(device, "/"); err != nil {
		return fmt.Errorf("failed to mount device: %w", err)
	}

	// Extract Alpine tarball
	logger.Info("Extracting Alpine tarball...")
	if err := g.Tar_in(alpineTarball, "/", &guestfs.OptargsTar_in{
		Compress_is_set: true,
		Compress:        "gzip",
	}); err != nil {
		return fmt.Errorf("failed to extract tarball: %w", err)
	}

	// Copy required libraries for dynamically linked binaries
	logger.Info("Copying required glibc libraries...")

	// Create /lib64 directory for glibc compatibility
	if err := g.Mkdir_p("/lib64"); err != nil {
		return fmt.Errorf("failed to create /lib64: %w", err)
	}

	// Copy ld-linux-x86-64.so.2 (dynamic linker) from host
	if err := g.Upload("/lib64/ld-linux-x86-64.so.2", "/lib64/ld-linux-x86-64.so.2"); err != nil {
		logger.Warn("Failed to copy dynamic linker, binary may not work if dynamically linked")
	}

	// Create init script
	logger.Info("Creating init script...")
	// Generate init script with the configured binary path
	// Use empty string if no binary path configured (binary injection disabled)
	initScript := fmt.Sprintf(initScriptTemplate, binaryDestPath, binaryDestPath, binaryDestPath)
	if err := g.Write("/init", []byte(initScript)); err != nil {
		return fmt.Errorf("failed to write init script: %w", err)
	}

	// Make init executable (mode 0755)
	if err := g.Chmod(0755, "/init"); err != nil {
		return fmt.Errorf("failed to chmod init script: %w", err)
	}

	// Create inittab
	inittab := `::sysinit:/init
::respawn:/sbin/getty 38400 ttyS0
::ctrlaltdel:/sbin/reboot
::shutdown:/bin/umount -a -r
`
	if err := g.Write("/etc/inittab", []byte(inittab)); err != nil {
		return fmt.Errorf("failed to write inittab: %w", err)
	}

	// Unmount and shutdown
	logger.Info("Finalizing rootfs...")
	if err := g.Umount_all(); err != nil {
		return fmt.Errorf("failed to unmount: %w", err)
	}

	if err := g.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown: %w", err)
	}

	return nil
}

// injectBinaryWithLibguestfs injects a binary into the rootfs using libguestfs
func injectBinaryWithLibguestfs(imagePath, binaryPath, binaryDestPath string, logger *rootfsLogger) error {
	// Create guestfs handle
	g, err := guestfs.Create()
	if err != nil {
		return fmt.Errorf("failed to create guestfs handle: %w", err)
	}
	defer g.Close()

	// Add the drive
	absImagePath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute image path: %w", err)
	}

	absBinaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute binary path: %w", err)
	}

	if err := g.Add_drive(absImagePath, &guestfs.OptargsAdd_drive{
		Format_is_set:   true,
		Format:          "raw",
		Readonly_is_set: false,
	}); err != nil {
		return fmt.Errorf("failed to add drive: %w", err)
	}

	// Launch the appliance
	if err := g.Launch(); err != nil {
		return fmt.Errorf("failed to launch guestfs: %w", err)
	}

	// Get devices
	devices, err := g.List_devices()
	if err != nil {
		return fmt.Errorf("failed to list devices: %w", err)
	}
	if len(devices) == 0 {
		return fmt.Errorf("no devices found")
	}
	device := devices[0]

	// Mount the filesystem
	if err := g.Mount(device, "/"); err != nil {
		return fmt.Errorf("failed to mount device: %w", err)
	}

	// Create parent directory if it doesn't exist
	destDir := filepath.Dir(binaryDestPath)
	if err := g.Mkdir_p(destDir); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	// Upload the binary
	logger.Info(fmt.Sprintf("Uploading binary to %s...", binaryDestPath))
	if err := g.Upload(absBinaryPath, binaryDestPath); err != nil {
		return fmt.Errorf("failed to upload binary: %w", err)
	}

	// Make binary executable (mode 0755)
	if err := g.Chmod(0755, binaryDestPath); err != nil {
		return fmt.Errorf("failed to chmod binary: %w", err)
	}

	// Unmount and shutdown
	if err := g.Umount_all(); err != nil {
		return fmt.Errorf("failed to unmount: %w", err)
	}

	if err := g.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown: %w", err)
	}

	logger.Info("Binary injection complete!")
	return nil
}
