// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"

	"github.com/Work-Fort/Anvil/pkg/config"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerConfigTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("config_get",
		gomcp.WithDescription("Get an anvil config value (from anvil.yaml or user config). CLI: anvil config get"),
		gomcp.WithString("key", gomcp.Required(), gomcp.Description("Config key (e.g. signing.key.name)")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleConfigGet)

	s.AddTool(gomcp.NewTool("config_set",
		gomcp.WithDescription("Set an anvil config value. CLI: anvil config set"),
		gomcp.WithString("key", gomcp.Required(), gomcp.Description("Config key")),
		gomcp.WithString("value", gomcp.Required(), gomcp.Description("Value to set")),
	), handleConfigSet)

	s.AddTool(gomcp.NewTool("config_list",
		gomcp.WithDescription("List all anvil config values with their sources. CLI: anvil config list"),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleConfigList)

	s.AddTool(gomcp.NewTool("config_get_paths",
		gomcp.WithDescription("Get all resolved directory paths for the current context"),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleConfigGetPaths)
}

func handleConfigGet(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return errResult(err)
	}

	// Re-read config from disk to pick up external changes (e.g. CLI edits)
	if err := config.ReloadConfig(); err != nil {
		return errResult(err)
	}

	val, err := config.GetConfigValue(key)
	if err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"key":    val.Key,
		"value":  val.Value,
		"source": val.Source,
	})
}

func handleConfigSet(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	key, err := req.RequireString("key")
	if err != nil {
		return errResult(err)
	}
	value, err := req.RequireString("value")
	if err != nil {
		return errResult(err)
	}

	// Determine scope based on current mode
	scope := config.ScopeUser
	if config.IsRepoMode() && !config.IsUserMode() {
		scope = config.ScopeRepo
	}

	if err := config.SetConfigValue(key, value, scope); err != nil {
		return errResult(err)
	}

	scopeName := "user"
	if scope == config.ScopeRepo {
		scopeName = "repo"
	}

	return jsonResult(map[string]any{
		"key":   key,
		"value": value,
		"scope": scopeName,
	})
}

func handleConfigList(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	// Re-read config from disk to pick up external changes (e.g. CLI edits)
	if err := config.ReloadConfig(); err != nil {
		return errResult(err)
	}

	values, err := config.ListConfigValues()
	if err != nil {
		return errResult(err)
	}

	items := make([]map[string]any, len(values))
	for i, v := range values {
		items[i] = map[string]any{
			"key":    v.Key,
			"value":  v.Value,
			"source": v.Source,
		}
	}

	return jsonResult(items)
}

func handleConfigGetPaths(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	return jsonResult(map[string]any{
		"data_dir":         config.GlobalPaths.DataDir,
		"cache_dir":        config.GlobalPaths.CacheDir,
		"config_dir":       config.GlobalPaths.ConfigDir,
		"bin_dir":          config.GlobalPaths.BinDir,
		"kernels_dir":      config.GlobalPaths.KernelsDir,
		"firecracker_dir":  config.GlobalPaths.FirecrackerDir,
		"kernel_build_dir": config.GlobalPaths.KernelBuildDir,
		"keys_dir":         config.GlobalPaths.KeysDir,
	})
}
