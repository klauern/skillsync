# Quick Start Guide

Welcome to SkillSync! This guide will help you get started with synchronizing AI coding skills across Claude Code, Cursor, and Codex platforms.

## What is SkillSync?

SkillSync is a command-line tool that helps you manage and synchronize AI coding skills across different platforms. It prevents duplication, ensures consistency, and gives you a unified repository for all your AI coding assistant skills.

**Key Features:**
- Synchronize skills between Claude Code, Cursor, and Codex
- Detect and resolve duplicate or similar skills
- Manage skills across different scopes (repo, user, system)
- Interactive TUI for visual skill management
- Automatic backups before sync operations
- Flexible conflict resolution strategies

## Installation

### Prerequisites

- Go 1.25.4 or later
- Git (for cloning the repository)
- One or more supported AI coding platforms:
  - Claude Code
  - Cursor
  - Codex

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/skillsync
cd skillsync

# Build the binary
make build

# Install to GOPATH/bin (optional)
make install

# Or run directly
./bin/skillsync --version
```

### Verify Installation

```bash
skillsync version
```

You should see output showing the version, git commit, build date, and Go version.

## First-Time Setup

### 1. Initialize Configuration

Create a default configuration file:

```bash
skillsync config init
```

This creates `~/.skillsync/config.yaml` with default settings.

### 2. View Current Configuration

```bash
# Show configuration in YAML format
skillsync config show

# Show configuration in JSON format
skillsync config show --format json

# Open configuration file in your editor
skillsync config edit
```

### 3. Find Configuration Location

```bash
skillsync config path
```

This shows where your config file is located.

## Discovering Your Skills

### Basic Discovery

List all skills across all platforms:

```bash
skillsync discover
```

This displays a table showing all skills found on your system with their:
- Name
- Platform (claude-code, cursor, codex)
- Scope (repo, user, admin, system, builtin, plugin)
- Status

### Filter by Platform

```bash
# Show only Claude Code skills
skillsync discover --platform claude-code

# Show only Cursor skills
skillsync discover --platform cursor

# Show only Codex skills
skillsync discover --platform codex
```

### Filter by Scope

```bash
# Show only repository-level skills
skillsync discover --scope repo

# Show only user-level skills
skillsync discover --scope user

# Show multiple scopes
skillsync discover --scope repo,user

# Show only plugin skills (Claude Code only)
skillsync discover --platform claude-code --scope plugin

# Show user and plugin scopes together
skillsync discover --scope user,plugin
```

### Interactive TUI Mode

Launch the interactive dashboard for visual exploration:

```bash
skillsync discover --interactive
```

**Navigation:**
- Use arrow keys to navigate
- Press Enter to select
- Press `q` to quit

### Export Discovery Results

```bash
# Export as JSON
skillsync discover --format json

# Export as YAML
skillsync discover --format yaml

# Save to file
skillsync discover --format json > my-skills.json
```

## Your First Sync

### Understanding Sync Direction

**Important:** Sync is always unidirectional (source → target). Changes flow from source to target only.

### Basic Sync

Sync all skills from one platform to another:

```bash
# Sync from Cursor to Claude Code
skillsync sync cursor claude-code

# Sync from Claude Code to Codex
skillsync sync claude-code codex
```

By default, this:
- Creates a backup before syncing
- Uses the "overwrite" strategy (source replaces target)
- Syncs to the user scope on the target platform

### Preview Before Syncing (Recommended)

Always preview changes with dry-run mode first:

```bash
skillsync sync --dry-run cursor claude-code
```

This shows what would happen without making any changes.

### Sync Specific Scopes

Use the platform:scope syntax to control which skills are synced:

```bash
# Sync only repo-level skills from Cursor to Claude Code
skillsync sync cursor:repo claude-code:user

# Sync both repo and user skills from Cursor to Codex repo
skillsync sync cursor:repo,user codex:repo

