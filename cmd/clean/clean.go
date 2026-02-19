// SPDX-License-Identifier: Apache-2.0
package clean

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/ui"
	"github.com/spf13/cobra"
)

// NewCleanCmd creates the clean command and its subcommands
func NewCleanCmd() *cobra.Command {
	var (
		removeInactive bool
		allDangerous   bool
		force          bool
		cleanArch      string
	)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean anvil data",
		Long:  `Clean cache and optionally remove versions.`,
	}

	// Create subcommands
	kernelCmd := newKernelCmd(&removeInactive, &allDangerous, &force)
	firecrackerCmd := newFirecrackerCmd(&removeInactive, &allDangerous, &force)
	buildKernelCmd := newBuildKernelCmd(&cleanArch)
	rootfsCmd := newRootfsCmd()

	// Add flags to kernel subcommand
	kernelCmd.Flags().BoolVarP(&removeInactive, "remove-inactive", "i", false, "Remove all non-default kernel versions")
	kernelCmd.Flags().BoolVarP(&allDangerous, "all-dangerous", "a", false, "Remove all kernel data (requires confirmation)")
	kernelCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt (use with --all-dangerous)")

	// Add flags to firecracker subcommand
	firecrackerCmd.Flags().BoolVarP(&removeInactive, "remove-inactive", "i", false, "Remove all non-default Firecracker versions")
	firecrackerCmd.Flags().BoolVarP(&allDangerous, "all-dangerous", "a", false, "Remove all Firecracker data (requires confirmation)")
	firecrackerCmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt (use with --all-dangerous)")

	// Add flags to build-kernel subcommand
	buildKernelCmd.Flags().StringVarP(&cleanArch, "arch", "a", "all", "Architecture to clean: x86_64, aarch64, or all")

	// Add subcommands to clean
	cmd.AddCommand(kernelCmd)
	cmd.AddCommand(firecrackerCmd)
	cmd.AddCommand(buildKernelCmd)
	cmd.AddCommand(rootfsCmd)

	return cmd
}

func cleanCache() error {
	log.Debug("Cleaning cache directory")

	theme := config.CurrentTheme
	subtleStyle := theme.SubtleStyle()
	itemStyle := theme.ErrorStyle()

	if _, err := os.Stat(config.GlobalPaths.CacheDir); os.IsNotExist(err) {
		fmt.Println()
		fmt.Println(theme.InfoMessage("Cache directory doesn't exist"))
		return nil
	}

	// Get cache size before cleaning
	entries, err := os.ReadDir(config.GlobalPaths.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	// Remove temporary files in cache, but skip permanent subdirectories
	var removedItems []string
	for _, entry := range entries {
		// Skip build-kernel directory (it's a permanent cache subdirectory, not temporary)
		if entry.Name() == "build-kernel" && entry.IsDir() {
			continue
		}

		path := filepath.Join(config.GlobalPaths.CacheDir, entry.Name())
		log.Debugf("Removing %s", entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("failed to remove %s: %w", path, err)
		}
		removedItems = append(removedItems, entry.Name())
	}

	fmt.Println()

	if len(removedItems) == 0 {
		fmt.Println(theme.SuccessMessage("Cache empty"))
	} else {
		fmt.Println(theme.SuccessMessage("Cache cleaned"))
		fmt.Println()
		for _, item := range removedItems {
			fmt.Println(subtleStyle.Render("  • ") + itemStyle.Render(item))
		}
	}

	return nil
}

func cleanInactiveKernels() error {
	theme := config.CurrentTheme
	confirmed, err := ui.Confirm(theme.WarningIndicator() + "  This will remove all non-default kernel versions. Continue?")
	if err != nil {
		return err
	}

	if !confirmed {
		return fmt.Errorf("operation cancelled")
	}
	subtleStyle := theme.SubtleStyle()
	itemStyle := theme.ErrorStyle()

	kernelName, err := config.GetKernelName()
	if err != nil {
		return err
	}

	removedCount := 0
	var removedItems []string

	// Get default kernel version
	kernelSymlink := filepath.Join(config.GlobalPaths.DataDir, kernelName)
	defaultKernelVersion := ""
	if target, err := os.Readlink(kernelSymlink); err == nil {
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "kernels" && i+1 < len(parts) {
				defaultKernelVersion = parts[i+1]
				break
			}
		}
	}

	// Remove non-default kernels
	entries, err := os.ReadDir(config.GlobalPaths.KernelsDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			version := entry.Name()
			if version != defaultKernelVersion {
				log.Debugf("Removing kernel %s", version)
				path := filepath.Join(config.GlobalPaths.KernelsDir, version)
				if err := os.RemoveAll(path); err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
				}
				removedItems = append(removedItems, fmt.Sprintf("kernel %s", version))
				removedCount++
			}
		}
	}

	fmt.Println()

	if removedCount == 0 {
		fmt.Println(theme.InfoMessage("No inactive kernel versions to remove"))
	} else {
		fmt.Println(theme.SuccessMessage(fmt.Sprintf("Removed %d inactive kernel version(s)", removedCount)))
		fmt.Println()
		for _, item := range removedItems {
			fmt.Println(subtleStyle.Render("  • ") + itemStyle.Render(item))
		}
	}

	fmt.Println()

	// Also clean cache
	return cleanCache()
}

