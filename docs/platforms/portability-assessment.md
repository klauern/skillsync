# Portability Assessment: Claude Code, Codex CLI, Pi.dev, and SkillSync

This note summarizes what is genuinely portable between Claude Code, Codex CLI,
and Pi.dev, what is only partially portable, and where this repository still
leaves gaps.

It is intentionally narrower than `docs/platforms/cross-platform-mapping.md`.
The goal here is not to restate every file location, but to answer a practical
question: **which artifact types can move across these CLIs without changing
behavior?**

The machine-readable companions are `docs/platforms/schema.yaml` and
`docs/platforms/portability-snapshot.yaml`. This note explains the portability
tradeoffs in prose; the schema carries the structured artifact inventory, and
the snapshot carries the structured portability claims that should stay aligned
with this assessment.

## Executive Summary

- SkillSync currently has first-class parser/runtime support for Claude Code,
  Cursor, and Codex. Copilot, Gemini CLI, and Pi.dev are documented here as
  adjacent portability references.
- `SKILL.md` skills are the most portable artifact type. The Agent Skills
  Standard is shared, and all three products use `name`, `description`,
  markdown body content, and supporting directories in broadly similar ways.
- Portability in this repo is mostly about carrying content and intent across
  tools. It is not, by itself, a guarantee that the destination CLI will
  preserve the source CLI's invocation, loading, or enforcement behavior.
- Commands/prompts are only partially portable. Claude Code has first-class
  slash-command files; Codex CLI’s documented surface is skills + `AGENTS.md`
  instructions, and this repo treats Codex prompt files as deprecated
  compatibility content rather than a behavior-preserving target.
- Pi.dev adds a useful middle ground: prompt templates are first-class markdown
  artifacts, but their placeholder expansion and system-prompt layers are still
  not portable 1:1 to other CLIs.
- Subagents/agents are not portable in a 1:1 way. Claude Code has explicit
  `.claude/agents/*.md` files; Codex CLI has `AGENTS.md` instruction chaining,
  not a matching subagent file model; Pi.dev packages/extensions/themes are not
  first-pass sync targets here.
- The unified `model.Skill` type is intentionally transport-first.
  Command-like content can ride along as `Type=prompt`, but that does not
  create a native Codex or Pi.dev runtime equivalent.
- Always-on instructions are conceptually similar but semantically different.
  `CLAUDE.md`, `AGENTS.md`, and Pi.dev’s instruction + system-prompt layers can
  all carry persistent project guidance, but they load differently and should
  not be treated as interchangeable.
- Claude plugin-installed skills and Pi.dev package-installed runtime features
  are provenance-heavy surfaces. Their content can be copied, but the install
  context does not transfer.
- The structured portability snapshot records these claims explicitly so they
  can be checked mechanically later instead of being inferred only from prose.

## Portability Classes Used In This Repo

SkillSync uses three portability classes in the docs and transport model:

| Class | Meaning | Typical examples |
|---|---|---|
| Portable | The destination can usually keep both the content shape and the core author intent with limited adaptation. | `SKILL.md` skill body, shared skill directories |
| Partially portable | The content can move, but runtime behavior, invocation, precedence, or interpolation semantics change. | Commands/prompts, `AGENTS.md`/`CLAUDE.md`, tool restrictions |
| Non-portable | The source behavior depends on a platform-specific runtime surface that has no direct equivalent. | Claude subagents, plugin/package provenance, Pi.dev system-prompt replacement semantics |

The rest of this note uses those labels deliberately:

- **Portable** means "safe to move as a first-pass artifact."
- **Partially portable** means "safe to transport, unsafe to claim runtime
  parity."
- **Non-portable** means "preserve as metadata, flatten into another artifact,
  or re-author manually."

## Artifact Portability Matrix

| Artifact | Claude Code | Codex CLI | Pi.dev | Portability |
|---|---|---|---|---|
| `SKILL.md` skill | Native, first-class | Native, first-class | Native, first-class | High for core skill content; medium for advanced fields |
| Slash command / prompt | Native `.claude/commands/*.md` | Not a first-class supported target in this repo; `~/.codex/prompts` is deprecated/unsupported here | Native `.pi/prompts/*.md` templates | Low to medium as content; low for behavior |
| Subagent / agent | Native `.claude/agents/*.md` | No matching file model | No first-pass sync target | None to low |
| Always-on instructions | `CLAUDE.md` | `AGENTS.md` | `AGENTS.md` + `SYSTEM.md` / `APPEND_SYSTEM.md` | Medium, but not behaviorally identical |
| Plugin / package-installed content | Native Claude plugin scope | No equivalent plugin provenance | Packages/extensions/themes exist but are not first-pass sync targets | Low, content-only copy |

## What Is Portable

### 1. Core skills

The portable core is the shared `SKILL.md` structure:

- `name`
- `description`
- markdown body instructions
- `scripts/`
- `references/`
- `assets/`

That aligns with the Agent Skills Standard and is the strongest interop story in
the repo.

Practical implication:

