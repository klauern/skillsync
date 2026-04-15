# Tool Detection Guide

How to detect formatters, linters, and build tools in a project.

## Detection Priority

1. **CI Workflows** (99% confidence) — What CI actually runs
2. **Config Files** (90-95%) — Explicit configuration
3. **Package Scripts** (75-85%) — Defined in package.json/pyproject.toml
4. **Dependencies** (60-70%) — Installed but may not be configured
5. **Language Defaults** (40-50%) — Best guess

## Quick Detection Commands

### Check CI Workflows First
```bash
# What does CI actually run?
grep -rh 'prettier\|eslint\|black\|gofumpt\|cargo fmt' .github/workflows/
```

### JavaScript/TypeScript
```bash
# Prettier
ls .prettierrc* prettier.config.* 2>/dev/null || grep '"prettier"' package.json

# ESLint
ls .eslintrc* eslint.config.* 2>/dev/null

# TypeScript
test -f tsconfig.json && echo "TypeScript project"
```

### Python
```bash
# Black
grep -q '\[tool\.black\]' pyproject.toml && echo "Black configured"

# Ruff
grep -q '\[tool\.ruff\]' pyproject.toml || test -f ruff.toml

# mypy
grep -q '\[tool\.mypy\]' pyproject.toml || test -f mypy.ini
```

### Go
```bash
# gofumpt (preferred per AGENTS.md)
grep -r 'gofumpt' .github/workflows/ || echo "Default: gofumpt"

# golangci-lint
ls .golangci.* 2>/dev/null
```

### Rust
```bash
# rustfmt (default for all Rust projects)
test -f Cargo.toml && echo "Use: cargo fmt"

# Config
test -f rustfmt.toml && echo "Custom rustfmt config"
```

## Config Files Reference

| Tool | Config Files |
|------|--------------|
| Prettier | `.prettierrc`, `.prettierrc.json`, `prettier.config.js` |
| ESLint | `.eslintrc`, `.eslintrc.json`, `eslint.config.js` |
| TypeScript | `tsconfig.json` |
| Black | `pyproject.toml [tool.black]`, `.black` |
| Ruff | `pyproject.toml [tool.ruff]`, `ruff.toml` |
| mypy | `pyproject.toml [tool.mypy]`, `mypy.ini` |
| gofmt/gofumpt | None (standard) |
| golangci-lint | `.golangci.yml`, `.golangci.yaml` |
| rustfmt | `rustfmt.toml`, `.rustfmt.toml` |

## Monorepo Handling

```bash
# Find all manifest files
fd -tf --max-depth 4 'package.json|pyproject.toml|go.mod|Cargo.toml'
```

**Guardrails**:
- Never run formatters at repo root if tools differ per workspace
- Scope commands to the failing package directory
- Warn if conflicting formatter versions detected

## Language Defaults

| Language | Formatter | Linter | Type Checker |
|----------|-----------|--------|--------------|
| JS/TS | prettier | eslint | typescript |
| Python | black | ruff | mypy |
| Go | gofumpt* | golangci-lint | - |
| Rust | rustfmt | clippy | - |

*User preference: use `gofumpt` over `gofmt` per AGENTS.md

## Model Strategy

- **Haiku**: File existence checks, config parsing, pattern matching
- **Sonnet**: Conflicting tools, trade-off explanations, recommendations when nothing detected