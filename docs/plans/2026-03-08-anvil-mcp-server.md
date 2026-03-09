# Anvil MCP Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an MCP stdio server to Anvil so AI agents can manage the full kernel lifecycle — config, build, test, sign, archive.

**Architecture:** Direct stdio server using `mcp-go` (`github.com/mark3labs/mcp-go`). The `anvil mcp-server` cobra command starts the server, which calls `pkg/` functions directly. Kernel builds run async via a `BuildManager` with goroutines. Context switching between repo mode and user mode is explicit via tools.

**Tech Stack:** Go, mcp-go, existing Anvil `pkg/` packages (kernel, config, firecracker, signing)

**Design Doc:** `docs/anvil-mcp-server-research.md`

---

## Task 1: Add mcp-go dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum` (auto-generated)

**Step 1: Add the dependency**

Run: `cd /home/kazw/Work/WorkFort/anvil && go get github.com/mark3labs/mcp-go@latest`

**Step 2: Verify it resolves**

Run: `go mod tidy`
Expected: Clean exit, `go.mod` now contains `github.com/mark3labs/mcp-go`

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat(mcp): add mcp-go dependency"
```

---

## Task 2: Server skeleton + cobra command

**Files:**
- Create: `cmd/mcp/mcp.go`
- Create: `internal/mcp/server.go`
- Modify: `cmd/root.go` (add `mcp-server` subcommand)

**Step 1: Create `internal/mcp/server.go`**

```go
// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates the Anvil MCP server with all tools registered.
func NewServer(version string) *server.MCPServer {
	s := server.NewMCPServer(
		"anvil",
		version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	registerContextTools(s)

	return s
}
```

**Step 2: Create `cmd/mcp/mcp.go`**

```go
// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"github.com/spf13/cobra"
	"github.com/mark3labs/mcp-go/server"

	mcpserver "github.com/Work-Fort/Anvil/internal/mcp"
)

// NewMCPServerCmd creates the mcp-server command.
func NewMCPServerCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mcp-server",
		Short:  "Start MCP stdio server for AI agent integration",
		Long:   "Start a Model Context Protocol server on stdin/stdout. Used by Claude Code and other MCP clients to manage kernels, configs, signing, and builds.",
		Hidden: true, // Not for direct user invocation
		RunE: func(cmd *cobra.Command, args []string) error {
			s := mcpserver.NewServer(version)
			return server.ServeStdio(s)
		},
	}

	return cmd
}
```

**Step 3: Register in `cmd/root.go`**

Add `rootCmd.AddCommand(mcpcmd.NewMCPServerCmd(Version))` alongside the other subcommands. Add import `mcpcmd "github.com/Work-Fort/Anvil/cmd/mcp"`.

**Step 4: Create stub `internal/mcp/tools_context.go`**

```go
// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerContextTools(s *server.MCPServer) {
	s.AddTool(mcp.NewTool("get_context",
		mcp.WithDescription("Get current mode (user/repo), resolved paths, and active config"),
	), handleGetContext)
}

func handleGetContext(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mode := "user"
	if config.IsRepoMode() {
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

	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to marshal context: %v", err)), nil
	}

	return mcp.NewToolResultText(string(b)), nil
}
```

**Step 5: Verify it compiles**

Run: `cd /home/kazw/Work/WorkFort/anvil && go build ./...`
Expected: Clean build

**Step 6: Smoke test**

Run: `echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | go run . mcp-server`
Expected: JSON response with server capabilities

**Step 7: Commit**

```bash
git add cmd/mcp/ internal/mcp/ cmd/root.go
git commit -m "feat(mcp): server skeleton with get_context tool"
```

---

## Task 3: Context tools (set_repo_root, set_user_mode)

**Files:**
- Modify: `internal/mcp/tools_context.go`
- Modify: `pkg/config/viper.go` (add `IsUserMode()`, mode override support)

**Step 1: Add `IsUserMode()` and mode override to `pkg/config/viper.go`**

Add a package-level `var userModeOverride bool` and:

```go
// SetUserModeOverride forces user mode regardless of cwd.
func SetUserModeOverride(override bool) {
	userModeOverride = override
}

