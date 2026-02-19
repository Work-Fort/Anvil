// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/firecracker/embedded"
	"github.com/Work-Fort/Anvil/pkg/rootfs"
	"github.com/Work-Fort/Anvil/pkg/vsock"
)

// TestOptions contains options for running a Firecracker test
type TestOptions struct {
	KernelVersion string
	RootfsPath    string
	Writer        io.Writer
	BootTimeout   time.Duration
	PingTimeout   time.Duration
}

// TestResult contains the results of a Firecracker test
type TestResult struct {
	Success       bool
	KernelVersion string
	RootfsPath    string
	BootTime      time.Duration
	PingRoundTrip time.Duration
	Error         error
}

// Test runs a full end-to-end test of Firecracker with vsock
func Test(opts TestOptions) (*TestResult, error) {
	startTime := time.Now()
	result := &TestResult{}

	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	if opts.BootTimeout == 0 {
		opts.BootTimeout = 10 * time.Second
	}
	if opts.PingTimeout == 0 {
		opts.PingTimeout = 10 * time.Second
	}

	logger := func(format string, args ...interface{}) {
		fmt.Fprintf(opts.Writer, format+"\n", args...)
	}

	// Step 1: Verify kernel exists
	logger("Checking kernel...")
	kernelPath, err := getKernelPath(opts.KernelVersion)
	if err != nil {
		result.Error = fmt.Errorf("kernel not found: %w", err)
		return result, result.Error
	}
	result.KernelVersion = opts.KernelVersion
	logger("  Found kernel: %s", kernelPath)

	// Step 2: Verify or create rootfs
	logger("Checking rootfs...")
	rootfsPath := opts.RootfsPath
	if rootfsPath == "" {
		rootfsPath = filepath.Join(config.GlobalPaths.DataDir, "alpine-rootfs.ext4")
	}

	if _, err := os.Stat(rootfsPath); os.IsNotExist(err) {
		logger("  Rootfs not found, creating with binary injection...")

		// Extract embedded vsock-server binary to temporary file
		logger("  Extracting embedded vsock-server binary...")
		vsockServerPath, cleanup, err := embedded.ExtractVsockServer()
		if err != nil {
			result.Error = fmt.Errorf("failed to extract vsock-server binary: %w", err)
			return result, result.Error
		}
		defer cleanup() // Clean up temp file after rootfs creation

		createOpts := rootfs.CreateOptions{
			OutputPath:     rootfsPath,
			SizeMB:         512,
			AlpineVersion:  "3.23",
			AlpinePatch:    "3",
			ForceOverwrite: false,
			InjectBinary:   true,
			BinaryPath:     vsockServerPath,
			BinaryDestPath: "/usr/bin/vsock-server",
			Writer:         opts.Writer,
		}
		if err := rootfs.Create(createOpts); err != nil {
			result.Error = fmt.Errorf("failed to create rootfs: %w", err)
			return result, result.Error
		}
	} else {
		logger("  Found rootfs: %s", rootfsPath)
	}
	result.RootfsPath = rootfsPath

	// Step 3: Find Firecracker binary
	logger("Checking Firecracker binary...")
	firecrackerPath, err := getFirecrackerBinary()
	if err != nil {
		result.Error = fmt.Errorf("firecracker binary not found: %w", err)
		return result, result.Error
	}
	logger("  Found Firecracker: %s", firecrackerPath)

	// Step 4: Create temporary directory for test
	logger("Setting up test environment...")
	testDir, err := os.MkdirTemp("", "anvil-test-*")
	if err != nil {
		result.Error = fmt.Errorf("failed to create temp dir: %w", err)
		return result, result.Error
	}
	defer func() {
		if result.Success {
			os.RemoveAll(testDir)
		} else {
			logger("\nTest artifacts preserved in: %s", testDir)
		}
	}()

	vsockPath := filepath.Join(testDir, "firecracker.vsock")
	apiSockPath := filepath.Join(testDir, "api.sock")

	// Step 5: Create Firecracker config
	logger("Creating Firecracker configuration...")
	configPath := filepath.Join(testDir, "config.json")
	if err := createTestConfig(configPath, kernelPath, rootfsPath, vsockPath); err != nil {
		result.Error = fmt.Errorf("failed to create config: %w", err)
		return result, result.Error
	}

	// Step 6: Start Firecracker VM
	logger("Starting Firecracker VM...")
	bootStart := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), opts.BootTimeout+opts.PingTimeout+5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, firecrackerPath,
		"--api-sock", apiSockPath,
		"--config-file", configPath,
	)

	// Capture output for debugging if needed
	logPath := filepath.Join(testDir, "firecracker.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to create log file: %w", err)
		return result, result.Error
	}
	defer logFile.Close()

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		result.Error = fmt.Errorf("failed to start Firecracker: %w", err)
		return result, result.Error
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	// Step 7: Wait for VM to boot and vsock server to be ready
	logger("Waiting for VM to boot...")

	// Probe for vsock readiness instead of blind sleep
	client := vsock.NewClient(vsockPath, 8000, nil)
	probeInterval := 100 * time.Millisecond
	maxProbeTime := opts.BootTimeout
	probeDeadline := time.Now().Add(maxProbeTime)

	var lastErr error
	for time.Now().Before(probeDeadline) {
		// Check if VM is still running
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			result.Error = fmt.Errorf("VM exited prematurely (see log: %s)", logPath)
			return result, result.Error
		}

		// Try to connect and ping
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		err := client.Ping(ctx, "probe")
		cancel()

		if err == nil {
			// Success! VM is ready
			break
		}
		lastErr = err
		time.Sleep(probeInterval)
	}

	// Check if we timed out
	if time.Now().After(probeDeadline) {
		result.Error = fmt.Errorf("VM boot timeout after %s (last error: %v, see log: %s)", maxProbeTime, lastErr, logPath)
		return result, result.Error
	}

	result.BootTime = time.Since(bootStart)
	logger("  VM booted in %s", result.BootTime)

	// Step 8: Establish persistent connection and measure ping performance
	logger("Testing vsock communication...")

	// Establish persistent connection
	connectCtx, connectCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := client.Connect(connectCtx); err != nil {
		connectCancel()
		result.Error = fmt.Errorf("failed to establish persistent connection: %w", err)
		return result, result.Error
	}
	connectCancel()
	defer client.Close()

	// First ping with persistent connection
	initialStart := time.Now()
	ctx1, cancel1 := context.WithTimeout(context.Background(), opts.PingTimeout)
	if err := client.Ping(ctx1, "initial"); err != nil {
		cancel1()
		result.Error = fmt.Errorf("initial ping failed: %w (see log: %s)", err, logPath)
		return result, result.Error
	}
	cancel1()
	initialLatency := time.Since(initialStart)
	logger("  Initial ping (persistent conn): %s", initialLatency)

	// Do 10 additional pings on the same connection
	logger("  Running 10 pings on same connection...")
	var latencies []time.Duration
	for i := 0; i < 10; i++ {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), opts.PingTimeout)
		if err := client.Ping(ctx, fmt.Sprintf("test-%d", i)); err != nil {
			cancel()
			result.Error = fmt.Errorf("ping %d failed: %w (see log: %s)", i+1, err, logPath)
			return result, result.Error
		}
		cancel()
		latencies = append(latencies, time.Since(start))
	}

	// Calculate statistics
	var sum time.Duration
	minLatency := latencies[0]
	maxLatency := latencies[0]
	for _, lat := range latencies {
		sum += lat
		if lat < minLatency {
			minLatency = lat
		}
		if lat > maxLatency {
			maxLatency = lat
		}
	}
	avgLatency := sum / time.Duration(len(latencies))

	logger("  Initial latency: %s", initialLatency)
	logger("  Reused connection - 10 pings: avg=%s min=%s max=%s", avgLatency, minLatency, maxLatency)

	result.PingRoundTrip = avgLatency

	// Step 9: Success!
	result.Success = true
	totalTime := time.Since(startTime)
	logger("\nTest completed successfully in %s", totalTime)
	logger("  Kernel: %s", filepath.Base(kernelPath))
	logger("  Boot time: %s", result.BootTime)
	logger("  Vsock ping: %s", result.PingRoundTrip)

	return result, nil
}

