# Contributing to skillsync

## Development Workflow

### Quality Gates

All code changes must pass local quality checks before pushing:

```bash
make audit  # Runs: tidy, fmt, vet, lint, test
```

Individual checks:
- `make test` - Run tests with race detection and coverage
- `make lint` - Run golangci-lint
- `make fmt` - Format code with gofumpt + goimports
- `make vet` - Run go vet
- `make tidy` - Clean up go.mod and verify dependencies

### CI/CD Pipeline

All pull requests automatically run through GitHub Actions CI with four parallel jobs:

1. **Test Job** - Multi-version testing (Go 1.22.x, 1.23.x, 1.25.x)
   - Unit tests with race detection
   - Integration tests
   - Coverage report generation

2. **Build Job** - Binary compilation and verification
   - Builds via `make build`
   - Verifies binary execution

3. **Lint Job** - Code quality checks
   - `go mod tidy` verification
   - Code formatting validation (gofumpt + goimports)
   - `go vet` static analysis
   - golangci-lint with 15+ enabled linters

4. **Benchmark Job** - Performance testing (PR only)
   - Runs benchmarks with 3s runtime
   - Auto-comments results on PR

All jobs must pass before merge.

### Branch Protection Setup

To enforce CI checks before merging, configure branch protection on the `main` branch:

#### Via GitHub Web UI

1. Navigate to **Settings** → **Branches** → **Branch protection rules**
2. Add rule for `main` branch
3. Enable the following settings:

   **Required status checks:**
   - ✅ Require status checks to pass before merging
   - ✅ Require branches to be up to date before merging
   - Select required checks:
     - `Test (Go 1.22.x)`
     - `Test (Go 1.23.x)`
     - `Test (Go 1.25.x)`
     - `Build`
     - `Lint`

   **Additional recommended settings:**
   - ✅ Require a pull request before merging
   - ✅ Require approvals (1 recommended)
   - ✅ Dismiss stale pull request approvals when new commits are pushed
   - ✅ Require linear history (optional, for clean git history)

#### Via GitHub CLI

```bash
# Require status checks for main branch
gh api repos/:owner/:repo/branches/main/protection \
  --method PUT \
  --field required_status_checks[strict]=true \
  --field required_status_checks[contexts][]=Test (Go 1.22.x) \
  --field required_status_checks[contexts][]=Test (Go 1.23.x) \
  --field required_status_checks[contexts][]=Test (Go 1.25.x) \
  --field required_status_checks[contexts][]=Build \
  --field required_status_checks[contexts][]=Lint \
  --field enforce_admins=true \
  --field required_pull_request_reviews[required_approving_review_count]=1
```

#### Via Terraform (Infrastructure as Code)

```hcl
resource "github_branch_protection" "main" {
  repository_id = github_repository.skillsync.node_id
  pattern       = "main"

  required_status_checks {
    strict   = true
    contexts = [
      "Test (Go 1.22.x)",
      "Test (Go 1.23.x)",
      "Test (Go 1.25.x)",
      "Build",
      "Lint"
    ]
  }

  required_pull_request_reviews {
    required_approving_review_count = 1
    dismiss_stale_reviews           = true
  }

  enforce_admins = true
}
```

### Testing Locally

Before opening a PR:

1. **Run all quality checks:**
   ```bash
   make audit
   ```

2. **Run benchmarks (optional):**
   ```bash
   make bench
   ```

3. **Verify coverage (optional):**
   ```bash
   make test-coverage  # Opens HTML coverage report in browser
   ```

### Code Style

- **Formatting:** Code must be formatted with `gofumpt` and `goimports`
- **Imports:** Use `-local github.com/klauern/skillsync` for local package ordering
- **Linting:** Must pass all golangci-lint checks (see `.golangci.yml`)
- **Error Handling:** All errors must be checked (enforced by `errcheck` linter)
- **Complexity:** Functions should have cyclomatic complexity ≤ 15

### Test Requirements

- **Unit tests:** Required for all new functionality
- **Integration tests:** Required for sync engine changes
- **Race detection:** All tests run with `-race` flag
- **Coverage:** Aim for >80% coverage on new code
- **Table-driven tests:** Preferred pattern for multiple test cases
- **Golden files:** Use for output comparison tests (update with `-update` flag)

### Issue Tracking

This project uses [beads](https://github.com/klauern/beads) for issue tracking:

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim issue
bd close <id>         # Mark complete
bd sync               # Sync with remote
```

See [beads documentation](../.beads/README.md) for full workflow.

## Release Process

Releases are automated via GoReleaser on version tags:

1. Create and push a version tag:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```

2. GitHub Actions automatically:
   - Runs `make audit` before release
   - Builds multi-platform binaries
   - Generates changelog from conventional commits
   - Creates GitHub release with artifacts

## Getting Help

- Check existing documentation in `docs/`
- Review test files for usage examples
- Open an issue via `bd create` for bugs or feature requests
