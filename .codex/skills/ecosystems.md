# Ecosystem Reference

Detailed commands and configuration for each supported package ecosystem.

## npm / Node.js

### Detection

```bash
# Check for package.json
ls package.json 2>/dev/null && echo "npm detected"

# Check for monorepo workspaces
jq -r '.workspaces // empty' package.json
```

### npm-check-updates (Preferred)

```bash
# Install globally
npm install -g npm-check-updates

# Basic usage
ncu                        # Check all
ncu -u                     # Update package.json
ncu -i                     # Interactive mode
ncu -t minor               # Only minor/patch
ncu -t patch               # Only patch

# Filtering
ncu -f "react*"            # Filter by pattern
ncu -x "lodash"            # Exclude packages
ncu --dep prod             # Production only
ncu --dep dev              # Dev only

# Output formats
ncu --jsonUpgraded         # JSON for parsing
ncu --format lines         # One per line
```

### npm outdated (Fallback)

```bash
npm outdated               # Human readable
npm outdated --json        # JSON format
npm outdated --long        # Include homepage
```

### Lock file operations

```bash
npm install                # Regenerate lock
npm ci                     # Clean install from lock
npm update                 # Update within ranges
```

### Monorepo Support

```bash
# With workspaces
ncu --workspaces
ncu -ws

# Specific workspace
ncu --workspace packages/core
```

## Poetry / Python

### Detection

```bash
# Check for pyproject.toml with poetry
grep -q "tool.poetry" pyproject.toml 2>/dev/null && echo "poetry detected"
```

### Commands

```bash
# Check outdated
poetry show --outdated          # All outdated
poetry show --outdated --top-level  # Direct deps only

# Update
poetry update                   # All dependencies
poetry update package-name      # Specific package
poetry update --dry-run         # Preview changes

# Lock operations
poetry lock                     # Update lock without install
poetry lock --no-update         # Regenerate without updating

# Add with constraints
poetry add "package@^2.0"       # Caret constraint
poetry add "package@~2.0"       # Tilde constraint
```

### Version Constraints

| Constraint | Meaning |
|------------|---------|
| `^1.2.3` | >=1.2.3 <2.0.0 |
| `~1.2.3` | >=1.2.3 <1.3.0 |
| `>=1.2,<2.0` | Explicit range |
| `*` | Any version |

### Groups

```bash
poetry show --outdated --only main
poetry show --outdated --only dev
poetry update --only main
```

## Go Modules

### Detection

```bash
# Check for go.mod
ls go.mod 2>/dev/null && echo "go detected"
```

### Commands

```bash
# List outdated
go list -m -u all                    # All modules with updates
go list -m -u -json all              # JSON format

# Update
go get -u ./...                      # All to latest
go get -u=patch ./...                # Patch versions only
go get github.com/pkg/name@latest    # Specific module
go get github.com/pkg/name@v1.2.3    # Specific version

# Maintenance
go mod tidy                          # Remove unused, add missing
go mod verify                        # Verify checksums
go mod download                      # Download dependencies
```

### Version Selection

```bash
go get pkg@v1.2.3     # Exact version
go get pkg@latest     # Latest tagged
go get pkg@upgrade    # Latest allowed by go.mod
go get pkg@patch      # Latest patch
```

### Workspace Support (Go 1.18+)

```bash
# go.work file
go work use ./module-a ./module-b
go list -m -u all     # Works across workspace
```

## Cargo / Rust

### Detection

```bash
# Check for Cargo.toml
ls Cargo.toml 2>/dev/null && echo "cargo detected"
```

### cargo-outdated

```bash
# Install
cargo install cargo-outdated

# Usage
cargo outdated                 # All outdated
cargo outdated --depth 1       # Direct deps only
cargo outdated --root-deps-only
cargo outdated --format json   # JSON format

# Filtering
cargo outdated --exclude pkg   # Exclude package
cargo outdated --packages pkg  # Only specific
```

### cargo-edit (for Cargo.toml updates)

```bash
# Install
cargo install cargo-edit

# Upgrade commands
cargo upgrade                  # All to latest
cargo upgrade --incompatible   # Include breaking
cargo upgrade pkg              # Specific package
cargo upgrade --dry-run        # Preview

# Add/remove
cargo add package
cargo rm package
```

### Native Commands

```bash
cargo update              # Update Cargo.lock only
cargo update -p pkg       # Update specific in lock
```

### Workspace Support

```bash
# In workspace root
cargo outdated --workspace
cargo upgrade --workspace
```

## Tool Installation Summary

| Ecosystem | Tool | Install Command |
|-----------|------|-----------------|
| npm | npm-check-updates | `npm i -g npm-check-updates` |
| poetry | poetry | `pipx install poetry` |
| go | go | Built-in |
| cargo | cargo-outdated | `cargo install cargo-outdated` |
| cargo | cargo-edit | `cargo install cargo-edit` |