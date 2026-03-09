# Rootfs Domain Hex-Arch Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove `Interactive` flag from `CreateOptions` and inject config paths into
`pkg/rootfs` functions instead of accessing globals.

**Architecture:** `CreateOptions.Interactive` is removed — the caller (CLI or MCP) decides
presentation by providing appropriate callbacks. Config paths are passed as parameters.

**Tech Stack:** Go, libguestfs

---

### Task 1: Remove `Interactive` flag from `CreateOptions`

**Files:**
- Modify: `pkg/rootfs/rootfs.go:30-44`
- Modify: all callers that set `Interactive`

**Step 1: Remove the field**

```go
type CreateOptions struct {
	OutputPath     string
	SizeMB         int
	AlpineVersion  string
	AlpinePatch    string
	Writer         io.Writer
	PhaseCallback  func(CreatePhase)
	StatsCallback  func(CreateStats)
	Context        context.Context
	ForceOverwrite bool
	InjectBinary   bool
	BinaryPath     string
	BinaryDestPath string
}
```

**Step 2: Remove any `opts.Interactive` checks** in `Create()` function

Search for `opts.Interactive` and remove conditional branches. The function already
uses `Writer`, `PhaseCallback`, and `StatsCallback` for all output — the `Interactive`
flag is likely unused or redundant.

**Step 3: Update callers** — remove `Interactive: true/false` assignments

- `cmd/firecracker/create_rootfs.go`
- `internal/mcp/tools_firecracker.go` (handleFirecrackerCreateRootfs)
- `pkg/firecracker/test.go` (rootfs creation during test)

**Step 4: Run tests and commit**

Run: `mise ci`

```bash
git commit -m "refactor: remove Interactive flag from rootfs.CreateOptions

Interactive vs non-interactive is decided by the caller through
callback presence/absence, not a boolean flag."
```

---

### Task 2: Pass paths to `Create()` instead of accessing config global

**Files:**
- Modify: `pkg/rootfs/rootfs.go:115-233`

**Step 1: Add `paths *config.Paths` parameter or use OutputPath directly**

Line 120 accesses `config.GlobalPaths.DataDir` for default output path. Instead,
require `OutputPath` to be set by the caller — remove the default fallback from domain.

```go
func Create(opts CreateOptions) error {
	if opts.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}
	// ... rest of function
}
```

**Step 2: Update callers to provide OutputPath explicitly**

- `cmd/firecracker/create_rootfs.go`: set `OutputPath` from flag or `config.GlobalPaths.DataDir`
- `internal/mcp/tools_firecracker.go`: set `OutputPath` from MCP parameter or default
- `pkg/firecracker/test.go`: set `OutputPath` from paths parameter

**Step 3: Remove `config` import from `pkg/rootfs/`**

**Step 4: Run tests and commit**

```bash
git commit -m "refactor: rootfs.Create requires explicit OutputPath

Callers provide the output path instead of the domain reading from
config globals. Removes config dependency from rootfs package."
```

---

## File Change Summary

| File | Change |
|------|--------|
| `pkg/rootfs/rootfs.go` | Remove `Interactive`, require `OutputPath`, remove config import |
| `cmd/firecracker/create_rootfs.go` | Set OutputPath explicitly |
| `internal/mcp/tools_firecracker.go` | Set OutputPath explicitly |
| `pkg/firecracker/test.go` | Set OutputPath explicitly |
