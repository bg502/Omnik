# Multi-stage build for Go bot
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod ./

# Copy source code (needed for go mod tidy)
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Download dependencies and generate go.sum
RUN go mod tidy && go mod download

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /omnik-bot ./cmd/main.go

# Production image
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tini

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -u 1000 -G appuser -s /bin/sh -D appuser

WORKDIR /app

# Copy binary from builder
COPY --from=builder --chown=appuser:appuser /omnik-bot /app/omnik-bot

# Create workspace directory
RUN mkdir -p /workspace && chown appuser:appuser /workspace

USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD pgrep -f omnik-bot || exit 1

# Use tini as PID 1
ENTRYPOINT ["/sbin/tini", "--"]

CMD ["/app/omnik-bot"]
