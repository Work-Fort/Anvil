// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"fmt"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/github"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/spf13/cobra"
)

func newVersionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "Show available kernel versions",
		Long:  `Show the latest available kernel versions from GitHub releases.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If terminal is interactive, show TUI selector
			if cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("kernel")
			}

			client := github.NewClient(config.GetGitHubToken(), config.GitHubAPI)
			versions, err := kernel.ShowVersions(client, config.GlobalPaths)
			if err != nil {
				return err
			}

			theme := config.CurrentTheme
			titleStyle := theme.InfoStyle().Bold(true)
			defaultMarkerStyle := theme.SuccessStyle()
			installedMarkerStyle := theme.InfoStyle()
			versionStyle := theme.InfoStyle()
			subtleStyle := theme.SubtleStyle()

			fmt.Println()
			fmt.Printf("%s %s\n", titleStyle.Render("Available kernel versions"), subtleStyle.Render("(latest 10)"))
			fmt.Println()

			if len(versions) == 0 {
				fmt.Println(subtleStyle.Render("  No kernel releases available yet"))
				fmt.Println()
				fmt.Println(subtleStyle.Render("Kernel releases are built automatically when new versions are released."))
				fmt.Println(subtleStyle.Render("You can also request a specific version by creating a build request issue."))
				return nil
			}

			for _, v := range versions {
				if v.IsDefault {
					fmt.Printf("  %s %s\n",
						defaultMarkerStyle.Render("·"),
						versionStyle.Render(v.Version))
				} else if v.IsInstalled {
					fmt.Printf("  %s %s\n",
						installedMarkerStyle.Render("-"),
						versionStyle.Render(v.Version))
				} else {
					fmt.Printf("    %s\n", versionStyle.Render(v.Version))
				}
			}

			fmt.Println()
			fmt.Println(subtleStyle.Render("Legend: · default  - installed"))
			fmt.Println()
			fmt.Println(subtleStyle.Render("Download with:"))
			fmt.Println(subtleStyle.Render("  anvil download kernel <version>"))

			return nil
		},
	}
}
