# CLI & MCP Naming Alignment Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Align CLI command names and MCP tool names before the initial AUR release. The CLI is the user-facing contract; MCP tool names mirror CLI hierarchy with underscores replacing spaces.

**Design Doc:** `docs/cli-mcp-naming-alignment.md`

**Architecture:** Two phases — Phase 1 renames existing tools, Phase 2 adds missing MCP tools for CLI parity. All MCP handlers follow the existing pattern: call `pkg/` functions directly, return JSON via `jsonResult()` / `errResult()`.

---

## Phase 1: Renames (COMPLETED)

These tasks have been implemented and verified with `mise ci`.

### Task 1: Move `build-kernel` → `kernel build` ✅

**Files modified:**
- `cmd/buildkernel/buildkernel.go` — `Use: "build [version]"`, removed `Aliases`
- `cmd/kernel/kernel.go` — added `cmd.AddCommand(buildkernel.NewBuildKernelCmd())`
- `cmd/root.go` — hidden backwards-compat alias with `Use: "build-kernel [version]"`, `Hidden: true`
- `cmd/init/init.go` — updated help text to `anvil kernel build`
- `pkg/kernel/build.go` — updated error message to `anvil kernel build`

### Task 2: Rename `clean build-kernel` → `clean build` ✅

**Files modified:**
- `cmd/clean/buildkernel.go` — `Use: "build"`, `Aliases: []string{"builds", "build-kernel"}`
- `cmd/clean/clean.go` — updated comment

### Task 3: Rename MCP tools ✅

**Files modified:**
- `internal/mcp/tools_kernel_build.go` — `kernel_list_versions` → `kernel_versions`, `kernel_validate_version` → `kernel_version_check`
- `internal/mcp/tools_kernel_mgmt.go` — `kernel_get` → `kernel_info`
- `internal/mcp/tools_signing.go` — `signing_generate_key` → `signing_generate`, `signing_rotate_key` → `signing_rotate`, `signing_export_backup` → `signing_export`, `signing_import_backup` → `signing_import`
- `internal/mcp/tools_utility.go` — `clean_build_cache` → `clean_build`

### Task 4: Add CLI references to MCP descriptions ✅

Added `CLI: anvil <command>` suffix to all MCP tool descriptions that have CLI equivalents. MCP-only tools omit the reference.

### Task 5: Fix MCP stats path bug ✅

**Files modified:**
- `internal/mcp/tools_kernel_mgmt.go` — `kernel_install` handler: `build/` → `artifacts/`
- `internal/mcp/tools_archive.go` — `archive_kernel` handler: `build/` → `artifacts/`

---

## Phase 2: New MCP tools for CLI parity

Six MCP tools need to be added so every CLI clean/signing subcommand has an MCP equivalent. All new tools go in existing registration files and follow the established pattern.

### Task 6: Add `signing_list` MCP tool

**File:** `internal/mcp/tools_signing.go`

**What:** Add a read-only MCP tool that lists all signing keys in the local keyring.

**Registration:**
```go
s.AddTool(gomcp.NewTool("signing_list",
    gomcp.WithDescription("List all signing keys in the local keyring. CLI: anvil signing list"),
    gomcp.WithReadOnlyHintAnnotation(true),
), handleSigningList)
```

**Handler:** Call `signing.ListKeys()`, return JSON array of key objects with fields: `key_id`, `fingerprint`, `name`, `email`, `created`, `expires`.

**Reference:** The existing `handleSigningKeyInfo` handler (same file) already calls `signing.ListKeys()` and formats the response — `signing_list` should return the same structure. Consider whether `signing_key_info` is now redundant (it returns identical data). If so, remove `signing_key_info` and keep only `signing_list`.

**Verify:** `mise ci` passes.

---

### Task 7: Add `signing_check_expiry` MCP tool

**File:** `internal/mcp/tools_signing.go`

**What:** Add a read-only MCP tool that checks if any signing keys are expiring soon.

