# GitHub Copilot (VS Code)

> Platform reference for SkillSync parser development and cross-platform mapping.

## Overview

GitHub Copilot in VS Code provides a layered customization system centered on the `.github/` directory. Copilot supports five distinct artifact types: repository instructions, scoped instruction files, reusable prompt files, custom agent definitions, and MCP server configurations. Each type uses Markdown with optional YAML frontmatter and activates under different conditions — from always-on to pattern-matched to manually invoked.

VS Code also recognizes `AGENTS.md` (and experimentally `CLAUDE.md`) files for cross-tool compatibility, making the Copilot ecosystem one of the most interoperable among AI coding platforms.

## Directory Structure

```text
repo/
├── .github/
│   ├── copilot-instructions.md        # always-on repo instructions
│   ├── instructions/
│   │   ├── python.instructions.md     # pattern-scoped instructions
│   │   └── react.instructions.md
│   ├── prompts/
│   │   ├── test-gen.prompt.md         # reusable slash commands
│   │   └── code-review.prompt.md
│   └── agents/
│       ├── planner.agent.md           # custom agent personas
│       └── reviewer.agent.md
├── .vscode/
│   └── mcp.json                       # MCP server configuration
├── AGENTS.md                          # always-on (cross-platform)
└── CLAUDE.md                          # detected if setting enabled
```

## Artifact Types

### Repository Instructions (`.github/copilot-instructions.md`)

A single Markdown file providing always-on, project-wide context. Automatically included in every chat request within the workspace.

| Aspect          | Detail                                         |
|-----------------|-------------------------------------------------|
| Location        | `.github/copilot-instructions.md`               |
| Format          | Plain Markdown (optional `applyTo: "**"` front) |
| Activation      | Automatic — every chat request                  |
| Scope           | Entire repository                               |
| Limit           | No hard limit; ~2 pages recommended             |
| Frontmatter     | Optional, generally not used                    |

The `/init` chat command auto-generates this file with workspace-tailored defaults.

**Example:**

```markdown
# Project Coding Standards

## Naming Conventions
- Use PascalCase for component names
- Use camelCase for variables and functions

## Architecture
- Follow hexagonal architecture in internal/
- All database access through repository interfaces
```

### Instruction Files (`*.instructions.md`)

Scoped instruction files that activate conditionally based on glob patterns or manual attachment. Stored in `.github/instructions/` (or configurable locations).

#### Frontmatter Schema

| Field         | Type   | Required | Description                                                                                           |
|---------------|--------|----------|-------------------------------------------------------------------------------------------------------|
| `name`        | string | No       | Display name in the UI; defaults to filename if omitted                                               |
| `description` | string | No       | Short hover text shown in the Chat view                                                               |
| `applyTo`     | string | No       | Glob pattern for automatic application (e.g., `**/*.ts`). If omitted, must be manually attached       |
| `excludeAgent`| string | No       | Exclude from a specific agent: `"coding-agent"` or `"code-review"` (GitHub.com feature; see Gaps)     |

When `applyTo` matches a file in the active context, the instruction is automatically included alongside any repository-wide instructions. Multiple instruction files combine additively; no specific ordering is guaranteed.

**Example:**

```markdown
---
name: 'React Standards'
description: 'Component conventions for React files'
applyTo: '**/*.tsx,**/*.jsx'
---

# React Component Guidelines

- Use functional components with hooks
- Export components as named exports
- Co-locate tests with components using .test.tsx suffix
```

#### Configurable Locations

The `chat.instructionsFilesLocations` setting adds additional workspace folders beyond the default `.github/instructions`. VS Code also detects files in `.claude/rules` (workspace and user) for Claude Code compatibility.

### Prompt Files (`*.prompt.md`)

Reusable task templates invoked as slash commands in chat. Stored in `.github/prompts/` by default.

#### Frontmatter Schema

| Field           | Type          | Required | Description                                                                         |
|-----------------|---------------|----------|-------------------------------------------------------------------------------------|
| `description`   | string        | No       | Brief explanation of the prompt's purpose                                           |
| `name`          | string        | No       | Display name after `/` in chat; defaults to filename                                |
| `argument-hint` | string        | No       | Hint text shown in the chat input field to guide users                              |
| `agent`         | string        | No       | Agent context: `ask`, `agent`, `plan`, or a custom agent name                       |
| `model`         | string        | No       | Language model to use (e.g., `GPT-4o`, `Claude Sonnet 4`)                           |
| `tools`         | string array  | No       | Available tools; format: `['toolName', 'set/name', '<server>/*']`                   |

#### Variable Interpolation

Prompt file bodies support variable interpolation using `${}` syntax:

