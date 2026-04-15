# Generate Rules and Hooks

Generate `.cursor/rules/*.mdc` files and `.cursor/hooks.json` following best practices.

## References

For detailed specifications, see:
- [Cursor Rules Format](mdc:references/cursor-rules-format.md) - Full .mdc format spec, frontmatter options, examples
- [AGENTS.md Specification](mdc:references/agents-md-spec.md) - Universal config format, symlink patterns, coexistence guidance
- [Cursor Hooks Format](mdc:references/cursor-hooks-format.md) - Hooks schema, events, response formats, examples

## Process

### 1. Analyze Existing Rules

Check for `.cursor/rules/` directory:
- **If exists**: Read all `.mdc` files, note existing patterns and customizations
- **If not exists**: Create the directory

### 2. Detect Project Stack

Scan for tech indicators:
- `package.json` → Node.js/TypeScript/React
- `go.mod` → Go
- `pyproject.toml`, `requirements.txt` → Python
- `Cargo.toml` → Rust
- `*.tf` files → Terraform
- `Taskfile.yml`, `Makefile` → Build tools
- `docker-compose.yml`, `Dockerfile` → Containers

### 3. Generate Rules by Type

#### Always Applied (`alwaysApply: true`)

Create `project.mdc` with minimal universal context:
- Project structure overview
- Key entry points and config files
- Essential commands (build, test, run)
- Tool preferences

**Example:**
```
---
alwaysApply: true
---
# Project Map

Entry: [main.go](mdc:cmd/main.go) | Config: [config.yaml](mdc:config.yaml)

## Commands
- `task build` - Build the project
- `task test` - Run tests
```

#### Auto Attached (`globs: *.ext`)

Create language-specific rules that load when editing those files:

| File | Globs | Content |
|------|-------|---------|
| `python.mdc` | `*.py` | Python conventions, uv usage, ruff formatting |
| `go.mdc` | `*.go` | Go conventions, error handling, testing patterns |
| `terraform.mdc` | `*.tf` | Module patterns, variable conventions |
| `typescript.mdc` | `*.ts,*.tsx` | Type patterns, import conventions |

**Example:**
```
---
globs: "*.py"
---
Use `uv run` for scripts. Format with ruff. Type hints required for public functions.
```

#### Agent Requested (`description: "..."`)

Create task-specific rules that load on demand:

| File | Description | Content |
|------|-------------|---------|
| `deployment.mdc` | Deployment procedures | Deploy workflows, safety checks |
| `testing.mdc` | Testing patterns | Test structure, mocking, fixtures |
| `api.mdc` | API development | Endpoint patterns, authentication |

**Example:**
```
---
description: Testing patterns and conventions
---
# Testing

Tests in `*_test.go` files. Use table-driven tests.
See [example_test.go](mdc:pkg/example/example_test.go) for patterns.
```

### 4. Merge Strategy

When existing rules found:
1. **Preserve customizations** - Don't overwrite user edits
2. **Add missing sections** - Append new guidance
3. **Update stale references** - Fix broken file paths
4. **Deduplicate** - Remove redundant content

### 5. Best Practices

- **Concise**: ~150-200 instructions max across all rules
- **References over copies**: Use `[name](mdc:path)` syntax
- **No style guides**: Use linters (ruff, gofumpt, prettier)
- **Progressive disclosure**: Link to docs for deep dives
- **Review output**: Always verify generated rules make sense

## Output

Create/update files in `.cursor/rules/`:
- `project.mdc` (always)
- Language-specific `.mdc` files (based on detected stack)
- Task-specific `.mdc` files (if patterns detected)

Report what was created/modified and suggest manual review.

---

## Generate Hooks

Generate `.cursor/hooks.json` and hook scripts in `.cursor/hooks/`.

## When to Generate Hooks

- User asks to "generate hooks" or "create hooks"
- Project has `.claude/hookify.*.md` files that need conversion
- User wants to add formatters, auditing, or command gating

