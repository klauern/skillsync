# Codex (OpenAI CLI)

> Platform reference for SkillSync parser development and cross-platform mapping.

## Overview

Codex is OpenAI's CLI-based coding agent. It stores skills, instructions, and
configuration under `.codex/` (project) and `~/.codex/` (user) directories.
SkillSync also treats `~/.agents/skills` as an alternate user-level Codex skill
root; when both locations exist, `.agents` is preferred for local dedupe.

For SkillSync, the primary Codex surfaces are:

- `SKILL.md` skills following the Agent Skills Standard
- hierarchical `AGENTS.md` instructions
- `config.toml` instruction and runtime settings

Codex selects skills implicitly based on task relevance, with the
`description` frontmatter field driving matching.

SkillSync treats command-like and agent-like Codex surfaces as transport
metadata, not as first-class runtime artifacts. In practice this means:

- command-like content may be preserved as `Type=prompt` compatibility metadata
- agent-like controls stay in `Metadata` or get flattened into normal skill
  instruction content
- `AGENTS.md` remains an instruction chain, not a subagent definition model

Codex also has a **deprecated custom prompts** system at
`~/.codex/prompts/*.md` (invoked as `/prompts:<name>`), but this repo documents
it only as compatibility context rather than a primary sync target.

## Portability Stance

When this repo talks about Codex portability, the intended meaning is:

- **Portable**: `SKILL.md` skills and supporting directories.
- **Partially portable**: `AGENTS.md` content, `config.toml` instruction text,
  and deprecated custom prompt content preserved as compatibility context.
- **Non-portable**: any claim that Codex has a first-class file-backed command
  or subagent model equivalent to Claude commands or Claude agents.

This distinction matters because SkillSync may preserve markdown and metadata
without preserving slash-trigger behavior, placeholder interpolation rules, or
platform-native routing.

## Layout

```text
.codex/                         # Project-level config root
  skills/                       # Project skills
    <skill-name>/
      SKILL.md                  # Required skill definition
      scripts/                  # Optional executable helpers
      references/               # Optional supporting docs
      assets/                   # Optional templates / data
      agents/
        openai.yaml             # Optional platform-specific metadata
  config.toml                   # Project config (overrides user config)

~/.codex/                       # User-level config root
  skills/
    <skill-name>/SKILL.md       # User skills
  config.toml                   # User config
  AGENTS.md                     # User-level instructions
  AGENTS.override.md            # User-level override instructions
  prompts/                      # Deprecated custom prompts (compatibility-only)
    *.md

/etc/codex/                     # System / admin-level
  skills/
    <skill-name>/SKILL.md       # Admin-managed skills
```

## Artifact Types

### Skills (`SKILL.md`)

Each skill lives in its own subdirectory with a `SKILL.md` file containing YAML
frontmatter and markdown body instructions. Codex loads only frontmatter
(`name` + `description`) at startup and defers reading the full body until the
skill is selected for a task.

#### Frontmatter Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Unique skill identifier. |
| `description` | string | yes | Describes when the skill should be used. |
| `version` | string | no | Skill version (semver recommended). |
| `allowed-tools` | string[] | no | Tool/permission allow-list for the skill. |

Additional fields are preserved as opaque metadata by the SkillSync parser.

#### Example

```markdown
---
name: codex-basic
description: Basic Codex skill
---
# Codex Basic Skill

A basic skill for Codex.
```

#### Supporting Directories

| Directory | Purpose |
|---|---|
| `scripts/` | Executable code for deterministic, repeatable operations. |
| `references/` | Supporting documentation the skill can reference. |
| `assets/` | Templates, schemas, and data files. |
| `agents/openai.yaml` | Platform-specific UI settings, invocation policy, and MCP metadata. |

### Instructions (`AGENTS.md`)

Codex discovers `AGENTS.md` files hierarchically from the repository root down
to the current working directory. Files closer to `cwd` appear later in the
concatenated prompt, so nearest instructions have de facto precedence.

Discovery order:

1. `~/.codex/AGENTS.override.md` (global override, if present)
2. `~/.codex/AGENTS.md` (global baseline)
3. `$REPO_ROOT/AGENTS.md`
4. Intermediate directories walking toward `cwd`
5. `$CWD/AGENTS.md`

