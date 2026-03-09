# Anvil MCP Server — Research & Proposed Architecture

Research into adding an MCP (Model Context Protocol) stdio server to Anvil,
enabling AI agents to manage the full kernel lifecycle: configuration,
compilation, signing, archiving, and releasing.

## Background

### Why MCP?

The Nexus team uses the `nexusctl` MCP to let Claude Code manage VMs, drives,
and device passthrough without manual CLI invocation. Adding an MCP to Anvil
would let an agent:

- Build kernels for multiple architectures in sequence
- Monitor build progress and react to failures
- Sign artifacts with the project signing key
- Archive and prepare releases
- Query installed kernels and build history
- Manage kernel configs programmatically

### Reference: nexusctl MCP

Nexusctl implements a **stdio-to-HTTP bridge** model because nexus runs as a
persistent daemon:

```
Claude Code → stdin/stdout → nexusctl mcp-bridge → HTTP POST → nexus daemon /mcp
```

Anvil is a pure CLI with no daemon, so we use a **direct stdio server** model
instead:

```
Claude Code → stdin/stdout → anvil mcp-server (in-process, calls pkg/ directly)
```

Both use `mcp-go` (github.com/mark3labs/mcp-go) for protocol handling.

## Proposed Architecture

### Transport: Direct stdio

A new `anvil mcp-server` command starts an MCP server on stdin/stdout using
`mcp-go`'s `server.NewStdioServer()`. The server calls into Anvil's existing
`pkg/` packages directly — no HTTP layer, no daemon.

```
┌──────────────┐     stdio      ┌──────────────────────────────┐
│ Claude Code  │ ──────────── → │ anvil mcp-server             │
│ (MCP client) │ ← ──────────── │                              │
└──────────────┘                │  ┌────────────────────────┐  │
                                │  │ mcp-go stdio server    │  │
                                │  └───────────┬────────────┘  │
                                │              │               │
                                │  ┌───────────┴────────────┐  │
                                │  │ Tool handlers           │  │
                                │  │ (kernel, signing, etc)  │  │
                                │  └───────────┬────────────┘  │
                                │              │               │
                                │  ┌───────────┴────────────┐  │
                                │  │ pkg/kernel  pkg/signing │  │
                                │  │ pkg/config  pkg/util    │  │
                                │  └────────────────────────┘  │
                                └──────────────────────────────┘
```

### Dual-Context Operation

Anvil already supports two contexts via its config system:

| Context | Config Source | Paths | Use Case |
|---------|-------------|-------|----------|
| **User** | `~/.config/anvil/config.yaml` | XDG dirs (`~/.cache/anvil/`, `~/.local/share/anvil/`) | Personal builds, default keys |
| **Repo** | `./anvil.yaml` | Repo-relative (`./archive/`, `./keys/`, `./configs/`) | Project builds, team signing keys |

The MCP server inherits this automatically. When Claude Code spawns
`anvil mcp-server` from a working directory that contains `anvil.yaml`, Anvil
detects the repo context and uses repo-relative paths. When spawned from
outside a repo, it falls back to XDG user paths.

This is already how Anvil works — `config.LoadConfig()` in `cmd/root.go`
handles the detection. The MCP server just needs to call the same
initialization path.

### Working Directory (cwd) Behavior

**Critical finding**: How the MCP client sets the subprocess cwd determines
whether Anvil detects repo mode. Research across MCP clients:

