# Sync Strategies and Conflict Resolution

This guide explains how skillsync handles synchronization between platforms, detects conflicts, and resolves differences between skill versions.

## Table of Contents

- [Overview](#overview)
- [Sync Direction: Unidirectional](#sync-direction-unidirectional)
- [Sync Strategies](#sync-strategies)
- [Conflict Detection](#conflict-detection)
- [Merge Algorithms](#merge-algorithms)
- [Conflict Resolution](#conflict-resolution)
- [Handling Renames and Moves](#handling-renames-and-moves)
- [Common Scenarios](#common-scenarios)

---

## Overview

Skillsync synchronizes AI coding assistant skills across platforms (Claude Code, Cursor, Codex). Understanding how sync works helps you:

- Choose the right strategy for your workflow
- Predict what will happen during sync
- Resolve conflicts when they occur
- Maintain consistency across platforms

**Key Concepts:**
- **Source**: The platform you're syncing FROM
- **Target**: The platform you're syncing TO
- **Strategy**: The rule that determines what happens when skills differ
- **Conflict**: When the same skill exists with different content on both platforms

---

## Sync Direction: Unidirectional

**Skillsync uses strictly unidirectional sync**: source → target.

```
┌─────────┐           ┌─────────┐
│ Source  │  ───────> │ Target  │
│ (read)  │           │ (write) │
└─────────┘           └─────────┘
```

**What this means:**

✅ **Skills flow one way**: Changes are copied from source to target
✅ **Target is modified**: Existing skills in the target may be updated or replaced
✅ **Source is never touched**: The source platform remains unchanged
❌ **No bidirectional sync**: Changes in the target don't flow back to source
❌ **No automatic propagation**: You must run separate sync commands for each direction

**Example:**

```bash
# Sync Claude Code → Cursor
skillsync sync claude-code cursor

# To sync back, you must explicitly reverse:
skillsync sync cursor claude-code
```

**Implications:**

- **Choose your source carefully**: The source is your "source of truth" for that sync operation
- **Conflicts are target-side**: Only target skills can conflict with incoming source skills
- **Multiple sources need separate syncs**: To consolidate from multiple platforms, run multiple sync commands

---

## Sync Strategies

A **strategy** determines what happens when a skill exists in both source and target. Choose based on your workflow needs.

### Available Strategies

| Strategy | Behavior | Best For |
|----------|----------|----------|
| `overwrite` | Replace target skill unconditionally | **Default**. One-way sync where source is always correct |
| `skip` | Keep target skill, don't update | Preserving local modifications in target |
| `newer` | Update only if source is newer (by timestamp) | Time-based precedence |
| `merge` | Concatenate source and target content | Combining different content from both sides |
| `three-way` | Intelligent merge with base version | Collaborative editing with shared base |
| `interactive` | Prompt user for each conflict | Full manual control |

### Strategy Details

#### 1. `overwrite` (Default)

**Behavior**: Replace target skill with source version, no questions asked.

```bash
skillsync sync claude-code cursor --strategy overwrite
```

**When to use:**
- Source is your canonical repository
- You want target to mirror source exactly
- You don't care about preserving target modifications

**Example:**
```
Source: "# My Skill\nVersion 2 content"
Target: "# My Skill\nVersion 1 content"
Result: Target becomes "# My Skill\nVersion 2 content"
```

---

#### 2. `skip`

**Behavior**: If skill exists in target, leave it alone. Only create new skills.

```bash
skillsync sync claude-code cursor --strategy skip
```

**When to use:**
- You've made local edits in target you want to preserve
- You only want to add new skills, not update existing ones
- You're cautiously testing sync

**Example:**
```
Source: "# My Skill\nUpdated content"
Target: "# My Skill\nLocal modifications"
Result: Target unchanged (skill skipped)
```

---

#### 3. `newer`

**Behavior**: Update target only if source modification time is more recent.

```bash
skillsync sync claude-code cursor --strategy newer
```

**When to use:**
- You edit skills on multiple platforms and want the latest version to win
- Time-based precedence makes sense for your workflow

**How it works:**
1. Compares source skill's `mtime` (modification time) with target skill's `mtime`
2. If `source.mtime > target.mtime`, replaces target
3. Otherwise, skips update

**Example:**
```
Source: "# My Skill\nContent A" (modified 2026-01-28 10:00)
Target: "# My Skill\nContent B" (modified 2026-01-28 09:00)
Result: Target updated to "Content A" (source is newer)
```

**⚠️ Limitation**: Relies on filesystem modification times, which can be unreliable if:
- Files are copied (preserving old timestamps)
- System clock is incorrect
- Files are generated/restored from backups

---

#### 4. `merge`

**Behavior**: Concatenate source and target content with a separator.

```bash
skillsync sync claude-code cursor --strategy merge
```

**When to use:**
- Both versions have valuable content
- You want to combine them and manually clean up later
- Quick-and-dirty content aggregation

**How it works:**
```
Result = Source Content + Separator + Target Content
```

**Example:**
```
Source: "# My Skill\nSource instructions"
Target: "# My Skill\nTarget instructions"

Result:
# My Skill
Source instructions

=== MERGED FROM TARGET ===

# My Skill
Target instructions
```

**⚠️ Note**: This creates duplicate content (both headers, both bodies). You'll need to manually clean up the result.

---

#### 5. `three-way`

**Behavior**: Intelligent merge comparing source and target against a common base version.

```bash
skillsync sync claude-code cursor --strategy three-way
```

**When to use:**
- Multiple people/platforms are editing the same skills
- You want smart conflict detection (only conflicts when both sides changed the same section)
- You can provide or compute a base version

**How it works:**

```
        Base
       /    \
      /      \
  Source   Target
      \      /
       \    /
      Merged
```

1. Compares source to base → identifies source changes
2. Compares target to base → identifies target changes
3. Applies non-overlapping changes from both
4. Marks overlapping changes as conflicts

**Example:**

```
Base:   Line 1
        Line 2
        Line 3

Source: Line 1 (modified)
        Line 2
        Line 3

Target: Line 1
        Line 2 (modified)
        Line 3

Result: Line 1 (modified)      ← Source change applied
        Line 2 (modified)      ← Target change applied (no conflict!)
        Line 3
```

**With conflict:**

```
Base:   Line 1
        Line 2

Source: Line 1 (modified A)
        Line 2

Target: Line 1 (modified B)
        Line 2

Result:
<<<<<<< SOURCE
Line 1 (modified A)
=======
Line 1 (modified B)
>>>>>>> TARGET
Line 2
```

**⚠️ Limitation**: Requires a base version. Skillsync doesn't currently track historical versions, so you may need to manually specify or compute the base.

---

#### 6. `interactive`

**Behavior**: Prompt you for each conflict and let you choose how to resolve it.

```bash
skillsync sync claude-code cursor --strategy interactive
```

**When to use:**
- You want full control over every difference
- Conflicts are rare and worth manual attention
- You're syncing important changes and want to review

**How it works:**

For each skill with differences, you'll see:

```
=== Conflict Resolution ===
Found 1 conflict(s) that require resolution.

--- Conflict 1 of 1: my-skill ---
Type: content
Changes: 2 hunk(s), +5/-3 lines

Preview of changes:
--------------------------------------------------
@@ -1,3 +1,5 @@
 # My Skill
-Old instruction
+New instruction
+Additional line
--------------------------------------------------

How would you like to resolve this conflict?
  1. Use source version (overwrite target)
  2. Keep target version (discard source changes)
  3. Attempt automatic merge (may have conflict markers)
  4. Skip this skill
  5. Show full source content
  6. Show full target content

Enter choice [1-6]:
```

**Resolution options:**
- **1. Use source**: Replace target with source (like `overwrite`)
- **2. Keep target**: Leave target unchanged (like `skip`)
- **3. Merge**: Apply two-way merge algorithm (may insert conflict markers)
- **4. Skip**: Don't sync this skill at all
- **5/6. Show content**: View full content before deciding

---

## Conflict Detection

Skillsync detects conflicts by comparing **content** and **metadata** between source and target versions of the same skill.

### What Constitutes a Conflict?

A conflict occurs when a skill exists in both source and target, but they differ in:

1. **Content**: The body of the skill (markdown text) is different
2. **Metadata**: Frontmatter fields differ:
   - `description` field
   - `tools` list
   - Other metadata key-value pairs

### Conflict Types

```go
ConflictTypeContent    // Only content differs
ConflictTypeMetadata   // Only metadata differs
ConflictTypeBoth       // Both content and metadata differ
```

**Example: Content Conflict**

```yaml
# Source
---
name: my-skill
description: A helpful skill
---
New instructions here
```

```yaml
# Target
---
name: my-skill
description: A helpful skill
---
Old instructions here
```

**Result**: `ConflictTypeContent` (same metadata, different content)

---

**Example: Metadata Conflict**

```yaml
# Source
---
name: my-skill
description: Updated description
tools: [Read, Write]
---
Same content
```

```yaml
# Target
---
name: my-skill
description: Old description
tools: [Read]
---
Same content
```

**Result**: `ConflictTypeMetadata` (same content, different metadata)

---

### How Detection Works

**Algorithm** (implemented in `internal/sync/conflict.go`):

1. **Compare content**: Byte-by-byte comparison of skill body
2. **Compare metadata fields**:
   - Description string
   - Tools array (order-sensitive)
   - Metadata map (key-value pairs)
3. **Classify conflict type**: Based on what differs
4. **Compute diff hunks**: If content differs, identify specific changed sections

### Diff Computation

Skillsync uses the **Longest Common Subsequence (LCS)** algorithm to compute detailed diffs:

**Process:**
1. Split content into lines
2. Find LCS between source and target lines
3. Identify hunks (contiguous blocks of changes)
4. Mark each line as:
   - ` ` (context - unchanged)
   - `+` (added in target)
   - `-` (removed from source)

**Example Output:**

```
@@ -1,5 +1,6 @@
 # My Skill
-Old line 1
-Old line 2
+New line 1
+New line 2
+New line 3
 Context line
```

This format is similar to `git diff` and shows:
- `@@` header: Line numbers in source and target
- `-` lines: Present in source, removed in merge
- `+` lines: Added in target or new in merge
- ` ` lines: Context (unchanged)

---

## Merge Algorithms

Skillsync implements two merge algorithms for combining conflicting versions.

### Two-Way Merge

**Used by**: `merge` and `interactive` (option 3) strategies

**How it works:**

1. Find LCS between source and target
2. Identify sections where both versions differ
3. If both changed the same section, mark as conflict
4. Insert conflict markers for manual resolution

**Algorithm:**

```
for each section of content:
  if section unchanged:
    include in result
  else if only source changed:
    use source version
  else if only target changed:
    use target version
  else: # both changed
    insert conflict markers
```

**Conflict Markers:**

```
<<<<<<< SOURCE
Content from source version
=======
Content from target version
>>>>>>> TARGET
```

**Example:**

```
Source:
Line 1
Source modification
Line 3

Target:
Line 1
Target modification
Line 3

Result:
Line 1
<<<<<<< SOURCE
Source modification
=======
Target modification
>>>>>>> TARGET
Line 3
```

**When to use:**
- You don't have a base version
- You want to see both versions side-by-side
- You'll manually resolve conflicts afterward

---

### Three-Way Merge

**Used by**: `three-way` strategy

**How it works:**

1. Compare source to base → find source changes
2. Compare target to base → find target changes
3. Apply non-conflicting changes from both
4. Mark conflicts only where both modified the same section

**Algorithm:**

```
sourceChanges = diff(base, source)
targetChanges = diff(base, target)

for each line in base:
  if both source and target changed this line:
    insert conflict markers
  else if source changed:
    use source version
  else if target changed:
    use target version
  else:
    use base (unchanged)
```

**Key Advantage**: Non-overlapping changes are merged cleanly without conflicts.

**Example:**

```
Base:
Line 1
Line 2
Line 3

Source (changed Line 1):
Line 1 MODIFIED
Line 2
Line 3

Target (changed Line 3):
Line 1
Line 2
Line 3 MODIFIED

Three-Way Merge Result:
Line 1 MODIFIED      ← Source change applied
Line 2
Line 3 MODIFIED      ← Target change applied (no conflict!)
```

**With Conflict:**

```
Base:
Line 1

Source:
Line 1 MODIFIED A

Target:
Line 1 MODIFIED B

Three-Way Merge Result:
<<<<<<< SOURCE
Line 1 MODIFIED A
=======
Line 1 MODIFIED B
>>>>>>> TARGET
```

**When to use:**
- You have a base version (common ancestor)
- Multiple contributors/platforms editing
- You want smart conflict detection

---

## Conflict Resolution

When conflicts are detected, you have several ways to resolve them depending on the strategy.

### Automatic Resolution Strategies

Some strategies resolve conflicts automatically without user input:

| Strategy | Resolution Method |
|----------|-------------------|
| `overwrite` | Always use source (no conflict reported) |
| `skip` | Always keep target (no conflict reported) |
| `newer` | Use whichever has newer timestamp |
| `merge` | Concatenate both versions |
| `three-way` | Merge intelligently, insert markers if conflict |

### Manual Resolution: Interactive Strategy

The `interactive` strategy gives you full control. For each conflict, choose:

**1. Use source version**
- Target is replaced with source
- Any target modifications are lost
- Equivalent to `overwrite` for this skill

**2. Keep target version**
- Target remains unchanged
- Source changes are discarded
- Equivalent to `skip` for this skill

**3. Attempt automatic merge**
- Runs two-way merge algorithm
- May insert conflict markers if both versions changed the same section
- You'll need to manually edit the result afterward

**4. Skip this skill**
- Skill is not synced at all
- Both source and target remain unchanged
- Useful if you want to handle this skill separately

**5/6. Show full content**
- View complete source or target content
- Helps make informed decision
- Returns to choice menu afterward

### Manual Resolution: TUI Dashboard

Skillsync provides a terminal UI for resolving conflicts visually:

```bash
skillsync dashboard conflicts
```

**Features:**
- Table view of all conflicts across platforms
- Side-by-side diff viewer
- Real-time resolution tracking
- Keyboard shortcuts for quick resolution

**Key Bindings:**
- `s` or `1`: Use source version
- `t` or `2`: Use target version
- `m` or `3`: Attempt merge
- `x` or `4`: Skip
- `y`: Apply all resolutions
- `?`: Toggle help

---

## Handling Renames and Moves

**⚠️ Important Limitation**: Skillsync **does not currently support rename or move detection**.

### What This Means

Skills are matched by **exact name comparison**. If you rename a skill:

```
Before: my-skill.md
After:  renamed-skill.md
```

**What happens during sync:**
- `renamed-skill.md` is treated as a **new skill** (created in target)
- `my-skill.md` remains in target (orphaned)
- No connection is made between the old and new names

### Workarounds

**Option 1: Manual cleanup**
```bash
# Sync with new name
skillsync sync source target

# Manually delete old skill from target
rm ~/.cursor/skills/my-skill.md
```

**Option 2: Rename on both sides before sync**
```bash
# Rename in source
mv ~/.claude/skills/my-skill.md ~/.claude/skills/renamed-skill.md

# Rename in target
mv ~/.cursor/skills/my-skill.md ~/.cursor/skills/renamed-skill.md

# Now sync works as update
skillsync sync claude-code cursor
```

**Option 3: Use `overwrite` strategy and clean up**
```bash
# Sync creates new skill
skillsync sync source target --strategy overwrite

# Use discover to find orphaned skills
skillsync discover target

# Delete orphaned skills manually
```

### Future Enhancement

Rename detection could be added using:
- Content similarity scoring
- Metadata tracking (UUID or hash)
- User-configurable rename mappings

---

## Common Scenarios

### Scenario 1: One-Way Mirror (Personal → Work)

**Goal**: Keep work machine in sync with personal repository.

```bash
# On work machine, sync from personal
skillsync sync claude-code:user cursor:repo --strategy overwrite
```

**Why `overwrite`**: Personal machine is source of truth. Work machine should mirror exactly.

---

### Scenario 2: Consolidate Multiple Platforms

**Goal**: Merge skills from Claude Code and Cursor into Codex.

```bash
# Copy from Claude Code
skillsync sync claude-code codex --strategy merge

# Copy from Cursor
skillsync sync cursor codex --strategy merge
```

**Why `merge`**: Combine content from both sources. Clean up duplicates manually afterward.

---

### Scenario 3: Collaborative Editing

**Goal**: You and a teammate both edit skills. Want to merge changes intelligently.

```bash
# Assuming you have a base version
skillsync sync claude-code cursor --strategy three-way
```

**Why `three-way`**: Detects non-conflicting concurrent changes and merges them automatically.

---

### Scenario 4: Cautious Initial Sync

**Goal**: Try syncing for the first time without overwriting existing skills.

```bash
# Only add new skills, don't touch existing ones
skillsync sync source target --strategy skip --dry-run

# Review what would happen
# Then actually sync
skillsync sync source target --strategy skip
```

**Why `skip`**: Preserves any existing skills in target. Use `--dry-run` to preview.

---

### Scenario 5: Interactive Review of Changes

**Goal**: Review each difference and decide manually.

```bash
skillsync sync source target --strategy interactive
```

**When to use**: Important changes, rare conflicts, want full control.

---

### Scenario 6: Time-Based Sync

**Goal**: Always keep the most recently edited version.

```bash
# Regular sync with time-based precedence
skillsync sync claude-code cursor --strategy newer
```

**⚠️ Caution**: Relies on file modification times, which can be unreliable.

---

## Best Practices

### 1. **Use Dry Run First**

Always preview changes before syncing:

```bash
skillsync sync source target --strategy overwrite --dry-run
```

### 2. **Backup Before Major Syncs**

Backups are automatic by default:

```bash
# Backup is created automatically
skillsync sync source target

# Skip backup if you're confident
skillsync sync source target --skip-backup
```

### 3. **Choose Strategy Based on Workflow**

- **Personal use, one platform is canonical**: `overwrite`
- **Multiple platforms, concurrent edits**: `three-way` or `interactive`
- **First-time sync**: `skip` or `interactive` to be cautious
- **Consolidating from multiple sources**: `merge`

### 4. **Use Scope Filtering**

Sync only what you need:

```bash
# Sync only repo-level skills
skillsync sync claude-code:repo cursor:repo

# Sync both repo and user levels from source
skillsync sync claude-code:repo,user cursor:repo
```

### 5. **Handle Renames Carefully**

Since renames aren't detected:
- Rename on both sides before syncing, OR
- Sync, then manually clean up orphaned skills

### 6. **Review Conflicts After Merge**

If using `merge` or `three-way`, check for conflict markers:

```bash
# After sync
grep -r "<<<<<<< SOURCE" ~/.cursor/skills/
```

Edit files with markers and remove them once resolved.

---

## Troubleshooting

### "Conflict detected but I want to force update"

**Solution**: Use `overwrite` strategy.

```bash
skillsync sync source target --strategy overwrite
```

---

### "I synced but old skills remain in target"

**Cause**: Skillsync doesn't delete skills that don't exist in source.

**Solution**: Use `discover` to find orphaned skills and delete manually.

```bash
skillsync discover target
# Review list, then delete unwanted skills manually
```

---

### "Three-way merge not working"

**Cause**: Three-way merge requires a base version, which skillsync doesn't currently track automatically.

**Solution**: Use `interactive` or `two-way merge` instead, or manually provide base version if using as library.

---

### "Timestamp-based sync (`newer`) behaving unexpectedly"

**Cause**: File modification times can be unreliable (copied files, backups, etc.).

**Solution**: Use a content-based strategy like `interactive` or `three-way` instead.

---

## Next Steps

- **[Commands Reference](commands.md)**: Detailed command documentation
- **[Quick Start Guide](quick-start.md)**: Get started with skillsync
- **[Migration Guide](migration.md)**: Migrate existing skills to skillsync

---

## Summary

- **Sync is unidirectional**: Source → Target only
- **Six strategies**: Choose based on your workflow needs
- **Conflicts are detected** by comparing content and metadata
- **Two merge algorithms**: Two-way and three-way merge with conflict markers
- **Interactive resolution**: Full control when needed
- **Renames not detected**: Manual workarounds required

For most users, start with `overwrite` for simple mirroring or `interactive` for full control. As you become comfortable, explore strategies like `three-way` for advanced collaborative workflows.
