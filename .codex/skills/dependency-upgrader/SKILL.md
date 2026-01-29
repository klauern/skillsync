---
allowed-tools: Bash Read Grep Glob Edit Write
description: Analyze package dependencies across npm, poetry, go.mod, cargo. Detects outdated packages, identifies breaking changes via semver analysis and changelogs, suggests migration paths.
name: dependency-upgrader
---

# Dependency Upgrader

Automates upgrading package dependencies: detects outdated versions, identifies breaking changes, and creates upgrade PRs.

## Quick Start

```
User: Check for outdated dependencies
User: Upgrade my npm packages
User: What packages need updating?
```

The skill will: detect ecosystem(s) → analyze outdated → categorize by semver → plan upgrades → execute with approval

## Workflow

1. **Detect**: Find manifest files (package.json, pyproject.toml, go.mod, Cargo.toml)
2. **Analyze**: Run ecosystem-specific outdated commands
3. **Categorize**: Group updates by patch/minor/major
4. **Plan**: Generate upgrade strategy with breaking change notes
5. **Execute**: Update manifest files (prompt for majors)
6. **Verify**: Run lock file update, optionally test

## Ecosystem Detection

| Ecosystem | Manifest | Check Outdated | Update |
|-----------|----------|----------------|--------|
| npm | package.json | `npx ncu` or `npm outdated` | `npx ncu -u` |
| poetry | pyproject.toml | `poetry show --outdated` | `poetry update` |
| go | go.mod | `go list -m -u all` | `go get -u ./...` |
| cargo | Cargo.toml | `cargo outdated` | `cargo update` |

## Key Commands

### npm (using npm-check-updates)

```bash
npx ncu                    # Check outdated (colorized)
npx ncu --jsonUpgraded     # JSON output for parsing
npx ncu -u                 # Update package.json
npx ncu --target minor     # Only minor/patch updates
```

### poetry

```bash
poetry show --outdated     # List outdated
poetry update              # Update all
poetry update pkg-name     # Update specific
```

### go

```bash
go list -m -u all         # List outdated modules
go get -u ./...           # Update all
go mod tidy               # Clean up go.mod
```

### cargo

```bash
cargo outdated            # Check outdated (requires cargo-outdated)
cargo update              # Update Cargo.lock
cargo upgrade             # Update Cargo.toml (requires cargo-edit)
```

## Decision Points

Prompt user for:

1. **Major versions**: "Apply major version upgrades?" (Apply all / Skip majors / Review each)
2. **Multiple ecosystems**: "Found npm and poetry. Upgrade both?" (All / Select)
3. **Breaking changes**: "Package X has breaking changes. Proceed?" (Yes / Skip / Show changelog)

## Breaking Change Detection

| Method | Source | Reliability |
|--------|--------|-------------|
| Semver major bump | Version comparison | High |
| CHANGELOG.md | Local file or GitHub | Medium |
| Release notes | `gh api repos/{owner}/{repo}/releases` | Medium |
| npm deprecation | `npm view <pkg> deprecated` | High |

**Strategy**: Flag major bumps → fetch changelog/releases for context → present to user

See [breaking-changes.md](references/breaking-changes.md) for detection methods.

## Model Strategy

| Task | Model | Rationale |
|------|-------|-----------|
| File discovery, manifest parsing | Haiku | Fast, deterministic |
| Command execution, version parsing | Haiku | Mechanical operations |
| Breaking change analysis | Sonnet | Complex reasoning |
| PR/commit message generation | Sonnet | Natural language synthesis |

## Git Operations

**Branch**: `chore/upgrade-{ecosystem}-deps-{date}` or `chore/upgrade-dependencies-{date}`

**Commit format**:
```
chore(deps): upgrade {ecosystem} dependencies

- package-a: 1.0.0 → 2.0.0 (major)
- package-b: 2.1.0 → 2.2.0 (minor)

Breaking changes:
- package-a: API change in foo() method
```

## Requirements

- `ncu` (npm-check-updates) for npm - optional but preferred
- `cargo-outdated` for Rust - optional
- `gh` CLI for changelog fetching

## Error Handling

| Issue | Solution |
|-------|----------|
| No manifest found | Report and exit gracefully |
| Tool not installed | Suggest installation, use fallback |
| Network error | Report, suggest retry |
| Lock file conflict | Guide through resolution |

## Configuration (Optional)

```yaml
# .dependency-upgrade-config.yml
excluded:
  - pinned-package@1.0.0
ecosystems:
  - npm
  - poetry
```

## References

- [ecosystems.md](references/ecosystems.md) - Detailed ecosystem commands
- [breaking-changes.md](references/breaking-changes.md) - Breaking change detection
- [examples.md](references/examples.md) - Real-world upgrade scenarios