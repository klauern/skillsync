# Migration Guide

This guide helps you transition from existing Claude Code, Cursor, or Codex skills to a unified SkillSync workflow.

## Table of Contents

- [Overview](#overview)
- [Who This Guide Is For](#who-this-guide-is-for)
- [Migration Benefits](#migration-benefits)
- [Pre-Migration Checklist](#pre-migration-checklist)
- [Understanding Platform Differences](#understanding-platform-differences)
- [Step-by-Step Migration](#step-by-step-migration)
  - [Phase 1: Assessment](#phase-1-assessment)
  - [Phase 2: Backup](#phase-2-backup)
  - [Phase 3: Initial Import](#phase-3-initial-import)
  - [Phase 4: Deduplication](#phase-4-deduplication)
  - [Phase 5: Validation](#phase-5-validation)
  - [Phase 6: Ongoing Sync](#phase-6-ongoing-sync)
- [Platform-Specific Guidance](#platform-specific-guidance)
  - [Migrating from Claude Code](#migrating-from-claude-code)
  - [Migrating from Cursor](#migrating-from-cursor)
  - [Migrating from Codex](#migrating-from-codex)
- [Common Migration Scenarios](#common-migration-scenarios)
- [Conflict Resolution During Migration](#conflict-resolution-during-migration)
- [Post-Migration Best Practices](#post-migration-best-practices)
- [Troubleshooting](#troubleshooting)
- [Rollback Procedures](#rollback-procedures)

---

## Overview

SkillSync enables you to manage skills across Claude Code, Cursor, and Codex from a single unified repository. Migration involves:

1. **Discovering** existing skills across all platforms
2. **Backing up** your current skills
3. **Importing** skills into SkillSync's management system
4. **Deduplicating** similar skills across platforms
5. **Synchronizing** skills across platforms going forward

**Migration is non-destructive.** SkillSync creates backups before making changes, and you can roll back at any time.

---

## Who This Guide Is For

This guide is for users who:

- Already have skills configured in one or more AI coding platforms
- Want to consolidate and synchronize skills across platforms
- Need to eliminate duplicate skills across Claude Code, Cursor, and Codex
- Want a unified workflow for managing all AI coding skills

**New users?** If you're starting fresh without existing skills, see the [Quick Start Guide](quick-start.md) instead.

---

## Migration Benefits

**Before SkillSync:**
```
~/.claude/skills/python-expert.md
~/.cursor/skills/python-best-practices.md
~/.codex/skills/SKILL.md
```
↓ Three separate, potentially duplicate skills to maintain

**After SkillSync:**
```
~/.skillsync/skills/python-expert/SKILL.md
```
↓ One authoritative source, synchronized across all platforms

**Key benefits:**
- **Single source of truth** - Edit once, sync everywhere
- **No duplication** - Detect and merge similar skills
- **Cross-platform** - Use the same skills in Claude Code, Cursor, and Codex
- **Version control** - Track skill changes over time
- **Easy migration** - Move skills between scopes (user ↔ repo)

---

## Pre-Migration Checklist

Before starting migration, ensure:

- [ ] SkillSync is installed and working (`skillsync version`)
- [ ] You have backups of important skills (manual or via platform-specific backup)
- [ ] You understand which platforms you use (Claude Code, Cursor, Codex)
- [ ] You have write access to skill directories on each platform
- [ ] You're comfortable using command-line tools

**Estimated time:** 15-30 minutes depending on the number of skills

---

## Understanding Platform Differences

### Skill Storage Locations

| Platform | Global Skills | Project Skills |
|----------|--------------|----------------|
| **Claude Code** | `~/.claude/skills/` | `.claude/skills/` |
| **Cursor** | `~/.cursor/skills/` | `.cursor/skills/` |
| **Codex** | `~/.codex/skills/` | `.codex/skills/` |

### File Format Differences

| Platform | Format | Extensions | Notes |
|----------|--------|------------|-------|
| **Claude Code** | Markdown + YAML frontmatter | `.md` | Uses `tools` array for permissions |
| **Cursor** | Markdown + YAML frontmatter | `.md`, `.mdc` | Uses `globs` for file pattern matching |
| **Codex** | TOML config + Markdown | `config.toml`, `AGENTS.md` | Separate config and instruction files |

**Agent Skills Standard (SKILL.md):**
All platforms now support the unified `SKILL.md` format. SkillSync prefers this format and can convert legacy formats automatically.

### Metadata Mapping

When SkillSync imports skills, it normalizes platform-specific metadata:

| Concept | Claude Code | Cursor | Codex |
|---------|-------------|--------|-------|
| **Skill name** | `name` in frontmatter | Filename or `name` | `name` in frontmatter |
| **Permissions** | `tools` array | `globs` + `alwaysApply` | `approval_policy` |
| **Scope** | `scope` field | Implied by location | Implied by location |
| **Content** | Markdown body | Markdown body | `instructions` in TOML |

**Platform-specific fields are preserved** in skill metadata for round-trip compatibility.

---

## Step-by-Step Migration

### Phase 1: Assessment

**Discover what skills you have across all platforms:**

```bash
skillsync discover
```

**Expected output:**
```
Discovered 12 skills across 3 platforms:

Claude Code (5 skills):
  python-expert        user    ~/.claude/skills/python-expert.md
  git-helper           repo    .claude/skills/git-helper.md
  ...

Cursor (4 skills):
  python-linter        user    ~/.cursor/skills/python-linter.md
  react-patterns       user    ~/.cursor/skills/react-patterns.md
  ...

Codex (3 skills):
  python-style         user    ~/.codex/skills/python-style/SKILL.md
  ...
```

**What to look for:**
- **Duplicates**: Similar skill names across platforms (e.g., `python-expert`, `python-linter`, `python-style`)
- **Scope differences**: Some skills in user scope, others in repo scope
- **Name conflicts**: Identical names that might mean different things

**Filter by platform:**
```bash
# View only Claude Code skills
skillsync discover --platform claudecode

# View only user-scoped skills
skillsync discover --scope user
```

**Export discovery results:**
```bash
# Save full skill list to JSON
skillsync discover --format json > skills-inventory.json

# Review in your editor
cat skills-inventory.json | jq '.[] | {name, platform, path}'
```

---

### Phase 2: Backup

**Always create backups before migration:**

```bash
# Create comprehensive backup
skillsync backup create --name "pre-migration-$(date +%Y%m%d)"
```

**Expected output:**
```
✓ Created backup: pre-migration-20260128
  Location: ~/.skillsync/backups/pre-migration-20260128/
  Platforms: claude-code, cursor, codex
  Skills: 12 total
```

**List existing backups:**
```bash
skillsync backup list
```

**Verify backup contents:**
```bash
skillsync backup show pre-migration-20260128
```

**Manual backup (optional redundancy):**
```bash
# Platform-specific manual backups
cp -r ~/.claude/skills ~/.claude/skills.backup
cp -r ~/.cursor/skills ~/.cursor/skills.backup
cp -r ~/.codex/skills ~/.codex/skills.backup
```

---

### Phase 3: Initial Import

**Use the interactive TUI for the easiest import experience:**

```bash
skillsync tui
```

**In the TUI:**
1. Navigate to **"Import"** option (press `i` or arrow keys + Enter)
2. **Phase 1 - Select Source**: Browse to your skills directory
   - For Claude Code: Select `~/.claude/skills/`
   - For Cursor: Select `~/.cursor/skills/`
   - For Codex: Select `~/.codex/skills/`
3. **Phase 2 - Select Skills**: Check which skills to import
   - Use `Space` to select/deselect individual skills
   - Use `a` to select all, `n` to deselect all
   - Use `/` to filter by name
4. **Phase 3 - Choose Destination**: Select target platform and scope
   - Platform: Choose where to sync (claudecode, cursor, codex, or all)
   - Scope: Choose `user` (global) or `repo` (project-specific)
5. **Phase 4 - Confirm**: Review and execute import
   - Press Enter to proceed
   - Press Esc to cancel

**Command-line import alternative:**

If you prefer CLI over TUI, you can use the sync command directly:

```bash
# Import from Claude Code to user scope
skillsync sync claudecode --scope user --dry-run

# Review what would happen, then execute
skillsync sync claudecode --scope user
```

**What happens during import:**
- SkillSync parses skills in their native format (`.md`, `.toml`, etc.)
- Skills are normalized to the Agent Skills Standard (SKILL.md format)
- Platform-specific metadata is preserved
- Existing skills with the same name are detected (see conflict resolution below)

---

### Phase 4: Deduplication

**After importing, find duplicate or similar skills:**

```bash
# Detect exact and similar duplicates
skillsync dedupe --similarity 0.8
```

**Expected output:**
```
Found 3 duplicate groups:

Group 1 (exact match):
  ✓ python-expert (claude-code, user)
  ✓ python-expert (cursor, user)

Group 2 (85% similar):
  ≈ python-linter (cursor, user)
  ≈ python-style (codex, user)
  Suggested name: python-style-guide

Group 3 (92% similar):
  ≈ react-patterns (cursor, user)
  ≈ react-helper (claude-code, repo)
```

**Interactive deduplication:**

```bash
# Launch interactive dedupe mode
skillsync dedupe --interactive
```

**In interactive mode:**
- Review each duplicate group
- Choose which version to keep (based on content quality, metadata, etc.)
- Merge or rename skills
- Press `Enter` to accept suggestion, `e` to edit, `s` to skip

**Manual deduplication:**

For precise control, manually compare and merge:

```bash
# Compare two specific skills side-by-side
skillsync compare claude-code:python-expert cursor:python-linter

# View diff of skill content
skillsync compare claude-code:python-expert cursor:python-linter --diff
```

After reviewing, choose a deduplication strategy:

1. **Keep best version** - Delete inferior copies
2. **Merge content** - Combine unique instructions from both
3. **Rename variations** - Keep both but clarify their purposes

---

### Phase 5: Validation

**Verify migration was successful:**

```bash
# Rediscover skills to see current state
skillsync discover
```

**Check for issues:**
```bash
# Validate all skills can be parsed correctly
skillsync discover --validate

# Check for orphaned or invalid skills
skillsync discover --check-health
```

**Test a sync to ensure skills work:**

```bash
# Dry-run sync to see what would happen
skillsync sync cursor claudecode --dry-run

# If output looks good, run actual sync
skillsync sync cursor claudecode
```

**Verify in each platform:**

1. **Claude Code**: Launch Claude Code and check skills appear in skill selector
2. **Cursor**: Open Cursor and verify skills in `.cursor/skills/`
3. **Codex**: Run `codex` and verify skills are available

**Expected results:**
- All skills discovered by SkillSync are accessible in target platforms
- No duplicate skills remain (unless intentional)
- Skill content is identical across platforms

---

### Phase 6: Ongoing Sync

**Now that migration is complete, use SkillSync for ongoing management:**

**Typical workflow:**

1. **Edit skills** in your preferred platform or directly in SkillSync format
2. **Sync changes** across platforms
3. **Commit to version control** (if using repo-scoped skills)

**Sync commands:**

```bash
# Sync from Claude Code to Cursor
skillsync sync claudecode cursor

# Sync to all platforms from Claude Code
skillsync sync claudecode cursor codex

# Sync with conflict detection
skillsync sync claudecode cursor --strategy prompt
```

**Use TUI for visual workflow:**

```bash
skillsync tui
```

Navigate to:
- **Dashboard** - Quick overview of skills across platforms
- **Sync** - Interactive sync with conflict resolution
- **Compare** - Side-by-side skill comparison

---

## Platform-Specific Guidance

### Migrating from Claude Code

**Typical Claude Code skill structure:**

```markdown
---
name: python-expert
description: Python coding best practices
tools: ["Read", "Write", "Bash"]
scope: user
---

# Python Expert

Always follow PEP 8 style guidelines...
```

**Migration notes:**

- **Tools array**: The `tools` field is Claude Code-specific. It's preserved in metadata but not used by other platforms.
- **Scope**: Claude Code uses explicit `scope` in frontmatter. This maps directly to SkillSync scopes.
- **Format**: Already uses YAML frontmatter, so conversion is minimal.

**Migration steps:**

1. **Backup**: `cp -r ~/.claude/skills ~/.claude/skills.backup`
2. **Import via TUI**: `skillsync tui` → Import → Select `~/.claude/skills/`
3. **Choose destination**: Select target platforms (cursor, codex, or both)
4. **Validate**: Check that `tools` field is preserved in metadata

**What gets converted:**

```markdown
Before (Claude Code):
~/.claude/skills/python-expert.md

After (SkillSync):
~/.skillsync/skills/python-expert/SKILL.md
  + metadata: tools=[Read, Write, Bash]
```

**Syncing back to Claude Code:**

```bash
# Sync from SkillSync to Claude Code
skillsync sync cursor claudecode --scope user
```

This recreates the `.md` file with original metadata intact.

---

### Migrating from Cursor

**Typical Cursor skill structure:**

```markdown
---
globs: ["*.py", "**/*.py"]
alwaysApply: false
name: python-linter
---

# Python Linting Rules

Use Ruff for linting. Run checks before committing...
```

**Migration notes:**

- **Globs**: The `globs` field specifies file patterns. This is Cursor-specific and stored in metadata.
- **alwaysApply**: Determines if skill applies to all files. Maps to `scope: user` when true.
- **Extensions**: Cursor supports `.mdc` (Cursor Markdown) files, treated same as `.md`.

**Migration steps:**

1. **Backup**: `cp -r ~/.cursor/skills ~/.cursor/skills.backup`
2. **Import via TUI**: `skillsync tui` → Import → Select `~/.cursor/skills/`
3. **Choose destination**: Select target platforms (claudecode, codex, or both)
4. **Handle globs**: Decide how to translate file pattern rules
   - **Option A**: Convert to general instructions ("For Python files...")
   - **Option B**: Keep in metadata for Cursor-specific behavior

**What gets converted:**

```markdown
Before (Cursor):
~/.cursor/skills/python-linter.md
  globs: ["*.py"]
  alwaysApply: false

After (SkillSync):
~/.skillsync/skills/python-linter/SKILL.md
  + metadata: globs=["*.py"], alwaysApply=false
```

**Syncing back to Cursor:**

```bash
# Sync from SkillSync to Cursor
skillsync sync claudecode cursor --scope user
```

The `globs` metadata is restored when syncing back to Cursor.

---

### Migrating from Codex

**Typical Codex skill structure:**

```
~/.codex/skills/python-style/
├── config.toml
└── AGENTS.md
```

**config.toml:**
```toml
model = "o4-mini"
instructions = "Follow Python PEP 8 style guidelines."
approval_policy = "on-failure"
sandbox_mode = "enabled"
```

**AGENTS.md:**
```markdown
# Python Style Guide

Always use type hints and docstrings...
```

**Migration notes:**

- **TOML config**: Codex uses separate config files for model settings. These are extracted as metadata.
- **AGENTS.md**: Legacy format for instructions. Content is imported into SKILL.md body.
- **Directory structure**: Codex skills can be directories with multiple files. SkillSync preserves this.

**Migration steps:**

1. **Backup**: `cp -r ~/.codex/skills ~/.codex/skills.backup`
2. **Import via TUI**: `skillsync tui` → Import → Select `~/.codex/skills/`
3. **Choose destination**: Select target platforms (claudecode, cursor, or both)
4. **Review metadata**: Check that model preferences and policies are preserved

**What gets converted:**

```
Before (Codex):
~/.codex/skills/python-style/config.toml
~/.codex/skills/python-style/AGENTS.md

After (SkillSync):
~/.skillsync/skills/python-style/SKILL.md
  + metadata: model=o4-mini, approval_policy=on-failure, sandbox_mode=enabled
```

**Syncing back to Codex:**

```bash
# Sync from SkillSync to Codex
skillsync sync claudecode codex --scope user
```

The TOML config is regenerated from metadata when syncing back to Codex.

---

## Common Migration Scenarios

### Scenario 1: Single Platform User Expanding to Multi-Platform

**Situation:** You use Claude Code exclusively but want to start using Cursor.

**Approach:**

1. Import existing Claude Code skills into SkillSync
2. Sync to Cursor
3. Test in Cursor to verify skills work
4. Continue editing in either platform, syncing as needed

**Commands:**

```bash
# Import from Claude Code
skillsync tui  # Select Import → ~/.claude/skills/

# Sync to Cursor
skillsync sync claudecode cursor --scope user

# Verify
skillsync discover --platform cursor
```

---

### Scenario 2: Multi-Platform User with Duplicates

**Situation:** You have similar skills scattered across Claude Code, Cursor, and Codex.

**Approach:**

1. Import all skills from all platforms
2. Run deduplication to find overlaps
3. Merge or consolidate duplicates
4. Sync unified skills back to all platforms

**Commands:**

```bash
# Import from all platforms (do this three times via TUI)
skillsync tui  # Import from ~/.claude/skills/
skillsync tui  # Import from ~/.cursor/skills/
skillsync tui  # Import from ~/.codex/skills/

# Find duplicates
skillsync dedupe --similarity 0.8 --interactive

# Sync back to all platforms
skillsync sync claudecode cursor codex --scope user
```

---

### Scenario 3: Project-Specific Skills

**Situation:** You have skills that should only apply to specific repositories.

**Approach:**

1. Import repo-scoped skills separately from user-scoped skills
2. Use `--scope repo` during import/sync
3. Store skills in version control (`.claude/skills/`, etc.)

**Commands:**

```bash
# Navigate to your project
cd ~/projects/my-app

# Import project-specific skills
skillsync tui  # Import → .claude/skills/ → Scope: repo

# Sync to other platforms in this repo
skillsync sync claudecode cursor --scope repo

# Commit to version control
git add .claude/skills/ .cursor/skills/
git commit -m "Add project-specific AI coding skills"
```

---

### Scenario 4: Transitioning from Manual Skill Management

**Situation:** You manually copy-paste skills between platforms.

**Approach:**

1. Consolidate skills into one platform first (choose your primary)
2. Delete obvious duplicates manually before importing
3. Import from primary platform
4. Sync to secondary platforms
5. Use SkillSync going forward for all skill edits

**Commands:**

```bash
# Consolidate to Claude Code first (manual step)
# Delete duplicates manually

# Import from Claude Code
skillsync tui  # Import → ~/.claude/skills/

# Sync to all platforms
skillsync sync claudecode cursor codex --scope user

# Going forward, edit in any platform and sync
skillsync sync claudecode cursor  # After editing in Claude Code
skillsync sync cursor claudecode  # After editing in Cursor
```

---

## Conflict Resolution During Migration

**Conflicts occur when:**
- A skill exists in both source and destination with different content
- Two platforms have skills with the same name but different purposes
- Manual edits were made after the last sync

**Conflict strategies:**

| Strategy | Behavior | Use When |
|----------|----------|----------|
| `overwrite` | Always overwrite destination with source | Initial migration (destination is outdated) |
| `skip` | Keep destination, don't overwrite | Destination has newer/better content |
| `prompt` | Ask for each conflict | You want manual control |
| `merge` | Attempt automatic merge (experimental) | Skills have complementary content |

**Setting strategy:**

```bash
# Overwrite conflicts (default for initial migration)
skillsync sync claudecode cursor --strategy overwrite

# Prompt for each conflict
skillsync sync claudecode cursor --strategy prompt

# Skip conflicts (keep destination)
skillsync sync claudecode cursor --strategy skip
```

**Interactive conflict resolution:**

When using `--strategy prompt`, you'll see:

```
Conflict: python-expert

Source (claude-code):
  Last modified: 2026-01-15
  Size: 2.3 KB
  Preview: "Always follow PEP 8..."

Destination (cursor):
  Last modified: 2026-01-20
  Size: 1.8 KB
  Preview: "Use Ruff for linting..."

Choose action:
  [o] Overwrite with source
  [k] Keep destination
  [m] Merge content
  [d] View full diff
  [r] Rename source
  [q] Quit sync
```

**Best practices for conflict resolution:**

1. **Use `--dry-run` first** to preview conflicts without making changes
2. **Review diffs** to understand what's different
3. **Merge manually** for important conflicts (edit skill file after sync)
4. **Use timestamps** to decide which version is newer
5. **Keep both** if skills serve different purposes (rename one)

---

## Post-Migration Best Practices

### 1. Establish a Primary Platform

**Choose one platform as your "source of truth"** where you make most edits:

- **Claude Code** - If you prefer structured frontmatter
- **Cursor** - If you use file pattern-based rules
- **SkillSync directly** - Edit `~/.skillsync/skills/` for platform independence

**Example workflow (Claude Code as primary):**

```bash
# Edit skill in Claude Code
code ~/.claude/skills/python-expert.md

# Sync changes to other platforms
skillsync sync claudecode cursor codex
```

---

### 2. Set Up Automatic Syncing

**Use shell aliases for convenience:**

Add to `~/.zshrc` or `~/.bashrc`:

```bash
# Sync after editing skills
alias ss-sync='skillsync sync claudecode cursor codex'

# Quick status check
alias ss-status='skillsync discover'

# Interactive TUI
alias ss='skillsync tui'
```

**Use file watchers (advanced):**

Set up automatic sync when skills change:

```bash
# Using fswatch (macOS)
fswatch -o ~/.claude/skills/ | xargs -n1 -I{} skillsync sync claudecode cursor

# Using inotifywait (Linux)
while inotifywait -r -e modify ~/.claude/skills/; do
  skillsync sync claudecode cursor
done
```

---

### 3. Version Control for Repo Skills

**Store project-specific skills in git:**

```bash
# Initialize project skills
cd ~/projects/my-app
mkdir -p .claude/skills .cursor/skills .codex/skills

# Add to version control
git add .claude/skills/ .cursor/skills/ .codex/skills/
git commit -m "Add project-specific skills"

# Sync when team members pull
git pull
skillsync sync claudecode cursor codex --scope repo
```

**Ignore platform-generated files:**

Add to `.gitignore`:

```
# Platform-specific generated files
.claude/.cache/
.cursor/.cache/
.codex/.cache/
```

---

### 4. Regular Deduplication

**Schedule periodic dedupe checks:**

```bash
# Weekly deduplication check
0 9 * * 1 /usr/local/bin/skillsync dedupe --similarity 0.8 --report ~/dedupe-report.txt
```

Or run manually:

```bash
# Monthly review
skillsync dedupe --similarity 0.8 --interactive
```

---

### 5. Keep Backups

**Create backups before major changes:**

```bash
# Before bulk edits
skillsync backup create --name "before-refactor-$(date +%Y%m%d)"

# Before OS upgrades
skillsync backup create --name "before-macos-upgrade"

# Automatic weekly backups (cron)
0 3 * * 0 /usr/local/bin/skillsync backup create --name "weekly-$(date +%Y%m%d)"
```

---

## Troubleshooting

### Issue: Skills Not Appearing After Import

**Symptoms:**
- SkillSync shows skills in `discover`
- Skills don't appear in Claude Code/Cursor/Codex

**Diagnosis:**

```bash
# Check where skills were imported
skillsync discover --format json | jq '.[] | {name, path}'

# Verify platform skill directories exist
ls -la ~/.claude/skills/
ls -la ~/.cursor/skills/
ls -la ~/.codex/skills/
```

**Solutions:**

1. **Sync explicitly to each platform:**
   ```bash
   skillsync sync claudecode --scope user
   skillsync sync cursor --scope user
   skillsync sync codex --scope user
   ```

2. **Check platform is running:**
   - Restart Claude Code / Cursor / Codex
   - Some platforms cache skill lists

3. **Verify file permissions:**
   ```bash
   chmod -R u+rw ~/.claude/skills/
   chmod -R u+rw ~/.cursor/skills/
   chmod -R u+rw ~/.codex/skills/
   ```

---

### Issue: Duplicate Skills After Migration

**Symptoms:**
- Same skill appears multiple times in `discover`
- Skills exist in multiple scopes or platforms

**Diagnosis:**

```bash
# Find duplicates
skillsync dedupe --similarity 1.0

# Check skill scopes
skillsync discover --format json | jq '.[] | select(.name == "python-expert")'
```

**Solutions:**

1. **Run deduplication:**
   ```bash
   skillsync dedupe --interactive
   ```

2. **Manually remove duplicates:**
   ```bash
   # Delete inferior version
   rm ~/.cursor/skills/python-expert.md

   # Sync to ensure consistency
   skillsync sync claudecode cursor
   ```

---

### Issue: Lost Platform-Specific Metadata

**Symptoms:**
- Cursor `globs` field missing after sync
- Claude Code `tools` array missing
- Codex `approval_policy` missing

**Diagnosis:**

```bash
# Check if metadata is preserved
skillsync discover --format json | jq '.[] | select(.name == "python-expert") | .metadata'
```

**Solutions:**

1. **Restore from backup:**
   ```bash
   skillsync backup restore pre-migration-20260128 --platform cursor --scope user
   ```

2. **Re-import with metadata preservation:**
   ```bash
   skillsync sync claudecode cursor --preserve-metadata
   ```

3. **Manually add metadata back:**
   Edit the SKILL.md file and add frontmatter:
   ```yaml
   ---
   name: python-expert
   globs: ["*.py"]
   alwaysApply: false
   ---
   ```

---

### Issue: Sync Fails with Permission Errors

**Symptoms:**
```
Error: failed to write skill: permission denied
```

**Diagnosis:**

```bash
# Check directory permissions
ls -ld ~/.claude/skills/
ls -ld ~/.cursor/skills/
```

**Solutions:**

1. **Fix permissions:**
   ```bash
   chmod u+w ~/.claude/skills/
   chmod u+w ~/.cursor/skills/
   ```

2. **Check disk space:**
   ```bash
   df -h ~
   ```

3. **Run with sudo (not recommended):**
   ```bash
   sudo skillsync sync claudecode cursor
   ```

---

### Issue: Conflict Resolution Loop

**Symptoms:**
- Sync keeps detecting the same conflict
- Conflict appears even after manual resolution

**Diagnosis:**

```bash
# Compare the conflicting skills
skillsync compare claudecode:python-expert cursor:python-expert --diff

# Check modification times
stat ~/.claude/skills/python-expert.md
stat ~/.cursor/skills/python-expert.md
```

**Solutions:**

1. **Force overwrite one direction:**
   ```bash
   skillsync sync claudecode cursor --strategy overwrite
   ```

2. **Delete destination and re-sync:**
   ```bash
   rm ~/.cursor/skills/python-expert.md
   skillsync sync claudecode cursor
   ```

3. **Use merge strategy:**
   ```bash
   skillsync sync claudecode cursor --strategy merge
   ```

---

### Issue: SKILL.md Format Not Recognized

**Symptoms:**
- Skills in SKILL.md format not discovered
- Parser errors when importing

**Diagnosis:**

```bash
# Validate SKILL.md syntax
cat ~/.claude/skills/my-skill/SKILL.md

# Check for frontmatter issues
head -n 10 ~/.claude/skills/my-skill/SKILL.md
```

**Solutions:**

1. **Ensure frontmatter delimiters are correct:**
   ```markdown
   ---
   name: my-skill
   description: Description here
   ---

   # Skill content starts here
   ```

2. **Use YAML validator:**
   ```bash
   # Extract frontmatter and validate
   sed -n '/^---$/,/^---$/p' ~/.claude/skills/my-skill/SKILL.md | yq eval
   ```

3. **Convert to Agent Skills Standard:**
   ```bash
   skillsync convert ~/.claude/skills/my-skill.md --format agent-skills-standard
   ```

---

## Rollback Procedures

### Complete Rollback

**Restore all skills to pre-migration state:**

```bash
# List available backups
skillsync backup list

# Restore from backup
skillsync backup restore pre-migration-20260128

# Verify restoration
skillsync discover
```

---

### Partial Rollback

**Restore only specific platforms:**

```bash
# Restore only Cursor skills
skillsync backup restore pre-migration-20260128 --platform cursor

# Restore only user-scoped skills
skillsync backup restore pre-migration-20260128 --scope user
```

---

### Manual Rollback

**Use manual backups if SkillSync backups fail:**

```bash
# Restore from manual backups
rm -rf ~/.claude/skills
cp -r ~/.claude/skills.backup ~/.claude/skills

rm -rf ~/.cursor/skills
cp -r ~/.cursor/skills.backup ~/.cursor/skills

rm -rf ~/.codex/skills
cp -r ~/.codex/skills.backup ~/.codex/skills
```

---

### Rollback Verification

**After rollback, verify skills are restored:**

```bash
# Check skills are present
ls -la ~/.claude/skills/
ls -la ~/.cursor/skills/
ls -la ~/.codex/skills/

# Verify in platforms
# - Launch Claude Code and check skill selector
# - Open Cursor and check .cursor/skills/
# - Run codex and verify skills are available
```

---

## Next Steps

**Migration complete!** Here's what to do next:

1. **Read the [Command Reference](commands.md)** to learn all SkillSync capabilities
2. **Review [Sync Strategies](sync-strategies.md)** for advanced conflict handling
3. **Explore the TUI** (`skillsync tui`) for visual skill management
4. **Set up automatic syncing** using shell aliases or file watchers
5. **Share skills with your team** by committing repo-scoped skills to git

**Need help?** See the [Troubleshooting Guide](troubleshooting.md) or open an issue on GitHub.

---

## See Also

- [Quick Start Guide](quick-start.md) - Getting started with SkillSync
- [Command Reference](commands.md) - Complete command documentation
- [Sync Strategies](sync-strategies.md) - Conflict resolution strategies
- [Skill Format Research](skill-formats-research.md) - Technical format specifications
