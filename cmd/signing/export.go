// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [output-path]",
		Short: "Export encrypted backup of signing key",
		Long: `Export an encrypted backup of the signing key to a file.

If the signing key is encrypted at rest, you will first be prompted for
the storage password to decrypt it, then prompted for a new passphrase
to encrypt the backup.

The backup file will NOT overwrite existing files.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputPath := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Exporting encrypted backup..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Output:"), valueStyle.Render(outputPath))
			fmt.Println()

			// Get email from key info
			keys, err := signing.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}
			if len(keys) == 0 {
				return fmt.Errorf("no signing key found")
			}

			if err := signing.ExportEncryptedBackup(keys[0].Email, outputPath); err != nil {
				return fmt.Errorf("failed to export backup: %w", err)
			}

			fmt.Printf("%s Encrypted backup exported successfully!\n", successStyle.Render("✓"))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("File:"), valueStyle.Render(outputPath))
			fmt.Println()

			return nil
		},
	}
}