func cleanInactiveFirecracker() error {
	theme := config.CurrentTheme
	confirmed, err := ui.Confirm(theme.WarningIndicator() + "  This will remove all non-default Firecracker versions. Continue?")
	if err != nil {
		return err
	}

	if !confirmed {
		return fmt.Errorf("operation cancelled")
	}
	subtleStyle := theme.SubtleStyle()
	itemStyle := theme.ErrorStyle()

	removedCount := 0
	var removedItems []string

	// Get default Firecracker version
	fcSymlink := filepath.Join(config.GlobalPaths.BinDir, "firecracker")
	defaultFCVersion := ""
	if target, err := os.Readlink(fcSymlink); err == nil {
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "firecracker" && i+1 < len(parts) {
				defaultFCVersion = parts[i+1]
				break
			}
		}
	}

	// Remove non-default Firecracker versions
	entries, err := os.ReadDir(config.GlobalPaths.FirecrackerDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			version := entry.Name()
			if version != defaultFCVersion {
				log.Debugf("Removing Firecracker %s", version)
				path := filepath.Join(config.GlobalPaths.FirecrackerDir, version)
				if err := os.RemoveAll(path); err != nil {
					return fmt.Errorf("failed to remove %s: %w", path, err)
				}
				removedItems = append(removedItems, fmt.Sprintf("firecracker %s", version))
				removedCount++
			}
		}
	}

	fmt.Println()

	if removedCount == 0 {
		fmt.Println(theme.InfoMessage("No inactive Firecracker versions to remove"))
	} else {
		fmt.Println(theme.SuccessMessage(fmt.Sprintf("Removed %d inactive Firecracker version(s)", removedCount)))
		fmt.Println()
		for _, item := range removedItems {
			fmt.Println(subtleStyle.Render("  • ") + itemStyle.Render(item))
		}
	}

	fmt.Println()

	// Also clean cache
	return cleanCache()
}

func cleanAllKernels(skipConfirm bool) error {
	theme := config.CurrentTheme
	if !skipConfirm {
		prompt := theme.WarningIndicator() + `  DANGER: This will remove ALL kernel data

This includes:
  • All kernel versions
  • Cache directory

Type 'DELETE' to confirm:`

		confirmed, err := ui.TypedConfirm(prompt, "DELETE")
		if err != nil {
			return err
		}

		if !confirmed {
			return fmt.Errorf("operation cancelled")
		}
	}

	log.Debug("Removing all kernel data")
	subtleStyle := theme.SubtleStyle()
	itemStyle := theme.ErrorStyle()

	var removedItems []string

	// Remove kernels directory
	if err := os.RemoveAll(config.GlobalPaths.KernelsDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove kernels: %w", err)
	}
	removedItems = append(removedItems, "All kernels")
	log.Debugf("Removed %s", config.GlobalPaths.KernelsDir)

	// Remove kernel symlink in bin
	kernelName, err := config.GetKernelName()
	if err == nil {
		symlinkPath := filepath.Join(config.GlobalPaths.DataDir, kernelName)
		os.Remove(symlinkPath)
		removedItems = append(removedItems, "Kernel symlink")
	}

	fmt.Println()
	fmt.Println(theme.SuccessMessage("All kernel data removed"))
	fmt.Println()
	for _, item := range removedItems {
		fmt.Println(subtleStyle.Render("  • ") + itemStyle.Render(item))
	}
	fmt.Println()

	// Clean cache (this preserves build-kernel directory)
	return cleanCache()
}

func cleanAllFirecracker(skipConfirm bool) error {
	theme := config.CurrentTheme
	if !skipConfirm {
		prompt := theme.WarningIndicator() + `  DANGER: This will remove ALL Firecracker data

This includes:
  • All Firecracker versions
  • Cache directory

Type 'DELETE' to confirm:`

		confirmed, err := ui.TypedConfirm(prompt, "DELETE")
		if err != nil {
			return err
		}

		if !confirmed {
			return fmt.Errorf("operation cancelled")
		}
	}

	log.Debug("Removing all Firecracker data")

	subtleStyle := theme.SubtleStyle()
	itemStyle := theme.ErrorStyle()

	var removedItems []string

	// Remove firecracker directory
	if err := os.RemoveAll(config.GlobalPaths.FirecrackerDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove Firecracker: %w", err)
	}
	removedItems = append(removedItems, "All Firecracker versions")
	log.Debugf("Removed %s", config.GlobalPaths.FirecrackerDir)

	// Remove firecracker symlink in bin
	symlinkPath := filepath.Join(config.GlobalPaths.BinDir, "firecracker")
	os.Remove(symlinkPath)
	removedItems = append(removedItems, "Firecracker symlink")

	fmt.Println()
	fmt.Println(theme.SuccessMessage("All Firecracker data removed"))
	fmt.Println()
	for _, item := range removedItems {
		fmt.Println(subtleStyle.Render("  • ") + itemStyle.Render(item))
	}
	fmt.Println()

	// Clean cache (this preserves build-kernel directory)
	return cleanCache()
}

