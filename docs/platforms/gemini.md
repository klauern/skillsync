# Gemini CLI

> Platform reference for SkillSync parser development and cross-platform mapping.

## Overview

Gemini CLI (`google-gemini/gemini-cli`) is Google's open-source terminal-based
coding agent powered by Gemini models. It stores configuration, context,
skills, commands, hooks, and extensions under the `.gemini/` directory (project)
and `~/.gemini/` directory (user). An alternate `.agents/` directory is also
recognized for skills (and takes precedence over `.gemini/` within the same
tier), aligning with the cross-platform Agent Skills Standard.

Gemini CLI uses a progressive disclosure model for skills, concatenated
context files across three tiers, TOML-based custom commands with injection
syntax, a rich hooks lifecycle system, and an extension packaging format that
bundles all artifact types together.

## Directory Structure

```text
~/.gemini/                             # User-level config root
  GEMINI.md                            # Global context file
  settings.json                        # User settings
  skills/                              # User skills
    <skill-name>/
      SKILL.md                         # Required -- skill definition
      scripts/                         # Optional executable helpers
      references/                      # Optional supporting docs
      assets/                          # Optional templates / data
  commands/                            # User custom commands
    <name>.toml                        # Command definition
    <ns>/<name>.toml                   # Namespaced command (/<ns>:<name>)
  agents/                              # User subagent definitions
    <agent-name>.md                    # Agent definition (frontmatter + body)
  extensions/                          # Installed extensions
    <ext-name>/
      gemini-extension.json            # Extension manifest
  .env                                 # Environment variables (e.g. API keys)

~/.agents/                             # Alternate user skills path (higher priority)
  skills/
    <skill-name>/SKILL.md

<project>/
  .gemini/                             # Project-level config root
    GEMINI.md                          # Workspace context file
    settings.json                      # Workspace settings (overrides user)
    system.md                          # System prompt override (env-var gated)
    skills/                            # Project skills
      <skill-name>/SKILL.md
    commands/                          # Project custom commands
      <name>.toml
    agents/                            # Project subagent definitions
      <agent-name>.md
    hooks/                             # Extension hooks (hooks.json)
    .env                               # Project environment variables

  .agents/                             # Alternate project skills path (higher priority)
    skills/
      <skill-name>/SKILL.md
```

## Artifact Types

### Context Files (`GEMINI.md`)

Plain markdown files (no frontmatter) that provide persistent project-specific
instructions. The CLI loads and concatenates context from three tiers:

**Three-tier loading:**

| Tier | Path | Purpose |
|---|---|---|
| Global | `~/.gemini/GEMINI.md` | Default instructions for all projects |
| Workspace | `<project>/.gemini/GEMINI.md` | Project-specific guidance |
| JIT (Just-In-Time) | Subdirectories accessed during session | Contextual instructions discovered when tools access files in subdirectories |

All discovered files are **concatenated** (not overridden) in hierarchical
order (global, workspace, JIT). The CLI footer displays the count of loaded
context files.

**Import syntax (`@path`):**

Context files can import external content using `@` references with relative or
absolute paths:

```markdown
# Main GEMINI.md

Primary project instructions.

@./components/instructions.md

@../shared/style-guide.md
```

**Configurable filename:**

The default filename `GEMINI.md` can be overridden in `settings.json`:

```json
{
  "context": {
    "fileName": ["AGENTS.md", "CONTEXT.md", "GEMINI.md"]
  }
}
```

The CLI checks files in the specified order, enabling multiple naming
conventions per project.

**Management commands:**

- `/memory show` -- display complete concatenated context
- `/memory refresh` -- re-scan and reload all context files
- `/memory add <text>` -- append text to global `~/.gemini/GEMINI.md`

### Skills (`SKILL.md`)

Each skill is a directory containing a `SKILL.md` file with minimal YAML
frontmatter and a markdown body. Skills follow the Agent Skills Standard.

#### Frontmatter Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Unique skill identifier; should match directory name. |
| `description` | string | yes | Describes purpose and when Gemini should use the skill. |

Only two frontmatter fields. The markdown body provides detailed instructions
and procedural workflows.

#### Example

```markdown
---
name: code-reviewer
description: Use this skill to review code. It supports both local changes and remote Pull Requests.
---

# Code Reviewer

Review the provided code changes for:
1. Correctness
2. Style consistency
3. Security issues
```

#### Progressive Disclosure Activation Model

Skills use a context-efficient loading strategy:

