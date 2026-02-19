// SPDX-License-Identifier: Apache-2.0
package buildkernel

import (
	"fmt"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/Work-Fort/Anvil/pkg/ui"
	"github.com/spf13/cobra"
)

// NewBuildKernelCmd creates the build-kernel command
func NewBuildKernelCmd() *cobra.Command {
	var (
		buildArch              string
		buildVersion           string
		buildVerificationLevel string
		buildConfig            string
		buildForceRebuild      bool
	)

	cmd := &cobra.Command{
		Use:     "build-kernel [version]",
		Aliases: []string{"build"},
		Short:   "Build kernel from source",
		Long: `Build Firecracker-compatible kernel from source.

Downloads kernel source from kernel.org, verifies integrity, and builds
with Firecracker-optimized configuration.

If no version is specified, builds the latest stable kernel.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			version := buildVersion
			if version == "" && len(args) > 0 {
				version = args[0]
			}

			// If interactive and no version specified, run wizard
			// Wizard handles EVERYTHING: version selection + build + progress
			if version == "" && cmdutil.IsInteractive() {
				err := ui.RunBuildKernelWizard(buildArch, buildVerificationLevel, buildConfig, buildForceRebuild)
				if err != nil {
					// Check if user cancelled - exit gracefully without error
					if err == ui.ErrUserCancelled {
						return nil
					}
					return err
				}
				// Wizard completed the build - we're done!
				theme := config.CurrentTheme
				fmt.Println()
				fmt.Println(theme.SuccessMessage("Kernel build completed"))
				fmt.Println()
				fmt.Printf("Built artifacts are in: %s/artifacts/\n", config.GlobalPaths.KernelBuildDir)
				return nil
			}

			// Non-interactive path: validate and build directly
			// If still no version, use latest (handled in kernel.Build())

			// Check for cached build in non-interactive mode
			if !buildForceRebuild {
				hasCached, _, err := kernel.CheckCachedBuild(version)
				if err != nil {
					return fmt.Errorf("failed to check for cached build: %w", err)
				}
				if hasCached {
					return fmt.Errorf("cached build exists. Use --force-rebuild to rebuild, or use the interactive wizard to install the cached build")
				}
			}

			// Validate version against kernel.org releases if specified
			if version != "" && version != "latest" {
				if err := kernel.ValidateVersion(version); err != nil {
					return err
				}
			}

			opts := kernel.BuildOptions{
				Version:           version,
				Arch:              buildArch,
				VerificationLevel: buildVerificationLevel,
				ConfigFile:        buildConfig,
				Interactive:       false, // Non-interactive path
			}

			if err := kernel.Build(opts); err != nil {
				return err
			}

			theme := config.CurrentTheme

			fmt.Println()
			fmt.Println(theme.SuccessMessage("Kernel build completed"))
			fmt.Println()
			fmt.Printf("Built artifacts are in: %s/artifacts/\n", config.GlobalPaths.KernelBuildDir)

			return nil
		},
	}

	cmd.Flags().StringVarP(&buildVersion, "version", "v", "", "Kernel version to build (default: latest, shows wizard if interactive)")
	cmd.Flags().StringVarP(&buildArch, "arch", "a", "", "Target architecture: x86_64, aarch64, or all (default: host)")
	cmd.Flags().StringVarP(&buildVerificationLevel, "verification-level", "q", "", "Verification level: high, medium, disabled (default: high)")
	cmd.Flags().StringVarP(&buildConfig, "config", "c", "", "Custom kernel config file")
	cmd.Flags().BoolVarP(&buildForceRebuild, "force-rebuild", "f", false, "Force rebuild even if cached build exists")

	return cmd
}
