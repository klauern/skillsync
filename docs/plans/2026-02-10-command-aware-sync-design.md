# Command-Aware Sync Design (skillsync-c1g.4)

Date: 2026-02-10
Status: Proposed (implementation-ready)

## Goal

Add first-class command/prompt support to discovery and sync without regressing
existing skill workflows.

## Non-goals

- Implementing parser/sync code in this design doc.
- Guaranteeing one-to-one runtime command behavior across platforms where the
  underlying products differ (for example explicit slash trigger semantics).

## Unified Model

Reuse `model.Skill` as the canonical artifact model and standardize these fields:

- `Type`:
  - `skill` (default)
  - `prompt` (covers slash-command-like artifacts)
- `Trigger`:
  - Normalized slash trigger (for example `/review`) when source platform has one.
  - Empty when unknown/unsupported.
- `Metadata`:
  - Preserve non-portable fields (for example `argument-hint`, per-command
    `model`) as passthrough metadata for round-tripping.

No new top-level struct is required for phase 1; `model.Skill` already has
`Type` and `Trigger`.

## Artifact Taxonomy by Platform

- Claude:
  - Skills: `.claude/skills/**/SKILL.md`
  - Commands/prompts: `.claude/commands/*.md`, `~/.claude/commands/*.md`
- Cursor:
  - Skills: `.cursor/skills/**/SKILL.md`
  - Prompt-like command artifacts: `.cursor/commands/*.md` (plus mode linkage)
  - Rules remain separate behavior (`.cursor/rules/*.mdc`) and are out of scope
    for phase 1 sync-as-command parity.
- Codex:
  - Skills/prompts: `.codex/skills/**/SKILL.md`
  - `AGENTS.md` + config instructions are instruction artifacts, not slash-command
    artifacts.
  - `~/.codex/prompts` exists in local environment but is undocumented in official
    docs; treat as experimental/off by default.

## Discovery Semantics

### Default behavior

- `discover`:
  - Includes both `skill` and `prompt` by default.
  - Existing `--type` filter remains the primary selector.

### Parser requirements

- Parsers must assign `Type=prompt` when artifact is a command/prompt source:
  - Claude command files
  - Cursor command files
  - SKILL.md with `type: prompt|command|slash-command` aliases
- For Claude/Cursor filename-based command files, derive `Trigger` as `/<stem>`.

## Sync/Delete CLI Behavior

Add type-scoped controls to `sync` and `delete`:

- New flags:
  - `--type` / `-t`: `skill`, `prompt`, `all` (comma-separated allowed)
  - `--include-prompts` (alias for `--type skill,prompt`)
- Default for `sync` and `delete`: `--type skill`
  - This is the explicit guardrail for "command-aware sync is opt-in."

Rationale:
- Keeps historical "skill sync" expectation stable.
- Gives deterministic enablement for command/prompt artifacts.

## Config Behavior

Add sync-level default type policy:

```yaml
sync:
  default_strategy: overwrite
  include_types: [skill]   # skill|prompt
```

Rules:
- CLI `--type` overrides config.
- Missing `include_types` defaults to `[skill]`.

## Precedence and Collision Rules

### Intra-platform discovery precedence

- Existing scope precedence remains (`builtin < system < admin < user < repo < plugin`),
  except platform-native exceptions.
- Claude exception from docs:
  - Same-name skill beats same-name command.
- Cursor command collision:
  - Project command file beats global command file.

### Cross-platform sync identity

Artifact key for conflict/match:

1. `(normalized name, type)` primary
2. If `type=prompt` and `Trigger` exists on both sides, include `Trigger` in
   comparison for conflict diagnostics (not primary identity in phase 1).

Reason:
- Target platforms may not preserve trigger semantics exactly (especially Codex).

## Mapping Strategy

### Claude prompt -> Codex

- Map to Codex `SKILL.md` with `Type=prompt`.
- Preserve `Trigger` in metadata and body comments when needed.
- Mark lossy fields in result warnings:
  - explicit slash trigger behavior
  - argument placeholder semantics (`$ARGUMENTS`, `$1`)
  - per-command model selection

### Claude prompt -> Cursor

- Map to `.cursor/commands/<name>.md` (preferred) with `Type=prompt`.
- Preserve unmapped fields in metadata/frontmatter comments.
- If mode metadata required by Cursor for slash invocation, mark as partial and
  emit warning unless mode config generation is enabled in a future phase.

### Codex prompt/skill -> Claude

- `Type=prompt`: prefer `.claude/commands/<name>.md`
- `Type=skill`: prefer `.claude/skills/<name>/SKILL.md`

## Lossy Conversion Policy

When transformation cannot preserve behavior exactly:

- Continue sync unless `--strict-types` (future flag) is enabled.
- Record warning in sync results:
  - `lossy_trigger_mapping`
  - `lossy_argument_semantics`
  - `lossy_platform_field_drop`

Warnings must include source path and target artifact path.

## Backward Compatibility and Migration

1. Default sync/delete remain skills-only (`type=skill`), preventing accidental
   command artifact propagation.
2. Existing discover behavior remains broad; users can still narrow via
   `discover --type`.
3. Current configs without `sync.include_types` continue to work unchanged.
4. Existing parser output with empty `Type` is interpreted as `skill`.
5. Experimental `.codex/prompts` support, if added, must be behind explicit
   opt-in config/flag and clearly labeled undocumented.

## Implementation Plan (linked tasks)

1. `skillsync-c1g.5`:
   - Add Claude command parser paths + type/trigger assignment.
2. `skillsync-c1g.6`:
   - Implement prompt-aware transform rules and lossy warnings.
3. `skillsync-c1g.7`:
   - Add `sync/delete --type` and config `sync.include_types`.
4. `skillsync-c1g.8`:
   - Add end-to-end tests for skill-only default and prompt opt-in behavior.

## Acceptance Criteria Mapping

- AC1 (model + architecture docs):
  - This design defines model semantics and precedence.
  - `docs/architecture.md` updated with command-aware extension.
- AC2 (CLI behavior):
  - `sync/delete --type` and `--include-prompts` behavior specified.
- AC3 (compat/migration):
  - Skills-only default + migration policy + lossy-warning strategy defined.