1. **Discovery**: Only metadata (name, description) loads into the system prompt.
2. **Selection**: The model autonomously decides when to employ a skill based on task relevance.
3. **Consent**: User confirmation prompt shows skill name, purpose, and granted directory access.
4. **Injection**: Upon approval, full `SKILL.md` content and directory listing enter conversation history; the skill directory gains file access permissions.

This prevents cluttering the model's context while maintaining large skill
libraries.

#### Supporting Directories

| Directory | Purpose |
|---|---|
| `scripts/` | Executable code for deterministic, repeatable operations. |
| `references/` | Supporting documentation the skill can reference. |
| `assets/` | Templates, schemas, data files. |

#### CLI Management

```shell
gemini skills list              # Display all discovered skills and status
gemini skills install <source>  # Install from Git repo, local path, or .skill file
gemini skills link <path>       # Create symlinks from local directory
gemini skills uninstall <name>  # Remove skill by name
gemini skills enable <name>     # Reactivate disabled skill
gemini skills disable <name>    # Prevent specific skill usage
gemini skills enable --all      # Enable all skills
gemini skills disable --all     # Disable all skills
```

The `install` command supports Git repositories, local directories, packaged
`.skill` files, and monorepo subdirectory extraction via `--path`.

**`--scope` flag**: Controls installation/management target:

- `--scope workspace` -- affects `.gemini/skills/` (or `.agents/skills/`) in the project
- `--scope user` -- affects `~/.gemini/skills/` (or `~/.agents/skills/`); default for enable/disable

**Interactive session commands:**

- `/skills list` -- view all available skills (default)
- `/skills link <path>` -- symlink external skill directories
- `/skills disable <name>` -- deactivate skill (defaults to user scope)
- `/skills enable <name>` -- reactivate skill
- `/skills reload` -- refresh discovery

### Custom Commands (`.toml`)

Custom commands use **TOML format** (not markdown). Command names derive from
file paths relative to the `commands/` directory using colon notation for
namespacing.

**Discovery paths (precedence order):**

1. Project commands: `<project>/.gemini/commands/` (highest)
2. User commands: `~/.gemini/commands/`
3. Extension commands: lowest (prefixed with extension name on conflict)

**Naming/namespacing examples:**

- `~/.gemini/commands/test.toml` -> `/test`
- `<project>/.gemini/commands/git/commit.toml` -> `/git:commit`

#### Schema

| Field | Type | Required | Description |
|---|---|---|---|
| `prompt` | string | yes | Text sent to the model; supports injection syntax. |
| `description` | string | no | One-line summary for `/help`; auto-generated from filename if omitted. |

#### Injection Syntax

**`{{args}}` -- argument injection:**

When the prompt contains `{{args}}`, the CLI replaces this placeholder with
user input. Inside `!{...}` blocks, arguments are automatically shell-escaped.
When `{{args}}` is absent, arguments are appended after two newlines, or the
prompt is sent as-is without arguments.

**`!{shell}` -- shell command injection:**

Execute shell commands and embed stdout. The CLI confirms the command before
execution. Failed commands include stderr and exit code.

```toml
prompt = """
Review these changes:
!{git diff --staged}
"""
```

**`@{path}` -- file content injection:**

Embed file or directory contents. Supports multimodal files (images, PDFs,
audio, video). Directory traversal includes all files respecting `.gitignore`
and `.geminiignore`. File injection occurs before shell commands and argument
substitution.

```toml
prompt = """
Follow these guidelines:
@{docs/best-practices.md}

Now review the code.
"""
```

#### Example

```toml
# ~/.gemini/commands/refactor/pure.toml -> /refactor:pure
description = "Refactor current context into a pure function."
prompt = """
Please analyze the code provided.
Refactor it into a pure function.

Include:
1. Refactored code block
2. Explanation of changes"""
```

After modifying `.toml` files, run `/commands reload` to apply changes without
restarting.

### Extensions (`gemini-extension.json`)

Extensions bundle multiple artifact types (MCP servers, commands, skills,
hooks, context, themes, subagents) into a distributable package.

#### Manifest Schema

