// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [backup-file]",
		Short: "Import a signing key from encrypted backup",
		Long:  `Restore a signing key from an encrypted backup file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupPath := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Importing signing key from encrypted backup..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Backup:"), valueStyle.Render(backupPath))
			fmt.Println()

			if err := signing.ImportEncryptedBackup(backupPath); err != nil {
				return fmt.Errorf("failed to import from backup: %w", err)
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
			fmt.Printf("  %s %s\n", labelStyle.Render("Fingerprint:"), valueStyle.Render(keys[0].Fingerprint))
			fmt.Println()

			return nil
		},
	}
}
