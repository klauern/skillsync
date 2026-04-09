# Cross-Platform Mapping

> Unified reference for concept mapping and conversion rules across all 6 documented platforms.
>
> SkillSync currently implements first-class parsing and sync for Claude Code, Cursor, and Codex.
> Copilot, Gemini CLI, and Pi.dev are documented here as reference mappings and portability targets,
> not as first-class runtime support in the current CLI.

## Support Status

| Platform | Status in SkillSync | Notes |
|---|---|---|
| Claude Code | Implemented | First-class parser and sync target |
| Cursor | Implemented | First-class parser and sync target |
| Codex | Implemented | First-class parser and sync target; deprecated prompts remain compatibility-only |
| Copilot | Reference-only | Documented for concept mapping and portability analysis |
| Gemini CLI | Reference-only | Documented for concept mapping and portability analysis |
| Pi.dev | Reference-only | Documented for concept mapping and portability analysis |

## Concept Mapping

What each platform calls equivalent concepts:

| Concept | Claude Code | Codex | Copilot (VS Code) | Cursor | Gemini CLI | Pi.dev |
|---|---|---|---|---|---|---|
| Always-on instructions | `CLAUDE.md` | `AGENTS.md` | `.github/copilot-instructions.md` | Rules (`alwaysApply: true`) | `GEMINI.md` | `AGENTS.md` (+ optional `SYSTEM.md` override layer) |
| Reusable skills | `.claude/skills/*/SKILL.md` | `.codex/skills/*/SKILL.md` | `.github/agents/*.agent.md` | `.cursor/skills/*/SKILL.md` | `.gemini/skills/*/SKILL.md` | `.pi/skills/*/SKILL.md` |
| Slash commands | `.claude/commands/*.md` | `~/.codex/prompts/*.md` (deprecated) | `.github/prompts/*.prompt.md` | `.cursor/commands/` + modes | `.gemini/commands/*.toml` | `.pi/prompts/*.md` |
| Pattern-based rules | -- | -- | `*.instructions.md` (`applyTo`) | Rules (`globs`) | -- | -- |
| Agent definitions | `.claude/agents/*.md` | -- | `.github/agents/*.agent.md` | Modes (`modes.json`) | `.gemini/agents/*.md` | -- |
| MCP config | `.claude/settings.json` | `config.toml` (`mcp_servers`) | `.vscode/mcp.json` | Mode/settings | `settings.json` (`mcpServers`) | -- |

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
| Pi.dev | `AGENTS.md` | Plain markdown | None | Global + ancestor walk + cwd, concatenated; optional `SYSTEM.md`/`APPEND_SYSTEM.md` layer is separate |

### Reusable Skills

Modular instruction packages that the AI can invoke based on task relevance.

| Platform | Location | Frontmatter Fields | Invocation |
|---|---|---|---|
| Claude | `.claude/skills/*/SKILL.md` | `name`, `description`, `allowed-tools`, `argument-hint`, `disable-model-invocation`, `user-invocable`, `model`, `context`, `agent`, `hooks` | User (`/name`) or automatic |
| Codex | `.codex/skills/*/SKILL.md` | `name`, `description`, `version`, `allowed-tools` | Automatic (task relevance) |
| Copilot | `.github/agents/*.agent.md` | `name`, `description`, `tools`, `model`, `agents`, `argument-hint`, `user-invokable`, `disable-model-invocation`, `target`, `handoffs`, `mcp-servers` | User (`@name`) or automatic |
| Cursor | `.cursor/skills/*/SKILL.md` | `name`, `description`, `tools` | Automatic (Agent Skills Standard) |
| Gemini | `.gemini/skills/*/SKILL.md` | `name`, `description` | Progressive disclosure (consent-gated) |
| Pi.dev | `.pi/skills/*/SKILL.md` | `name`, `description` | Automatic / task-relevance driven |

### Slash Commands / Prompts

User-invocable prompt templates triggered by a name or filename.

