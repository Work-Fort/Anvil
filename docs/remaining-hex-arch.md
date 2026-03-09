# Hexagonal Architecture Migration

Tracking document for adopting hexagonal architecture across all anvil domains.
Domain functions accept data and interfaces, never acquire them. I/O, config
access, and presentation are adapter concerns (CLI, MCP, TUI).

## 1. Signing Domain ✅

[Plan](plans/2026-03-09-hexagonal-signing.md)

Remove interactive prompts from `pkg/signing`. Three domain functions had
embedded TUI/stdin password acquisition that deadlocked the MCP server.
Passwords are now parameters — CLI acquires via TUI/env/stdin, MCP reads
from env var.

## 2. Config Foundation

[Plan](plans/2026-03-09-hex-config.md)

Fix I/O leaks in `GetPaths()` (stderr + os.Exit). Establish dependency
injection pattern: domain functions accept `*config.Paths` and `config.Theme`
as parameters instead of accessing globals. Document the pattern for
subsequent phases.

## 3. Kernel Domain

[Plan](plans/2026-03-09-hex-kernel.md)

Remove I/O from `pkg/kernel`. `List()` and `ShowVersions()` return data
instead of printing. Remove `Interactive` flag from `BuildOptions`. All
functions accept `*config.Paths` parameter. Remove theme dependency.

## 4. Firecracker Domain

[Plan](plans/2026-03-09-hex-firecracker.md)

Same treatment as kernel — `List()`, `Set()`, `ShowVersions()` return data
instead of printing (47+ fmt.Print calls removed). All functions accept
`*config.Paths`. Remove theme dependency.

## 5. UI/Theme Extraction

[Plan](plans/2026-03-09-hex-ui-theme.md)

Thread theme as parameter into all UI components instead of accessing
`config.CurrentTheme` global. Extract build wizard's direct `kernel.Build()`
calls to callbacks, matching the version selector's existing pattern.

## 6. Rootfs Domain

[Plan](plans/2026-03-09-hex-rootfs.md)

Remove `Interactive` flag from `CreateOptions`. Require explicit `OutputPath`
instead of defaulting from config globals. Remove config dependency from
rootfs package.

## 7. GitHub Client

[Plan](plans/2026-03-09-hex-github.md)

`NewClient()` accepts token and API URL as parameters instead of reading
from config. Domain packages accept `*github.Client` as parameter instead
of creating one internally.

## 8. MCP Handler Cleanup

[Plan](plans/2026-03-09-hex-mcp.md)

MCP handlers delegate to domain functions instead of reimplementing logic
with direct filesystem access. After domains have clean return-data APIs
from earlier phases, handlers become thin adapters: parse → call → serialize.