// getKernelPath finds the kernel binary path for the given version
func getKernelPath(version string) (string, error) {
	if version == "" {
		// Use default kernel
		kernelName, err := config.GetKernelName()
		if err != nil {
			return "", fmt.Errorf("failed to get kernel name: %w", err)
		}
		defaultLink := filepath.Join(config.GlobalPaths.DataDir, kernelName)
		target, err := os.Readlink(defaultLink)
		if err != nil {
			return "", fmt.Errorf("no default kernel set: %w", err)
		}

		// Extract version from symlink target path
		// Path format: .../kernels/6.19-20260210T183541/vmlinux-6.19-20260210T183541-x86_64
		// Get parent directory name which is the version
		version = filepath.Base(filepath.Dir(target))
	}

	kernelDir := filepath.Join(config.GlobalPaths.DataDir, "kernels", version)
	entries, err := os.ReadDir(kernelDir)
	if err != nil {
		return "", fmt.Errorf("kernel version not found: %s", version)
	}

	// Find vmlinux file (not compressed)
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) != ".xz" && filepath.Ext(entry.Name()) != ".sha256" {
			if filepath.Ext(entry.Name()) == "" || entry.Name()[:7] == "vmlinux" {
				return filepath.Join(kernelDir, entry.Name()), nil
			}
		}
	}

	return "", fmt.Errorf("vmlinux not found in kernel directory: %s", kernelDir)
}

// getFirecrackerBinary finds the Firecracker binary
func getFirecrackerBinary() (string, error) {
	// Use default Firecracker
	defaultLink := filepath.Join(config.GlobalPaths.DataDir, "firecracker", "default")
	target, err := os.Readlink(defaultLink)
	if err != nil {
		return "", fmt.Errorf("no default Firecracker version set: %w", err)
	}

	version := filepath.Base(target)
	binaryPath := filepath.Join(config.GlobalPaths.DataDir, "firecracker", version, "firecracker")

	if _, err := os.Stat(binaryPath); err != nil {
		return "", fmt.Errorf("firecracker binary not found: %s", binaryPath)
	}

	return binaryPath, nil
}

// createTestConfig creates a Firecracker configuration file for testing
func createTestConfig(configPath, kernelPath, rootfsPath, vsockPath string) error {
	// Build architecture-specific boot args
	bootArgs := "console=ttyS0 reboot=k panic=1 pci=off root=/dev/vda rw init=/init"

	// Add keep_bootcon for aarch64
	arch, err := config.GetArch()
	if err == nil && arch == "aarch64" {
		bootArgs = "keep_bootcon " + bootArgs
	}

	cfg := map[string]interface{}{
		"boot-source": map[string]interface{}{
			"kernel_image_path": kernelPath,
			"boot_args":         bootArgs,
		},
		"drives": []map[string]interface{}{
			{
				"drive_id":       "rootfs",
				"path_on_host":   rootfsPath,
				"is_root_device": true,
				"is_read_only":   false,
			},
		},
		"machine-config": map[string]interface{}{
			"vcpu_count":   1,
			"mem_size_mib": 512,
			"smt":          false,
		},
		"vsock": map[string]interface{}{
			"guest_cid": 3,
			"uds_path":  vsockPath,
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
