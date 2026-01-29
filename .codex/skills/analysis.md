# Diff Analysis Techniques

## File-Level Analysis

### Getting Change Summary

```bash
# List files with change type (Added/Modified/Deleted)
git diff --name-status

# Show stats for each file
git diff --stat

# Combined for staged and unstaged
git diff HEAD --name-status
git diff HEAD --stat
```

### Categorizing Files by Type

Parse file paths to determine category:

| Pattern | Category | Typical Scope |
|---------|----------|---------------|
| `src/**/*.ts` | source | directory name |
| `test/**/*` or `*.test.*` | test | module being tested |
| `docs/**/*` or `*.md` | docs | topic |
| `*.json`, `*.yaml`, `*.toml` | config | tool name |
| `.github/**/*` | ci | github/actions |
| `Dockerfile`, `docker-compose.*` | build | docker |

### Detecting Change Relationships

Files likely belong together if:

1. **Same directory**: `src/auth/login.ts` + `src/auth/types.ts`
2. **Test + source**: `src/foo.ts` + `tests/foo.test.ts`
3. **Implementation + types**: `handler.ts` + `types.ts`
4. **Config + consumer**: `tsconfig.json` + new `.ts` files

## Hunk-Level Analysis

When a single file contains unrelated changes:

### Viewing Hunks

```bash
# Show diff with hunk headers
git diff <file>

# Show hunks as patches
git diff -p <file>
```

### Hunk Boundaries

Look for these indicators of separate concerns:

- **Function boundaries**: Different functions modified
- **Import sections**: New imports for new features
- **Comment blocks**: Separate logical sections
- **Line gaps**: Non-adjacent changes often unrelated

### Splitting Hunks

```bash
# Interactive hunk selection
git add -p <file>

# During interactive add:
# y - stage this hunk
# n - skip this hunk
# s - split into smaller hunks
# e - manually edit hunk
```

## Change Classification

### Determining Commit Type

| Change Pattern | Type |
|---------------|------|
| New file with functionality | feat |
| Bug fix, error handling | fix |
| Code restructure, no behavior change | refactor |
| New/modified tests | test |
| README, comments, docs files | docs |
| Dependencies, build config | chore/build |
| Performance improvement | perf |
| CI/CD changes | ci |

### Scope Detection

1. **Directory-based**: Use immediate parent directory
   - `src/auth/login.ts` → scope: `auth`
   - `packages/core/index.ts` → scope: `core`

2. **Feature-based**: When changes span directories
   - Multiple auth-related files → scope: `auth`

3. **Tool-based**: For config files
   - `tsconfig.json` → scope: `typescript`
   - `.eslintrc` → scope: `eslint`

## Dependency Detection

### Commit Order Matters

Detect dependencies to order commits correctly:

```bash
# Check if file A imports file B
rg "import.*from.*fileB" fileA

# Check if changes reference each other
git diff | rg "import|require|from"
```

### Common Dependencies

- Types/interfaces before implementations
- Base classes before derived
- Utilities before consumers
- Config before code that uses it