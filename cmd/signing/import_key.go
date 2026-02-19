// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"
	"os"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newImportKeyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import-key [file]",
		Short: "Import a public key",
		Long:  `Import a public key from a file into the local keyring.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyPath := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Importing signing key..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("File:"), valueStyle.Render(keyPath))
			fmt.Println()

			// Read key file
			keyData, err := os.ReadFile(keyPath)
			if err != nil {
				return fmt.Errorf("failed to read key file: %w", err)
			}

			if err := signing.ImportKey(keyData); err != nil {
				return fmt.Errorf("failed to import key: %w", err)
			}

			// Get imported key info
			keys, err := signing.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}
			if len(keys) == 0 {
				return fmt.Errorf("key imported but not found")
			}

			fmt.Printf("%s Signing key imported successfully!\n", successStyle.Render("✓"))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("Key ID:"), valueStyle.Render(keys[0].KeyID))
			fmt.Printf("  %s %s\n", labelStyle.Render("Name:"), valueStyle.Render(keys[0].Name))
			fmt.Printf("  %s %s\n", labelStyle.Render("Email:"), valueStyle.Render(keys[0].Email))
			fmt.Println()

			return nil
		},
	}
}