- A skill that stays inside the common Agent Skills subset can usually move
  between Claude Code, Codex CLI, and Pi.dev with limited loss.
- Once a skill leans on platform-specific metadata, portability drops quickly.

### 2. High-value metadata that usually survives

These fields are still useful when moving a skill across the CLIs, even if the
runtime does not interpret them identically:

- `allowed-tools`
- `license`
- `compatibility`
- `metadata`

Those fields are mostly safe as transport metadata.

### 3. Portable content vs. non-portable behavior

The most important distinction in this assessment is the difference between:

- **portable content**: markdown instructions, supporting directories, and a
  small shared frontmatter subset
- **non-portable behavior**: when an artifact appears in the UI, how it is
  invoked, how it participates in prompt construction, and which runtime
  controls the CLI enforces

That distinction explains why a synced artifact can still be useful even when
it is lossy:

- Claude skill instructions can move to Codex or Pi.dev as skill content and
  remain actionable.
- Claude-specific controls such as `disable-model-invocation` or
  `user-invocable` do not become native Codex or Pi.dev skill controls.
- Claude subagent routing via `context: fork` and `agent` can survive as
  metadata, but it does not create a Codex or Pi.dev subagent runtime.
- Claude `hooks` and dynamic context injection remain Claude runtime features
  even if the raw text is preserved.
- The shared model does not add a separate command or agent top-level type; it
  keeps prompt/trigger metadata inside `model.Skill` for lossy transport.

So SkillSync should describe many conversions as **content-preserving but
behavior-changing** rather than as full portability.

## What Is Only Partially Portable

### 1. Command and prompt artifacts

Claude Code’s command files are user-visible, slash-invokable prompt artifacts.
Codex CLI’s current documented path is different:

- Codex docs emphasize skills, `AGENTS.md`, configuration, and related
  surfaces.
- This repo currently treats Codex prompt files as deprecated and does not
  parse them.
- The shared model can store a prompt as `Type=prompt`, but that is only a
  transport marker, not a guarantee of equivalent behavior.

Pi.dev prompt templates improve the content story because they are first-class
markdown prompt files, but the runtime still differs:

- filename-derived `/name` triggers are similar but not identical across CLIs
- placeholder expansion in Pi.dev uses template-style `{{name}}` syntax
- Pi.dev prompt behavior does not recreate Claude command frontmatter semantics
- Codex prompt files remain deprecated compatibility content

The big loss points are:

- slash-trigger behavior
- argument interpolation semantics
- model override semantics
- tool permissions that differ by platform
- invocation gating such as `disable-model-invocation` and `user-invocable`
- Claude-only subagent routing such as `context: fork` and `agent`

So the right mental model is: **content may transfer, behavior does not.**

### 2. Instruction files

Claude Code `CLAUDE.md` and Codex `AGENTS.md` are the closest conceptual match,
but they are not the same abstraction.

Differences that matter:

- Claude loads memory hierarchically and merges multiple `CLAUDE.md` files.
- Codex concatenates discovered `AGENTS.md` files root-to-cwd, with deeper
  files appearing later in the prompt.
- Codex also has `config.toml`-backed instruction channels that do not exist in
  Claude Code.

Pi.dev adds a second instruction-like layer:

- `AGENTS.md` is concatenated from global scope plus parent directories to
  `cwd`
- `SYSTEM.md` replaces the default system prompt
- `APPEND_SYSTEM.md` appends to that prompt without replacing it

This means an instruction file can be ported, but it should usually be
**re-authored** rather than blindly copied.

At the prose level, `CLAUDE.md` content can often be reused in `AGENTS.md`. At
the behavior level, the files remain different:

- `CLAUDE.md` participates in Claude's memory-loading rules.
- `AGENTS.md` participates in Codex's root-to-cwd concatenation chain.
- Pi.dev splits persistent instructions between `AGENTS.md` and a separate
  system-prompt layer.
- Codex can also inject instructions from `config.toml`, which has no direct
  `CLAUDE.md` equivalent.

The text may transfer, but the surrounding prompt-construction semantics do
not.

### 3. Tool restriction metadata

`allowed-tools` exists in both Claude and Codex ecosystems, but the surrounding
execution model differs:

- Claude Code supports command files, subagents, and more runtime-specific
  controls.
- Codex CLI uses skill matching plus config/session behavior instead of
  Claude’s exact command/file model.
- Pi.dev does not expose the same first-pass per-skill tool field in this repo
  model.

This makes tool lists portable as intent, not as identical enforcement.

### 4. Claude-only execution controls

Several Claude frontmatter fields are useful metadata when translating content,
but they are not portable behaviors:

- `context: fork`
- `agent`
- `hooks`
- `disable-model-invocation`
- `user-invocable`
- `model`

When content moves to Codex CLI or Pi.dev, these fields should be preserved as
metadata or re-authored as native instructions, not treated as equivalent
runtime controls.

### 5. Pi.dev system-prompt layers

Pi.dev has useful, first-class instruction surfaces that other platforms do not
match exactly:

