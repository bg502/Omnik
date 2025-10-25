# Omnik Architecture

**Technical Overview of Telegram Bot for Claude Code**

This document provides a detailed technical overview of Omnik's architecture, design decisions, and implementation details.

---

## Table of Contents

1. [High-Level Architecture](#high-level-architecture)
2. [Container Architecture](#container-architecture)
3. [Session Management](#session-management)
4. [Claude CLI Integration](#claude-cli-integration)
5. [Message Flow](#message-flow)
6. [Data Persistence](#data-persistence)
7. [Security Model](#security-model)
8. [Design Decisions](#design-decisions)

---

## High-Level Architecture

Omnik uses a **unified container architecture** where all components run in a single Docker container:

```
┌─────────────────────────────────────────────────┐
│           Docker Container (omnik)               │
│                                                  │
│  ┌────────────────────────────────────────┐    │
│  │         Go Telegram Bot                │    │
│  │  - Telegram API client                 │    │
│  │  - Command routing                     │    │
│  │  - Session manager                     │    │
│  │  - Direct command execution            │    │
│  └──────────────┬─────────────────────────┘    │
│                 │                               │
│                 │ exec.Command()                │
│                 ▼                               │
│  ┌────────────────────────────────────────┐    │
│  │         Claude CLI                     │    │
│  │  - Claude Code v2.0.27                 │    │
│  │  - Stream JSON output                  │    │
│  │  - Session resumption (--resume)       │    │
│  │  - Tool execution (Bash, Read, etc.)   │    │
│  │  - Docker & Git access                 │    │
│  └────────────────────────────────────────┘    │
│                                                  │
│  ┌────────────────────────────────────────┐    │
│  │         Shared Filesystem              │    │
│  │  /workspace - User files               │    │
│  │  /home/node/.claude - Auth & sessions  │    │
│  │  /workspace/.omnik-sessions.json       │    │
│  │  /var/run/docker.sock (mounted)        │    │
│  └────────────────────────────────────────┘    │
│                                                  │
└─────────────────────────────────────────────────┘
         ▲                           │
         │                           │
    Telegram API              Anthropic API
```

### Component Responsibilities

**Go Telegram Bot** (`go-bot/`)
- Handles Telegram API communication
- Authenticates users via whitelist
- Routes commands and messages
- Manages sessions and working directories
- Executes direct filesystem commands

**Claude CLI** (Node.js + @anthropic-ai/claude-code)
- Provides AI coding assistance
- Executes tool calls (Bash, Read, Write, Edit, etc.)
- Maintains conversation context across turns
- Streams responses in real-time

**Session Manager** (`go-bot/internal/session/`)
- Persists session metadata
- Tracks working directories per session
- Maps session names to Claude conversation IDs
- Handles session switching

---

## Container Architecture

### Multi-Stage Docker Build

The `Dockerfile` uses a two-stage build process:

#### Stage 1: Go Builder
```dockerfile
FROM golang:1.21-bookworm AS go-builder
WORKDIR /app
COPY go-bot/ ./
RUN go build -o /omnik-bot ./cmd/main.go
```

**Purpose**: Compile the Go binary without including build tools in the final image.

#### Stage 2: Runtime
```dockerfile
FROM node:20-bookworm-slim
RUN apt-get update && apt-get install -y docker-ce-cli docker-compose-plugin git
RUN npm install -g @anthropic-ai/claude-code
RUN usermod -aG docker node
COPY --from=go-builder /omnik-bot /app/omnik-bot
USER node
CMD ["/app/omnik-bot"]
```

**Purpose**: Minimal runtime with Node.js, Docker tools, Git, and Claude CLI. The `node` user is added to the docker group for socket access.

### Why Unified Container?

**Previous Architecture Issues:**
- Go bot in one container, Claude SDK in another
- Filesystem isolation prevented Claude from accessing workspace
- Complex networking for inter-container communication
- No bash in Node container prevented command execution

**Unified Solution Benefits:**
- Single filesystem for both bot and Claude
- Direct execution of Claude CLI via Go's `exec.Command()`
- Shared `/workspace` accessible by both components
- Claude can execute Bash commands in the same environment

---

## Session Management

### Session Data Model

```go
type Session struct {
    ID          string    `json:"id"`           // Claude conversation ID (UUID)
    Name        string    `json:"name"`         // User-friendly name
    WorkingDir  string    `json:"working_dir"`  // Absolute path
    CreatedAt   time.Time `json:"created_at"`
    LastUsedAt  time.Time `json:"last_used_at"`
    Description string    `json:"description,omitempty"`
}
```

### Session Manager Operations

**Create Session:**
1. Generate unique name-based identifier
2. Initialize with `/workspace` as working directory
3. Save to persistence layer
4. Return session object (Claude ID assigned on first query)

**Switch Session:**
1. Look up session by name or ID
2. Update bot's working directory to session's `WorkingDir`
3. Mark session as last used
4. Return session object for Claude `--resume`

**Update Working Directory:**
1. Validate path exists
2. Update current session's `WorkingDir`
3. Save to persistence
4. Update bot's in-memory `workingDir`

### Session Persistence

**Storage Location:** `/workspace/.omnik-sessions.json`

**Format:**
```json
{
  "sessions": {
    "myproject": {
      "id": "35f966d7-7d2e-4343-8e65-c947f21b36c1",
      "name": "myproject",
      "working_dir": "/workspace/projects/scraper",
      "created_at": "2025-10-25T10:30:00Z",
      "last_used_at": "2025-10-25T11:45:00Z",
      "description": "Building a web scraper"
    }
  },
  "current_session": "myproject"
}
```

**Lifecycle:**
- Loaded on bot startup
- Saved after every modification
- Persists across container restarts via volume mount

---

## Claude CLI Integration

### CLI Client Architecture

**Location:** `go-bot/internal/claude/cli.go`

**Key Components:**

```go
type CLIClient struct {
    model          string  // e.g., "sonnet"
    permissionMode string  // e.g., "bypassPermissions"
}

func (c *CLIClient) Query(ctx context.Context, req QueryRequest) (
    <-chan StreamResponse, <-chan error)
```

### Command Execution

**Arguments:**
```bash
claude \
  --print \
  --output-format stream-json \
  --verbose \
  --permission-mode bypassPermissions \
  --allowed-tools Bash Read Write Edit Glob Grep \
  --resume <session-id> \
  --model sonnet \
  "<user prompt>"
```

**Flags Explained:**
- `--print`: Output responses to stdout instead of terminal UI
- `--output-format stream-json`: Emit JSON objects per line
- `--verbose`: Include all message types (required for stream-json)
- `--permission-mode bypassPermissions`: Auto-approve tool use
- `--allowed-tools`: Explicitly permit tool execution
- `--resume`: Continue existing conversation (uses session ID)
- `--model`: Select Claude model (sonnet, opus, haiku)

### Streaming Response Parsing

**Process:**
1. Execute `claude` command via `exec.CommandContext`
2. Set working directory via `cmd.Dir = req.Workspace`
3. Create stdout pipe for streaming JSON
4. Scan line-by-line with `bufio.Scanner`
5. Parse each line as JSON message
6. Convert to `StreamResponse` and send to channel
7. Send `done` signal when process completes

**Message Format:**
```json
{
  "type": "message_start",
  "message": {...}
}
{
  "type": "content_block_delta",
  "delta": {"type": "text_delta", "text": "Hello"}
}
{
  "type": "message_stop"
}
```

### Session Continuity

**How --resume Works:**
- Claude CLI maintains session state in `/home/node/.claude/`
- Session ID (UUID) identifies conversation thread
- `--resume <id>` loads previous context and continues
- No session ID limit - can maintain unlimited parallel sessions
- Each session has independent conversation history

---

## Message Flow

### User Message → Claude Response

```
1. User sends message in Telegram
   │
   ▼
2. Telegram API delivers update to bot
   │
   ▼
3. Bot validates user is authorized
   │
   ▼
4. Bot checks if command or chat message
   │
   ├─── Command → Execute directly (e.g., /ls, /pwd)
   │
   └─── Chat message → Route to Claude
        │
        ▼
5. Bot retrieves current session
   │
   ▼
6. Bot calls CLIClient.Query() with:
   - Prompt: user message
   - SessionID: current session's Claude ID
   - Workspace: current session's working directory
   │
   ▼
7. CLIClient executes `claude --resume <id> ...`
   │
   ▼
8. Claude CLI processes request:
   - Loads conversation context
   - Generates response
   - May execute tools (Bash, Read, Write, etc.)
   - Streams JSON output
   │
   ▼
9. CLIClient parses streaming JSON
   │
   ▼
10. Bot accumulates text deltas
    │
    ▼
11. Bot sends buffered text to Telegram
    │
    ▼
12. Bot updates session's last used timestamp
    │
    ▼
13. Bot saves session state to disk
```

### Direct Command Execution

```
1. User sends /ls or /cd or /pwd or /cat or /exec
   │
   ▼
2. Bot parses command and arguments
   │
   ▼
3. Bot executes via exec.Command():
   - Command: ls, cd, pwd, cat, or user command
   - Dir: current working directory
   │
   ▼
4. Bot captures stdout/stderr
   │
   ▼
5. Bot sends output to Telegram
   │
   ▼
6. For /cd: Bot updates session working directory
```

---

## Data Persistence

### Volume Mounts

**Workspace Volume** (`workspace:/workspace`)
- User files and projects
- Session metadata (`.omnik-sessions.json`)
- Accessible by both bot and Claude
- Persists across container restarts

**Claude Auth Volume** (`claude-auth:/home/node/.claude`)
- Claude CLI authentication token
- Claude session state and conversation history
- Persists across container restarts
- Required for `--resume` to work

### File Paths

| Path | Purpose | Owner | Permissions |
|------|---------|-------|-------------|
| `/workspace` | User workspace | node:node | rwxr-xr-x |
| `/workspace/.omnik-sessions.json` | Session metadata | node:node | rw-r--r-- |
| `/home/node/.claude` | Claude auth & sessions | node:node | rwx------ |
| `/app/omnik-bot` | Go binary | node:node | rwxr-xr-x |

---

## Security Model

### Authentication

**Whitelist-Based:**
- Single `AUTHORIZED_USER_ID` environment variable
- Bot checks every update against this ID
- Unauthorized users receive no response
- No rate limiting (single user only)

**Implementation:**
```go
if update.Message.From.ID != b.authorizedUID {
    log.Printf("Unauthorized access attempt from user %d", update.Message.From.ID)
    continue
}
```

### Containerization

**Non-Root Execution:**
- Container runs as `node` user (UID 1000, GID 1000)
- No sudo or privilege escalation
- Limited to container filesystem

**Permission Boundaries:**
- Claude runs with `bypassPermissions` mode
- Tool execution limited to allowed list
- Filesystem access restricted to `/workspace`

**Network Isolation:**
- Bridge network (`omnik-net`)
- Only outbound connections to Telegram and Anthropic APIs
- No exposed ports (polling mode for Telegram)

### Resource Limits

**CPU and Memory:**
```yaml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 2G
    reservations:
      cpus: '0.5'
      memory: 256M
```

**Purpose:**
- Prevent resource exhaustion
- Ensure responsive bot performance
- Limit Claude execution impact

---

## Design Decisions

### Why Go for the Bot?

**Advantages:**
- Fast, compiled binary
- Excellent concurrency with goroutines
- Strong standard library for process execution
- Small memory footprint
- Easy deployment (single binary)

**Alternative Considered:** Pure Node.js
- **Rejected because:** Heavier runtime, slower execution, less efficient process management

### Why Claude CLI Instead of SDK?

**CLI Advantages:**
- Official tool with full feature support
- Reliable session management via `--resume`
- Streaming output via `--output-format stream-json`
- No need to maintain SDK wrapper logic
- Simpler error handling

**SDK Disadvantages (attempted in earlier versions):**
- Complex Node.js child process management
- Additional serialization layer
- More failure points (Go → Node → SDK → API)

### Why Unified Container?

**Previous Multi-Container Issues:**
- Filesystem isolation prevented file sharing
- Complex volume mounts and networking
- Claude couldn't access bot's workspace
- No bash in Node container

**Unified Benefits:**
- Single shared filesystem
- Direct process execution
- Simpler deployment (one service)
- Reduced resource overhead
- Easier debugging

### Why Session Persistence in JSON?

**Advantages:**
- Human-readable format
- Easy to inspect and debug
- No database dependency
- Simple backup/restore (copy file)
- Fast read/write for small datasets

**Alternative Considered:** SQLite
- **Rejected because:** Overkill for single-user, small dataset; adds dependency

---

## Component Reference

### File Structure

```
omnik/
├── go-bot/
│   ├── cmd/
│   │   └── main.go              # Entry point
│   ├── internal/
│   │   ├── bot/
│   │   │   └── bot.go           # Telegram bot logic
│   │   ├── claude/
│   │   │   └── cli.go           # Claude CLI client
│   │   └── session/
│   │       └── manager.go       # Session management
│   ├── go.mod                   # Go dependencies
│   └── go.sum
├── Dockerfile                   # Container build
├── docker-compose.yml           # Service definition
├── .env.example                 # Environment template
├── README.md                    # User documentation
├── CHANGELOG.md                 # Release history
└── ARCHITECTURE.md              # This file
```

### Dependencies

**Go Packages:**
- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram client

**Node.js Packages:**
- `@anthropic-ai/claude-code@2.0.27` - Claude CLI

**System Packages:**
- `bash` - Command execution
- `git` - Version control
- `docker-ce-cli` - Docker command-line client
- `docker-compose-plugin` - Docker Compose v2
- `ca-certificates` - HTTPS support

---

## Future Enhancements

### Planned for v1.1

**Multiple User Support:**
- Extend whitelist to array of user IDs
- Per-user session isolation
- Shared workspace or user-specific workspaces

**File Upload:**
- Handle Telegram document uploads
- Save to current working directory
- Automatically mention uploaded file to Claude

**Enhanced Error Handling:**
- Better error messages for common issues
- Retry logic for transient failures
- Graceful degradation

### Planned for v2.0

**Web UI:**
- Browser-based session management
- Real-time conversation viewer
- File browser for workspace

**Multi-Model Support:**
- Dynamic model selection (Opus, Haiku, Sonnet)
- Per-session model preference
- Cost tracking

**Collaborative Sessions:**
- Multiple users in same session
- Session sharing and permissions
- Concurrent conversation handling

---

## Troubleshooting Reference

### Common Issues

**Bot not responding:**
- Check container is running: `docker compose ps`
- Verify environment variables in `.env`
- Check logs: `docker compose logs -f omnik`

**Claude authentication failed:**
- Re-run: `docker compose run --rm omnik claude setup-token`
- Verify `ANTHROPIC_API_KEY` is set

**Session not persisting:**
- Ensure `/workspace` volume exists
- Check file permissions on `.omnik-sessions.json`
- Verify container user is `node`

**Tool execution blocked:**
- Check `--allowed-tools` list in `cli.go:43`
- Verify `--permission-mode` is set to `bypassPermissions`
- Review Claude CLI logs in stderr

---

## Performance Characteristics

**Response Times:**
- Direct commands (/ls, /pwd): < 100ms
- Claude query (simple): 2-5 seconds
- Claude query (complex with tools): 10-30 seconds

**Memory Usage:**
- Idle: ~250 MB
- Active conversation: ~500 MB
- Peak (multiple tool executions): ~1.5 GB

**Concurrent Sessions:**
- Unlimited sessions (limited by disk space)
- Single active conversation at a time (Telegram UI constraint)
- Fast session switching (< 1 second)

---

**Document Version:** 1.0
**Last Updated:** 2025-10-25
**Omnik Version:** 1.0.0
