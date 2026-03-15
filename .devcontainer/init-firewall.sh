#!/usr/bin/env bash
# init-firewall.sh — Default-deny outbound firewall with allowlist for Go/Claude ecosystem
# Adapted from https://github.com/anthropics/claude-code/tree/main/.devcontainer
set -euo pipefail

# ── Allowlisted domains ─────────────────────────────────────────────
ALLOWED_DOMAINS=(
    # Claude / Anthropic
    api.anthropic.com
    sentry.io
    statsig.anthropic.com
    statsig.com

    # npm registry (Claude Code dependencies)
    registry.npmjs.org

    # Bun (runtime for MCP servers)
    bun.sh
    install.bun.sh

    # VS Code extensions / marketplace
    marketplace.visualstudio.com
    vscode.blob.core.windows.net
    update.code.visualstudio.com

    # Go ecosystem
    proxy.golang.org
    sum.golang.org
    storage.googleapis.com

    # GitHub (releases, raw content, API)
    github.com
    api.github.com
    objects.githubusercontent.com
    raw.githubusercontent.com

    # Claude Code install
    claude.ai
)

echo "==> Initializing firewall allowlist..."

# ── Create ipset for allowed IPs ────────────────────────────────────
ipset create allowed_ips hash:net -exist
ipset flush allowed_ips

# Resolve each domain and add IPs to the set
for domain in "${ALLOWED_DOMAINS[@]}"; do
    ips=$(dig +short "$domain" A 2>/dev/null | grep -E '^[0-9]+\.' || true)
    for ip in $ips; do
        ipset add allowed_ips "$ip/32" -exist
    done
done

# ── GitHub IP ranges (from API meta endpoint) ───────────────────────
echo "==> Fetching GitHub IP ranges..."
GH_META=$(curl -fsSL https://api.github.com/meta 2>/dev/null || echo '{}')
for key in web api git packages; do
    cidrs=$(echo "$GH_META" | jq -r ".${key}[]? // empty" 2>/dev/null || true)
    for cidr in $cidrs; do
        # Only add IPv4 CIDRs
        if [[ "$cidr" =~ ^[0-9]+\. ]]; then
            ipset add allowed_ips "$cidr" -exist
        fi
    done
done

# Aggregate CIDRs to reduce rule count (if aggregate is available)
if command -v aggregate &>/dev/null; then
    AGGREGATED=$(ipset list allowed_ips | grep -E '^[0-9]' | aggregate 2>/dev/null || true)
    if [ -n "$AGGREGATED" ]; then
        ipset create allowed_ips_agg hash:net -exist
        ipset flush allowed_ips_agg
        while IFS= read -r cidr; do
            ipset add allowed_ips_agg "$cidr" -exist
        done <<< "$AGGREGATED"
        ipset swap allowed_ips_agg allowed_ips
        ipset destroy allowed_ips_agg 2>/dev/null || true
    fi
fi

# ── iptables rules ──────────────────────────────────────────────────
echo "==> Applying iptables rules..."

# Flush existing OUTPUT rules
iptables -F OUTPUT

# Allow loopback
iptables -A OUTPUT -o lo -j ACCEPT

# Allow established/related connections
iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Allow DNS (UDP and TCP port 53)
iptables -A OUTPUT -p udp --dport 53 -j ACCEPT
iptables -A OUTPUT -p tcp --dport 53 -j ACCEPT

# Allow SSH (port 22)
iptables -A OUTPUT -p tcp --dport 22 -j ACCEPT

# Allow connections to Docker host network (common gateway ranges)
iptables -A OUTPUT -d 172.16.0.0/12 -j ACCEPT
iptables -A OUTPUT -d 192.168.0.0/16 -j ACCEPT
iptables -A OUTPUT -d 10.0.0.0/8 -j ACCEPT

# Allow allowlisted IPs (HTTP/HTTPS)
iptables -A OUTPUT -p tcp --dport 80 -m set --match-set allowed_ips dst -j ACCEPT
iptables -A OUTPUT -p tcp --dport 443 -m set --match-set allowed_ips dst -j ACCEPT

# Default deny outbound
iptables -A OUTPUT -p tcp --dport 80 -j REJECT
iptables -A OUTPUT -p tcp --dport 443 -j REJECT

# ── Verification ────────────────────────────────────────────────────
echo "==> Verifying firewall rules..."

ERRORS=0

# Blocked: example.com should fail
if curl -sf --max-time 5 https://example.com >/dev/null 2>&1; then
    echo "  FAIL: example.com should be blocked"
    ERRORS=$((ERRORS + 1))
else
    echo "  OK: example.com is blocked"
fi

# Allowed: api.github.com should succeed
if curl -sf --max-time 10 https://api.github.com >/dev/null 2>&1; then
    echo "  OK: api.github.com is reachable"
else
    echo "  WARN: api.github.com may not be reachable (could be transient)"
fi

# Allowed: proxy.golang.org should succeed
if curl -sf --max-time 10 https://proxy.golang.org >/dev/null 2>&1; then
    echo "  OK: proxy.golang.org is reachable"
else
    echo "  WARN: proxy.golang.org may not be reachable (could be transient)"
fi

# Allowed: registry.npmjs.org should succeed (needed by bun)
if curl -sf --max-time 10 https://registry.npmjs.org >/dev/null 2>&1; then
    echo "  OK: registry.npmjs.org is reachable"
else
    echo "  WARN: registry.npmjs.org may not be reachable (could be transient)"
fi

echo "==> Firewall initialized ($(ipset list allowed_ips | grep -c '^[0-9]') CIDRs allowlisted)"