| Client | cwd for stdio MCP servers |
|--------|--------------------------|
| **Claude Code** | Session project root (inherited from where `claude` was launched) |
| **Claude Desktop** | Undefined — could be `/` on macOS |
| **Cursor** | Server's install location (not workspace) |
| **VS Code** | Home directory by default; `cwd` config available |
| **Cline** | Workspace root (after PR #2990 bugfix) |

**The MCP specification says nothing about cwd** — it's entirely
client-defined. The `.mcp.json` `cwd` field is documented in Claude Code's
plugin reference but is **ignored at runtime** (GitHub issue #17565, closed
NOT_PLANNED).

**Why this works for us**: Claude Code spawns MCP servers with cwd set to the
project root. When `.mcp.json` lives at the Anvil repo root alongside
`anvil.yaml`, Anvil's `IsRepoMode()` check will find the config file and
activate repo mode. For user-scoped config (`~/.claude/settings.json`), the
cwd will be whatever project the user is in — no `anvil.yaml` means user mode.
Both cases behave correctly.

**Defensive option for non-Claude clients**: Accept an optional `--repo-root`
flag on `anvil mcp-server` that overrides the cwd-based detection. This is
not needed for Claude Code but would help with clients that don't reliably set
cwd (Claude Desktop, Cursor). Low priority for initial implementation.

### .mcp.json Configuration

**Repo-level** (`.mcp.json` at repo root):

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

**User-level** (`~/.claude/settings.json`):

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

Both use the same binary and command. The context is determined by the cwd
when the MCP server starts, just like running `anvil` from the terminal.

## Proposed Tool Surface

### Kernel Build Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `kernel_build` | Start a kernel build (async) | `version`, `arch`, `config_file`, `verification_level` |
| `kernel_build_status` | Check build progress/result by ID | `build_id` |
| `kernel_build_log` | Get recent build output lines | `build_id`, `lines` |
| `kernel_build_wait` | Block until build completes, return result | `build_id` |
| `kernel_build_cancel` | Cancel a running build | `build_id` |
| `kernel_list_versions` | List available versions from kernel.org | `count` |
| `kernel_validate_version` | Check if a version exists on kernel.org | `version` |

**Async build model**: Kernel builds take 10-15 minutes. A synchronous
tool call would block the agent for the entire duration, preventing it
from doing any other work. Instead, `kernel_build` starts the build in a
background goroutine and returns immediately with a `build_id`.

**Agent workflow** (async — agent has other work to do):
1. Call `kernel_build` → returns `{ "build_id": "b-1720000000-x86_64", "status": "running" }`
2. Agent does other work (edit configs, manage firecracker, etc.)
3. Periodically call `kernel_build_status` → returns phase, progress %, elapsed time

**Agent workflow** (sync — user wants to wait):
1. Call `kernel_build` → returns `{ "build_id": "b-1720000000-x86_64", "status": "running" }`
2. Call `kernel_build_wait` with the build ID → blocks until done, returns final result
4. When status is `completed` or `failed`, result includes `BuildStats` or error details
5. If build fails, call `kernel_build_log` to inspect compiler output

**Build ID format**: `b-{unix_timestamp}-{arch}` — simple, sortable,
identifies the target architecture at a glance.

**Build state**: The MCP server holds a `map[string]*BuildJob` in memory.
Each job tracks:
- `Status`: `running`, `completed`, `failed`, `cancelled`
- `Phase`: current build phase (download, verify, extract, configure, compile, package)
- `Progress`: percentage within current phase
- `StartedAt`, `CompletedAt`: timestamps
- `Stats`: `BuildStats` (populated on completion)
- `Error`: error message (populated on failure)
- `LogRing`: circular buffer of recent build output lines
- `CancelFunc`: `context.CancelFunc` for cancellation

**Concurrent builds**: Only one build per architecture can run at a time.
Attempting to start an x86_64 build while one is running returns an error
with the existing build ID. Builds for different architectures can run
concurrently.

**Notifications**: In addition to polling, the server sends MCP
notifications for phase transitions and completion. Agents that support
notifications get real-time updates without polling. Agents that don't
(or prefer polling) use `kernel_build_status`.

**Build persistence**: Build state lives in MCP server memory only — it
does not survive server restarts. However, completed builds leave
artifacts on disk (`build-stats.json`, kernel binary, etc.) which are
discoverable via `kernel_list` and `kernel_get`.

### Kernel Management Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `kernel_list` | List installed kernels | `arch` |
| `kernel_get` | Get details of an installed kernel | `id` or `version` |
| `kernel_set_default` | Set the default kernel | `id` |
| `kernel_remove` | Remove an installed kernel | `id` |
| `kernel_install` | Install a built kernel from cache | `version`, `arch` |

These wrap the existing `cmd/kernel/` subcommands, calling the same
underlying functions.

### Signing Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `signing_generate_key` | Generate a new PGP signing key | `name`, `email`, `expiry` |
| `signing_sign` | Sign artifacts in a directory | `path` |
| `signing_verify` | Verify signatures | `path` |
| `signing_key_info` | Get current signing key details | — |
| `signing_rotate_key` | Rotate the signing key | `name`, `email`, `expiry` |
| `signing_export_public_key` | Export public key | `output_path` |

**Password handling**: The MCP server cannot prompt for a password
interactively. Options:
1. `ANVIL_SIGNING_PASSWORD` env var (set in `.mcp.json` env block)
2. A `signing_unlock` tool that accepts the password once per session
3. Support for unencrypted keys in non-repo context (user choice)

Recommended: option 1 for CI/automated use, option 2 for interactive agent
sessions.

### Archive & Release Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `archive_kernel` | Copy built kernel to repo archive | `version`, `arch` |
| `archive_list` | List archived kernels | `arch` |
| `archive_get` | Get details of an archived kernel | `version`, `arch` |

These wrap `kernel.ArchiveInstalledKernel()` and read from the archive
`index.json`.

### Anvil Configuration Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `config_get` | Get an anvil.yaml / user config value | `key` |
| `config_set` | Set an anvil.yaml / user config value | `key`, `value` |
| `config_list` | List all config values | — |
| `config_get_paths` | Get resolved directory paths | — |

These operate on **Anvil's own config** (`anvil.yaml` or
`~/.config/anvil/config.yaml`), not on kernel `.config` files. See
"Kernel Config Tools" below for `.config` editing.

