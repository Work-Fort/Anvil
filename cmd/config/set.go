// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/spf13/cobra"
)

func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set configuration value",
		Long: `Set a configuration key to a value.

Keys use dot notation for nested values (e.g., firecracker.version).

Boolean values support natural language:
  - true:  true, yes, on, enable, enabled
  - false: false, no, off, disable, disabled

Numeric values are automatically detected and typed.`,
		Args: cobra.ExactArgs(2),
		Example: `  # Set boolean values (multiple formats supported)
  crbl config set use-tui true
  crbl config set use-tui enable
  crbl config set use-tui yes

  # Set string values
  crbl config set log-level debug
  crbl config set default-arch x86_64

  # Set numeric values
  crbl config set build-jobs 8

  # Set nested values with dot notation
  crbl config set firecracker.version v1.5.0

  # Set in user config instead of local
  crbl config set --global github-token ghp_xxxxx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			// Determine scope
			scope := config.ScopeRepo
			if globalFlag {
				scope = config.ScopeUser
			}

			// Call business logic
			if err := config.SetConfigValue(key, value, scope); err != nil {
				return err
			}

			// Show success message
			scopeName := "local"
			configFile := config.LocalConfigFile + config.DefaultConfigExt
			if globalFlag {
				scopeName = "global"
				configFile = "~/.config/anvil/" + config.ConfigFileName + config.DefaultConfigExt
			}
			fmt.Printf("Set %s = %s (%s: %s)\n", key, value, scopeName, configFile)

			return nil
		},
	}

	addGlobalFlag(cmd)
	return cmd
}
