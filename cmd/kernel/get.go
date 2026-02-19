// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get [version]",
		Aliases: []string{"download"},
		Short:   "Get a kernel (download or build)",
		Long:    `Get a Firecracker-compatible kernel from GitHub releases or build from source.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}

			// If no version specified and terminal is interactive, show TUI selector
			if version == "" && cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("kernel")
			}

			// Try download first, build if not available
			buildOpts := kernel.BuildOptions{
				Version:     version,
				Interactive: cmdutil.IsInteractive(),
			}
			return kernel.Get(version, &buildOpts)
		},
	}
}
