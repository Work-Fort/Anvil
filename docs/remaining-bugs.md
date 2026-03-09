# Remaining Bugs

Bugs found during post-hex-arch QA pass (2026-03-09).

## 1. MCP `config_get` returns stale value after `config_set`

`config_set log-level info` writes to `anvil.yaml` successfully (CLI confirms
the new value), but immediately calling `config_get log-level` via MCP returns
the old value (`debug`). The MCP server reads from a cached viper state that
doesn't reflect the file write.

**Reproduce:**
1. `config_set` key=`log-level` value=`info` → reports success
2. `config_get` key=`log-level` → returns `debug` (stale)
3. CLI `anvil config get log-level` → returns `info` (correct)

**Impact:** MCP consumers can't verify config changes they just made.

## ~~2. CLI `kernel versions` shows empty, MCP `kernel_versions` works~~ Not a bug

CLI `kernel versions` shows GitHub releases (pre-built kernel binaries).
MCP `kernel_versions` shows kernel.org versions (source available to build).
These are intentionally different: CLI is for downloading pre-built, MCP is
for checking what's buildable. No GitHub releases published yet = correct.

## 3. MCP server running stale binary after rebuild

After rebuilding with `mise run build` and `mise run install:local`, the MCP
server process continues using the old binary. MCP tools like
`firecracker_versions` and `firecracker_list` return stale results (static
help string, wrong `is_default` values) that don't match CLI output.

CLI tests pass correctly with the new binary. MCP tests that fail here are
expected to pass after the MCP server restarts with the freshly built binary.

**Not a code bug** — operational issue. The MCP server needs to be restarted
after binary rebuilds to pick up changes.

**Affected MCP tests in this QA pass:**
- `firecracker_versions` returned static help string instead of version list
- `firecracker_list` returned `is_default: false` for the default version

## 4. MCP `kernel_build_wait` returns zeroed stats

`kernel_build_wait` returns `status: completed` but all stats fields are zero.
The build-stats.json file on disk has correct data (durations, sizes, hashes).
The build manager isn't reading or forwarding the stats file to the response.

**Reproduce:**
1. `kernel_build` version=6.19.6 arch=x86_64 → returns build_id
2. `kernel_build_wait` build_id=`<id>` → stats all zero
3. `cat ~/.cache/anvil/build-kernel/artifacts/build-stats.json` → correct data

**Impact:** MCP consumers can't get build stats without reading the file directly.
