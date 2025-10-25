# Multi-stage build for omnik Claude Bridge
FROM node:20-alpine AS builder

# Install build dependencies
RUN apk add --no-cache python3 make g++

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install --production=false

# Copy source code
COPY tsconfig.json ./
COPY src/ ./src/

# Build TypeScript
RUN npm run build

# Production image
FROM node:20-alpine

# Install Claude Code CLI globally and runtime dependencies
RUN npm install -g @anthropic-ai/claude-code && \
    apk add --no-cache tini su-exec

# Use existing node user (UID 1000 already exists in node:20-alpine)
WORKDIR /app

# Copy built application from builder
COPY --from=builder --chown=node:node /app/dist ./dist
COPY --from=builder --chown=node:node /app/node_modules ./node_modules
COPY --from=builder --chown=node:node /app/package.json ./

# Copy entrypoint script
COPY --chmod=755 docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# Create directories for Claude auth and workspace
RUN mkdir -p /home/node/.claude /workspace && \
    chown -R node:node /home/node/.claude /workspace

# Configure Claude CLI installation mode
RUN mkdir -p /home/node/.claude && \
    echo '{"installationType":"npm-global"}' > /home/node/.claude/.installation-config.json && \
    chown -R node:node /home/node/.claude

# Switch to non-root user
USER node

# Set environment to ensure Claude CLI uses correct paths
ENV CLAUDE_CLI_CONFIG_DIR=/home/node/.claude

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD node -e "require('http').get('http://localhost:9000/health', (res) => { process.exit(res.statusCode === 200 ? 0 : 1); }).on('error', () => process.exit(1));"

# Expose port
EXPOSE 9000

# Use tini as PID 1 for proper signal handling, with entrypoint wrapper
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/docker-entrypoint.sh"]

# Start the server
CMD ["node", "dist/server.js"]
