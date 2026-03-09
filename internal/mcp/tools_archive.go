// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/kernel"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerArchiveTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("archive_kernel",
		gomcp.WithDescription("Archive a built kernel to the repo archive directory"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Kernel version to archive")),
		gomcp.WithString("arch", gomcp.Required(), gomcp.Description("Architecture: x86_64 or aarch64")),
	), handleArchiveKernel)

	s.AddTool(gomcp.NewTool("archive_list",
		gomcp.WithDescription("List archived kernels from the archive index"),
		gomcp.WithString("arch", gomcp.Description("Filter by architecture")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleArchiveList)

	s.AddTool(gomcp.NewTool("archive_get",
		gomcp.WithDescription("Get details of an archived kernel"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Kernel version")),
		gomcp.WithString("arch", gomcp.Required(), gomcp.Description("Architecture: x86_64 or aarch64")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleArchiveGet)
}

func getArchiveDir() (string, error) {
	dir := config.GetKernelsArchiveLocation()
	if dir == "" {
		return "", fmt.Errorf("kernels.archive.location not configured")
	}
	return dir, nil
}

func handleArchiveKernel(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}
	arch, err := req.RequireString("arch")
	if err != nil {
		return errResult(err)
	}

	// Read build stats to get artifact paths
	buildDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "build")
	statsFile := filepath.Join(buildDir, "build-stats.json")
	stats, err := kernel.ReadBuildStats(statsFile)
	if err != nil {
		return errResult(fmt.Errorf("no build stats found — build kernel %s for %s first: %w", version, arch, err))
	}

	if stats.KernelVersion != version {
		return errResult(fmt.Errorf("cached build is for %s, not %s", stats.KernelVersion, version))
	}

	archiveDir, err := getArchiveDir()
	if err != nil {
		return errResult(err)
	}
	if err := kernel.ArchiveInstalledKernel(stats, archiveDir); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"version":     version,
		"arch":        arch,
		"archive_dir": archiveDir,
		"status":      "archived",
	})
}

func handleArchiveList(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	archFilter := req.GetString("arch", "")

	archiveDir, err := getArchiveDir()
	if err != nil {
		return jsonResult(map[string]any{"archives": []any{}, "count": 0, "message": err.Error()})
	}

	indexPath := filepath.Join(archiveDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return jsonResult(map[string]any{"archives": []any{}, "count": 0})
		}
		return errResult(err)
	}

	var index map[string]map[string]string
	if err := json.Unmarshal(data, &index); err != nil {
		return errResult(fmt.Errorf("failed to parse archive index: %w", err))
	}

	var archives []map[string]string
	for arch, versions := range index {
		if archFilter != "" && arch != archFilter {
			continue
		}
		for version, path := range versions {
			archives = append(archives, map[string]string{
				"arch":    arch,
				"version": version,
				"path":    path,
			})
		}
	}

	if archives == nil {
		archives = []map[string]string{}
	}

	return jsonResult(map[string]any{
		"archives": archives,
		"count":    len(archives),
	})
}

func handleArchiveGet(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}
	arch, err := req.RequireString("arch")
	if err != nil {
		return errResult(err)
	}

	archiveDir, err := getArchiveDir()
	if err != nil {
		return errResult(err)
	}

	indexPath := filepath.Join(archiveDir, "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return errResult(fmt.Errorf("archive index not found: %w", err))
	}

	var index map[string]map[string]string
	if err := json.Unmarshal(data, &index); err != nil {
		return errResult(fmt.Errorf("failed to parse archive index: %w", err))
	}

	archVersions, ok := index[arch]
	if !ok {
		return errResult(fmt.Errorf("no archives for architecture %s", arch))
	}

	kernelPath, ok := archVersions[version]
	if !ok {
		return errResult(fmt.Errorf("kernel %s not found in %s archive", version, arch))
	}

	fullPath := filepath.Join(archiveDir, kernelPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return errResult(fmt.Errorf("archived file missing: %w", err))
	}

	return jsonResult(map[string]any{
		"version":   version,
		"arch":      arch,
		"path":      kernelPath,
		"full_path": fullPath,
		"size":      info.Size(),
	})
}
