FROM golang:1.25-bookworm

ARG JUST_VERSION=1.46.0
ARG DELTA_VERSION=0.18.2

# ── System packages ──────────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
        git zsh fzf vim nano jq curl wget unzip sudo \
        iptables ipset iproute2 dnsutils aggregate \
        fd-find ripgrep bat \
        ca-certificates gnupg \
    && rm -rf /var/lib/apt/lists/*

# Debian uses different binary names for fd, bat
RUN ln -sf /usr/bin/fdfind /usr/local/bin/fd \
    && ln -sf /usr/bin/batcat /usr/local/bin/bat

# ── Node.js 20 (needed for some Claude Code dependencies) ───────────
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# ── just (command runner) ────────────────────────────────────────────
RUN ARCH=$(dpkg --print-architecture) \
    && case "${ARCH}" in \
        amd64) JUST_ARCH="x86_64-unknown-linux-musl" ;; \
        arm64) JUST_ARCH="aarch64-unknown-linux-musl" ;; \
        *) echo "Unsupported arch: ${ARCH}" && exit 1 ;; \
    esac \
    && curl -fsSL "https://github.com/casey/just/releases/download/${JUST_VERSION}/just-${JUST_VERSION}-${JUST_ARCH}.tar.gz" \
        | tar xz -C /usr/local/bin just

# ── git-delta ────────────────────────────────────────────────────────
RUN ARCH=$(dpkg --print-architecture) \
    && curl -fsSL "https://github.com/dandavison/delta/releases/download/${DELTA_VERSION}/git-delta_${DELTA_VERSION}_${ARCH}.deb" \
        -o /tmp/delta.deb \
    && dpkg -i /tmp/delta.deb \
    && rm /tmp/delta.deb

# ── Non-root user ───────────────────────────────────────────────────
RUN groupadd -g 1000 dev \
    && useradd -m -u 1000 -g dev -s /bin/zsh dev \
    && echo "dev ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/dev

# ── Directory setup ─────────────────────────────────────────────────
RUN mkdir -p /workspace /commandhistory \
    && mkdir -p /home/dev/go/pkg/mod /home/dev/.cache/go-build /home/dev/.claude \
    && chown -R dev:dev /home/dev /workspace /commandhistory

# ── Firewall script ─────────────────────────────────────────────────
COPY init-firewall.sh /usr/local/bin/init-firewall.sh
RUN chmod +x /usr/local/bin/init-firewall.sh \
    && echo "dev ALL=(ALL) NOPASSWD: /usr/local/bin/init-firewall.sh" >> /etc/sudoers.d/dev

# ── Switch to dev user for tool installs ─────────────────────────────
USER dev
WORKDIR /home/dev

ENV GOPATH=/home/dev/go \
    GOMODCACHE=/home/dev/go/pkg/mod \
    GOPROXY=https://proxy.golang.org,direct \
    SHELL=/bin/zsh \
    DEVCONTAINER=true \
    PATH="/home/dev/.bun/bin:/home/dev/go/bin:/home/dev/.npm-global/bin:/home/dev/.local/bin:${PATH}"

# ── Go tools ─────────────────────────────────────────────────────────
RUN go install mvdan.cc/gofumpt@latest \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && go install golang.org/x/vuln/cmd/govulncheck@latest \
    && go install golang.org/x/tools/gopls@latest

# golangci-lint v2 via official install script
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
    | sh -s -- -b "$(go env GOPATH)/bin"

# ── Bun (needed for MCP servers) ────────────────────────────────────
RUN curl -fsSL https://bun.sh/install | bash

# ── Claude Code ──────────────────────────────────────────────────────
RUN curl -fsSL https://claude.ai/install.sh | bash

# ── ZSH setup ────────────────────────────────────────────────────────
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended \
    && git clone --depth=1 https://github.com/romkatv/powerlevel10k.git "${ZSH_CUSTOM:-$HOME/.oh-my-zsh/custom}/themes/powerlevel10k" \
    && sed -i 's|ZSH_THEME="robbyrussell"|ZSH_THEME="powerlevel10k/powerlevel10k"|' ~/.zshrc

RUN cat >> ~/.zshrc <<'ZSHEOF'

# Go
export GOPATH=/home/dev/go
export PATH="$HOME/.bun/bin:$GOPATH/bin:$HOME/.npm-global/bin:$HOME/.local/bin:$PATH"

# FZF
[ -f /usr/share/doc/fzf/examples/key-bindings.zsh ] && source /usr/share/doc/fzf/examples/key-bindings.zsh
[ -f /usr/share/doc/fzf/examples/completion.zsh ] && source /usr/share/doc/fzf/examples/completion.zsh

# History (shared via volume)
HISTFILE=/commandhistory/.zsh_history
HISTSIZE=10000
SAVEHIST=10000
ZSHEOF

WORKDIR /workspace
