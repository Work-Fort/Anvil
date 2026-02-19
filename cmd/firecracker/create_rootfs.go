// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"path/filepath"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/rootfs"
	"github.com/spf13/cobra"
)

func newCreateRootfsCmd() *cobra.Command {
	var (
		createRootfsOutput        string
		createRootfsSizeMB        int
		createRootfsForce         bool
		createRootfsAlpineVersion string
		createRootfsAlpinePatch   string
		createRootfsInjectBinary  bool
		createRootfsBinaryPath    string
		createRootfsBinaryDest    string
	)

	cmd := &cobra.Command{
		Use:     "create-rootfs",
		Aliases: []string{"mkrootfs"},
		Short:   "Create an Alpine-based rootfs for Firecracker",
		Long: `Create an Alpine Linux-based ext4 rootfs for Firecracker VMs.

The rootfs contains:
- Alpine Linux minirootfs (configurable version)
- Init script that mounts essential filesystems
- Optional binary injection with automatic vsock server startup

This is useful for running Firecracker VMs with the anvil agent.`,
		Example: `  # Create default rootfs (512MB, Alpine 3.23.3)
  anvil firecracker create-rootfs

  # Inject anvil binary into rootfs
  anvil firecracker create-rootfs --inject-binary

  # Specific Alpine version
  anvil firecracker create-rootfs --alpine-version 3.23 --alpine-patch 2

  # Custom output and size
  anvil firecracker create-rootfs --output /tmp/my-rootfs.ext4 --size 1024

  # Inject custom binary at custom path
  anvil firecracker create-rootfs --inject-binary \
    --binary-path ./my-agent --binary-dest /usr/local/bin/agent`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set default output path if not specified
			if createRootfsOutput == "" {
				createRootfsOutput = filepath.Join(config.GlobalPaths.DataDir, "alpine-rootfs.ext4")
			}

			opts := rootfs.CreateOptions{
				OutputPath:     createRootfsOutput,
				SizeMB:         createRootfsSizeMB,
				AlpineVersion:  createRootfsAlpineVersion,
				AlpinePatch:    createRootfsAlpinePatch,
				ForceOverwrite: createRootfsForce,
				InjectBinary:   createRootfsInjectBinary,
				BinaryPath:     createRootfsBinaryPath,
				BinaryDestPath: createRootfsBinaryDest,
			}

			return rootfs.Create(opts)
		},
	}

	// Add flags to create-rootfs command
	cmd.Flags().StringVarP(&createRootfsOutput, "output", "o", "", "Output file path (default: ~/.local/share/anvil/alpine-rootfs.ext4)")
	cmd.Flags().IntVarP(&createRootfsSizeMB, "size", "s", 512, "Size in MB")
	cmd.Flags().BoolVarP(&createRootfsForce, "force", "f", false, "Overwrite existing file")
	cmd.Flags().StringVar(&createRootfsAlpineVersion, "alpine-version", "3.23", "Alpine Linux version (major.minor)")
	cmd.Flags().StringVar(&createRootfsAlpinePatch, "alpine-patch", "3", "Alpine Linux patch version")
	cmd.Flags().BoolVar(&createRootfsInjectBinary, "inject-binary", false, "Inject binary into rootfs")
	cmd.Flags().StringVar(&createRootfsBinaryPath, "binary-path", "", "Path to binary to inject (default: current executable)")
	cmd.Flags().StringVar(&createRootfsBinaryDest, "binary-dest", "/usr/bin/anvil", "Destination path in rootfs")

	return cmd
}