Codex concatenates all discovered files, separated by blank lines, up to
`project_doc_max_bytes` (default 32 KiB). Empty files are skipped. The
instruction chain is rebuilt on every session start. Custom fallback filenames
can be configured with `project_doc_fallback_filenames` in `config.toml`.

### Config Instructions (`config.toml`)

`~/.codex/config.toml` (user) and `.codex/config.toml` (project) control model
selection, security policy, and inline instructions.

Instruction-related fields:

| Field | Type | Description |
|---|---|---|
| `developer_instructions` | string | Additional developer instructions injected into the session. |
| `model_instructions_file` | string | Path to a file replacing built-in instructions instead of `AGENTS.md`. |
| `project_doc_max_bytes` | int | Maximum bytes read from the `AGENTS.md` chain (default 32768). |
| `project_doc_fallback_filenames` | string[] | Alternative filenames to discover alongside `AGENTS.md`. |
| `skills.config` | array | Skill enablement overrides (`path` + `enabled` per entry). |

Other notable fields:

| Field | Type | Description |
|---|---|---|
| `model` | string | Model to use (for example `gpt-5.3-codex`). |
| `model_provider` | string | Provider ID from `model_providers` (default `openai`). |
| `approval_policy` | string | `untrusted`, `on-request`, or `never`. |
| `sandbox_mode` | string | `read-only`, `workspace-write`, or `danger-full-access`. |

### Deprecated Custom Prompts (`prompts/*.md`)

Codex has a file-backed custom prompts system
(`~/.codex/prompts/*.md`, user scope only) invoked from the slash menu. This
feature is deprecated; OpenAI recommends using Skills instead. SkillSync keeps
this section only to explain compatibility semantics for prompt-like artifacts.
The repo does **not** position prompts as a primary Codex target, and the
parser does **not** currently parse this path.

Treat prompts as **partially portable content only**. They are transport
references, not behavior-preserving command targets, and not a first-class
Codex sync surface in this repo.

#### Frontmatter Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `description` | string | no | One-line summary shown in the slash menu. |
| `argument-hint` | string | no | Hint for expected arguments. |

#### Argument Placeholders

| Syntax | Description |
|---|---|
| `$1`--`$9` | Positional arguments. |
| `$ARGUMENTS` | All arguments joined by spaces. |
| `$NAME` | Named placeholder (for example `$FILE` -> `FILE=path`). |
| `$$` | Literal `$`. |

SkillSync's Codex parser does not currently parse this path.

### Hooks (`hooks.json`)

Codex hook events are runtime behavior, not portable file content. SkillSync
documents them so the portability model can preserve lifecycle boundaries while
still distinguishing behavior from markdown and frontmatter that move across
platforms.

Codex hook events tracked by SkillSync:

| Event | Scope | Meaning | Portability |
|---|---|---|---|
| `SessionStart` | Thread | Session begins, resumes, or clears. | Partially portable lifecycle metadata only. |
| `PreToolUse` | Turn | Before a tool runs. | Non-portable runtime behavior. |
| `PostToolUse` | Turn | After a tool runs. | Non-portable runtime behavior. |
| `UserPromptSubmit` | Turn | User prompt submission boundary. | Non-portable runtime behavior. |
| `Stop` | Turn | Session stop / cleanup boundary. | Non-portable runtime behavior. |

Notes:

- `SessionStart` is the only Codex hook event with thread-level semantics.
- `UserPromptSubmit` and `Stop` are turn-scoped control-plane events.
- `PreToolUse` and `PostToolUse` are tool lifecycle hooks.
- Hook configuration lives in `hooks.json`, separate from `AGENTS.md`, skills,
  and deprecated prompt files.

## Scope Levels

Codex searches skills across multiple scope levels, highest precedence first:

| Priority | Scope | Path | Description |
|---|---|---|---|
| 1 | Project (cwd) | `.codex/skills` | Current working directory skills. |
| 2 | Project (parent) | `$CWD/../.codex/skills` | Parent directory skills (nested repos). |
| 3 | Project (root) | `$REPO_ROOT/.codex/skills` | Repository root skills. |
| 4 | User | `~/.agents/skills` / `~/.codex/skills` | User-wide skills; `.agents` wins when both exist. |
| 5 | Admin | `/etc/codex/skills` | System-administered skills. |
| 6 | System | (bundled) | Skills shipped with Codex. |

