// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"fmt"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "set [version]",
		Aliases: []string{"default"},
		Short:   "Set default kernel version",
		Long:    `Set a kernel version as the default.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no version specified and terminal is interactive, show TUI selector
			if len(args) == 0 && cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("kernel")
			}
			if len(args) == 0 {
				return cmd.Usage()
			}
			version := args[0]
			if err := kernel.Set(version, config.GlobalPaths); err != nil {
				return err
			}
			fmt.Printf("Kernel %s set as default\n", version)
			return nil
		},
	}
}
