# omnik Claude Bridge

Node.js/TypeScript service that provides HTTP/SSE API for Claude Code SDK integration.

## Overview

This service acts as a bridge between the Go/Gin bot and the Claude Code SDK, providing:
- **Native SDK Integration**: Uses `@anthropic-ai/claude-code` for structured interaction
- **Server-Sent Events**: Streams Claude responses in real-time
- **Session Continuity**: Supports resuming conversations via session IDs
- **Subscription Auth**: Works with Anthropic Pro account (no API key needed)

## API Endpoints

### Health Check
```bash
GET /health

Response:
{
  "status": "ok",
  "version": "1.0.0",
  "claudeVersion": "2.0.26"
}
```

### Query Claude
```bash
POST /api/query
Content-Type: application/json

Request:
{
  "prompt": "Create a hello world Python script",
  "sessionId": "optional-session-id",
  "workspace": "/workspace/abc123",
  "permissionMode": "default",
  "allowedTools": ["read", "write"]
}

Response: text/event-stream
data: {"type":"claude_message","data":{...}}

data: {"type":"claude_message","data":{...}}

data: {"type":"done"}
```

## Development

### Prerequisites
- Node.js 20+
- Claude CLI installed and authenticated

### Setup
```bash
# Install dependencies
npm install

# Run in development mode (with auto-reload)
npm run dev

# Build
npm run build

# Run production build
npm start
```

### Environment Variables
```bash
PORT=9000                    # Server port (default: 9000)
HOST=0.0.0.0                # Bind address (default: 0.0.0.0)
CLAUDE_PATH=/path/to/claude  # Optional: custom Claude CLI path
```

## Docker

### Build
```bash
docker build -t omnik-claude-bridge .
```

### Run
```bash
docker run -d \
  --name claude-bridge \
  -p 9000:9000 \
  -v claude-auth:/home/nodeuser/.claude \
  -v workspace:/workspace \
  omnik-claude-bridge
```

### Authenticate Claude
```bash
# Run authentication in container
docker exec -it claude-bridge claude auth login
```

## Testing

### Test Health Endpoint
```bash
curl http://localhost:9000/health
```

### Test Query Endpoint
```bash
curl -X POST http://localhost:9000/api/query \
  -H "Content-Type: application/json" \
  -d '{"prompt": "What is 2+2?"}'
```

## Architecture

```
Go Bot Service
     │
     ├─ HTTP POST /api/query
     │  with QueryRequest
     │
     ▼
Node.js Bridge (this service)
     │
     ├─ @anthropic-ai/claude-code SDK
     │  query({ prompt, options })
     │
     ▼
Claude CLI
     │
     └─ ~/.claude/.credentials.json
        (OAuth tokens from Pro subscription)
```

## Error Handling

The bridge returns structured errors via SSE:

```json
{
  "type": "error",
  "error": "Authentication failed",
  "code": "AUTHENTICATION_ERROR"
}
```

Error codes:
- `INVALID_REQUEST` - Missing or invalid request parameters
- `CLAUDE_NOT_FOUND` - Claude CLI not found in PATH
- `CLAUDE_EXEC_ERROR` - Claude CLI execution failed
- `AUTHENTICATION_ERROR` - Not authenticated with Claude
- `WORKSPACE_ERROR` - Workspace directory issues
- `INTERNAL_ERROR` - Unexpected server error

## Performance

- **Latency**: <50ms overhead for message forwarding
- **Streaming**: Messages forwarded in real-time as SDK yields them
- **Memory**: ~50MB baseline, scales with concurrent queries
- **Concurrency**: Handles multiple simultaneous query streams

## Security

- Runs as non-root user (UID 1000)
- No Docker socket access
- Workspace isolation via volume mounts
- Uses Claude's built-in permission system
- No sensitive data in logs (prompts truncated)

## Troubleshooting

### "Claude CLI not found"
Ensure Claude is installed:
```bash
npm install -g @anthropic-ai/claude-code
claude --version
```

### "Authentication required"
Authenticate Claude:
```bash
claude auth login
# Or in Docker:
docker exec -it claude-bridge claude auth login
```

### "Workspace permission denied"
Ensure workspace volume has correct permissions:
```bash
docker run --user 1000:1000 ...
```

## License

MIT
