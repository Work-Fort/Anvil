// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify [artifacts-dir]",
		Short: "Verify release artifacts signature",
		Long:  `Verify the PGP signature on SHA256SUMS in the artifacts directory.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactsDir := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Verifying artifacts signature..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Directory:"), valueStyle.Render(artifactsDir))
			fmt.Println()

			if err := signing.VerifyArtifacts(artifactsDir); err != nil {
				return fmt.Errorf("failed to verify artifacts: %w", err)
			}

			fmt.Printf("%s Signature verified!\n", successStyle.Render("✓"))
			fmt.Println()

			return nil
		},
	}
}
