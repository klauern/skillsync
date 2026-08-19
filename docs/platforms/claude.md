# Claude Code

> Platform reference for SkillSync parser development and cross-platform mapping.

## Overview

Claude Code is Anthropic's agentic CLI tool for software engineering. It uses a `.claude/` directory
ecosystem to store skills (reusable agent instructions), commands (legacy slash-command prompts),
and instructions (`CLAUDE.md` memory files). Skills follow the
[Agent Skills Standard](https://agentskills.io/home) and are the primary extensibility mechanism.

Several Claude skill frontmatter fields describe runtime behavior that does not port cleanly to
Codex CLI. Treat `context: fork`, `agent`, `hooks`, `disable-model-invocation`, `user-invocable`,
and `model` as Claude-native controls unless a Codex-native reimplementation is explicitly added.

Custom slash commands have been merged into skills. A file at `.claude/commands/review.md` and a
skill at `.claude/skills/review/SKILL.md` can both expose `/review`, but they do not behave the
same way. Legacy `.claude/commands/` files remain a compatibility layer, while skills are the
canonical format going forward.

## Directory Structure

```text
~/.claude/                          # Personal (user-scope) root
  skills/
    <skill-name>/
      SKILL.md                      # Skill entrypoint (required)
      scripts/                      # Executable helper scripts
      references/                   # Reference documentation
      assets/                       # Static assets (templates, schemas)
      examples/                     # Example output files
  commands/
    <name>.md                       # Legacy personal slash commands

<project>/
  .claude/
    skills/
      <skill-name>/
        SKILL.md                    # Project-scoped skill
    commands/
      <name>.md                     # Legacy project-scoped slash commands
    agents/
      <name>.md                     # Subagent definitions
    settings.json                   # Project settings (committed)
    settings.local.json             # Local project settings (gitignored)
    CLAUDE.md                       # Project-level instructions

<project>/packages/<pkg>/
  .claude/
    skills/                         # Nested monorepo skills (auto-discovered)

~/.claude/CLAUDE.md                 # Global personal instructions
```

Enterprise-managed skills are deployed through managed settings (see Claude Code permissions docs).

Plugin skills live at `<plugin>/skills/<skill-name>/SKILL.md` and are namespaced as
`plugin-name:skill-name`.

Skills from directories added via `--add-dir` are loaded automatically and support live change
detection.

## Artifact Types

### Skills (`SKILL.md`)

Skills are the primary artifact type. Each skill is a directory containing a `SKILL.md` entrypoint
file with optional YAML frontmatter and markdown body content.

#### Frontmatter Schema

All fields are optional. Only `description` is recommended.

| Field                      | Type       | Required?   | Description |
|:---------------------------|:-----------|:------------|:------------|
| `name`                     | `string`   | No          | Display name and `/slash-command` trigger. If omitted, uses the directory name. Lowercase letters, numbers, and hyphens only (max 64 characters). |
| `description`              | `string`   | Recommended | What the skill does and when to use it. Claude uses this to decide when to apply the skill automatically. If omitted, uses the first paragraph of markdown content. |
| `argument-hint`            | `string`   | No          | Hint shown during autocomplete to indicate expected arguments (e.g., `[issue-number]`, `[filename] [format]`). |
| `disable-model-invocation` | `bool`     | No          | Claude-native runtime control to prevent automatic loading. Preserve as metadata when translating to other CLIs. Default: `false`. |
| `user-invocable`           | `bool`     | No          | Claude-native runtime control for `/` menu visibility. Preserve as metadata when translating to other CLIs. Default: `true`. |
| `allowed-tools`            | `string \| string[]` | No | Comma-separated list or array of tools Claude can use without permission when this skill is active (e.g., `Read, Grep, Glob`). Accepts tool patterns like `Bash(gh *)`. |
| `model`                    | `string`   | No          | Claude model override for this skill. Preserve as metadata when translating to other CLIs. |
| `context`                  | `string`   | No          | Claude-specific subagent context. Set to `fork` to run in a forked subagent context; Codex CLI has no equivalent. |
| `agent`                    | `string`   | No          | Claude-specific subagent selector when `context: fork` is set. Options: `Explore`, `Plan`, `general-purpose`, or custom agent names from `.claude/agents/`. |
| `hooks`                    | `object`   | No          | Claude-specific hook declarations scoped to this skill's lifecycle. Preserve only as frontmatter metadata when syncing; the runtime behavior remains Claude-specific. |
| `type`                     | `string`   | No          | SkillSync extension: `skill` (default) or `prompt` for slash-command semantics. |
| `trigger`                  | `string`   | No          | SkillSync extension: explicit slash trigger (e.g., `/my-command`). |
| `scope`                    | `string`   | No          | SkillSync extension: `user`, `repo`, `system`, `plugin`, etc. |

#### Codex Portability Notes

The common Claude-to-Codex portable subset is smaller than the full Claude
schema:

- Usually portable as native skill content: `name`, `description`, markdown
  body content, and supporting directories.
- Usually portable only as intent/metadata: `allowed-tools`.
- Portable only as descriptive metadata, not as behavior: `hooks`.
- Claude-specific runtime controls with no native Codex skill equivalent:
  `disable-model-invocation`, `user-invocable`, `model`, `context`, and
  `agent`.
- SkillSync extension fields such as `type`, `trigger`, and `scope` are
  transport metadata in this repo. They do not mean Codex will reproduce
  Claude slash-command, invocation, or scope behavior.

When these Claude-only fields move to Codex, treat them as lossy compatibility
data or re-author them as Codex-native instructions. For `hooks` specifically,
SkillSync may preserve the declared hook object as metadata for inspection or
round-tripping, but the trigger timing, lifecycle scope, and side effects do
not become portable runtime behavior.

#### Invocation Control Matrix

| Frontmatter                      | User can invoke | Claude can invoke | Context loading behavior |
|:---------------------------------|:----------------|:------------------|:-------------------------|
| (defaults)                       | Yes             | Yes               | Description always in context; full content loads on invocation |
| `disable-model-invocation: true` | Yes             | No                | Description not in context; full content loads on user invocation |
| `user-invocable: false`          | No              | Yes               | Description always in context; full content loads on invocation |

#### Body Format

The Markdown body after frontmatter contains the skill instructions. It supports:

- Standard Markdown formatting
- `$ARGUMENTS` -- replaced with all arguments passed when invoking the skill
- `$ARGUMENTS[N]` -- access a specific argument by 0-based index
- `$N` -- shorthand for `$ARGUMENTS[N]` (e.g., `$0`, `$1`)
- `` !`command` `` -- dynamic context injection; shell command runs before content is sent to Claude
- `${CLAUDE_SESSION_ID}` -- replaced with the current session ID

If `$ARGUMENTS` is not present in the content but arguments are provided, they are appended as
`ARGUMENTS: <value>`.

#### Supporting Directories

A skill directory can include supporting files alongside `SKILL.md`:

| Directory      | Purpose |
|:---------------|:--------|
| `scripts/`     | Executable helper scripts (Python, shell, etc.) |
| `references/`  | Reference documentation loaded on demand |
| `assets/`      | Static assets: templates, schemas, config files |
| `examples/`    | Example output files showing expected format |

Supporting files are referenced from `SKILL.md` so Claude knows when to load them. The
recommendation is to keep `SKILL.md` under 500 lines and move detailed reference material to
separate files.

#### Example

```yaml
---
name: fix-issue
description: Fix a GitHub issue
disable-model-invocation: true
---

Fix GitHub issue $ARGUMENTS following our coding standards.

1. Read the issue description
2. Understand the requirements
3. Implement the fix
4. Write tests
5. Create a commit
```

### Commands (legacy `*.md`)

Legacy command files are single markdown files in `.claude/commands/` or `~/.claude/commands/`.
They create slash commands and continue to work, but skills are the recommended replacement.

#### Frontmatter Schema

| Field            | Type       | Required? | Description |
|:-----------------|:-----------|:----------|:------------|
| `description`    | `string`   | No        | Human-readable description of the command. |
| `allowed-tools`  | `string[]` | No        | Tools Claude can use when this command is active. |
| `argument-hint`  | `string`   | No        | Hint shown during autocomplete for expected arguments. |
| `model`          | `string`   | No        | Model override for this command. |

Commands also support the same frontmatter fields as skills (`disable-model-invocation`,
`user-invocable`, `context`, `agent`, `hooks`), but those fields should be read as Claude runtime
controls rather than portable Codex behavior.

#### Codex Portability Notes

Claude commands are portable mainly as prompt **content**. They are not
portable as equivalent runtime objects:

- The filename-derived slash trigger does not become a first-class Codex skill
  capability.
- `argument-hint`, `model`, `disable-model-invocation`, `user-invocable`,
  `context`, `agent`, and `hooks` do not map to native Codex per-skill command
  behavior.
- When SkillSync maps Claude commands into Codex skill output, the result is a
  lossy content transformation rather than a behavior-preserving conversion.

#### Name Derivation

The command name (and slash trigger) is derived from the filename stem:
`review.md` creates `/review`, `fix-bug.md` creates `/fix-bug`.

#### Argument Interpolation

Commands support the same `$ARGUMENTS`/`$N` substitution as skills:

- `$ARGUMENTS` -- all arguments as a single string
- `$ARGUMENTS[N]` -- specific argument by 0-based index
- `$N` -- shorthand for `$ARGUMENTS[N]`

#### Example

```yaml
---
description: Review code changes
allowed-tools:
  - Read
  - Grep
  - Glob
argument-hint: [file-or-directory]
---

Review the code at $ARGUMENTS. Focus on:
- Correctness and edge cases
- Security vulnerabilities
- Performance issues
```

### Instructions (`CLAUDE.md`)

`CLAUDE.md` files are plain markdown loaded as persistent instructions (memory). They have no
frontmatter -- the entire file content is injected into Claude's context.

#### Hierarchical Loading

Instructions are loaded from multiple locations and merged, with more specific files taking
precedence:

1. `~/.claude/CLAUDE.md` -- global personal instructions
2. `<project>/CLAUDE.md` -- project-level instructions
3. `<project>/.claude/CLAUDE.md` -- project `.claude/` directory instructions
4. Parent directories are also scanned up to the repo root

Claude Code also discovers `CLAUDE.md` files from `--add-dir` directories when the
`CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` environment variable is set.

#### Codex Portability Notes

`CLAUDE.md` content can often be reused in `AGENTS.md`, but the loading model is
different:

- Claude merges memory from multiple `CLAUDE.md` locations.
- Codex concatenates `AGENTS.md` files from root to cwd.
- Codex can also inject instruction text from `config.toml`.

The prose may be portable while the prompt-construction behavior is not.

## Scope Levels

Skills operate within a tiered scope system where more specific scopes override more general ones:

| Scope      | Precedence | Location | Description |
|:-----------|:-----------|:---------|:------------|
| Enterprise | Highest    | Managed settings | Organization-wide, deployed by admins |
| Personal   | High       | `~/.claude/skills/` | User-level, available across all projects |
| Project    | Medium     | `.claude/skills/` | Repository-level, specific to one project |
| Plugin     | Namespaced | `<plugin>/skills/` | Installed via plugin system, uses `plugin-name:skill-name` namespace |

Enterprise > Personal > Project for same-name conflicts. Plugin skills use namespacing and cannot
conflict with other levels.

## Discovery & Precedence

1. **SKILL.md wins over legacy `*.md`**: When a `SKILL.md` format skill and a legacy markdown file
   share the same name, the `SKILL.md` version takes precedence.
2. **Skills win over commands**: If a skill at `.claude/skills/review/SKILL.md` and a command at
   `.claude/commands/review.md` share the same name, the skill takes precedence.
3. **Scope precedence**: Enterprise > Personal > Project for same-name skills.
4. **Plugin namespacing**: Plugin skills use `plugin-name:skill-name` format and cannot collide
   with non-plugin skills.
5. **Nested discovery**: Skills in nested `.claude/skills/` directories (e.g., in monorepo
   packages) are auto-discovered when working with files in those subdirectories.
6. **Context budget**: Skill descriptions are loaded into context up to a character budget
   (2% of context window, fallback 16,000 characters). Override with the
   `SLASH_COMMAND_TOOL_CHAR_BUDGET` environment variable.

## Tool Restrictions

The `allowed-tools` field in frontmatter controls which tools Claude can use without per-use
permission approval when a skill is active.

Syntax variants:
- Simple tool name: `Read`, `Grep`, `Glob`
- Tool with pattern: `Bash(gh *)`, `Bash(python *)`
- Comma-separated in frontmatter: `allowed-tools: Read, Grep, Glob`
- Array in frontmatter: `allowed-tools: [Read, Grep, Glob]`

Permission rules for skills themselves use the `Skill(name)` syntax:
- `Skill(commit)` -- exact match
- `Skill(review-pr *)` -- prefix match with any arguments

Baseline permission settings still govern tool approval for tools not listed in `allowed-tools`.

## Parser Implementation Notes

### Parser Location

`internal/parser/claude/claude.go`

### Parsing Strategy

The Claude parser uses a two-phase approach:

1. **Phase 1 -- SKILL.md parsing**: Uses the shared `internal/parser/skills/skills.go` parser
   to discover and parse `SKILL.md` files (case-insensitive: `SKILL.md`, `skill.md`, `Skill.md`).
   These take precedence and their names are recorded in a `seenNames` map.

2. **Phase 2 -- Legacy file parsing**: Discovers `*.md` and `**/*.md` files, filtering out:
   - `SKILL.md` files (already parsed in phase 1)
   - Files inside skill directories (prevents reference files from being treated as skills)

   Each legacy file is parsed with `parseSkillFile()`.

### Frontmatter Handling

- Split via `parser.SplitFrontmatter()` (detects `---` delimiters)
- Parsed via `parser.ParseYAMLFrontmatter()` (YAML to `map[string]any`)
- Tool extraction checks both `tools` and `allowed-tools` keys
- CamelCase frontmatter keys are normalized to kebab-case via `skills.NormalizeKey()`

### Command Detection

The `isClaudeCommandFile()` function at `internal/parser/claude/claude.go:337` identifies command
files by checking if any path component is `commands`. Command files default to:
- `SkillTypePrompt` (instead of `SkillTypeSkill`)
- Slash trigger derived from filename stem (e.g., `review.md` becomes `/review`)

### Plugin Symlink Detection

`internal/parser/claude/pluginindex.go` handles plugin detection:

- `LoadPluginIndex()` reads `~/.claude/installed_plugins.json` to build a lookup index
- `DetectPluginSource()` checks if a skill directory is a symlink:
  - If target is within `~/.claude/plugins/cache/`, it is an installed plugin
  - Otherwise, it is a development symlink
- Plugin metadata (`PluginName`, `Marketplace`, `Version`) is extracted from the cache path
  structure or the installed plugins manifest

### Skill Directory Structure Detection

`internal/parser/skills/skills.go` detects supporting directories:
- `scripts/` -- executable helpers
- `references/` -- reference documentation
- `assets/` -- static assets

Discovered files are appended to the respective `Skill` fields alongside any declared in frontmatter.

### Test Fixtures

Test fixtures are in `testdata/skills/claude/`:

| Fixture | Purpose |
|:--------|:--------|
| `basic-skill/SKILL.md` | Minimal skill with name and description |
| `full-agent-skill/SKILL.md` | All Agent Skills Standard fields, with scripts/, references/, assets/ |
| `camelcase-frontmatter/SKILL.md` | CamelCase key normalization testing |
| `repo-scope-skill/SKILL.md` | Repo scope precedence testing |
| `system-scope-skill/SKILL.md` | System scope precedence testing |

Legacy format fixtures are in `testdata/skills/legacy/`:

| Fixture | Purpose |
|:--------|:--------|
| `flat-skill.md` | Legacy flat file with frontmatter |
| `minimal-frontmatter.md` | Minimal frontmatter |
| `no-frontmatter.md` | No frontmatter (name derived from filename) |
| `plus-delimiter.md` | Alternative `+++` delimiter testing |

## Gaps

- **Project vs personal command collision precedence**: When the same command filename exists in
  both `.claude/commands/` (project) and `~/.claude/commands/` (personal), the precedence is not
  documented by Anthropic. Needs empirical validation.
- **Enterprise scope parsing**: The parser does not currently handle enterprise-managed skills
  (deployed via managed settings). These are injected at a different layer.
- **Dynamic context injection**: The `` !`command` `` preprocessing syntax is a runtime feature and
  is not parsed or evaluated by SkillSync.
- **Subagent definitions**: `.claude/agents/*.md` files are not yet parsed by SkillSync.
- **Nested monorepo discovery**: The parser does not yet replicate Claude Code's automatic
  discovery of skills from nested `.claude/skills/` directories within subdirectories.
- **Context budget awareness**: SkillSync does not model the 2% context window budget for skill
  description loading.
- **`hooks` field parsing**: The `hooks` frontmatter field is stored as metadata but not
  structurally parsed. That metadata is portable only as a descriptive record of
  Claude configuration, not as cross-platform hook execution semantics.
- **Codex portability of Claude-only controls**: `disable-model-invocation`,
  `user-invocable`, `model`, `context: fork`, and `agent` should be documented
  and treated as lossy when Claude content is synced to Codex CLI.
- **Lossy hook translation**: Claude skill hooks can be carried forward as
  metadata only. Platforms with their own hook systems, such as Codex config
  hooks or Gemini settings hooks, require manual re-authoring because the event
  model and runtime ownership differ.

## Sources

- [Claude Code Skills documentation](https://code.claude.com/docs/en/skills) -- retrieved 2026-02-22
- [Claude Code Slash Commands documentation](https://code.claude.com/docs/en/slash-commands) -- retrieved 2026-02-22
- [Agent Skills Standard](https://agentskills.io/home) -- open standard for portable AI agent skills
- `docs/platforms/portability-assessment.md` -- focused Claude/Codex portability assessment
- `docs/platforms/cross-platform-mapping.md` -- broad cross-platform reference
- `internal/parser/claude/claude.go` -- SkillSync Claude parser implementation
- `internal/parser/skills/skills.go` -- shared SKILL.md parser
- `internal/parser/claude/pluginindex.go` -- plugin symlink detection
- `internal/model/skill.go` -- unified skill model
<!-- v2 reference: verified 2026-08-18; official source https://code.claude.com/docs/en/skills -->
