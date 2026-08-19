# Cursor

> Platform reference for SkillSync parser development and cross-platform mapping.

## Overview

Cursor is an AI-powered IDE (fork of VS Code) that provides contextual AI assistance through rules,
skills, commands, and custom modes. Configuration lives in the `.cursor/` directory at both project
and user level. Cursor supports the [Agent Skills Standard](https://agentskills.io/home) (`SKILL.md`)
alongside its own legacy rule format (`.md`/`.mdc` files with frontmatter).

Cursor also auto-discovers skills from other platforms' directories (`.claude/skills/`,
`.codex/skills/`) for cross-platform compatibility, making skills portable without duplication.

## Directory Structure

```text
<project>/
  .cursor/
    skills/               # Project-scoped Agent Skills Standard skills
      <skill-name>/
        SKILL.md
        scripts/
        references/
        assets/
    rules/                # Project-scoped rules (auto-applied)
      *.md
      *.mdc               # Legacy Cursor markdown format
    commands/             # Project-scoped command/prompt files
      *.md
  .claude/skills/         # Also discovered (cross-platform compat)
  .codex/skills/          # Also discovered (cross-platform compat)
  AGENTS.md               # Alternative to rules (simpler, single-file)

~/.cursor/
  skills/                 # User-scoped skills (global)
    <skill-name>/
      SKILL.md
  rules/                  # User-scoped rules (global)
    *.md
    *.mdc
  commands/               # User-scoped command files (global)
    *.md
  modes.json              # Custom mode definitions
~/.claude/skills/         # Also discovered (cross-platform compat)
~/.codex/skills/          # Also discovered (cross-platform compat)
```

## Artifact Types

### Skills (`SKILL.md`)

Skills follow the Agent Skills Standard, shared across Claude Code, Codex, and Cursor. Each skill
lives in a named subdirectory containing a `SKILL.md` file with YAML frontmatter. Optional
subdirectories (`scripts/`, `references/`, `assets/`) provide supporting files.

#### Frontmatter Schema

| Field                      | Type               | Required | Description |
|:---------------------------|:-------------------|:---------|:------------|
| `name`                     | `string`           | yes      | Lowercase identifier matching folder name |
| `description`              | `string`           | yes      | Explains purpose and relevance; Cursor uses this to decide when to apply the skill |
| `tools`                    | `string[]`         | no       | Tool permissions required |
| `type`                     | `string`           | no       | `skill` (default) or `prompt` |
| `trigger`                  | `string`           | no       | Slash command trigger (e.g., `/review`) |
| `scope`                    | `string`           | no       | `user`, `repo`, `plugin`, `system` |
| `disable-model-invocation` | `bool`             | no       | When `true`, requires explicit `/skill-name` invocation |
| `license`                  | `string`           | no       | License identifier |
| `compatibility`            | `map[string]string`| no       | Platform/environment requirements |
| `metadata`                 | `map[string]string`| no       | Custom key-value pairs |
| `scripts`                  | `string[]`         | no       | Supporting script files |
| `references`               | `string[]`         | no       | Reference document files |
| `assets`                   | `string[]`         | no       | Asset files (templates, images, data) |

Body is markdown instructions. Supporting directories are auto-detected from the skill folder.

When Cursor starts, it automatically discovers skills from skill directories and makes them
available to Agent. The agent contextually determines relevance and can also be manually invoked
via `/skill-name` commands.

#### Example

```yaml
---
name: go-review
description: Review Go code for idiomatic patterns and common pitfalls
tools: ["Read", "Grep"]
---

Review the Go code for:
- Idiomatic error handling with %w
- Proper context propagation
- Resource cleanup with defer
```

### Rules (`*.md` / `*.mdc`)

Rules are auto-applied context instructions stored only as `.cursor/rules/*.mdc`. They use YAML frontmatter
for scope control. The `.mdc` extension is the legacy Cursor Markdown format; `.md` is also
supported. Cursor recommends keeping rules under 500 lines and using `@filename` references
rather than copying content.

An `AGENTS.md` file in the project root serves as a simpler alternative to the rules directory
for basic project-wide instructions.

#### Frontmatter Schema