// IsUserMode returns true when operating in user mode (XDG paths).
func IsUserMode() bool {
	if userModeOverride {
		return true
	}
	return !IsRepoMode()
}
```

**Step 2: Add `set_repo_root` and `set_user_mode` tools**

In `internal/mcp/tools_context.go`, add registrations inside `registerContextTools`:

```go
s.AddTool(mcp.NewTool("set_repo_root",
    mcp.WithDescription("Switch to repo mode by setting the repo root path (must contain anvil.yaml)"),
    mcp.WithString("path", mcp.Required(), mcp.Description("Path to directory containing anvil.yaml")),
), handleSetRepoRoot)

s.AddTool(mcp.NewTool("set_user_mode",
    mcp.WithDescription("Switch to user mode (XDG paths), ignoring any anvil.yaml"),
), handleSetUserMode)
```

`handleSetRepoRoot` validates path exists, contains `anvil.yaml`, calls `os.Chdir(path)`, clears user mode override, reloads config, returns new context.

`handleSetUserMode` sets `config.SetUserModeOverride(true)`, reloads config, returns new context.

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/tools_context.go pkg/config/viper.go
git commit -m "feat(mcp): add set_repo_root and set_user_mode context tools"
```

---

## Task 4: Helper utilities

**Files:**
- Create: `internal/mcp/helpers.go`

**Step 1: Create helpers**

```go
// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// jsonResult marshals any value to a JSON text result.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	return mcp.NewToolResultText(string(b)), nil
}

// errResult returns an MCP error result.
func errResult(err error) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(err.Error()), nil
}
```

**Step 2: Refactor `handleGetContext` to use `jsonResult`**

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/helpers.go internal/mcp/tools_context.go
git commit -m "feat(mcp): add helper utilities (jsonResult, errResult)"
```

---

## Task 5: Anvil config tools

**Files:**
- Create: `internal/mcp/tools_config.go`
- Modify: `internal/mcp/server.go` (register)

**Step 1: Implement config tools**

Four tools: `config_get`, `config_set`, `config_list`, `config_get_paths`.

These are thin wrappers around `config.GetConfigValue()`, `config.SetConfigValue()`, `config.ListConfigValues()`, and `config.GlobalPaths`.

`config_set` must determine the correct scope: if `config.IsRepoMode()` and the key allows repo scope, use `config.ScopeRepo`; otherwise use `config.ScopeUser`. The existing `ValidateScope()` in `schema.go` handles constraint checking.

**Step 2: Register in `server.go`**

Add `registerConfigTools(s)` call.

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/tools_config.go internal/mcp/server.go
git commit -m "feat(mcp): add anvil config tools (get, set, list, paths)"
```

---

## Task 6: Kernel config parser

**Files:**
- Create: `pkg/kconfig/kconfig.go`
- Create: `pkg/kconfig/kconfig_test.go`

This is the `.config` file parser/editor. It's a standalone package because it's useful beyond MCP (future CLI commands, etc.).

**Step 1: Write failing tests**