| Variable                            | Description                          |
|-------------------------------------|--------------------------------------|
| `${file}`                           | Active file path                     |
| `${fileBasename}`                   | Filename only                        |
| `${fileDirname}`                    | Directory of active file             |
| `${fileBasenameNoExtension}`        | Filename without extension           |
| `${selection}` / `${selectedText}`  | Currently selected text              |
| `${workspaceFolder}`                | Root workspace path                  |
| `${workspaceFolderBasename}`        | Workspace folder name                |
| `${input:variableName}`             | Prompt the user for a value          |
| `${input:variableName:placeholder}` | Prompt with placeholder text         |

#### Context References

- `#file:path/to/file` — attach a specific file as context
- `#tool:toolName` — reference a tool for the model to use
- Standard Markdown links `[desc](../path)` — reference other files

#### Slash Command Behavior

The filename (minus `.prompt.md`) becomes the slash command. A file at `.github/prompts/test-gen.prompt.md` is invoked as `/test-gen` in chat. Workspace prompts override user-level prompts with the same name.

**Example:**

```markdown
---
agent: 'agent'
model: 'GPT-4o'
tools: ['githubRepo', 'search/codebase']
description: 'Generate unit tests for the active file'
argument-hint: 'Describe the test scenarios'
---

Generate comprehensive unit tests for ${file}.

Requirements:
- Use the project's existing test framework
- Cover edge cases and error paths
- Follow patterns in #file:test/examples/sample.test.ts
```

#### Configurable Locations

The `chat.promptFilesLocations` setting adds additional workspace folders beyond `.github/prompts`. User-level prompts live in the `prompts/` folder of the current VS Code profile.

### Agent Files (`*.agent.md`)

Persistent personas with the richest frontmatter of any Copilot artifact type. Define specialized AI roles with tool restrictions, model preferences, and orchestrated handoffs.

SkillSync treats these as native custom agents, separate from standard skills.
The canonical repository destination is `.github/agents/<name>.agent.md`.
Same-platform round trips preserve native frontmatter. Explicit mappings to
Claude or Gemini preserve common fields and warn when Copilot-only fields do
not transfer.

#### Frontmatter Schema

| Field                      | Type          | Required    | Description                                                                                      |
|----------------------------|---------------|-------------|--------------------------------------------------------------------------------------------------|
| `name`                     | string        | No          | Display name for the agent; defaults to filename                                                 |
| `description`              | string        | **Yes**     | Purpose and capabilities of the agent; shown as placeholder text in chat                         |
| `tools`                    | string array  | No          | Tools available to the agent; defaults to all tools if omitted                                   |
| `model`                    | string/array  | No          | Language model(s); array provides fallback order                                                 |
| `agents`                   | string array  | No          | Subagents available for delegation                                                               |
| `argument-hint`            | string        | No          | Guidance text displayed in the chat input field                                                  |
| `user-invokable`           | boolean       | No          | Whether the agent appears in the agents dropdown (default: `true`)                               |
| `disable-model-invocation` | boolean       | No          | Prevents automatic delegation to this agent by other agents (default: `false`)                   |
| `target`                   | string        | No          | Environment context: `vscode` or `github-copilot`; defaults to both                             |
| `handoffs`                 | object array  | No          | Guided workflow transitions between agents (VS Code only)                                        |
| `mcp-servers`              | object        | No          | MCP server configs scoped to this agent (org/enterprise level on GitHub.com)                     |
| `metadata`                 | object        | No          | Arbitrary key-value annotations                                                                  |

**Deprecated:** `infer` (boolean) — replaced by `user-invokable` and `disable-model-invocation`.

#### Handoff Structure

Each entry in the `handoffs` array:

| Field    | Type    | Description                                            |
|----------|---------|--------------------------------------------------------|
| `label`  | string  | Display text on the handoff button                     |
| `agent`  | string  | Target agent identifier                                |
| `prompt` | string  | Text to send to the target agent                       |
| `send`   | boolean | Auto-submit the prompt (default: `false`)              |
| `model`  | string  | Optional qualified model name for the target           |

#### Tool Aliases

Agent (and prompt) files can reference tool aliases instead of individual tool names:

| Alias      | Expands to                              |
|------------|-----------------------------------------|
| `execute`  | shell, Bash, powershell                 |
| `read`     | Read, NotebookRead                      |
| `edit`     | Edit, MultiEdit, Write, NotebookEdit    |
| `search`   | Grep, Glob                              |
| `agent`    | custom-agent, Task                      |
| `web`      | WebSearch, WebFetch                     |
| `todo`     | TodoWrite                               |

MCP server tools use namespacing: `server-name/tool-name` or `server-name/*` for all tools from a server.

**Example:**

