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