| Field         | Type       | Required | Description |
|:--------------|:-----------|:---------|:------------|
| `description` | `string`   | no       | Presented to the agent to decide if the rule should be applied |
| `paths`       | `string[]` | no       | Current path patterns for file matching |
| `globs`       | `string[]` | no       | Legacy fallback when `paths` is absent |
| `alwaysApply` | `bool`     | no       | If `true`, applies to all files regardless of globs |

#### Application Modes

Cursor supports four rule application modes:

| Mode                     | Trigger | Description |
|:-------------------------|:--------|:------------|
| **Always**               | `alwaysApply: true` | Activates for every chat session |
| **Apply Intelligently**  | `description` set, no `globs`/`alwaysApply` | Agent decides relevance based on `description` field |
| **Apply to Specific Files** | `paths: [...]` (`globs` legacy fallback) | Triggers when files match path patterns |
| **Apply Manually**       | none of the above | Invoked via `@rule-name` in chat |

#### Example

```markdown
---
description: Go error handling conventions
paths: ["*.go"]
alwaysApply: false
---

Always wrap errors with fmt.Errorf and %w verb...
```

### Commands (`.cursor/commands/*.md`)

Command files provide prompt templates that can be invoked as slash commands or linked to custom
modes. They are stored in `.cursor/commands/` at project or global scope.

Command files use Markdown format. The filename becomes the command name (e.g., `review.md` becomes
available as a prompt). Cursor's command system is tightly integrated with the modes system.

### Modes (`~/.cursor/modes.json`)

Custom modes define named AI behavior profiles that can include:

- A system prompt / instruction set
- Linked command files
- Tool/capability restrictions
- Slash command aliases

Modes are configured globally in `~/.cursor/modes.json`. Each mode can reference prompt files and
define slash-command-style invocation behavior through the `slashCommand` metadata field.

## Scope Levels

| Level              | Path | Precedence |
|:-------------------|:-----|:-----------|
| Team (Enterprise)  | Organization-managed | Highest |
| Project            | `.cursor/skills/`, `.cursor/rules/`, `.cursor/commands/` | High |
| Global (user)      | `~/.cursor/skills/`, `~/.cursor/rules/`, `~/.cursor/commands/` | Lower |

Application order for rules: Team Rules > Project Rules > User Rules. Rules merge when applicable;
earlier sources take precedence during conflicts.

Project-level artifacts override global artifacts when names collide.

## Discovery & Precedence

1. **Skills**: `SKILL.md` in subdirectories takes precedence over legacy `.md`/`.mdc` files with
   the same name.
2. **Cross-platform discovery**: Skills from `.claude/skills/` and `.codex/skills/` are
   auto-discovered alongside `.cursor/skills/` at both project and user level.
3. **Rules**: Applied based on glob specificity. More specific glob patterns take precedence over
   broader ones. `alwaysApply` rules are always included. Nested `AGENTS.md` files provide
   directory-specific instructions with inheritance.
4. **Commands**: Project-level command files override global command files with the same filename.
5. **Modes**: Global only (no project-level override mechanism).

Reference files inside skill directories (e.g., `patterns/`, `references/`, `templates/`) are
excluded from legacy rule parsing to prevent misidentification.

## Tool Restrictions

Cursor does not have a universal per-artifact `tools` field across all artifact types:

- **Skills** (`SKILL.md`): support the `tools` field in frontmatter (via Agent Skills Standard)
- **Rules**: no `tools` field; tool access is controlled by mode settings and IDE configuration
- **Commands/Modes**: tool restrictions are configured within mode definitions, not individual
  command files

## Parser Implementation Notes

### Parser Location

`internal/parser/cursor/cursor.go`

### Parsing Strategy

The Cursor parser uses a two-phase approach:

1. **Phase 1 -- SKILL.md parsing**: Uses the shared `internal/parser/skills/skills.go` parser to
   discover and parse `SKILL.md` files (case-insensitive: `SKILL.md`, `skill.md`, `Skill.md`).
   These take precedence and their names are recorded in a `seenNames` map.

