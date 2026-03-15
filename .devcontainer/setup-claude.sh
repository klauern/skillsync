#!/usr/bin/env bash
# setup-claude.sh — Sync host Claude configs into the container
# Source: /home/dev/.claude-host (readonly bind mount from host ~/.claude)
# Target: /home/dev/.claude (writeable named volume)
set -euo pipefail

HOST_DIR="/home/dev/.claude-host"
TARGET_DIR="/home/dev/.claude"

if [ ! -d "$HOST_DIR" ]; then
    echo "setup-claude: host mount not found at $HOST_DIR, skipping config sync"
    exit 0
fi

echo "setup-claude: syncing configs from host..."

# ── Always-copy files (small, ensures freshness) ─────────────────────
for file in settings.json settings.local.json .mcp.json mcp.json CLAUDE.md; do
    if [ -f "$HOST_DIR/$file" ]; then
        cp "$HOST_DIR/$file" "$TARGET_DIR/$file"
        echo "  copied $file"
    fi
done

# ── Copy-once files (avoid clobbering active auth) ───────────────────
if [ -f "$HOST_DIR/.credentials.json" ] && [ ! -f "$TARGET_DIR/.credentials.json" ]; then
    cp "$HOST_DIR/.credentials.json" "$TARGET_DIR/.credentials.json"
    echo "  copied .credentials.json (first-time)"
fi

# ── Directory sync (full replace) ────────────────────────────────────
for dir in skills commands agents plugins; do
    if [ -d "$HOST_DIR/$dir" ]; then
        rm -rf "$TARGET_DIR/$dir"
        # -rL follows symlinks; broken symlinks (external paths) are skipped gracefully
        cp -rL "$HOST_DIR/$dir" "$TARGET_DIR/$dir" 2>/dev/null || {
            # If cp -rL fails (e.g., all symlinks broken), try without -L
            cp -r "$HOST_DIR/$dir" "$TARGET_DIR/$dir" 2>/dev/null || true
        }
        echo "  synced $dir/"
    fi
done

echo "setup-claude: done"
