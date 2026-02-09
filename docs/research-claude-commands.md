# Claude Command Artifact Research (skillsync-c1g.1)

Date: 2026-02-09

## Scope

This note captures authoritative Claude Code behavior for command artifacts
(`.claude/commands/*.md`), plus local observations from this repository and
developer machine state.

## Authoritative Behavior (Claude docs)

Source pages:
- https://code.claude.com/docs/en/slash-commands
- https://platform.claude.com/docs/en/agent-sdk/slash-commands

### Artifact locations

- Project-scoped custom commands: `.claude/commands/`
- User-scoped custom commands: `~/.claude/commands/`
- Skills are in `.claude/skills/<name>/SKILL.md` and `~/.claude/skills/<name>/SKILL.md`
- Custom slash commands have been merged into skills; legacy
  `.claude/commands/*.md` still work.

### Input format

For `.claude/commands/<name>.md`:
- Command name is the filename stem (`review.md` => `/review`)
- Body is markdown instructions
- Optional YAML frontmatter fields are supported, including:
  - `allowed-tools`
  - `argument-hint`
  - `description`
  - `model`

From current skill frontmatter reference (shared behavior used by commands in
merged model), additional relevant fields include:
- `disable-model-invocation`
- `user-invocable`
- `context`
- `agent`
- `hooks`

### Invocation semantics

- Users invoke by slash command (`/name ...args`).
- Arguments can be consumed by `$ARGUMENTS`, `$ARGUMENTS[N]`, and `$N`.
- In SDK usage, slash commands are sent in prompt text and appear in
  `init` message `slash_commands`.

### Discovery precedence and collision rules

Documented explicitly:
- If a skill and a command share the same name, the skill takes precedence.
- Skill precedence by level is: enterprise > personal > project.
- Plugin skills use namespacing (`plugin-name:skill-name`) to avoid conflicts.

Not explicitly documented on slash-command SDK page:
- Project vs personal command collision precedence (same command filename in both
  scopes) is not stated there; treat as unresolved until validated empirically.

## Local Observations

### Host filesystem state (2026-02-09)

- Both `~/.claude/commands` and `~/.claude/skills` exist.
- Sample command files present in `~/.claude/commands`:
  `agents-md.md`, `bd-work.md`, `commit-push.md`, etc.
- Observed frontmatter in local command files commonly includes:
  `description`, `allowed-tools`.

### Current skillsync implementation behavior

Relevant code paths:
- `internal/parser/claude/claude.go`
- `internal/parser/skills/skills.go`

Current parser behavior:
- Claude parser defaults to `~/.claude/skills` only.
- It parses:
  1) `SKILL.md` (agent-skill format) first
  2) legacy `*.md` files under the configured base path second
- Existing code does not yet separately parse `.claude/commands` as a first-class
  artifact source unless the caller explicitly sets base path to commands.
- Name collision behavior implemented in parser:
  `SKILL.md` entry wins over legacy markdown with same name.

## Implications for skillsync-c1g follow-on design

- Discovery must include both commands and skills paths for Claude by default.
- Unified model should represent command/skill type distinctly, while preserving
  shared frontmatter fields.
- Sync mapping should preserve precedence:
  skill-over-command for same slash name, and documented scope precedence for
  skills.
- Command-scope collision precedence (project vs personal) needs a small
  empirical validation task if required for deterministic sync guarantees.
