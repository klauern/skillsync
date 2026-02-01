# SkillSync

Synchronize AI coding skills across Claude Code, Cursor, and Codex with a
single CLI.

## Requirements

- Go 1.25.4

## Build and Run

```bash
make build
./bin/skillsync --help
```

```bash
make run
```

```bash
make install
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
make fmt
make lint
make test
make audit
```
