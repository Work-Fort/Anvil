// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"fmt"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [version]",
		Short: "Remove an installed Firecracker version",
		Long:  `Remove a locally installed Firecracker version.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no version specified and terminal is interactive, show TUI selector
			if len(args) == 0 && cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("firecracker")
			}
			if len(args) == 0 {
				return cmd.Usage()
			}
			if err := firecracker.Remove(args[0], config.GlobalPaths); err != nil {
				return err
			}
			theme := config.CurrentTheme
			fmt.Println()
			fmt.Println(theme.SuccessMessage(fmt.Sprintf("Deleted firecracker version %s", args[0])))
			return nil
		},
	}
}
