// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/spf13/cobra"
)

func newUnsetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset [key]",
		Short: "Remove configuration value",
		Long: `Remove a configuration key from a config file.

Keys use dot notation for nested values (e.g., firecracker.version).

**Note:**
  - Removing a parent key removes all nested values (e.g., unsetting 'firecracker' removes 'firecracker.version' and all other children)
  - Environment variables and defaults will still apply after removal`,
		Args: cobra.ExactArgs(1),
		Example: `  # Remove from local config
  crbl config unset use-tui
  crbl config unset build-jobs

  # Remove from user config
  crbl config unset --global github-token

  # Remove nested value
  crbl config unset firecracker.version

  # Remove parent (removes all children)
  crbl config unset firecracker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			// Determine scope
			scope := config.ScopeRepo
			if globalFlag {
				scope = config.ScopeUser
			}

			// Call business logic
			if err := config.UnsetConfigValue(key, scope); err != nil {
				return err
			}

			// Show success message
			scopeName := "local"
			configFile := config.LocalConfigFile + config.DefaultConfigExt
			if globalFlag {
				scopeName = "global"
				configFile = "~/.config/anvil/" + config.ConfigFileName + config.DefaultConfigExt
			}
			fmt.Printf("Removed %s from %s config (%s)\n", key, scopeName, configFile)

			return nil
		},
	}

	addGlobalFlag(cmd)
	return cmd
}
