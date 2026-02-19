// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/spf13/cobra"
)

func newVersionCheckCmd() *cobra.Command {
	var outputJSON bool

	cmd := &cobra.Command{
		Use:   "version-check [version]",
		Short: "Check if a kernel version is buildable",
		Long: `Check if a kernel version is available on kernel.org and has
checksums ready for verified builds.

Exits 0 if the version is buildable, exits 1 with a descriptive message if not.
If no version is specified, checks the latest stable kernel.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}

			result, err := kernel.CheckVersion(version)
			if err != nil {
				return err
			}

			if outputJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if encErr := enc.Encode(result); encErr != nil {
					return encErr
				}
			} else {
				// Human-readable output
				fmt.Printf("Version: %s\n", result.Version)
				fmt.Printf("Available: %v\n", result.Available)
				fmt.Printf("Checksums Ready: %v\n", result.ChecksumsReady)
				fmt.Printf("Buildable: %v\n", result.Buildable)
				if result.Message != "" {
					fmt.Printf("Message: %s\n", result.Message)
				}
			}

			if !result.Buildable {
				return fmt.Errorf("version %s is not buildable: %s", result.Version, result.Message)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output result as JSON")

	return cmd
}