| Platform | Location | Format | Arguments | Trigger |
|---|---|---|---|---|
| Claude | `.claude/commands/*.md` | Markdown + frontmatter | `$ARGUMENTS`, `$1`, `$ARGUMENTS[N]` | Filename stem -> `/name` |
| Codex | `~/.codex/prompts/*.md` | Markdown + frontmatter | `$ARGUMENTS`, `$1`--`$9`, `$NAME` | `/prompts:<name>` (deprecated; use Skills) |
| Copilot | `.github/prompts/*.prompt.md` | Markdown + frontmatter | `${file}`, `${input:var}`, `#file:`, `#tool:` | Filename stem -> `/name` |
| Cursor | `.cursor/commands/*.md` | Markdown | (mode-dependent) | Mode-linked |
| Gemini | `.gemini/commands/*.toml` | TOML | `{{args}}`, `!{shell}`, `@{path}` | File path -> `/ns:name` |
| Pi.dev | `.pi/prompts/*.md` | Markdown | `{{name}}`-style placeholders | Filename stem -> `/name` |

### Pattern-Based Rules

Instructions that activate conditionally based on file patterns.

| Platform | Location | Pattern Field | Description |
|---|---|---|---|
| Claude | -- | -- | No native pattern-based rules |
| Codex | -- | -- | No native pattern-based rules |
| Copilot | `*.instructions.md` | `applyTo` (glob) | Auto-applied when matching files in context |
| Cursor | `.cursor/rules/*.md` | `globs` (glob array) | Auto-applied when working with matching files |
| Gemini | -- | -- | No native pattern-based rules |
| Pi.dev | -- | -- | No first-pass pattern-rule surface documented here |

## Frontmatter Field Mapping

Field-by-field comparison showing which platform supports which fields.

| Field | Claude | Codex | Copilot | Cursor | Gemini | Pi.dev |
|---|---|---|---|---|---|---|
| `name` | Skills, Commands | Skills | Instructions, Prompts, Agents | Rules (optional) | Skills | Skills |
| `description` | Skills, Commands | Skills | Instructions, Prompts, Agents | Rules | Skills | Skills |
| `tools` / `allowed-tools` | Skills (`allowed-tools`) | Skills (`allowed-tools`) | Prompts, Agents (`tools`) | Skills (`tools`) | -- | -- |
| `argument-hint` | Skills, Commands | -- | Prompts, Agents | -- | -- | -- |
| `model` | Skills, Commands | -- (config-level) | Prompts, Agents | -- | -- (subagents only) | -- |
| `disable-model-invocation` | Skills | -- | Agents | -- | -- | -- |
| `user-invocable` / `user-invokable` | Skills (`user-invocable`) | -- | Agents (`user-invokable`) | -- | -- | -- |
| `globs` / `applyTo` | -- | -- | Instructions (`applyTo`) | Rules (`globs`) | -- | -- |
| `alwaysApply` | -- | -- | -- | Rules | -- | -- |
| `context` | Skills (`fork`) | -- | -- | -- | -- | -- |
| `agent` | Skills | -- | Prompts, Agents (`agents`) | -- | -- | -- |
| `hooks` | Skills | -- | -- | -- | -- (settings-level) | -- |
| `version` | -- | Skills | -- | -- | -- | -- |
| `target` | -- | -- | Agents | -- | -- | -- |
| `handoffs` | -- | -- | Agents | -- | -- | -- |
| `mcp-servers` | -- | -- | Agents | -- | -- | -- |
| `excludeAgent` | -- | -- | Instructions | -- | -- | -- |

## Path Convention Reference

Default paths side-by-side for all 6 documented platforms.

### Project-Level Paths

| Purpose | Claude | Codex | Copilot | Cursor | Gemini | Pi.dev |
|---|---|---|---|---|---|---|
| Skills | `.claude/skills/` | `.codex/skills/` | `.github/agents/` | `.cursor/skills/` | `.gemini/skills/` (or `.agents/skills/`) | `.pi/skills/` |
| Instructions | `.claude/CLAUDE.md` | `AGENTS.md` | `.github/copilot-instructions.md` | `.cursor/rules/` | `.gemini/GEMINI.md` | `AGENTS.md` + `.pi/SYSTEM.md` |
| Commands | `.claude/commands/` | -- | `.github/prompts/` | `.cursor/commands/` | `.gemini/commands/` | `.pi/prompts/` |
| Config | `.claude/settings.json` | `.codex/config.toml` | `.vscode/mcp.json` | -- | `.gemini/settings.json` | `.pi/settings.json` |

### User-Level Paths

