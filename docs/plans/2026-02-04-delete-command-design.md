# Delete Command Design (Sync Alias)

Date: 2026-02-04

## Summary
Add a first-class `delete` command that behaves like `sync --delete` used to, but without requiring a `--delete` flag. The new command reuses the same flags and parsing rules as `sync` and routes directly into the existing delete flow. The `sync` command no longer accepts `--delete` and its help text removes the delete mode section.

## Goals
- Provide `skillsync delete <source> <target>` as a full alias for delete mode.
- Keep flag parity with `sync` (e.g., `--dry-run`, `--interactive`, `--strategy`, `--include-plugins`).
- Preserve existing delete behavior (delete by source name, backup, confirmation prompts, TUI selection).
- Update onboarding, LLM guidance, and docs to reference `delete` directly.

## Non-Goals
- Change delete semantics (still name-based, target-only removals).
- Introduce new flags or strategies specific to delete.
- Alter sync conflict resolution behavior.

## CLI Design
- Add a new top-level command: `delete`.
- Remove `--delete` from `sync` flags and help.
- Both commands share a unified flag set and argument parser. `delete` forces delete mode via the shared execution path.
- Sync help points users to `skillsync delete` for removal operations.

## Implementation Notes
- Extract shared sync flags into a helper and reuse for both commands.
- Extract shared execution into a helper to avoid duplication; pass `deleteMode` explicitly.
- Update argument parser to accept a command name for accurate error messages.
- Register `delete` in the CLI command list.

## Docs & Testing
- Update onboarding (`onboard`/`llm`) text to include delete examples.
- Add a quick-start section showing delete usage and `--dry-run`.
- Extend help/command-registration tests and add delete command tests.