When naming conflicts occur across scopes, both skills appear in selector
merging rather than one silently shadowing the other.

## Discovery & Precedence

**Skills**: Searched in scope order (project > user > admin > system). Custom
directories can be configured via the config system. When both
`~/.agents/skills` and `~/.codex/skills` exist, SkillSync treats them as one
logical user scope with `.agents` taking precedence.

**Instructions**: `AGENTS.md` files are concatenated root-to-cwd. Later entries
(deeper directories) override earlier guidance by appearing later in the
prompt. `AGENTS.override.md` takes priority over `AGENTS.md` at the same level.

**Config**: Project `config.toml` overrides user `config.toml`. Profiles
provide named configuration sets selectable at runtime.

## Tool Restrictions

Codex skills can declare `allowed-tools` in `SKILL.md` frontmatter to restrict
which tools the agent may use during skill execution.

Codex also has deprecated custom prompts at `~/.codex/prompts/*.md` and
built-in slash commands in the CLI/TUI (for example `/status`, `/diff`, and
`/review`), but the recommended user-defined reusable instruction surface is
Skills.

## Parser Implementation Notes

- **Parser**: `internal/parser/codex/codex.go`
- **Shared skills parser**: `internal/parser/skills/skills.go`
- **Unified model**: `internal/model/skill.go`

The Codex parser handles three artifact types in precedence order:

1. `SKILL.md` files via the shared `skills.Parser`
2. `config.toml` instructions (`instructions` + `developer_instructions`)
3. `AGENTS.md` files discovered recursively

Name deduplication:

- if a `SKILL.md` skill and a deprecated custom prompt share the same name, the
  `SKILL.md` version wins
- if a `SKILL.md` skill and a config/instruction-derived artifact share the
  same name, the `SKILL.md` version wins

Test fixtures in `testdata/skills/codex/`:

| Fixture | Description |
|---|---|
| `codex-basic/SKILL.md` | Minimal skill with `name` + `description` frontmatter. |
| `codex-structured/SKILL.md` | Skill with `scripts/`, `assets/`, and metadata. |

The parser does **not** currently handle:

- `~/.codex/prompts/`
- any first-class `command` or `agent` runtime model
- `agents/openai.yaml` platform metadata files
- `AGENTS.override.md` as a parsed artifact

## Gaps

- **Deprecated custom prompts are compatibility-only**: Codex has a file-backed
  custom prompts system (`~/.codex/prompts/*.md`, invoked as
  `/prompts:<name>`), but it is deprecated in favor of Skills and not parsed by
  this repo. Claude commands mapped to Codex become skills or lossy prompt
  metadata rather than preserved trigger semantics.
- **`agents/openai.yaml` not parsed**: Platform-specific metadata
  (`allow_implicit_invocation`, UI settings, MCP dependencies) is not yet
  handled by the SkillSync parser.
- **Argument placeholders in skills**: Codex custom prompts support `$1`--`$9`,
  `$ARGUMENTS`, and `$NAME` placeholders, but Skills have no equivalent.
  Claude's `$ARGUMENTS` / `$1` conventions are preserved as literal text during
  sync.
- **Per-skill model hints**: Claude supports per-command `model` fields; Codex
  handles this at config/profile level, not per-skill.

## Sources

- OpenAI Codex Skills docs: https://developers.openai.com/codex/skills/ (accessed 2026-02-22)
- OpenAI Codex AGENTS.md guide: https://developers.openai.com/codex/guides/agents-md (accessed 2026-02-22)
- OpenAI Codex config reference: https://developers.openai.com/codex/config-reference (accessed 2026-02-22)
- Codex hook lifecycle discussions: https://github.com/openai/codex/issues/14754, https://github.com/openai/codex/issues/15490, https://github.com/openai/codex/issues/15497, https://github.com/openai/codex/issues/16226, https://github.com/openai/codex/issues/16933
- SkillSync portability assessment: `docs/platforms/portability-assessment.md`
- SkillSync cross-platform mapping: `docs/platforms/cross-platform-mapping.md`
- Parser source: `internal/parser/codex/codex.go`, `internal/parser/skills/skills.go`
