// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

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

	s.AddTool(gomcp.NewTool("clean_build_cache",
		gomcp.WithDescription("Clean kernel build cache"),
		gomcp.WithBoolean("all", gomcp.Description("Remove entire build cache (default: false)")),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleCleanBuildCache)
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
