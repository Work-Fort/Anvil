// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"path/filepath"
	"time"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/firecracker"
	"github.com/Work-Fort/Anvil/pkg/rootfs"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerFirecrackerTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("firecracker_test",
		gomcp.WithDescription("Run Firecracker acceptance test: boot VM and test vsock communication. CLI: anvil firecracker test"),
		gomcp.WithString("kernel_version", gomcp.Description("Kernel version to test (default: default kernel)")),
		gomcp.WithString("rootfs", gomcp.Description("Path to rootfs image")),
		gomcp.WithNumber("boot_timeout_secs", gomcp.Description("Boot timeout in seconds (default: 10)")),
		gomcp.WithNumber("ping_timeout_secs", gomcp.Description("Vsock ping timeout in seconds (default: 10)")),
	), handleFirecrackerTest)

	s.AddTool(gomcp.NewTool("firecracker_list",
		gomcp.WithDescription("List installed Firecracker versions. CLI: anvil firecracker list"),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleFirecrackerList)

	s.AddTool(gomcp.NewTool("firecracker_get",
		gomcp.WithDescription("Download a Firecracker binary version. CLI: anvil firecracker get"),
		gomcp.WithString("version", gomcp.Description("Version to download (default: latest)")),
	), handleFirecrackerGet)

	s.AddTool(gomcp.NewTool("firecracker_set_default",
		gomcp.WithDescription("Set default Firecracker version. CLI: anvil firecracker set"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Version to set as default")),
	), handleFirecrackerSetDefault)

	s.AddTool(gomcp.NewTool("firecracker_remove",
		gomcp.WithDescription("Remove an installed Firecracker version. CLI: anvil firecracker remove"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Version to remove")),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleFirecrackerRemove)

	s.AddTool(gomcp.NewTool("firecracker_versions",
		gomcp.WithDescription("List available Firecracker versions from GitHub. CLI: anvil firecracker versions"),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleFirecrackerVersions)

	s.AddTool(gomcp.NewTool("firecracker_create_rootfs",
		gomcp.WithDescription("Create an Alpine Linux rootfs for Firecracker testing. CLI: anvil firecracker create-rootfs"),
		gomcp.WithString("output", gomcp.Description("Output file path")),
		gomcp.WithNumber("size_mb", gomcp.Description("Size in MB (default: 512)")),
		gomcp.WithBoolean("inject_binary", gomcp.Description("Inject anvil binary into rootfs")),
		gomcp.WithBoolean("force", gomcp.Description("Overwrite existing rootfs")),
	), handleFirecrackerCreateRootfs)
}

func handleFirecrackerTest(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	kernelVersion := req.GetString("kernel_version", "")
	rootfsPath := req.GetString("rootfs", "")
	bootTimeout := time.Duration(req.GetInt("boot_timeout_secs", 10)) * time.Second
	pingTimeout := time.Duration(req.GetInt("ping_timeout_secs", 10)) * time.Second

	opts := firecracker.TestOptions{
		KernelVersion: kernelVersion,
		RootfsPath:    rootfsPath,
		BootTimeout:   bootTimeout,
		PingTimeout:   pingTimeout,
	}

	result, err := firecracker.Test(opts, config.GlobalPaths)
	if err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"success":         result.Success,
		"kernel_version":  result.KernelVersion,
		"rootfs_path":     result.RootfsPath,
		"boot_time":       result.BootTime.String(),
		"ping_round_trip": result.PingRoundTrip.String(),
	})
}

func handleFirecrackerList(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	versions, err := firecracker.List(config.GlobalPaths)
	if err != nil {
		return errResult(err)
	}

	if versions == nil {
		versions = []firecracker.FirecrackerInfo{}
	}

	result := make([]map[string]any, len(versions))
	for i, v := range versions {
		result[i] = map[string]any{
			"version":    v.Version,
			"is_default": v.IsDefault,
			"path":       v.Path,
		}
	}

	return jsonResult(map[string]any{
		"versions": result,
		"count":    len(result),
	})
}

func handleFirecrackerGet(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version := req.GetString("version", "")
	if err := firecracker.Download(version, config.GlobalPaths); err != nil {
		return errResult(err)
	}
	return jsonResult(map[string]any{
		"version": version,
		"status":  "downloaded",
	})
}

func handleFirecrackerSetDefault(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}
	if err := firecracker.Set(version, config.GlobalPaths); err != nil {
		return errResult(err)
	}
	return jsonResult(map[string]any{
		"version": version,
		"status":  "set as default",
	})
}

func handleFirecrackerRemove(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}

	if err := firecracker.Remove(version, config.GlobalPaths); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"version": version,
		"status":  "removed",
	})
}

func handleFirecrackerVersions(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	versions, err := firecracker.ShowVersions(config.GlobalPaths)
	if err != nil {
		return errResult(err)
	}

	if versions == nil {
		versions = []firecracker.AvailableFirecracker{}
	}

	result := make([]map[string]any, len(versions))
	for i, v := range versions {
		result[i] = map[string]any{
			"version":      v.Version,
			"is_installed": v.IsInstalled,
			"is_default":   v.IsDefault,
		}
	}

	return jsonResult(map[string]any{
		"versions": result,
		"count":    len(result),
	})
}

func handleFirecrackerCreateRootfs(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	output := req.GetString("output", "")
	if output == "" {
		output = filepath.Join(config.GlobalPaths.DataDir, "alpine-rootfs.ext4")
	}

	sizeMB := req.GetInt("size_mb", 512)
	inject := req.GetBool("inject_binary", false)
	force := req.GetBool("force", false)

	opts := rootfs.CreateOptions{
		OutputPath:     output,
		SizeMB:         sizeMB,
		InjectBinary:   inject,
		ForceOverwrite: force,
	}

	if err := rootfs.Create(opts); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"output":        output,
		"size_mb":       sizeMB,
		"inject_binary": inject,
		"status":        "created",
	})
}
