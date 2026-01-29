# Compatibility Matrix

This document outlines version compatibility requirements, platform support, and feature availability across different environments for SkillSync.

## Table of Contents

- [System Requirements](#system-requirements)
- [Go Version Support](#go-version-support)
- [Platform Support](#platform-support)
- [Operating System Support](#operating-system-support)
- [Skill Format Versions](#skill-format-versions)
- [Feature Compatibility](#feature-compatibility)
- [Installation Method Compatibility](#installation-method-compatibility)
- [Upgrade and Migration Notes](#upgrade-and-migration-notes)
- [Known Issues](#known-issues)

---

## System Requirements

| Component | Minimum Version | Recommended Version | Notes |
|-----------|----------------|---------------------|-------|
| **Go** | 1.22.x | 1.25.4 | Required for building from source or `go install` |
| **Git** | 2.0+ | Latest stable | Required for sync operations |
| **Homebrew** | 3.0+ | Latest stable | macOS/Linux only, for Homebrew installation |

---

## Go Version Support

SkillSync is tested and supported on multiple Go versions to ensure backward compatibility.

| Go Version | Support Status | CI Tested | Notes |
|------------|---------------|-----------|-------|
| **1.25.x** | ✓ Fully Supported | Yes | Current development version (go.mod: 1.25.4) |
| **1.23.x** | ✓ Fully Supported | Yes | Backward compatibility tested in CI |
| **1.22.x** | ✓ Fully Supported | Yes | Minimum supported version |
| **1.21.x and older** | ✗ Not Supported | No | May work but not tested or guaranteed |

**Note**: CI pipeline tests against 1.22.x, 1.23.x, and 1.25.x on every pull request to ensure compatibility.

---

## Platform Support

SkillSync supports three major AI coding platforms with varying levels of feature support.

### Supported Platforms

| Platform | Status | Scope Support | Plugin System | Legacy Format Support |
|----------|--------|---------------|---------------|----------------------|
| **Claude Code** | ✓ Full | repo, user, plugin | ✓ Yes | YAML frontmatter (.md) |
| **Cursor** | ✓ Full | repo, user, system | ✗ No | YAML frontmatter (.md, .mdc), globs/alwaysApply |
| **Codex** | ✓ Full | repo, user, system | ✓ System-level | TOML config, AGENTS.md |

### Platform-Specific Features

#### Claude Code
- **Config directory**: `.claude/` (repo) or `~/.claude/` (user)
- **Skills paths**:
  - Repository: `./.claude/skills/`
  - User: `~/.claude/skills/`
  - Plugin: `~/.claude/plugins/cache/{marketplace}/{plugin}/{version}/`
- **Plugin tracking**: Version-aware plugin installation tracking
- **Special features**: Symlink-aware plugin management

#### Cursor
- **Config directory**: `.cursor/` (repo) or `~/.cursor/` (user)
- **Skills paths**:
  - Repository: `./.cursor/skills/`
  - User: `~/.cursor/skills/`
  - System: `/usr/local/share/cursor/skills/` or similar
- **File extensions**: `.md`, `.mdc`
- **Special features**: `globs` (file pattern matching), `alwaysApply` (global skills)

#### Codex
- **Config directory**: `.codex/` (repo) or `~/.codex/` (user)
- **Skills paths**:
  - Repository: `./.codex/skills/`
  - User: `~/.codex/skills/`
  - System: `/etc/codex/skills/`
- **Config format**: TOML (`config.toml`)
- **Special features**: `AGENTS.md` legacy format support

---

## Operating System Support

| Operating System | Installation Methods | Status | Notes |
|------------------|---------------------|--------|-------|
| **macOS** | Homebrew, Go install, Binary | ✓ Fully Supported | Intel (amd64) and Apple Silicon (arm64) |
| **Linux** | Homebrew, Go install, Binary | ✓ Fully Supported | amd64 and arm64 architectures |
| **Windows** | Go install, Binary | ⚠ Partially Tested | Pre-built binaries available, less tested |

### Architecture Support

- **amd64** (x86_64): Fully supported on all platforms
- **arm64** (Apple Silicon, ARM64): Fully supported on macOS and Linux

---

## Skill Format Versions

SkillSync supports multiple skill format versions with automatic detection and transformation.

### Modern Format: Agent Skills Standard (SKILL.md)

**Precedence**: Highest - Takes precedence over legacy formats

**File pattern**: `SKILL.md` in skill subdirectories

**Supported fields**:
```yaml
---
name: skill-name                    # Required: Unique identifier
description: Brief description      # Required: Human-readable description
scope: user|repo|plugin|system|admin|builtin  # Scope level
disable-model-invocation: false     # Disable AI model invocation
license: MIT                        # License identifier
compatibility:                      # Platform version constraints
  claude-code: ">=1.0.0"
  cursor: ">=0.5.0"
  codex: ">=0.1.0"
scripts:                            # Associated script files
  - setup.sh
  - validate.sh
references:                         # Documentation references
  - docs/guide.md
  - https://example.com
assets:                             # Related asset files
  - templates/config.yaml
tools:                              # Legacy: tool permissions (preserved)
  - Read
  - Write
  - Bash
---

# Skill content in Markdown
...
```

### Legacy Format: Claude Code

**File pattern**: `*.md` files with YAML frontmatter

**Supported fields**:
```yaml
---
name: skill-name
description: Description
tools: ["Read", "Write", "Bash"]
scope: user                         # Optional, explicit scope
---
```

**Conversion**: Automatically converted to SKILL.md format during sync

### Legacy Format: Cursor

**File pattern**: `*.md` or `*.mdc` files with YAML frontmatter

**Cursor-specific fields**:
```yaml
---
name: skill-name
globs: ["*.py", "**/*.py"]         # File pattern matching
alwaysApply: false                  # Boolean: applies to all files
---
```

**Metadata preservation**: `globs` and `alwaysApply` stored in metadata for cross-platform compatibility

### Legacy Format: Codex

**Format 1 - TOML Config** (`config.toml`):
```toml
model = "o4-mini"
approval_policy = "on-failure"
sandbox_mode = "enabled"
instructions = "Follow PEP 8 style guidelines."
```

**Format 2 - AGENTS.md**:
- Plain markdown without frontmatter
- Instructions in file body
- Treated as legacy format

**Conversion**: Codex also supports modern SKILL.md format

### Format Precedence Rules

When multiple format files exist with the same skill name:

1. **SKILL.md** (Agent Skills Standard) - Highest precedence
2. **Legacy platform-specific formats** - Lower precedence
3. **Name collision**: Modern format wins, legacy is skipped

---

## Feature Compatibility

### Skill Scopes by Platform

| Scope | Claude Code | Cursor | Codex | Description |
|-------|-------------|--------|-------|-------------|
| **builtin** | ✓ | ✓ | ✓ | Built-in skills shipping with the platform |
| **system** | ⚠ Limited | ✓ | ✓ | System-wide skills (OS-level) |
| **admin** | ⚠ Limited | ✓ | ✓ | Administrator-defined skills |
| **user** | ✓ | ✓ | ✓ | User-level skills in home directory |
| **repo** | ✓ | ✓ | ✓ | Repository-level skills (project-specific) |
| **plugin** | ✓ | ✗ | ✗ | Plugin-installed skills (Claude Code only) |

**Precedence order** (lowest to highest): builtin → system → admin → user → repo → plugin

### Command Availability

All SkillSync commands work across all supported platforms:

| Command | Claude Code | Cursor | Codex | Notes |
|---------|-------------|--------|-------|-------|
| `skillsync init` | ✓ | ✓ | ✓ | Initialize skill directories |
| `skillsync list` | ✓ | ✓ | ✓ | List skills from all scopes |
| `skillsync sync` | ✓ | ✓ | ✓ | Sync skills between platforms |
| `skillsync diff` | ✓ | ✓ | ✓ | Compare skill differences |
| `skillsync validate` | ✓ | ✓ | ✓ | Validate skill format |
| `skillsync version` | ✓ | ✓ | ✓ | Show version information |

### Sync Strategy Compatibility

All six sync strategies work across all platforms:

- **overwrite**: Always use source version
- **skip**: Keep destination, don't overwrite
- **rename**: Create copy with incremental suffix
- **merge**: Automatic content merge (experimental)
- **prompt**: Manual resolution per conflict
- **skip-newer**: Keep destination if newer timestamp

---

## Installation Method Compatibility

| Installation Method | Supported Platforms | Auto-Update | Version Control | Notes |
|---------------------|---------------------|-------------|-----------------|-------|
| **Homebrew** | macOS, Linux | ✓ Yes (`brew upgrade`) | ✓ Yes | Recommended for macOS/Linux |
| **Go install** | All (with Go) | Manual | ✓ Yes (via `@version`) | Requires Go toolchain |
| **Pre-built binaries** | All | Manual | ✓ Yes | Download from GitHub releases |
| **Source build** | All (with Go) | Manual | Git-based | For contributors |

### Installation Commands by Method

```bash
# Homebrew (macOS/Linux)
brew install klauern/tap/skillsync
brew upgrade skillsync  # Update

# Go install (specific version)
go install github.com/klauern/skillsync@v0.1.0
go install github.com/klauern/skillsync@latest  # Latest

# Pre-built binary (example for Linux amd64)
wget https://github.com/klauern/skillsync/releases/download/v0.1.0/skillsync_Linux_x86_64.tar.gz
tar -xzf skillsync_Linux_x86_64.tar.gz
sudo mv skillsync /usr/local/bin/

# Source build
git clone https://github.com/klauern/skillsync.git
cd skillsync
make build
```

---

## Upgrade and Migration Notes

### Version Upgrade Path

SkillSync follows semantic versioning (SemVer):
- **Major version** (X.0.0): Breaking changes, migration required
- **Minor version** (0.X.0): New features, backward compatible
- **Patch version** (0.0.X): Bug fixes, backward compatible

### Breaking Changes Policy

**Current status**: No breaking changes introduced yet (v0.1.0)

**Future breaking changes will be announced with**:
- GitHub release notes with migration instructions
- Deprecation warnings in the CLI (where applicable)
- Documentation updates in this file
- Minimum one minor version deprecation period before removal

### Configuration Migration

**Backward compatibility maintained**:
- Old `SKILLSYNC_*_PATH` environment variables still supported
- Old `SkillsPath` config field coexists with new `SkillsPaths` list
- Legacy skill formats automatically detected and transformed

**Recommended actions on upgrade**:
1. Run `skillsync version` to verify installation
2. Run `skillsync validate` to check skill format compatibility
3. Review [migration guide](migration.md) for platform-specific considerations
4. Test sync operations with `--dry-run` flag before applying changes

### Skill Format Migration

**Automatic handling**:
- Legacy formats (Claude Code, Cursor, Codex) are preserved during sync
- Platform-specific metadata stored in `Metadata` map for round-trip compatibility
- No manual intervention required for format upgrades

**Manual migration** (optional, for modernization):
- Convert legacy formats to SKILL.md manually if desired
- Use `skillsync validate` to verify SKILL.md format correctness
- Modern format enables better cross-platform compatibility

---

## Known Issues

### Platform-Specific Issues

#### Claude Code
- **Plugin symlinks**: Complex plugin structures may require manual verification
- **Workaround**: Check `~/.claude/plugins/cache/` for symlink integrity

#### Cursor
- **`.mdc` extension**: Less common, ensure Cursor recognizes it
- **Workaround**: Use `.md` extension for better compatibility

#### Codex
- **System-level paths**: May require elevated permissions on some systems
- **Workaround**: Use user-level or repo-level scopes instead

### Cross-Platform Issues

- **Line endings**: Windows CRLF vs Unix LF may cause spurious diffs
  - **Workaround**: Configure Git to normalize line endings (`git config core.autocrlf true`)

- **Path separators**: Windows backslash vs Unix forward slash
  - **Workaround**: SkillSync normalizes paths automatically

- **File permissions**: Unix permission bits not preserved on Windows
  - **Workaround**: Not critical for skill files (text-based)

### Performance Considerations

- **Large repositories**: Syncing 100+ skills may take several seconds
  - **Recommendation**: Use `--dry-run` to preview changes before applying

- **Network latency**: Git operations depend on remote repository speed
  - **Recommendation**: Use local-first workflows, push periodically

---

## See Also

- [Quick Start Guide](quick-start.md) - Getting started with SkillSync
- [Commands Reference](commands.md) - Complete CLI command documentation
- [Migration Guide](migration.md) - Detailed migration instructions
- [Sync Strategies](sync-strategies.md) - Understanding conflict resolution
- [Contributing Guide](contributing.md) - Development and release process
