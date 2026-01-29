# Git Commands for Splitting Commits

## File-Level Staging

### Stage Specific Files

```bash
# Stage individual files
git add src/auth/login.ts src/auth/types.ts

# Stage by pattern
git add src/auth/*.ts

# Stage directory
git add src/auth/
```

### Unstaging Files

```bash
# Unstage specific file (keep changes)
git restore --staged <file>

# Unstage all (keep changes)
git restore --staged .

# Reset staging to start fresh
git reset HEAD
```

## Hunk-Level Staging

### Interactive Add

```bash
git add -p [file]
```

Interactive prompts:
- `y` - stage this hunk
- `n` - do not stage this hunk
- `q` - quit; do not stage this or remaining hunks
- `a` - stage this and all remaining hunks in file
- `d` - do not stage this or any remaining hunks in file
- `s` - split current hunk into smaller hunks
- `e` - manually edit current hunk
- `?` - print help

### Manual Hunk Editing

When using `e` in interactive mode:

1. Lines starting with `-` will be removed
2. Lines starting with `+` will be added
3. Remove `+` lines you don't want staged
4. Change `-` to ` ` (space) for deletions you don't want staged
5. Don't modify context lines (no prefix)

### Track File Without Staging Content

```bash
# Add file to index but don't stage content
git add -N <file>

# Then use -p to stage specific hunks
git add -p <file>
```

## Splitting Already-Committed Changes

### Soft Reset

```bash
# Undo last commit, keep changes staged
git reset --soft HEAD~1

# Undo last commit, keep changes unstaged
git reset HEAD~1
```

### Interactive Rebase

```bash
# Edit last N commits
git rebase -i HEAD~N

# Mark commit to split with 'edit'
# When stopped at commit:
git reset HEAD~1
# Stage and commit in parts
git add <files1>
git commit -m "first part"
git add <files2>
git commit -m "second part"
git rebase --continue
```

## Workflow Patterns

### Clean Split Workflow

```bash
# 1. Ensure clean state
git stash  # if needed

# 2. Review all changes
git diff HEAD --stat

# 3. Stage first group
git add <files-for-commit-1>

# 4. Verify staging
git diff --cached --stat
git diff --stat  # remaining unstaged

# 5. Commit first group
git commit -m "type(scope): description"

# 6. Repeat for remaining groups
git add <files-for-commit-2>
git commit -m "type(scope): description"
```

### Partial File Split Workflow

```bash
# 1. Review file changes
git diff <file>

# 2. Stage specific hunks
git add -p <file>
# Select hunks for first commit

# 3. Verify
git diff --cached <file>  # staged
git diff <file>           # remaining

# 4. Commit staged hunks
git commit -m "type(scope): first change"

# 5. Stage remaining hunks
git add <file>
git commit -m "type(scope): second change"
```

## Verification Commands

```bash
# Show what will be committed
git diff --cached

# Show what remains unstaged
git diff

# Show commit history
git log --oneline -n 10

# Show files in last commit
git show --stat HEAD

# Verify each commit builds (if applicable)
git stash
# run build/tests
git stash pop
```