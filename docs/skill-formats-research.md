# Skill Formats Research

This document catalogs the skill/rule formats used by different AI coding platforms to inform the implementation of skillsync parsers.

## Table of Contents

- [Claude Code](#claude-code)
- [Cursor](#cursor)
- [Codex](#codex)
- [References](#references)

---

## Claude Code

### Storage Locations

Claude Code stores skills in multiple locations:

1. **Project-local skills**: `.claude/skills/` directory within a project
2. **Project instructions**: `.claude/` directory with configuration files
3. **Global instructions**: `~/.claude/` for user-wide settings

### File Formats

#### 1. Markdown Skills (`.md`)

Skills are typically stored as markdown files with frontmatter:

```markdown
---
name: skill-name
description: A brief description
tools: ["tool1", "tool2"]
---

# Skill Name

Detailed instructions for the AI assistant...
```

#### 2. JSON Configuration

Settings and permissions are stored in JSON:

```json
{
  "permissions": {
    "allow": [
      "Bash(/usr/bin/find:*)",
      "WebFetch(domain:github.com)",
      "Skill(skill-name)"
    ]
  }
}
```

Common configuration files:

- `.claude/settings.local.json` - Local project settings
- `.claude/settings.json` - Project settings (committed)
- `~/.claude/config.json` - Global settings

### Key Conventions

- **Frontmatter**: Uses YAML frontmatter delimited by `---`
- **Metadata fields**:
  - `name`: Skill identifier
  - `description`: Human-readable description
  - `tools`: Array of required tool permissions
- **Content**: Markdown-formatted instructions
- **File extensions**: `.md` for skills, `.json` for configuration
- **Permissions**: Specified as patterns like `Tool(domain:pattern)` or `Tool(command:*)`

### References

- Official docs: https://code.claude.com/docs/en/skills
- Skills convention: https://code.claude.com/docs/en/skills/conventions

---

## Cursor

### Storage Locations

Cursor stores skills in:

1. **Project skills**: `.cursor/skills/` directory
2. **Global skills**: `~/.cursor/skills/` for user-wide skills
3. **Legacy rules**: `.cursor/rules/` (project) or `~/.cursor/rules/` (global)

### File Format

#### Agent Skills Standard (`SKILL.md`)

Newer Cursor skills follow the Agent Skills Standard with `SKILL.md` files in
subdirectories.

#### Legacy Rules (`.md` or `.mdc`)

Legacy Cursor rules use a special markdown format with YAML frontmatter:

```markdown
---
globs: ["*.go", "*.rs"]
alwaysApply: false
---

# Rule Name

Instructions for Cursor AI...
```

### Key Conventions

- **Frontmatter fields**:
  - `globs`: Array of glob patterns for file matching
  - `alwaysApply`: Boolean (if true, applies to all files)
- **Content**: Markdown-formatted instructions
- **File extensions**: `.md` or `.mdc` (Cursor markdown, legacy)
- **Matching**: Rules apply based on glob patterns matching file paths
- **Precedence**: More specific globs take precedence

### Example Patterns

```yaml
# Language-specific
globs: ["*.go"]

# Directory-specific
globs: ["internal/**/*.go"]

# Multiple patterns
globs: ["*.ts", "*.tsx", "spec.ts"]

# All files
alwaysApply: true
```

### References

- Official docs: https://cursor.com/docs/context/skills
- Rules guide: https://cursor.com/docs/context/rules

---

## Codex

### Storage Locations

Codex (OpenAI CLI) loads skills from multiple locations in order of precedence (high to low):

1. **Project skills**: `.codex/skills/` in working directory and repo root
2. **User skills**: `~/.codex/skills/` for user-wide skills
3. **Admin skills**: `/etc/codex/skills/` for system-wide skills

See: https://developers.openai.com/codex/skills/

### File Format

#### JSON Schema with Function Definitions

```json
{
  "skills": [
    {
      "name": "skill_name",
      "description": "A description of what the skill does",
      "parameters": {
        "type": "object",
        "properties": {
          "param1": {
            "type": "string",
            "description": "Parameter description"
          }
        },
        "required": ["param1"]
      },
      "implementation": {
        "type": "executable",
        "command": ["python", "-c", "..."]
      }
    }
  ]
}
```

### Key Conventions

- **Format**: JSON Schema compatible with OpenAI function calling
- **Structure**: Array of skill/function definitions
- **Parameters**: JSON Schema for type safety
- **Implementation**: Can be command-line, HTTP, or plugin-based
- **File extensions**: `.json` for schemas, `.yaml`/`.yml` for alternative format

### Alternative: YAML Format

```yaml
skills:
  - name: skill_name
    description: Description here
    parameters:
      type: object
      properties:
        param1:
          type: string
          description: Parameter description
      required:
        - param1
```

### References

- OpenAI docs: https://platform.openai.com/docs/guides/function-calling
- Codex skills: https://developers.openai.com/codex/skills (if available)
- Function calling: https://platform.openai.com/docs/guides/function-calling

---

## References

### General Skill Specifications

- **AgentSkills**: https://agentskills.io/home - Emerging standard for portable AI agent skills
- **OpenAPI Specification**: For HTTP-based skill implementations
- **JSON Schema**: For parameter validation across platforms

### Cross-Platform Considerations

1. **Metadata normalization**: Each platform uses different metadata fields
   - Claude Code: `name`, `description`, `tools`
   - Cursor: `globs`, `alwaysApply`
   - Codex: `parameters`, `implementation`

2. **Content format**: All use Markdown for instructions, but with different frontmatter

3. **File discovery**: Different default paths and conventions
   - Claude: `.claude/skills/*.md`
   - Cursor: `.cursor/rules/*.mdc`
   - Codex: `.codex/*.json`

4. **Platform detection**: Can infer from presence of config directories

### Implementation Notes for SkillSync

1. **Parser abstraction**: Each platform needs a dedicated parser
   - `internal/parser/claude.go`
   - `internal/parser/cursor.go`
   - `internal/parser/codex.go`

2. **Common utilities needed**:
   - Frontmatter parsing (YAML)
   - Glob pattern matching
   - JSON schema validation
   - File system discovery

3. **Unified model**: `internal/model/skill.go` already defines common fields
   - Name, Description, Platform, Path, Content
   - Metadata map for platform-specific fields

4. **Error handling**: Need graceful handling of:
   - Missing directories
   - Malformed frontmatter
   - Invalid schemas
   - Permission errors

---

## Appendix: Quick Reference

### Metadata Field Mapping

| Concept      | Claude Code   | Cursor        | Codex            |
| ------------ | ------------- | ------------- | ---------------- |
| Skill Name   | `name`        | (filename)    | `name`           |
| Description  | `description` | (content)     | `description`    |
| Scope        | `tools`       | `globs`       | `parameters`     |
| Always Apply | -             | `alwaysApply` | -                |
| Instructions | (content)     | (content)     | `implementation` |

### File Extensions by Platform

| Platform    | Extensions               | MIME Type                           |
| ----------- | ------------------------ | ----------------------------------- |
| Claude Code | `.md`, `.json`           | `text/markdown`, `application/json` |
| Cursor      | `.md`, `.mdc`            | `text/markdown`                     |
| Codex       | `.json`, `.yaml`, `.yml` | `application/json`, `text/yaml`     |

### Default Paths

| Platform    | Local Path        | Global Path                              |
| ----------- | ----------------- | ---------------------------------------- |
| Claude Code | `.claude/skills/` | `~/.claude/skills/`                      |
| Cursor      | `.cursor/rules/`  | `~/.cursor/rules/`                       |
| Codex       | `.codex/skills/`  | `~/.codex/skills/`, `/etc/codex/skills/` |
