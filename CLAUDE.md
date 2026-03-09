Read AGENTS.md

## QA Rules

When running QA checklists, execute every single item in order. Never skip items, never substitute prior results, never mark something as passed without running it this session. If an item takes a long time (e.g. kernel build), run it and wait. Do not rationalize skipping it.

## General Rules

When you notice a bug or problem, document it immediately in `docs/remaining-bugs.md`. Do not say "worth noting" and then move on without noting it.

## Testing Rules

- Never touch production config or keys. Always use an isolated temp directory for testing.
- This includes config set/get/unset tests — never modify the production `anvil.yaml`.
- If `ANVIL_SIGNING_PASSWORD` is already in the environment, do not redundantly add it to commands.
- Always rebuild and install (`mise run build && mise run install:local`) before testing changes.
- Use `mise` tasks for all build/test operations. Never call `go build`, `go test`, etc. directly.
