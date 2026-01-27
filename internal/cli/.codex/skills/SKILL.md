# Worktree Management Skill

Expert git worktree management using the `wt` CLI tool. Enables parallel development on multiple branches without switching.

## When to Use

Invoke this skill when:
- User wants to work on multiple branches simultaneously
- User mentions "worktree", "wt", or parallel branch work
- User starts work on a new ticket/feature and wants isolation
- User needs to review a PR while keeping WIP on another branch
- User asks to create, list, sync, or remove worktrees

## Core Concepts

### Bare Repository Pattern
The `wt` tool uses a bare repository pattern:
```
repo-root/
├── .bare/              # Git bare repository (all git metadata)
├── main/               # Worktree for main branch
├── feature-branch/     # Worktree for feature branch
└── user-ticket-123/    # Another worktree
```

### Branch Name Flattening
Branch names with slashes become directory names with dashes:
- `nklauer/FSEC-1234` → `nklauer-FSEC-1234/`
- `feat/new-feature` → `feat-new-feature/`

Commands accept either format (branch name or directory name).

## Workflow

### 1. Detect Context
First, check if we're in a wt-managed repo:
```bash
# Check for .bare directory
wt list
```

If not in a wt repo, guide user to either:
- Initialize a new repo: `wt init <url>`
- Navigate to an existing wt repo

### 2. Common Operations

**List worktrees:**
```bash
wt list
```

**Create new branch worktree:**
```bash
wt new klauern/FSEC-1234-feature
# Creates klauern-FSEC-1234-feature/ from default branch
```

**Create worktree from specific base:**
```bash
wt new klauern/FSEC-1234-feature master
```

**Add worktree for existing remote branch:**
```bash
wt add              # Interactive fzf picker
wt add feature-x    # Specific branch
```

**Sync worktree with main/master:**
```bash
wt sync klauern-FSEC-1234-feature
```

**Remove worktree:**
```bash
wt remove klauern-FSEC-1234-feature
```

**Update all worktrees:**
```bash
wt pull
```

### 3. Initialize New Repo
```bash
wt init git@github.com:org/repo.git
# or with custom directory name
wt init git@github.com:org/repo.git my-repo
```

## Auto-Execute Commands

This skill executes `wt` commands directly. For destructive operations (remove), confirm with user first.

**Safe to auto-execute:**
- `wt list` / `wt ls`
- `wt new <branch>`
- `wt add <branch>`
- `wt sync <name>`
- `wt pull`
- `wt init <url>`

**Confirm before executing:**
- `wt remove <name>` - May lose uncommitted changes

## Integration with Beads

When using beads issue tracking:
1. Each beads issue can have its own worktree
2. Work on issue in isolation without affecting other branches
3. Beads state syncs across worktrees (shared `.beads/` directory)

```bash
# Start work on a beads issue
bd update beads-123 --status=in_progress
wt new klauern/FSEC-$(bd show beads-123 --format=jira-id)
cd ../klauern-FSEC-*
```

## Integration with Claude Sessions

### Starting Session in Worktree
```bash
# Navigate to worktree directory first
cd ~/dev/guardians/zig-workspaces/zendesk-identity-governance/nklauer-FSEC-1234
claude  # Start Claude session in this worktree
```

### Multi-Session Workflow
- Each terminal can have Claude running in different worktrees
- Work on multiple features in parallel
- Each session has its own working directory context

### Context Switching
```bash
# In one terminal (feature A)
cd repo/feature-a/
claude

# In another terminal (feature B)
cd repo/feature-b/
claude
```

## Error Handling

### "Not in a wt-managed repository"
```bash
# Solution: Initialize or navigate to existing repo
wt init <url>
# or
cd /path/to/existing/wt-repo
```

### "Branch not found on remote"
```bash
# For new branches, use 'new' not 'add'
wt new my-new-branch  # Creates new branch
wt add existing-branch  # Checks out existing remote branch
```

### "Worktree already exists"
```bash
# Check existing worktrees
wt list
# Navigate to existing worktree instead
cd ../existing-branch/
```

### "Rebase failed"
```bash
# Resolve conflicts manually in the worktree directory
cd ../worktree-name/
git status
# Fix conflicts, then
git rebase --continue
```

## Best Practices

1. **One branch per worktree** - Keep worktrees focused on single features/tickets
2. **Sync before PRs** - Run `wt sync` before creating pull requests
3. **Clean up merged branches** - Remove worktrees after branches are merged
4. **Use descriptive names** - Include ticket numbers: `klauern/FSEC-1234-description`
5. **Keep main/master clean** - Don't make direct changes in the default branch worktree

## Shell Completions

Install Zsh completions for tab completion:
```bash
wt completions --install
```

Then restart shell or run `exec zsh`.

## Notes

- The `wt` tool is located at `~/bin/wt`
- Requires Python 3.11+ and is run via `uv`
- Uses `fzf` for interactive branch selection (optional but recommended)
- All worktrees share the same Git objects (space efficient)