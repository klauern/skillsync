# CI Failure Patterns

Quick reference for failure detection and resolution.

## Pattern Detection Table

| Type | Tool/Signal | Log Pattern | Auto-Fix | Command |
|------|-------------|-------------|----------|---------|
| **Formatting** | Prettier | `prettier`, `Code style issues` | ✅ 99% | `npx prettier --write .` |
| | Black | `black`, `would reformat` | ✅ 99% | `black .` |
| | gofmt | `gofmt`, `not formatted` | ✅ 99% | `gofumpt -w .` |
| | rustfmt | `rustfmt`, `cargo fmt` | ✅ 99% | `cargo fmt` |
| **Linting** | ESLint | `eslint`, `error`, rule name | ⚠️ 60-80% | `npx eslint --fix .` |
| | Ruff | `ruff`, `F841`, `E501` | ⚠️ 70-90% | `ruff check --fix .` |
| | Clippy | `clippy`, warning format | ❌ 20% | Manual |
| | golangci-lint | `golangci-lint` | ⚠️ 40-60% | `golangci-lint run --fix` |
| **Types** | TypeScript | `TS\d{4}`, `error TS` | ❌ 20-30% | Manual (Sonnet) |
| | mypy | `mypy`, `error:` | ❌ 15-25% | Manual |
| **Tests** | Jest | `FAIL`, `expect().toBe()` | ❌ 5-10% | Manual (Sonnet) |
| | pytest | `FAILED`, `AssertionError` | ❌ 5-10% | Manual |
| | Go test | `--- FAIL:`, `want/got` | ❌ 5-10% | Manual |
| **Deps** | npm ci | `out of sync`, lock mismatch | ✅ 95% | `npm install` |
| | poetry | `lock file`, mismatch | ✅ 90% | `poetry lock --no-update` |
| | cargo | `Cargo.lock` | ✅ 90% | `cargo update` |
| **Build** | Webpack/Vite | `Module not found` | ⚠️ 30-40% | Fix imports |
| | tsc build | `TS\d{4}` in build | ❌ 20-30% | Manual |
| **Security** | npm audit | `vulnerabilities found` | ⚠️ 50-70% | `npm audit fix` |
| **Infra** | Secrets | `Resource not accessible`, `##[error]No value` | ❌ 0% | Config fix |
| | Cache | `Failed to restore`, `tar: short read` | ❌ 0% | Bump key / rerun |
| | Timeout | `timeout`, exit 124/143 | ❌ 5% | Optimize / increase limit |
| | Matrix | Only some combinations fail | Varies | Target failing axis |

## Detection Regex Patterns

```regex
# Prettier
prettier|Code style issues found

# Black
black|would reformat|file.*would be reformatted

# ESLint
eslint|\.js:\d+:\d+.*error

# TypeScript
TS\d{4}|\.ts\(\d+,\d+\):.*error

# Jest
FAIL.*\.test\.(js|ts)x?|expect\(.*\)\.toBe\(

# pytest
FAILED\s+([^:]+)::(\w+)

# Lock file
package-lock\.json.*out of sync|lock file.*mismatch

# Import errors
Cannot find module|ModuleNotFoundError

# Secrets/Permissions
Resource not accessible|##\[error\]No value for required secret|exit code 78

# Cache
Failed to restore cache|tar: short read|Artifact has expired

# Timeout
timeout|exceeded.*time limit|operation was canceled
```

## Fix Strategy by Category

### Auto-Fixable (Haiku)
1. **Formatting**: Run formatter, show diff, verify with `--check`
2. **Lock files**: `npm install`, show diff, verify with `npm ci`
3. **Safe lint rules**: Run `--fix`, report remaining manual issues

### Consult User (Sonnet)
1. **Type errors**: Show error, suggest options, wait for approval
2. **Test failures**: Explain expected vs received, ask before fixing
3. **Breaking changes**: Present migration options
4. **Unused variables**: Could be intentional—ask first

### Non-Code Fixes
1. **Secrets**: Direct to Settings → Secrets (never guess values)
2. **Cache corruption**: Bump cache key, rerun
3. **Permissions**: Update workflow `permissions:` block

## Error Codes Reference

| Code | Meaning | Action |
|------|---------|--------|
| Exit 0 | Success | - |
| Exit 1 | General failure | Analyze logs |
| Exit 2 | Compilation error | Type/build issue |
| Exit 78 | Permission denied | Secrets/permissions |
| Exit 124 | Timeout | Optimize or increase limit |
| Exit 143 | SIGTERM (killed) | Resource limit |

## Model Selection

| Confidence | Model | Action |
|------------|-------|--------|
| High (95%+) | Haiku | Auto-fix immediately |
| Medium (70-90%) | Haiku | Attempt fix, verify result |
| Low (<70%) | Sonnet | Analyze before acting |