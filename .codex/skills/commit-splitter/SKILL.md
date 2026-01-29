---
description: Analyzes large diffs and suggests how to split them into multiple atomic commits. Identifies logical groupings by file, function, or feature, then proposes conventional commit messages for each split. Supports automatic or guided splitting.
name: commit-splitter
---

# Commit Splitter

## Overview

This skill analyzes large, mixed changes and helps split them into multiple atomic commits following Conventional Commits. It identifies logical boundaries in diffs and proposes commit groupings with appropriate types, scopes, and messages.

## When to Use This Skill

Use this skill when:

- User has many unstaged changes that should be multiple commits
- A diff contains mixed changes (features, fixes, refactoring, tests)
- User asks to "split commits", "break up changes", or "atomic commits"
- Working with large PRs that need to be broken down
- Reviewing staged changes and realizing they should be separate commits

## Quick Reference

**Analysis approach**:
1. Parse diff to identify all changed files
2. Group changes by logical boundaries (type, scope, feature)
3. Propose split points with commit messages
4. Execute splits interactively or automatically

**Common split boundaries**:
- By file type: source vs tests vs docs vs config
- By directory/scope: different modules or components
- By change type: features vs fixes vs refactoring
- By feature: related changes across files

## Workflow Decision Tree

1. **Check change status**:
   - Staged only → Analyze staged changes, may need to unstage and re-stage
   - Unstaged only → Full flexibility for grouping
   - Mixed → Warn user, recommend starting fresh

2. **Determine split strategy**:
   - Clear file boundaries → Group by directory/scope
   - Mixed file changes → Analyze hunks within files
   - Cross-cutting changes → Group by logical feature

3. **Execution mode**:
   - Interactive → Present plan, confirm each commit
   - Automatic → Execute all commits in sequence

## Sub-Agent Strategy

### Use Haiku 4.5 for

- File listing and categorization
- Simple hunk parsing
- Git command execution

### Use Sonnet 4.5 for

- Commit boundary determination
- Cross-file dependency analysis
- Message composition for complex changes
- Hunk-level splitting decisions

## Progressive Disclosure

Load additional context only when needed:

- **@references/analysis.md** - Detailed diff analysis techniques, hunk parsing, and grouping algorithms
- **@references/splitting.md** - Git commands for staging hunks, interactive add, and partial commits
- **@references/examples.md** - Real-world split scenarios with before/after examples

## Essential Instructions

### Analysis Phase

1. Get current state:
   ```bash
   git status
   git diff --stat           # Summary of unstaged changes
   git diff --cached --stat  # Summary of staged changes
   ```

2. List all changed files with their change types:
   ```bash
   git diff --name-status    # Shows A/M/D status
   ```

3. Categorize files by:
   - **Type**: source, test, docs, config, build
   - **Scope**: directory-based module/component
   - **Change nature**: new feature, fix, refactor, cleanup

### Planning Phase

1. Identify logical commit groups based on:
   - Files that change together for one purpose
   - Changes that could be reverted independently
   - Natural semantic boundaries

2. For each proposed commit, determine:
   - Commit type (feat, fix, refactor, test, docs, chore)
   - Scope (from directory or module name)
   - Description (what this group accomplishes)

3. Order commits logically:
   - Infrastructure/config changes first
   - Core changes before dependent changes
   - Tests with or after their features
   - Docs last

### Execution Phase

**For file-level splits** (all changes in a file go to one commit):
```bash
git add <file1> <file2>
git commit -m "$(cat <<'EOF'
<type>(scope): description
EOF
)"
```

**For hunk-level splits** (partial file changes):
```bash
git add -p <file>  # Interactive hunk selection
# Or stage specific hunks programmatically
git add -N <file>  # Track file without staging
git add -p         # Select hunks
```

**For complex splits, load @references/splitting.md**

### Output Format

Present the split plan as:

```
Proposed Split Plan (N commits):

1. feat(auth): add login endpoint
   Files: src/auth/login.ts, src/auth/types.ts

2. test(auth): add login tests
   Files: tests/auth/login.test.ts

3. docs(auth): document login API
   Files: docs/api/auth.md
```

Then ask: "Proceed with this split? [Y/n/modify]"

## Key Principles

- **Atomic commits**: Each commit should be self-contained and buildable
- **Logical grouping**: Related changes stay together
- **Clear boundaries**: When unsure, prefer more commits over fewer
- **Test alignment**: Tests should accompany their features when possible
- **Order matters**: Later commits can depend on earlier ones, not vice versa

## Common Patterns

| Change Mix | Split Strategy |
|------------|---------------|
| Feature + tests | Two commits: feat, then test |
| Feature + docs | Two commits: feat, then docs |
| Multiple fixes | One commit per fix |
| Refactor + feature | Two commits: refactor first |
| Config + code | Two commits: config first |

**For more patterns and examples, load @references/examples.md**