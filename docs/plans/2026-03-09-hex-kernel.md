# Kernel Domain Hex-Arch Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Commit to master. Do not push.

**Goal:** Remove all I/O and config global access from `pkg/kernel`, making domain functions
pure: they accept paths/config as parameters, return structured data, and never print.

**Architecture:** Domain functions return data structs instead of printing. CLI commands
format and display the data. MCP handlers serialize to JSON. The `Interactive` flag is
removed from `BuildOptions` — the CLI decides presentation, not the domain.

**Tech Stack:** Go, spf13/cobra, charmbracelet/lipgloss v2, gopenpgp/v3

---

### Task 1: Make `List()` return data instead of printing

Currently `kernel.List()` prints a styled table to stdout. It should return data.

**Files:**
- Modify: `pkg/kernel/kernel.go:262-340`
- Modify: `cmd/kernel/list.go`
- Modify: `internal/mcp/tools_kernel_mgmt.go` (handleKernelList)

**Step 1: Define `KernelInfo` return type**

Add near top of `pkg/kernel/kernel.go`:
```go
// KernelInfo describes an installed kernel version
type KernelInfo struct {
	Version   string   `json:"version"`
	IsDefault bool     `json:"is_default"`
	Files     []string `json:"files"`
}
```

**Step 2: Rewrite `List()` to return data**

```go
// List returns installed kernel versions with their metadata.
func List(paths *config.Paths) ([]KernelInfo, string, error) {
	arch, err := config.GetArch()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get architecture: %w", err)
	}
	kernelName, err := config.GetKernelName()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get kernel name: %w", err)
	}

	// Determine default version from symlink
	defaultVersion := ""
	kernelSymlink := filepath.Join(paths.DataDir, kernelName)
	if target, err := os.Readlink(kernelSymlink); err == nil {
		parts := strings.Split(target, "/")
		for i, part := range parts {
			if part == "kernels" && i+1 < len(parts) {
				defaultVersion = parts[i+1]
				break
			}
		}
	}

	entries, err := os.ReadDir(paths.KernelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, arch, nil
		}
		return nil, arch, fmt.Errorf("failed to read kernels directory: %w", err)
	}

	var kernels []KernelInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		version := entry.Name()
		ki := KernelInfo{
			Version:   version,
			IsDefault: version == defaultVersion,
		}

		// List files in version directory
		versionDir := filepath.Join(paths.KernelsDir, version)
		files, err := os.ReadDir(versionDir)
		if err == nil {
			for _, f := range files {
				ki.Files = append(ki.Files, f.Name())
			}
		}
		kernels = append(kernels, ki)
	}

	return kernels, arch, nil
}
```

**Step 3: Update `cmd/kernel/list.go` to format and print**

The CLI command calls `kernel.List(config.GlobalPaths)`, then formats the result
with theme styling and prints it. Move all the `fmt.Printf` presentation logic here.

**Step 4: Update `handleKernelList` in MCP to call domain**

Replace the direct `os.ReadDir` + symlink parsing with `kernel.List(config.GlobalPaths)`.

**Step 5: Run tests**

Run: `mise ci`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/kernel/kernel.go cmd/kernel/list.go internal/mcp/tools_kernel_mgmt.go
git commit -m "refactor: kernel.List returns data instead of printing

List() now returns []KernelInfo instead of printing to stdout.
CLI command formats with theme. MCP handler delegates to domain
instead of reading filesystem directly."
```

---

### Task 2: Make `Set()` accept paths parameter, remove printing

**Files:**
- Modify: `pkg/kernel/kernel.go:344-378`
- Modify: `cmd/kernel/set.go`

**Step 1: Update `Set()` signature and remove fmt.Print calls**

```go
func Set(version string, paths *config.Paths) error {
	arch, err := config.GetArch()
	if err != nil {
		return fmt.Errorf("failed to get architecture: %w", err)
	}
	kernelName, err := config.GetKernelName()
	if err != nil {
		return fmt.Errorf("failed to get kernel name: %w", err)
	}

	kernelPath := filepath.Join(paths.KernelsDir, version, kernelName)
	if _, err := os.Stat(kernelPath); os.IsNotExist(err) {
		return fmt.Errorf("kernel %s not found for %s", version, arch)
	}

	log.Debugf("Setting kernel %s as default", version)

	target := filepath.Join(paths.KernelsDir, version, kernelName)
	link := filepath.Join(paths.DataDir, kernelName)

	os.Remove(link) // Remove existing symlink
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("failed to set default: %w", err)
	}

	return nil
}
```

**Step 2: Move presentation to CLI command**

`cmd/kernel/set.go` prints success message after calling `kernel.Set(version, config.GlobalPaths)`.

**Step 3: Update all callers** — `cmdutil/helpers.go`, `internal/mcp/tools_kernel_mgmt.go`

Pass `config.GlobalPaths` to `kernel.Set()`.

**Step 4: Run tests and commit**

Run: `mise ci`

```bash
git add pkg/kernel/kernel.go cmd/kernel/set.go cmd/cmdutil/helpers.go internal/mcp/tools_kernel_mgmt.go
git commit -m "refactor: kernel.Set accepts paths, removes stdout printing"
```

---

### Task 3: Make `ShowVersions()` return data instead of printing

**Files:**
- Modify: `pkg/kernel/kernel.go:381-460`
- Modify: `cmd/kernel/versions.go`

**Step 1: Define return type and rewrite**

```go
// AvailableVersion describes a kernel version available for download
type AvailableVersion struct {
	Version     string `json:"version"`
	IsInstalled bool   `json:"is_installed"`
	IsDefault   bool   `json:"is_default"`
}

