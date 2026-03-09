// SPDX-License-Identifier: Apache-2.0
package kernel

import (
	"fmt"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List installed kernels",
		Long:  `List all locally installed kernel versions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If terminal is interactive, show TUI selector
			if cmdutil.IsInteractive() {
				return cmdutil.ShowVersionSelector("kernel")
			}

			kernels, arch, err := kernel.List(config.GlobalPaths)
			if err != nil {
				return err
			}

			theme := config.CurrentTheme
			titleStyle := theme.InfoStyle().Bold(true)
			markerStyle := theme.SuccessStyle()
			versionStyle := theme.InfoStyle()
			subtleStyle := theme.SubtleStyle()

			fmt.Println()
			fmt.Printf("%s %s\n", titleStyle.Render("Installed kernels"), subtleStyle.Render(fmt.Sprintf("(%s)", arch)))
			fmt.Println()

			if len(kernels) == 0 {
				fmt.Println(subtleStyle.Render("  No kernels installed"))
				fmt.Println()
				fmt.Println(subtleStyle.Render("Download a kernel with:"))
				fmt.Println(subtleStyle.Render("  anvil download kernel <version>"))
				return nil
			}

			for _, ki := range kernels {
				if ki.IsDefault {
					fmt.Printf("  %s %s %s\n",
						markerStyle.Render("●"),
						versionStyle.Render(ki.Version),
						subtleStyle.Render("(default)"))
				} else {
					fmt.Printf("    %s\n", versionStyle.Render(ki.Version))
				}
			}

			fmt.Println()
			fmt.Println(subtleStyle.Render("Set default with:"))
			fmt.Println(subtleStyle.Render("  anvil set kernel <version>"))

			return nil
		},
	}
}
