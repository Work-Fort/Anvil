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

## Code Change Process (MANDATORY)

Every code change must follow this exact sequence. No exceptions, no shortcuts:

1. **Make the code change.**
2. **Run `mise ci`** — confirm all tests pass.
3. **Commit the change** — uncommitted changes are invisible to future sessions and easy to lose.
4. **Run `mise run build && mise run install:local`** — deploy the new binary.
5. **Ask the user to restart the MCP server** — the MCP server runs the old binary until restarted.
6. **Verify the new version** — run `get_context` and confirm the version matches the commit.
7. **Only then continue testing.**

Never skip steps. Never combine steps. Never keep testing against a stale MCP server.

## Debugging Rules

- When encountering any bug or test failure, use the `systematic-debugging` skill FIRST. Do not jump to fixes.
- Never edit code based on an unconfirmed hypothesis. Confirm root cause before writing a fix.
- If a fix doesn't resolve the issue, revert it before investigating further. Do not leave wrong fixes in the codebase.
