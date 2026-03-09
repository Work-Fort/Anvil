// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"fmt"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/Work-Fort/Anvil/pkg/github"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "get [version]",
		Aliases: []string{"download"},
		Short:   "Download a Firecracker binary",
		Long: `Download a Firecracker binary from GitHub releases.

If no version is specified, the latest version will be downloaded.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := ""
			if len(args) > 0 {
				version = args[0]
			}

			// If no version specified and terminal is interactive, show TUI selector
			if version == "" && cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("firecracker")
			}

			client := github.NewClient(config.GetGitHubToken(), config.GitHubAPI)
			if err := firecracker.Download(version, client, config.GlobalPaths); err != nil {
				return err
			}

			fmt.Printf("✓ Firecracker %s downloaded successfully\n", version)
			return nil
		},
	}
}
