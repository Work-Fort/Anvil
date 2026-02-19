// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/spf13/cobra"
)

func newSchemaCmd() *cobra.Command {
	var outputFile string
	var scopeFlag string

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Export configuration schema",
		Long: `Export the configuration schema in JSON Schema Draft 2020-12 format.

The schema can be used for:
  - IDE autocomplete and validation
  - Documentation generation
  - Third-party tooling integration

By default, the schema includes all keys. Use --scope to filter by user or repo keys.`,
		Example: `  # Print full schema to stdout
  crbl config schema

  # Generate user-scope schema (for ~/.config/anvil/config.yaml)
  crbl config schema --scope user --output user.schema.json

  # Generate repo-scope schema (for ./anvil.yaml)
  crbl config schema --scope repo --output repo.schema.json

  # Use with VS Code (in .vscode/settings.json):
  {
    "yaml.schemas": {
      "./repo.schema.json": "anvil.yaml",
      "./user.schema.json": ".config/anvil/config.yaml"
    }
  }`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse scope filter
			var scope *config.ConfigScope
			if scopeFlag != "" {
				switch scopeFlag {
				case "user":
					s := config.ScopeUser
					scope = &s
				case "repo":
					s := config.ScopeRepo
					scope = &s
				default:
					return fmt.Errorf("invalid scope: %s (must be 'user' or 'repo')", scopeFlag)
				}
			}

			// Generate JSON schema
			schema, err := config.GenerateJSONSchemaForScope(scope)
			if err != nil {
				return fmt.Errorf("failed to generate schema: %w", err)
			}

			// Write to file or stdout
			if outputFile != "" {
				if err := os.WriteFile(outputFile, schema, 0644); err != nil {
					return fmt.Errorf("failed to write schema to file: %w", err)
				}
				fmt.Printf("Schema written to %s\n", outputFile)
			} else {
				fmt.Println(string(schema))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write schema to file instead of stdout")
	cmd.Flags().StringVar(&scopeFlag, "scope", "", "Filter by scope: user or repo (default: all)")

	return cmd
}
