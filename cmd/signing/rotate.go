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

func newRotateCmd(keyName, keyEmail, keyExpiry, keyFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:   "rotate",
		Short: "Rotate the signing key",
		Long: `Rotate the PGP signing key by generating a new key and backing up the old one.

This:
  - Archives the current key to keys/backups/<timestamp>/
  - Generates a new key with the same or updated parameters
  - Updates the current signing keys

You will be prompted to enter a password for the new signing key.
The old key is backed up (encrypted) but no longer used for signing new releases.`,
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

			// Get password for new key encryption if enabled
			var password string
			var err error
			if config.GetSigningEncryptedKeys() {
				password, err = ui.PasswordInputConfirm(
					"Enter password for new signing key",
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
			fmt.Println(subtleStyle.Render("Rotating PGP signing key..."))
			fmt.Println()

			keyInfo, err := signing.RotateKey(opts)
			if err != nil {
				return fmt.Errorf("failed to rotate key: %w", err)
			}

			fmt.Printf("%s Signing key rotated successfully!\n", successStyle.Render("✓"))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("New Key ID:"), valueStyle.Render(keyInfo.KeyID))
			fmt.Printf("  %s %s\n", labelStyle.Render("Fingerprint:"), valueStyle.Render(keyInfo.Fingerprint))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("Public key:"), valueStyle.Render(filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")))
			fmt.Printf("  %s %s\n", labelStyle.Render("Private key:"), valueStyle.Render(filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")))
			fmt.Println()

			return nil
		},
	}
}
