// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed Firecracker versions",
		Long:  `List all locally installed Firecracker versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If terminal is interactive, show TUI selector
			if cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("firecracker")
			}
			return firecracker.List()
		},
	}
}
