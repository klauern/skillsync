# Sync Strategies

Sync strategies control how conflicts are handled when skills already exist in
the target platform.

Use `--strategy` on the `sync` command:

```bash
skillsync sync --strategy=skip cursor claudecode
```

## Available Strategies

### overwrite

Replace target skills with source skills unconditionally (default).

### skip

Skip skills that already exist in the target.

### newer

Copy a skill only if the source is newer than the target.

### merge

Merge source and target content (simple concatenation with headers).

### three-way

Perform a three-way merge with conflict detection when possible.

### interactive

Prompt for each conflict, allowing manual resolution in the TUI.

## Claude Directory Linking

When the source is Claude Code and the skill is a directory-based `SKILL.md`
skill, SkillSync links the source directory into Codex, Cursor, and Pi
targets instead of copying the directory contents.

- `overwrite` replaces the target entry with the symlink
- `skip` leaves the existing target entry alone
- `newer` relinks only when the source is newer than the target
- `merge` and other merge-oriented strategies do not attempt content merges
  for linked directory skills

## Examples

```bash
skillsync sync cursor claudecode --dry-run
skillsync sync cursor:repo claudecode:user --strategy=skip
skillsync sync cursor codex --strategy=three-way
skillsync sync cursor codex --strategy=interactive
```
