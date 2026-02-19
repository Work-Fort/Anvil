// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all configuration values",
		Long: `List all configuration values with their sources.

Shows all configuration keys currently set, along with their values
and where they come from (ENV, local config, user config, or default).

Output format: key = value (source)`,
		Example: `  # List all configuration
  anvil config list

  # Example output:
  # build-jobs = 8 (from ./anvil.yaml)
  # config-save = false (default)
  # default-arch = x86_64 (from ~/.config/anvil/config.yaml)
  # github-token = ghp_xxxxx (from ~/.config/anvil/config.yaml)
  # log-level = debug (default)
  # use-tui = false (from ENV: ANVIL_USE_TUI)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Call business logic
			values, err := config.ListConfigValues()
			if err != nil {
				return err
			}

			if len(values) == 0 {
				fmt.Println("No configuration set")
				return nil
			}

			// Display each key with its source
			for _, cv := range values {
				fmt.Printf("%s = %v (%s)\n", cv.Key, cv.Value, cv.Source)
			}

			// Show configuration precedence info
			fmt.Println("\n" + config.CurrentTheme.SubtleStyle().Render("Configuration precedence: ENV > local config > user config > defaults"))

			return nil
		},
	}

	return cmd
}
