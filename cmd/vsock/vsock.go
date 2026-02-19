// SPDX-License-Identifier: Apache-2.0
package vsock

import (
	"github.com/spf13/cobra"
)

// NewVsockCmd creates the vsock command group
func NewVsockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vsock",
		Short: "Manage vsock communication",
		Long: `Manage vsock (Virtual Socket) communication between host and guest VMs.

vsock provides a zero-configuration communication channel between the host
and Firecracker VMs using virtio-vsock. This is useful for testing that
kernels and rootfs images have working virtio features.`,
	}

	// Add subcommands
	cmd.AddCommand(newServerCmd())
	cmd.AddCommand(newClientCmd())

	return cmd
}
