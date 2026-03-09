# GitHub Client Hex-Arch Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make `pkg/github` injectable by accepting token and API URL as constructor
parameters instead of reading from config globals.

**Architecture:** `NewClient()` takes explicit parameters. Domain packages that need GitHub
access accept a `*github.Client` as a parameter instead of calling `github.NewClient()`
internally. Callers (CLI, MCP) create the client and pass it down.

**Tech Stack:** Go, net/http

---

### Task 1: Accept token and API URL in constructor

**Files:**
- Modify: `pkg/github/client.go:35-39`

**Step 1: Change `NewClient()` to accept parameters**

```go
// NewClient creates a GitHub API client with the given token and API URL.
func NewClient(token, apiURL string) *Client {
	return &Client{
		token:  token,
		apiURL: apiURL,
	}
}
```

Add `apiURL` field to the `Client` struct:
```go
type Client struct {
	token  string
	apiURL string
}
```

**Step 2: Replace `config.GitHubAPI` references**

Lines 43, 49, 55 — replace `config.GitHubAPI` with `c.apiURL`.

**Step 3: Remove `config` import from `pkg/github/client.go`**

**Step 4: Update all callers** that call `github.NewClient()`:

- `pkg/kernel/kernel.go:62, 391` → `github.NewClient(config.GetGitHubToken(), config.GitHubAPI)`
- `cmd/cmdutil/helpers.go` (if applicable)
- Any other callers

**Step 5: Run tests and commit**

Run: `mise ci`

```bash
git commit -m "refactor: github.NewClient accepts token and API URL

Removes config dependency from github package. Callers provide
token and API URL explicitly."
```

---

### Task 2: Inject GitHub client into kernel domain

After Phase 3, kernel functions accept paths. Now also accept GitHub client.

**Files:**
- Modify: `pkg/kernel/kernel.go`

**Step 1: Add `client *github.Client` parameter to functions that fetch releases**

- `DownloadWithProgress()` at line 50 — currently creates client at line 62
- `ShowVersions()` at line 381 — currently creates client at line 391

```go
func DownloadWithProgress(version string, client *github.Client, paths *config.Paths,
	progressCallback func(float64), statusCallback func(string)) error {
	// Remove: client := github.NewClient()
	// Use parameter instead
}
```

**Step 2: Update callers** to create client and pass it

- `cmd/kernel/get.go`: `github.NewClient(config.GetGitHubToken(), config.GitHubAPI)`
- `cmd/cmdutil/helpers.go`: same
- `internal/mcp/tools_kernel_build.go`: same

**Step 3: Run tests and commit**

```bash
git commit -m "refactor: inject GitHub client into kernel download functions"
```

---

### Task 3: Inject GitHub client into firecracker domain

Same pattern as Task 2.

**Files:**
- Modify: `pkg/firecracker/firecracker.go`

**Step 1: Add `client *github.Client` parameter**

- `DownloadWithProgress()` at line 22 — creates client at line 29
- `ShowVersions()` at line 221 — creates client at line 231

**Step 2: Update callers**

**Step 3: Run tests and commit**

```bash
git commit -m "refactor: inject GitHub client into firecracker download functions"
```

---

## File Change Summary

| File | Change |
|------|--------|
| `pkg/github/client.go` | Accept token + API URL in constructor, remove config import |
| `pkg/kernel/kernel.go` | Accept `*github.Client` in download/version functions |
| `pkg/firecracker/firecracker.go` | Accept `*github.Client` in download/version functions |
| `cmd/kernel/get.go` | Create and pass GitHub client |
| `cmd/firecracker/get.go` | Create and pass GitHub client |
| `cmd/cmdutil/helpers.go` | Create and pass GitHub client |
| `internal/mcp/tools_kernel_build.go` | Create and pass GitHub client |
| `internal/mcp/tools_firecracker.go` | Create and pass GitHub client |