2. **Phase 2 -- Legacy file parsing**: Discovers `*.md` and `*.mdc` files via
   `parser.DiscoverFiles()` with patterns `["*.md", "*.mdc", "**/*.md", "**/*.mdc"]`, filtering
   out:
   - `SKILL.md` files (already parsed in phase 1)
   - Files inside skill directories (prevents reference files from being treated as skills)

   Each legacy file is parsed with `parseSkillFile()`.

### Frontmatter Handling

- Split via `parser.SplitFrontmatter()` (detects `---` and `+++` delimiters)
- Parsed via `parser.ParseYAMLFrontmatter()` (YAML to `map[string]any`)
- All frontmatter fields except `name` stored in `Metadata` map, including `globs` (converted to
  string representation like `[*.go internal/**/*.go]`) and `alwaysApply`
- Array values are joined into space-separated bracket notation

### Name Derivation

- **SKILL.md**: `name` from frontmatter, or parent directory name if absent
- **Legacy `.md`/`.mdc`**: `name` from frontmatter if present, otherwise filename stem (e.g.,
  `go-rule.md` becomes `go-rule`)

### Description Extraction

Cursor rules typically do not use a `description` frontmatter field; the parser sets description
to empty string for legacy files. SKILL.md files use the Agent Skills Standard `description` field.

### No First-Class Commands Parser

The current parser handles skills and rules but does not separately parse `.cursor/commands/`
command files as a distinct artifact type.

### Test Fixtures

Test fixtures are in `testdata/skills/cursor/`:

| Fixture | Purpose |
|:--------|:--------|
| `cursor-basic/SKILL.md` | Minimal skill with name and description |
| `cursor-with-globs/SKILL.md` | Skill with glob patterns and `alwaysApply` |

Additional coverage via inline test cases in `internal/parser/cursor/cursor_test.go`:
- Legacy `.mdc` files with `globs`/`alwaysApply`; plain `.md` files under `.cursor/rules` are inactive
- Mixed format parsing (SKILL.md + legacy)
- SKILL.md precedence over legacy files with same name
- Skill directory exclusion (reference files not treated as skills)
- Nested directory structures
- Invalid skill name handling

## Gaps

- **Command behavior tied to modes**: `.cursor/commands/*.md` invocation semantics depend on mode
  configuration. There is no standalone slash-command file format equivalent to Claude's
  `.claude/commands/*.md`.
- **No `argument-hint` equivalent**: Cursor has no documented frontmatter field for providing
  argument hints to slash commands.
- **No per-command tool field**: unlike Claude's `allowed-tools`, individual command files cannot
  restrict tool access.
- **Mode metadata not file-based**: `modes.json` is a single JSON file rather than per-mode
  markdown files, making it harder to map from individual Claude/Codex prompt artifacts.
- **`.mdc` format**: the legacy `.mdc` extension is functionally identical to `.md` with YAML
  frontmatter, but is specific to Cursor and not recognized by other platforms.
- **Team/Enterprise rules**: team-level rules (Team/Enterprise plans) are not parsed by SkillSync
  as they are managed server-side.
- **`AGENTS.md` parsing**: Cursor supports `AGENTS.md` as an alternative to the rules directory
  but SkillSync does not parse these files in the Cursor context.

## Sources

- [Cursor Skills documentation](https://cursor.com/docs/context/skills) -- retrieved 2026-02-22
- [Cursor Rules documentation](https://cursor.com/docs/context/rules) -- retrieved 2026-02-22
- Cursor full LLM docs: https://cursor.com/docs/llms-full.txt (accessed 2026-02-09)
- [Agent Skills Standard](https://agentskills.io/home) -- open standard for portable AI agent skills
- `internal/parser/cursor/cursor.go` -- SkillSync Cursor parser implementation
- `internal/parser/skills/skills.go` -- shared SKILL.md parser
- `internal/model/skill.go` -- unified skill model
- `docs/platforms/portability-assessment.md` -- focused portability assessment
- `docs/platforms/cross-platform-mapping.md` -- broad cross-platform reference
## v2 reference boundary (verified 2026-08-18)

Official source: https://cursor.com/docs/skills. Canonical roots are
`.cursor/skills/` and `~/.cursor/skills/`; `.agents/skills/` is discovery-only.
`.mdc` rules, nested AGENTS.md, subagents, hooks, and plugins are native-only.