**Registration:**
```go
s.AddTool(gomcp.NewTool("signing_check_expiry",
    gomcp.WithDescription("Check if signing keys are expiring soon. CLI: anvil signing check-expiry"),
    gomcp.WithNumber("days", gomcp.Description("Warn if key expires within this many days (default: 60)")),
    gomcp.WithReadOnlyHintAnnotation(true),
), handleSigningCheckExpiry)
```

**Handler logic:**
1. Call `signing.ListKeys()`
2. Check each key's `Expires` field against `time.Now()` + `days` parameter
3. Return JSON with: `expiring` (array of keys expiring soon), `expired` (array of already-expired keys), `all_valid` (bool)

**Reference:** CLI implementation at `cmd/signing/check_expiry.go` — replicate the `now.AddDate(0, 0, days)` logic.

**Verify:** `mise ci` passes.

---

### Task 8: Add `signing_remove` MCP tool

**File:** `internal/mcp/tools_signing.go`

**What:** Add a destructive MCP tool that removes the signing key.

**Registration:**
```go
s.AddTool(gomcp.NewTool("signing_remove",
    gomcp.WithDescription("Remove the signing key from the local keyring. CLI: anvil signing remove"),
    gomcp.WithDestructiveHintAnnotation(true),
), handleSigningRemove)
```

**Handler:** Call `signing.RemoveKey()`. Return `{"status": "removed"}`.

**Note:** `signing.RemoveKey()` does not take a key ID parameter — it removes the current signing key. The CLI at `cmd/signing/remove.go` accepts `[key-id]` as an argument but does not actually pass it to `RemoveKey()`. The MCP tool should match the actual `pkg/` behavior (no key ID parameter).

**Verify:** `mise ci` passes.

---

### Task 9: Add `clean_kernel` MCP tool

**File:** `internal/mcp/tools_utility.go`

**What:** Add a destructive MCP tool that removes installed kernel versions.

**Registration:**
```go
s.AddTool(gomcp.NewTool("clean_kernel",
    gomcp.WithDescription("Remove installed kernel versions. CLI: anvil clean kernel"),
    gomcp.WithBoolean("all", gomcp.Description("Remove ALL kernel data including default (default: false, removes only non-default)")),
    gomcp.WithDestructiveHintAnnotation(true),
), handleCleanKernel)
```

**Handler logic (replicate from `cmd/clean/clean.go`):**
1. Read `config.GlobalPaths.KernelsDir`
2. Determine default kernel version via symlink at `kernels/default`
3. If `all` is false: remove only non-default kernel directories
4. If `all` is true: remove entire `KernelsDir` and kernel symlink
5. Return JSON with `removed` (list of removed versions) and `count`

**Note:** The CLI version uses interactive confirmation prompts (`ui.Confirm`, `ui.TypedConfirm`). MCP tools skip confirmation — the MCP client (Claude) is responsible for confirming with the user before calling destructive tools. The `DestructiveHintAnnotation` signals this.

**Reference:** `cmd/clean/clean.go` functions `cleanInactiveKernels()` (lines 109-180) and `cleanAllKernels()` (lines 250-301).

**Verify:** `mise ci` passes.

---

### Task 10: Add `clean_firecracker` MCP tool

**File:** `internal/mcp/tools_utility.go`

**What:** Add a destructive MCP tool that removes installed Firecracker versions.

**Registration:**
```go
s.AddTool(gomcp.NewTool("clean_firecracker",
    gomcp.WithDescription("Remove installed Firecracker versions. CLI: anvil clean firecracker"),
    gomcp.WithBoolean("all", gomcp.Description("Remove ALL Firecracker data including default (default: false, removes only non-default)")),
    gomcp.WithDestructiveHintAnnotation(true),
), handleCleanFirecracker)
```

**Handler logic (replicate from `cmd/clean/clean.go`):**
1. Read `config.GlobalPaths.FirecrackerDir`
2. Determine default version via symlink
3. If `all` is false: remove only non-default version directories
4. If `all` is true: remove entire `FirecrackerDir` and firecracker symlink in `BinDir`
5. Return JSON with `removed` (list) and `count`

**Reference:** `cmd/clean/clean.go` functions `cleanInactiveFirecracker()` (lines 182-248) and `cleanAllFirecracker()` (lines 304-354).

