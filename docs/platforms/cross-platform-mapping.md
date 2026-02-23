# Cross-Platform Mapping

> Unified reference for concept mapping and conversion rules across all 5 supported platforms.

## Concept Mapping

What each platform calls equivalent concepts:

| Concept | Claude Code | Codex | Copilot (VS Code) | Cursor | Gemini CLI |
|---|---|---|---|---|---|
| Always-on instructions | `CLAUDE.md` | `AGENTS.md` | `.github/copilot-instructions.md` | Rules (`alwaysApply: true`) | `GEMINI.md` |
| Reusable skills | `.claude/skills/*/SKILL.md` | `.codex/skills/*/SKILL.md` | `.github/agents/*.agent.md` | `.cursor/skills/*/SKILL.md` | `.gemini/skills/*/SKILL.md` |
| Slash commands | `.claude/commands/*.md` | `~/.codex/prompts/*.md` (deprecated) | `.github/prompts/*.prompt.md` | `.cursor/commands/` + modes | `.gemini/commands/*.toml` |
| Pattern-based rules | -- | -- | `*.instructions.md` (`applyTo`) | Rules (`globs`) | -- |
| Agent definitions | `.claude/agents/*.md` | -- | `.github/agents/*.agent.md` | Modes (`modes.json`) | `.gemini/agents/*.md` |
| MCP config | `.claude/settings.json` | `config.toml` (`mcp_servers`) | `.vscode/mcp.json` | Mode/settings | `settings.json` (`mcpServers`) |

## Artifact Type Equivalences

Detailed mapping of which artifact types serve the same purpose across platforms.

### Always-On Instructions

Files loaded automatically into every session.

| Platform | File | Format | Frontmatter | Loading |
|---|---|---|---|---|
| Claude | `CLAUDE.md` | Plain markdown | None | Hierarchical: global > project, merged |
| Codex | `AGENTS.md` | Plain markdown | None | Root-to-cwd, concatenated; nearest file appears last (de-facto precedence) |
| Copilot | `.github/copilot-instructions.md` | Plain markdown | Optional | Always included in chat |
| Cursor | Rules with `alwaysApply: true` | Markdown + frontmatter | Yes (`globs`, `alwaysApply`) | Auto-injected |
| Gemini | `GEMINI.md` | Plain markdown | None | Three-tier concatenation (global + workspace + JIT) |

### Reusable Skills

Modular instruction packages that the AI can invoke based on task relevance.

| Platform | Location | Frontmatter Fields | Invocation |
|---|---|---|---|
| Claude | `.claude/skills/*/SKILL.md` | `name`, `description`, `allowed-tools`, `argument-hint`, `disable-model-invocation`, `user-invocable`, `model`, `context`, `agent`, `hooks` | User (`/name`) or automatic |
| Codex | `.codex/skills/*/SKILL.md` | `name`, `description`, `version`, `allowed-tools` | Automatic (task relevance) |
| Copilot | `.github/agents/*.agent.md` | `name`, `description`, `tools`, `model`, `agents`, `argument-hint`, `user-invokable`, `disable-model-invocation`, `target`, `handoffs`, `mcp-servers` | User (`@name`) or automatic |
| Cursor | `.cursor/skills/*/SKILL.md` | `name`, `description`, `tools` | Automatic (Agent Skills Standard) |
| Gemini | `.gemini/skills/*/SKILL.md` | `name`, `description` | Progressive disclosure (consent-gated) |

### Slash Commands / Prompts

User-invocable prompt templates triggered by a name or filename.

| Platform | Location | Format | Arguments | Trigger |
|---|---|---|---|---|
| Claude | `.claude/commands/*.md` | Markdown + frontmatter | `$ARGUMENTS`, `$1`, `$ARGUMENTS[N]` | Filename stem -> `/name` |
| Codex | `~/.codex/prompts/*.md` | Markdown + frontmatter | `$ARGUMENTS`, `$1`--`$9`, `$NAME` | `/prompts:<name>` (deprecated; use Skills) |
| Copilot | `.github/prompts/*.prompt.md` | Markdown + frontmatter | `${file}`, `${input:var}`, `#file:`, `#tool:` | Filename stem -> `/name` |
| Cursor | `.cursor/commands/*.md` | Markdown | (mode-dependent) | Mode-linked |
| Gemini | `.gemini/commands/*.toml` | TOML | `{{args}}`, `!{shell}`, `@{path}` | File path -> `/ns:name` |

### Pattern-Based Rules

Instructions that activate conditionally based on file patterns.

| Platform | Location | Pattern Field | Description |
|---|---|---|---|
| Claude | -- | -- | No native pattern-based rules |
| Codex | -- | -- | No native pattern-based rules |
| Copilot | `*.instructions.md` | `applyTo` (glob) | Auto-applied when matching files in context |
| Cursor | `.cursor/rules/*.md` | `globs` (glob array) | Auto-applied when working with matching files |
| Gemini | -- | -- | No native pattern-based rules |

