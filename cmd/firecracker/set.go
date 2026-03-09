// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"fmt"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
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
			if err := firecracker.Set(args[0], config.GlobalPaths); err != nil {
				return err
			}
			fmt.Printf("✓ Firecracker %s set as default\n\n", args[0])
			fmt.Println("Run 'firecracker --version' to verify")
			return nil
		},
	}
}
