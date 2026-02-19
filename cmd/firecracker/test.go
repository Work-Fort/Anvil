// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"fmt"
	"os"
	"time"

	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	var (
		kernelVersion string
		rootfsPath    string
		bootTimeout   time.Duration
		pingTimeout   time.Duration
	)

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test Firecracker with vsock communication",
		Long: `Run an end-to-end test of Firecracker functionality.

This command performs a full integration test:
1. Verifies kernel and Firecracker binary are available
2. Creates a rootfs with anvil binary if needed
3. Starts a Firecracker VM with vsock configured
4. Waits for the VM to boot
5. Tests vsock communication with ping/pong
6. Cleans up and reports results

This validates that your kernels and rootfs have working virtio-vsock features.`,
		Example: `  # Test with default kernel
  anvil firecracker test

  # Test specific kernel version
  anvil firecracker test --kernel-version 6.19

  # Test with custom rootfs
  anvil firecracker test --rootfs /path/to/rootfs.ext4

  # Test with custom timeouts
  anvil firecracker test --boot-timeout 15s --ping-timeout 5s`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := firecracker.TestOptions{
				KernelVersion: kernelVersion,
				RootfsPath:    rootfsPath,
				Writer:        os.Stdout,
				BootTimeout:   bootTimeout,
				PingTimeout:   pingTimeout,
			}

			result, err := firecracker.Test(opts)
			if err != nil {
				// Print error details
				fmt.Fprintf(os.Stderr, "\nTest failed: %v\n", err)
				return err
			}

			if !result.Success {
				return fmt.Errorf("test completed but was not successful")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&kernelVersion, "kernel-version", "", "Kernel version to test (default: use default kernel)")
	cmd.Flags().StringVar(&rootfsPath, "rootfs", "", "Path to rootfs image (default: ~/.local/share/anvil/alpine-rootfs.ext4)")
	cmd.Flags().DurationVar(&bootTimeout, "boot-timeout", 10*time.Second, "Timeout for VM boot")
	cmd.Flags().DurationVar(&pingTimeout, "ping-timeout", 10*time.Second, "Timeout for vsock ping")

	return cmd
}