| Field | Type | Description |
|---|---|---|
| `name` | string | Unique identifier (lowercase letters, numbers, dashes); must match directory name. |
| `version` | string | Extension version. |
| `description` | string | Short summary for the extension gallery. |
| `contextFileName` | string or string[] | Context file name(s) to load (defaults to `GEMINI.md`). |
| `mcpServers` | object | MCP server definitions (same format as `settings.json`). |
| `settings` | array | User-configurable settings (`name`, `description`, `envVar`, `sensitive`). |
| `excludeTools` | string[] | Tools to block; supports argument-level granularity (e.g., `"run_shell_command(rm -rf)"`). |
| `themes` | array | Custom UI color themes. |

Extension directories may also contain:

| Path | Content |
|---|---|
| `commands/` | Custom command `.toml` files |
| `skills/` | Skill directories with `SKILL.md` |
| `hooks/hooks.json` | Lifecycle hooks |
| `agents/` | Subagent definition `.md` files (preview feature) |

#### Variable Substitution

Supported in `gemini-extension.json` and `hooks/hooks.json`:

| Variable | Expansion |
|---|---|
| `${extensionPath}` | Absolute path to extension directory |
| `${workspacePath}` | Absolute path to current workspace |
| `${/}` | Platform-specific path separator |

#### Extension Management

```shell
gemini extensions install <source>        # Install from Git URL or local path
gemini extensions install <source> --ref <ref>  # Specific branch/tag/commit
gemini extensions uninstall <name>        # Remove extension
gemini extensions list                    # List installed extensions
gemini extensions update <name>           # Update specific extension
gemini extensions update --all            # Update all
gemini extensions enable <name>           # Enable
gemini extensions disable <name>          # Disable
gemini extensions link <path>             # Link local extension for dev
gemini extensions new <path>              # Create from template
gemini extensions validate <path>         # Validate structure
gemini extensions config <name> [setting] --scope <user|workspace>
```

### Settings (`settings.json`)

Settings are stored in `settings.json` at two scope levels. Workspace settings
override user settings.

- User: `~/.gemini/settings.json`
- Workspace: `<project>/.gemini/settings.json`

#### Key Configuration Categories

| Category | Notable Keys |
|---|---|
| `general` | `vimMode`, `defaultApprovalMode`, `enableAutoUpdate`, `enableNotifications`, `sessionRetention` |
| `output` | `format` (`"text"` or `"json"`) |
| `ui` | Theme switching, footer, line numbers, accessibility, screen reader mode |
| `model` | `maxSessionTurns`, `compressionThreshold`, `disableLoopDetection` |
| `context` | `discoveryMaxDirs`, `fileFiltering` (gitignore, geminiignore, fuzzy search), `fileName` |
| `tools` | `shell.enableInteractiveShell`, `useRipgrep`, `truncateToolOutputThreshold` |
| `security` | `disableYoloMode`, `enablePermanentToolApproval`, `blockGitExtensions`, `allowedExtensions`, `folderTrust` |
| `skills` | `enabled` (boolean toggle) |
| `hooksConfig` | `enabled`, `notifications` |
| `experimental` | `toolOutputMasking`, `plan`, `modelSteering`, `enableAgents` |

The `/settings` interactive command opens a dialog for viewing and modifying
settings without direct file editing.

### System Prompt Override (`.gemini/system.md`)

A complete replacement of the built-in system prompt, activated via the
`GEMINI_SYSTEM_MD` environment variable.

**Activation:**

| Value | Behavior |
|---|---|
| `true` / `1` | Use `./.gemini/system.md` from project root |
| `/path/to/file.md` | Use custom path (absolute or relative, tilde expansion supported) |
| `false` / `0` / unset | Disabled (default) |

Can be set temporarily (shell export) or persistently via `.gemini/.env`.

**Placeholder variables** for dynamic injection:

| Variable | Expansion |
|---|---|
| `${AgentSkills}` | Complete agent skills section with header |
| `${SubAgents}` | Available sub-agents section with header |
| `${AvailableTools}` | Bulleted list of enabled tool names |
| `${toolName_ToolName}` | Tool-specific name (e.g., `${write_file_ToolName}`) |

**Export built-in prompt**: `GEMINI_WRITE_SYSTEM_MD=1 gemini` writes the
default prompt to `./.gemini/system.md` for use as a starting template.

When active, the CLI displays a `|>-<_>-<|` indicator.

### Hooks (`settings.json` hooks section)

Hooks intercept lifecycle events for validation, context injection, auditing,
and security gating. Defined in the `hooks` key of `settings.json` (or in
`hooks/hooks.json` within extensions).

#### Event Types

