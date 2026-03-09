# UI/Theme Extraction Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Decouple `pkg/ui` from `config.CurrentTheme` global. Make theme an injected
parameter. Fix the build kernel wizard's direct domain calls.

**Architecture:** Theme is passed as a parameter to all UI constructors and functions.
`config.CurrentTheme` remains as the source of truth in CLI entry points, but UI components
don't access it directly. The build kernel wizard stops calling `kernel.Build()` directly —
it uses callbacks like the version selector already does.

**Tech Stack:** Go, charmbracelet/bubbletea v2, charmbracelet/lipgloss v2

---

### Task 1: Thread theme into `RenderTabs` and `RenderTabContent`

**Files:**
- Modify: `pkg/ui/tabs.go:38, 177`
- Modify: all callers of `RenderTabs` and `RenderTabContent`

**Step 1: Add `theme config.Theme` parameter**

```go
func RenderTabs(tabs []Tab, cfg TabsConfig, theme config.Theme) string {
	// Replace config.CurrentTheme with theme parameter
}

func RenderTabContent(content string, width int, theme config.Theme) string {
	// Replace config.CurrentTheme with theme parameter
}
```

**Step 2: Update callers** — `build_kernel_wizard.go`, `version_selector.go`, and any
callers in `cmd/init/`

**Step 3: Run tests and commit**

```bash
git commit -m "refactor: thread theme into tab rendering functions"
```

---

### Task 2: Thread theme into `VersionSelectorModel`

**Files:**
- Modify: `pkg/ui/version_selector.go`

**Step 1: Add `theme` field to `VersionSelectorModel`**

```go
type VersionSelectorModel struct {
	theme config.Theme
	// ... existing fields
}
```

**Step 2: Accept theme in `NewVersionSelector` and `RunVersionSelector`**

```go
func NewVersionSelector(theme config.Theme, ...) VersionSelectorModel {
	return VersionSelectorModel{
		theme: theme,
		// ...
	}
}

func RunVersionSelector(theme config.Theme, ...) error {
	model := NewVersionSelector(theme, ...)
	// ...
}
```

**Step 3: Replace all `config.CurrentTheme` with `m.theme`** in the model's methods

Lines 42, 69, 128, 551 — replace `config.CurrentTheme` with `m.theme`.

**Step 4: Update callers** — `cmd/cmdutil/helpers.go` passes `config.CurrentTheme`

**Step 5: Run tests and commit**

```bash
git commit -m "refactor: thread theme into version selector"
```

---

### Task 3: Thread theme into `BuildKernelWizard`

**Files:**
- Modify: `pkg/ui/build_kernel_wizard.go`

**Step 1: Add `theme` field to `BuildKernelWizard`**

**Step 2: Accept theme in `NewBuildKernelWizard` and `RunBuildKernelWizard`**

**Step 3: Replace `config.CurrentTheme`** at lines 193, 1049, 1069, 1148, 1271

**Step 4: Update caller** — `cmd/buildkernel/buildkernel.go` passes `config.CurrentTheme`

**Step 5: Run tests and commit**

```bash
git commit -m "refactor: thread theme into build kernel wizard"
```

---

### Task 4: Thread theme into helper functions

**Files:**
- Modify: `pkg/ui/helpers.go`

**Step 1: Add `theme` parameter to `RenderProgressModal`** (line 112)

**Step 2: Add `theme` field to `ConfirmationForm`** (line 169)

**Step 3: Update `ConfirmationForm.View()`** (line 208) to use `cf.theme`

**Step 4: Run tests and commit**

```bash
git commit -m "refactor: thread theme into UI helper functions"
```

---

### Task 5: Extract build wizard domain calls to callbacks

The build kernel wizard directly calls `kernel.Build()` at line 1428 and
`kernel.CheckCachedBuild()` at line 255. This violates hex-arch — UI should
not call domain directly.

