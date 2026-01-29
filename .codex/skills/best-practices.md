# CI Failure Analyzer Best Practices

Guidelines for autonomous fixes and user communication.

## Core Philosophy

> "Handle mechanical issues automatically, but respect user control over meaningful code changes."

**Mechanical** (auto-fix): Formatting, linting, lock files — changes *how* code looks, not *what* it does.

**Meaningful** (ask first): Type errors, test failures, breaking changes — changes *what* code does.

## Safety Checklist

### Before Fixing
1. `git status` — verify clean working directory
2. Check branch — warn if on main/master
3. `command -v <tool>` — verify tool available
4. Estimate impact — warn if >10 files affected

### After Fixing
1. `git diff --stat` — show what changed
2. Re-run tool with `--check` — verify fix worked
3. Summarize changes for user

## User Communication

### Before
```
I found [issue type] in [N] files.
Running: [exact command]
This [is/is not] safe to auto-fix.
```

### After
```
✓ Fixed [category]
Changed files:
  src/index.ts (+2, -2)
  src/utils.ts (+1, -1)
Next step: Review with `git diff` or run `/commit`
```

### For Manual Issues
```
I found [issue] that requires your input:

[File:line]: [error message]

Options:
A) [Option 1]
B) [Option 2]
C) Show me the code

Which would you prefer?
```

## Progressive Fixing

1. **Level 1** (Haiku): Formatting, linting, lock files
2. **Level 2** (Sonnet): Analyze remaining issues, provide plan
3. **Level 3** (Sonnet + user): Apply manual fixes with approval

**Don't batch** — fix one category, verify, then move to next.

## When Fixes Fail

1. Re-check logs — was issue correctly identified?
2. Compare CI vs local config
3. Explain to user with specific next steps

## Trust Principles

1. **Transparency**: Always show what will be done
2. **Reversibility**: Prefer changes that can be undone
3. **Predictability**: Behave consistently
4. **User control**: Never surprise the user with unexpected changes