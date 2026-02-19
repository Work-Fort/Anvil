# Agent Guidelines: Go CLI & Taskfile Architecture

This document describes the architecture and conventions for AI agents and developers working on this codebase.

## Architecture Overview

**anvil uses a Go CLI-first architecture:**
- Core functionality implemented in **Go** (`cmd/` and `pkg/`)
- Taskfiles provide build automation and CI integration
- Shell scripts minimized (only for extractable operations like `libguestfs:vendor`)

**Structure:**
```
cmd/, pkg/          → Go CLI implementation (Cobra + Bubble Tea)
Taskfile.dist.yaml  → 4 root wrapper tasks (build, clean, ci, install)
tasks/go/           → CLI build, test, lint tasks
tasks/libguestfs/   → Extractable vendoring operations
docs/               → Documentation build tasks
```

## Core Principles

### 1. Namespace-First Organization

**Root Taskfile has ONLY 4 wrapper tasks:**
```yaml
includes:
  go:
    taskfile: tasks/go
  docs:
    taskfile: docs

tasks:
  build:   # Orchestrates go:build + docs:build
  clean:   # Orchestrates go:clean + docs:clean
  ci:      # Orchestrates go:ci + docs:linkcheck
  install: # Delegates to go:install
```

**All implementation lives in namespaces:**
- `go:*` - CLI build, test, lint
- `docs:*` - Documentation operations
- `libguestfs:*` - Vendoring operations

### 2. Go CLI First

**Implement features in Go, not Taskfiles:**
```go
// pkg/kernel/build.go - Core logic in Go
func BuildKernel(version, arch string) error {
    // Implementation here
}

// cmd/build_kernel.go - CLI command
func buildKernelCmd() *cobra.Command {
    // Cobra command that calls pkg/kernel/build.go
}
```

**Taskfiles only for build automation:**
```yaml
# tasks/go/Taskfile.dist.yaml
build:
  cmds:
    - go build -ldflags "-X main.Version={{.VERSION}}" -o build/anvil
```

### 3. Decision Tree: Where to Put Code

```
Is this core functionality (kernel ops, signing, testing)?
├─ YES → Implement in Go CLI (cmd/ and pkg/)
└─ NO → Is this build automation (compile, test, lint)?
    ├─ YES → Add to tasks/go/ namespace
    └─ NO → Is this extractable to other projects?
        ├─ YES → Create new namespace (tasks/name/)
        └─ NO → Reconsider if it belongs in the project
```

## Key Patterns

### Variable Passing to CLI

```yaml
# tasks/go/Taskfile.dist.yaml
vars:
  VERSION:
    sh: git describe --tags --always --dirty

build:
  cmds:
    - go build -ldflags "-X main.Version={{.VERSION}}" -o build/anvil
```

### CLI-Overridable Variables

```yaml
# Bad - sh: always executes, ignores CLI overrides
vars:
  ARCH:
    sh: uname -m

# Good - allows: task go:build ARCH=arm64
vars:
  DETECTED_ARCH:
    sh: uname -m
  ARCH: '{{.ARCH | default .DETECTED_ARCH}}'
```

### Incremental Builds

```yaml
build:
  sources:
    - "**/*.go"
    - go.mod
    - go.sum
  generates:
    - "build/anvil"
  cmds:
    - go build -o build/anvil
```

### Environment Detection (Go, not shell)

```go
// pkg/signing/signing.go
func DetectEnvironment() (*Config, error) {
    if signingKey := os.Getenv("SIGNING_KEY"); signingKey != "" {
        return &Config{Source: "ci", KeyData: signingKey}, nil
    }
    if _, err := os.Stat(".gnupg"); err == nil {
        return &Config{Source: "local", GnupgDir: ".gnupg"}, nil
    }
    return nil, errors.New("no signing key found")
}
```

## Common Tasks

### Adding a New Feature

1. Implement in Go: `cmd/` for CLI interface, `pkg/` for logic
2. Add tests: `pkg/<package>/<package>_test.go`
3. Update CLI help: Markdown-based help rendered with Glamour
4. Test locally: `task go:build && ./build/anvil <command>`

### Adding a Build Task

1. Add to `tasks/go/Taskfile.dist.yaml` (or appropriate namespace)
2. Include `desc` and `summary` fields
3. Use namespace prefix in examples: `task go:build`
4. Test: `task --summary go:taskname && task go:taskname`

### Creating a New Namespace

Only if the namespace is:
- Self-contained
- Potentially extractable to other projects
- Not core business logic (that belongs in Go CLI)

Example: `tasks/libguestfs/` is extractable vendoring logic.

## CI/CD Integration

**Pattern:**
```yaml
# GitHub Actions
- name: Build CLI
  run: task go:build VERSION=${{ github.ref_name }}

- name: Run all checks
  run: task ci  # Orchestrates go:ci + docs:linkcheck

# Core operations use CLI directly
- name: Build kernel
  run: ./build/anvil build-kernel --version ${{ matrix.version }}
```

**Why this works:**
- Same commands locally and in CI
- Root wrappers (`task ci`) for multi-namespace operations
- Namespace tasks (`task go:ci`) for specific operations
- CLI handles business logic

## Anti-Patterns

### ❌ Don't: Duplicate CLI functionality in Taskfiles
```yaml
# Bad - reimplements ./anvil build-kernel
kernel:build:
  cmds:
    - # 100 lines of kernel build logic
```

### ❌ Don't: Put implementation in root Taskfile
```yaml
# Bad - root should only orchestrate
build:
  cmds:
    - go build -o build/anvil  # This belongs in go:build
```

### ❌ Don't: Use shell for core logic
```bash
# Bad - core functionality should be Go
./scripts/kernel/build-kernel.sh
```
```go
// Good - core functionality in Go
pkg/kernel/build.go
```

## Quick Reference

**Discovery:**
```bash
task --list                    # See all tasks
task --summary go:build        # Task details
./build/anvil --help  # CLI help
```

**Common workflows:**
```bash
# Development
task build               # Build CLI + docs
task go:build            # Build CLI only
task go:test             # Run tests
task ci                  # Run all checks

# Users
./build/anvil build-kernel   # Interactive wizard
./build/anvil test-kernel    # Test with Firecracker
./build/anvil sign-artifacts # Sign release
```

**Namespace structure:**
- Root: `build`, `clean`, `ci`, `install` (orchestration only)
- `go:*` - CLI build/test/lint
- `docs:*` - Documentation
- `libguestfs:*` - Vendoring

## Additional Notes

- CLI uses **Markdown-based help** rendered with Glamour (not default Cobra output)
- Always use `dir: .` in namespace includes
- Keep `desc` fields concise (shows in `task --list`)
- Put detailed help in `summary` (shows in `task --summary`)
- Test idempotency (safe to run twice)
