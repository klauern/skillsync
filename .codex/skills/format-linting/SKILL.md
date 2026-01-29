---
description: Format and lint code files (.md, .py, .tf, .go, etc.) using standard linters.
name: format-linting
---

# Format and Lint Skill

Expert code formatting and linting specialist. Automatically detects project type and runs appropriate formatters and linters.

## When to Use

Invoke this skill when:
- User asks to format code or "run formatters"
- User asks to lint code or "check code quality"
- User asks to "clean up code" or "fix style issues"
- After making significant code changes
- Before committing code

## Supported Languages and Tools

### Go
- **Formatter**: `gofumpt` (preferred) or `gofmt`
- **Linters**: `golangci-lint` (preferred) or `go vet`
- **Commands**:
  ```bash
  gofumpt -w .          # Format all Go files
  golangci-lint run     # Run comprehensive linting
  go vet ./...          # Fallback linter
  ```

### Python
- **Formatter**: `ruff format` (via `uvx`)
- **Linter**: `ruff check --fix`
- **Commands**:
  ```bash
  uvx ruff format .              # Format all Python files
  uvx ruff check --fix .         # Fix auto-fixable issues
  uvx ruff check .               # Show remaining issues
  ```

### JavaScript/TypeScript
- **Formatter**: `bunx prettier`
- **Linter**: `eslint` (if configured)
- **Commands**:
  ```bash
  prettier --write "**/*.{js,ts,jsx,tsx}"   # Format JS/TS files
  eslint --fix .                             # Fix auto-fixable issues
  ```

### YAML
- **Formatter**: `bunx prettier`
- **Commands**:
  ```bash
  bunx prettier --write "**/*.{yml,yaml}"
  ```

### Other Languages
- **Rust**: `cargo fmt`, `cargo clippy`
- **Ruby**: `bundle exec rubocop -a`
- **Shell**: `shfmt -w .`
- **Markdown**: `bunx prettier --write "**/*.md"`

## Workflow

1. **Detect Project Type**
   - Check for language-specific files (go.mod, pyproject.toml, package.json, etc.)
   - Identify which formatters and linters are needed

2. **Check Tool Availability**
   - Verify required tools are installed
   - Suggest installation commands if missing

3. **Run Formatters First**
   - Format code to ensure consistent style
   - Report which files were modified

4. **Run Linters Second**
   - Check for code quality issues
   - Report issues with file paths and line numbers
   - Fix auto-fixable issues when possible

5. **Report Results**
   - Summarize what was done
   - List any remaining issues that need manual fixes
   - Provide actionable next steps

## Detection Patterns

```bash
# Go project
test -f go.mod && echo "Go project detected"

# Python project
test -f pyproject.toml -o -f requirements.txt -o -f setup.py && echo "Python project detected"

# Node.js project
test -f package.json && echo "Node.js project detected"

# Rust project
test -f Cargo.toml && echo "Rust project detected"
```

## Example Execution

### For Go Projects

```bash
# Format
if command -v gofumpt >/dev/null 2>&1; then
  echo "Running gofumpt..."
  gofumpt -w .
else
  echo "Running gofmt..."
  gofmt -w .
fi

# Tidy dependencies
go mod tidy

# Lint
if command -v golangci-lint >/dev/null 2>&1; then
  echo "Running golangci-lint..."
  golangci-lint run
else
  echo "Running go vet..."
  go vet ./...
fi
```

### For Python Projects

```bash
# Format
echo "Running ruff format..."
uvx ruff format .

# Fix auto-fixable issues
echo "Running ruff check --fix..."
uvx ruff check --fix .

# Show remaining issues
echo "Checking for remaining issues..."
uvx ruff check .
```

### For Multi-Language Projects

```bash
# Run formatters for all detected languages in parallel
if test -f go.mod; then
  echo "Formatting Go code..."
  gofumpt -w . &
fi

if test -f pyproject.toml; then
  echo "Formatting Python code..."
  uvx ruff format . &
fi

if test -f package.json; then
  echo "Formatting JS/TS code..."
  prettier --write "**/*.{js,ts,jsx,tsx}" &
fi

wait
echo "All formatting complete!"
```

## Tool Installation

### Go Tools
```bash
go install mvdan.cc/gofumpt@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Python Tools
```bash
# uv auto-installs tools via uvx
uv tool install ruff
```

### JavaScript Tools
```bash
bun add -D prettier eslint
# or
npm install -D prettier eslint
```

## Best Practices

1. **Always format before linting** - formatters fix style, linters check logic
2. **Run in project root** - ensures all files are processed
3. **Check exit codes** - non-zero means issues were found
4. **Preserve user's formatter config** - respect .prettierrc, .golangci.yml, etc.
5. **Report clearly** - show what was fixed vs what needs manual attention
6. **Use parallel execution** - format multiple languages simultaneously for speed

## Integration with Task Runners

If the project uses Task, Makefile, or similar:
```bash
# Check for task runner first
if command -v task >/dev/null 2>&1 && test -f Taskfile.yml; then
  echo "Using task runner..."
  task format
  task lint
elif test -f Makefile; then
  echo "Using Makefile..."
  make format
  make lint
else
  # Run directly
  echo "Running formatters directly..."
fi
```

## Common Issues and Solutions

### Issue: "command not found"
- Check if tool is installed: `command -v <tool>`
- Suggest installation command
- Fall back to alternative tool if available

### Issue: "permission denied"
- Check file permissions
- Suggest running with appropriate permissions

### Issue: "configuration file not found"
- Many tools work with defaults
- Suggest creating config file if customization needed

### Issue: Conflicting formatters
- Use project's preferred tool (check .pre-commit-config.yaml, package.json scripts)
- Document which formatter was used

## Output Format

Provide clear, actionable output:

```
✅ Format and Lint Complete

Go:
  ✓ Formatted 15 files with gofumpt
  ✓ go mod tidy completed
  ⚠ 3 issues found by golangci-lint:
    - internal/core/registry.go:42: unused variable 'ctx'
    - internal/hooks/format.go:105: error return not checked

Python:
  ✓ Formatted 8 files with ruff
  ✓ Fixed 12 auto-fixable issues
  ✓ No remaining issues

Next steps:
  1. Review and fix the 3 Go linting issues above
  2. Run tests to ensure nothing broke
  3. Commit the formatted code
```

## When NOT to Auto-Fix

- Don't auto-fix if it might break tests
- Don't auto-fix complex logic errors that need human judgment
- Don't auto-fix in files with merge conflicts
- Always review auto-fixes in critical files (security, config)

## Notes

- This skill respects the user's existing formatter configurations
- Uses modern tools (gofumpt, ruff, prettier) with sensible defaults
- Follows the user's global instructions preference for modern CLI tools (fd, rg, uvx, bun)
- Can be invoked explicitly or runs automatically when code quality is important