`config_get_paths` is important for agents — it tells them where artifacts,
keys, and builds live in the current context.

### Kernel Config Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `kernel_config_get` | Get value of a CONFIG_* option | `config_file`, `option` |
| `kernel_config_set` | Set a CONFIG_* option | `config_file`, `option`, `value` |
| `kernel_config_list` | List CONFIG_* options with optional filter | `config_file`, `filter` |
| `kernel_config_diff` | Compare two kernel configs | `file_a`, `file_b` |

**Why these tools exist**: Kernel `.config` files are 80-90 KB with thousands
of options — too large for an agent to read and edit via text manipulation.
These tools provide structured access to individual options.

**Format handling**: Linux kernel `.config` files use a specific format where
disabled options are written as comments (`# CONFIG_FOO is not set`) rather
than `CONFIG_FOO=n`. Both formats are accepted as input by kconfig, but
kconfig only emits the comment form. The parser must handle:

- `CONFIG_FOO=y` — enabled (boolean)
- `CONFIG_FOO=<value>` — enabled with string/int value
- `# CONFIG_FOO is not set` — disabled (kconfig canonical output)
- `CONFIG_FOO=n` — disabled (accepted input, never seen in practice)
- `# <text>` without `CONFIG_` prefix — section comment (pass through)

Verified against kernel 6.19.6 `scripts/kconfig/confdata.c`: the
`print_symbol_for_dotconfig()` function writes disabled options as
`# CONFIG_FOO is not set` (via `OUTPUT_N_AS_UNSET`). The 2022 patch to
switch output to `=n` format was never merged.

**No module support**: Anvil compiles monolithic/statically-linked kernels
only — no `=m` (module) values. Firecracker and Kata VMs boot from a single
vmlinux with no initramfs or module loading. If `kernel_config_set` receives
`value=m`, it returns an error explaining that Anvil only supports static
compilation. If `kernel_config_get` or `kernel_config_list` encounters an
existing `=m` entry, it reports it with a warning.

**Dependency resolution**: The tools edit the `.config` file directly without
resolving Kconfig dependencies. Dependency resolution happens automatically
when `make olddefconfig` runs during the build's configure phase. This is
the same approach kernel developers use — edit the config, then let the
build system reconcile dependencies.

**`kernel_config_set` behavior**:
- `value=y`: writes `CONFIG_FOO=y` (removes any existing `is not set` line)
- `value=n`: writes `# CONFIG_FOO is not set` (removes any existing
  `CONFIG_FOO=...` line)
- `value=<string>`: writes `CONFIG_FOO=<string>` (for integer/hex/string
  options like `CONFIG_LOG_BUF_SHIFT=17`)

**`kernel_config_diff` behavior**: Returns options that differ between two
configs — additions, removals, and value changes. Useful for comparing a
project config against a reference or reviewing what changed after
`make olddefconfig` reconciles dependencies.

