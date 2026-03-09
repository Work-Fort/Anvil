// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"os/exec"
	"runtime"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	"github.com/Work-Fort/Anvil/pkg/rootfs"
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

	status, path, err := kernel.CleanBuildCache(all, config.GlobalPaths)
	if err != nil {
		return errResult(err)
	}

	if status == "clean" {
		return jsonResult(map[string]any{"status": "clean", "message": "build cache already empty"})
	}

	return jsonResult(map[string]any{"status": status, "path": path})
}

func handleCleanKernel(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	all := req.GetBool("all", false)

	removed, err := kernel.Clean(!all, config.GlobalPaths)
	if err != nil {
		return errResult(err)
	}

	if all {
		return jsonResult(map[string]any{
			"status":  "cleaned",
			"removed": removed,
			"count":   len(removed),
		})
	}

	return jsonResult(map[string]any{
		"removed": removed,
		"count":   len(removed),
	})
}

func handleCleanFirecracker(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	all := req.GetBool("all", false)

	removed, err := firecracker.Clean(!all, config.GlobalPaths)
	if err != nil {
		return errResult(err)
	}

	if all {
		return jsonResult(map[string]any{
			"status":  "cleaned",
			"removed": removed,
			"count":   len(removed),
		})
	}

	return jsonResult(map[string]any{
		"removed": removed,
		"count":   len(removed),
	})
}

func handleCleanRootfs(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	removed, err := rootfs.Clean(config.GlobalPaths.DataDir)
	if err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"removed": removed,
		"count":   len(removed),
	})
}
