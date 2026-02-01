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
- `just run` - build and run (see `Justfile`)
- `./bin/skillsync` - direct binary execution (after `just build`)

### Testing
- `just test` - run tests with coverage
- `just test-coverage` - view coverage report in browser
- Convention: use stdlib `testing`; do not add testify

### Building
- `just build` - build to `bin/skillsync`
- `just install` - install to `$GOPATH/bin`

### Quality Gates
- `just audit` - run all checks (tidy, fmt, vet, lint, test)
- `just fmt` - format with gofumpt + goimports
- `just lint` - run golangci-lint (see `.golangci.yml`)

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

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
