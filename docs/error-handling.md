# Error Handling Strategy

This guide documents skillsync's error handling patterns, conventions, and best practices for consistent and maintainable error management throughout the codebase.

## Table of Contents

- [Overview](#overview)
- [Error Wrapping](#error-wrapping)
- [Custom Error Types](#custom-error-types)
- [Error Checking Patterns](#error-checking-patterns)
- [Structured Error Logging](#structured-error-logging)
- [Recoverable vs Fatal Errors](#recoverable-vs-fatal-errors)
- [Error Categories](#error-categories)
- [Best Practices](#best-practices)
- [Examples](#examples)

---

## Overview

Skillsync follows Go's idiomatic error handling patterns with these core principles:

- **Error wrapping**: Preserve error chains with context using `fmt.Errorf` with `%w`
- **Custom types**: Use typed errors for semantic error handling
- **Explicit checking**: Handle errors at every call site, never ignore
- **Structured logging**: Log errors with contextual attributes
- **Clear messages**: Error messages include operation context and file paths

**Error Flow Pattern:**

```
[Operation] → [Error] → [Wrap with Context] → [Log] → [Return or Handle]
```

---

## Error Wrapping

### Standard Pattern

Use `fmt.Errorf` with `%w` verb to wrap errors while preserving the error chain:

```go
if err != nil {
    return nil, fmt.Errorf("failed to parse skill file %q: %w", path, err)
}
```

**Why wrap?**
- Adds operation context for debugging
- Preserves original error for `errors.Is` and `errors.As`
- Creates meaningful error traces

### Wrapping Locations

**File operations** (parser modules):
```go
return nil, fmt.Errorf("failed to discover skill files in %q: %w", p.basePath, err)
return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
```

**Configuration** (internal/config/config.go):
```go
return nil, fmt.Errorf("failed to load config from %q: %w", path, err)
```

**Backup operations** (internal/backup/backup.go):
```go
return "", fmt.Errorf("failed to create backup directory: %w", err)
return "", fmt.Errorf("failed to write backup metadata: %w", err)
```

---

## Custom Error Types

### Validation Error Type

**Location:** `internal/validation/validation.go:17-37`

The `Error` struct provides detailed validation failure information:

```go
type Error struct {
    Field   string // Field that failed validation
    Message string // Human-readable error message
    Err     error  // Underlying error (optional)
}

// Error implements the error interface
func (e *Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Field, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap allows errors.Is and errors.As to traverse the error chain
func (e *Error) Unwrap() error {
    return e.Err
}
```

### Error Collection Pattern

**Location:** `internal/validation/validation.go:40-51`

The `Errors` type aggregates multiple validation failures:

```go
type Errors []error

func (e Errors) Error() string {
    if len(e) == 0 {
        return "no errors"
    }
    if len(e) == 1 {
        return e[0].Error()
    }
    return fmt.Sprintf("%d validation errors: %s (and %d more)",
        len(e), e[0].Error(), len(e)-1)
}
```

**Usage:**
```go
result := &validation.Result{}
result.AddError(&validation.Error{
    Field:   "skill.name",
    Message: "name is required",
})
if result.HasErrors() {
    return result.Error() // Returns aggregated errors
}
```

---

## Error Checking Patterns

### Type Assertions with errors.As

Use `errors.As` to check for specific error types and extract values:

```go
var validationErr *validation.Error
if errors.As(err, &validationErr) {
    // Handle validation-specific error
    fmt.Printf("Validation failed for field: %s\n", validationErr.Field)
}
```

**Location example:** `internal/cli/commands.go:1412`

### Error Comparison with errors.Is

Use `errors.Is` to check if an error matches a specific sentinel error:

```go
if errors.Is(err, ErrSkillNotFound) {
    // Handle not found case specifically
}
```

**Location example:** `internal/sync/result_test.go:81`

### OS Error Checking

Use `os.IsNotExist` and similar helpers for OS-specific errors:

```go
if os.IsNotExist(err) {
    // Handle missing file gracefully
    return nil // or default value
}
```

**Common in:** Parser modules (20+ files)

---

## Structured Error Logging

### Error Logging Attribute

**Location:** `internal/logging/logger.go:181-187`

Use the `Err()` helper for consistent error logging:

```go
logging.Error("sync failed",
    logging.Platform("claude-code"),
    logging.Operation("sync"),
    logging.Err(err),
)
```

The `Err()` function is nil-safe:

```go
func Err(err error) slog.Attr {
    if err == nil {
        return slog.Attr{} // Returns empty attribute
    }
    return slog.Any(KeyError, err)
}
```

### Contextual Error Logging

Always include relevant context attributes:

```go
logging.Error("failed to write backup",
    logging.Path(backupPath),
    logging.Platform(platform),
    logging.Err(err),
)
```

**Available attribute helpers:**
- `logging.Platform(p)` - AI platform identifier
- `logging.Skill(name)` - Skill name
- `logging.Path(p)` - File path
- `logging.Operation(op)` - Operation name
- `logging.Err(err)` - Error value

---

## Recoverable vs Fatal Errors

### Recoverable Errors

Errors that allow partial operation success:

**Validation warnings** (non-fatal):
```go
result.AddWarning(&validation.Error{
    Field:   "skill.description",
    Message: "description is empty",
})
// Continue processing
```

**Missing optional files**:
```go
if os.IsNotExist(err) {
    logging.Warn("optional config file not found", logging.Path(path))
    return defaultConfig, nil // Use defaults
}
```

**Sync conflicts** (resolvable):
```go
if conflict != nil {
    result.Action = ActionConflict // Mark as conflict, not failure
    result.Conflict = conflict
    return result // Allow user to resolve
}
```

### Fatal Errors

Errors that prevent operation completion:

**File system failures**:
```go
if err != nil {
    return fmt.Errorf("cannot create backup directory: %w", err)
}
```

**Parse failures**:
```go
if err := yaml.Unmarshal(data, &skill); err != nil {
    return nil, fmt.Errorf("invalid YAML syntax: %w", err)
}
```

**Invalid required configuration**:
```go
if cfg.SkillsDir == "" {
    return errors.New("skills_dir is required in config")
}
```

---

## Error Categories

### Sync Result Actions

**Location:** `internal/sync/result.go`

```go
const (
    ActionFailed   Action = "failed"   // Fatal error during sync
    ActionConflict Action = "conflict" // Resolvable conflict detected
)
```

### Conflict Types

**Location:** `internal/sync/conflict.go`

```go
const (
    ConflictTypeContent  ConflictType = "content"  // Content differs
    ConflictTypeMetadata ConflictType = "metadata" // Metadata differs
    ConflictTypeBoth     ConflictType = "both"     // Both differ
)
```

### Validation Result

**Location:** `internal/validation/validation.go:73-107`

```go
type Result struct {
    Valid    bool     // Overall validation status
    Warnings []error  // Non-fatal issues
    Errors   []error  // Fatal validation failures
}

// Methods:
func (r *Result) AddError(err error)      // Adds error, sets Valid=false
func (r *Result) AddWarning(err error)    // Adds warning, keeps Valid=true
func (r *Result) HasErrors() bool         // Check if validation failed
func (r *Result) Error() error            // Returns aggregated error or nil
func (r *Result) Summary() string         // Human-readable status
```

---

## Best Practices

### 1. Always Wrap Errors with Context

**Good:**
```go
return fmt.Errorf("failed to sync skill %q from %s to %s: %w",
    skillName, source, target, err)
```

**Bad:**
```go
return err // No context about what failed
```

### 2. Use Typed Errors for Semantic Handling

**Good:**
```go
var validationErr *validation.Error
if errors.As(err, &validationErr) {
    // Can access validationErr.Field for specific handling
}
```

**Bad:**
```go
if strings.Contains(err.Error(), "validation") {
    // Fragile string matching
}
```

### 3. Log Errors with Structured Attributes

**Good:**
```go
logging.Error("backup failed",
    logging.Path(path),
    logging.Platform(platform),
    logging.Err(err),
)
```

**Bad:**
```go
log.Printf("Error: %v", err) // Unstructured, no context
```

### 4. Handle Missing Files Gracefully

**Good:**
```go
if os.IsNotExist(err) {
    logging.Debug("config file not found, using defaults",
        logging.Path(configPath))
    return DefaultConfig(), nil
}
```

**Bad:**
```go
return err // Crash on missing optional file
```

### 5. Aggregate Validation Errors

**Good:**
```go
result := &validation.Result{}
for _, skill := range skills {
    if skill.Name == "" {
        result.AddError(&validation.Error{
            Field: "name", Message: "required",
        })
    }
}
return result // All errors reported at once
```

**Bad:**
```go
if skill.Name == "" {
    return errors.New("name required") // Only reports first error
}
```

---

## Examples

### Example 1: Parser Error Handling

**Location:** `internal/parser/claude/claude.go:44-80`

```go
func (p *Parser) Discover() ([]parser.SkillFile, error) {
    // Check if directory exists
    if _, err := os.Stat(p.basePath); err != nil {
        if os.IsNotExist(err) {
            logging.Debug("skills directory not found",
                logging.Platform("claude-code"),
                logging.Path(p.basePath),
            )
            return nil, nil // Not an error - return empty
        }
        return nil, fmt.Errorf("failed to stat directory %q: %w", p.basePath, err)
    }

    // Read directory
    files, err := p.discoverFiles()
    if err != nil {
        return nil, fmt.Errorf("failed to discover skill files in %q: %w",
            p.basePath, err)
    }

    return files, nil
}
```

### Example 2: Validation with Error Aggregation

**Location:** `internal/validation/validation.go:129-200`

```go
func ValidateSkills(skills []parser.SkillFile) *Result {
    result := &Result{Valid: true}

    for _, skill := range skills {
        // Required field checks
        if skill.Name == "" {
            result.AddError(&Error{
                Field:   fmt.Sprintf("skills[%s].name", skill.Path),
                Message: "skill name is required",
            })
        }

        // Warning for optional fields
        if skill.Description == "" {
            result.AddWarning(&Error{
                Field:   fmt.Sprintf("skills[%s].description", skill.Path),
                Message: "description is empty",
            })
        }
    }

    return result
}
```

### Example 3: Sync with Conflict Handling

**Location:** `internal/sync/sync.go:77-176`

```go
func (s *Syncer) SyncSkill(ctx context.Context, skill parser.SkillFile) *Result {
    result := &Result{
        Source: skill,
        Action: ActionCreated,
    }

    // Check if target exists
    existing, err := s.target.Read(skill.Name)
    if err != nil && !os.IsNotExist(err) {
        result.Action = ActionFailed
        result.Error = fmt.Errorf("failed to check existing skill: %w", err)
        logging.Error("sync check failed",
            logging.Skill(skill.Name),
            logging.Err(err),
        )
        return result
    }

    // Detect conflicts
    if existing != nil {
        conflict := detectConflict(skill, existing)
        if conflict != nil {
            result.Action = ActionConflict
            result.Conflict = conflict
            logging.Warn("conflict detected",
                logging.Skill(skill.Name),
                logging.Any("type", conflict.Type),
            )
            return result
        }
    }

    // Write skill
    if err := s.target.Write(skill); err != nil {
        result.Action = ActionFailed
        result.Error = fmt.Errorf("failed to write skill: %w", err)
        logging.Error("sync write failed",
            logging.Skill(skill.Name),
            logging.Err(err),
        )
        return result
    }

    return result
}
```

### Example 4: CLI Error Handling

**Location:** `cmd/skillsync/main.go:12-18`

```go
func main() {
    if err := cli.Run(context.Background(), os.Args); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

**Location:** `internal/cli/commands.go:1412-1428`

```go
// Check for validation errors
var validationErr *validation.Error
if errors.As(err, &validationErr) {
    fmt.Fprintf(os.Stderr, "Validation Error:\n")
    fmt.Fprintf(os.Stderr, "  Field: %s\n", validationErr.Field)
    fmt.Fprintf(os.Stderr, "  Error: %s\n", validationErr.Message)
    if validationErr.Err != nil {
        fmt.Fprintf(os.Stderr, "  Cause: %v\n", validationErr.Err)
    }
    return errors.New("skill validation failed - fix the issues above and try again")
}
```

---

## Summary

Skillsync's error handling strategy provides:

✅ **Contextual errors**: Every error includes operation context and relevant paths
✅ **Error chains**: Wrapped errors preserve the full error history
✅ **Typed errors**: Custom types enable semantic error handling
✅ **Structured logging**: Errors are logged with consistent attributes
✅ **Error aggregation**: Multiple errors can be collected and reported together
✅ **Graceful degradation**: Optional operations fail gracefully with defaults
✅ **Clear messages**: Error messages guide users to resolution

**Key Takeaway:** Follow existing patterns in the codebase. When in doubt, wrap errors with context and log with structured attributes.