```markdown
---
description: 'Security-focused code reviewer'
name: 'Security Reviewer'
tools: ['read', 'search', 'web']
model: ['Claude Opus 4.5', 'GPT-4o']
user-invokable: true
handoffs:
  - label: 'Start Implementation'
    agent: implementer
    prompt: 'Implement the security fixes outlined above.'
    send: false
---

You are a security-focused code reviewer. Analyze code for:

- OWASP Top 10 vulnerabilities
- Authentication and authorization issues
- Input validation gaps
- Secrets or credentials in code

Provide severity ratings and remediation guidance for each finding.
```

### MCP Config (`.vscode/mcp.json`)

JSON configuration for Model Context Protocol servers, providing external tool integrations to Copilot.

#### Top-Level Structure

```json
{
  "inputs": [],
  "servers": {}
}
```

#### Transport Types

| Transport | Required Fields            | Description                              |
|-----------|----------------------------|------------------------------------------|
| `stdio`   | `command`                  | Local server via stdin/stdout streams     |
| `http`    | `type: "http"`, `url`      | Remote server over HTTP                  |
| `sse`     | `type: "sse"`, `url`       | Legacy Server-Sent Events transport      |

#### stdio Server Fields

| Field     | Type         | Required | Description                               |
|-----------|--------------|----------|-------------------------------------------|
| `command` | string       | Yes      | Executable command (e.g., `npx`, `node`)  |
| `args`    | string array | No       | Command arguments                         |
| `env`     | object       | No       | Environment variables                     |
| `envFile` | string       | No       | Path to `.env` file                       |

#### Input Prompts

Avoid hardcoded secrets by using `${input:id}` references resolved at runtime:

```json
{
  "inputs": [
    {
      "type": "promptString",
      "id": "api-key",
      "description": "API Key for the service",
      "password": true
    }
  ],
  "servers": {
    "my-server": {
      "command": "npx",
      "args": ["-y", "my-mcp-server"],
      "env": {
        "API_KEY": "${input:api-key}"
      }
    }
  }
}
```

#### Additional Features

- **Predefined variables**: `${workspaceFolder}` references the workspace root
- **Unix sockets / Windows pipes**: `unix:///path/to/server.sock` or `pipe:///named-pipe`
- **Dev Container integration**: MCP servers can be configured in `devcontainer.json` under `customizations.vscode.mcp.servers`

## Scope Levels

Copilot customization operates at three scope levels, from highest to lowest precedence:

| Level        | Mechanism                                                                  | Scope                    |
|--------------|----------------------------------------------------------------------------|--------------------------|
| Personal     | VS Code user profile (`prompts/`, instructions in profile folder)          | All workspaces for user  |
| Repository   | `.github/` directory files, `AGENTS.md`, `.vscode/mcp.json`               | Single workspace         |
| Organization | Organization-level instructions (GitHub.com; Copilot Enterprise required)  | All repos in org         |

Personal instructions take highest precedence. Repository-level files are the most commonly used. Organization instructions are configured via GitHub.com settings (not file-based).

## Discovery & Precedence

VS Code discovers customization files through settings-driven location configuration:

| Setting                            | Default Location           | Purpose                                     |
|------------------------------------|----------------------------|---------------------------------------------|
| `chat.instructionsFilesLocations`  | `.github/instructions`     | Additional folders for `*.instructions.md`   |
| `chat.promptFilesLocations`        | `.github/prompts`          | Additional folders for `*.prompt.md`         |
| (built-in)                         | `.github/agents`           | Location for `*.agent.md` files              |
| `chat.useAgentsMdFile`             | `true`                     | Enable `AGENTS.md` detection                 |
| `chat.useNestedAgentsMdFiles`      | `false` (experimental)     | Enable subfolder `AGENTS.md` files           |
| `chat.useClaudeMdFile`             | (experimental)             | Detect `CLAUDE.md` files                     |
| `chat.includeApplyingInstructions` | `true`                     | Enable pattern-based instruction matching    |

When multiple instruction files match, VS Code combines them additively. No specific ordering is guaranteed between files of the same type.

Cross-compatibility flags:
- `chat.useAgentsMdFile` — reads `AGENTS.md` files (shared with Codex, Gemini)
- `chat.useClaudeMdFile` — reads `CLAUDE.md` files (shared with Claude Code)
- VS Code detects `.claude/rules` as an instruction file source

## Tool Restrictions

Tools are controlled at the agent and prompt file level via the `tools` frontmatter field:

- **Allowlist model**: When `tools` is specified, only listed tools are available
- **Default**: If `tools` is omitted, all available tools are accessible
- **MCP scoping**: Use `server-name/*` to allow all tools from an MCP server, or `server-name/tool-name` for a specific tool
- **Tool aliases**: Use shorthand like `read`, `edit`, `search`, `execute` (see Agent Files section)
- **Graceful degradation**: If a listed tool is unavailable at runtime, it is silently ignored

## Parser Implementation Notes

