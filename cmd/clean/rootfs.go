// SPDX-License-Identifier: Apache-2.0
package clean

import (
	"github.com/spf13/cobra"
)

func newRootfsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rootfs",
		Short: "Clean rootfs images",
		Long:  `Remove Alpine rootfs images created for Firecracker VMs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanRootfs()
		},
	}
}
