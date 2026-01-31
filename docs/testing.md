# Testing

## Overview

Testing focuses on deterministic, table-driven unit tests plus e2e coverage
through the harness. Keep fixtures minimal, prefer stable outputs for golden
files, and rely on stdlib `testing` for assertions.

## Running
- `make test` - All tests with coverage
- `make test-coverage` - View coverage report in browser

## Unit Tests
- Location: `*_test.go` alongside code under `internal/`
- Patterns: table-driven tests and `t.Run` for subcases
- Conventions: stdlib `testing` only (no testify)

## E2E Harness
- Location: `internal/e2e/harness.go`
- Usage: create a harness with `NewHarness(t)` and run commands with `Run("subcommand", "args")`

## Golden Files
- Export formats: `testdata/export/*.golden`
- Sync results: `testdata/sync/*.golden`
- Update flow: run tests with the `-update` flag

## Tips
- Keep fixtures minimal and focused on one edge case
- Prefer deterministic outputs for golden comparisons
