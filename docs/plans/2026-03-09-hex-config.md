# Config Foundation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix I/O leaks in `pkg/config`, make `Paths` and `Theme` passable as parameters,
and establish the dependency injection pattern that all subsequent phases follow.

**Architecture:** Replace `config.GlobalPaths` and `config.CurrentTheme` globals with
parameters threaded from entry points (CLI root, MCP server). Keep backwards compatibility
during migration — globals stay but are deprecated. Each subsequent domain phase updates
its own functions to accept parameters instead of reading globals.

**Tech Stack:** Go, spf13/viper, spf13/cobra, charmbracelet/lipgloss v2

---

### Task 1: Fix `GetPaths()` error handling

`GetPaths()` calls `fmt.Fprintf(os.Stderr)` and `os.Exit(1)` three times — I/O in domain.

**Files:**
- Modify: `pkg/config/config.go:53-101`

**Step 1: Change `GetPaths` to return error**

```go
func GetPaths() (*Paths, error) {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory for XDG_DATA_HOME: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory for XDG_CACHE_HOME: %w", err)
		}
		cacheHome = filepath.Join(home, ".cache")
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory for XDG_CONFIG_HOME: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}

	dataDir := filepath.Join(dataHome, "anvil")
	cacheDir := filepath.Join(cacheHome, "anvil")
	configDir := filepath.Join(configHome, "anvil")
	binDir := filepath.Join(dataDir, "bin")

	return &Paths{
		DataDir:        dataDir,
		CacheDir:       cacheDir,
		ConfigDir:      configDir,
		BinDir:         binDir,
		KernelsDir:     filepath.Join(dataDir, "kernels"),
		FirecrackerDir: filepath.Join(dataDir, "firecracker"),
		KernelBuildDir: filepath.Join(cacheDir, "build-kernel"),
		KeysDir:        filepath.Join(dataDir, "keys"),
		GnupgDir:       filepath.Join(dataDir, "gnupg"),
	}, nil
}
```

**Step 2: Update `init()` to handle the error**

```go
func init() {
	paths, err := GetPaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	GlobalPaths = paths
}
```

The `init()` still does stderr+exit, but that's the CLI entry point — acceptable there.
The important change is that `GetPaths()` itself is now a pure function that returns errors.

**Step 3: Remove `fmt` and `os` imports if no longer needed in GetPaths**

The `fmt` and `os` imports are still needed by `init()` and other functions in the file.
No import changes needed.

**Step 4: Run tests**

Run: `mise ci`
Expected: All tests pass. `GetPaths()` callers that ignored the single return value will
now get a compile error — fix any that exist.

**Step 5: Commit**

```bash
git add pkg/config/config.go
git commit -m "refactor: make GetPaths return error instead of os.Exit

GetPaths() no longer writes to stderr or calls os.Exit(1). It returns
an error that callers can handle. The init() function still exits on
error since it runs at program startup."
```

---

### Task 2: Create `InitPaths()` for explicit initialization

Currently `GlobalPaths` is set in `init()` — there's no way to initialize paths
after program startup (needed for MCP `set_repo_root` which changes paths).

**Files:**
- Modify: `pkg/config/config.go`

**Step 1: Add `InitPaths` function**

```go
// InitPaths initializes GlobalPaths explicitly. Call this instead of relying on init()
// when you need to reinitialize paths (e.g., after changing mode).
func InitPaths() error {
	paths, err := GetPaths()
	if err != nil {
		return fmt.Errorf("failed to initialize paths: %w", err)
	}
	GlobalPaths = paths
	return nil
}
```

**Step 2: Run tests**

Run: `mise ci`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/config/config.go
git commit -m "feat: add InitPaths for explicit path initialization"
```

---

### Task 3: Document the dependency injection pattern

Create a short reference document that subsequent phases follow.

**Files:**
- Create: `docs/hex-arch-pattern.md`

**Step 1: Write the pattern guide**

```markdown
# Hexagonal Architecture: Dependency Injection Pattern

## Rule

Domain functions accept data as parameters. They never:
- Access `config.GlobalPaths` directly
- Access `config.CurrentTheme` directly
- Call `config.Get*()` functions
- Print to stdout/stderr
- Read from stdin
- Launch TUI prompts

## How to Thread Dependencies

**CLI commands** (`cmd/`): Read from config globals, then pass as parameters.

```go
// cmd/kernel/list.go — CLI adapter
func runList(cmd *cobra.Command, args []string) error {
    return kernel.List(config.GlobalPaths, config.CurrentTheme)
}
```

**MCP handlers** (`internal/mcp/`): Read from config globals, pass as parameters.

```go
// internal/mcp/tools_kernel_mgmt.go — MCP adapter
func handleKernelList(...) {
    result, err := kernel.List(config.GlobalPaths)
    return jsonResult(result)
}
```

**Domain functions** (`pkg/`): Accept what they need, return data.

```go
// pkg/kernel/kernel.go — domain
func List(paths *config.Paths) ([]KernelInfo, error) {
    entries, _ := os.ReadDir(paths.KernelsDir)
    // ... return data, no printing
}
```

## Presentation vs Domain

| Concern | Where | Example |
|---------|-------|---------|
| Password acquisition | CLI adapter | `cmd/signing/sign.go` |
| Theme styling | CLI adapter | `cmd/kernel/list.go` |
| Progress display | CLI adapter | `cmd/buildkernel/buildkernel.go` |
| JSON formatting | MCP adapter | `internal/mcp/tools_signing.go` |
| Business logic | Domain | `pkg/signing/signing.go` |
| Data validation | Domain | `pkg/config/schema.go` |

## Functions That Return Data (Not Print)

After refactoring, domain functions return structured data:

```go
// Before (prints to stdout):
func List() error

// After (returns data):
func List(paths *config.Paths) ([]KernelInfo, error)
```

The CLI adapter formats and prints. The MCP adapter serializes to JSON.
```

**Step 2: Commit**

```bash
git add docs/hex-arch-pattern.md
git commit -m "docs: add hexagonal architecture dependency injection pattern guide"
```

---

## Scope Boundary

This phase does NOT:
- Remove `GlobalPaths` or `CurrentTheme` globals (they stay for backwards compat)
- Update domain packages to accept parameters (that's Phases 3-8)
- Move Theme to a separate package (happens in Phase 5)
- Create formal port interfaces (YAGNI — Go's implicit interfaces mean domains
  can define small interfaces at the point of use when needed)

## File Change Summary

| File | Change |
|------|--------|
| `pkg/config/config.go` | `GetPaths()` returns `(*Paths, error)`, add `InitPaths()` |
| `docs/hex-arch-pattern.md` | New — DI pattern guide for subsequent phases |