func cleanBuildKernel(arch string) error {
	theme := config.CurrentTheme
	subtleStyle := theme.SubtleStyle()
	itemStyle := theme.ErrorStyle()

	// Validate architecture
	if arch != "x86_64" && arch != "aarch64" && arch != "all" {
		return fmt.Errorf("invalid architecture: %s (must be x86_64, aarch64, or all)", arch)
	}

	var removedItems []string
	removedCount := 0

	// Use XDG KernelBuildDir for kernel build artifacts
	// The build script creates build/ and artifacts/ inside KernelBuildDir
	buildDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "build")
	artifactsDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "artifacts")

	if arch == "all" {
		// Remove entire build and artifacts directories
		if _, err := os.Stat(buildDir); err == nil {
			log.Debugf("Removing build directory: %s", buildDir)
			if err := os.RemoveAll(buildDir); err != nil {
				return fmt.Errorf("failed to remove build directory: %w", err)
			}
			removedItems = append(removedItems, "Kernel source (build/)")
			removedCount++
		}

		if _, err := os.Stat(artifactsDir); err == nil {
			log.Debugf("Removing artifacts directory: %s", artifactsDir)
			if err := os.RemoveAll(artifactsDir); err != nil {
				return fmt.Errorf("failed to remove artifacts directory: %w", err)
			}
			removedItems = append(removedItems, "Build artifacts (artifacts/)")
			removedCount++
		}
	} else {
		// Remove only architecture-specific artifacts
		if _, err := os.Stat(artifactsDir); err == nil {
			entries, err := os.ReadDir(artifactsDir)
			if err != nil {
				return fmt.Errorf("failed to read artifacts directory: %w", err)
			}

			for _, entry := range entries {
				// Match files containing the architecture (e.g., vmlinux-6.1-x86_64)
				if strings.Contains(entry.Name(), arch) {
					path := filepath.Join(artifactsDir, entry.Name())
					log.Debugf("Removing %s artifact: %s", arch, entry.Name())
					if err := os.Remove(path); err != nil {
						return fmt.Errorf("failed to remove %s: %w", path, err)
					}
					removedItems = append(removedItems, entry.Name())
					removedCount++
				}
			}
		}
	}

	fmt.Println()

	if removedCount == 0 {
		if arch == "all" {
			fmt.Println(theme.SuccessMessage("No build artifacts"))
		} else {
			fmt.Println(theme.SuccessMessage(fmt.Sprintf("No build artifacts (%s)", arch)))
		}
	} else {
		if arch == "all" {
			fmt.Println(theme.SuccessMessage("Build artifacts cleaned"))
		} else {
			fmt.Println(theme.SuccessMessage(fmt.Sprintf("Build artifacts cleaned (%s)", arch)))
		}
		fmt.Println()
		for _, item := range removedItems {
			fmt.Println(subtleStyle.Render("  • ") + itemStyle.Render(item))
		}
	}

	return nil
}

func cleanRootfs() error {
	theme := config.CurrentTheme
	subtleStyle := theme.SubtleStyle()
	itemStyle := theme.ErrorStyle()

	var removedItems []string
	removedCount := 0

	// Look for rootfs files in data directory (*.ext4 files)
	entries, err := os.ReadDir(config.GlobalPaths.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println()
			fmt.Println(theme.InfoMessage("No rootfs images found"))
			return nil
		}
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	for _, entry := range entries {
		// Match .ext4 files (rootfs images)
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".ext4") {
			path := filepath.Join(config.GlobalPaths.DataDir, entry.Name())
			log.Debugf("Removing rootfs: %s", entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove %s: %w", path, err)
			}
			removedItems = append(removedItems, entry.Name())
			removedCount++
		}
	}

	fmt.Println()

	if removedCount == 0 {
		fmt.Println(theme.InfoMessage("No rootfs images found"))
	} else {
		fmt.Println(theme.SuccessMessage(fmt.Sprintf("Removed %d rootfs image(s)", removedCount)))
		fmt.Println()
		for _, item := range removedItems {
			fmt.Println(subtleStyle.Render("  • ") + itemStyle.Render(item))
		}
	}

	return nil
}
