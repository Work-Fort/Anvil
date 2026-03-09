// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newSignCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sign [artifacts-dir]",
		Short: "Sign release artifacts",
		Long: `Sign the SHA256SUMS file in the artifacts directory using the current signing key.

If the signing key is encrypted, you will be prompted to enter the password.
The password can be provided via:
  - Interactive prompt (default)
  - Environment variable: ANVIL_SIGNING_PASSWORD
  - Stdin (for scripts)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactsDir := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Signing artifacts..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Directory:"), valueStyle.Render(artifactsDir))
			fmt.Println()

			// Acquire password at the CLI layer (interface concern)
			password, err := GetSigningPassword(
				PasswordSourceAuto,
				"Enter password to unlock signing key",
			)
			if err != nil {
				return fmt.Errorf("failed to get password: %w", err)
			}

			if err := signing.SignArtifacts(artifactsDir, password); err != nil {
				return fmt.Errorf("failed to sign artifacts: %w", err)
			}

			fmt.Printf("%s Artifacts signed successfully!\n", successStyle.Render("✓"))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("Signature:"), valueStyle.Render(artifactsDir+"/SHA256SUMS.asc"))
			fmt.Println()

			return nil
		},
	}
}
