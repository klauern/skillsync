# Conflict Patterns and Resolution Strategies

## Conflict Type Reference

| Type | Detection | Strategy | Auto-Fix |
|------|-----------|----------|----------|
| Identical | `ours == theirs` | Keep either | ✅ |
| Whitespace | Same after normalizing whitespace | Apply formatter | ✅ |
| Import order | Same imports, different order | Sort by convention | ✅ |
| Non-overlapping | Different functions/sections added | Keep both | ✅ |
| Signature change | Parameters added/modified | Update call sites | ⚠️ Suggest |
| Variable rename | Identifier changed, usages outdated | Complete rename | ⚠️ Suggest |
| Version conflict | Same dep, different versions | Choose newer | ⚠️ Suggest |
| Logic conflict | Same function, different impl | Evaluate approaches | ❌ Guide |
| State conflict | Different state management | Align approach | ❌ Guide |
| API contract | Breaking interface changes | Design migration | ❌ Guide |

## Strategy Decision Tree

```
Is content identical?
├─ YES → Keep either (auto-resolve)
└─ NO → Is it whitespace/formatting only?
    ├─ YES → Apply formatter (auto-resolve)
    └─ NO → Are changes non-overlapping?
        ├─ YES → Keep both (auto-resolve)
        └─ NO → Do changes serve different purposes?
            ├─ YES → Merge both implementations
            └─ NO → Is one a bug fix, other a feature?
                ├─ YES → Prioritize bug fix
                └─ NO → Does one supersede the other?
                    ├─ YES → Keep better implementation
                    └─ NO → Manual resolution with guidance
```

## Priority Rules

When multiple strategies apply:

1. **Correctness**: Bug fixes > features, working > broken
2. **Safety**: Conservative > aggressive, tested > untested
3. **Intent**: Preserve both intents when possible
4. **Simplicity**: Simple > complex, clear > clever
5. **Consistency**: Match project conventions

## Intent Analysis

### Extract from Commits
```bash
git log --format="%h %s" origin/main..HEAD -- file
git log --format="%h %s" origin/main..MERGE_HEAD -- file
```

### Intent Keywords
| Intent | Keywords |
|--------|----------|
| Feature | add, implement, create, new |
| Bug fix | fix, bug, resolve, correct |
| Refactor | refactor, restructure, extract |
| Performance | optimize, improve, cache |
| Breaking | breaking, remove, deprecate |

## Resolution Strategies

### 1. Auto-Resolve
**When**: Simple conflicts with clear, safe resolution
- Whitespace, import order, identical changes
- **Action**: Apply algorithm, stage file

### 2. Merge Both
**When**: Both changes valuable, no semantic conflict
- Non-overlapping additions, compatible features
- **Action**: Combine preserving both intents

### 3. Choose Side
**When**: One supersedes other
- Bug fix vs outdated feature
- Better implementation identified
- **Action**: Keep chosen, document reason

### 4. Refactor to Accommodate
**When**: Both approaches have merit, need abstraction
- **Action**: Design new structure for both, implement, test

### 5. Manual with Guidance
**When**: Complex logic, domain knowledge required
- **Action**: Present analysis, explain trade-offs, recommend approach, assist implementation

## Language-Specific Patterns

### Python
- Import sorting: stdlib → third-party → local (PEP 8)
- Type hint conflicts: Usually additive, merge both
- Docstring conflicts: Combine content

### JavaScript/TypeScript
- Import sorting: absolute before relative
- Type definition conflicts: Check for breaking changes
- Semicolon differences: Apply project eslint/prettier

### Go
- Import grouping: std → external → internal
- Use gofumpt for formatting conflicts
- Interface additions: Usually additive

## Anti-Patterns to Avoid

| Anti-Pattern | Problem | Fix |
|--------------|---------|-----|
| Mixed sync/async | Won't compile | Choose one paradigm |
| Dead code | Unreachable branches | Remove superseded code |
| Broken encapsulation | Exposes internals | Respect abstraction |
| Duplicate logic | Same code twice | Consolidate or parameterize |