## Hooks Process

### 1. Detect Existing Hooks

Check for:
- `.cursor/hooks.json` - Existing hook configuration
- `.cursor/hooks/` - Existing hook scripts
- `.claude/hookify.*.md` - Claude Code hookify rules to convert

### 2. Convert Hookify Rules (if present)

When `.claude/hookify.*.md` files exist, convert them to Cursor format.

#### Event Mapping

| Hookify `event:` | Cursor Event | Notes |
|------------------|--------------|-------|
| `bash` | `beforeShellExecution` | Direct mapping |
| `file` (with `action: warn`) | `afterFileEdit` | Post-edit, cannot block |
| `file` (with `action: block`) | `beforeReadFile` | Can block reads |
| `stop` | `stop` | Direct mapping |
| `prompt` | `beforeSubmitPrompt` | Uses `continue` not `permission` |

#### Action Mapping

| Hookify `action:` | Cursor Response |
|-------------------|-----------------|
| `block` | `"permission": "deny"` |
| `warn` | `"permission": "ask"` |
| `allow` | `"permission": "allow"` |

#### Multi-Condition Handling

Hookify rules can have multiple `conditions`. In Cursor, implement ALL conditions in the shell script logic:

**Hookify (two conditions):**
```yaml
conditions:
  - field: command
    pattern: git\s+commit.*docs:
  - field: command
    pattern: \.(py|ts|js)
```

**Cursor (both checks in script):**
```bash
#!/bin/bash
input=$(cat)
command=$(echo "$input" | jq -r '.command')

# Check BOTH conditions
if echo "$command" | grep -qE 'git\s+commit.*docs:'; then
  # First condition matched, check second
  staged=$(git diff --staged --name-only 2>/dev/null)
  if echo "$staged" | grep -qE '\.(py|ts|js)$'; then
    # Both conditions met - block
    echo '{"permission": "deny", "user_message": "docs: commits should not contain code files"}'
    exit 0
  fi
fi

echo '{"permission": "allow"}'
```

### 3. Generate Common Hooks

Based on project patterns, suggest these hooks:

| Pattern Detected | Hook Type | Purpose |
|------------------|-----------|---------|
| `ruff.toml`, `pyproject.toml` | `afterFileEdit` | Auto-format Python |
| `go.mod` | `afterFileEdit` | Run gofumpt |
| `.prettierrc` | `afterFileEdit` | Run prettier |
| `plugins/` directory | `stop` | Version bump reminder |
| `.env*` files | `beforeReadFile` | Warn on secrets access |

### 4. Create Hook Scripts

Place scripts in `.cursor/hooks/`:

```
.cursor/
├── hooks.json
└── hooks/
    ├── format.sh
    ├── audit.sh
    └── version-check.sh
```

Make scripts executable: `chmod +x .cursor/hooks/*.sh`

### 5. Best Practices

- **Always consume stdin** - Read all JSON input even if unused
- **Output valid JSON** - Hook must output valid JSON on stdout
- **Exit cleanly** - Use `exit 0` for success
- **Keep hooks fast** - Slow hooks delay agent operations
- **Test manually** - `echo '{"command":"test"}' | ./hooks/script.sh`
- **Log for debugging** - Write to `/tmp/hooks.log` during development

## Example hooks.json

```json
{
  "version": 1,
  "hooks": {
    "beforeShellExecution": [
      { "command": ".cursor/hooks/audit.sh" },
      { "command": ".cursor/hooks/block-dangerous.sh" }
    ],
    "afterFileEdit": [
      { "command": ".cursor/hooks/format.sh" }
    ],
    "stop": [
      { "command": ".cursor/hooks/version-check.sh" }
    ]
  }
}
```

## Hooks Output

Create/update:
- `.cursor/hooks.json` - Hook configuration
- `.cursor/hooks/*.sh` - Hook scripts (made executable)

Report what was created and suggest testing with the Hooks output panel.