# CLI & MCP Naming Alignment

Aligns CLI command names and MCP tool names before the initial AUR release.
The CLI is the user-facing contract; MCP tool names mirror CLI hierarchy with
underscores replacing spaces.

## Design Principles

1. **Resource-oriented hierarchy.** Commands group under the resource they
   operate on: `kernel build`, `kernel list`, `signing sign`.
2. **CLI is the source of truth.** MCP tool names derive from CLI paths:
   `anvil kernel build` → `kernel_build`.
3. **Clean is a separate top-level command.** Destructive operations stay
   explicit and deliberate — never hidden as flags.
4. **MCP-only tools use the same naming convention.** Tools without CLI
   equivalents still follow `resource_action` naming.

## CLI Changes

### `build-kernel` → `kernel build`

Move from root-level to `kernel` subcommand. This is the primary structural
change.

| Before | After |
|--------|-------|
| `anvil build-kernel` | `anvil kernel build` |
| `anvil build-kernel 6.19.6` | `anvil kernel build 6.19.6` |
| `anvil build-kernel --arch aarch64` | `anvil kernel build --arch aarch64` |

Keep `build-kernel` and `build` as hidden aliases for backwards compatibility
during the transition.

### `clean` subcommands

Simplify names now that `build-kernel` is under `kernel`:

| Before | After |
|--------|-------|
| `anvil clean build-kernel` | `anvil clean build` |
| `anvil clean kernel` | `anvil clean kernel` |
| `anvil clean firecracker` | `anvil clean firecracker` |
| `anvil clean rootfs` | `anvil clean rootfs` |

### Everything else stays

The rest of the CLI hierarchy is already clean:

```
anvil kernel build          Build kernel from source (kernel.org)
anvil kernel get            Download pre-built kernel (GitHub releases)
anvil kernel list           List installed kernels
anvil kernel set            Set default kernel version
anvil kernel remove         Remove an installed kernel
anvil kernel versions       Show available kernel versions
anvil kernel version-check  Check if a kernel version is buildable

anvil signing generate      Generate a new PGP signing key
anvil signing rotate        Rotate the signing key
anvil signing sign          Sign release artifacts
anvil signing verify        Verify release artifacts signature
anvil signing export        Export encrypted key backup
anvil signing import        Import key from encrypted backup
anvil signing import-key    Import a public key
anvil signing list          List all signing keys
anvil signing remove        Remove a signing key
anvil signing check-expiry  Check if keys are expiring soon

anvil firecracker get           Download a Firecracker binary
anvil firecracker list          List installed Firecracker versions
anvil firecracker set           Set default Firecracker version
anvil firecracker remove        Remove an installed Firecracker version
anvil firecracker versions      Show available Firecracker versions
anvil firecracker test          Test Firecracker with vsock
anvil firecracker create-rootfs Create an Alpine-based rootfs

anvil clean build           Clean kernel build cache (source, artifacts)
anvil clean kernel          Remove installed kernels
anvil clean firecracker     Remove installed Firecracker binaries
anvil clean rootfs          Clean rootfs images

anvil config get / set / list / unset / schema
anvil init
anvil update
anvil version
```

## MCP Tool Renames

### Naming convention

`anvil <resource> <action>` → `<resource>_<action>`

### Description convention

Every MCP tool description that has a CLI equivalent must include it on the
first line. Format:

```
"Description of what the tool does. CLI: anvil <command> <subcommand>"
```

Example:
```go
Name:        "kernel_build",
Description: "Start a kernel build from source. CLI: anvil kernel build",
```

MCP-only tools (no CLI equivalent) omit the CLI reference.

### Renames

| Before | After | Reason |
|--------|-------|--------|
| `kernel_build` | `kernel_build` | Already aligned |
| `kernel_get` | `kernel_info` | Frees `kernel_get` for download; `info` matches "show details" |
| `kernel_list_versions` | `kernel_versions` | Matches CLI `kernel versions` |
| `kernel_validate_version` | `kernel_version_check` | Matches CLI `kernel version-check` |
| `kernel_set_default` | `kernel_set_default` | Keep explicit — MCP benefits from clarity |
| `signing_generate_key` | `signing_generate` | Matches CLI `signing generate` |
| `signing_rotate_key` | `signing_rotate` | Matches CLI `signing rotate` |
| `signing_export_backup` | `signing_export` | Matches CLI `signing export` |
| `signing_import_backup` | `signing_import` | Matches CLI `signing import` |
| `signing_key_info` | `signing_key_info` | MCP-only, name is clear |
| `firecracker_set_default` | `firecracker_set_default` | Keep explicit |
| `firecracker_create_rootfs` | `firecracker_create_rootfs` | Already aligned |
| `clean_build_cache` | `clean_build` | Matches CLI `clean build` |

### Unchanged MCP tools

These already match or are MCP-only with clear names:

- `kernel_list`, `kernel_remove`, `kernel_install`
- `kernel_build_status`, `kernel_build_log`, `kernel_build_wait`, `kernel_build_cancel`
- `kernel_config_get`, `kernel_config_set`, `kernel_config_list`, `kernel_config_diff`
- `signing_sign`, `signing_verify`
- `firecracker_get`, `firecracker_list`, `firecracker_remove`, `firecracker_versions`, `firecracker_test`
- `archive_kernel`, `archive_list`, `archive_get`
- `config_get`, `config_set`, `config_list`, `config_get_paths`
- `get_context`, `set_repo_root`, `set_user_mode`
- `check_build_tools`

### New MCP tools (to add for parity)

| MCP Tool | CLI Equivalent |
|----------|---------------|
| `signing_list` | `signing list` |
| `signing_check_expiry` | `signing check-expiry` |
| `signing_remove` | `signing remove` |
| `clean_kernel` | `clean kernel` |
| `clean_firecracker` | `clean firecracker` |
| `clean_rootfs` | `clean rootfs` |

## Implementation Order

1. Move `build-kernel` → `kernel build` (CLI only, keep hidden alias)
2. Rename `clean build-kernel` → `clean build`
3. Rename MCP tools (server.go tool registrations)
4. Fix MCP `kernel_install` and `archive_kernel` stats path bug
5. Update `.mcp.json` tool descriptions if needed
6. Run `mise ci`
