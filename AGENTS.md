# AGENTS.md

## What This Is

Sync AI coding skills (Claude Code, Cursor, Codex) across platforms and projects.

**Tech Stack**: Go 1.25.4, urfave/cli v3
**Architecture**: CLI entry in `cmd/skillsync`, core packages in `internal/` (see `docs/architecture.md`)

## Why It Exists

Manage a unified skill repository that works seamlessly across different AI coding assistants,
preventing duplication and ensuring consistency.

## How to Work With It

### Running Locally
- `make run` - build and run (see `Makefile`)
- `./bin/skillsync` - direct binary execution (after `make build`)

### Testing
- `make test` - run tests with coverage
- `make test-coverage` - view coverage report in browser
- Convention: use stdlib `testing`; do not add testify

### Building
- `make build` - build to `bin/skillsync`
- `make install` - install to `$GOPATH/bin`

### Quality Gates
- `make audit` - run all checks (tidy, fmt, vet, lint, test)
- `make fmt` - format with gofumpt + goimports
- `make lint` - run golangci-lint (see `.golangci.yml`)

### Go Conventions
- Package structure: `cmd/` for binaries, `internal/` for private packages
- Error handling: always check errors (errcheck enabled)
- Interfaces: define in consuming package

### Issue Tracking (bd)
Track work in beads issues:
- `bd ready` → `bd show <id>` → `bd update <id> --status in_progress` → `bd close <id>`

**Session completion**: `bd sync --flush-only` (local export)

### Deep Dives
- Architecture: `docs/architecture.md`
- Testing & harness: `docs/testing.md`
- Parser development: `docs/parser.md`