### Firecracker Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `firecracker_test` | Run acceptance test (boot VM, vsock ping) | `kernel_version`, `rootfs`, `boot_timeout`, `ping_timeout` |
| `firecracker_list` | List installed Firecracker versions | — |
| `firecracker_get` | Download a Firecracker binary | `version` |
| `firecracker_set_default` | Set default Firecracker version | `version` |
| `firecracker_remove` | Remove an installed version | `version` |
| `firecracker_versions` | List available versions from GitHub | — |
| `firecracker_create_rootfs` | Create Alpine rootfs for testing | `output`, `size_mb`, `alpine_version`, `inject_binary` |

**`firecracker_test` is the acceptance test tool.** It wraps
`firecracker.Test()` which:

1. Verifies kernel and Firecracker binary are available
2. Creates a rootfs with the anvil binary injected (if needed)
3. Boots a Firecracker VM with vsock configured
4. Waits for VM boot
5. Tests vsock communication (ping/pong)
6. Cleans up and returns pass/fail with timing

This is critical for the agent workflow: build a kernel, then immediately
validate it boots and has working virtio-vsock. The agent can run this
after every kernel build or config change to catch regressions.

**`firecracker_create_rootfs`** creates the test rootfs. The agent needs
this before running acceptance tests — it builds an Alpine Linux ext4
image with the anvil vsock agent injected. Typically only needs to be run
once, but the agent should be able to recreate it if needed (e.g., after
an anvil binary update).

### Context Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `set_repo_root` | Set the repo root path, switching to repo mode | `path` |
| `set_user_mode` | Switch to standalone mode (XDG paths), ignoring any anvil.yaml | — |
| `get_context` | Get current mode (user/repo), resolved paths, active config | — |

`set_repo_root` is the most important context tool. It solves the cwd problem
across all MCP clients: an agent calls `set_repo_root` with the path to a
directory containing `anvil.yaml`, and the server reloads config in repo mode.
This works even when the MCP client doesn't set cwd to the project root
(Claude Desktop, Cursor, etc.).

Behavior:
- Validates that `path` exists and contains `anvil.yaml`
- Calls `os.Chdir(path)` and re-runs `config.LoadConfig()`
- All subsequent tool calls use repo-relative paths and repo config
- Returns the new context (mode, paths, config summary)

`set_user_mode` switches to user mode (XDG paths), regardless of the
current cwd. Useful for testing builds against user-scoped config and
keys without leaving the repo directory. Internally it sets a flag that
makes `IsRepoMode()` return false and re-runs `config.LoadConfig()` so
paths resolve to XDG defaults.

Note: The codebase currently has no formal name for non-repo mode — it's
referred to as "not in repo mode" or "outside repo context." We formalize
this as **user mode**, the named counterpart to **repo mode**. This aligns
with the XDG spec (which calls these "user directories") and with Anvil's
own config system (`~/.config/anvil/config.yaml` is the user config).
Add `IsUserMode()` as a helper (inverse of `IsRepoMode()` plus the MCP
override flag).

`get_context` lets the agent inspect the current state before deciding
whether to call `set_repo_root`. Returns mode, all resolved paths
(`GlobalPaths`), active config source, and signing key status.

### Utility Tools

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `check_build_tools` | Verify required build tools are installed | `arch` |
| `clean_build_cache` | Clean build cache | `version`, `arch`, `all` |

## Implementation Structure

### File Layout

```
cmd/
└── mcp/
    └── mcp.go                  # NewMCPServerCmd() — cobra command

internal/
└── mcp/
    ├── server.go               # Server setup, tool registration
    ├── tools_kernel_build.go   # kernel_build, kernel_build_status, etc.
    ├── tools_kernel_mgmt.go    # kernel_list, kernel_get, etc.
    ├── tools_signing.go        # signing_*, password session state
    ├── tools_archive.go        # archive_*
    ├── tools_config.go         # config_* (anvil.yaml)
    ├── tools_kernel_config.go  # kernel_config_* (.config files)
    ├── tools_firecracker.go    # firecracker_* (versions, test, rootfs)
    ├── tools_context.go        # set_repo_root, set_user_mode, get_context
    └── helpers.go              # jsonResult, errResult, requireString, etc.
```

