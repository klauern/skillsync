# Installation Guide

## Overview

git-trim is distributed as a single binary written in Rust. Multiple installation methods are available depending on your platform and preferences.

## Installation Methods

### macOS (Homebrew)

```bash
brew install foriequal0/git-trim/git-trim
```

Verify installation:
```bash
git-trim --version
```

### Cargo (All Platforms)

If you have Rust installed:

```bash
cargo install git-trim
```

This compiles from source and installs to `~/.cargo/bin/`.

**Dependencies for building**:
- Linux: `libssl-dev`, `pkg-config`
- macOS: OpenSSL (via Homebrew: `brew install openssl`)

### Pre-built Binaries

Download from [GitHub Releases](https://github.com/foriequal0/git-trim/releases):

1. Download the binary for your platform:
   - `git-trim-linux-x86_64`
   - `git-trim-darwin-x86_64` (Intel Mac)
   - `git-trim-darwin-aarch64` (Apple Silicon)
   - `git-trim-windows-x86_64.exe`

2. Make executable and move to PATH:
   ```bash
   chmod +x git-trim-*
   sudo mv git-trim-* /usr/local/bin/git-trim
   ```

3. Verify:
   ```bash
   git-trim --version
   ```

## Verification

After installation, verify git-trim works:

```bash
# Check version
git-trim --version

# Show help
git-trim --help

# Dry run in current repo
git-trim --dry-run
```

## Troubleshooting

### Command not found

If `git-trim` is not in PATH:

```bash
# Find where it's installed
which git-trim

# Add to PATH in ~/.bashrc or ~/.zshrc
export PATH="$HOME/.cargo/bin:$PATH"
```

### OpenSSL errors (Linux)

Install development libraries:

```bash
# Ubuntu/Debian
sudo apt-get install libssl-dev pkg-config

# Fedora/RHEL
sudo dnf install openssl-devel

# Arch
sudo pacman -S openssl pkg-config
```

### Permission denied

Binary needs execute permissions:

```bash
chmod +x $(which git-trim)
```

## Next Steps

Once installed, configure git-trim:
- [Configuration Guide](configuration.md)
- [Example Workflows](example_workflows.md)