## Frontmatter Field Mapping

Field-by-field comparison showing which platform supports which fields.

| Field | Claude | Codex | Copilot | Cursor | Gemini |
|---|---|---|---|---|---|
| `name` | Skills, Commands | Skills | Instructions, Prompts, Agents | Rules (optional) | Skills |
| `description` | Skills, Commands | Skills | Instructions, Prompts, Agents | Rules | Skills |
| `tools` / `allowed-tools` | Skills (`allowed-tools`) | Skills (`allowed-tools`) | Prompts, Agents (`tools`) | Skills (`tools`) | -- |
| `argument-hint` | Skills, Commands | -- | Prompts, Agents | -- | -- |
| `model` | Skills, Commands | -- (config-level) | Prompts, Agents | -- | -- (subagents only) |
| `disable-model-invocation` | Skills | -- | Agents | -- | -- |
| `user-invocable` / `user-invokable` | Skills (`user-invocable`) | -- | Agents (`user-invokable`) | -- | -- |
| `globs` / `applyTo` | -- | -- | Instructions (`applyTo`) | Rules (`globs`) | -- |
| `alwaysApply` | -- | -- | -- | Rules | -- |
| `context` | Skills (`fork`) | -- | -- | -- | -- |
| `agent` | Skills | -- | Prompts, Agents (`agents`) | -- | -- |
| `hooks` | Skills | -- | -- | -- | -- (settings-level) |
| `version` | -- | Skills | -- | -- | -- |
| `target` | -- | -- | Agents | -- | -- |
| `handoffs` | -- | -- | Agents | -- | -- |
| `mcp-servers` | -- | -- | Agents | -- | -- |
| `excludeAgent` | -- | -- | Instructions | -- | -- |

## Path Convention Reference

Default paths side-by-side for all 5 platforms.

### Project-Level Paths

| Purpose | Claude | Codex | Copilot | Cursor | Gemini |
|---|---|---|---|---|---|
| Skills | `.claude/skills/` | `.codex/skills/` | `.github/agents/` | `.cursor/skills/` | `.gemini/skills/` (or `.agents/skills/`) |
| Instructions | `.claude/CLAUDE.md` | `AGENTS.md` | `.github/copilot-instructions.md` | `.cursor/rules/` | `.gemini/GEMINI.md` |
| Commands | `.claude/commands/` | -- | `.github/prompts/` | `.cursor/commands/` | `.gemini/commands/` |
| Config | `.claude/settings.json` | `.codex/config.toml` | `.vscode/mcp.json` | -- | `.gemini/settings.json` |

### User-Level Paths

| Purpose | Claude | Codex | Copilot | Cursor | Gemini |
|---|---|---|---|---|---|
| Skills | `~/.claude/skills/` | `~/.codex/skills/` | VS Code profile | `~/.cursor/skills/` | `~/.gemini/skills/` (or `~/.agents/skills/`) |
| Instructions | `~/.claude/CLAUDE.md` | `~/.codex/AGENTS.md` | VS Code profile | `~/.cursor/rules/` | `~/.gemini/GEMINI.md` |
| Commands | `~/.claude/commands/` | -- | VS Code profile (`prompts/`) | `~/.cursor/commands/` | `~/.gemini/commands/` |
| Config | `~/.claude/config.json` | `~/.codex/config.toml` | VS Code settings | `~/.cursor/modes.json` | `~/.gemini/settings.json` |

### System-Level Paths

| Platform | Path | Description |
|---|---|---|
| Claude | Managed settings (enterprise) | Admin-deployed, not file-based |
| Codex | `/etc/codex/skills/` | System-wide admin skills |
| Copilot | GitHub.com org settings | Org-level instructions (not file-based) |
| Cursor | -- | No system-level path |
| Gemini | Extension directory | Extension-bundled content |

## Scope Level Comparison

How global/user/project/extension maps across platforms.

| Scope | Claude | Codex | Copilot | Cursor | Gemini |
|---|---|---|---|---|---|
| Organization / Enterprise | Enterprise (highest) | Admin (`/etc/`) | Organization (lowest) | -- | -- |
| User / Personal | Personal | User (`~/.codex/`) | Personal (VS Code profile, highest) | Global (`~/.cursor/`) | User (`~/.gemini/`) |
| Project / Repository | Project | Project (`.codex/`) | Repository | Project (`.cursor/`) | Workspace (`.gemini/`) |
| Plugin / Extension | Namespaced | -- | -- | -- | Extension (lowest) |

**Precedence direction varies by platform:**
- Claude: Enterprise > Personal > Project
- Codex: Project > User > Admin
- Copilot: Personal > Repository > Organization
- Cursor: Project > Global
- Gemini: Workspace > User > Extension

## Lossy Conversion Warnings

When converting artifacts between platforms, some information is lost or requires adaptation.

### Field-Level Losses