- SkillSync parses the markdown Copilot repository surfaces in `internal/parser/copilot/copilot.go`
- **Multiple artifact types** with different frontmatter schemas; the parser must distinguish by file extension (`.instructions.md`, `.prompt.md`, `.agent.md`) and location (`copilot-instructions.md`)
- `.instructions.md` `applyTo` maps conceptually to Cursor's `globs` field
- `.prompt.md` maps to Claude Code's commands (slash-command invocation model)
- `.agent.md` is the richest artifact type — no direct equivalent in other platforms; closest analog is Claude Code's agent definitions but with far more frontmatter
- `copilot-instructions.md` maps to Claude's `CLAUDE.md` and Cursor's `.cursor/rules` (always-on instructions)
- `AGENTS.md` is a shared format across platforms (Copilot, Codex, Gemini)
- SkillSync preserves the original Copilot surface in transport metadata so sync can round-trip between Copilot prompts, agents, scoped instruction files, and repository instructions

**Suggested test fixtures:**

```text
testdata/copilot/
├── copilot-instructions.md          # basic always-on
├── instructions/
│   ├── python.instructions.md       # with applyTo glob
│   ├── minimal.instructions.md      # no frontmatter
│   └── excluded.instructions.md     # with excludeAgent
├── prompts/
│   ├── test-gen.prompt.md           # full frontmatter
│   ├── simple.prompt.md             # description only
│   └── variables.prompt.md          # interpolation syntax
├── agents/
│   ├── reviewer.agent.md            # with handoffs
│   ├── minimal.agent.md             # description only
│   └── scoped.agent.md              # with tools + model array
└── mcp.json                         # stdio + http transports
```

## Gaps

- **`excludeAgent`** was announced for GitHub.com (coding agent and code review); VS Code IDE support for this field is unverified as of February 2026
- **User-scoped profile folder path** varies by OS and VS Code profile; no stable cross-platform path constant
- **Organization-level instructions** are configured via GitHub.com settings, not file-based — cannot be parsed from repo content
- **`.vscode/mcp.json` remains a documented non-goal for first-pass sync**: SkillSync does not currently transform Copilot MCP config into other platforms' MCP formats
- **`mcp-servers` in agent frontmatter** is supported at org/enterprise level on GitHub.com but not in repository-level agent files or VS Code IDE agent files
- **Agent skills** (`.github/skills/` folders containing instructions, scripts, and resources) are a distinct artifact type referenced in VS Code docs but not yet well-documented for file format details
- **`chat.useNestedAgentsMdFiles`** is experimental; subfolder-level `AGENTS.md` behavior may change
- **Instruction file ordering** is explicitly unguaranteed — no precedence between multiple matching `.instructions.md` files

## Sources

- [Customize AI in Visual Studio Code](https://code.visualstudio.com/docs/copilot/copilot-customization) — retrieved 2026-02-22
- [Use custom instructions in VS Code](https://code.visualstudio.com/docs/copilot/customization/custom-instructions) — retrieved 2026-02-22
- [Use prompt files in VS Code](https://code.visualstudio.com/docs/copilot/customization/prompt-files) — retrieved 2026-02-22
- [Custom agents in VS Code](https://code.visualstudio.com/docs/copilot/customization/custom-agents) — retrieved 2026-02-22
- [Use MCP servers in VS Code](https://code.visualstudio.com/docs/copilot/customization/mcp-servers) — retrieved 2026-02-22
- [Custom agents configuration (GitHub Docs)](https://docs.github.com/en/copilot/reference/custom-agents-configuration) — retrieved 2026-02-22
- [Adding repository custom instructions (GitHub Docs)](https://docs.github.com/copilot/customizing-copilot/adding-custom-instructions-for-github-copilot) — retrieved 2026-02-22
- [Copilot code review and coding agent now support agent-specific instructions (GitHub Blog)](https://github.blog/changelog/2025-11-12-copilot-code-review-and-coding-agent-now-support-agent-specific-instructions/) — retrieved 2026-02-22
- [Copilot coding agent now supports AGENTS.md (GitHub Blog)](https://github.blog/changelog/2025-08-28-copilot-coding-agent-now-supports-agents-md-custom-instructions/) — retrieved 2026-02-22
- [Your first prompt file (GitHub Docs)](https://docs.github.com/en/copilot/tutorials/customization-library/prompt-files/your-first-prompt-file) — retrieved 2026-02-22
- [Support for Custom Attributes in .prompt.md YAML Front Matter (microsoft/vscode #284849)](https://github.com/microsoft/vscode/issues/284849) — retrieved 2026-02-22
## v2 reference boundary (verified 2026-08-18)

Official source: https://docs.github.com/en/copilot/concepts/agents/about-agent-skills.
Canonical skill roots are `.github/skills/` and `~/.copilot/skills/`; `.agents/skills/`
and `.claude/skills/` are discovery compatibility roots. Skills are distinct from
agent artifacts; custom agents and handoffs are native/documented-only.
