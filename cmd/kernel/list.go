// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed kernels",
		Long:  `List all locally installed kernel versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If terminal is interactive, show TUI selector
			if cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("kernel")
			}
			return kernel.List()
		},
	}
}
