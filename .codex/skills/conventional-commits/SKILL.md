---
description: Creates conventional commits following conventionalcommits.org. Analyzes git changes and generates properly formatted commit messages with types (feat, fix, docs, etc.) and scopes. Supports single/multi-commit workflows and commit-and-push operations.
name: conventional-commits
---

# Conventional Commits

## Overview

This skill creates well-formatted commit messages following the Conventional Commits specification. It analyzes git changes, determines commit types and scopes, and creates structured commits supporting semantic versioning and automated changelog generation.

## When to Use This Skill

Use this skill when:

- Creating commits that follow Conventional Commits format
- User requests "conventional commits" or "semantic commits"
- Breaking down changes into multiple logical commits with proper scoping
- Committing and pushing changes with structured messages
- Working in repositories that enforce commit message conventions

## Quick Format

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Common types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`

**Breaking changes**: Use '!' after type/scope or `BREAKING CHANGE:` footer

## Workflow Decision Tree

1. **Check staging status**:
   - Staged changes → Single commit workflow
   - Unstaged changes → Multi-commit workflow

2. **Push requirement**:
   - User mentions "push" → Push after committing
   - Otherwise → Commit only

## Sub-Agent Strategy

### Use Haiku 4.5 for

- Quick diff analysis and file categorization
- Simple commit message drafting

### Use Sonnet 4.5 for

- Commit breakpoint determination and multi-commit planning
- Scope identification and complex message composition
- Cross-cutting change analysis

## Progressive Disclosure

Load additional context only when needed:

- **@references/workflows.md** - Detailed single/multi-commit workflows with bash commands and staging strategies
- **@references/examples.md** - Real-world commit examples for features, fixes, breaking changes, and multi-commit scenarios
- **@references/best-practices.md** - Guidelines, common pitfalls, atomic commit patterns, and scope naming conventions
- **@references/format-reference.md** - Complete Conventional Commits specification with all types, components, and breaking change syntax

## Essential Instructions

### Single Commit Workflow

When changes are already staged:

1. Review: `git diff --cached` and `git log -10 --oneline`
2. Determine type, scope, and breaking change status
3. Create message following format above
4. Commit with heredoc:
   ```bash
   git commit -m "$(cat <<'EOF'
   <type>(scope): description

   Optional body explaining rationale.

   BREAKING CHANGE: if applicable
   EOF
   )"
   ```
5. Push if requested: `git push`

**For detailed steps, load @references/workflows.md**

### Multi-Commit Workflow

When nothing is staged and changes need splitting:

1. Review: `git status`, `git diff`, `git log -10 --oneline`
2. Categorize changes by type, scope, and logical boundaries (use Haiku 4.5)
3. Plan commit breakdown with atomic, self-contained commits (use Sonnet 4.5)
4. For each commit: stage files, create commit with heredoc
5. Verify: `git log --oneline -n <count>`
6. Push if requested: `git push`

**For detailed steps and examples, load @references/workflows.md**

## Key Principles

- **Atomic commits**: One logical change per commit
- **Imperative mood**: "add" not "added", "fix" not "fixed"
- **Concise descriptions**: ≤72 characters, lowercase, no period
- **Meaningful bodies**: Explain "why" not "what" (diff shows "what")
- **Explicit breaking changes**: Always use '!' or `BREAKING CHANGE:` footer
- **Multiple small commits**: Better than one large mixed commit

**For comprehensive guidelines, load @references/best-practices.md**