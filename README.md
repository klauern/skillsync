# skillsync

[![Go Version](https://img.shields.io/badge/Go-1.25.4-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> Synchronize AI coding skills across Claude Code, Cursor, and Codex platforms

**skillsync** is a CLI tool that helps you manage and synchronize your AI coding assistant skills across multiple platforms. Keep your skills organized, avoid duplicates, and ensure consistency across all your development environments.

## Features

- **Multi-Platform Support**: Sync skills across Claude Code, Cursor, and Codex
- **Intelligent Synchronization**: Multiple merge strategies including conflict detection and resolution
- **Duplicate Detection**: Find and manage duplicate skills with content similarity analysis
- **Scope Management**: Organize skills across repo, user, admin, and system scopes
- **Backup & Rollback**: Automatic backups with easy rollback functionality
- **Interactive TUI**: Rich terminal interface for visual skill management
- **Platform Auto-Detection**: Automatically detects installed AI coding platforms
- **Plugin Discovery**: Discovers and manages Claude Code plugin skills
- **Flexible Export**: Export skills to JSON, YAML, or Markdown formats
- **Configurable**: YAML-based configuration with environment variable overrides

## Quick Start

### Installation

#### From Source

```bash
# Clone the repository
git clone https://github.com/klauern/skillsync.git
cd skillsync

# Build and install
make install
```

#### Manual Build

```bash
# Build to bin/skillsync
make build

# Run directly
./bin/skillsync --help
```

### Prerequisites

- Go 1.25.4 or later
- One or more AI coding platforms installed (Claude Code, Cursor, or Codex)

### Basic Usage

```bash
# Detect installed platforms
skillsync platforms

# List all skills
skillsync discover

# Sync skills from Claude Code to Cursor
skillsync sync claude-code cursor

# Launch interactive TUI
skillsync tui
```

## Installation

### From Source (Recommended for Development)

```bash
git clone https://github.com/klauern/skillsync.git
cd skillsync
make install
```

This will:
1. Build the binary with version information
2. Install it to `$GOPATH/bin/skillsync`
3. Make it available in your PATH (if GOPATH/bin is in PATH)

### Direct Binary Execution

```bash
make build
./bin/skillsync --help
```

### Development Installation

For development work, install the required tools:

```bash
make install-tools
```

This installs:
- `gofumpt` - Code formatting
- `goimports` - Import management
- `golangci-lint` - Comprehensive linting

## Usage

### Global Options

All commands support these global flags:

- `--verbose` - Enable verbose logging (info level)
- `--debug` - Enable debug logging with source locations
- `--no-color` - Disable colored output
- `--help, -h` - Show help
- `--version, -v` - Print version information

### Commands

#### `platforms` / `detect`

Detect installed AI coding platforms and show their configuration.

```bash
# Detect all platforms
skillsync platforms

# Show detailed detection info
skillsync platforms --verbose

# Output as JSON
skillsync platforms --format json
```

#### `discover` / `list`

List all skills across platforms with filtering options.

```bash
# List all skills (table format)
skillsync discover

# List skills from specific platform
skillsync discover -p claude-code

# Interactive TUI mode
skillsync discover -i

# List skills with plugin discovery
skillsync discover --repo

# Output as JSON
skillsync discover --format json

# Disable plugin discovery
skillsync discover --no-plugins

# Disable cache
skillsync discover --no-cache
```

#### `sync`

Synchronize skills between platforms with various strategies.

```bash
# Sync from Claude Code to Cursor (overwrite strategy)
skillsync sync claude-code cursor

# Sync with interactive TUI mode for skill selection
skillsync sync claude-code cursor --interactive

# Use interactive conflict resolution strategy
skillsync sync claude-code cursor --strategy interactive

# Dry-run to preview changes
skillsync sync claude-code cursor --dry-run

# Use three-way merge with automatic conflict detection
skillsync sync claude-code cursor --strategy three-way

# Use merge strategy (concatenates content)
skillsync sync claude-code cursor --strategy merge

# Skip backup creation
skillsync sync claude-code cursor --skip-backup

# Auto-confirm all prompts
skillsync sync claude-code cursor --yes
```

**Available Strategies:**
- `overwrite` - Replace destination skills (default)
- `skip` - Keep existing destination skills
- `newer` - Copy only if source is newer
- `merge` - Merge content from both sources with separator
- `three-way` - Intelligent three-way merge using LCS algorithm with conflict detection
- `interactive` - Prompt for each conflict with TUI interface (falls back to CLI in non-TTY environments)

**Interactive Modes:**
- `--interactive` flag: TUI mode for selecting which skills to sync, with diff preview
- `--strategy interactive`: TUI/CLI conflict resolution for handling conflicts during sync

**Platform Specification Format:**
```
platform[:scope[,scope2,...]]
```

**Supported Scopes:**
- `repo` - Repository-local skills (.claude/skills, .cursor/skills)
- `user` - User-level skills (~/.claude/skills, ~/.cursor/skills)
- `admin` - Admin-level skills (varies by platform)
- `system` - System-wide skills (/etc/codex/skills)
- `builtin` - Built-in platform skills
- `plugin` - Plugin-provided skills (Claude Code only)

Examples:
```bash
# Sync only user-level skills
skillsync sync claude-code:user cursor:user

# Sync repo and user scopes
skillsync sync claude-code:repo,user cursor:repo,user
```

#### `compare` / `cmp`

Find similar skills across or within platforms.

```bash
# Compare all skills across all platforms
skillsync compare

# Compare specific platforms
skillsync compare -p claude-code,cursor

# Show detailed comparison
skillsync compare --format unified

# Use custom similarity threshold
skillsync compare --threshold 0.8

# Output as JSON
skillsync compare --format json
```

**Output Formats:**
- `table` - Summary table (default)
- `unified` - Unified diff view
- `side-by-side` - Side-by-side comparison
- `summary` - High-level summary
- `json` - Machine-readable JSON
- `yaml` - YAML format

#### `dedupe` / `cleanup`

Remove or rename duplicate skills.

```bash
# Delete duplicates from Cursor's user scope
skillsync dedupe delete -p cursor -s user

# Rename duplicates instead of deleting
skillsync dedupe rename -p cursor -s user

# Preview changes without applying
skillsync dedupe delete -p cursor -s user --dry-run

# Force deletion without confirmation
skillsync dedupe delete -p cursor -s user --force
```

#### `promote`

Move skills to higher precedence scopes.

```bash
# Promote skill from repo to user scope
skillsync promote my-skill -p claude-code -f repo -t user

# Rename during promotion
skillsync promote my-skill -p cursor -f user -t admin --rename my-skill-v2

# Preview promotion
skillsync promote my-skill -p codex -f repo -t user --dry-run

# Force overwrite existing skill in target
skillsync promote my-skill -p claude-code -f repo -t user --force
```

#### `demote`

Move skills to lower precedence scopes.

```bash
# Demote skill from user to repo scope
skillsync demote my-skill -p claude-code -f user -t repo

# Rename during demotion
skillsync demote my-skill -p cursor -f admin -t user --rename my-skill-local

# Preview demotion
skillsync demote my-skill -p codex -f user -t repo --dry-run
```

#### `scope`

Manage skill scopes and locations.

```bash
# List all locations where a skill exists
skillsync scope list my-skill

# Remove skill from shadowed scopes (keep highest precedence only)
skillsync scope prune my-skill

# Preview prune operation
skillsync scope prune my-skill --dry-run

# Output as JSON
skillsync scope list my-skill --format json
```

#### `backup`

Manage skill backups and rollback changes.

```bash
# List all backups
skillsync backup list

# Restore from specific backup
skillsync backup restore <backup-id>

# Rollback last changes (undo)
skillsync backup rollback

# Delete old backups
skillsync backup delete <backup-id>

# Verify backup integrity
skillsync backup verify <backup-id>

# List backups as JSON
skillsync backup list --format json
```

Backups are automatically created before sync operations unless `--skip-backup` is specified.

#### `export`

Export skills to portable formats.

```bash
# Export all skills to JSON
skillsync export -o skills.json

# Export specific platform
skillsync export -p claude-code -o claude-skills.yaml --format yaml

# Export to stdout
skillsync export --format json

# Export as Markdown documentation
skillsync export -o skills.md --format markdown
```

#### `config`

Manage skillsync configuration.

```bash
# Show current configuration
skillsync config show

# Show as JSON
skillsync config show --format json

# Initialize default config file
skillsync config init

# Show config file path
skillsync config path

# Edit config in $EDITOR
skillsync config edit
```

#### `tui` / `ui`

Launch interactive terminal UI for visual skill management.

```bash
skillsync tui
```

The TUI provides:
- Visual skill browsing with keyboard navigation
- Sync wizard with strategy selection
- Backup management interface
- Compare and dedupe interactive selection
- Configuration editor
- Scope management visualization

**Keyboard Shortcuts:**
- `↑/↓` or `j/k` - Navigate
- `Enter` - Select
- `Tab` - Switch panels
- `?` - Help
- `q` - Quit

#### `version`

Display version and build information.

```bash
skillsync version
```

Shows:
- Version number
- Git commit hash
- Build date
- Go version

## Configuration

### Configuration File

Default location: `~/.skillsync/config.yaml`

Create a config file with:

```bash
skillsync config init
```

### Configuration Structure

```yaml
# Platform-specific settings
platforms:
  claude_code:
    skills_paths:
      - .claude/skills      # Repo-level
      - ~/.claude/skills    # User-level
    backup_enabled: true

  cursor:
    skills_paths:
      - .cursor/skills
      - ~/.cursor/skills
    backup_enabled: true

  codex:
    skills_paths:
      - .codex/skills
      - ~/.codex/skills
      - /etc/codex/skills   # System-level
    backup_enabled: true

# Synchronization settings
sync:
  default_strategy: overwrite
  auto_backup: true
  backup_retention_days: 30

# Cache settings
cache:
  enabled: true
  ttl: 1h
  location: ~/.skillsync/cache

# Plugin settings
plugins:
  enabled: true
  cache_plugins: true
  auto_fetch: false

# Output settings
output:
  format: table           # table, json, yaml
  color: auto            # auto, always, never
  verbose: false

# Backup settings
backup:
  enabled: true
  location: ~/.skillsync/backups
  max_backups: 10
  cleanup_on_sync: true

# Similarity detection settings
similarity:
  name_threshold: 0.7
  content_threshold: 0.6
  algorithm: combined     # name, content, combined
```

### Environment Variables

Override config settings with environment variables:

```bash
# Set custom config path
export SKILLSYNC_CONFIG_PATH=/path/to/config.yaml

# Set platform paths
export SKILLSYNC_CLAUDE_CODE_PATH=~/.config/claude/skills
export SKILLSYNC_CURSOR_PATH=~/.config/cursor/skills
export SKILLSYNC_CODEX_PATH=~/.config/codex/skills

# Cache settings
export SKILLSYNC_CACHE_ENABLED=true
export SKILLSYNC_CACHE_TTL=2h

# Logging
export SKILLSYNC_LOG_LEVEL=debug

# Disable color output
export NO_COLOR=1
```

All config keys can be overridden with `SKILLSYNC_` prefix (case-insensitive, dots become underscores).

## Platform Support

### Claude Code

- **Config Directory**: `~/.claude/`
- **Skill Locations**:
  - Repo: `.claude/skills/`
  - User: `~/.claude/skills/`
  - Plugins: `~/.claude/plugins/*/`
- **Supported Formats**:
  - Agent Skills Standard (SKILL.md)
  - Legacy markdown (.md)
  - Plugin-based skills with symlinks
- **Plugin Discovery**: Automatic detection of installed Claude Code plugins

### Cursor

- **Config Directory**: `~/.cursor/`
- **Skill Locations**:
  - Repo: `.cursor/skills/`
  - User: `~/.cursor/skills/`
- **Supported Formats**:
  - Agent Skills Standard (SKILL.md)
  - Legacy markdown (.md)
  - Cursor-specific .mdc format
- **Metadata Support**: Globs, alwaysApply fields

### Codex

- **Config Directory**: `~/.codex/`
- **Skill Locations**:
  - Repo: `.codex/skills/`
  - User: `~/.codex/skills/`
  - System: `/etc/codex/skills/`
- **Supported Formats**:
  - Agent Skills Standard (SKILL.md)
  - AGENTS.md files
  - Legacy config.toml
- **Advanced Features**: Model configuration, approval policies, MCP server integration

### Scope Precedence

Skills are resolved with the following precedence (highest to lowest):

1. **repo** - Project-local skills
2. **user** - User-specific skills
3. **admin** - Admin-managed skills
4. **system** - System-wide skills
5. **plugin** - Plugin-provided skills (Claude Code only)
6. **builtin** - Platform built-in skills

When a skill exists in multiple scopes, the highest precedence version is used.

## Development

### Building

```bash
# Build binary
make build

# Build and run
make run

# Install to GOPATH/bin
make install

# Uninstall
make uninstall
```

### Testing

```bash
# Run all tests with coverage
make test

# View coverage report in browser
make test-coverage

# Run tests with race detection
go test -race ./...
```

### Code Quality

```bash
# Run all quality checks
make audit

# Format code
make fmt

# Run linter
make lint

# Run go vet
make vet

# Tidy dependencies
make tidy
```

Quality gates:
- **Formatting**: gofumpt + goimports
- **Linting**: golangci-lint with 15+ linters
- **Testing**: Race detection enabled, 5-minute timeout
- **Error Handling**: errcheck enforced
- **Complexity**: Max cyclomatic complexity of 15

### Project Structure

```
skillsync/
├── cmd/skillsync/          # CLI entry point
├── internal/               # Private packages
│   ├── cli/               # Command definitions and handlers
│   ├── parser/            # Platform-specific parsers
│   │   ├── claude/        # Claude Code parser
│   │   ├── cursor/        # Cursor parser
│   │   ├── codex/         # Codex parser
│   │   ├── plugin/        # Plugin discovery
│   │   └── skills/        # Common skill handling
│   ├── model/             # Data types (Platform, Skill, Scope)
│   ├── sync/              # Synchronization logic
│   ├── backup/            # Backup/restore functionality
│   ├── config/            # Configuration management
│   ├── detector/          # Platform auto-detection
│   ├── similarity/        # Duplicate detection
│   ├── validation/        # Skill validation
│   ├── cache/             # Caching system
│   ├── export/            # Export functionality
│   ├── ui/                # User interface components
│   │   ├── color.go       # Color management
│   │   └── tui/           # Terminal UI (bubbletea)
│   ├── logging/           # Structured logging (slog)
│   └── util/              # Utilities and test helpers
├── testdata/              # Test fixtures
├── Makefile               # Build automation
├── go.mod                 # Go module definition
└── README.md              # This file
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run quality checks (`make audit`)
5. Commit your changes (`git commit -m 'feat: add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Conventions

- Follow Go best practices and idioms
- Use gofumpt for formatting (enforced)
- All exported functions must have documentation
- Always check errors (errcheck linter enabled)
- Write table-driven tests with testify
- Keep cyclomatic complexity ≤ 15
- Use structured logging with slog

## Architecture

### Core Concepts

#### Skills

A skill is a reusable instruction or capability for an AI coding assistant. Skills can contain:
- Instructions for the AI
- Code templates
- Workflow definitions
- Tool integrations
- Context and examples

#### Platforms

skillsync supports three AI coding platforms:
- **Claude Code** (`claude-code`) - Anthropic's coding assistant
- **Cursor** (`cursor`) - AI-first code editor
- **Codex** (`codex`) - OpenAI's coding platform

#### Scopes

Skills are organized into scopes with precedence:
- **repo** - Project-specific (`.claude/skills/`)
- **user** - User-specific (`~/.claude/skills/`)
- **admin** - Admin-managed (platform-dependent)
- **system** - System-wide (`/etc/codex/skills/`)
- **plugin** - Plugin-provided (Claude Code only)
- **builtin** - Platform built-in

### Synchronization

The sync engine uses pluggable strategies:

1. **Overwrite** - Replace destination unconditionally
2. **Skip** - Keep existing destination skills
3. **Newer** - Copy only if source is newer (mtime-based)
4. **Merge** - Merge content from source and destination
5. **Three-way** - Intelligent merge with common ancestor detection
6. **Interactive** - Prompt user for each conflict

Conflict detection uses:
- Content hashing (SHA256)
- Modification timestamps
- Three-way comparison for merge conflicts

### Similarity Detection

Duplicate detection uses multiple algorithms:

#### Name-Based
- **Levenshtein Distance** - Character edit distance
- **Jaro-Winkler** - String similarity with prefix weighting

#### Content-Based
- **LCS (Longest Common Subsequence)** - Shared content length
- **Jaccard Similarity** - Set intersection/union ratio

Configurable thresholds allow tuning sensitivity.

### Backup System

Automatic backups before sync operations:
- SHA256 content hashing for integrity
- Metadata tracking (timestamp, platform, tags)
- Indexed backup management
- Rollback to any previous state
- Automatic cleanup of old backups (configurable retention)

### Caching

Intelligent caching improves performance:
- JSON-based cache with version tracking
- Source file modification detection
- TTL-based expiration (default 1 hour)
- Automatic invalidation on source changes
- Cache location: `~/.skillsync/cache`

## Troubleshooting

### Platform Not Detected

If `skillsync platforms` doesn't detect your platform:

1. **Check installation**: Verify the platform is installed
   ```bash
   ls ~/.claude    # Claude Code
   ls ~/.cursor    # Cursor
   ls ~/.codex     # Codex
   ```

2. **Set explicit path** with environment variable:
   ```bash
   export SKILLSYNC_CLAUDE_CODE_PATH=~/.config/claude
   ```

3. **Verify config file**:
   ```bash
   skillsync config show
   ```

### Sync Conflicts

If sync operations produce conflicts:

1. **Use interactive conflict resolution** (recommended):
   ```bash
   skillsync sync claude-code cursor --strategy interactive
   ```

   This launches a TUI where you can:
   - View detailed diffs for each conflict
   - Choose to use source, target, or merge content
   - Resolve conflicts one by one with visual feedback
   - Auto-falls back to CLI prompts in non-TTY environments (CI/CD)

2. **Use three-way merge** (automatic):
   ```bash
   skillsync sync claude-code cursor --strategy three-way
   ```

   Uses the Longest Common Subsequence (LCS) algorithm to intelligently merge:
   - Detects conflicts only when both sides modify the same region
   - Successfully merges non-overlapping changes automatically
   - Marks unresolvable conflicts with conflict markers (`<<<<<<< SOURCE`, `=======`, `>>>>>>> TARGET`)

3. **Preview with dry-run**:
   ```bash
   skillsync sync claude-code cursor --dry-run
   ```

4. **Use skill selection TUI**:
   ```bash
   skillsync sync claude-code cursor --interactive
   ```

   Select which skills to sync with diff preview before syncing

5. **Check backup**:
   ```bash
   skillsync backup list
   skillsync backup rollback
   ```

### Performance Issues

If operations are slow:

1. **Enable caching**:
   ```yaml
   cache:
     enabled: true
     ttl: 2h
   ```

2. **Disable plugin discovery**:
   ```bash
   skillsync discover --no-plugins
   ```

3. **Clear cache**:
   ```bash
   rm -rf ~/.skillsync/cache
   ```

### Duplicate Skills

If you have many duplicates:

1. **Find them**:
   ```bash
   skillsync compare
   ```

2. **Preview cleanup**:
   ```bash
   skillsync dedupe delete -p cursor --dry-run
   ```

3. **Remove duplicates**:
   ```bash
   skillsync dedupe delete -p cursor
   ```

4. **Or rename instead**:
   ```bash
   skillsync dedupe rename -p cursor
   ```

### Debug Mode

For troubleshooting, enable debug logging:

```bash
skillsync --debug discover
```

Or set environment variable:

```bash
export SKILLSYNC_LOG_LEVEL=debug
skillsync discover
```

## FAQ

### Q: How do I sync skills between multiple machines?

Use `export` and import functionality, or sync your config directories directly:

```bash
# Machine 1: Export skills
skillsync export -o skills.json

# Machine 2: Import skills (copy file and import)
# Or use git to sync ~/.claude/skills directory
```

### Q: Can I sync only specific skills?

Currently, sync operates on all skills in a scope. Use scope filtering:

```bash
# Sync only user-level skills
skillsync sync claude-code:user cursor:user
```

### Q: How do I backup before making changes?

Backups are automatic unless disabled:

```bash
# Automatic backup (default)
skillsync sync claude-code cursor

# Skip backup
skillsync sync claude-code cursor --skip-backup

# Manual rollback
skillsync backup rollback
```

### Q: What happens during a conflict?

Depends on your strategy:
- `overwrite` - Destination replaced
- `skip` - Destination kept
- `newer` - Newer version wins
- `merge` - Content merged
- `three-way` - Intelligent merge with conflict detection
- `interactive` - You decide for each conflict

### Q: How do I contribute a new platform parser?

1. Implement the `parser.Parser` interface in `internal/parser/`
2. Add platform constants to `internal/model/platform.go`
3. Register parser in platform detection
4. Add tests with fixtures in `testdata/`
5. Update documentation

### Q: Can I use skillsync in CI/CD?

Yes, with non-interactive mode:

```bash
skillsync sync claude-code cursor --yes --no-color
```

### Q: How do I uninstall?

```bash
make uninstall
rm -rf ~/.skillsync
```

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [urfave/cli](https://github.com/urfave/cli) - CLI framework
- TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- Styling with [Lipgloss](https://github.com/charmbracelet/lipgloss)

## Links

- [GitHub Repository](https://github.com/klauern/skillsync)
- [Issue Tracker](https://github.com/klauern/skillsync/issues)
- [Claude Code](https://claude.ai/code)
- [Cursor](https://cursor.sh)
- [Agent Skills Standard](https://github.com/anthropics/agent-skills-standard)

---

Made with ❤️ for AI coding assistant users