**Verify:** `mise ci` passes.

---

### Task 11: Add `clean_rootfs` MCP tool

**File:** `internal/mcp/tools_utility.go`

**What:** Add a destructive MCP tool that removes rootfs images.

**Registration:**
```go
s.AddTool(gomcp.NewTool("clean_rootfs",
    gomcp.WithDescription("Remove rootfs images. CLI: anvil clean rootfs"),
    gomcp.WithDestructiveHintAnnotation(true),
), handleCleanRootfs)
```

**Handler logic (replicate from `cmd/clean/clean.go`):**
1. Read `config.GlobalPaths.DataDir`
2. Find all `.ext4` files (rootfs images)
3. Remove them
4. Return JSON with `removed` (list of filenames) and `count`

**Reference:** `cmd/clean/clean.go` function `cleanRootfs()` (lines 439-484).

**Verify:** `mise ci` passes.

---

### Task 12: Final verification

**Step 1:** Run `mise ci` — all lint, vet, staticcheck, and tests must pass.

**Step 2:** Verify complete alignment by listing all MCP tools:
```bash
grep 'gomcp.NewTool(' internal/mcp/tools_*.go | sed 's/.*NewTool("\(.*\)".*/\1/' | sort
```

**Step 3:** Cross-reference against CLI commands:
```bash
anvil --help           # root commands
anvil kernel --help    # kernel subcommands
anvil signing --help   # signing subcommands
anvil firecracker --help
anvil clean --help
anvil config --help
```

**Expected alignment:**

| CLI Command | MCP Tool | Notes |
|---|---|---|
| `kernel build` | `kernel_build` | |
| `kernel list` | `kernel_list` | |
| `kernel versions` | `kernel_versions` | |
| `kernel version-check` | `kernel_version_check` | |
| `kernel remove` | `kernel_remove` | |
| `kernel set` | `kernel_set_default` | MCP keeps `_default` for clarity |
| `kernel get` | *(no MCP equivalent)* | CLI downloads from GitHub releases |
| `signing generate` | `signing_generate` | |
| `signing rotate` | `signing_rotate` | |
| `signing sign` | `signing_sign` | |
| `signing verify` | `signing_verify` | |
| `signing export` | `signing_export` | |
| `signing import` | `signing_import` | |
| `signing list` | `signing_list` | **NEW in Phase 2** |
| `signing check-expiry` | `signing_check_expiry` | **NEW in Phase 2** |
| `signing remove` | `signing_remove` | **NEW in Phase 2** |
| `firecracker test` | `firecracker_test` | |
| `firecracker get` | `firecracker_get` | |
| `firecracker list` | `firecracker_list` | |
| `firecracker versions` | `firecracker_versions` | |
| `firecracker remove` | `firecracker_remove` | |
| `firecracker set` | `firecracker_set_default` | MCP keeps `_default` for clarity |
| `firecracker create-rootfs` | `firecracker_create_rootfs` | |
| `clean build` | `clean_build` | |
| `clean kernel` | `clean_kernel` | **NEW in Phase 2** |
| `clean firecracker` | `clean_firecracker` | **NEW in Phase 2** |
| `clean rootfs` | `clean_rootfs` | **NEW in Phase 2** |
| `config get` | `config_get` | |
| `config set` | `config_set` | |
| `config list` | `config_list` | |

MCP-only tools (no CLI equivalent): `kernel_info`, `kernel_install`, `kernel_build_status`, `kernel_build_log`, `kernel_build_wait`, `kernel_build_cancel`, `kernel_config_get`, `kernel_config_set`, `kernel_config_list`, `kernel_config_diff`, `archive_kernel`, `archive_list`, `archive_get`, `config_get_paths`, `get_context`, `set_repo_root`, `set_user_mode`, `check_build_tools`

CLI-only commands (no MCP equivalent): `kernel get` (download), `signing import-key`, `config unset`, `config schema`, `init`, `update`, `version`, `vsock`, `completion`

**Step 4:** Rebuild and install:
```bash
mise run install:local
```