- `SYSTEM.md`
- `APPEND_SYSTEM.md`

Those files can carry portable intent, but not portable runtime semantics.
SkillSync should treat them as instruction-layer content with explicit caveats,
not as universally equivalent files.

## What Is Not Portable

### 1. Claude subagents

Claude Code has explicit subagent files under `.claude/agents/`.

Codex CLI does not have a matching first-class file model in this repo or in
the current repo docs. The closest thing is instruction content in `AGENTS.md`,
which is not a subagent definition.

If a project depends on Claude subagents, that content will need to be
flattened into:

- a skill,
- a command/prompt,
- or plain instruction text.

### 2. Claude plugin provenance and Pi.dev package provenance

Claude plugin skills are special because the skill content is installed from a
plugin cache and carries provenance metadata.

That provenance does not translate to Codex CLI or Pi.dev. At best, the skill
body can be copied into a normal repo or user scope.

Pi.dev packages, extensions, and themes have the same problem in reverse: they
may bundle useful prompts or instructions, but the package/runtime layer itself
is not a first-pass sync target in this repo.

### 3. Runtime-only behavior

These features are not safely portable:

- dynamic context injection
- slash-menu behavior
- session-specific tool permission policy
- plugin-specific installation precedence
- package/extension/theme installation precedence
- subagent delegation semantics
- Claude skill visibility and invocation gating
- Claude per-skill hook lifecycle behavior
- Claude per-skill model override behavior
- Pi.dev `SYSTEM.md` replacement and `APPEND_SYSTEM.md` append semantics

## Gaps In This Repository

These are the main gaps I see after comparing the repo docs with the current
product surfaces:

1. The portability story is spread across several docs.
   `docs/platforms/schema.yaml` is the structured artifact inventory, while
   `docs/platforms/portability-snapshot.yaml` is the structured portability
   claim set. Both need to stay in lockstep with this narrative assessment.
2. `docs/platforms/codex.md` still centers deprecated prompt files, but it
   should more explicitly distinguish:
   - what Codex CLI officially supports today,
   - what this repo parses today,
   - and what is only kept as compatibility metadata.
3. `docs/platforms/claude.md` is strong on Claude-specific behavior, but the
   portability boundaries would be clearer if it called out which fields are
   inherently non-portable to Codex CLI and Pi.dev.
4. `docs/platforms/pidev.md` should keep the first-pass scope explicit:
   skills, prompt templates, `AGENTS.md`, and `SYSTEM.md` layers are
   documented, while packages, extensions, themes, and similar runtime features
   remain out of scope.
5. `docs/architecture.md` correctly introduces `Type=prompt` and `Trigger`, but
   the doc should emphasize that these are transport concepts, not evidence of
   semantic equivalence across CLIs.
6. `internal/sync/transformer.go` currently maps prompt artifacts into Codex
   `SKILL.md` output. That is useful, but it is a lossy mapping and should be
   documented that way in the user-facing docs.

## Recommended Conclusion

If this repo is trying to model portability honestly, it should treat the
artifact layers like this:

- **Skills**: portable by default, with a well-defined common subset.
- Common subset: `name`, `description`, markdown body, and supporting
  directories. Claude-only runtime fields should be treated as metadata, not as
  parity guarantees.
- **Commands/prompts**: portable only as content, not as behavior. Pi.dev
  prompt templates fit here too.
- **Agents/subagents**: not directly portable; flatten or redesign them.
- **Packages/extensions/themes**: out of scope for first-pass sync. Re-author
  useful content into skills, prompts, or instructions instead of syncing the
  runtime packaging layer.
- **Structured snapshot**: `docs/platforms/schema.yaml` and
  `docs/platforms/portability-snapshot.yaml` should remain synchronized with the
  narrative docs so the portability story can be checked mechanically later.

That framing matches the current code better than a simple "everything syncs
everywhere" story.

## Revalidation Workflow

Run `just portability-check` after editing this assessment, the portability
snapshot, or the Claude/Codex/Pi.dev platform reference docs. The check is
intentionally narrow: it flags drift in the portable/non-portable claims, not
every possible documentation typo.

## Sources

- [Agent Skills overview](https://agentskills.io/home)
- [Agent Skills implementation guide](https://agentskills.io/client-implementation/adding-skills-support)
- [Claude Code slash commands](https://docs.anthropic.com/en/docs/claude-code/slash-commands)
- [Claude Code subagents](https://docs.anthropic.com/en/docs/claude-code/sub-agents)
- [Claude Code memory](https://docs.anthropic.com/en/docs/claude-code/memory)
- [OpenAI developers docs index](https://developers.openai.com/)
- [openai/skills](https://github.com/openai/skills)
- [Pi.dev packages index](https://pi.dev/packages)
- [Pi Monorepo](https://github.com/badlogic/pi-mono)
- [SkillSync cross-platform mapping](cross-platform-mapping.md)
- [SkillSync Claude platform reference](claude.md)
- [SkillSync Codex platform reference](codex.md)
- [SkillSync Pi.dev platform reference](pidev.md)
