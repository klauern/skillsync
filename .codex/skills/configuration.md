# Configuration

Git-trim and git aliases use Git's native configuration system.

## git-trim Options

### trim.bases

Define base branches (comma-separated):
```bash
git config trim.bases "develop,master"     # Git-flow
git config trim.bases "main"               # GitHub flow
```

### trim.exclude

Exclude branches from cleanup (space-separated):
```bash
git config trim.exclude "staging production qa"
```

## Git Aliases

Your optimization aliases in `~/.gitconfig`:

```ini
[alias]
    cleanup = "!git branch --merged | grep -v '\\*\\|master' | xargs -n 1 git branch -d"

    sweep = "!f(){ git branch --merged $([[ $1 != \"-f\" ]] && git rev-parse master) | egrep -v \"(^\\*|^\\s*(master|develop)$)\" | xargs git branch -d; }; f"

    trimall = "!f() { \
        echo '1. Fetching and pruning remotes...'; \
        git fetch --all --prune; \
        echo '2. Running git-trim...'; \
        git trim --no-confirm -d merged:*,stray,diverged:*,local,remote:*; \
        echo '3. Running cleanup...'; \
        git cleanup; \
        echo '4. Running sweep...'; \
        git sweep; \
        echo '5. Optimizing repository...'; \
        git optimize; \
        echo 'Done!'; \
    }; f"

    pruner = "!git prune --expire=now; git reflog expire --expire-unreachable=now --rewrite --all"

    repacker = "!git repack -a -d --depth=250 --window=250"

    optimize = "!git pruner; git repacker; git prune-packed"
```

## Common Configurations

| Workflow | trim.bases | trim.exclude |
|----------|------------|--------------|
| GitHub Flow | `main` | (none) |
| Git-Flow | `develop,master` | `staging production` |
| Trunk-Based | `trunk` | `release-*` |

## View/Remove Config

```bash
# View
git config trim.bases
git config --list | grep trim

# Remove
git config --unset trim.bases
git config --unset trim.exclude
```