### Server Setup (server.go)

```go
func NewServer() *server.MCPServer {
    s := server.NewMCPServer(
        "anvil",
        version.Version,  // from pkg/config or build-time
        server.WithToolCapabilities(true),
    )

    registerKernelBuildTools(s)
    registerKernelMgmtTools(s)
    registerSigningTools(s)
    registerArchiveTools(s)
    registerConfigTools(s)
    registerUtilityTools(s)

    return s
}
```

### Tool Registration Pattern (following nexusctl)

```go
func registerKernelBuildTools(s *server.MCPServer, bm *BuildManager) {
    s.AddTool(mcp.NewTool("kernel_build",
        mcp.WithDescription("Start a kernel build (returns immediately with build ID)"),
        mcp.WithString("version", mcp.Description("Kernel version (e.g. 6.19.6)")),
        mcp.WithString("arch", mcp.Description("Target: x86_64 or aarch64")),
        mcp.WithString("config_file", mcp.Description("Custom kernel config file path")),
        mcp.WithString("verification_level",
            mcp.Description("Verification: high, medium, disabled"),
            mcp.Enum("high", "medium", "disabled"),
        ),
    ), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // s and bm captured in closure
        return handleKernelBuild(s, bm, req)
    })

    s.AddTool(mcp.NewTool("kernel_build_status",
        mcp.WithDescription("Check build progress and result"),
        mcp.WithString("build_id", mcp.Required(), mcp.Description("Build ID from kernel_build")),
    ), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        return handleKernelBuildStatus(bm, req)
    })
}
```

### Async Build Manager

The MCP server holds a `BuildManager` that tracks in-flight and completed
builds:

```go
type BuildJob struct {
    ID          string              // b-{timestamp}-{arch}
    Status      string              // running, completed, failed, cancelled
    Phase       string              // download, verify, extract, configure, compile, package
    Progress    float64             // 0.0-1.0 within current phase
    StartedAt   time.Time
    CompletedAt time.Time
    Stats       *kernel.BuildStats  // populated on completion
    Error       string              // populated on failure
    LogRing     *ring.Buffer        // last N lines of build output
    cancel      context.CancelFunc
}

type BuildManager struct {
    mu   sync.Mutex
    jobs map[string]*BuildJob
}
```

**`kernel_build` handler** — starts the goroutine and returns immediately:

```go
func handleKernelBuild(s *server.MCPServer, bm *BuildManager, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    arch := mcp.ParseString(req, "arch", "")

    // Reject if a build for this arch is already running
    if existing := bm.RunningForArch(arch); existing != nil {
        return errResult(fmt.Errorf("build already running: %s", existing.ID))
    }

    // Create job with cancellable context
    ctx, cancel := context.WithCancel(context.Background())
    job := bm.NewJob(arch, cancel)

    opts := kernel.BuildOptions{
        Version:           mcp.ParseString(req, "version", ""),
        Arch:              arch,
        VerificationLevel: mcp.ParseString(req, "verification_level", "high"),
        Interactive:       false,
        Context:           ctx,
        PhaseCallback: func(phase kernel.BuildPhase) {
            job.SetPhase(phase.String())
            _ = s.SendNotificationToClient(ctx, "kernel_build.phase", map[string]any{
                "build_id": job.ID, "phase": phase.String(),
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
        job.Complete(stats)
        _ = s.SendNotificationToClient(ctx, "kernel_build.completed", map[string]any{
            "build_id": job.ID, "status": "completed",
        })
    }()

    // Return immediately with build ID
    return jsonResult(map[string]any{
        "build_id": job.ID,
        "status":   "running",
        "arch":     arch,
    })
}
```

**`kernel_build_status` response examples**:

```json
// In progress
{ "build_id": "b-1720000000-x86_64", "status": "running",
  "phase": "compile", "progress": 0.73, "elapsed": "8m12s" }

// Completed
{ "build_id": "b-1720000000-x86_64", "status": "completed",
  "elapsed": "12m34s", "stats": { "kernel_size": 20414976, ... } }

// Failed
{ "build_id": "b-1720000000-x86_64", "status": "failed",
  "phase": "compile", "error": "make failed with exit code 2",
  "elapsed": "3m45s" }
```

