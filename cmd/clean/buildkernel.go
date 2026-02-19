// SPDX-License-Identifier: Apache-2.0
package clean

import (
	"github.com/spf13/cobra"
)

func newBuildKernelCmd(cleanArch *string) *cobra.Command {
	return &cobra.Command{
		Use:     "build-kernel",
		Aliases: []string{"builds", "build"},
		Short:   "Clean kernel source and build artifacts",
		Long:    `Clean kernel source code and build artifacts created during kernel compilation.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cleanBuildKernel(*cleanArch)
		},
	}
}
