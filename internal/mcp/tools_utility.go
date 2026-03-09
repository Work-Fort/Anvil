// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Work-Fort/Anvil/pkg/config"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerUtilityTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("check_build_tools",
		gomcp.WithDescription("Verify required build tools are installed for kernel compilation"),
		gomcp.WithString("arch", gomcp.Description("Target architecture: x86_64 or aarch64 (default: host arch)")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleCheckBuildTools)

	s.AddTool(gomcp.NewTool("clean_build",
		gomcp.WithDescription("Clean kernel build cache. CLI: anvil clean build"),
		gomcp.WithBoolean("all", gomcp.Description("Remove entire build cache (default: false)")),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleCleanBuildCache)

	s.AddTool(gomcp.NewTool("clean_kernel",
		gomcp.WithDescription("Remove installed kernel versions. CLI: anvil clean kernel"),
		gomcp.WithBoolean("all", gomcp.Description("Remove ALL kernel data including default (default: false, removes only non-default)")),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleCleanKernel)

	s.AddTool(gomcp.NewTool("clean_firecracker",
		gomcp.WithDescription("Remove installed Firecracker versions. CLI: anvil clean firecracker"),
		gomcp.WithBoolean("all", gomcp.Description("Remove ALL Firecracker data including default (default: false, removes only non-default)")),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleCleanFirecracker)

	s.AddTool(gomcp.NewTool("clean_rootfs",
		gomcp.WithDescription("Remove rootfs images. CLI: anvil clean rootfs"),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleCleanRootfs)
}

func handleCheckBuildTools(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	arch := req.GetString("arch", runtime.GOARCH)
	if arch == "amd64" {
		arch = "x86_64"
	} else if arch == "arm64" {
		arch = "aarch64"
	}

	crossCompile := arch != runtime.GOARCH && !(arch == "x86_64" && runtime.GOARCH == "amd64") && !(arch == "aarch64" && runtime.GOARCH == "arm64")

	// Check required tools
	tools := []string{"make", "gcc", "flex", "bison", "bc", "perl", "xz"}
	if crossCompile {
		if arch == "aarch64" {
			tools = append(tools, "aarch64-linux-gnu-gcc")
		} else if arch == "x86_64" {
			tools = append(tools, "x86_64-linux-gnu-gcc")
		}
	}

	var missing []string
	var found []string
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		} else {
			found = append(found, tool)
		}
	}

	return jsonResult(map[string]any{
		"arch":          arch,
		"cross_compile": crossCompile,
		"found":         found,
		"missing":       missing,
		"ready":         len(missing) == 0,
	})
}

func handleCleanBuildCache(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	all := req.GetBool("all", false)

	buildDir := config.GlobalPaths.KernelBuildDir
	if _, err := os.Stat(buildDir); err != nil {
		if os.IsNotExist(err) {
			return jsonResult(map[string]any{"status": "clean", "message": "build cache already empty"})
		}
		return errResult(err)
	}

	if all {
		if err := os.RemoveAll(buildDir); err != nil {
			return errResult(fmt.Errorf("failed to clean build cache: %w", err))
		}
		return jsonResult(map[string]any{"status": "cleaned", "path": buildDir})
	}

	// Remove only the build subdirectory, keep source cache
	buildSubdir := filepath.Join(buildDir, "build")
	if err := os.RemoveAll(buildSubdir); err != nil {
		return errResult(fmt.Errorf("failed to clean build output: %w", err))
	}

	return jsonResult(map[string]any{"status": "cleaned", "path": buildSubdir})
}

func handleCleanKernel(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	all := req.GetBool("all", false)
	kernelsDir := config.GlobalPaths.KernelsDir

	if all {
		// Remove entire kernels directory
		if err := os.RemoveAll(kernelsDir); err != nil && !os.IsNotExist(err) {
			return errResult(fmt.Errorf("failed to remove kernels: %w", err))
		}

		removed := []string{"All kernels"}

		// Remove kernel symlink
		kernelName, err := config.GetKernelName()
		if err == nil {
			symlinkPath := filepath.Join(config.GlobalPaths.DataDir, kernelName)
			os.Remove(symlinkPath)
			removed = append(removed, "Kernel symlink")
		}

		return jsonResult(map[string]any{
			"status":  "cleaned",
			"removed": removed,
			"count":   len(removed),
		})
	}

	// Remove only non-default kernel versions
	// Determine the default kernel version from the data dir symlink (matches CLI behavior)
	defaultKernelVersion := ""
	kernelName, err := config.GetKernelName()
	if err == nil {
		kernelSymlink := filepath.Join(config.GlobalPaths.DataDir, kernelName)
		if target, linkErr := os.Readlink(kernelSymlink); linkErr == nil {
			parts := strings.Split(target, "/")
			for i, part := range parts {
				if part == "kernels" && i+1 < len(parts) {
					defaultKernelVersion = parts[i+1]
					break
				}
			}
		}
	}

	entries, err := os.ReadDir(kernelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return jsonResult(map[string]any{"removed": []string{}, "count": 0})
		}
		return errResult(fmt.Errorf("failed to read kernels directory: %w", err))
	}

	var removed []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "default" {
			continue
		}

		version := entry.Name()
		if version == defaultKernelVersion {
			continue
		}

		path := filepath.Join(kernelsDir, version)
		if err := os.RemoveAll(path); err != nil {
			return errResult(fmt.Errorf("failed to remove %s: %w", path, err))
		}
		removed = append(removed, version)
	}

	if removed == nil {
		removed = []string{}
	}

	return jsonResult(map[string]any{
		"removed": removed,
		"count":   len(removed),
	})
}

