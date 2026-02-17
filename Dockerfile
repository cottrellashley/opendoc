# ──────────────────────────────────────────────────────────────
# OpenDoc — Single Go binary with embedded assets
# ──────────────────────────────────────────────────────────────

# ── Stage 1: Build the Go binary ─────────────────────────────
FROM golang:1.26 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /opendoc ./cmd/opendoc

# ── Stage 2: Runtime image ───────────────────────────────────
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=C.UTF-8
ENV LC_ALL=C.UTF-8

# Basic tools (kept minimal)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    ca-certificates \
    ttyd \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI (optional, for AI terminal usage)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && npm install -g @anthropic-ai/claude-code \
    && rm -rf /var/lib/apt/lists/*

# Copy the single Go binary
COPY --from=builder /opendoc /usr/local/bin/opendoc

# Workspace
RUN mkdir -p /workspace
WORKDIR /workspace

# The Go binary serves everything — no nginx, no supervisord needed
EXPOSE 3000

ENTRYPOINT ["opendoc"]
CMD ["workbench", "/workspace", "-p", "3000"]