| Event | Phase | Description |
|---|---|---|
| `SessionStart` | Lifecycle | Session begins, resumes, or `/clear` |
| `BeforeAgent` | Agent | After user prompt, before agent planning |
| `BeforeToolSelection` | Model | Before model decides which tools to call |
| `BeforeTool` | Tool | Before individual tool invocation |
| `AfterTool` | Tool | After tool execution |
| `BeforeModel` | Model | Before LLM request |
| `AfterModel` | Model | After LLM response (per chunk during streaming) |
| `AfterAgent` | Agent | After model generates final response |
| `SessionEnd` | Lifecycle | CLI exit or session clear |
| `Notification` | System | System alerts (e.g., tool permissions) |
| `PreCompress` | System | Before history summarization |

#### Configuration Structure

```json
{
  "hooks": {
    "BeforeTool": [
      {
        "matcher": "write_file",
        "sequential": false,
        "hooks": [
          {
            "name": "security-scanner",
            "type": "command",
            "command": "/path/to/scanner.sh",
            "timeout": 5000,
            "description": "Scan file writes for secrets"
          }
        ]
      }
    ]
  }
}
```

**Matcher**: Regex pattern (for tools) or exact string (for lifecycle events).
Use `"*"` to match all.

**Execution**: Scripts receive JSON via stdin, must output JSON to stdout.
Logs go to stderr. Exit code 0 returns structured JSON; exit code 2 is an
emergency block.

**Key capabilities:**

- `BeforeTool` / `AfterTool`: validate, block, or rewrite tool arguments/outputs
- `BeforeAgent`: inject dynamic context (git history, RAG, memory)
- `AfterAgent`: validate response quality with retry logic
- `BeforeToolSelection`: filter available toolset (whitelists combined across hooks)
- `BeforeModel`: override or mock LLM requests/responses

### Subagents (`.gemini/agents/*.md`)

Specialized agents operating within a session, defined as markdown files with
YAML frontmatter.

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Unique slug identifier |
| `description` | string | yes | Routing description for the main agent |
| `kind` | string | no | `local` (default) or `remote` |
| `tools` | string[] | no | Restricted tool set |
| `model` | string | no | Specific model override |
| `temperature` | number | no | 0.0--2.0 range |
| `max_turns` | number | no | Maximum conversation turns (default: 15) |
| `timeout_mins` | number | no | Execution time limit in minutes (default: 5) |

Requires `experimental.enableAgents: true` in settings. Subagents operate in
auto-approval mode (no per-tool user confirmation).

## Scope Levels

| Priority | Scope | Config Path | Skills Path |
|---|---|---|---|
| 1 | Workspace | `<project>/.gemini/settings.json` | `.agents/skills/` > `.gemini/skills/` |
| 2 | User | `~/.gemini/settings.json` | `~/.agents/skills/` > `~/.gemini/skills/` |
| 3 | Extension | Bundled in extension directory | `<ext>/skills/` |

**Settings**: Workspace overrides user; extensions have lowest priority.

**Context files**: Concatenated across all tiers (not overridden). Global +
workspace + JIT all contribute to the final context.

**Custom commands**: Project commands override user commands with identical
names. Extension commands have lowest precedence and are prefixed with the
extension name on conflict.

## Discovery & Precedence

**Skills**: Within each tier, `.agents/skills/` takes precedence over
`.gemini/skills/`. Across tiers: workspace > user > extension.

**Context files (`GEMINI.md`)**: Discovered across all three tiers and
concatenated in order (global -> workspace -> JIT). All content contributes;
nothing is shadowed.

**Custom commands**: Project-level commands override identically-named user
commands. Extension commands are lowest priority and gain a namespace prefix
on conflict.

**Settings**: Workspace `settings.json` overrides user `settings.json`.
Extension settings are configured separately via `gemini extensions config`.

## Tool Restrictions

- **Extension manifest**: `excludeTools` array blocks tools, with argument-level
  granularity (e.g., `"run_shell_command(rm -rf)"`).
- **Hooks**: `BeforeToolSelection` dynamically filters available tools;
  `BeforeTool` can deny specific invocations.
- **Subagents**: The `tools` field restricts which tools a subagent can use.
- **No per-skill tool field**: Unlike Codex's `allowed-tools` or Claude's
  per-command restrictions, Gemini skills have no frontmatter field for tool
  restrictions.

## Parser Implementation Notes

- **No existing parser** in the SkillSync codebase. Needs new
  `internal/parser/gemini/gemini.go`.
