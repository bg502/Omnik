# Multi-stage build for unified omnik bot with Go + Node.js + Claude Code
# This container runs both the Telegram bot and Claude Code SDK in one place

# Stage 1: Build Go application
FROM golang:1.21-bookworm AS go-builder

WORKDIR /app

# Copy Go module files
COPY go-bot/go.mod ./

# Copy source code
COPY go-bot/cmd/ ./cmd/
COPY go-bot/internal/ ./internal/

# Download dependencies and build
RUN go mod tidy && go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /omnik-bot ./cmd/main.go

# Stage 2: Final runtime image with Node.js, Go binary, and Claude Code
FROM node:20-bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
SHELL ["/bin/bash", "-lc"]

# Install system dependencies including Docker
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    wget \
    git \
    tini \
    bash \
    gnupg \
    lsb-release \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
      $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
      docker-ce-cli \
      docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI globally
RUN npm install -g @anthropic-ai/claude-code

# Use existing node user from base image (UID/GID 1000)
ARG USER=node

# Add node user to docker group (GID 999 is standard for docker group)
RUN groupadd -g 999 docker || true \
 && usermod -aG docker ${USER}

# Set up directories
WORKDIR /app
RUN mkdir -p /workspace /home/${USER}/.claude \
 && chown -R ${USER}:${USER} /app /workspace /home/${USER}/.claude

# Copy Go binary from builder
COPY --from=go-builder --chown=${USER}:${USER} /omnik-bot /app/omnik-bot

# Copy git credential helper script, entrypoint, and bashrc template
COPY --chown=${USER}:${USER} git-credential-helper.sh /app/git-credential-helper.sh
COPY --chown=${USER}:${USER} entrypoint.sh /app/entrypoint.sh
COPY --chown=${USER}:${USER} bashrc-node /app/.bashrc-template
RUN chmod +x /app/git-credential-helper.sh /app/entrypoint.sh

# Configure Claude CLI
RUN echo '{"installationType":"npm-global"}' > /home/${USER}/.claude/.installation-config.json \
 && chown ${USER}:${USER} /home/${USER}/.claude/.installation-config.json

# Switch to application user
USER ${USER}
ENV HOME=/home/${USER}

# Configure git to use credential helper and set user info from env vars
RUN git config --global credential.helper "/app/git-credential-helper.sh" \
 && git config --global credential.useHttpPath true

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD pgrep -f omnik-bot || exit 1

# Use tini as PID 1 for proper signal handling, with entrypoint script
ENTRYPOINT ["/usr/bin/tini", "--", "/app/entrypoint.sh"]

# Start the Go bot (which will spawn Claude SDK as needed)
CMD ["/app/omnik-bot"]
