// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove [key-id]",
		Short: "Remove a signing key",
		Long:  `Remove a PGP key from the local keyring.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyID := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Removing signing key..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Key ID:"), valueStyle.Render(keyID))
			fmt.Println()

			if err := signing.RemoveKey(); err != nil {
				return fmt.Errorf("failed to remove key: %w", err)
			}

			fmt.Printf("%s Signing key removed successfully!\n", successStyle.Render("✓"))
			fmt.Println()

			return nil
		},
	}
}
