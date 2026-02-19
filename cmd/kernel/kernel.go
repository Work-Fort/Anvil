// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/spf13/cobra"
)

// NewKernelCmd creates the kernel command and its subcommands
func NewKernelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kernel",
		Short: "Manage Firecracker kernel binaries",
		Long:  `Download, list, and manage Firecracker-compatible kernel binaries.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If terminal is interactive, show TUI selector
			if cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("kernel")
			}
			// Non-interactive: show help
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newVersionsCmd())
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newRemoveCmd())
	cmd.AddCommand(newVersionCheckCmd())

	return cmd
}
