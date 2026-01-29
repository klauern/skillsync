---
description: Intelligently resolve merge conflicts by analyzing both branches, classifying conflict types, and suggesting or auto-applying resolution strategies.
name: pr-conflict-resolver
---

# PR Conflict Resolver

Automated analysis and resolution of Git merge conflicts.

## Quick Start

**Command**: `/merge-conflicts` — Analyze and resolve merge conflicts

**Documentation**:
- [Workflows](references/workflows.md) — Detection, parsing, and execution flows
- [Patterns & Strategies](references/patterns-and-strategies.md) — Conflict types and resolution approaches
- [Examples](references/examples.md) — Real-world resolution scenarios

## When to Use

**Invoke when**:
- User asks to resolve merge conflicts
- Repository is in merge state with conflicts
- User runs `/merge-conflicts` command

**Don't use for**:
- Rebasing (different workflow)
- Cherry-picking conflicts (use git directly)
- Non-Git version control

## Workflow Overview

```
User Request → Detection (Haiku) → Parse (Haiku) → Classify (Sonnet)
    → Strategy Selection (Sonnet) → [Auto-fix (Haiku) | Guide (Sonnet)]
```

**Phases**:
1. **Detect**: Find conflicted files, check merge state
2. **Parse**: Extract ours/base/theirs content from markers
3. **Classify**: Categorize by type and complexity
4. **Strategize**: Select resolution approach
5. **Execute**: Auto-resolve or guide manual resolution
6. **Verify**: Check no markers remain, run tests

## Conflict Categories

| Type | Auto-Fix | Model | Description |
|------|----------|-------|-------------|
| Whitespace | ✅ Yes | Haiku | Formatting, indentation differences |
| Import order | ✅ Yes | Haiku | Same imports, different order |
| Identical | ✅ Yes | Haiku | Both branches made same change |
| Non-overlapping | ✅ Yes | Haiku | Different additions, no overlap |
| Signature change | ⚠️ Suggest | Sonnet | Parameter additions/modifications |
| Variable rename | ⚠️ Suggest | Sonnet | Incomplete rename across files |
| Logic conflict | ❌ Guide | Sonnet | Different implementations |
| API contract | ❌ Guide | Sonnet | Breaking interface changes |

## Resolution Strategies

| Strategy | When | Action |
|----------|------|--------|
| Auto-resolve | Simple conflicts | Apply algorithm, stage file |
| Merge both | Non-overlapping changes | Combine both additions |
| Choose side | One supersedes other | Keep better, document why |
| Refactor | Both have merit | Design abstraction for both |
| Manual guidance | Complex logic | Present analysis, assist |

## Model Strategy

| Task | Model |
|------|-------|
| Git commands, file I/O, parsing | Haiku |
| Pattern matching for simple conflicts | Haiku |
| Auto-resolve execution | Haiku |
| Conflict classification | Sonnet |
| Intent analysis from commits | Sonnet |
| Strategy selection | Sonnet |
| Resolution guidance | Sonnet |

**Rule**: Mechanical operations → Haiku. Reasoning/analysis → Sonnet.

## Autonomy Guardrails

| Action | Auto-run? | Notes |
|--------|-----------|-------|
| Whitespace/import fixes | ✅ Yes | Show diff afterward |
| Identical/non-overlapping | ✅ Yes | Log resolution |
| Signature changes | ⚠️ Suggest | Present strategy, await approval |
| Logic conflicts | ❌ Never | Explain trade-offs, guide user |
| API changes | ❌ Never | User must decide direction |

## Git Commands Reference

```bash
# Check merge state
git rev-parse --verify MERGE_HEAD

# Find conflicts
git status --porcelain | grep '^UU'

# Get three versions
git show :1:file  # Base
git show :2:file  # Ours
git show :3:file  # Theirs

# After resolution
git add <file>
git diff --check  # Verify no markers
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Not in merge state | Check `git status`, ensure merge started |
| Binary file conflict | Use `git checkout --ours/--theirs` |
| Nested markers | Manual fix, likely editing error |
| Tests fail after resolve | May be semantic conflict, review both sides |
| Cannot determine strategy | Ask user for context about intent |