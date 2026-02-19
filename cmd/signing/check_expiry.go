// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"
	"time"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newCheckExpiryCmd() *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "check-expiry",
		Short: "Check if signing keys are expiring soon",
		Long:  `Check if any signing keys are expiring within the next 60 days.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			theme := config.CurrentTheme
			titleStyle := theme.InfoStyle().Bold(true)
			successStyle := theme.SuccessStyle()
			warningStyle := theme.WarningStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			keys, err := signing.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}

			fmt.Println()
			fmt.Println(titleStyle.Render("Key expiration status"))
			fmt.Println()

			now := time.Now()
			warnBefore := now.AddDate(0, 0, days)
			hasExpiringKeys := false

			for _, key := range keys {
				if key.Expires.IsZero() {
					continue
				}

				if key.Expires.Before(now) {
					fmt.Printf("%s Key expired: %s\n", warningStyle.Render("⚠"), key.KeyID)
					fmt.Printf("  %s %s\n", labelStyle.Render("Expired:"), valueStyle.Render(key.Expires.Format("2006-01-02")))
					hasExpiringKeys = true
				} else if key.Expires.Before(warnBefore) {
					fmt.Printf("%s Key expiring soon: %s\n", warningStyle.Render("⚠"), key.KeyID)
					fmt.Printf("  %s %s\n", labelStyle.Render("Expires:"), valueStyle.Render(key.Expires.Format("2006-01-02")))
					daysUntilExpiry := int(time.Until(key.Expires).Hours() / 24)
					fmt.Printf("  %s %d days\n", labelStyle.Render("Days remaining:"), daysUntilExpiry)
					hasExpiringKeys = true
				}
			}

			if !hasExpiringKeys {
				fmt.Printf("%s All keys are valid\n", successStyle.Render("✓"))
				fmt.Println()
				return nil
			}

			fmt.Println()
			return fmt.Errorf("signing key is expiring soon or has expired")
		},
	}

	cmd.Flags().IntVar(&days, "days", 60, "Warn if key expires within this many days")
	return cmd
}
