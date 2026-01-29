# Conventional Commits Format Reference

Complete specification following [conventionalcommits.org](https://www.conventionalcommits.org/).

## Format Structure

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

## Commit Types

### Required Types (SemVer)

- **`fix`**: Patches a bug (PATCH in SemVer)
- **`feat`**: Introduces a new feature (MINOR in SemVer)

### Additional Types

- **`build`**: Build system or external dependency changes
- **`chore`**: Routine tasks, maintenance, or tooling
- **`ci`**: CI configuration files and scripts
- **`docs`**: Documentation only
- **`style`**: Formatting, whitespace (no code behavior change)
- **`refactor`**: Code restructuring (no behavior change)
- **`perf`**: Performance improvements
- **`test`**: Adding or correcting tests

## Components

### Type (Required)

Indicates the nature of the change. Lowercase (except `BREAKING CHANGE` in footers).

### Scope (Optional)

Noun describing codebase section, enclosed in parentheses.

**Common patterns**:
- Component names: `(auth)`, `(user-profile)`, `(dashboard)`
- Layers: `(api)`, `(ui)`, `(database)`, `(cli)`
- Modules: `(parser)`, `(compiler)`, `(router)`

**Examples**:
```
feat(parser): add ability to parse arrays
fix(api): correct endpoint response format
docs(readme): update installation instructions
```

### Description (Required)

Short summary immediately after type/scope and colon+space.

**Rules**:
- Imperative, present tense: "change" not "changed"
- Lowercase first letter
- No period at end
- ≤72 characters

### Body (Optional)

Detailed explanation, one blank line after description.

**Guidelines**:
- Imperative, present tense
- Explain motivation and context
- Contrast with previous behavior
- Wrap at 72 characters

### Footer(s) (Optional)

Metadata following blank line after body.

**Format**: `Token: value` or `Token #value`

**Common tokens**:
- `BREAKING CHANGE`: Documents breaking changes
- `Fixes`, `Closes`: Issue references
- `Refs`: Related issues
- `Reviewed-by`, `Acked-by`: Co-authors/reviewers

**Note**: Use hyphens for whitespace (e.g., `Acked-by`)

## Breaking Changes

Breaking changes (MAJOR in SemVer) MUST use one or both methods:

### Method 1: Exclamation Mark

Append '!' before colon:
```
feat!: remove support for Node 6
refactor(api)!: change authentication method
```

### Method 2: BREAKING CHANGE Footer

```
feat: allow provided config object to extend other configs

BREAKING CHANGE: `extends` key in config file is now used for extending other config files
```

## Rules Summary

- Types are lowercase (except `BREAKING CHANGE` footer must be uppercase)
- `BREAKING-CHANGE` is synonymous with `BREAKING CHANGE`
- '!' in `type!:` is alternative to `BREAKING CHANGE` footer
- Scope is optional but recommended for larger projects
- Body and footers are optional but encouraged for complex changes
- Footers use hyphens for whitespace tokens

## Common Patterns by Category

### Database Changes

```
feat(db): add user preferences table
refactor(db): normalize user address data
fix(db): correct migration rollback script
```

### API Changes

```
feat(api): add pagination to user list endpoint
fix(api): correct status code for validation errors
refactor(api)!: standardize error response format
```

### Documentation

```
docs: add API authentication examples
docs(readme): update installation steps for macOS
docs(api): document new rate limiting headers
```

### Dependencies

```
build(deps): bump lodash from 4.17.19 to 4.17.21
chore(deps-dev): update eslint to version 8.0.0
build(npm)!: upgrade to Node.js 18 minimum
```

### CI/CD

```
ci: add automated security scanning
ci(github): update Node version in actions
ci(deploy): add staging environment workflow
```