| Purpose | Claude | Codex | Copilot | Cursor | Gemini | Pi.dev |
|---|---|---|---|---|---|---|
| Skills | `~/.claude/skills/` | `~/.codex/skills/` | VS Code profile | `~/.cursor/skills/` | `~/.gemini/skills/` (or `~/.agents/skills/`) | `~/.pi/agent/skills/` |
| Instructions | `~/.claude/CLAUDE.md` | `~/.codex/AGENTS.md` | VS Code profile | `~/.cursor/rules/` | `~/.gemini/GEMINI.md` | `~/.pi/agent/AGENTS.md` + `~/.pi/agent/SYSTEM.md` |
| Commands | `~/.claude/commands/` | `~/.codex/prompts/` (deprecated) | VS Code profile (`prompts/`) | `~/.cursor/commands/` | `~/.gemini/commands/` | `~/.pi/agent/prompts/` |
| Config | `~/.claude/config.json` | `~/.codex/config.toml` | VS Code settings | `~/.cursor/modes.json` | `~/.gemini/settings.json` | `~/.pi/agent/settings.json` |

### System-Level Paths

| Platform | Path | Description |
|---|---|---|
| Claude | Managed settings (enterprise) | Admin-deployed, not file-based |
| Codex | `/etc/codex/skills/` | System-wide admin skills |
| Copilot | GitHub.com org settings | Org-level instructions (not file-based) |
| Cursor | -- | No system-level path |
| Gemini | Extension directory | Extension-bundled content |
| Pi.dev | -- | Packages, extensions, and themes exist, but they are out of scope for first-pass sync |

## Scope Level Comparison

How global/user/project/extension maps across platforms.

| Scope | Claude | Codex | Copilot | Cursor | Gemini | Pi.dev |
|---|---|---|---|---|---|---|
| Organization / Enterprise | Enterprise (highest) | Admin (`/etc/`) | Organization (lowest) | -- | -- | -- |
| User / Personal | Personal | User (`~/.codex/`) | Personal (VS Code profile, highest) | Global (`~/.cursor/`) | User (`~/.gemini/`) | User (`~/.pi/agent/`) |
| Project / Repository | Project | Project (`.codex/`) | Repository | Project (`.cursor/`) | Workspace (`.gemini/`) | Project (`.pi/` + repo `AGENTS.md`) |
| Plugin / Extension | Namespaced | -- | -- | -- | Extension (lowest) | Packages / themes / extensions (not first-pass sync targets) |

**Precedence direction varies by platform:**
- Claude: Enterprise > Personal > Project
- Codex: Project > User > Admin
- Copilot: Personal > Repository > Organization
- Cursor: Project > Global
- Gemini: Workspace > User > Extension
- Pi.dev: project skills/prompts/settings override user defaults; `AGENTS.md` concatenates global + ancestors + cwd

## Lossy Conversion Warnings

When converting artifacts between platforms, some information is lost or requires adaptation.

### Field-Level Losses

| Field | Supported | Unsupported | Conversion Behavior |
|---|---|---|---|
| `argument-hint` | Claude, Copilot | Codex, Cursor, Gemini, Pi.dev | Preserved in `Metadata` map; no runtime effect on unsupported platforms |
| `$ARGUMENTS` interpolation | Claude, Codex (deprecated custom prompts) | Copilot, Cursor, Gemini, Pi.dev | Kept as literal text on unsupported platforms; Gemini uses `{{args}}`, Copilot uses `${input:var}`, Pi.dev prompt templates use `{{name}}`-style placeholders |
| `allowed-tools` / `tools` | Claude, Codex, Copilot, Cursor (skills only) | Gemini, Pi.dev | Preserved in `Metadata`; Gemini uses hooks/extension `excludeTools` instead |
| `model` | Claude, Copilot | Codex (config-level), Cursor, Gemini (subagents), Pi.dev | Preserved in `Metadata`; no per-skill effect on unsupported platforms |
| `disable-model-invocation` | Claude, Copilot (agents) | Codex, Cursor, Gemini, Pi.dev | Preserved in `Metadata`; no equivalent toggle |
| `user-invocable` | Claude, Copilot (agents) | Codex, Cursor, Gemini, Pi.dev | Preserved in `Metadata`; no equivalent toggle |
| `globs` / `applyTo` | Copilot, Cursor | Claude, Codex, Gemini, Pi.dev | No conversion target; pattern rules do not map to instruction/context files |
| `alwaysApply` | Cursor | Claude, Codex, Copilot, Gemini, Pi.dev | Maps to always-on instruction semantics but not 1:1 |
| `hooks` | Claude | Codex, Copilot, Cursor, Gemini (settings), Pi.dev | Claude-specific object; no per-skill hooks elsewhere |
| `handoffs` | Copilot (agents) | All others | Copilot-unique; no cross-platform equivalent |
| `target` | Copilot (agents) | All others | Copilot-unique (`vscode` / `github-copilot`) |
| `mcp-servers` (per-agent) | Copilot (agents) | All others | Copilot-unique at agent level |
| `context: fork` | Claude | All others | Claude-specific subagent forking |
| `SYSTEM.md` / `APPEND_SYSTEM.md` | Pi.dev | Claude, Codex, Copilot, Cursor, Gemini | Separate Pi.dev system-prompt layer; preserve as instructions metadata or re-author manually |

