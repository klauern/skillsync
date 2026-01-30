# Worktree Quick Reference

## Command Summary

| Command | Alias | Description |
|---------|-------|-------------|
| `wt init <url> [dir]` | - | Clone repo with bare+worktree setup |
| `wt list` | `ls` | List all worktrees |
| `wt add [branch]` | - | Add worktree for existing remote branch |
| `wt new <branch> [base]` | - | Create new branch worktree |
| `wt remove <name>` | `rm` | Remove a worktree |
| `wt sync <name>` | - | Rebase branch on default branch |
| `wt pull` | - | Fetch and update all worktrees |
| `wt migrate [--apply]` | - | Move branches/* worktrees to root |
| `wt completions [--install]` | - | Generate/install shell completions |
| `wt help` | `-h` | Show help |

## Common Commands

```bash
# List worktrees
wt list

# Create new feature branch
wt new klauern/FSEC-1234-feature

# Add existing remote branch
wt add feature-branch
wt add                      # Interactive (fzf)

# Sync with main
wt sync klauern-FSEC-1234

# Remove worktree
wt remove klauern-FSEC-1234

# Update all
wt pull

# Initialize new repo
wt init git@github.com:org/repo.git
```

## Name Resolution

The `<name>` argument accepts multiple formats:

| Input | Matches |
|-------|---------|
| `klauern/FSEC-1234` | Branch name |
| `klauern-FSEC-1234` | Directory name |
| `/full/path/to/worktree` | Absolute path |

## Directory Structure

```
repo-root/
├── .bare/           # Git bare repository
├── main/            # Default branch worktree
├── feature-a/       # Feature branch worktree
└── klauern-ticket/  # User branch worktree
```

## Branch → Directory Mapping

| Branch Name | Directory Name |
|-------------|----------------|
| `main` | `main/` |
| `klauern/FSEC-1234` | `klauern-FSEC-1234/` |
| `feat/new-feature` | `feat-new-feature/` |
| `fix/bug-123` | `fix-bug-123/` |

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (check stderr for details) |

## Error Messages

| Message | Cause | Solution |
|---------|-------|----------|
| "Not in a wt-managed repository" | No `.bare/` found | Run `wt init` or `cd` to wt repo |
| "Branch not found on remote" | Branch doesn't exist | Use `wt new` to create it |
| "Worktree already exists" | Directory exists | Navigate to existing worktree |
| "Worktree not found" | Invalid name | Check `wt list` for valid names |
| "Rebase failed" | Merge conflicts | Resolve conflicts manually |

## Dependencies

| Tool | Required | Purpose |
|------|----------|---------|
| Python 3.11+ | Yes | Script runtime |
| uv | Yes | Script runner |
| git | Yes | Version control |
| fzf | Optional | Interactive branch picker |

## Locations

| Item | Path |
|------|------|
| wt tool | `~/bin/wt` |
| Completions | `~/.completions/_wt` |
| Workspaces | `~/dev/guardians/zig-workspaces/` |

## Shell Completions Setup

```bash
# Install completions
wt completions --install

# Add to ~/.zshrc (before compinit)
fpath=(~/.completions $fpath)

# Restart shell
exec zsh
```

## Integration Points

### With Beads
```bash
bd ready                    # Find work
wt new klauern/FSEC-XXX     # Create worktree
bd close beads-XXX          # Complete work
wt remove klauern-FSEC-XXX  # Clean up
```

### With gh CLI
```bash
wt new klauern/FSEC-1234    # Create worktree
# ... make changes ...
git push -u origin klauern/FSEC-1234
gh pr create                # Create PR
```

### With Claude
```bash
cd repo/worktree-name/      # Navigate to worktree
claude                      # Start session in context
```

## Flags and Options

### wt init
```
wt init <url>           # Auto-detect directory name
wt init <url> <dir>     # Specify directory name
```

### wt new
```
wt new <branch>         # Create from default branch
wt new <branch> <base>  # Create from specific base
```

### wt migrate
```
wt migrate              # Dry run (preview)
wt migrate --apply      # Execute migration
```

### wt completions
```
wt completions          # Print to stdout
wt completions --install # Install to ~/.completions/
```