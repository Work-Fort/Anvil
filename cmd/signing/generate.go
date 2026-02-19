// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"
	"path/filepath"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/Work-Fort/Anvil/pkg/ui"
	"github.com/spf13/cobra"
)

func newGenerateCmd(keyName, keyEmail, keyExpiry, keyFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:   "generate",
		Short: "Generate a new PGP signing key",
		Long: `Generate a new PGP key for signing kernel release artifacts.

This creates:
  - Public key exported to keys/signing-key.asc
  - Private key exported to keys/signing-key-private.asc (encrypted)
  - Initial backup in keys/backups/initial-* (global mode only)

You will be prompted to enter a password to encrypt the private key.
The password can be provided via:
  - Interactive prompt (default)
  - Environment variable: ANVIL_SIGNING_PASSWORD
  - Stdin (for scripts)

Options can be set via flags or will use defaults.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			// Resolve effective values: explicit flags override repo config defaults.
			// config.GetSigningKey*() are called here (after LoadConfig) so they
			// reflect the full precedence chain: ENV > repo config > user config > defaults.
			name := config.GetSigningKeyName()
			if cmd.Flags().Changed("name") {
				name = *keyName
			}
			email := config.GetSigningKeyEmail()
			if cmd.Flags().Changed("email") {
				email = *keyEmail
			}
			expiry := config.GetSigningKeyExpiry()
			if cmd.Flags().Changed("expiry") {
				expiry = *keyExpiry
			}
			fmtStr := config.GetSigningKeyFormat()
			if cmd.Flags().Changed("format") {
				fmtStr = *keyFormat
			}

			// Parse format
			format := signing.KeyFormatArmored
			if fmtStr == "binary" {
				format = signing.KeyFormatBinary
			}

			// Get password for encryption if enabled
			var password string
			var err error
			if config.GetSigningEncryptedKeys() {
				password, err = ui.PasswordInputConfirm(
					"Enter password to encrypt signing key",
					"Confirm password",
				)
				if err != nil {
					return fmt.Errorf("failed to get password: %w", err)
				}
			}

			opts := signing.GenerateKeyOptions{
				Name:     name,
				Email:    email,
				Expiry:   expiry,
				Format:   format,
				Password: password,
			}

			fmt.Println()
			fmt.Println(subtleStyle.Render("Generating PGP signing key..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Name:"), valueStyle.Render(name))
			fmt.Printf("  %s %s\n", labelStyle.Render("Email:"), valueStyle.Render(email))
			fmt.Printf("  %s %s\n", labelStyle.Render("Expiry:"), valueStyle.Render(expiry))
			fmt.Println()

			keyInfo, err := signing.GenerateKey(opts)
			if err != nil {
				return fmt.Errorf("failed to generate key: %w", err)
			}

			fmt.Printf("%s Signing key generated successfully!\n", successStyle.Render("✓"))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("Key ID:"), valueStyle.Render(keyInfo.KeyID))
			fmt.Printf("  %s %s\n", labelStyle.Render("Fingerprint:"), valueStyle.Render(keyInfo.Fingerprint))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("Public key:"), valueStyle.Render(filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")))
			fmt.Printf("  %s %s\n", labelStyle.Render("Private key:"), valueStyle.Render(filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")))
			fmt.Println()

			return nil
		},
	}
}
