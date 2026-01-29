# Command Reference

Complete reference for all SkillSync commands, flags, and usage patterns.

## Table of Contents

- [Overview](#overview)
- [Global Flags](#global-flags)
- [Commands](#commands)
  - [version](#version)
  - [config](#config)
  - [discover](#discover)
  - [sync](#sync)
  - [compare](#compare)
  - [dedupe](#dedupe)
  - [export](#export)
  - [backup](#backup)
  - [promote](#promote)
  - [demote](#demote)
  - [scope](#scope)
  - [tui](#tui)
- [Common Patterns](#common-patterns)
- [Quick Reference Table](#quick-reference-table)

---

## Overview

SkillSync provides 12 commands for managing AI coding skills across platforms:

- **Configuration**: `config`, `version`
- **Discovery**: `discover`, `tui`
- **Synchronization**: `sync`, `compare`, `dedupe`
- **Export/Backup**: `export`, `backup`
- **Scope Management**: `promote`, `demote`, `scope`

### Command Syntax

```
skillsync [global-flags] <command> [command-flags] [arguments]
```

### Getting Help

```bash
skillsync --help              # List all commands
skillsync <command> --help    # Command-specific help
skillsync <command> <subcommand> --help  # Subcommand help
```

---

## Global Flags

These flags work with all commands:

| Flag | Aliases | Description |
|------|---------|-------------|
| `--verbose` | | Enable info-level logging |
| `--debug` | | Enable debug-level logging (implies --verbose) |
| `--no-color` | | Disable colored output |

**Examples:**

```bash
skillsync --debug sync cursor claudecode       # Debug logging
skillsync --no-color discover --platform cursor # No colors
skillsync --verbose backup list                # Verbose output
```

---

## Commands

### version

Display version and build information.

**Usage:**

```bash
skillsync version
skillsync v
```

**Output:**

```
SkillSync v1.0.0
Build: 2026-01-28
Commit: abc1234
Go: go1.25.4
```

**Aliases:** `v`, `version`

---

### config

Manage skillsync configuration.

**Usage:**

```bash
skillsync config <subcommand> [flags]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `show` | Display current configuration |
| `init` | Initialize configuration file |
| `path` | Show configuration file path |
| `edit` | Open configuration in $EDITOR |

#### config show

Display current configuration with resolved values.

```bash
skillsync config show              # YAML format
skillsync config show -f json      # JSON format
skillsync config show -f table     # Table format
```

**Flags:**
- `-f, --format`: Output format (yaml, json, table) [default: yaml]

**Example Output:**

```yaml
platforms:
  claudecode:
    path: ~/.config/claude/skills
    scopes: [repo, user]
  cursor:
    path: ~/.cursor/skills
    scopes: [repo, user, system]

compare:
  name_threshold: 0.7
  content_threshold: 0.8
  algorithm: combined

backup:
  enabled: true
  max_count: 5
  dir: ~/.local/share/skillsync/backups
```

#### config init

Create a new configuration file interactively.

```bash
skillsync config init              # Interactive setup
skillsync config init --force      # Overwrite existing
```

**Flags:**
- `-f, --force`: Overwrite existing configuration

Walks through platform paths, default scopes, and preferences.

#### config path

Show the path to the configuration file.

```bash
skillsync config path
```

**Output:**

```
/Users/username/.config/skillsync/config.yaml
```

#### config edit

Open configuration in your default editor (`$EDITOR`).

```bash
skillsync config edit
```

Opens the config file in vim, nano, or your configured editor.

---

### discover

List and explore skills from platforms.

**Usage:**

```bash
skillsync discover [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `-p` | all | Platform to discover (claudecode, cursor, codex) |
| `--scope` | `-s` | all | Scope filter (repo, user, system, admin, builtin) |
| `--interactive` | `-i` | false | Launch interactive picker |
| `--format` | `-f` | table | Output format (table, json, yaml, markdown) |
| `--no-plugins` | | false | Skip plugin-loaded skills |
| `--repo` | | . | Repository path for repo-scoped skills |

**Examples:**

```bash
# Discover all skills
skillsync discover

# Discover from specific platform
skillsync discover --platform cursor

# Discover specific scope
skillsync discover --platform cursor --scope repo

# Interactive selection
skillsync discover --interactive

# JSON output for scripting
skillsync discover --platform codex --format json

# Skip plugins (builtin only)
skillsync discover --no-plugins

# Discover in different repo
skillsync discover --scope repo --repo ~/projects/myapp
```

**Output (table format):**

```
PLATFORM     SCOPE   NAME            SIZE  MODIFIED
cursor       repo    commit-helper   1.2KB 2026-01-27
cursor       user    debug-wizard    3.4KB 2026-01-26
claudecode   user    test-runner     2.1KB 2026-01-25
```

**Interactive Mode:**

Launches a searchable, keyboard-navigable picker:
- `↑/↓` or `j/k`: Navigate
- `/`: Search by name
- `Enter`: View skill details
- `q`: Quit

---

### sync

Synchronize skills between platforms.

**Usage:**

```bash
skillsync sync [flags] <source> <target>
```

**Arguments:**

- `<source>`: Source platform spec (platform[:scope[,scope2,...]])
- `<target>`: Target platform spec (platform[:scope])

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--strategy` | `-s` | overwrite | Conflict resolution (see below) |
| `--dry-run` | `-d` | false | Preview changes without writing |
| `--interactive` | `-i` | false | Prompt for each conflict |
| `--skip-backup` | | false | Skip automatic backup |
| `--yes` | `-y` | false | Skip confirmation prompts |

**Conflict Resolution Strategies:**

| Strategy | Behavior |
|----------|----------|
| `overwrite` | Replace target skills unconditionally (default) |
| `skip` | Keep target skills, skip source if exists |
| `newer` | Copy only if source is newer than target |
| `merge` | Merge source and target content |
| `three-way` | Intelligent merge with conflict markers |
| `interactive` | Prompt for each conflict |

See [sync-strategies.md](sync-strategies.md) for detailed explanations.

**Platform Spec Format:**

```
platform[:scope[,scope2,...]]
```

- `cursor` → All scopes from cursor (source), user scope (target)
- `cursor:repo` → Only repo scope
- `cursor:repo,user` → Both repo and user scopes (source only)

**Valid Scopes:**
- `repo` - Repository-specific (writable)
- `user` - User-level (writable)
- `system` - System-level (read-only)
- `admin` - Admin-level (read-only)
- `builtin` - Built-in platform skills (read-only)

**Target scope must be writable** (`repo` or `user`).

**Examples:**

```bash
# Sync all cursor skills to claude-code user scope
skillsync sync cursor claudecode

# Sync repo skills to user scope
skillsync sync cursor:repo claudecode:user

# Multiple source scopes to repo
skillsync sync cursor:repo,user codex:repo

# Preview sync without changes
skillsync sync --dry-run cursor claudecode

# Use skip strategy (preserve target)
skillsync sync --strategy=skip cursor codex

# Interactive conflict resolution
skillsync sync --interactive cursor claudecode

# Newer-only sync
skillsync sync --strategy=newer cursor claudecode

# Skip backup and confirmation
skillsync sync --skip-backup --yes cursor claudecode
```

**Dry Run Output:**

```
[DRY RUN] Sync Plan: cursor → claudecode:user

Would copy (3 skills):
  + commit-helper (repo) → user
  + debug-wizard (user) → user
  + test-runner (user) → user

Would skip (1 skill):
  - existing-skill (already exists)

No changes made (dry run).
```

**Sync Output:**

```
Sync: cursor → claudecode:user
Creating backup: claudecode-20260128-143022

Copying skills:
  ✓ commit-helper (repo → user)
  ✓ debug-wizard (user → user)
  ⊕ test-runner (merged with existing)

Summary:
  Copied: 2
  Merged: 1
  Skipped: 1
  Total: 4

Backup saved: ~/.local/share/skillsync/backups/claudecode-20260128-143022.tar.gz
```

---

### compare

Find duplicate and similar skills across platforms.

**Usage:**

```bash
skillsync compare [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--name-threshold` | `-n` | 0.7 | Minimum name similarity (0.0-1.0) |
| `--content-threshold` | `-c` | 0.8 | Minimum content similarity (0.0-1.0) |
| `--platform` | `-p` | all | Filter by platform |
| `--format` | `-f` | table | Output format (table, json, yaml) |
| `--name-only` | | false | Compare only by name (skip content) |
| `--same-platform` | | false | Only find duplicates within same platform |

**Similarity Algorithms:**

The compare command uses a **combined similarity score**:

1. **Name Similarity** (Jaro-Winkler distance):
   - Handles typos and minor variations
   - Prefix-weighted (e.g., "commit-helper" vs "commit-assist")
   - Score: 0.0 (completely different) to 1.0 (identical)

2. **Content Similarity** (Levenshtein distance):
   - Measures character-level edit distance
   - Normalized by length
   - Score: 0.0 (no match) to 1.0 (exact match)

3. **Combined Score**:
   - Weighted average: `(name * 0.4) + (content * 0.6)`
   - Content weighted higher (actual functionality matters more)

**Threshold Interpretation:**

- `1.0` - Exact match
- `0.9-0.99` - Near-identical (minor whitespace/comment differences)
- `0.8-0.89` - Very similar (likely duplicates)
- `0.7-0.79` - Similar (worth reviewing)
- `< 0.7` - Different (not shown by default)

**Examples:**

```bash
# Find all similar skills
skillsync compare

# Strict matching (only very similar)
skillsync compare --name-threshold 0.9 --content-threshold 0.9

# Relaxed matching (catch more potential duplicates)
skillsync compare --name-threshold 0.6 --content-threshold 0.7

# Only compare names (fast)
skillsync compare --name-only

# Find duplicates within cursor only
skillsync compare --platform cursor --same-platform

# JSON output for scripting
skillsync compare --format json
```

**Output (table format):**

```
SKILL 1                  SKILL 2                  NAME SIM  CONTENT SIM  COMBINED
cursor:user/commit-help  claudecode:user/commit-helper  0.92    0.88        0.90
cursor:repo/test-runner  codex:user/run-tests           0.78    0.85        0.82
```

**Interpreting Results:**

- **High name similarity, low content** → Same purpose, different implementation
- **Low name similarity, high content** → Renamed or copied skill
- **Both high** → Likely duplicate, consider using `dedupe`

---

### dedupe

Remove duplicate and similar skills.

**Usage:**

```bash
skillsync dedupe <subcommand> [flags]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `delete` | Delete duplicate skills |
| `rename` | Rename conflicting skills |

#### dedupe delete

Delete duplicate skills after confirmation.

```bash
skillsync dedupe delete [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--name-threshold` | `-n` | 0.9 | Minimum name similarity to consider duplicate |
| `--content-threshold` | `-c` | 0.95 | Minimum content similarity to consider duplicate |
| `--platform` | `-p` | all | Platform to dedupe |
| `--keep` | | newest | Which to keep (newest, oldest, first, last) |
| `--dry-run` | `-d` | false | Preview deletions |
| `--yes` | `-y` | false | Skip confirmation |

**Keep Strategies:**

- `newest` - Keep most recently modified (default)
- `oldest` - Keep oldest skill
- `first` - Keep first in alphabetical order
- `last` - Keep last in alphabetical order

**Examples:**

```bash
# Preview deletions
skillsync dedupe delete --dry-run

# Delete duplicates, keep newest
skillsync dedupe delete --keep newest

# Delete with strict matching
skillsync dedupe delete -n 0.95 -c 0.98

# Skip confirmation
skillsync dedupe delete --yes

# Dedupe specific platform
skillsync dedupe delete --platform cursor
```

**Output:**

```
Found 3 duplicate groups:

Group 1 (similarity: 0.96):
  cursor:user/commit-helper (2026-01-27) ← keep (newest)
  claudecode:user/commit-help (2026-01-25) → delete

Group 2 (similarity: 0.92):
  cursor:repo/test-runner (2026-01-26) ← keep (newest)
  codex:user/run-tests (2026-01-24) → delete

Delete 2 skills? [y/N]:
```

#### dedupe rename

Rename skills to avoid conflicts.

```bash
skillsync dedupe rename [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--pattern` | | {name}-{n} | Rename pattern with placeholders |
| `--platform` | `-p` | all | Platform to rename in |
| `--dry-run` | `-d` | false | Preview renames |

**Pattern Placeholders:**

- `{name}` - Original skill name
- `{platform}` - Platform name
- `{scope}` - Scope name
- `{n}` - Sequential number

**Examples:**

```bash
# Rename with sequential numbers
skillsync dedupe rename --pattern "{name}-{n}"
# commit-helper → commit-helper-1, commit-helper-2

# Add platform prefix
skillsync dedupe rename --pattern "{platform}-{name}"
# commit-helper → cursor-commit-helper

# Add scope suffix
skillsync dedupe rename --pattern "{name}-{scope}"
# commit-helper → commit-helper-user

# Preview renames
skillsync dedupe rename --dry-run
```

**Output:**

```
Rename Plan:

  cursor:user/commit-helper → cursor:user/commit-helper-1
  claudecode:user/commit-helper → claudecode:user/commit-helper-2

Rename 2 skills? [y/N]:
```

---

### export

Export skills to various formats.

**Usage:**

```bash
skillsync export [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `-p` | all | Platform to export |
| `--format` | `-f` | json | Export format (json, yaml, markdown, tar) |
| `--output` | `-o` | stdout | Output file path |
| `--no-metadata` | | false | Exclude metadata (format-dependent) |
| `--compact` | | false | Compact JSON/YAML (no pretty-print) |

**Export Formats:**

| Format | Extension | Use Case |
|--------|-----------|----------|
| `json` | .json | API integration, scripting |
| `yaml` | .yaml | Human-readable, version control |
| `markdown` | .md | Documentation, sharing |
| `tar` | .tar.gz | Backup, archive, transfer |

**Examples:**

```bash
# Export all skills to JSON
skillsync export

# Export to file
skillsync export --output skills.json

# Export specific platform
skillsync export --platform cursor --output cursor-skills.yaml -f yaml

# Compact JSON
skillsync export --format json --compact

# Markdown documentation
skillsync export --format markdown --output SKILLS.md

# Create tarball backup
skillsync export --format tar --output backup.tar.gz

# Minimal JSON (no metadata)
skillsync export --format json --no-metadata
```

**JSON Output Structure:**

```json
{
  "metadata": {
    "exported_at": "2026-01-28T14:30:00Z",
    "skillsync_version": "1.0.0",
    "platforms": ["cursor", "claudecode"]
  },
  "skills": [
    {
      "name": "commit-helper",
      "platform": "cursor",
      "scope": "user",
      "content": "# Commit Helper\n...",
      "modified": "2026-01-27T10:15:00Z",
      "size": 1234
    }
  ]
}
```

**Markdown Output:**

```markdown
# Skills Export

Exported 3 skills on 2026-01-28

## cursor:user/commit-helper

Modified: 2026-01-27

```
# Commit Helper
...
```

## cursor:user/debug-wizard

...
```

**Tar Archive:**

Creates a compressed tarball with platform/scope directory structure:

```
backup.tar.gz
├── cursor/
│   ├── user/
│   │   ├── commit-helper.md
│   │   └── debug-wizard.md
│   └── repo/
│       └── test-runner.md
└── claudecode/
    └── user/
        └── my-skill.md
```

---

### backup

Manage skill backups.

**Usage:**

```bash
skillsync backup <subcommand> [flags]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `list` | List available backups |
| `restore` | Restore from backup |
| `delete` | Delete backups |
| `verify` | Verify backup integrity |

Backups are automatically created before sync operations (unless `--skip-backup` is used).

#### backup list

List all available backups.

```bash
skillsync backup list [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `-p` | all | Filter by platform |
| `--format` | `-f` | table | Output format (table, json, yaml) |

**Examples:**

```bash
# List all backups
skillsync backup list

# Filter by platform
skillsync backup list --platform cursor

# JSON output
skillsync backup list --format json
```

**Output:**

```
ID                        PLATFORM     DATE                SIZE   SKILLS
claudecode-20260128-1430  claudecode   2026-01-28 14:30    4.5MB  12
cursor-20260128-1015      cursor       2026-01-28 10:15    3.2MB  8
claudecode-20260127-0900  claudecode   2026-01-27 09:00    4.4MB  12
```

#### backup restore

Restore skills from a backup.

```bash
skillsync backup restore <backup-id> [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--force` | `-f` | false | Overwrite existing skills |
| `--interactive` | `-i` | false | Prompt for each conflict |
| `--dry-run` | `-d` | false | Preview restore |

**Examples:**

```bash
# Preview restore
skillsync backup restore claudecode-20260128-1430 --dry-run

# Restore backup
skillsync backup restore claudecode-20260128-1430

# Force overwrite
skillsync backup restore cursor-20260128-1015 --force

# Interactive restore
skillsync backup restore claudecode-20260127-0900 --interactive
```

**Output:**

```
Restoring backup: claudecode-20260128-1430

  ✓ commit-helper
  ⊕ debug-wizard (conflict: interactive merge)
  ✓ test-runner

Restored: 12 skills
Conflicts: 1 (merged)
```

#### backup delete

Delete one or more backups.

```bash
skillsync backup delete [flags] [backup-id...]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `-p` | all | Delete all for platform |
| `--older-than` | | | Delete backups older than duration (e.g., "30d") |
| `--keep-last` | | | Keep N most recent backups |
| `--yes` | `-y` | false | Skip confirmation |

**Examples:**

```bash
# Delete specific backup
skillsync backup delete claudecode-20260127-0900

# Delete multiple backups
skillsync backup delete claudecode-20260127-0900 cursor-20260126-1200

# Delete old backups (keep last 5)
skillsync backup delete --keep-last 5

# Delete backups older than 30 days
skillsync backup delete --older-than 30d

# Delete all cursor backups
skillsync backup delete --platform cursor --yes
```

**Output:**

```
Delete backups:
  claudecode-20260127-0900 (4.4MB)
  cursor-20260126-1200 (3.1MB)

Total: 7.5MB

Delete 2 backups? [y/N]:
```

#### backup verify

Verify backup integrity and list contents.

```bash
skillsync backup verify <backup-id> [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--format` | `-f` | table | Output format (table, json) |

**Examples:**

```bash
# Verify backup
skillsync backup verify claudecode-20260128-1430

# JSON output
skillsync backup verify claudecode-20260128-1430 --format json
```

**Output:**

```
Backup: claudecode-20260128-1430
Status: ✓ Valid

Contents:
  commit-helper.md (1.2KB)
  debug-wizard.md (3.4KB)
  test-runner.md (2.1KB)

Total: 12 skills, 4.5MB
Integrity: All files verified
```

---

### promote

Move skills to higher scope.

**Usage:**

```bash
skillsync promote <skill-name> [flags]
```

**Arguments:**

- `<skill-name>`: Name of skill to promote

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `-p` | auto | Platform (auto-detected if unique) |
| `--from` | | repo | Source scope |
| `--to` | | user | Target scope |
| `--force` | `-f` | false | Overwrite if exists in target |
| `--rename` | | | New name for promoted skill |
| `--remove-source` | | false | Delete from source after promote |
| `--dry-run` | `-d` | false | Preview promotion |

**Scope Hierarchy:**

```
repo → user → system
```

You can only promote upward in the hierarchy.

**Examples:**

```bash
# Promote repo skill to user scope
skillsync promote my-skill

# Promote with explicit scopes
skillsync promote my-skill --from repo --to user

# Promote and rename
skillsync promote my-skill --rename new-name

# Promote and remove source
skillsync promote my-skill --remove-source

# Force overwrite
skillsync promote my-skill --force

# Preview promotion
skillsync promote my-skill --dry-run
```

**Output:**

```
Promote: cursor:repo/my-skill → cursor:user/my-skill

  ✓ Copied to user scope
  ✓ Removed from repo scope

Promoted: my-skill (repo → user)
```

---

### demote

Move skills to lower scope.

**Usage:**

```bash
skillsync demote <skill-name> [flags]
```

**Arguments:**

- `<skill-name>`: Name of skill to demote

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `-p` | auto | Platform (auto-detected if unique) |
| `--from` | | user | Source scope |
| `--to` | | repo | Target scope |
| `--force` | `-f` | false | Overwrite if exists in target |
| `--dry-run` | `-d` | false | Preview demotion |

**Scope Hierarchy:**

```
system → user → repo
```

You can only demote downward in the hierarchy.

**Examples:**

```bash
# Demote user skill to repo scope
skillsync demote my-skill

# Demote with explicit scopes
skillsync demote my-skill --from user --to repo

# Force overwrite
skillsync demote my-skill --force

# Preview demotion
skillsync demote my-skill --dry-run

# Demote system skill to user
skillsync demote system-skill --from system --to user
```

**Output:**

```
Demote: cursor:user/my-skill → cursor:repo/my-skill

  ✓ Copied to repo scope
  ✓ Removed from user scope

Demoted: my-skill (user → repo)
```

---

### scope

Manage skill scopes.

**Usage:**

```bash
skillsync scope <subcommand> [flags]
```

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `list` | List all scopes and their contents |
| `info` | Show scope information |
| `validate` | Validate scope configuration |

**Scope Types:**

| Scope | Writable | Description |
|-------|----------|-------------|
| `repo` | ✓ | Repository-specific skills |
| `user` | ✓ | User-level skills |
| `system` | ✗ | System-wide skills (read-only) |
| `admin` | ✗ | Admin-level skills (read-only) |
| `builtin` | ✗ | Platform built-in skills (read-only) |

#### scope list

List all scopes and their skill counts.

```bash
skillsync scope list [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `-p` | all | Filter by platform |
| `--format` | `-f` | table | Output format (table, json, yaml) |

**Examples:**

```bash
# List all scopes
skillsync scope list

# Platform-specific
skillsync scope list --platform cursor

# JSON output
skillsync scope list --format json
```

**Output:**

```
PLATFORM     SCOPE    WRITABLE  PATH                                    SKILLS
cursor       repo     ✓         ~/dev/myproject/.cursor/skills          3
cursor       user     ✓         ~/.cursor/skills                        8
cursor       system   ✗         /usr/local/share/cursor/skills          12
claudecode   repo     ✓         ~/dev/myproject/.claude/skills          2
claudecode   user     ✓         ~/.config/claude/skills                 5
```

#### scope info

Show detailed information about a specific scope.

```bash
skillsync scope info <platform>:<scope>
```

**Examples:**

```bash
# Show cursor user scope info
skillsync scope info cursor:user

# Show claudecode repo scope
skillsync scope info claudecode:repo
```

**Output:**

```
Scope: cursor:user

Path: /Users/username/.cursor/skills
Writable: ✓
Skills: 8
Total Size: 24.5KB

Skills:
  commit-helper.md (1.2KB)
  debug-wizard.md (3.4KB)
  test-runner.md (2.1KB)
  ...

Last Modified: 2026-01-27 14:30:00
```

#### scope validate

Validate scope configuration and permissions.

```bash
skillsync scope validate [flags]
```

**Flags:**

| Flag | Aliases | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `-p` | all | Validate specific platform |

**Examples:**

```bash
# Validate all scopes
skillsync scope validate

# Validate cursor
skillsync scope validate --platform cursor
```

**Output:**

```
Validating scopes...

✓ cursor:repo (/Users/username/dev/project/.cursor/skills)
  - Path exists
  - Writable
  - 3 valid skills

✓ cursor:user (/Users/username/.cursor/skills)
  - Path exists
  - Writable
  - 8 valid skills

✗ claudecode:system (/usr/local/share/claude/skills)
  - Path does not exist
  - Not writable (expected)

Summary: 2/3 scopes valid
```

---

### tui

Launch interactive terminal UI dashboard.

**Usage:**

```bash
skillsync tui
```

The TUI provides a visual, keyboard-driven interface for:

- Browsing skills across platforms
- Viewing skill details
- Performing sync operations
- Managing backups
- Comparing and deduplicating skills

**Keyboard Shortcuts:**

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `←/→` or `h/l` | Switch panels |
| `Enter` | Select/open |
| `/` | Search |
| `s` | Sync wizard |
| `b` | Backups |
| `c` | Compare |
| `d` | Dedupe |
| `?` | Help |
| `q` | Quit |

**Example:**

```bash
skillsync tui
```

**TUI Interface:**

```
┌─ SkillSync Dashboard ──────────────────────────────────────────┐
│ Platforms: cursor (11) · claudecode (7) · codex (5)            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  cursor:user                       claudecode:user              │
│  ┌──────────────────────┐         ┌──────────────────────┐     │
│  │ ● commit-helper      │         │ ● my-skill           │     │
│  │   debug-wizard       │         │   test-runner        │     │
│  │   test-runner        │         │   deploy-script      │     │
│  └──────────────────────┘         └──────────────────────┘     │
│                                                                 │
│  [s] Sync  [c] Compare  [b] Backups  [?] Help  [q] Quit       │
└─────────────────────────────────────────────────────────────────┘
```

---

## Common Patterns

### First-Time Setup

```bash
# 1. Initialize configuration
skillsync config init

# 2. Discover existing skills
skillsync discover

# 3. Check for duplicates
skillsync compare

# 4. Perform first sync
skillsync sync --dry-run cursor claudecode
skillsync sync cursor claudecode
```

### Regular Workflow

```bash
# Daily sync: cursor → claudecode
skillsync sync --strategy=newer cursor claudecode

# Check for new duplicates
skillsync compare --name-threshold 0.8

# Clean up duplicates
skillsync dedupe delete --dry-run
skillsync dedupe delete
```

### Safe Experimentation

```bash
# Always preview first
skillsync sync --dry-run cursor claudecode

# Use interactive mode for conflicts
skillsync sync --interactive cursor claudecode

# Keep backups
skillsync backup list
```

### Scripting and Automation

```bash
# Export for version control
skillsync export --format yaml --output skills.yaml

# Backup before CI/CD
skillsync backup verify latest
skillsync sync --yes --skip-backup cursor claudecode

# JSON output for parsing
skillsync discover --format json | jq '.skills[] | .name'
```

### Multi-Repository Workflow

```bash
# Sync repo skills from multiple projects
cd ~/project1
skillsync sync --scope repo cursor:repo claudecode:repo

cd ~/project2
skillsync sync --scope repo cursor:repo claudecode:repo

# Sync user skills (global)
skillsync sync cursor:user claudecode:user
```

### Troubleshooting

```bash
# Enable debug logging
skillsync --debug sync cursor claudecode

# Verify configuration
skillsync config show

# Validate scopes
skillsync scope validate

# Check backup integrity
skillsync backup verify <backup-id>
```

---

## Quick Reference Table

| Task | Command |
|------|---------|
| **Setup** | |
| Initialize config | `skillsync config init` |
| View config | `skillsync config show` |
| Edit config | `skillsync config edit` |
| **Discovery** | |
| List all skills | `skillsync discover` |
| Platform skills | `skillsync discover -p cursor` |
| Interactive picker | `skillsync discover -i` |
| **Sync** | |
| Preview sync | `skillsync sync --dry-run cursor claudecode` |
| Sync platforms | `skillsync sync cursor claudecode` |
| Sync with strategy | `skillsync sync -s newer cursor claudecode` |
| Interactive sync | `skillsync sync -i cursor claudecode` |
| **Duplicates** | |
| Find duplicates | `skillsync compare` |
| Strict matching | `skillsync compare -n 0.9 -c 0.95` |
| Preview deletion | `skillsync dedupe delete --dry-run` |
| Delete duplicates | `skillsync dedupe delete` |
| **Export/Backup** | |
| Export JSON | `skillsync export -f json -o skills.json` |
| Create backup | `skillsync export -f tar -o backup.tar.gz` |
| List backups | `skillsync backup list` |
| Restore backup | `skillsync backup restore <id>` |
| **Scope** | |
| List scopes | `skillsync scope list` |
| Promote skill | `skillsync promote my-skill` |
| Demote skill | `skillsync demote my-skill` |
| **TUI** | |
| Launch dashboard | `skillsync tui` |
| **Info** | |
| Version | `skillsync version` |
| Command help | `skillsync <command> --help` |

---

## See Also

- [Quick Start Guide](quick-start.md) - Get started with SkillSync
- [Compatibility Matrix](compatibility.md) - Version and platform compatibility
- [Sync Strategies](sync-strategies.md) - Detailed conflict resolution
- [Skill Formats](skill-formats-research.md) - Platform format specifications
- [Development Guide](../AGENTS.md) - Contributing to SkillSync

For questions or issues, visit: https://github.com/yourusername/skillsync/issues
