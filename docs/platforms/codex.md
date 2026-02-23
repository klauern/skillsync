# Codex (OpenAI CLI)

> Platform reference for SkillSync parser development and cross-platform mapping.

## Overview

Codex is OpenAI's CLI-based coding agent. It stores skills, instructions, and
configuration under the `.codex/` (project) and `~/.codex/` (user) directories.
Skills follow the Agent Skills Standard (`SKILL.md` in a named subdirectory),
while broader instructions are delivered through hierarchical `AGENTS.md` files
and `config.toml` settings.

Codex selects skills implicitly based on task relevance -- the `description`
frontmatter field drives matching. Codex also has a **custom prompts** system
at `~/.codex/prompts/*.md` (invoked as `/prompts:<name>`), but this feature is
**deprecated** in favor of Skills.

## Directory Structure

```
.codex/                          # Project-level config root
  skills/                        # Project skills
    <skill-name>/
      SKILL.md                   # Required -- skill definition
      scripts/                   # Optional executable helpers
      references/                # Optional supporting docs
      assets/                    # Optional templates / data
      agents/
        openai.yaml              # Optional platform-specific metadata
  config.toml                    # Project config (overrides user config)

~/.codex/                        # User-level config root
  skills/                        # User skills
    <skill-name>/SKILL.md
  config.toml                    # User config
  AGENTS.md                      # User-level instructions
  AGENTS.override.md             # User-level override instructions
  prompts/                       # Custom Prompts (deprecated; use Skills instead)
    *.md

/etc/codex/                      # System / admin-level
  skills/                        # Admin-managed skills
    <skill-name>/SKILL.md
```

## Artifact Types

### Skills (`SKILL.md`)

Each skill lives in its own subdirectory with a `SKILL.md` file containing YAML
frontmatter and markdown body instructions. Codex loads only the frontmatter
(name + description) at startup and defers reading the full body until the skill
is selected for a task.

#### Frontmatter Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Unique skill identifier. |
| `description` | string | yes | Describes when the skill triggers; drives implicit matching. |
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

