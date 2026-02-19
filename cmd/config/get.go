// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get configuration value",
		Long: `Get a configuration value and show its source.

The source indicates where the value comes from in precedence order:
  - ENV: Environment variable (ANVIL_*)
  - Local: Local config file (./anvil.yaml)
  - User: User config file (~/.config/anvil/config.yaml)
  - Default: Built-in default value`,
		Args: cobra.ExactArgs(1),
		Example: `  # Get a configuration value
  crbl config get use-tui

  # Get nested value
  crbl config get firecracker.version

  # Output shows value and source:
  # use-tui = true (from ENV: ANVIL_USE_TUI)
  # log-level = debug (from ./anvil.yaml)
  # github-token = ghp_xxxxx (from ~/.config/anvil/config.yaml)
  # default-arch = x86_64 (default)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			// Call business logic
			configValue, err := config.GetConfigValue(key)
			if err != nil {
				return err
			}

			// Display value with source
			fmt.Printf("%s = %v (%s)\n", configValue.Key, configValue.Value, configValue.Source)

			return nil
		},
	}

	return cmd
}
