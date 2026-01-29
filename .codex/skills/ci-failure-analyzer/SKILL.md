---
allowed-tools: Bash Read Grep Glob Edit Write
description: Analyze GitHub Actions CI failures, parse logs, identify root causes, and apply fixes. Use when CI checks fail, tests break, or user asks to fix CI, check failures, or debug GitHub Actions.
name: ci-failure-analyzer
---

# CI Failure Analyzer

Automated analysis and resolution of GitHub Actions CI failures.

## Quick Start

**Command**: `/gh-checks` — Analyze and fix CI failures

**Documentation**:
- [Workflows](references/workflows.md) — Step-by-step analysis flows
- [Failure Patterns](references/failure-patterns.md) — Detection patterns and fix commands
- [Tool Detection](references/tool-detection.md) — Project tool/formatter detection
- [Log Parsing](references/log-parsing.md) — Error extraction techniques
- [Examples](references/examples.md) — Real-world scenarios

## When to Use

**Invoke when**:
- User asks "why is CI failing?" or "fix CI"
- User mentions failing tests or checks
- After pushing code, user asks about status
- `/gh-checks` command is executed

**Don't use for**:
- Local test runs (not CI)
- CI configuration questions (not failures)
- Setting up new CI (not debugging)

## Workflow Overview

```
1. Context     → git status, branch, PR existence
2. List Fails  → gh pr checks / gh run list
3. Get Logs    → gh run view <id> --log-failed
4. Analyze     → Categorize, determine fixability
5. Fix         → Auto-fix or guide user
6. Verify      → git diff, optional rerun
```

## Fix Strategy

### Auto-Fix (Haiku)
| Category | Command |
|----------|---------|
| Formatting | `npx prettier --write .`, `black .`, `gofumpt -w .` |
| Linting | `npx eslint --fix .`, `ruff check --fix .` |
| Lock files | `npm install`, `poetry lock --no-update` |

**Always**: Show intent before running, verify with `git diff --stat`

### Consult User (Sonnet)
- Type errors (code intent matters)
- Test failures (logic vs test expectation)
- Breaking changes (migration strategy)
- Anything affecting business logic

### Non-Code Issues
- **Secrets**: Direct to Settings → Secrets (never guess values)
- **Cache**: Bump key, rerun (no code changes)
- **Permissions**: Update workflow YAML `permissions:` block

## Autonomy Guardrails

| Action | Auto-run? | Notes |
|--------|-----------|-------|
| Formatters | ✅ Yes | Show diff afterward |
| Lint --fix | ✅ Yes | Surface remaining manual issues |
| Lock files | ✅ Yes | Warn if major versions changed |
| Type/test fixes | ⚠️ Ask first | Present options, wait for approval |
| Workflow edits | ❌ Never | Guidance only |

## Model Strategy

| Task | Model |
|------|-------|
| Command execution, pattern matching | Haiku |
| Root cause analysis, explanations | Sonnet |

**Rule**: If fix is mechanical → Haiku. If reasoning needed → Sonnet.

## Failure Categories

| Type | Auto-Fix Rate | Primary Model |
|------|--------------|---------------|
| Formatting | 99% | Haiku |
| Linting (--fix) | 60-80% | Haiku |
| Type checking | 20-30% | Sonnet |
| Tests | 5-10% | Sonnet |
| Dependencies | 90-95% | Haiku |
| Infrastructure | 5% | Sonnet |

See [failure-patterns.md](references/failure-patterns.md) for full taxonomy.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| No failing checks | Verify branch, wait for checks to start |
| Cannot retrieve logs | Wait for run to complete |
| Auto-fix didn't work | Check CI config vs local |
| Too many failures | Fix root cause first (build > tests > lint) |