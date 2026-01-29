---
allowed-tools: Bash Read
description: Git repository optimization including branch cleanup (git-trim, cleanup, sweep), garbage collection, and maintenance
name: git-optimize
---

# Git Optimize

Repository maintenance through branch cleanup and optimization operations.

## Quick Start

**Command**: `/git-optimize` — Clean branches and optimize repository

**Documentation**:
- [Configuration](references/configuration.md) — Alias and git-trim setup
- [Merge Detection](references/merge_detection.md) — How git-trim detects merges
- [Installation](references/installation.md) — git-trim installation

## When to Use

**Invoke when**:
- User asks to clean up branches or merged PRs
- Repository performance is slow or .git is large
- Preparing repository for archival
- Running maintenance workflows

**Don't use for**:
- Branch creation or checkout
- Commit operations
- Remote management (use standard git)

## Commands

| Command | Purpose | Time | Safety |
|---------|---------|------|--------|
| `git cleanup` | Delete branches merged to master | Seconds | High |
| `git sweep` | Aggressive cleanup (master/develop) | Seconds | High |
| `git trim` | Smart detection (merged/stray/squash) | Seconds | High |
| `git trim --dry-run` | Preview what trim would delete | Seconds | Safe |
| `git pruner` | Remove unreachable objects | Minutes-Hours | Medium |
| `git repacker` | Optimal delta compression | Hours | High |
| `git optimize` | pruner + repacker + prune-packed | Hours | Medium |
| `git trimall` | Full workflow (fetch→trim→cleanup→optimize) | 10min-Hours | Medium |

## Workflows

**After PR merge** (daily):
```bash
git checkout main && git pull && git cleanup
```

**Weekly maintenance**:
```bash
git fetch --all --prune && git trim --dry-run && git trim
```

**Monthly deep clean**:
```bash
git trimall
```

## Model Strategy

| Task | Model |
|------|-------|
| Command execution, alias listing | Haiku |
| Analyzing branches, recommending strategy | Sonnet |

## Configuration

**Git-flow setup** (multiple base branches):
```bash
git config trim.bases "develop,master"
git config trim.exclude "staging production"
```

**Verify aliases**:
```bash
git config alias.cleanup
git config alias.trimall
```

See [configuration.md](references/configuration.md) for full alias definitions.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| git-trim not found | `brew install foriequal0/git-trim/git-trim` |
| Alias not working | Check `git config alias.<name>` |
| Repo still large | `git gc --prune=now --aggressive` |
| Stray branches flagged | `git log origin/main..<branch> --oneline` to check unmerged commits |

## Safety

**Always safe**: cleanup, sweep, trim --dry-run, repacker

**Use caution**: pruner (removes objects), optimize, trimall, sweep -f

**Best practices**:
1. Use `--dry-run` first
2. Push important work before aggressive cleanup
3. Schedule optimize/repacker overnight
4. Use `git reflog` for recovery

## Version History

- **1.1.0**: Optimized skill documentation (62% reduction)
- **1.0.0**: Initial release with git-trim integration