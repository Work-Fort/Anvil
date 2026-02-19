// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/spf13/cobra"
)

func newVersionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "Show available kernel versions",
		Long:  `Show the latest available kernel versions from GitHub releases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If terminal is interactive, show TUI selector
			if cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("kernel")
			}
			return kernel.ShowVersions()
		},
	}
}