**Files:**
- Modify: `pkg/ui/build_kernel_wizard.go`

**Step 1: Add build callback to wizard options**

```go
type BuildKernelCallbacks struct {
	BuildFn          func(opts kernel.BuildOptions) error
	CheckCachedFn    func(version string) (bool, string, error)
	InstallFn        func(stats kernel.BuildStats, setDefault bool) (string, error)
	ArchiveFn        func(stats kernel.BuildStats, archiveDir string) error
}
```

**Step 2: Accept callbacks in constructor**

```go
func NewBuildKernelWizard(callbacks BuildKernelCallbacks, theme config.Theme, ...) *BuildKernelWizard {
```

**Step 3: Replace direct domain calls with callback invocations**

Line 255: `hasCached, statsFile, err := kernel.CheckCachedBuild("")`
becomes: `hasCached, statsFile, err := m.callbacks.CheckCachedFn("")`

Line 1428: `return m.performBuild(...)`
Change `performBuild` to use `m.callbacks.BuildFn(opts)`

**Step 4: Update `cmd/buildkernel/buildkernel.go`** to provide callbacks

```go
callbacks := ui.BuildKernelCallbacks{
	BuildFn:       func(opts kernel.BuildOptions) error { return kernel.Build(opts, config.GlobalPaths) },
	CheckCachedFn: func(v string) (bool, string, error) { return kernel.CheckCachedBuild(v, config.GlobalPaths) },
	InstallFn:     func(s kernel.BuildStats, d bool) (string, error) { return kernel.InstallBuiltKernel(s, d, config.GlobalPaths) },
	ArchiveFn:     func(s kernel.BuildStats, dir string) error { return kernel.ArchiveInstalledKernel(s, dir) },
}
ui.RunBuildKernelWizard(callbacks, config.CurrentTheme, ...)
```

**Step 5: Run tests and commit**

```bash
git commit -m "refactor: extract domain calls from build wizard to callbacks

Build wizard no longer imports pkg/kernel directly. All domain
operations go through callbacks provided by the CLI adapter."
```

---

### Task 6: Remove `config.GlobalPaths` from build wizard

**Files:**
- Modify: `pkg/ui/build_kernel_wizard.go`

Line 1419-1420 accesses `config.GlobalPaths.KernelBuildDir` and line 1466 accesses
`config.GetKernelsArchiveLocation()`.

**Step 1: Pass these as parameters in the callbacks or constructor**

Add `kernelBuildDir string` and `archiveLocation string` to the wizard constructor
or to the callbacks struct.

**Step 2: Run tests and commit**

```bash
git commit -m "refactor: remove config.GlobalPaths from build wizard"
```

---

### Task 7: Clean up imports

After all theme threading and callback extraction, `pkg/ui/` should no longer import
`config` (for `CurrentTheme`) or `kernel` (for domain calls).

**Step 1: Remove `config` import** from version_selector.go, build_kernel_wizard.go, etc.

Note: `config.Theme` type is still referenced as a parameter type. The import stays
for the type, but no global variables are accessed.

**Step 2: Remove `kernel` import** from build_kernel_wizard.go (replaced by callbacks)

**Step 3: Run tests and commit**

```bash
git commit -m "refactor: clean up UI package imports after hex-arch"
```

---

## File Change Summary

| File | Change |
|------|--------|
| `pkg/ui/tabs.go` | Theme parameter on RenderTabs, RenderTabContent |
| `pkg/ui/version_selector.go` | Theme field, accept in constructor |
| `pkg/ui/build_kernel_wizard.go` | Theme field, callbacks for domain ops, remove config/kernel imports |
| `pkg/ui/helpers.go` | Theme parameter on RenderProgressModal, ConfirmationForm |
| `cmd/buildkernel/buildkernel.go` | Provide callbacks and theme to wizard |
| `cmd/cmdutil/helpers.go` | Pass theme to RunVersionSelector |
