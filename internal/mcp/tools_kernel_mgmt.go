// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerKernelMgmtTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("kernel_list",
		gomcp.WithDescription("List installed kernel versions"),
		gomcp.WithString("arch", gomcp.Description("Filter by architecture: x86_64 or aarch64")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleKernelList)

	s.AddTool(gomcp.NewTool("kernel_get",
		gomcp.WithDescription("Get details of an installed kernel version"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Kernel version")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleKernelGet)

	s.AddTool(gomcp.NewTool("kernel_set_default",
		gomcp.WithDescription("Set the default kernel version"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Version to set as default")),
	), handleKernelSetDefault)

	s.AddTool(gomcp.NewTool("kernel_remove",
		gomcp.WithDescription("Remove an installed kernel version"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Version to remove")),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleKernelRemove)

	s.AddTool(gomcp.NewTool("kernel_install",
		gomcp.WithDescription("Install a built kernel from the build cache"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Version to install")),
		gomcp.WithString("arch", gomcp.Required(), gomcp.Description("Architecture: x86_64 or aarch64")),
		gomcp.WithBoolean("set_default", gomcp.Description("Set as default after install (default: true)")),
	), handleKernelInstall)
}

func handleKernelList(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	archFilter := req.GetString("arch", "")

	kernelsDir := config.GlobalPaths.KernelsDir
	entries, err := os.ReadDir(kernelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return jsonResult(map[string]any{"kernels": []any{}, "count": 0})
		}
		return errResult(err)
	}

	// Check for default symlink
	defaultTarget := ""
	defaultLink := filepath.Join(kernelsDir, "default")
	if target, err := os.Readlink(defaultLink); err == nil {
		defaultTarget = filepath.Base(target)
	}

	var kernels []map[string]any
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "default" {
			continue
		}

		name := entry.Name()
		if archFilter != "" && !strings.Contains(name, archFilter) {
			continue
		}

		isDefault := name == defaultTarget
		kernels = append(kernels, map[string]any{
			"version":    name,
			"is_default": isDefault,
			"path":       filepath.Join(kernelsDir, name),
		})
	}

	if kernels == nil {
		kernels = []map[string]any{}
	}

	return jsonResult(map[string]any{
		"kernels": kernels,
		"count":   len(kernels),
	})
}

func handleKernelGet(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}

	kernelDir := filepath.Join(config.GlobalPaths.KernelsDir, version)
	if _, err := os.Stat(kernelDir); err != nil {
		return errResult(fmt.Errorf("kernel version %s not found", version))
	}

	// List files in the kernel directory
	entries, err := os.ReadDir(kernelDir)
	if err != nil {
		return errResult(err)
	}

	var files []string
	for _, e := range entries {
		files = append(files, e.Name())
	}

	// Check if this is the default
	defaultLink := filepath.Join(config.GlobalPaths.KernelsDir, "default")
	isDefault := false
	if target, err := os.Readlink(defaultLink); err == nil {
		isDefault = filepath.Base(target) == version
	}

	return jsonResult(map[string]any{
		"version":    version,
		"path":       kernelDir,
		"files":      files,
		"is_default": isDefault,
	})
}

func handleKernelSetDefault(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}

	if err := kernel.Set(version); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"version": version,
		"status":  "set as default",
	})
}

func handleKernelRemove(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}

	kernelDir := filepath.Join(config.GlobalPaths.KernelsDir, version)
	if _, err := os.Stat(kernelDir); err != nil {
		return errResult(fmt.Errorf("kernel version %s not found", version))
	}

	if err := os.RemoveAll(kernelDir); err != nil {
		return errResult(fmt.Errorf("failed to remove kernel: %w", err))
	}

	return jsonResult(map[string]any{
		"version": version,
		"status":  "removed",
	})
}

func handleKernelInstall(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}
	arch, err := req.RequireString("arch")
	if err != nil {
		return errResult(err)
	}

	setDefault := req.GetBool("set_default", true)

	// Read build stats from cache to get the BuildStats struct
	buildDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "build")
	statsFile := filepath.Join(buildDir, "build-stats.json")
	stats, err := kernel.ReadBuildStats(statsFile)
	if err != nil {
		return errResult(fmt.Errorf("no build stats found — build kernel %s for %s first: %w", version, arch, err))
	}

	if stats.KernelVersion != version {
		return errResult(fmt.Errorf("cached build is for %s, not %s", stats.KernelVersion, version))
	}

	installPath, err := kernel.InstallBuiltKernel(stats, setDefault)
	if err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"version":      version,
		"arch":         arch,
		"install_path": installPath,
		"set_default":  setDefault,
		"status":       "installed",
	})
}
