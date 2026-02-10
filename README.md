# SkillSync

Synchronize AI coding skills across Claude Code, Cursor, and Codex with a
single CLI.

## Requirements

- Go 1.25.4
- `just` (https://just.systems) for task running

## Build and Run

```bash
just build
./bin/skillsync --help
```

```bash
just run
```

```bash
just install
skillsync --help
```

## Quickstart

```bash
skillsync config init
skillsync discover --format table
skillsync sync cursor claudecode --dry-run
```

## Commands

- `config` manage config file and defaults
- `discover` list skills across platforms/scopes
- `sync` copy skills between platforms with conflict strategies
- `compare` compare skill sets across platforms
- `dedupe` identify duplicates by name/content similarity
- `export` export skills to JSON/YAML/Markdown
- `backup` create and manage backups
- `onboard` LLM-friendly onboarding guide (alias: `llm`)
- `promote`/`demote` move skills between repo/user scopes
- `scope` browse skills by scope
- `tui` interactive dashboard

Run `skillsync --help` for full command help.

## Configuration

Config lives at `~/.skillsync/config.yaml`. Generate or inspect it with:

```bash
skillsync config init
skillsync config show
skillsync config path
```

Platform skills paths are configured in `platforms.*.skills_paths`. You can
override them with colon-separated environment variables:

- `SKILLSYNC_CLAUDE_CODE_SKILLS_PATHS`
- `SKILLSYNC_CURSOR_SKILLS_PATHS`
- `SKILLSYNC_CODEX_SKILLS_PATHS`

By default, Claude Code discovery checks both `commands` and `skills` paths
(`.claude/commands`, `.claude/skills`, `~/.claude/commands`, `~/.claude/skills`)
so command-style prompts and standard skills are both synced.

## Command-Aware Sync

SkillSync models both traditional skills and prompt/command artifacts.

- Discovery includes both artifact types. Filter with `discover --type`.
- Sync/delete default to `skill` artifacts only (safe default).
- Include prompt/command artifacts explicitly with:
  - `--include-prompts`
  - `--type prompt` (or `--type skill,prompt`)

Examples:

```bash
# Discover only prompt/command artifacts
skillsync discover --platform claudecode --type prompt

# Sync Claude command artifacts to Codex (opt-in)
skillsync sync --include-prompts claudecode codex

# Sync prompt artifacts only from Claude to Cursor
skillsync sync --type prompt claudecode cursor
```

Compatibility notes:

| Source -> Target | Mapping | Fidelity |
|---|---|---|
| Claude command -> Codex | `name/SKILL.md` prompt artifact | Medium (slash-trigger semantics may be lossy) |
| Claude command -> Cursor | markdown prompt artifact | Medium (may require Cursor mode config for exact trigger behavior) |
| Codex prompt skill -> Claude | markdown artifact with prompt metadata | Medium |

Known limitations:

- Explicit slash trigger behavior is not guaranteed to be identical across
  platforms.
- Claude `argument-hint` is preserved as metadata on non-Claude targets.
- Codex user prompt directories outside documented skills/AGENTS flows remain
  experimental and are not enabled by default.

Legacy single-path overrides are still supported:

- `SKILLSYNC_CLAUDE_CODE_PATH`
- `SKILLSYNC_CURSOR_PATH`
- `SKILLSYNC_CODEX_PATH`

Use `SKILLSYNC_HOME` to relocate the config directory.

## Docs

- Architecture overview: `docs/architecture.md`
- Sync strategies: `docs/strategies.md`
- Skill format research: `docs/skill-formats-research.md`

## Development

```bash
just fmt
just lint
just test
just audit
```