### Argument Syntax Mapping

| Platform | Syntax | Description |
|---|---|---|
| Claude | `$ARGUMENTS`, `$ARGUMENTS[N]`, `$N` | Positional argument substitution |
| Copilot | `${file}`, `${input:var}`, `#file:path` | VS Code variable interpolation |
| Gemini | `{{args}}`, `!{shell}`, `@{path}` | Template injection (args, shell output, file content) |
| Codex | `$ARGUMENTS`, `$1`--`$9`, `$NAME` | Positional + named placeholders (custom prompts only; deprecated) |
| Pi.dev | `{{name}}` | Template-style placeholder expansion in markdown prompts |
| Cursor | -- | No argument substitution syntax |

These are not interchangeable. Each uses different delimiters and supports different features (shell execution, file embedding, user prompts).

### Format Differences

| Aspect | Impact |
|---|---|
| Gemini commands use TOML | Requires format conversion to/from Markdown for all other platforms |
| Copilot uses multiple file extensions | `.instructions.md`, `.prompt.md`, `.agent.md` each have different schemas |
| Cursor legacy `.mdc` format | Functionally identical to `.md` but extension is Cursor-specific |
| Claude plugin namespacing | `plugin-name:skill-name` format has no equivalent on other platforms |
| Pi.dev package ecosystem | Packages, extensions, and themes are runtime surfaces, not first-pass sync targets |

## SkillSync Unified Model Mapping

How `model.Skill` fields map to each platform's native representation.

| `model.Skill` Field | Claude | Codex | Copilot | Cursor | Gemini | Pi.dev |
|---|---|---|---|---|---|---|
| `Name` | `name` frontmatter or dir name | `name` frontmatter or dir name | `name` frontmatter or filename | `name` frontmatter or filename | `name` frontmatter or dir name | `name` frontmatter or dir name |
| `Description` | `description` frontmatter | `description` frontmatter | `description` frontmatter | `description` frontmatter | `description` frontmatter | `description` frontmatter |
| `Platform` | `ClaudeCode` | `Codex` | `Copilot` (new) | `Cursor` | `Gemini` (new) | `PiDev` (planned) |
| `Path` | Filesystem path to file | Filesystem path to file | Filesystem path to file | Filesystem path to file | Filesystem path to file | Filesystem path to file |
| `Tools` | `tools` / `allowed-tools` | `allowed-tools` | `tools` | `tools` (skills only) | -- | -- |
| `Metadata` | All other frontmatter | All other frontmatter | All other frontmatter | All frontmatter except `name` | All other frontmatter | Prompt placeholder syntax, system-prompt layer markers, and other non-portable fields |
| `Content` | Markdown body | Markdown body | Markdown body | Markdown body | Markdown body (or TOML `prompt`) | Markdown body |
| `ModifiedAt` | File mtime | File mtime | File mtime | File mtime | File mtime | File mtime |
| `Type` | `skill` or `prompt` | `skill` | `skill`, `prompt`, or `rule` (new) | `skill` or `rule` | `skill` or `prompt` | `skill` or `prompt` |
| `Trigger` | `/name` (from filename or frontmatter) | -- | `/name` (from filename) | (mode-linked) | `/name` or `/ns:name` | `/name` (from filename stem) |
| `Scope` | `user`, `repo`, `plugin`, `system` | `user`, `repo`, `admin`, `system` | `user`, `repo`, `org` | `user`, `repo` | `user`, `repo`, `extension` | `user`, `repo` |
| `DisableModelInvocation` | From frontmatter | -- | From agent frontmatter | -- | -- | -- |
| `PluginInfo` | Plugin symlink detection | -- | -- | -- | -- | -- |
