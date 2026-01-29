# Breaking Change Detection

Methods for identifying breaking changes when upgrading dependencies.

## Semver Analysis

### Major Version Detection

```bash
# Compare versions - major bump indicates breaking
current="1.2.3"
latest="2.0.0"

# Extract major versions
current_major=$(echo $current | cut -d. -f1)
latest_major=$(echo $latest | cut -d. -f1)

# Check if major increased
if [ "$latest_major" -gt "$current_major" ]; then
  echo "Breaking change: major version bump"
fi
```

### Version Parsing Patterns

| Pattern | Breaking? |
|---------|-----------|
| 1.x.x → 2.x.x | Yes (major) |
| 1.2.x → 1.3.x | No (minor) |
| 1.2.3 → 1.2.4 | No (patch) |
| 0.x.x → 0.y.x | Maybe (pre-1.0 rules vary) |

## Changelog Fetching

### Local CHANGELOG

```bash
# Common changelog locations
for file in CHANGELOG.md CHANGELOG CHANGES.md HISTORY.md NEWS.md; do
  if [ -f "node_modules/package-name/$file" ]; then
    head -100 "node_modules/package-name/$file"
    break
  fi
done
```

### GitHub Release Notes

```bash
# Fetch latest release
gh api repos/{owner}/{repo}/releases/latest --jq '.body'

# Fetch specific version release
gh api repos/{owner}/{repo}/releases/tags/v2.0.0 --jq '.body'

# List recent releases
gh api repos/{owner}/{repo}/releases --jq '.[0:5] | .[] | "\(.tag_name): \(.name)"'
```

### npm Package Info

```bash
# Check for deprecation
npm view package-name deprecated

# Get repository URL for changelog lookup
npm view package-name repository.url

# Get changelog URL if specified
npm view package-name changelog
```

## Common Breaking Patterns

### npm/Node.js

| Package | Common Breaking Changes |
|---------|------------------------|
| express | Middleware signature changes |
| react | Hook API changes, component lifecycle |
| webpack | Config schema changes |
| typescript | Stricter type checking |

### Python/Poetry

| Package | Common Breaking Changes |
|---------|------------------------|
| django | Settings, URL patterns |
| flask | Blueprint changes |
| pandas | API deprecations |
| numpy | Array behavior changes |

### Go

| Pattern | Common Breaking Changes |
|---------|------------------------|
| Major version | Import path changes (v2+) |
| Go version | Language features, stdlib changes |

### Rust/Cargo

| Pattern | Common Breaking Changes |
|---------|------------------------|
| Major version | API removals, trait changes |
| Edition | Syntax changes |

## Automated Detection Strategy

### Phase 1: Semver Check

```
1. Parse current version from manifest
2. Get latest version from registry
3. Compare major versions
4. Flag if major increased
```

### Phase 2: Changelog Lookup

```
For each major bump:
1. Try local CHANGELOG in dependency
2. Try GitHub releases API
3. Try npm/PyPI package metadata
4. Extract relevant section for version
```

### Phase 3: User Presentation

```
Package: example-lib
Current: 1.5.0 → Latest: 2.0.0 (BREAKING)

Breaking Changes (from CHANGELOG):
- Removed deprecated `oldMethod()`
- Changed `config` parameter to object format
- Minimum Node.js version now 18

Proceed? [Yes / Skip / Show full changelog]
```

## Risk Assessment

| Factor | Risk Level |
|--------|------------|
| Major bump, active project | Medium |
| Major bump, 1+ year old | High |
| Large diff in releases | High |
| Few dependents | Higher risk |
| Many dependents | Lower risk (well-tested) |

## Migration Resources

### Finding Migration Guides

```bash
# Search for migration guide in repo
gh api search/code -X GET \
  -f q="migration+guide+repo:{owner}/{repo}" \
  --jq '.items[].path'

# Common locations
MIGRATION.md
UPGRADING.md
docs/migration.md
docs/upgrading.md
```

### Framework-Specific Resources

| Framework | Migration Resource |
|-----------|-------------------|
| React | reactjs.org/blog (version posts) |
| Vue | v3-migration.vuejs.org |
| Angular | update.angular.io |
| Django | docs.djangoproject.com/releases |