Test cases:
- Parse `CONFIG_FOO=y` → option "FOO", value "y"
- Parse `# CONFIG_FOO is not set` → option "FOO", value "n"
- Parse `CONFIG_BAR=17` → option "BAR", value "17"
- Parse section comments (pass through, don't treat as options)
- Roundtrip: parse then write produces identical output
- Set option from "n" to "y": replaces `# CONFIG_FOO is not set` with `CONFIG_FOO=y`
- Set option from "y" to "n": replaces `CONFIG_FOO=y` with `# CONFIG_FOO is not set`
- Set option to "m": returns error (modules not supported)
- Diff two configs: detect additions, removals, value changes
- Get nonexistent option: returns not-found indicator
- List with filter: regex/substring match on option names

Run: `go test ./pkg/kconfig/ -v`
Expected: FAIL (package doesn't exist yet)

**Step 2: Implement `pkg/kconfig/kconfig.go`**

Key types:

```go
type Option struct {
    Name  string // Without CONFIG_ prefix
    Value string // "y", "n", or arbitrary string
}

type Config struct {
    lines []configLine // preserves file structure
}

type configLine struct {
    kind    lineKind // option, comment, blank
    raw     string   // original text
    option  string   // option name (without CONFIG_ prefix) if kind==option
    value   string   // value if kind==option
}
```

Key functions:
- `Parse(r io.Reader) (*Config, error)`
- `ParseFile(path string) (*Config, error)`
- `(c *Config) Get(name string) (string, bool)` — returns value, found
- `(c *Config) Set(name, value string) error` — rejects "m"
- `(c *Config) List(filter string) []Option`
- `(c *Config) WriteTo(w io.Writer) error`
- `(c *Config) WriteFile(path string) error`
- `Diff(a, b *Config) []DiffEntry`

**Step 3: Run tests**

Run: `go test ./pkg/kconfig/ -v`
Expected: All PASS

**Step 4: Commit**

```bash
git add pkg/kconfig/
git commit -m "feat(kconfig): kernel .config parser with get/set/list/diff"
```

---

## Task 7: Kernel config MCP tools

**Files:**
- Create: `internal/mcp/tools_kernel_config.go`
- Modify: `internal/mcp/server.go` (register)

**Step 1: Implement MCP tools**

Four tools wrapping `pkg/kconfig`: `kernel_config_get`, `kernel_config_set`, `kernel_config_list`, `kernel_config_diff`.

Each tool takes `config_file` as a required parameter (path to the `.config` file). In repo mode this is typically `configs/microvm-kernel-x86_64.config`.

**Step 2: Register in `server.go`**

Add `registerKernelConfigTools(s)` call.

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/tools_kernel_config.go internal/mcp/server.go
git commit -m "feat(mcp): add kernel config tools (get, set, list, diff)"
```

---

## Task 8: Kernel management tools

**Files:**
- Create: `internal/mcp/tools_kernel_mgmt.go`
- Modify: `internal/mcp/server.go` (register)

**Step 1: Implement tools**

Five tools: `kernel_list`, `kernel_get`, `kernel_set_default`, `kernel_remove`, `kernel_install`. Plus `kernel_list_versions` and `kernel_validate_version`.

These wrap existing `pkg/kernel/` functions:
- `kernel_list` → reads kernel directories from `config.GlobalPaths.KernelsDir`
- `kernel_list_versions` → `kernel.GetLatestKernelVersion()` and related
- `kernel_validate_version` → `kernel.ValidateVersion()`

Note: Some kernel management functions are in `cmd/kernel/` rather than `pkg/kernel/`. Check if the underlying functions exist in `pkg/` or if they need thin wrappers. For `list` and `remove`, the logic may live in `cmd/cmdutil/` (the `ShowVersionSelector` pattern). If so, call the underlying filesystem operations directly.

**Step 2: Register in `server.go`**

Add `registerKernelMgmtTools(s)` call.

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/tools_kernel_mgmt.go internal/mcp/server.go
git commit -m "feat(mcp): add kernel management tools (list, get, set, remove, install, versions)"
```

---

## Task 9: Build manager + async kernel build tools

**Files:**
- Create: `internal/mcp/build_manager.go`
- Create: `internal/mcp/build_manager_test.go`
- Create: `internal/mcp/tools_kernel_build.go`
- Modify: `internal/mcp/server.go` (register, pass BuildManager)

**Step 1: Write failing tests for BuildManager**

Test cases:
- `NewJob` creates a job with status "running" and correct ID format
- `RunningForArch` returns the running job for an arch
- `RunningForArch` returns nil when no job is running
- `GetJob` returns a job by ID
- `Job.SetPhase` updates phase
- `Job.SetProgress` updates progress
- `Job.Complete` sets status to "completed" and populates stats
- `Job.Fail` sets status to "failed" and populates error
- Concurrent build rejection: second build for same arch is rejected

Run: `go test ./internal/mcp/ -run TestBuildManager -v`
Expected: FAIL

**Step 2: Implement `internal/mcp/build_manager.go`**

```go
type BuildJob struct {
    ID          string
    Arch        string
    Status      string    // running, completed, failed, cancelled
    Phase       string
    Progress    float64
    StartedAt   time.Time
    CompletedAt time.Time
    Stats       *kernel.BuildStats
    Error       string
    LogLines    []string  // circular buffer of last 200 lines
    cancel      context.CancelFunc
    done        chan struct{} // closed when build completes (for kernel_build_wait)
    mu          sync.RWMutex
}

type BuildManager struct {
    mu   sync.RWMutex
    jobs map[string]*BuildJob
}
```

Key methods: `NewJob`, `GetJob`, `RunningForArch`, `SetPhase`, `SetProgress`, `Complete`, `Fail`, `Cancel`, `AppendLog`, `GetLogLines`, `Wait`.

The `done` channel is closed on `Complete` or `Fail`, allowing `kernel_build_wait` to select on it.

**Step 3: Run tests**

Run: `go test ./internal/mcp/ -run TestBuildManager -v`
Expected: All PASS

**Step 4: Implement `internal/mcp/tools_kernel_build.go`**

Five tools: `kernel_build`, `kernel_build_status`, `kernel_build_log`, `kernel_build_wait`, `kernel_build_cancel`.

`kernel_build`: validates arch, checks no running build for that arch, creates job, launches goroutine calling `kernel.Build(opts)`, returns build ID immediately.

`kernel_build_status`: looks up job by ID, returns status/phase/progress/elapsed/stats/error.

`kernel_build_log`: returns last N lines from `job.LogLines`.

`kernel_build_wait`: calls `job.Wait()` (blocks on `done` channel), then returns final status same as `kernel_build_status`.

`kernel_build_cancel`: calls `job.Cancel()` which triggers the context cancellation.

**Step 5: Update `server.go`**

Change `NewServer` to create a `BuildManager` and pass it to `registerKernelBuildTools(s, bm)`.

**Step 6: Verify it compiles**

Run: `go build ./...`

**Step 7: Commit**

```bash
git add internal/mcp/build_manager.go internal/mcp/build_manager_test.go internal/mcp/tools_kernel_build.go internal/mcp/server.go
git commit -m "feat(mcp): async kernel build with BuildManager and build tools"
```

---

## Task 10: Firecracker tools

**Files:**
- Create: `internal/mcp/tools_firecracker.go`
- Modify: `internal/mcp/server.go` (register)

**Step 1: Implement tools**

Seven tools wrapping existing `pkg/firecracker/` functions:

- `firecracker_test` → `firecracker.Test(opts)` — returns TestResult JSON
- `firecracker_list` → `firecracker.List()` or read directory
- `firecracker_get` → `firecracker.Download(version)`
- `firecracker_set_default` → `firecracker.Set(version)`
- `firecracker_remove` → same pattern as kernel remove
- `firecracker_versions` → `firecracker.ShowVersions()` or GitHub API
- `firecracker_create_rootfs` → `rootfs.Create(opts)`

Note: Some of these functions (like `firecracker.List()`) may write directly to stdout. If so, capture output via a buffer. Check the function signatures — if they take `io.Writer`, pass a `bytes.Buffer`. If they write to `os.Stdout` directly, this needs a wrapper or refactor.

**Step 2: Register in `server.go`**

Add `registerFirecrackerTools(s)` call.

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/tools_firecracker.go internal/mcp/server.go
git commit -m "feat(mcp): add firecracker tools (test, list, get, set, remove, versions, create-rootfs)"
```

---

## Task 11: Signing tools

**Files:**
- Create: `internal/mcp/tools_signing.go`
- Modify: `internal/mcp/server.go` (register, add session password state)

**Step 1: Implement tools**

Seven tools: `signing_generate_key`, `signing_sign`, `signing_verify`, `signing_key_info`, `signing_rotate_key`, `signing_export_public_key`, `signing_unlock`.

Password handling:
- Check `ANVIL_SIGNING_PASSWORD` env var first
- If not set, check session password (set via `signing_unlock`)
- If neither, return error explaining options

`signing_unlock` stores the password in a `*string` field on a `SigningState` struct held by the server. The password is passed to signing functions that need it.

Check `pkg/signing/signing.go` for function signatures. The signing functions likely accept password as a parameter or via the config system.

**Step 2: Register in `server.go`**

Create `SigningState` in `NewServer`, pass to `registerSigningTools(s, ss)`.

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/tools_signing.go internal/mcp/server.go
git commit -m "feat(mcp): add signing tools with session-based password unlock"
```

---

## Task 12: Archive tools

**Files:**
- Create: `internal/mcp/tools_archive.go`
- Modify: `internal/mcp/server.go` (register)

**Step 1: Implement tools**

Three tools: `archive_kernel`, `archive_list`, `archive_get`.

These wrap `kernel.ArchiveInstalledKernel()` and read from the archive `index.json` at the configured `kernels.archive.location`.

`archive_kernel`: requires repo mode (archive is repo-specific). Returns error in user mode.

`archive_list` and `archive_get`: read the archive index JSON file.

**Step 2: Register in `server.go`**

Add `registerArchiveTools(s)` call.

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/tools_archive.go internal/mcp/server.go
git commit -m "feat(mcp): add archive tools (kernel, list, get)"
```

---

## Task 13: Utility tools

**Files:**
- Create: `internal/mcp/tools_utility.go`
- Modify: `internal/mcp/server.go` (register)

**Step 1: Implement tools**

Two tools: `check_build_tools`, `clean_build_cache`.

`check_build_tools`: checks for required binaries (`make`, `gcc`, cross-compile toolchain for aarch64, `xz`, etc.). Returns a list of found/missing tools.

`clean_build_cache`: removes build artifacts from `config.GlobalPaths.KernelBuildDir`. Supports filtering by version/arch or cleaning all.

**Step 2: Register in `server.go`**

Add `registerUtilityTools(s)` call.

**Step 3: Verify it compiles**

Run: `go build ./...`

**Step 4: Commit**

```bash
git add internal/mcp/tools_utility.go internal/mcp/server.go
git commit -m "feat(mcp): add utility tools (check_build_tools, clean_build_cache)"
```

---

## Task 14: .mcp.json and end-to-end verification

**Files:**
- Create: `.mcp.json` at repo root

**Step 1: Create `.mcp.json`**

```json
{
  "mcpServers": {
    "anvil": {
      "command": "anvil",
      "args": ["mcp-server"]
    }
  }
}
```

**Step 2: Build the binary**

Run: `go build -o anvil .`

**Step 3: Manual end-to-end test**

Test the full MCP flow with JSON-RPC messages on stdin:

```bash
# Initialize
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./anvil mcp-server

# List tools
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./anvil mcp-server
```

Expected: Response listing all registered tools.

**Step 4: Test get_context**

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_context","arguments":{}}}' | ./anvil mcp-server
```

Expected: JSON showing repo mode (since cwd has anvil.yaml), with all paths.

**Step 5: Commit**

```bash
git add .mcp.json
git commit -m "feat(mcp): add .mcp.json for Claude Code integration"
```

---

## Implementation Notes

### Import Path

The Go module is `github.com/Work-Fort/Anvil`. Internal packages go under `internal/mcp/` so they're not importable outside the module.

### Version String

The version is set at build time via `cmd.Version` (ldflags). Pass it from `root.go` when creating the MCP command: `mcpcmd.NewMCPServerCmd(Version)`.

### Logging

The MCP server must NOT write to stdout (that's the MCP transport). Use `charmbracelet/log` which writes to stderr, or use the file-based debug logger from `cmd.GetDebugLogger()`.

### Error Conventions

Tool handlers should return `(result, nil)` for both success and tool-level errors. Use `mcp.NewToolResultError()` for errors the agent should see. Only return `(nil, err)` for protocol-level failures.

### mcp-go API Quick Reference

```go
// Parameter extraction
name := req.GetString("name", "default")
name, err := req.RequireString("name")
count := req.GetInt("count", 10)

// Results
mcp.NewToolResultText("success message")
mcp.NewToolResultError("error message")

// JSON results (use our helper)
jsonResult(map[string]any{"key": "value"})
errResult(fmt.Errorf("something failed"))

// Notifications
s.SendNotificationToClient(ctx, "method.name", map[string]any{"key": "value"})

// Server creation
s := server.NewMCPServer("name", "version", server.WithToolCapabilities(true))
server.ServeStdio(s)
```
