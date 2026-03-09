# Firecracker Domain Hex-Arch Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Commit to master. Do not push.

**Goal:** Remove all I/O and config global access from `pkg/firecracker`. Same treatment
as kernel — functions return data, accept paths, never print.

**Architecture:** Follows the pattern established in Phase 3 (kernel). Domain functions
return data structs. CLI formats output. MCP serializes to JSON. 47+ `fmt.Print*` calls
removed from domain.

**Tech Stack:** Go, spf13/cobra, charmbracelet/lipgloss v2

---

### Task 1: Make `List()` return data instead of printing

Currently has 19 `fmt.Print*` calls (lines 145-189) and accesses `config.CurrentTheme`.

**Files:**
- Modify: `pkg/firecracker/firecracker.go:123-193`
- Modify: `cmd/firecracker/list.go`
- Modify: `internal/mcp/tools_firecracker.go`

**Step 1: Define return type**

```go
// FirecrackerInfo describes an installed Firecracker version
type FirecrackerInfo struct {
	Version   string `json:"version"`
	IsDefault bool   `json:"is_default"`
	Path      string `json:"path"`
}
```

**Step 2: Rewrite `List()` to return data**

```go
func List(paths *config.Paths) ([]FirecrackerInfo, error) {
	// Determine default version from symlink
	defaultVersion := ""
	fcSymlink := filepath.Join(paths.BinDir, "firecracker")
	if target, err := os.Readlink(fcSymlink); err == nil {
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "firecracker" && i+1 < len(parts) {
				defaultVersion = parts[i+1]
				break
			}
		}
	}

	entries, err := os.ReadDir(paths.FirecrackerDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read firecracker directory: %w", err)
	}

	var versions []FirecrackerInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := entry.Name()
		versions = append(versions, FirecrackerInfo{
			Version:   version,
			IsDefault: version == defaultVersion,
			Path:      filepath.Join(paths.FirecrackerDir, version, "firecracker"),
		})
	}

	return versions, nil
}
```

**Step 3: Move presentation to `cmd/firecracker/list.go`**

**Step 4: Update MCP handler to delegate to domain**

Replace `handleFirecrackerList`'s direct `os.ReadDir` with `firecracker.List(config.GlobalPaths)`.

**Step 5: Run tests and commit**

Run: `mise ci`

```bash
git commit -m "refactor: firecracker.List returns data instead of printing"
```

---

### Task 2: Make `Set()` accept paths, remove printing

**Files:**
- Modify: `pkg/firecracker/firecracker.go:195-220`
- Modify: `cmd/firecracker/set.go`

**Step 1: Update signature and remove prints**

```go
func Set(version string, paths *config.Paths) error {
	versionDir := filepath.Join(paths.FirecrackerDir, version)
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("firecracker %s not found", version)
	}

	log.Debugf("Setting Firecracker %s as default", version)

	target := filepath.Join(versionDir, "firecracker")
	link := filepath.Join(paths.BinDir, "firecracker")

	os.Remove(link)
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	return nil
}
```

**Step 2: Update all callers** — `cmd/firecracker/set.go`, `cmd/cmdutil/helpers.go`, MCP handler

**Step 3: Run tests and commit**

```bash
git commit -m "refactor: firecracker.Set accepts paths, removes printing"
```

---

### Task 3: Make `ShowVersions()` return data instead of printing

**Files:**
- Modify: `pkg/firecracker/firecracker.go:221-290`
- Modify: `cmd/firecracker/versions.go`

**Step 1: Return structured data**

```go
type AvailableFirecracker struct {
	Version     string `json:"version"`
	IsInstalled bool   `json:"is_installed"`
	IsDefault   bool   `json:"is_default"`
}

func ShowVersions(paths *config.Paths) ([]AvailableFirecracker, error) {
	// ... fetch from GitHub, cross-reference with installed, return data
}
```

**Step 2: Move presentation to CLI**

**Step 3: Run tests and commit**

```bash
git commit -m "refactor: firecracker.ShowVersions returns data instead of printing"
```

---

### Task 4: Make `Download`/`DownloadWithProgress` accept paths

**Files:**
- Modify: `pkg/firecracker/firecracker.go:17-120`

**Step 1: Add `paths *config.Paths` parameter**

Replace `config.GlobalPaths.FirecrackerDir`, `config.GlobalPaths.CacheDir` with `paths.*`.

**Step 2: Remove 7 `fmt.Print*` calls** (lines 111-117)

Return success silently — CLI prints success message.

**Step 3: Update callers** — `cmd/firecracker/get.go`, `cmd/cmdutil/helpers.go`, MCP handler

**Step 4: Run tests and commit**

```bash
git commit -m "refactor: firecracker download functions accept paths, remove printing"
```

---

### Task 5: Make `Test()` accept paths

**Files:**
- Modify: `pkg/firecracker/test.go`

**Step 1: Add `paths *config.Paths` parameter to `Test()`**

```go
func Test(opts TestOptions, paths *config.Paths) (*TestResult, error) {
```

**Step 2: Replace config global access**

Lines 72, 294, 306, 327, 337 — replace `config.GlobalPaths.DataDir` with `paths.DataDir`.

**Step 3: Update private helpers** — `getKernelPath`, `getFirecrackerBinary`, `createTestConfig`

Pass `paths` through to each.

**Step 4: Update callers** — `cmd/firecracker/test.go`, MCP handler

**Step 5: Update existing tests** in `test_internal_test.go`

The 3 existing tests override `config.GlobalPaths.DataDir` — update them to pass `paths`
as a parameter instead.

**Step 6: Run tests and commit**

Run: `mise ci`

```bash
git commit -m "refactor: firecracker.Test accepts paths parameter"
```

---

### Task 6: Remove `config.CurrentTheme` from firecracker package

After Tasks 1 and 3, `List()` and `ShowVersions()` no longer render.
Remove the `config.CurrentTheme` references (lines 124, 222) and clean imports.

```bash
git commit -m "refactor: remove theme dependency from firecracker domain"
```

---

## File Change Summary

| File | Change |
|------|--------|
| `pkg/firecracker/firecracker.go` | All functions accept paths, return data, remove 47+ prints |
| `pkg/firecracker/test.go` | `Test()` accepts paths |
| `pkg/firecracker/test_internal_test.go` | Update tests to pass paths |
| `cmd/firecracker/list.go` | Format and print firecracker list |
| `cmd/firecracker/set.go` | Print success after domain call |
| `cmd/firecracker/versions.go` | Format and print available versions |
| `cmd/firecracker/get.go` | Pass paths to download |
| `cmd/firecracker/test.go` | Pass paths to Test() |
| `cmd/cmdutil/helpers.go` | Pass paths to firecracker calls |
| `internal/mcp/tools_firecracker.go` | Delegate to domain, pass paths |