| Field | Supported | Unsupported | Conversion Behavior |
|---|---|---|---|
| `argument-hint` | Claude, Copilot | Codex, Cursor, Gemini | Preserved in `Metadata` map; no runtime effect on unsupported platforms |
| `$ARGUMENTS` interpolation | Claude | Codex, Copilot, Cursor, Gemini | Kept as literal text; Gemini uses `{{args}}`, Copilot uses `${input:var}` |
| `allowed-tools` / `tools` | Claude, Codex, Copilot, Cursor (skills only) | Gemini | Preserved in `Metadata`; Gemini uses hooks/extension `excludeTools` instead |
| `model` | Claude, Copilot | Codex (config-level), Cursor, Gemini (subagents) | Preserved in `Metadata`; no per-skill effect on unsupported platforms |
| `disable-model-invocation` | Claude, Copilot (agents) | Codex, Cursor, Gemini | Preserved in `Metadata`; no equivalent toggle |
| `user-invocable` | Claude, Copilot (agents) | Codex, Cursor, Gemini | Preserved in `Metadata`; no equivalent toggle |
| `globs` / `applyTo` | Copilot, Cursor | Claude, Codex, Gemini | No conversion target; pattern rules do not map to instruction/context files |
| `alwaysApply` | Cursor | Claude, Codex, Copilot, Gemini | Maps to always-on instruction semantics but not 1:1 |
| `hooks` | Claude | Codex, Copilot, Cursor, Gemini (settings) | Claude-specific object; no per-skill hooks elsewhere |
| `handoffs` | Copilot (agents) | All others | Copilot-unique; no cross-platform equivalent |
| `target` | Copilot (agents) | All others | Copilot-unique (`vscode` / `github-copilot`) |
| `mcp-servers` (per-agent) | Copilot (agents) | All others | Copilot-unique at agent level |
| `context: fork` | Claude | All others | Claude-specific subagent forking |

### Argument Syntax Mapping

| Platform | Syntax | Description |
|---|---|---|
| Claude | `$ARGUMENTS`, `$ARGUMENTS[N]`, `$N` | Positional argument substitution |
| Copilot | `${file}`, `${input:var}`, `#file:path` | VS Code variable interpolation |
| Gemini | `{{args}}`, `!{shell}`, `@{path}` | Template injection (args, shell output, file content) |
| Codex | `$ARGUMENTS`, `$1`--`$9`, `$NAME` | Positional + named placeholders (custom prompts only; deprecated) |
| Cursor | -- | No argument substitution syntax |

These are **not interchangeable**. Each uses different delimiters and supports different features (shell execution, file embedding, user prompts).

### Format Differences

| Aspect | Impact |
|---|---|
| Gemini commands use TOML | Requires format conversion to/from Markdown for all other platforms |
| Copilot uses multiple file extensions | `.instructions.md`, `.prompt.md`, `.agent.md` each have different schemas |
| Cursor legacy `.mdc` format | Functionally identical to `.md` but extension is Cursor-specific |
| Claude plugin namespacing | `plugin-name:skill-name` format has no equivalent on other platforms |

## SkillSync Unified Model Mapping

How `model.Skill` fields map to each platform's native representation.

| `model.Skill` Field | Claude | Codex | Copilot | Cursor | Gemini |
|---|---|---|---|---|---|
| `Name` | `name` frontmatter or dir name | `name` frontmatter or dir name | `name` frontmatter or filename | `name` frontmatter or filename | `name` frontmatter or dir name |
| `Description` | `description` frontmatter | `description` frontmatter | `description` frontmatter | `description` frontmatter | `description` frontmatter |
| `Platform` | `ClaudeCode` | `Codex` | `Copilot` (new) | `Cursor` | `Gemini` (new) |
| `Path` | Filesystem path to file | Filesystem path to file | Filesystem path to file | Filesystem path to file | Filesystem path to file |
| `Tools` | `tools` / `allowed-tools` | `allowed-tools` | `tools` | `tools` (skills only) | -- |
| `Metadata` | All other frontmatter | All other frontmatter | All other frontmatter | All frontmatter except `name` | All other frontmatter |
| `Content` | Markdown body | Markdown body | Markdown body | Markdown body | Markdown body (or TOML `prompt`) |
| `ModifiedAt` | File mtime | File mtime | File mtime | File mtime | File mtime |
| `Type` | `skill` or `prompt` | `skill` | `skill`, `prompt`, or `rule` (new) | `skill` or `rule` | `skill` or `prompt` |
| `Trigger` | `/name` (from filename or frontmatter) | -- | `/name` (from filename) | (mode-linked) | `/name` or `/ns:name` |
| `Scope` | `user`, `repo`, `plugin`, `system` | `user`, `repo`, `admin`, `system` | `user`, `repo`, `org` | `user`, `repo` | `user`, `repo`, `extension` |
| `DisableModelInvocation` | From frontmatter | -- | From agent frontmatter | -- | -- |
| `PluginInfo` | Plugin symlink detection | -- | -- | -- | -- |