# Sync user skills to user scope (default for target if not specified)
skillsync sync cursor:user claude-code
```

**Valid Source Scopes:** repo, user, admin, system, builtin, plugin (can specify multiple)
**Valid Target Scopes:** repo, user (writable locations only, single scope)

> **Tip:** Use `plugin` scope to sync skills from installed Claude Code plugins to other platforms like Cursor or Codex.

### Interactive Sync

Use interactive mode for full control:

```bash
skillsync sync --interactive cursor claude-code
skillsync sync --interactive --delete cursor claude-code
```

This launches a TUI where you can:
- Select which skills to sync
- Preview diffs for each skill
- Choose resolution strategy per-skill
- Confirm before applying changes
- In delete mode, select which matching target skills to remove

## Understanding Sync Strategies

SkillSync offers six conflict resolution strategies:

### 1. **overwrite** (Default)

Source replaces target unconditionally. Best for one-way synchronization from a primary source.

```bash
skillsync sync --strategy overwrite cursor claude-code
```

### 2. **skip**

Preserves existing target skills; only adds new skills. Best for initial population without overwriting.

```bash
skillsync sync --strategy skip cursor claude-code
```

### 3. **newer**

Only copies if source is more recent (by modification time). Best for multi-directional workflows.

```bash
skillsync sync --strategy newer cursor claude-code
```

### 4. **merge**

Appends source content to target with a separator. Best for combining content from multiple sources.

```bash
skillsync sync --strategy merge cursor claude-code
```

Result format:
```markdown
[original target content]

---

## Merged from: skill-name

[source content]
```

### 5. **three-way**

Intelligent merge with automatic conflict detection. Attempts to merge non-conflicting changes automatically.

```bash
skillsync sync --strategy three-way cursor claude-code
```

If conflicts are detected, they're marked with conflict markers:
```
<<<<<<< SOURCE
[source version]
=======
[target version]
>>>>>>> TARGET
```

### 6. **interactive**

Prompts for resolution choice on each conflict. Best for high-control scenarios.

```bash
skillsync sync --strategy interactive cursor claude-code
```

You'll be prompted to choose:
- **source**: Use source version
- **target**: Keep target version
- **merge**: Attempt automatic merge
- **skip**: Leave target unchanged

## Common Workflows

### Workflow 1: Sync from Primary Platform

You primarily use Cursor and want to sync to other platforms:

```bash
# Preview changes
skillsync sync --dry-run cursor:repo,user claude-code:user

# Apply sync
skillsync sync cursor:repo,user claude-code:user

# Sync to Codex too
skillsync sync cursor:repo,user codex:user
```

### Workflow 2: Keep Platforms in Sync

Sync between platforms while preserving newer versions:

```bash
# Claude Code to Cursor (only if Claude Code is newer)
skillsync sync --strategy newer claude-code:user cursor:user

# Cursor to Claude Code (only if Cursor is newer)
skillsync sync --strategy newer cursor:user claude-code:user
```

### Workflow 3: Merge Skills from Multiple Sources

Combine skills from different platforms:

```bash
# Merge Claude Code into Cursor
skillsync sync --strategy merge claude-code:user cursor:user

# Merge Codex into Cursor
skillsync sync --strategy merge codex:user cursor:user
```

### Workflow 4: Populate New Platform

Setting up a new platform for the first time:

```bash
# Skip existing skills (if any)
skillsync sync --strategy skip cursor:user claude-code:user
```

### Workflow 5: Interactive Review Before Sync

Carefully review each change:

```bash
skillsync sync --interactive --strategy three-way cursor claude-code
```

## Managing Backups

SkillSync automatically creates backups before sync operations.

### List Backups

```bash
# Show all backups
skillsync backup list

# Show backups for specific platform
skillsync backup list --platform claude-code

# Interactive TUI
skillsync backup list --interactive

# Export as JSON
skillsync backup list --format json
```

### Restore a Backup

```bash
# Preview restoration
skillsync backup restore <backup-id> --dry-run

# Restore backup
skillsync backup restore <backup-id>

# Force overwrite existing files
skillsync backup restore <backup-id> --overwrite
```

### Delete Old Backups

```bash
# Preview deletion
skillsync backup delete <backup-id> --dry-run

# Delete backup
skillsync backup delete <backup-id>

# Skip confirmation
skillsync backup delete <backup-id> --force
```

### Verify Backup Integrity

```bash
skillsync backup verify <backup-id>
```

## Finding Duplicate Skills

### Compare Skills Across Platforms

Find similar or duplicate skills:

```bash
# Find all similar skills
skillsync compare

