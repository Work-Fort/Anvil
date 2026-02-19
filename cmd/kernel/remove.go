// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [version]",
		Short: "Remove an installed kernel",
		Long:  `Remove a locally installed kernel version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no version specified and terminal is interactive, show TUI selector
			if len(args) == 0 && cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("kernel")
			}
			if len(args) == 0 {
				return cmd.Usage()
			}
			return cmdutil.DeleteVersion("kernel", args[0])
		},
	}
}
