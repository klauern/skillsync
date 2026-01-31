# Testing

## Overview

[Describe the testing strategy and coverage expectations.]

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
