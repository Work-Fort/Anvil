// SPDX-License-Identifier: Apache-2.0
package firecracker

import (
	"fmt"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed Firecracker versions",
		Long:  `List all locally installed Firecracker versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If terminal is interactive, show TUI selector
			if cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("firecracker")
			}

			versions, err := firecracker.List(config.GlobalPaths)
			if err != nil {
				return err
			}

			theme := config.CurrentTheme
			titleStyle := theme.InfoStyle().Bold(true)
			markerStyle := theme.SuccessStyle()
			versionStyle := theme.InfoStyle()
			subtleStyle := theme.SubtleStyle()

			fmt.Println()
			fmt.Println(titleStyle.Render("Installed Firecracker versions"))
			fmt.Println()

			if len(versions) == 0 {
				fmt.Println(subtleStyle.Render("  No Firecracker versions installed"))
				fmt.Println()
				fmt.Println(subtleStyle.Render("Download Firecracker with:"))
				fmt.Println(subtleStyle.Render("  anvil download firecracker <version>"))
				return nil
			}

			for _, v := range versions {
				if v.IsDefault {
					fmt.Printf("  %s %s %s\n",
						markerStyle.Render("●"),
						versionStyle.Render(v.Version),
						subtleStyle.Render("(default)"))
				} else {
					fmt.Printf("    %s\n", versionStyle.Render(v.Version))
				}
			}

			fmt.Println()
			fmt.Println(subtleStyle.Render("Set default with:"))
			fmt.Println(subtleStyle.Render("  anvil set firecracker <version>"))

			return nil
		},
	}
}
