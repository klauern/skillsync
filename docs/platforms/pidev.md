# Pi.dev

> Platform reference for SkillSync parser development and cross-platform mapping.

## Overview

Pi.dev is an AI coding agent CLI that stores project-local customization under
`.pi/` while also loading shared instruction files from `AGENTS.md`. For the
first pass in SkillSync, Pi.dev should be treated as having four relevant
artifact surfaces:

- `SKILL.md` skills under project and user roots
- markdown prompt templates under `prompts/`
- concatenated `AGENTS.md` instruction files
- `SYSTEM.md` / `APPEND_SYSTEM.md` system-prompt customization

Pi.dev also has a broader package ecosystem with extensions, themes, and other
runtime features. Those are not sync targets in the first pass. This reference
documents them only as non-goals so the portability boundary is explicit.

## Directory Structure

```text
repo/
  AGENTS.md                          # Repo/root instructions
  .pi/
    skills/
      <skill-name>/
        SKILL.md                     # Project skill entrypoint
        scripts/                     # Optional helper scripts
        references/                  # Optional supporting docs
        assets/                      # Optional templates / data
    prompts/
      <name>.md                      # Project prompt template
    SYSTEM.md                        # Project system prompt replacement
    APPEND_SYSTEM.md                 # Project system prompt append-only content
    settings.json                    # Project settings

~/.pi/agent/
  AGENTS.md                          # Global instructions
  skills/
    <skill-name>/SKILL.md            # User skill
  prompts/
    <name>.md                        # User prompt template
  SYSTEM.md                          # Global system prompt replacement
  APPEND_SYSTEM.md                   # Global system prompt append-only content
  settings.json                      # Global settings
```

## Artifact Types

### Skills (`SKILL.md`)

Pi.dev supports first-class skills using the same directory-oriented
`SKILL.md` layout used by other Agent Skills Standard clients.

#### Frontmatter Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Unique skill identifier. |
| `description` | string | yes | Human-readable purpose and trigger guidance. |

Keep the first-pass mapping conservative: model only the shared skill body and
the common `name`/`description` fields unless Pi.dev-specific frontmatter needs
to be added later.

#### Supporting Directories

| Directory | Purpose |
|---|---|
| `scripts/` | Executable helper code. |
| `references/` | Supporting documentation loaded on demand. |
| `assets/` | Templates, schemas, data files. |

### Prompt Templates (`prompts/*.md`)

Prompt templates are reusable markdown snippets invoked by filename stem. A
template at `.pi/prompts/review.md` or `~/.pi/agent/prompts/review.md` is
expanded via `/review`.

Prompt bodies can use placeholder-style interpolation such as `{{focus}}`. The
placeholder syntax is useful transport metadata, but SkillSync should not claim
cross-platform runtime equivalence for expansion semantics.

For the first pass, treat prompt templates as:

- markdown prompt artifacts
- filename-driven slash triggers
- content-preserving but behavior-changing when synced to other platforms

### Instructions (`AGENTS.md`)

Pi.dev loads `AGENTS.md` files as always-on instructions. The chain starts with
the global file at `~/.pi/agent/AGENTS.md`, then walks parent directories from
the repository root down to the current working directory, concatenating all
matches.

This makes Pi.dev instruction files conceptually close to Codex `AGENTS.md`,
but still separate from system-prompt replacement.

### System Prompt Customization (`SYSTEM.md`, `APPEND_SYSTEM.md`)

Pi.dev exposes a separate system-prompt layer:

| File | Behavior |
|---|---|
| `.pi/SYSTEM.md`, `~/.pi/agent/SYSTEM.md` | Replace the default system prompt at project or user scope. |
| `.pi/APPEND_SYSTEM.md`, `~/.pi/agent/APPEND_SYSTEM.md` | Append to the default system prompt without replacing it. |

SkillSync should document this layer, but treat it as instruction-level
portability with caveats, not as a skill or prompt artifact.

## Scope Levels

| Scope | Paths | Notes |
|---|---|---|
| User | `~/.pi/agent/skills/`, `~/.pi/agent/prompts/`, `~/.pi/agent/AGENTS.md`, `~/.pi/agent/SYSTEM.md` | Baseline personal configuration |
| Project | `.pi/skills/`, `.pi/prompts/`, `AGENTS.md`, `.pi/SYSTEM.md` | Repo-local overrides and prompt content |
| Package / Extension / Theme | Pi package ecosystem | Explicitly out of scope for the first SkillSync pass |

Precedence is surface-dependent:

- Skills/prompts/settings are project-first over user defaults.
- `AGENTS.md` instructions concatenate from global to repo-root to `cwd`.
- `SYSTEM.md` customization is a separate replacement/append layer, not just
  another `AGENTS.md` file.

## First-Pass Non-Goals

The following Pi.dev features should not be treated as sync targets in this doc
set or in the first implementation pass:

- packages
- extensions
- themes
- workspace runtime settings beyond documented instruction/prompt roots
- package-installed prompt or skill provenance

If content from those surfaces is valuable later, it should be re-authored into
portable skills, prompt templates, or instruction files instead of synced as a
runtime-native Pi package.

## Portability Notes

- Pi.dev skills are high-value portable content because they align with the
  shared `SKILL.md` model.
- Pi.dev prompt templates are portable as content, but not as guaranteed
  placeholder or slash-trigger behavior.
- Pi.dev `AGENTS.md` content is portable as instructions, though loading order
  still differs by platform.
- `SYSTEM.md` and `APPEND_SYSTEM.md` are useful instruction-layer references,
  but they are lossy when moved to platforms without a separate system-prompt
  override file.

## Sources

- [Pi.dev packages index](https://pi.dev/packages)
- [Pi Monorepo](https://github.com/badlogic/pi-mono)
