# Test Fixtures for Skill Parsers

This directory contains comprehensive test fixtures for testing skill parsers across Claude Code, Cursor, and Codex platforms.

## Directory Structure

```
testdata/skills/
├── claude/          # Claude Code-specific fixtures
├── cursor/          # Cursor-specific fixtures
├── codex/           # Codex-specific fixtures
├── legacy/          # Legacy format fixtures (cross-platform)
└── invalid/         # Invalid fixtures for error testing
```

## Claude Code Fixtures

Located in `claude/`:

### Agent Skills Standard Format
- `basic-skill/` - Minimal SKILL.md with required fields only
- `full-agent-skill/` - Complete Agent Skills Standard with scripts/, references/, assets/
- `with-tools/` - Skill with multiple MCP tools specified
- `with-compatibility/` - Skill with platform version constraints
- `disable-invocation/` - Skill with model invocation disabled

### Scope Testing
- `builtin-scope-skill/` - Builtin scope (lowest precedence)
- `system-scope-skill/` - System scope
- `admin-scope-skill/` - Admin scope
- `repo-scope-skill/` - Repo scope (high precedence)
- `plugin-skill/` - Plugin scope (highest precedence)

### Metadata Variations
- `camelcase-frontmatter/` - CamelCase keys that should normalize to kebab-case

## Cursor Fixtures

Located in `cursor/`:

### File Formats
- `cursor-basic/` - Basic Cursor skill with SKILL.md
- `cursor-with-globs/` - Skill with glob patterns for file matching
- `cursor-single-glob/` - Skill with single glob pattern
- `cursor-complex-globs/` - Complex glob patterns with exclusions
- `cursor-mdc-format.mdc` - Cursor-specific .mdc file extension
- `cursor-no-description.md` - Cursor rule without description (optional field)

### Cursor-Specific Features
- `globs` array for file pattern matching
- `alwaysApply` boolean flag
- Optional description field (unlike Agent Skills Standard)

## Codex Fixtures

Located in `codex/`:

### Format Variations
- `codex-basic/` - Basic Codex skill with SKILL.md
- `codex-structured/` - Structured skill with scripts/ and assets/ subdirectories
- `codex-with-license/` - Skill with license specification

### Codex-Specific Formats
- `config-with-instructions/config.toml` - config.toml with embedded instructions
- `config-no-instructions/config.toml` - config.toml without instructions (no skill created)
- `agents-format/AGENTS.md` - Legacy AGENTS.md format
- `nested-agents/subproject/AGENTS.md` - Nested AGENTS.md for recursive discovery

### Codex Parser Behavior
- SKILL.md > config.toml > AGENTS.md (precedence order)
- config.toml only creates skill if `instructions` or `developer_instructions` present
- AGENTS.md skill names derived from directory (e.g., `subproject-agents`)

## Legacy Format Fixtures

Located in `legacy/`:

### Minimal Frontmatter
- `minimal-frontmatter.md` - Only name field, no description
- `no-frontmatter.md` - No YAML frontmatter at all (name from filename)
- `flat-skill.md` - Legacy flat .md file (not SKILL.md directory structure)

### Delimiter Variations
- `plus-delimiter.md` - Uses `+++` delimiter instead of `---`

### Metadata Key Variations
- `snake-case-keys.md` - snake_case keys (should normalize to kebab-case)
- `mixed-case-keys.md` - Mixed casing styles (PascalCase, camelCase, snake_case)

### Content Variations
- `windows-line-endings.md` - CRLF line endings (should normalize to LF)
- `unicode-content.md` - Unicode characters (emojis, math symbols, etc.)
- `whitespace-handling.md` - Extra whitespace in frontmatter and content

### Agent Skills Standard Compatibility
All legacy formats should parse on any platform (Claude Code, Cursor, or Codex).

## Invalid/Edge Case Fixtures

Located in `invalid/`:

### Malformed YAML
- `malformed-yaml/` - Invalid YAML indentation
- `unclosed-frontmatter.md` - Missing closing `---` delimiter

### Validation Errors
- `invalid-skill-name/` - Name with spaces and special characters
- `invalid-scope/` - Unrecognized scope value
- `missing-name/` - No name field (should derive from directory)
- `missing-description/` - No description field (required for Agent Skills Standard)

### Edge Cases
- `empty-file.md` - Completely empty file
- `very-large-content.md` - Very large file (1000+ lines) for size limit testing

### Expected Parser Behavior
- Malformed YAML: Error logged, skill skipped
- Invalid name: Error logged, skill skipped
- Missing name: Derived from filename or directory name
- Missing description: Valid for legacy formats, invalid for Agent Skills Standard
- Empty file: Skipped with warning
- Large content: Should parse successfully (no size limits)

## Testing Guidelines

### Parser Discovery Tests
Each parser should:
1. Discover all valid fixtures in its platform directory
2. Correctly parse frontmatter and content
3. Handle legacy formats with backward compatibility
4. Skip invalid fixtures with appropriate error messages

### Precedence Testing
Use fixtures with matching names across different scopes:
- `builtin-scope-skill` < `system-scope-skill` < `admin-scope-skill` < `repo-scope-skill` < `plugin-skill`

### File Format Testing
- Claude Code: Discovers `*.md` and `SKILL.md`
- Cursor: Discovers `*.md`, `*.mdc`, and `SKILL.md`
- Codex: Discovers `SKILL.md`, `AGENTS.md`, and `config.toml`

### Metadata Normalization
Test that parsers normalize:
- CamelCase → kebab-case
- snake_case → kebab-case
- Whitespace trimming
- Line ending normalization (CRLF → LF)

### Error Handling
Invalid fixtures should:
- Not crash the parser
- Log appropriate errors
- Skip the invalid skill
- Continue parsing other skills

## Usage in Tests

### Table-Driven Tests
```go
tests := map[string]struct {
    fixturePath string
    wantErr     bool
    wantSkills  int
}{
    "claude basic": {
        fixturePath: "testdata/skills/claude/basic-skill",
        wantErr:     false,
        wantSkills:  1,
    },
    "invalid yaml": {
        fixturePath: "testdata/skills/invalid/malformed-yaml",
        wantErr:     true,
        wantSkills:  0,
    },
}
```

### End-to-End Tests
```go
func TestParserDiscovery(t *testing.T) {
    h := e2e.NewHarness(t)
    fixture := h.ClaudeCodeFixture()

    // Copy fixtures to test environment
    fixture.CopyDir("testdata/skills/claude", ".claude/skills")

    // Run parser and verify results
    skills, err := parser.Parse()
    // assertions...
}
```

## Fixture Maintenance

When adding new fixtures:
1. Choose appropriate directory (claude/, cursor/, codex/, legacy/, invalid/)
2. Follow existing naming conventions
3. Update this README with fixture description
4. Add corresponding parser tests
5. Verify fixtures work with all relevant parsers

When modifying parsers:
1. Update fixtures to reflect new behavior
2. Add fixtures for new features
3. Update README documentation
4. Run full test suite to ensure backward compatibility