A basic skill for the Codex platform.
```

#### Supporting Directories

| Directory | Purpose |
|---|---|
| `scripts/` | Executable code for deterministic, repeatable operations. |
| `references/` | Supporting documentation the skill can reference. |
| `assets/` | Templates, schemas, data files. |
| `agents/openai.yaml` | Platform-specific UI display settings, invocation policy (`allow_implicit_invocation`), and MCP tool dependencies. |

### Instructions (`AGENTS.md`)

Codex discovers `AGENTS.md` files hierarchically from the repository root down
to the current working directory. Files closer to `cwd` appear later in the
concatenated prompt, so **nearest (deepest) instructions take precedence**.

Discovery order:

1. `~/.codex/AGENTS.override.md` (global override, if present)
2. `~/.codex/AGENTS.md` (global baseline)
3. `$REPO_ROOT/AGENTS.md`
4. Intermediate directories walking toward `cwd`
5. `$CWD/AGENTS.md`

Codex concatenates all discovered files (separated by blank lines) up to
`project_doc_max_bytes` (default 32 KiB). Empty files are skipped. The
instruction chain is rebuilt on every session start.

Custom fallback filenames can be configured via
`project_doc_fallback_filenames` in `config.toml`.

### Config Instructions (`config.toml`)

The `~/.codex/config.toml` (user) and `.codex/config.toml` (project) files
control model selection, security policy, and inline instructions.

Instruction-related fields:

| Field | Type | Description |
|---|---|---|
| `developer_instructions` | string | Additional developer instructions injected into the session. |
| `model_instructions_file` | string | Path to a file replacing built-in instructions (instead of `AGENTS.md`). |
| `project_doc_max_bytes` | int | Maximum bytes read from `AGENTS.md` chain (default 32768). |
| `project_doc_fallback_filenames` | string[] | Alternative filenames to discover alongside `AGENTS.md`. |
| `skills.config` | array | Skill enablement overrides (`path` + `enabled` per entry). |

Other notable config fields:

| Field | Type | Description |
|---|---|---|
| `model` | string | Model to use (e.g., `gpt-5.3-codex`). |
| `model_provider` | string | Provider ID from `model_providers` (default `openai`). |
| `approval_policy` | string | `untrusted`, `on-request`, or `never`. |
| `sandbox_mode` | string | `read-only`, `workspace-write`, or `danger-full-access`. |
| `profiles.<name>.*` | table | Named profile overrides for any supported setting. |

### Prompts (`~/.codex/prompts/*.md`) — Deprecated

Custom prompts are user-scoped markdown files invoked as `/prompts:<name>` from
the slash menu. This feature is **deprecated**; OpenAI recommends using Skills
instead.

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
| `$NAME` | Named placeholder (e.g., `$FILE` -> `FILE=path`). |
| `$$` | Literal `$`. |

The SkillSync Codex parser does not currently parse this path.

## Scope Levels

Codex searches for skills across multiple scope levels, highest precedence
first:

| Priority | Scope | Path | Description |
|---|---|---|---|
| 1 | Project (cwd) | `.agents/skills` / `.codex/skills` | Current working directory skills. |
| 2 | Project (parent) | `$CWD/../.agents/skills` | Parent directory skills (nested repos). |
| 3 | Project (root) | `$REPO_ROOT/.agents/skills` | Repository root skills. |
| 4 | User | `~/.codex/skills` | User-wide skills. |
| 5 | Admin | `/etc/codex/skills` | System-administered skills. |
| 6 | System | (bundled) | Skills shipped with Codex. |

When naming conflicts occur across scopes, both skills appear in the selector
rather than merging.

## Discovery & Precedence

**Skills**: Searched in scope order (project > user > admin > system). Custom
directories can be configured via the config system. Name collisions across
scopes surface both versions rather than silently shadowing.

**Instructions**: `AGENTS.md` files are concatenated root-to-cwd. Later entries
(deeper directories) override earlier guidance by appearing later in the prompt.
`AGENTS.override.md` takes priority over `AGENTS.md` at the same level.

**Config**: Project `config.toml` overrides user `config.toml`. Profiles
provide named configuration sets selectable at runtime.

## Tool Restrictions

Codex skills can declare `allowed-tools` in SKILL.md frontmatter to restrict
which tools the agent may use during skill execution.

Codex has a **deprecated custom prompts system** (`~/.codex/prompts/*.md`)
invoked as `/prompts:<name>`. Built-in slash commands also exist in the CLI/TUI
(e.g., `/status`, `/diff`, `/review`). The recommended approach for user-defined
reusable instructions is Skills.

## Parser Implementation Notes

- **Parser**: `internal/parser/codex/codex.go`
- **Shared skills parser**: `internal/parser/skills/skills.go` (Agent Skills Standard `SKILL.md` format)
- **Unified model**: `internal/model/skill.go`

The Codex parser handles three artifact types in precedence order:

1. **SKILL.md** files (via shared `skills.Parser`) -- highest precedence
2. **config.toml** instructions (`instructions` + `developer_instructions`)
3. **AGENTS.md** files (recursive discovery)

Name deduplication: if a `SKILL.md` skill and a `config.toml` or `AGENTS.md`
skill share the same name, the `SKILL.md` version wins.

**Test fixtures** in `testdata/skills/codex/`:

| Fixture | Description |
|---|---|
| `codex-basic/SKILL.md` | Minimal skill with name + description frontmatter. |
| `codex-structured/SKILL.md` | Skill with `scripts/`, `assets/`, and license metadata. |

The parser does **not** currently handle:
- `~/.codex/prompts/` directory
- `agents/openai.yaml` platform metadata files
- `AGENTS.override.md` variant

## Gaps

- **Custom prompts deprecated**: Codex has a file-backed custom prompts system
  (`~/.codex/prompts/*.md`, invoked as `/prompts:<name>`) but it is deprecated
  in favor of Skills. Claude commands mapped to Codex become skills with lossy
  trigger semantics.
- **`agents/openai.yaml` not parsed**: Platform-specific metadata
  (`allow_implicit_invocation`, UI settings, MCP dependencies) is not yet
  handled by the SkillSync parser.
- **Argument placeholders in skills**: Codex custom prompts support `$1`--`$9`,
  `$ARGUMENTS`, and `$NAME` placeholders, but Skills have no equivalent. Claude's
  `$ARGUMENTS` / `$1` conventions are preserved as literal text during sync.
- **Per-skill model hints**: Claude supports per-command `model` fields; Codex
  handles this at the config/profile level, not per-skill.

## Sources

- OpenAI Codex Skills docs: https://developers.openai.com/codex/skills/ (accessed 2026-02-22)
- OpenAI Codex AGENTS.md guide: https://developers.openai.com/codex/guides/agents-md (accessed 2026-02-22)
- OpenAI Codex config reference: https://developers.openai.com/codex/config-reference (accessed 2026-02-22)
- SkillSync research: `docs/research-codex-prompts-commands.md` (2026-02-09, superseded by this document)
- SkillSync cross-platform research: `docs/skill-formats-research.md` (superseded by `docs/platforms/`)
- Parser source: `internal/parser/codex/codex.go`, `internal/parser/skills/skills.go`
