// SPDX-License-Identifier: Apache-2.0
package config

import (
	"github.com/spf13/cobra"
)

var (
	// globalFlag determines whether to operate on user config vs local config
	globalFlag bool
)

// NewConfigCmd creates the config command and its subcommands
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage anvil configuration",
		Long: `Manage anvil configuration settings.

Configuration precedence (highest to lowest):
  1. Environment variables (ANVIL_*)
  2. Local config (./anvil.yaml)
  3. User config (~/.config/anvil/config.yaml)
  4. Defaults

By default, config commands operate on local config (./anvil.yaml).
Use --global to operate on user config instead.`,
		Example: `  # Set local config (project-specific)
  crbl config set use-tui false
  crbl config set build-jobs 8

  # Set global config (user preferences)
  crbl config set --global github-token ghp_xxxxx
  crbl config set --global default-arch x86_64

  # Get configuration value
  crbl config get use-tui

  # Remove configuration value
  crbl config unset build-jobs
  crbl config unset --global github-token

  # List all configuration
  crbl config list`,
	}

	// Add subcommands
	cmd.AddCommand(newSetCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newUnsetCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newSchemaCmd())

	return cmd
}

// addGlobalFlag adds the --global flag to a command
func addGlobalFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Operate on user config instead of local config")
}