func handleCleanFirecracker(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	all := req.GetBool("all", false)
	fcDir := config.GlobalPaths.FirecrackerDir

	if all {
		// Remove entire firecracker directory
		if err := os.RemoveAll(fcDir); err != nil && !os.IsNotExist(err) {
			return errResult(fmt.Errorf("failed to remove Firecracker: %w", err))
		}

		removed := []string{"All Firecracker versions"}

		// Remove firecracker symlink in bin
		symlinkPath := filepath.Join(config.GlobalPaths.BinDir, "firecracker")
		os.Remove(symlinkPath)
		removed = append(removed, "Firecracker symlink")

		return jsonResult(map[string]any{
			"status":  "cleaned",
			"removed": removed,
			"count":   len(removed),
		})
	}

	// Remove only non-default Firecracker versions
	// Determine the default version from the bin dir symlink (matches CLI behavior)
	defaultFCVersion := ""
	fcSymlink := filepath.Join(config.GlobalPaths.BinDir, "firecracker")
	if target, linkErr := os.Readlink(fcSymlink); linkErr == nil {
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "firecracker" && i+1 < len(parts) {
				defaultFCVersion = parts[i+1]
				break
			}
		}
	}

	entries, err := os.ReadDir(fcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return jsonResult(map[string]any{"removed": []string{}, "count": 0})
		}
		return errResult(fmt.Errorf("failed to read Firecracker directory: %w", err))
	}

	var removed []string
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "default" {
			continue
		}

		version := entry.Name()
		if version == defaultFCVersion {
			continue
		}

		path := filepath.Join(fcDir, version)
		if err := os.RemoveAll(path); err != nil {
			return errResult(fmt.Errorf("failed to remove %s: %w", path, err))
		}
		removed = append(removed, version)
	}

	if removed == nil {
		removed = []string{}
	}

	return jsonResult(map[string]any{
		"removed": removed,
		"count":   len(removed),
	})
}

func handleCleanRootfs(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	dataDir := config.GlobalPaths.DataDir

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return jsonResult(map[string]any{"removed": []string{}, "count": 0})
		}
		return errResult(fmt.Errorf("failed to read data directory: %w", err))
	}

	var removed []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ext4") {
			continue
		}

		path := filepath.Join(dataDir, entry.Name())
		if err := os.Remove(path); err != nil {
			return errResult(fmt.Errorf("failed to remove %s: %w", path, err))
		}
		removed = append(removed, entry.Name())
	}

	if removed == nil {
		removed = []string{}
	}

	return jsonResult(map[string]any{
		"removed": removed,
		"count":   len(removed),
	})
}