- **SKILL.md** has only 2 frontmatter fields (`name`, `description`) -- the
  simplest schema among supported platforms. Can reuse the shared skills parser
  (`internal/parser/skills/skills.go`).
- **Custom commands are TOML**, not markdown -- unique among SkillSync platforms.
  Requires a TOML parser dependency or lightweight custom parser.
- **`.agents/` alternate path** adds complexity: discovery must check both
  `.agents/skills/` and `.gemini/skills/` at each tier, with `.agents/` winning.
- **Context files use `@path` imports**: The parser should resolve these to
  produce complete context content.
- **Three injection syntaxes** in custom commands (`{{args}}`, `!{shell}`,
  `@{path}`) need representation in the unified model. `{{args}}` maps roughly
  to Claude's `$ARGUMENTS`; `!{shell}` and `@{path}` have no direct Claude
  equivalent.
- **Extensions bundle multiple artifact types**: The parser should extract
  skills, commands, and context from extension directories.

**Suggested test fixtures** (`testdata/skills/gemini/`):

| Fixture | Description |
|---|---|
| `gemini-basic/SKILL.md` | Minimal skill with name + description frontmatter. |
| `gemini-with-assets/SKILL.md` | Skill with `scripts/`, `references/`, `assets/` dirs. |
| `commands/deploy.toml` | Basic custom command with prompt + description. |
| `commands/git/commit.toml` | Namespaced command demonstrating injection syntax. |
| `gemini-extension.json` | Minimal extension manifest. |
| `context/GEMINI.md` | Context file with `@path` imports. |

## Gaps

- **`.agents/` rationale underdocumented**: The `.agents/` alternate directory
  is recognized but its design rationale and full precedence semantics are not
  formally specified beyond "takes precedence within the same tier."
- **Subagent `.md` format partially documented**: Frontmatter schema is
  specified, but body format conventions (beyond "system prompt") and
  inter-agent communication patterns are not fully documented. Marked as
  preview feature.
- **`excludeTools` argument grammar not formally specified**: The
  `"tool_name(args)"` syntax for argument-level granularity lacks a formal
  grammar definition. Observed from examples only.
- **Skills have no tool field**: Unlike Codex (`allowed-tools`) or Claude
  (per-command model hints), Gemini skills cannot declare tool restrictions.
  Tool filtering is only available through hooks or extension manifests.
- **`--scope` flag not documented for skills commands**: The skills CLI
  reference mentions workspace and user scope behavior but does not
  explicitly list `--scope` as a flag in all skill subcommands.
- **No documented `.skill` package format**: The `gemini skills install`
  command accepts `.skill` files, but the packaging format is not publicly
  documented.
- **`context.fileName` interaction with JIT**: Whether the custom filename
  setting applies to JIT discovery (subdirectory scanning) is not explicitly
  documented.
- **Hooks in extensions vs settings.json**: Hooks can be defined in both
  `settings.json` and extension `hooks/hooks.json`, but precedence and merge
  behavior between these sources is not formally documented.

## Sources

- Gemini CLI GitHub repository: https://github.com/google-gemini/gemini-cli (accessed 2026-02-22)
- Skills documentation: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/skills.md (accessed 2026-02-22)
- Creating skills guide: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/creating-skills.md (accessed 2026-02-22)
- Custom commands docs: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/custom-commands.md (accessed 2026-02-22)
- Context files (GEMINI.md) docs: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/gemini-md.md (accessed 2026-02-22)
- Settings reference: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/settings.md (accessed 2026-02-22)
- Extension writing guide: https://github.com/google-gemini/gemini-cli/blob/main/docs/extensions/writing-extensions.md (accessed 2026-02-22)
- Extension reference: https://github.com/google-gemini/gemini-cli/blob/main/docs/extensions/reference.md (accessed 2026-02-22)
- Hooks writing guide: https://github.com/google-gemini/gemini-cli/blob/main/docs/hooks/writing-hooks.md (accessed 2026-02-22)
- Hooks reference: https://github.com/google-gemini/gemini-cli/blob/main/docs/hooks/reference.md (accessed 2026-02-22)
- System prompt override docs: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/system-prompt.md (accessed 2026-02-22)
- CLI reference: https://github.com/google-gemini/gemini-cli/blob/main/docs/cli/cli-reference.md (accessed 2026-02-22)
- Subagents docs: https://github.com/google-gemini/gemini-cli/blob/main/docs/core/subagents.md (accessed 2026-02-22)