## Key Design Decisions

### 1. mcp-go Library

Use `github.com/mark3labs/mcp-go` — same as nexusctl. Provides:
- `server.NewMCPServer()` — protocol handling
- `server.NewStdioServer()` — stdio transport (reads JSON-RPC from stdin)
- `mcp.NewTool()` — typed tool definitions
- `mcp.CallToolRequest` / `mcp.CallToolResult` — request/response types
- `server.SendNotificationToClient()` — streaming notifications

### 2. No New Dependencies Beyond mcp-go

The MCP server calls existing `pkg/` functions. No new abstractions or
service layers needed. The tool handlers are thin wrappers around existing
code, similar to how `cmd/` cobra commands wrap `pkg/` functions.

### 3. Initialization Shares CLI Path

The `anvil mcp-server` command uses the same `root.go` initialization:
`config.InitDirs()`, `config.LoadConfig()`, etc. This ensures the MCP server
sees the same paths and config as the CLI.

### 4. Signing Password via Environment or Session

For non-interactive signing, the password is provided via:
- `ANVIL_SIGNING_PASSWORD` env var (recommended for `.mcp.json`)
- `signing_unlock` tool (accepts password once, stores in server memory for
  the session lifetime)

The session approach is safer — the password lives only in the MCP server's
memory, not in config files.

### 5. Async Builds with In-Memory State

Kernel builds run in background goroutines managed by a `BuildManager`.
Build state lives in server memory — it does not survive server restarts.
Completed build artifacts (kernel binary, `build-stats.json`) persist on
disk and are discoverable via `kernel_list`. `kernel.Build()` already
accepts a `context.Context`, so cancellation via `kernel_build_cancel`
works by calling the stored `CancelFunc`.

## Open Questions

1. **Resource exposure**: Should the MCP server expose MCP Resources (read-only
   data like kernel configs, build stats, archive index) in addition to Tools?
   Resources are useful for agents that want to inspect state without calling
   a tool.

2. **Prompts**: Should we define MCP Prompts (pre-built prompt templates) for
   common workflows like "build and sign a release"? This could simplify
   agent orchestration.

3. **Multi-arch builds**: Should `kernel_build` with `arch=all` run both
   architectures sequentially and return combined stats? Or should the agent
   call it twice?

## Comparison: nexusctl vs Anvil MCP

| Aspect | nexusctl | Anvil (proposed) |
|--------|----------|------------------|
| Transport | stdio-to-HTTP bridge | Direct stdio server |
| Library | mcp-go | mcp-go |
| Daemon | nexus daemon (always running) | No daemon (server is the process) |
| Tools | 24 (CRUD for VMs, drives, etc) | ~31 (build, sign, archive, config, kernel config, firecracker) |
| Streaming | vm_exec output | Build output, progress |
| State | Stateless (daemon holds state) | Session state (signing password) |
| Context | Single (daemon config) | Dual (user XDG / repo anvil.yaml) |
| Long ops | vm_exec (seconds, sync) | kernel_build (minutes, async with build ID) |

## Implementation Order

1. **Phase 1**: Server skeleton + context tools (`set_repo_root`, `get_context`)
   — validates mcp-go integration and establishes the dual-context foundation
2. **Phase 2**: Config/utility tools (quick wins, builds on context)
3. **Phase 3**: Kernel management tools (list, get, set, remove — simple CRUD)
4. **Phase 4**: Kernel build tool with streaming (most complex, needs
   notification plumbing)
5. **Phase 5**: Firecracker tools (version management + acceptance testing)
6. **Phase 6**: Signing tools with session-based password
7. **Phase 7**: Archive tools + `.mcp.json` setup

## References

- nexusctl MCP implementation: `nexus/lead/internal/infra/mcp/`
- nexusctl bridge: `nexus/ctl/mcp_bridge.go`
- mcp-go: `github.com/mark3labs/mcp-go`
- MCP spec: `modelcontextprotocol.io`
- Anvil config system: `pkg/config/`
- Anvil kernel build: `pkg/kernel/build.go`
- Anvil signing: `pkg/signing/signing.go`
