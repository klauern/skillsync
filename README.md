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
