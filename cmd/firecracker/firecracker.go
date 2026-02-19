// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/spf13/cobra"
)

// NewFirecrackerCmd creates the firecracker command and its subcommands
func NewFirecrackerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firecracker",
		Short: "Manage Firecracker binaries",
		Long:  `Download, list, and manage Firecracker binary versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If terminal is interactive, show TUI selector
			if cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("firecracker")
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
	cmd.AddCommand(newCreateRootfsCmd())
	cmd.AddCommand(newTestCmd())

	return cmd
}
