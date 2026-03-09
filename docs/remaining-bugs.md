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

## 2. CLI `kernel versions` shows empty, MCP `kernel_versions` works

CLI `kernel versions` calls `ShowVersions()` which checks GitHub releases
(empty repo — no releases published). MCP `kernel_versions` calls
`GetLatestKernelVersion()` which checks kernel.org directly. These are
different code paths returning different data.

Not a regression — the GitHub repo has no releases. But the discrepancy
between CLI and MCP is confusing. Consider unifying the version source.

**Severity:** Low (cosmetic/UX discrepancy, not data loss)

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
