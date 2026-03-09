// SPDX-License-Identifier: Apache-2.0
package buildkernel

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Work-Fort/Anvil/cmd/cmdutil"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/Work-Fort/Anvil/pkg/ui"
	"github.com/spf13/cobra"
)

// NewBuildKernelCmd creates the kernel build command
func NewBuildKernelCmd() *cobra.Command {
	var (
		buildArch              string
		buildVersion           string
		buildVerificationLevel string
		buildConfig            string
		buildForceRebuild      bool
	)

	cmd := &cobra.Command{
		Use:   "build [version]",
		Short: "Build kernel from source",
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
				callbacks := ui.BuildKernelCallbacks{
					BuildFn: func(opts kernel.BuildOptions) error {
						return kernel.Build(opts, config.GlobalPaths)
					},
					CheckCachedFn: func(v string) (bool, string, error) {
						return kernel.CheckCachedBuild(v, buildArch, config.GlobalPaths)
					},
					ReadStatsFn: func(path string) (kernel.BuildStats, error) {
						return kernel.ReadBuildStats(path)
					},
					CheckInstalledFn: func(stats kernel.BuildStats) (bool, string, error) {
						return kernel.CheckKernelInstalled(stats, config.GlobalPaths)
					},
					InstallFn: func(stats kernel.BuildStats, setAsDefault bool) (string, error) {
						return kernel.InstallBuiltKernel(stats, setAsDefault, config.GlobalPaths)
					},
					ArchiveFn: func(stats kernel.BuildStats, archiveDir string) error {
						return kernel.ArchiveInstalledKernel(stats, archiveDir)
					},
					ClearBuildCacheFn: func() error {
						buildDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "build")
						artifactsDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "artifacts")
						if err := os.RemoveAll(buildDir); err != nil {
							return fmt.Errorf("failed to clear build directory: %w", err)
						}
						if err := os.RemoveAll(artifactsDir); err != nil {
							return fmt.Errorf("failed to clear artifacts directory: %w", err)
						}
						return nil
					},
					GetArchiveLocationFn: func() string {
						return config.GetKernelsArchiveLocation()
					},
				}
				err := ui.RunBuildKernelWizard(config.CurrentTheme, callbacks, buildArch, buildVerificationLevel, buildConfig, buildForceRebuild)
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
				hasCached, _, err := kernel.CheckCachedBuild(version, buildArch, config.GlobalPaths)
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
			}

			if err := kernel.Build(opts, config.GlobalPaths); err != nil {
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
