---
allowed-tools: Bash Read Grep Glob Edit Write
description: Automatically detect, analyze, and upgrade GitHub Actions in workflows. Identifies forked actions and recommends upstream equivalents, handles major version upgrades with breaking change detection.
name: gh-actions-upgrader
---

# GitHub Actions Upgrader

Automates upgrading GitHub Actions: detects outdated versions, identifies forks, handles breaking changes, and creates upgrade PRs.

## Workflow

1. **Detect**: Find workflow files in `.github/workflows/` and extract `uses:` directives
2. **Analyze**: Check fork status and current versions using `gh api`
3. **Discover**: Query GitHub API for latest releases
4. **Plan**: Generate upgrade strategy with breaking change notes
5. **Execute**: Update workflow files, preserving comments/formatting
6. **PR**: Create branch and PR with migration details

## Action Reference Formats

```yaml
# Standard (analyze these)
uses: owner/repo@ref           # e.g., actions/checkout@v4
uses: owner/repo/subdir@ref    # e.g., github/codeql-action/init@v3

# Skip these
uses: ./.github/actions/local  # Local actions
uses: docker://alpine:3.8     # Docker images
```

## Key Commands

### Fork Detection

```bash
# Check if action is a fork
gh api repos/{owner}/{repo} --jq '{fork: .fork, parent: .parent.full_name}'

# Compare fork with upstream (ahead/behind)
gh api repos/{owner}/{repo}/compare/{parent_branch}...{fork_branch} \
  --jq '{ahead: .ahead_by, behind: .behind_by}'
```

### Version Checking

```bash
# Get latest release
gh api repos/{owner}/{repo}/releases/latest --jq '.tag_name'

# Get all tags (if no releases)
gh api repos/{owner}/{repo}/tags --jq '.[].name' | head -5
```

### Extract Actions from Workflows

```bash
# Using yq (preferred)
yq eval '.jobs.*.steps[].uses' .github/workflows/*.yml | sort -u

# Fallback with grep
rg 'uses:\s+([^#\n]+)' .github/workflows/ -o -r '$1'
```

## Decision Points

Prompt user for these decisions:

1. **Fork migration**: "Migrate forked actions to upstream?" (Yes/Keep forks/Selective)
2. **Major versions**: "Apply major version upgrades?" (Apply all/Skip breaking/Review each)
3. **Parameters**: "When parameters change?" (Use new defaults/Keep existing/Add TODOs)

## Fork Recommendations

| Fork Status | Custom Commits | Recommendation |
|-------------|----------------|----------------|
| Behind upstream | 0 | Migrate to upstream |
| Behind upstream | >0 | Review manually |
| Identical | 0 | Migrate to upstream |
| Ahead only | >0 | Keep fork |

## Git Operations

**Branch naming**: `chore/upgrade-github-actions-{date}`

**Commit format**:
```
chore(ci): upgrade GitHub Actions to latest versions

- actions/checkout: v3 → v4
- Migrate custom-org/checkout → actions/checkout (upstream)
```

**PR includes**: Summary, breaking changes per action, fork migration notes, testing checklist.

## Model Strategy

- **Haiku**: File discovery, YAML parsing, version comparison, pattern matching
- **Sonnet**: Breaking change analysis, risk assessment, PR generation, fork migration decisions

## Common Breaking Changes

**actions/checkout v3→v4**: Node.js 16→20, fetch-depth default 1→0
**actions/setup-node v3→v4**: Node.js 16→20
**github/codeql-action v2→v3**: Config schema changes

## Requirements

- `gh` CLI installed and authenticated
- Write access to repository
- `.github/workflows/` directory exists

## Error Handling

- Missing workflows: Exit gracefully
- Invalid YAML: Report and skip
- API rate limits: Wait and retry
- Inaccessible repos: Report as warning

## Configuration (Optional)

```yaml
# .github/actions-upgrade-config.yml
excluded_actions:
  - pinned/action@sha  # Keep pinned
fork_mappings:
  custom-org/checkout: actions/checkout
```