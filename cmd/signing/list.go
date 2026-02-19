// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all signing keys",
		Long:  `List all PGP keys in the local keyring.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			theme := config.CurrentTheme
			titleStyle := theme.InfoStyle().Bold(true)
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()
			subtleStyle := theme.SubtleStyle()

			keys, err := signing.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}

			fmt.Println()
			fmt.Println(titleStyle.Render("Local signing keys"))
			fmt.Println()

			if len(keys) == 0 {
				fmt.Println(subtleStyle.Render("  No signing keys found"))
				fmt.Println()
				fmt.Println(subtleStyle.Render("Generate a key with:"))
				fmt.Println(subtleStyle.Render("  anvil signing generate"))
				return nil
			}

			for _, key := range keys {
				fmt.Printf("  %s %s\n", labelStyle.Render("Key ID:"), valueStyle.Render(key.KeyID))
				fmt.Printf("  %s %s\n", labelStyle.Render("Name:"), valueStyle.Render(key.Name))
				fmt.Printf("  %s %s\n", labelStyle.Render("Email:"), valueStyle.Render(key.Email))
				fmt.Printf("  %s %s\n", labelStyle.Render("Fingerprint:"), valueStyle.Render(key.Fingerprint))
				fmt.Printf("  %s %s\n", labelStyle.Render("Created:"), valueStyle.Render(key.Created.Format("2006-01-02")))
				if !key.Expires.IsZero() {
					fmt.Printf("  %s %s\n", labelStyle.Render("Expires:"), valueStyle.Render(key.Expires.Format("2006-01-02")))
				} else {
					fmt.Printf("  %s %s\n", labelStyle.Render("Expires:"), valueStyle.Render("Never"))
				}
			}

			fmt.Println()
			return nil
		},
	}
}
