# Log Parsing Guide

Techniques for parsing GitHub Actions logs and extracting errors.

## Log Structure

GitHub Actions logs via `gh run view --log-failed`:
```
job-name	step-name	2024-01-15T10:30:45.1234567Z	log-line
```

Extract just the log lines:
```bash
gh run view <run-id> --log-failed | cut -f4-
```

## ANSI Color Stripping

```bash
# Strip color codes
sed 's/\x1b\[[0-9;]*m//g'
```

## Error Location Patterns

### File:Line:Column Formats

| Language | Format | Example |
|----------|--------|---------|
| TypeScript | `file(line,col): error TS####` | `src/index.ts(15,10): error TS2322` |
| ESLint | `file:line:col  level  message  rule` | `/src/file.ts:15:10  error  ...` |
| Python | `file:line: level: message` | `src/utils.py:23: error: ...` |
| Go | `file:line:col: message` | `./main.go:15:10: undefined: foo` |
| Rust | `error[CODE]: message\n  --> file:line:col` | `error[E0425]: ...\n  --> src/main.rs:15:10` |

### Universal Regex
```regex
([A-Za-z0-9_/.-]+\.[A-Za-z]+)[:(\[](\d+)[,:](\d+)?[\])]?.*(?:error|Error|ERROR)
```

## Test Framework Output

### Jest
```
FAIL src/api.test.ts
  ✕ test name (15 ms)
    expect(received).toBe(expected)
    Expected: 5
    Received: 3
    > 12 |     expect(result).toBe(5);
```

Extract: `✕\s+(.+?)\s+\(\d+ ms\)` → test name

### pytest
```
FAILED tests/test_api.py::test_get_data - AssertionError: assert 3 == 5
```

Extract: `FAILED\s+([^:]+)::(\w+)` → file, test name

### Go test
```
--- FAIL: TestGetData (0.00s)
    api_test.go:15: got 3, want 5
```

Extract: `---\s+FAIL:\s+(\w+)` → test name

## Stack Trace Patterns

### JavaScript
```regex
at\s+(?:([\w.<>]+)\s+)?\(([^:]+):(\d+):(\d+)\)
```

### Python
```regex
File\s+"([^"]+)",\s+line\s+(\d+),\s+in\s+(.+)
```

## Matrix Job Parsing

```bash
# Get all job results
gh run view <run-id> --json jobs --jq '.jobs[] | {name, conclusion}'

# Target specific matrix child
gh run view <run-id> --job "test (node-version: 18, os: ubuntu-latest)" --log-failed
```

## Secret/Permission Indicators

Look for:
- `Resource not accessible by integration`
- `##[error]No value for required secret`
- `HttpError: 403 Forbidden`
- Exit code `78`

Extract secret name: `secrets\.([A-Z0-9_]+)`

## Truncation Handling

If logs are truncated:
```bash
# Focus on errors near the end
gh run view --log-failed | grep -i error | tail -50

# Or download full logs
gh run download <run-id> --name <artifact-name>
```

## Quick Reference Patterns

```regex
# Error keywords
(?i)(error|warning|fail|failed|failure)

# Error codes
(TS|E|W|F)\d{3,4}

# File with line number
([A-Za-z0-9_/.-]+\.[A-Za-z]+):(\d+)

# Expected/Received
Expected:\s*(.+?)\s*Received:\s*(.+)

# Package versions
(\d+)\.(\d+)\.(\d+)
```

## Model Strategy

- **Haiku**: Pattern matching, regex extraction, file path parsing
- **Sonnet**: Error semantics, failure correlation, complex diagnostics