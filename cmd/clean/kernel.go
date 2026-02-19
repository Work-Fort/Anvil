// SPDX-License-Identifier: Apache-2.0
package clean

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/spf13/cobra"
)

func newKernelCmd(removeInactive, allDangerous, force *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "kernel [version]",
		Short: "Clean kernel data",
		Long:  `Clean kernel cache and optionally remove kernel versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// Remove specific version
				return cmdutil.DeleteVersion("kernel", args[0])
			}

			if *allDangerous {
				return cleanAllKernels(*force)
			} else if *removeInactive {
				return cleanInactiveKernels()
			}
			return cleanCache()
		},
	}
}
