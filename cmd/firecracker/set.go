// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "set [version]",
		Aliases: []string{"default"},
		Short:   "Set default Firecracker version",
		Long:    `Set a Firecracker version as the default.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no version specified and terminal is interactive, show TUI selector
			if len(args) == 0 && cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("firecracker")
			}
			if len(args) == 0 {
				return cmd.Usage()
			}
			return firecracker.Set(args[0])
		},
	}
}
