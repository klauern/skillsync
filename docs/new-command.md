# skillsync new - Skill Scaffolding Command

The `skillsync new` command creates new skills from templates, providing a quick way to scaffold properly formatted skill files.

## Quick Start

```bash
# Create a basic command wrapper skill
skillsync new my-skill --platform claude-code --scope repo

# Create a workflow skill
skillsync new my-workflow --platform cursor --template workflow

# Create with custom description
skillsync new my-tool --platform claude-code --description "My custom tool"

# Preview without creating files
skillsync new test-skill --platform cursor --dry-run
```

## Usage

```
skillsync new <skill-name> [options]
```

### Required Arguments

- `<skill-name>` - Name of the skill to create
  - Must contain only alphanumeric characters, hyphens, underscores, colons, or slashes
  - Examples: `my-skill`, `git-helper`, `tools:formatter`

### Required Flags

- `--platform, -p` - Target platform
  - Values: `claude-code`, `cursor`, `codex`
  - Determines where the skill will be created

### Optional Flags

- `--scope, -s` - Target scope (default: `repo`)
  - Values: `builtin`, `system`, `admin`, `user`, `repo`, `plugin`
  - `repo` - Project-local skills (`.claude/skills/`, `.cursor/skills/`, etc.)
  - `user` - User-level skills (`~/.claude/skills/`, `~/.cursor/skills/`, etc.)

- `--template, -t` - Template type (default: `command-wrapper`)
  - Values: `command-wrapper`, `workflow`, `utility`
  - See [Built-in Templates](#built-in-templates) below

- `--template-file` - Path to custom template file
  - Use your own template instead of built-in ones
  - See [Custom Templates](#custom-templates) below

- `--description` - Brief description of the skill
  - Will prompt in interactive mode if not provided

- `--interactive, -i` - Interactive setup wizard
  - Prompts for additional configuration

- `--dry-run` - Preview generated content without creating files
  - Useful for testing templates

## Built-in Templates

### command-wrapper

Creates a skill that wraps an external command or tool.

**Use for:**
- Wrapping CLI tools (git, docker, kubectl, etc.)
- Adding custom behavior to existing commands
- Standardizing command interfaces

**Features:**
- Command execution and error handling
- Output parsing and formatting
- Parameter validation
- Default tools: `Bash`, `Read`

**Example:**
```bash
skillsync new git-helper --platform claude-code --template command-wrapper
```

### workflow

Creates a skill that orchestrates multiple steps in a workflow.

**Use for:**
- Multi-step processes
- CI/CD workflows
- Complex automation tasks
- Sequential operations

**Features:**
- Step-by-step execution
- Conditional branching
- Error recovery and rollback
- Progress tracking
- Default tools: `Bash`, `Read`, `Write`

**Example:**
```bash
skillsync new deploy-app --platform cursor --template workflow
```

### utility

Creates a skill that provides helper functionality.

**Use for:**
- Data transformation
- Helper functions
- Reusable components
- Shared utilities

**Features:**
- Data processing and formatting
- Integration helpers
- Reusable functions
- Default tools: `Read`, `Write`

**Example:**
```bash
skillsync new data-formatter --platform codex --template utility
```

## Examples

### Basic Command Wrapper

```bash
skillsync new git-status --platform claude-code --scope repo \
  --description "Enhanced git status with custom formatting"
```

This creates `.claude/skills/git-status/SKILL.md` with:
- Frontmatter with skill metadata
- Command wrapper structure
- Documentation sections
- Usage examples

### Workflow with Interactive Setup

```bash
skillsync new ci-pipeline --platform cursor --template workflow --interactive
```

Prompts for:
- Skill description
- Additional configuration (future expansion)

### Preview Before Creating

```bash
skillsync new test-skill --platform claude-code --dry-run
```

Shows the generated content without creating files, useful for:
- Testing templates
- Verifying configuration
- Learning the format

### User-Level Utility

```bash
skillsync new string-utils --platform claude-code --scope user --template utility \
  --description "String manipulation utilities"
```

Creates skill in `~/.claude/skills/string-utils/SKILL.md` instead of project-local.

## Custom Templates

You can create your own templates using Go's `text/template` syntax.

### Template Structure

```yaml
---
name: {{.Name}}
description: {{.Description}}
scope: {{.Scope}}
license: MIT
tools:{{range .Tools}}
  - {{.}}{{end}}
---

# {{.Name}}

Your custom content here.

Available template variables:
- {{.Name}} - Skill name
- {{.Description}} - Skill description
- {{.Platform}} - Target platform
- {{.Scope}} - Scope level
- {{.Year}} - Current year
```

### Available Template Data

```go
type TemplateData struct {
    Name        string   // Skill name
    Description string   // Brief description
    Platform    string   // claude-code, cursor, codex
    Scope       string   // builtin, system, admin, user, repo, plugin
    Author      string   // Author name (future)
    Year        int      // Current year
    Tools       []string // Required tools
    Scripts     []string // Script files (future)
    References  []string // Reference docs (future)
}
```

### Using Custom Templates

```bash
# Create custom template file
cat > my-template.md <<'EOF'
---
name: {{.Name}}
description: {{.Description}}
scope: {{.Scope}}
---

# {{.Name}}

Custom template for my organization.

Created: {{.Year}}
Platform: {{.Platform}}
EOF

# Use custom template
skillsync new my-skill --platform claude-code --template-file my-template.md
```

## Generated Structure

When you run `skillsync new`, it creates:

```
<platform>-skills/
└── <skill-name>/
    └── SKILL.md       # Main skill file with frontmatter and content
```

Future expansion may include:
```
<skill-name>/
├── SKILL.md           # Main skill file
├── scripts/           # Optional executable scripts
├── references/        # Optional documentation
└── assets/            # Optional data files
```

## Integration with skillsync

After creating a skill with `skillsync new`, you can:

1. **Edit the skill** - Customize the generated content
2. **Test the skill** - Use `/<skill-name>` in your AI assistant
3. **Sync across platforms** - Use `skillsync sync` to copy to other platforms
4. **Version control** - Commit repo-scoped skills to git
5. **Export/import** - Use `skillsync export` and `skillsync import`

## Tips

1. **Start with dry-run** - Use `--dry-run` to preview before creating
2. **Use repo scope for projects** - Keep project-specific skills in `.claude/skills/`
3. **Use user scope for personal tools** - Keep personal utilities in `~/.claude/skills/`
4. **Customize after creation** - Templates are starting points, edit freely
5. **Follow naming conventions** - Use lowercase with hyphens for consistency
6. **Add descriptive comments** - Future you will thank present you

## Troubleshooting

### "invalid skill name" error

Skill names must contain only:
- Lowercase/uppercase letters
- Numbers
- Hyphens (`-`)
- Underscores (`_`)
- Colons (`:`)
- Slashes (`/`)

Bad: `my skill!`, `my@skill`
Good: `my-skill`, `tools:my-skill`

### "platform is required" error

The `--platform` flag is mandatory. Specify one of:
- `claude-code`
- `cursor`
- `codex`

### "invalid scope" error

Valid scopes: `builtin`, `system`, `admin`, `user`, `repo`, `plugin`

Default is `repo` if not specified.

### "template not found" error

Use one of the built-in templates:
- `command-wrapper` (default)
- `workflow`
- `utility`

Or provide a custom template with `--template-file`.

## See Also

- `skillsync sync` - Sync skills across platforms
- `skillsync export` - Export skills to archives
- `skillsync import` - Import skills from archives
- `skillsync scope` - Manage skill scopes
- Agent Skills Standard - [Skill format specification](https://github.com/anthropics/agent-skills-standard)
