// SPDX-License-Identifier: Apache-2.0
package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCmd creates the version command
func NewVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the current version of anvil.`,
		Run: func(cmd *cobra.Command, args []string) {
			if version == "" {
				version = "dev"
			}
			fmt.Printf("anvil version %s\n", version)
		},
	}
}
