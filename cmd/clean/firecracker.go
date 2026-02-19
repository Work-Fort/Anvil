// SPDX-License-Identifier: Apache-2.0
package clean

import (
	"github.com/spf13/cobra"
)

func newFirecrackerCmd(removeInactive, allDangerous, force *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "firecracker",
		Short: "Clean Firecracker data",
		Long:  `Clean Firecracker cache and optionally remove Firecracker versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if *allDangerous {
				return cleanAllFirecracker(*force)
			} else if *removeInactive {
				return cleanInactiveFirecracker()
			}
			return cleanCache()
		},
	}
}
