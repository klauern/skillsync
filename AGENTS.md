# AGENTS.md

## What This Is

Sync AI coding skills (Claude Code, Cursor, Codex) across platforms and projects.

**Tech Stack**: Go 1.25.4, urfave/cli v3
**Architecture**: CLI tool with cmd/ for entry point, internal/ for packages

## Why It Exists

Manage a unified skill repository that works seamlessly across different AI coding assistants,
preventing duplication and ensuring consistency.

## How to Work With It

### Running Locally
```bash
make run              # Build and run
./bin/skillsync       # Direct binary execution
```

### Testing
```bash
make test             # Run tests with coverage
make test-coverage    # View coverage report in browser
```

### Building
```bash
make build            # Build to bin/skillsync
make install          # Install to GOPATH/bin
```

### Quality Gates
```bash
make audit            # Run all checks (tidy, fmt, vet, lint, test)
make fmt              # Format with gofumpt + goimports
make lint             # Run golangci-lint (see .golangci.yml:1)
```

### Go Conventions
- Format: gofumpt + goimports (Makefile:36)
- Lint: golangci-lint with custom rules (.golangci.yml:1)
- Package structure: cmd/ for binaries, internal/ for private packages
- Error handling: Always check errors (errcheck enabled in .golangci.yml:6)

### Issue Tracking (bd)
Track work in beads issues:
```bash
bd ready              # Find available work
bd show <id>          # View details
bd update <id> --status in_progress
bd close <id>         # Complete work
```

**Session completion**: Always run `bd sync` and `git push` before ending.
