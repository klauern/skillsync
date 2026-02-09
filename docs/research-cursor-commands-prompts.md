# Cursor Command/Prompt Research (skillsync-c1g.3)

Date: 2026-02-09

## Scope

Assess Cursor support for prompt-like artifacts and interoperability with Claude
commands and Codex skills/prompts.

## Authoritative Behavior (Cursor docs)

Sources:
- https://cursor.com/docs/llms-full.txt

Relevant sections in `llms-full.txt`:
- Custom modes and command file location (`.cursor/commands`)
- Rules system (`.cursor/rules/*.mdc`)
- Agent skills (`.cursor/skills`)

### Supported artifacts

1. Custom modes / command-like artifacts
- Global mode config file in `~/.cursor/modes.json`
- Optional global instruction files in `~/.cursor/commands`
- Project-specific instruction files in `.cursor/commands`

2. Rules
- Stored in `.cursor/rules`
- Use frontmatter fields like `description`, `globs`, `alwaysApply`
- Applied automatically by scope/matching rules

3. Skills
- Stored in `.cursor/skills`
- Described as reusable procedures with examples and references

### Invocation semantics

- Custom modes can define slash-command style behavior in mode config (for
  example `slashCommand` metadata in docs text) and reference prompt files.
- Rules are auto-applied context instructions, not slash commands.
- Skills are reusable contextual behaviors selected by the assistant.

### Discovery / precedence

From docs text:
- Project-level command files (`.cursor/commands`) override global command files
  (`~/.cursor/commands`) when names collide.
- Rules may vary by scope and matching mode; file-pattern rules use glob matching.

## Local Observations (2026-02-09)

- `~/.cursor/skills` exists locally.
- No local `~/.cursor/commands` directory currently exists on this machine.
- Current `skillsync` parser has dedicated Cursor skill parsing but no first-class
  parser for `.cursor/commands` command artifacts yet.

## Interop Matrix

### Claude <-> Cursor

| Dimension | Claude commands | Cursor equivalent | Fidelity |
|---|---|---|---|
| File location | `.claude/commands/*.md` | `.cursor/commands/*.md` | High |
| Trigger naming | filename => `/name` | mode/command config and prompt file linkage | Medium |
| Tool allowlist | `allowed-tools` | no single universal field across commands/rules/skills | Medium/Low |
| Argument hints | `argument-hint` | no direct guaranteed equivalent | Low |
| Freeform prompt body | markdown | markdown instruction files | High |

### Codex <-> Cursor

| Dimension | Codex skills | Cursor equivalent | Fidelity |
|---|---|---|---|
| Skill artifact | `.codex/skills/*/SKILL.md` | `.cursor/skills/*/SKILL.md` | High |
| Slash command parity | built-in slash commands; skill invocation | custom mode command config | Medium |
| AGENTS-style global instructions | `AGENTS.md` hierarchy | rules + mode prompts | Medium/Low |
| Tool allowlist | `allowed-tools` (skill frontmatter) | rules/modes settings vary | Medium |

## Gaps and Lossy Conversion Risks

1. Cursor command behavior is tied to modes + command files; mapping from pure
   Claude command markdown may require generating mode metadata.
2. Claude `argument-hint` has no strong one-to-one equivalent in Cursor docs.
3. Codex and Cursor have different command invocation abstractions (skill
   relevance vs mode slash-command config), so trigger semantics are not fully
   portable.
4. Rule-specific fields (`globs`, `alwaysApply`) are not implied by Claude/Codex
   command artifacts and need defaults or user policy.

## Conclusion for c1g.4 design

- Support Cursor `.cursor/commands` as a first-class discovery target.
- Keep a unified model with explicit `type` (`skill`, `prompt`, `rule`), plus
  platform-specific extension metadata.
- Require explicit warning path for lossy conversions when generating Cursor mode
  command metadata from Claude/Codex sources.
