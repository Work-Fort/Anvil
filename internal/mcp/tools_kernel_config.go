// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/Work-Fort/Anvil/pkg/kconfig"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerKernelConfigTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("kernel_config_get",
		gomcp.WithDescription("Get the value of a CONFIG_* option from a kernel .config file"),
		gomcp.WithString("config_file", gomcp.Required(), gomcp.Description("Path to .config file")),
		gomcp.WithString("option", gomcp.Required(), gomcp.Description("Option name (with or without CONFIG_ prefix)")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleKernelConfigGet)

	s.AddTool(gomcp.NewTool("kernel_config_set",
		gomcp.WithDescription("Set a CONFIG_* option in a kernel .config file. Use y to enable, n to disable, or a string/int value."),
		gomcp.WithString("config_file", gomcp.Required(), gomcp.Description("Path to .config file")),
		gomcp.WithString("option", gomcp.Required(), gomcp.Description("Option name (with or without CONFIG_ prefix)")),
		gomcp.WithString("value", gomcp.Required(), gomcp.Description("Value: y, n, or a string/int value. Module (m) is not supported.")),
	), handleKernelConfigSet)

	s.AddTool(gomcp.NewTool("kernel_config_list",
		gomcp.WithDescription("List CONFIG_* options from a kernel .config file, with optional filter"),
		gomcp.WithString("config_file", gomcp.Required(), gomcp.Description("Path to .config file")),
		gomcp.WithString("filter", gomcp.Description("Substring filter on option names (case-insensitive)")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleKernelConfigList)

	s.AddTool(gomcp.NewTool("kernel_config_diff",
		gomcp.WithDescription("Compare two kernel .config files and show differences"),
		gomcp.WithString("file_a", gomcp.Required(), gomcp.Description("Path to first .config file")),
		gomcp.WithString("file_b", gomcp.Required(), gomcp.Description("Path to second .config file")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleKernelConfigDiff)
}

func normalizeOptionName(name string) string {
	return strings.TrimPrefix(name, "CONFIG_")
}

func handleKernelConfigGet(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := req.RequireString("config_file")
	if err != nil {
		return errResult(err)
	}
	option, err := req.RequireString("option")
	if err != nil {
		return errResult(err)
	}

	cfg, err := kconfig.ParseFile(file)
	if err != nil {
		return errResult(err)
	}

	name := normalizeOptionName(option)
	val, found := cfg.Get(name)
	if !found {
		return jsonResult(map[string]any{
			"option": "CONFIG_" + name,
			"found":  false,
		})
	}

	return jsonResult(map[string]any{
		"option": "CONFIG_" + name,
		"value":  val,
		"found":  true,
	})
}

func handleKernelConfigSet(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := req.RequireString("config_file")
	if err != nil {
		return errResult(err)
	}
	option, err := req.RequireString("option")
	if err != nil {
		return errResult(err)
	}
	value, err := req.RequireString("value")
	if err != nil {
		return errResult(err)
	}

	cfg, err := kconfig.ParseFile(file)
	if err != nil {
		return errResult(err)
	}

	name := normalizeOptionName(option)
	if err := cfg.Set(name, value); err != nil {
		return errResult(err)
	}

	if err := cfg.WriteFile(file); err != nil {
		return errResult(fmt.Errorf("failed to write config: %w", err))
	}

	return jsonResult(map[string]any{
		"option": "CONFIG_" + name,
		"value":  value,
		"status": "updated",
	})
}

func handleKernelConfigList(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	file, err := req.RequireString("config_file")
	if err != nil {
		return errResult(err)
	}
	filter := req.GetString("filter", "")

	cfg, err := kconfig.ParseFile(file)
	if err != nil {
		return errResult(err)
	}

	options := cfg.List(filter)

	items := make([]map[string]string, len(options))
	for i, opt := range options {
		items[i] = map[string]string{
			"option": "CONFIG_" + opt.Name,
			"value":  opt.Value,
		}
	}

	return jsonResult(map[string]any{
		"count":   len(items),
		"options": items,
	})
}

func handleKernelConfigDiff(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	fileA, err := req.RequireString("file_a")
	if err != nil {
		return errResult(err)
	}
	fileB, err := req.RequireString("file_b")
	if err != nil {
		return errResult(err)
	}

	cfgA, err := kconfig.ParseFile(fileA)
	if err != nil {
		return errResult(fmt.Errorf("file_a: %w", err))
	}
	cfgB, err := kconfig.ParseFile(fileB)
	if err != nil {
		return errResult(fmt.Errorf("file_b: %w", err))
	}

	entries := kconfig.Diff(cfgA, cfgB)

	items := make([]map[string]string, len(entries))
	for i, e := range entries {
		items[i] = map[string]string{
			"option":   "CONFIG_" + e.Name,
			"type":     string(e.Type),
			"value_a":  e.ValueA,
			"value_b":  e.ValueB,
		}
	}

	return jsonResult(map[string]any{
		"count":   len(items),
		"changes": items,
	})
}