# Only compare by name
skillsync compare --name-only

# Only compare by content
skillsync compare --content-only

# Adjust similarity thresholds
skillsync compare --name-threshold 0.8 --content-threshold 0.7

# Find duplicates within same platform only
skillsync compare --same-platform

# Export results
skillsync compare --format json > duplicates.json
```

### Remove Duplicates

Once you've identified duplicates:

```bash
# Delete a duplicate skill
skillsync dedupe delete my-skill --platform cursor --scope user

# Preview deletion first
skillsync dedupe delete my-skill --platform cursor --scope user --dry-run

# Rename a skill to avoid conflicts
skillsync dedupe rename old-name new-name --platform cursor --scope user
```

## Managing Skill Scopes

### Understanding Scope Precedence

Skills can exist at different scope levels (from highest to lowest priority):

1. **plugin** - Claude Code plugin skills (`~/.claude/plugins/cache/*`) - *read-only*
2. **repo** - Repository-level (`.claude/skills`, `.cursor/skills`, `.codex/skills`)
3. **user** - User-level (`~/.claude/skills`, `~/.cursor/skills`, `~/.codex/skills`)
4. **admin** - Administrator-defined
5. **system** - System-wide installations
6. **builtin** - Built-in platform skills

> **Note:** Plugin scope skills are installed from Claude Code plugins and cannot be directly modified. They have the highest precedence, meaning a plugin skill will override any same-named skill in other scopes during discovery.

### List Skill Locations

Find all locations where a skill exists:

```bash
# Show all locations for a specific skill
skillsync scope list my-skill

# Filter by platform
skillsync scope list my-skill --platform claude-code

# List all skills grouped by scope
skillsync scope list --all
```

### Promote Skills to Higher Scope

Copy a skill from repo to user scope:

```bash
# Promote from repo to user
skillsync promote my-skill

# Preview first
skillsync promote my-skill --dry-run

# Promote and remove from source (move)
skillsync promote my-skill --remove-source

# Rename during promotion
skillsync promote my-skill --rename my-skill-v2
```

### Demote Skills to Lower Scope

Copy a skill from user to repo scope:

```bash
# Demote from user to repo
skillsync demote my-skill

# Preview first
skillsync demote my-skill --dry-run

# Demote and remove from source (move)
skillsync demote my-skill --remove-source
```

### Clean Up Duplicate Scopes

Remove duplicate skills from a scope:

```bash
# Preview cleanup for user scope
skillsync scope prune --scope user --dry-run

# Prune duplicates from user scope (keeps repo versions)
skillsync scope prune --scope user --keep-repo

# Prune from specific platform
skillsync scope prune --platform cursor --scope user
```

### Working with Plugin Skills

Plugin skills are installed from Claude Code plugins and have special characteristics:

- **Read-only:** Plugin skills cannot be modified directly; they're managed by the plugin system
- **Highest precedence:** Plugin skills override same-named skills from other scopes
- **Claude Code only:** Plugin scope is exclusive to Claude Code

**Discover plugin skills:**

```bash
# List all plugin skills
skillsync discover --platform claude-code --scope plugin

# See plugin skills alongside user skills
skillsync discover --scope user,plugin
```

**Sync plugin skills to other platforms:**

```bash
# Sync all plugin skills to Cursor user scope
skillsync sync claude-code:plugin cursor:user

# Sync plugin skills to Codex repo scope
skillsync sync claude-code:plugin codex:repo

# Preview what plugin skills would sync
skillsync sync --dry-run claude-code:plugin cursor:user
```

> **Note:** When syncing plugin skills to other platforms, the skills are copied to writable scopes (repo or user) in the target platform. The original plugin skills remain unchanged.

## Exporting Skills

Export skills to various formats for documentation or backup:

```bash
# Export all skills as JSON (default)
skillsync export

# Export as YAML
skillsync export --format yaml

# Export as Markdown
skillsync export --format markdown

# Filter by platform
skillsync export --platform claude-code --format yaml

# Save to file
skillsync export --output skills.json

# Compact output (no pretty-printing)
skillsync export --compact --output skills-compact.json

# Exclude metadata
skillsync export --no-metadata --format json
```

## Interactive TUI Dashboard

Launch the unified interactive dashboard:

```bash
skillsync tui
```

The TUI provides access to all SkillSync features:
- Discover and browse skills
- Manage backups
- Perform sync operations
- Compare and dedupe skills
- Promote/demote skills
- View configuration

**Navigation:**
- Arrow keys: Navigate menus
- Enter: Select option
- `q`: Quit
- `?`: Help (context-sensitive)

## Troubleshooting

### Common Issues

#### Issue: "No skills found"

**Possible causes:**
- Platforms not installed or skills directories don't exist
- Skills are in non-standard locations

**Solutions:**
```bash
# Check configuration
skillsync config show

# Verify paths exist
ls -la ~/.claude/skills
ls -la ~/.cursor/skills
ls -la ./.claude/skills  # repo-level

# Use discover with debug output
skillsync --debug discover
```

#### Issue: "Sync fails with permission errors"

**Possible causes:**
- Target directory doesn't exist
- Insufficient permissions on target directory

**Solutions:**
```bash
# Create target directory manually
mkdir -p ~/.claude/skills

# Check permissions
ls -ld ~/.claude/skills

# Fix permissions
chmod 755 ~/.claude/skills
```

#### Issue: "Conflicts detected but strategy is 'overwrite'"

**Explanation:**
The "overwrite" strategy doesn't detect conflicts—it always replaces target. If you're seeing conflict messages, you may be using "three-way" or "interactive" strategy.

**Solutions:**
```bash
# Explicitly set strategy
skillsync sync --strategy overwrite cursor claude-code

# Or use interactive to resolve manually
skillsync sync --interactive cursor claude-code
```

#### Issue: "Backup restore fails"

**Possible causes:**
- Backup file corrupted
- Target files already exist and --overwrite not specified

**Solutions:**
```bash
# Verify backup integrity first
skillsync backup verify <backup-id>

# Use --overwrite flag
skillsync backup restore <backup-id> --overwrite

# Check backup list for valid IDs
skillsync backup list
```

### Debug Mode

Enable verbose logging to troubleshoot issues:

```bash
# Info-level logging
skillsync --verbose discover

# Debug-level logging (includes source locations)
skillsync --debug sync cursor claude-code

# Disable colored output
skillsync --no-color discover
```

### Getting Help

```bash
# General help
skillsync --help

# Command-specific help
skillsync sync --help
skillsync discover --help
skillsync backup --help

# Show version and build info
skillsync version
```

## Configuration Reference

### Config File Location

Default: `~/.skillsync/config.yaml`

Find your config file:
```bash
skillsync config path
```

### Configuration Options

```yaml
# ~/.skillsync/config.yaml

# Supported platforms (claude-code, cursor, codex)
platforms:
  - claude-code
  - cursor
  - codex

# Default sync strategy (overwrite, skip, newer, merge, three-way, interactive)
default_strategy: overwrite

# Automatically create backups before sync
auto_backup: true

# Enable colored output
color: true

# Default output format (table, json, yaml)
output_format: table

# Backup retention (days, 0 = keep forever)
backup_retention_days: 30

# Plugin discovery
plugins:
  enabled: true
  cache_enabled: true
  cache_duration: 24h
```

### Environment Variables

Override config with environment variables:

```bash
# Disable colored output
export SKILLSYNC_NO_COLOR=1

# Set default strategy
export SKILLSYNC_DEFAULT_STRATEGY=three-way

# Disable auto-backup
export SKILLSYNC_AUTO_BACKUP=false
```

## Next Steps

- **Read the full command reference:** `docs/commands.md` (coming soon)
- **Learn about sync strategies:** `docs/sync-strategies.md` (coming soon)
- **Migration guide:** `docs/migration.md` (if you have existing skills)
- **Development guide:** See `AGENTS.md` if you want to contribute

## Getting Help

- View command help: `skillsync <command> --help`
- Report issues: [GitHub Issues](https://github.com/yourusername/skillsync/issues)
- Ask questions: [GitHub Discussions](https://github.com/yourusername/skillsync/discussions)

---

**Happy syncing!** 🚀
