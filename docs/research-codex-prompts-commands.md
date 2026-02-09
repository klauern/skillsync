# Codex Prompt/Command Research (skillsync-c1g.2)

Date: 2026-02-09

## Scope

Determine Codex representations for reusable prompt/command behavior, required
metadata, invocation model, and mapping constraints vs Claude command artifacts.

## Authoritative Behavior (OpenAI docs)

Sources:
- https://developers.openai.com/codex/skills/
- https://developers.openai.com/codex/slash-commands/
- https://developers.openai.com/codex/config/
- https://developers.openai.com/codex/agents-md/
- https://developers.openai.com/codex/ide/

### Supported artifacts

1. Skills (`SKILL.md` in a directory), loaded from:
- `./.codex/skills`
- `~/.codex/skills`
- `/etc/codex/skills`

2. Instruction files:
- `AGENTS.md` (root + parent dirs)
- Optional config instructions via `instructions_file` in `~/.codex/config.toml`

3. Slash commands:
- Built-in CLI/App commands (for example `/status`, `/diff`, `/review`)
- Docs do not specify user-defined slash commands from markdown files.

### Skill schema and invocation

From Codex skills docs:
- Skill frontmatter requires `name` and `description`.
- Common optional fields: `version`, `allowed-tools`, metadata keys.
- Skills may include supporting `scripts/`, `assets/`, `references/`.
- Codex chooses/invokes skills based on task relevance; they are not documented as
  filename-driven custom slash command aliases.

### Discovery precedence

From config + AGENTS docs:
- Skill directories are searched in order and can be customized with
  `skills_dir` / `skills_dir_add`.
- `AGENTS.md` instructions are merged from root toward cwd, with nearest
  (deeper) instructions taking precedence.

## Local Observations (2026-02-09)

- `~/.codex/skills` exists and contains many `SKILL.md` artifacts.
- `~/.codex/prompts/*.md` exists locally and is used in this environment, but this
  prompt directory is not currently documented in the official Codex docs above.
- Current `skillsync` Codex parser (`internal/parser/codex/codex.go`) parses:
  - `SKILL.md`
  - `config.toml` (`instructions` and `developer_instructions`)
  - `AGENTS.md`
- Current parser does not parse `~/.codex/prompts`.

## Hard Constraints / Non-portable Fields

1. No documented Codex equivalent to Claude file-backed custom slash commands
   (`.claude/commands/<name>.md`) with explicit `/name` trigger.
2. Claude command argument placeholders (`$ARGUMENTS`, `$1`) have no guaranteed,
   documented one-to-one Codex command-file equivalent.
3. Claude command-specific fields are partially portable only:
   - `description`: portable
   - `allowed-tools`: portable
   - `argument-hint`: no direct documented Codex field
   - `model`: partially portable via config/profile, not per-command in a documented
     slash-command file format

## Viable Claude -> Codex Mapping

### Preferred target: Codex skill directory

Map Claude command file to `SKILL.md` with `type: prompt` metadata in skillsync
model (internal distinction), even if Codex treats it as a skill artifact.

| Claude command field | Codex target | Notes |
|---|---|---|
| filename trigger (`review.md` => `/review`) | `name: review` | Trigger semantics may be lossy (skill selection vs explicit slash alias). |
| `description` | `description` | Direct mapping. |
| `allowed-tools` | `allowed-tools` | Direct mapping. |
| command body markdown | SKILL.md body | Direct mapping. |
| `argument-hint` | metadata passthrough | Preserve in metadata; no documented native Codex behavior. |
| `model` | metadata or profile note | Preserve as metadata; optionally surface in config migration hints. |
| `$ARGUMENTS`, `$1` conventions | body text unchanged | Kept as literal instruction text; runtime semantics may differ. |

### Fallback target: `AGENTS.md` snippet

Only when user opts out of skill creation; lower fidelity and no per-command
granularity.

## Conclusion for c1g.4 design

- Treat Codex first-class command parity as "skill-backed prompt behavior" rather
  than true custom slash-command files.
- Add explicit "lossy mapping" warnings for trigger and argument semantics.
- Optionally add experimental support for `.codex/prompts` gated as
  "observed/undocumented" until official docs confirm format + precedence.
