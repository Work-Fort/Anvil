// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/kernel"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerKernelBuildTools(s *server.MCPServer, bm *BuildManager) {
	s.AddTool(gomcp.NewTool("kernel_build",
		gomcp.WithDescription("Start a kernel build (returns immediately with build ID). Use kernel_build_status or kernel_build_wait to monitor."),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Kernel version (e.g. 6.19.6)")),
		gomcp.WithString("arch", gomcp.Required(), gomcp.Description("Target architecture: x86_64 or aarch64")),
		gomcp.WithString("config_file", gomcp.Description("Custom kernel config file path (overrides anvil.yaml)")),
		gomcp.WithString("verification_level", gomcp.Description("Source verification: high (default), medium, or disabled"),
			gomcp.Enum("high", "medium", "disabled")),
	), func(ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		return handleKernelBuild(s, bm, ctx, req)
	})

	s.AddTool(gomcp.NewTool("kernel_build_status",
		gomcp.WithDescription("Check build progress and result by build ID"),
		gomcp.WithString("build_id", gomcp.Required(), gomcp.Description("Build ID from kernel_build")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), func(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		return handleKernelBuildStatus(bm, req)
	})

	s.AddTool(gomcp.NewTool("kernel_build_log",
		gomcp.WithDescription("Get recent build output lines for debugging"),
		gomcp.WithString("build_id", gomcp.Required(), gomcp.Description("Build ID from kernel_build")),
		gomcp.WithNumber("lines", gomcp.Description("Number of lines to return (default 50)")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), func(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		return handleKernelBuildLog(bm, req)
	})

	s.AddTool(gomcp.NewTool("kernel_build_wait",
		gomcp.WithDescription("Block until a build completes, then return the result"),
		gomcp.WithString("build_id", gomcp.Required(), gomcp.Description("Build ID from kernel_build")),
	), func(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		return handleKernelBuildWait(bm, req)
	})

	s.AddTool(gomcp.NewTool("kernel_build_cancel",
		gomcp.WithDescription("Cancel a running build"),
		gomcp.WithString("build_id", gomcp.Required(), gomcp.Description("Build ID from kernel_build")),
		gomcp.WithDestructiveHintAnnotation(true),
	), func(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		return handleKernelBuildCancel(bm, req)
	})

	s.AddTool(gomcp.NewTool("kernel_list_versions",
		gomcp.WithDescription("List available kernel versions from kernel.org"),
		gomcp.WithReadOnlyHintAnnotation(true),
	), func(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		latest, err := kernel.GetLatestKernelVersion()
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"latest": latest})
	})

	s.AddTool(gomcp.NewTool("kernel_validate_version",
		gomcp.WithDescription("Check if a kernel version exists on kernel.org"),
		gomcp.WithString("version", gomcp.Required(), gomcp.Description("Kernel version to check")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), func(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
		version, err := req.RequireString("version")
		if err != nil {
			return errResult(err)
		}
		if err := kernel.ValidateVersion(version); err != nil {
			return jsonResult(map[string]any{"version": version, "valid": false, "error": err.Error()})
		}
		return jsonResult(map[string]any{"version": version, "valid": true})
	})
}

var phaseNames = map[kernel.BuildPhase]string{
	kernel.PhaseDownload:  "download",
	kernel.PhaseVerify:    "verify",
	kernel.PhaseExtract:   "extract",
	kernel.PhaseConfigure: "configure",
	kernel.PhaseCompile:   "compile",
	kernel.PhasePackage:   "package",
}

func phaseName(p kernel.BuildPhase) string {
	if name, ok := phaseNames[p]; ok {
		return name
	}
	return fmt.Sprintf("phase-%d", int(p))
}

func handleKernelBuild(s *server.MCPServer, bm *BuildManager, ctx context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	version, err := req.RequireString("version")
	if err != nil {
		return errResult(err)
	}
	arch, err := req.RequireString("arch")
	if err != nil {
		return errResult(err)
	}

	if arch != "x86_64" && arch != "aarch64" {
		return errResult(fmt.Errorf("invalid arch %q: must be x86_64 or aarch64", arch))
	}

	// Reject if a build for this arch is already running
	if existing := bm.RunningForArch(arch); existing != nil {
		return errResult(fmt.Errorf("build already running for %s: %s", arch, existing.ID))
	}

	configFile := req.GetString("config_file", "")
	verLevel := req.GetString("verification_level", "high")

	// Create job with cancellable context
	buildCtx, cancel := context.WithCancel(context.Background())
	job := bm.NewJob(version, arch, cancel)

	var logBuf bytes.Buffer

	opts := kernel.BuildOptions{
		Version:           version,
		Arch:              arch,
		ConfigFile:        configFile,
		VerificationLevel: verLevel,
		Interactive:       false,
		Writer:            &logBuf,
		Context:           buildCtx,
		PhaseCallback: func(phase kernel.BuildPhase) {
			name := phaseName(phase)
			job.SetPhase(name)
			_ = s.SendNotificationToClient(ctx, "kernel_build.phase", map[string]any{
				"build_id": job.ID, "phase": name,
			})
		},
		ProgressCallback: func(pct float64) {
			job.SetProgress(pct)
		},
	}

	// Run build in background goroutine
	go func() {
		if err := kernel.Build(opts); err != nil {
			job.Fail(err)
			_ = s.SendNotificationToClient(ctx, "kernel_build.completed", map[string]any{
				"build_id": job.ID, "status": "failed", "error": err.Error(),
			})
			return
		}

		// Try to read build stats
		statsPath := "" // The build function stores stats alongside the output
		stats, _ := kernel.ReadBuildStats(statsPath)
		job.Complete(&stats)
		_ = s.SendNotificationToClient(ctx, "kernel_build.completed", map[string]any{
			"build_id": job.ID, "status": "completed",
		})
	}()

	return jsonResult(map[string]any{
		"build_id": job.ID,
		"status":   "running",
		"version":  version,
		"arch":     arch,
	})
}

func handleKernelBuildStatus(bm *BuildManager, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	id, err := req.RequireString("build_id")
	if err != nil {
		return errResult(err)
	}

	job := bm.GetJob(id)
	if job == nil {
		return errResult(fmt.Errorf("build not found: %s", id))
	}

	return jsonResult(job.Snapshot())
}

func handleKernelBuildLog(bm *BuildManager, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	id, err := req.RequireString("build_id")
	if err != nil {
		return errResult(err)
	}

	job := bm.GetJob(id)
	if job == nil {
		return errResult(fmt.Errorf("build not found: %s", id))
	}

	lines := req.GetInt("lines", 50)
	return jsonResult(map[string]any{
		"build_id": id,
		"lines":    job.GetLogLines(lines),
	})
}

func handleKernelBuildWait(bm *BuildManager, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	id, err := req.RequireString("build_id")
	if err != nil {
		return errResult(err)
	}

	job := bm.GetJob(id)
	if job == nil {
		return errResult(fmt.Errorf("build not found: %s", id))
	}

	job.Wait()
	return jsonResult(job.Snapshot())
}

func handleKernelBuildCancel(bm *BuildManager, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	id, err := req.RequireString("build_id")
	if err != nil {
		return errResult(err)
	}

	job := bm.GetJob(id)
	if job == nil {
		return errResult(fmt.Errorf("build not found: %s", id))
	}

	job.Cancel()
	return jsonResult(map[string]any{
		"build_id": id,
		"status":   "cancelled",
	})
}
