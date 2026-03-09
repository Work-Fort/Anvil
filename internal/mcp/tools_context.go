// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Work-Fort/Anvil/pkg/config"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerContextTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("get_context",
		gomcp.WithDescription("Get current mode (user/repo), resolved paths, and active config"),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleGetContext)

	s.AddTool(gomcp.NewTool("set_repo_root",
		gomcp.WithDescription("Switch to repo mode by setting the repo root path (must contain anvil.yaml)"),
		gomcp.WithString("path", gomcp.Required(), gomcp.Description("Path to directory containing anvil.yaml")),
	), handleSetRepoRoot)

	s.AddTool(gomcp.NewTool("set_user_mode",
		gomcp.WithDescription("Switch to user mode (XDG paths), ignoring any anvil.yaml"),
	), handleSetUserMode)
}

func handleGetContext(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	return jsonResult(buildContextResponse())
}

func handleSetRepoRoot(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return errResult(err)
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return errResult(fmt.Errorf("invalid path: %w", err))
	}

	// Check directory exists
	info, err := os.Stat(absPath)
	if err != nil {
		return errResult(fmt.Errorf("path does not exist: %s", absPath))
	}
	if !info.IsDir() {
		return errResult(fmt.Errorf("path is not a directory: %s", absPath))
	}

	// Check anvil.yaml exists
	anvilYaml := filepath.Join(absPath, config.LocalConfigFile+config.DefaultConfigExt)
	if _, err := os.Stat(anvilYaml); err != nil {
		return errResult(fmt.Errorf("no anvil.yaml found at %s", absPath))
	}

	// Switch to repo mode
	if err := os.Chdir(absPath); err != nil {
		return errResult(fmt.Errorf("failed to chdir: %w", err))
	}

	config.SetUserModeOverride(false)

	if err := config.LoadConfig(); err != nil {
		return errResult(fmt.Errorf("failed to reload config: %w", err))
	}

	return jsonResult(buildContextResponse())
}

func handleSetUserMode(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	config.SetUserModeOverride(true)

	if err := config.LoadConfig(); err != nil {
		return errResult(fmt.Errorf("failed to reload config: %w", err))
	}

	return jsonResult(buildContextResponse())
}

func buildContextResponse() map[string]any {
	mode := "user"
	if config.IsRepoMode() && !config.IsUserMode() {
		mode = "repo"
	}

	result := map[string]any{
		"mode": mode,
		"paths": map[string]any{
			"data_dir":         config.GlobalPaths.DataDir,
			"cache_dir":        config.GlobalPaths.CacheDir,
			"config_dir":       config.GlobalPaths.ConfigDir,
			"kernels_dir":      config.GlobalPaths.KernelsDir,
			"firecracker_dir":  config.GlobalPaths.FirecrackerDir,
			"kernel_build_dir": config.GlobalPaths.KernelBuildDir,
			"keys_dir":         config.GlobalPaths.KeysDir,
		},
	}

	cwd, _ := os.Getwd()
	result["cwd"] = cwd

	return result
}
