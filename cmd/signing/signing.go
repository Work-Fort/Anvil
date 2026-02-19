// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/spf13/cobra"
)

// NewSigningCmd creates the signing command and its subcommands
func NewSigningCmd() *cobra.Command {
	var (
		// Flags for generate-key and rotate commands
		keyName   string
		keyEmail  string
		keyExpiry string
		keyFormat string // "armored" or "binary"
	)

	cmd := &cobra.Command{
		Use:   "signing",
		Short: "Manage PGP signing keys",
		Long:  `Generate, rotate, and manage PGP signing keys for artifact signing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Show help by default
			return cmd.Help()
		},
	}

	// Create subcommands
	generateCmd := newGenerateCmd(&keyName, &keyEmail, &keyExpiry, &keyFormat)
	rotateCmd := newRotateCmd(&keyName, &keyEmail, &keyExpiry, &keyFormat)

	// Add flags to generate and rotate commands (defaults from config)
	generateCmd.Flags().StringVar(&keyName, "name", config.GetSigningKeyName(), "Key owner name")
	generateCmd.Flags().StringVar(&keyEmail, "email", config.GetSigningKeyEmail(), "Key email")
	generateCmd.Flags().StringVar(&keyExpiry, "expiry", config.GetSigningKeyExpiry(), "Key expiration (0=never, <n>=days, <n>w=weeks, <n>m=months, <n>y=years)")
	generateCmd.Flags().StringVar(&keyFormat, "format", config.GetSigningKeyFormat(), "Key format: armored (ASCII .asc) or binary (.gpg)")

	rotateCmd.Flags().StringVar(&keyName, "name", config.GetSigningKeyName(), "Key owner name")
	rotateCmd.Flags().StringVar(&keyEmail, "email", config.GetSigningKeyEmail(), "Key email")
	rotateCmd.Flags().StringVar(&keyExpiry, "expiry", config.GetSigningKeyExpiry(), "Key expiration (0=never, <n>=days, <n>w=weeks, <n>m=months, <n>y=years)")
	rotateCmd.Flags().StringVar(&keyFormat, "format", config.GetSigningKeyFormat(), "Key format: armored (ASCII .asc) or binary (.gpg)")

	// Add all subcommands
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(generateCmd)
	cmd.AddCommand(rotateCmd)
	cmd.AddCommand(newSignCmd())
	cmd.AddCommand(newVerifyCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newImportKeyCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newCheckExpiryCmd())
	cmd.AddCommand(newRemoveCmd())

	return cmd
}