// ShowVersions returns available kernel versions from GitHub.
func ShowVersions(paths *config.Paths) ([]AvailableVersion, error) {
	client := github.NewClient()
	releases, err := client.GetReleases(config.KernelRepo, 20)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	// ... build []AvailableVersion, return data
}
```

**Step 2: Move presentation to `cmd/kernel/versions.go`**

**Step 3: Run tests and commit**

```bash
git commit -m "refactor: kernel.ShowVersions returns data instead of printing"
```

---

### Task 4: Make `Download`/`DownloadWithProgress` accept paths

**Files:**
- Modify: `pkg/kernel/kernel.go:45-260`

**Step 1: Add `paths *config.Paths` parameter to both functions**

Remove direct `config.GlobalPaths` access (lines 76, 92, 114, 127, 140).
Pass `paths` from callers.

**Step 2: Remove `fmt.Println` calls** (lines 26-28, 241-247)

Return a result struct or let callers handle messaging.

**Step 3: Update callers** — `cmd/kernel/get.go`, `cmd/cmdutil/helpers.go`, `internal/mcp/`

**Step 4: Run tests and commit**

```bash
git commit -m "refactor: kernel download functions accept paths parameter"
```

---

### Task 5: Remove `Interactive` flag from `BuildOptions`

**Files:**
- Modify: `pkg/kernel/build.go:37-48`
- Modify: `cmd/buildkernel/buildkernel.go`
- Modify: `cmd/kernel/get.go`

**Step 1: Remove `Interactive` field from `BuildOptions`**

```go
type BuildOptions struct {
	Version           string
	Arch              string
	VerificationLevel string
	ConfigFile        string
	Writer            io.Writer
	ProgressCallback  func(float64)
	PhaseCallback     func(BuildPhase)
	StatsCallback     func(BuildStats)
	Context           context.Context
}
```

**Step 2: Move interactive decision to CLI**

In `cmd/buildkernel/buildkernel.go`, the decision to run the wizard vs direct build
stays in the CLI layer — it was already there. Just remove the `Interactive: true/false`
assignments.

In `cmd/kernel/get.go`, same — the interactive check is already in the CLI command.

**Step 3: Remove any `opts.Interactive` checks in `pkg/kernel/build.go`**

Search for `opts.Interactive` and remove conditional branches.

**Step 4: Run tests and commit**

```bash
git commit -m "refactor: remove Interactive flag from BuildOptions

Interactive vs non-interactive is a presentation decision made by the
CLI adapter, not a domain concern."
```

---

### Task 6: Make `Build()` accept paths, remove config global access

**Files:**
- Modify: `pkg/kernel/build.go`

**Step 1: Add paths parameter**

```go
func Build(opts BuildOptions, paths *config.Paths) error {
```

**Step 2: Replace all `config.GlobalPaths.*` with `paths.*`**

Lines 181-182, 412, 425, 511, 547 — replace `config.GlobalPaths.KernelBuildDir`,
`config.GlobalPaths.KernelsDir`, `config.GlobalPaths.DataDir` with `paths.KernelBuildDir`,
`paths.KernelsDir`, `paths.DataDir`.

**Step 3: Update callers** — `cmd/buildkernel/`, `cmd/kernel/get.go`, MCP build handler

**Step 4: Run tests and commit**

```bash
git commit -m "refactor: kernel.Build accepts paths parameter"
```

---

### Task 7: Make remaining functions accept paths

**Files:**
- Modify: `pkg/kernel/build.go` — `CheckKernelInstalled`, `CheckCachedBuild`,
  `InstallBuiltKernel`, `ArchiveInstalledKernel`

**Step 1: Add `paths *config.Paths` to each function**

Replace `config.GlobalPaths.*` access with `paths.*` in each function.

**Step 2: Update callers** — MCP handlers, CLI commands, `pkg/ui/build_kernel_wizard.go`

**Step 3: Run tests and commit**

```bash
git commit -m "refactor: remaining kernel functions accept paths parameter"
```

---

### Task 8: Remove `config.CurrentTheme` from kernel package

**Files:**
- Modify: `pkg/kernel/kernel.go`

After Tasks 1 and 3, `List()` and `ShowVersions()` no longer render — they return data.
Remove the `config.CurrentTheme` import if no other functions use it.

Check lines 263, 382 — these should have been removed in Tasks 1 and 3.

**Step 1: Verify and clean up imports**

Remove `config.CurrentTheme` references and the config import if unused.

**Step 2: Run tests and commit**

```bash
git commit -m "refactor: remove theme dependency from kernel domain"
```

---

## File Change Summary

| File | Change |
|------|--------|
| `pkg/kernel/kernel.go` | `List()` returns data, `Set()` accepts paths, `ShowVersions()` returns data, `Download*()` accepts paths, remove theme access |
| `pkg/kernel/build.go` | Remove `Interactive`, `Build()` + helpers accept paths |
| `cmd/kernel/list.go` | Format and print kernel list |
| `cmd/kernel/set.go` | Print success after domain call |
| `cmd/kernel/versions.go` | Format and print available versions |
| `cmd/kernel/get.go` | Pass paths to domain functions |
| `cmd/buildkernel/buildkernel.go` | Remove Interactive flag, pass paths |
| `cmd/cmdutil/helpers.go` | Pass paths to kernel/firecracker calls |
| `internal/mcp/tools_kernel_mgmt.go` | Delegate to domain instead of direct filesystem |
| `internal/mcp/tools_kernel_build.go` | Pass paths to Build() |
