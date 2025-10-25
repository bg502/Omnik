# omnik Implementation Plan: Go/Gin + Node.js/TypeScript Migration

## 🎯 Project Goal

Migrate omnik from Python to **Go (Gin) + Node.js/TypeScript** architecture to:
- Leverage existing Go/Gin expertise
- Use Claude Code SDK natively via Node.js
- Improve performance and resource usage
- Maintain Anthropic subscription authentication (no API key required)

---

## 📋 Architecture Overview

```
┌──────────────────────────────────────┐
│      Telegram Bot API                │
└────────────────┬─────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────┐
│   Go/Gin Bot Service (Port 8080)     │
│                                      │
│  ✓ Telegram message handling        │
│  ✓ User authorization                │
│  ✓ SQLite database (GORM)            │
│  ✓ Session management                │
│  ✓ Audit logging                     │
│  ✓ File upload/download              │
│  ✓ Command routing (/start, /new..) │
└────────────────┬─────────────────────┘
                 │ HTTP/SSE
                 ▼
┌──────────────────────────────────────┐
│  Node.js Claude Bridge (Port 9000)   │
│                                      │
│  ✓ @anthropic-ai/claude-code SDK    │
│  ✓ Server-Sent Events streaming     │
│  ✓ Session resume support            │
│  ✓ Permission mode control           │
│  ✓ Workspace management              │
└────────────────┬─────────────────────┘
                 │
                 ▼
         Claude CLI (subscription auth)
```

---

## 📦 Phase 1: Node.js Claude Bridge Setup (Week 1)

### 1.1 Project Structure
```
omnik/
├── go-bot/              # Go/Gin service
│   ├── cmd/
│   │   └── main.go
│   ├── internal/
│   │   ├── bot/         # Telegram handlers
│   │   ├── database/    # GORM models
│   │   ├── claude/      # Claude bridge client
│   │   └── middleware/  # Auth, logging
│   ├── go.mod
│   └── Dockerfile
│
├── node-bridge/         # Node.js/TypeScript service
│   ├── src/
│   │   ├── server.ts    # Express/Hono server
│   │   ├── claude.ts    # SDK wrapper
│   │   └── types.ts     # Shared types
│   ├── package.json
│   ├── tsconfig.json
│   └── Dockerfile
│
├── shared/              # Shared types/schemas
│   └── api.md           # API documentation
│
├── docker-compose.yml
└── IMPLEMENTATION_PLAN.md (this file)
```

### 1.2 Node.js Bridge Implementation

**Files to create:**
- [x] `node-bridge/package.json` - Dependencies
- [x] `node-bridge/tsconfig.json` - TypeScript config
- [x] `node-bridge/src/types.ts` - Request/response types
- [x] `node-bridge/src/claude.ts` - SDK wrapper
- [x] `node-bridge/src/server.ts` - HTTP server with SSE
- [x] `node-bridge/Dockerfile` - Multi-stage build

**Dependencies:**
```json
{
  "@anthropic-ai/claude-code": "^1.0.108",
  "express": "^4.18.2",
  "@types/express": "^4.17.21",
  "typescript": "^5.3.3"
}
```

**API Endpoints:**
```
POST /api/query
  Body: { prompt, sessionId?, workspace?, permissionMode?, allowedTools? }
  Response: SSE stream of Claude SDK messages

GET /health
  Response: { status: "ok", version: "..." }
```

### 1.3 Testing Strategy
- Unit tests for Claude SDK wrapper
- Integration test: call query endpoint, verify streaming
- Docker build test
- Subscription auth verification

**Acceptance Criteria:**
- ✅ Node bridge responds to /health
- ✅ Can execute Claude query with subscription auth
- ✅ Streams responses via SSE
- ✅ Session resume works

---

## 🐹 Phase 2: Go/Gin Bot Service (Week 2-3)

### 2.1 Database Layer (GORM)

**Models to implement:**
```go
type Session struct {
    ID            string    `gorm:"primaryKey"`
    UserID        int64     `gorm:"index"`
    WorkspacePath string
    Name          string
    Status        string    // active, paused, crashed, terminated
    CreatedAt     time.Time
    LastActivity  time.Time
    TokenUsage    int
    CostUSD       float64
}

type Message struct {
    ID        uint      `gorm:"primaryKey"`
    SessionID string    `gorm:"index"`
    Role      string    // user, assistant, system
    Content   string
    Timestamp time.Time
    TokenCount int
}

type AuditLog struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    int64     `gorm:"index"`
    Action    string
    Workspace string
    Timestamp time.Time
    Details   string
    Success   bool
}
```

**Files:**
- `internal/database/models.go` - Model definitions
- `internal/database/db.go` - Connection & migrations
- `internal/database/repository.go` - CRUD operations

### 2.2 Claude Bridge Client

**Interface:**
```go
type ClaudeBridge interface {
    Query(ctx context.Context, req QueryRequest) (<-chan ClaudeMessage, error)
    Health(ctx context.Context) error
}

type QueryRequest struct {
    Prompt         string
    SessionID      string
    Workspace      string
    PermissionMode string
    AllowedTools   []string
}

type ClaudeMessage struct {
    Type    string          // "system", "assistant", "result"
    Content json.RawMessage // Raw SDK message
}
```

**Features:**
- HTTP client with SSE parsing
- Automatic reconnection
- Context cancellation support
- Error handling & retry logic

**Files:**
- `internal/claude/client.go`
- `internal/claude/sse.go` - SSE parser
- `internal/claude/types.go`

### 2.3 Telegram Bot Handlers

**Command handlers:**
```go
// Commands
/start       -> handlers.StartCommand
/help        -> handlers.HelpCommand
/new [name]  -> handlers.NewSessionCommand
/list        -> handlers.ListSessionsCommand
/switch <id> -> handlers.SwitchSessionCommand
/status      -> handlers.StatusCommand
/kill        -> handlers.KillSessionCommand
/restart     -> handlers.RestartSessionCommand
/pwd         -> handlers.PwdCommand
/ls [path]   -> handlers.LsCommand

// Message handlers
text         -> handlers.MessageHandler (forward to Claude)
document     -> handlers.FileHandler (upload to workspace)
callback     -> handlers.CallbackHandler (button responses)
```

**Files:**
- `internal/bot/bot.go` - Main bot setup
- `internal/bot/handlers.go` - Command implementations
- `internal/bot/middleware.go` - Auth, logging
- `internal/bot/streaming.go` - Claude → Telegram streaming

### 2.4 Session Manager

**Responsibilities:**
```go
type SessionManager struct {
    db     *gorm.DB
    claude ClaudeBridge
}

// Core operations
CreateSession(userID int64, name string) (*Session, error)
GetSession(sessionID string) (*Session, error)
ListSessions(userID int64) ([]*Session, error)
SetActiveSession(userID int64, sessionID string) error
GetActiveSession(userID int64) (*Session, error)
TerminateSession(sessionID string) error

// Session recovery (container restart)
RecoverSessions() error
```

**Files:**
- `internal/bot/session_manager.go`

### 2.5 Workspace Management

**Features:**
- Create isolated workspace directories per session
- File upload/download
- Path sanitization (prevent traversal attacks)
- Directory listing
- Workspace cleanup on session termination

**Files:**
- `internal/workspace/manager.go`
- `internal/workspace/security.go` - Path validation

---

## 🐳 Phase 3: Docker Integration (Week 3)

### 3.1 Dockerfiles

**Go Bot Dockerfile:**
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o omnik-bot ./cmd

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/omnik-bot /usr/local/bin/
CMD ["omnik-bot"]
```

**Node Bridge Dockerfile:**
```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM node:20-alpine
RUN npm install -g @anthropic-ai/claude-code
WORKDIR /app
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
CMD ["node", "dist/server.js"]
```

### 3.2 Docker Compose

```yaml
version: '3.8'

services:
  omnik-bot:
    build:
      context: ./go-bot
      dockerfile: Dockerfile
    container_name: omnik-bot
    restart: unless-stopped
    user: "1000:1000"
    environment:
      - TELEGRAM_BOT_TOKEN_FILE=/run/secrets/telegram_bot_token
      - AUTHORIZED_USER_ID=${AUTHORIZED_USER_ID}
      - CLAUDE_BRIDGE_URL=http://claude-bridge:9000
      - DATABASE_PATH=/data/omnik.db
      - WORKSPACE_BASE=/workspace
    volumes:
      - workspace:/workspace
      - ./data:/data
      - ./logs:/logs
      - ./secrets/telegram_token.txt:/run/secrets/telegram_bot_token:ro
    depends_on:
      - claude-bridge
    networks:
      - omnik-net

  claude-bridge:
    build:
      context: ./node-bridge
      dockerfile: Dockerfile
    container_name: claude-bridge
    restart: unless-stopped
    user: "1000:1000"
    volumes:
      - workspace:/workspace
      - claude-auth:/home/node/.claude
    networks:
      - omnik-net
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:9000/health"]
      interval: 30s
      timeout: 10s
      retries: 3

networks:
  omnik-net:
    driver: bridge

volumes:
  workspace:
    driver: local
  claude-auth:
    driver: local
```

### 3.3 Volume Sharing

**Critical:** Share Claude authentication between containers
```yaml
volumes:
  claude-auth:/home/node/.claude  # Contains .credentials.json with OAuth tokens
```

**Authentication flow:**
1. Authenticate once: `docker compose exec claude-bridge claude auth login`
2. OAuth tokens stored in `claude-auth` volume
3. Both containers can access subscription

---

## 🧪 Phase 4: Testing & Migration (Week 4)

### 4.1 Feature Parity Checklist

Compare with existing Python implementation:

**Core Features:**
- [ ] Session creation with workspace isolation
- [ ] Multiple concurrent sessions
- [ ] Session switching
- [ ] Message forwarding to Claude
- [ ] Real-time streaming to Telegram
- [ ] Interactive prompt detection
- [ ] Inline button creation for prompts
- [ ] Button callback handling
- [ ] File upload to workspace
- [ ] Session persistence across restarts
- [ ] Audit logging
- [ ] User authorization

**Commands:**
- [ ] `/start` - Welcome message
- [ ] `/help` - Command reference
- [ ] `/new [name]` - Create session
- [ ] `/list` - List sessions
- [ ] `/switch <id>` - Switch session
- [ ] `/status` - Session details
- [ ] `/kill` - Terminate session
- [ ] `/restart` - Restart session
- [ ] `/pwd` - Current directory
- [ ] `/ls [path]` - List files

**Missing from PRD (to implement):**
- [ ] `/cd <path>` - Change directory
- [ ] `/download <path>` - Download file
- [ ] `/export [format]` - Export conversation
- [ ] `/logs [n]` - Last n lines of output
- [ ] Token usage tracking
- [ ] Cost tracking
- [ ] Context window management (50 messages)
- [ ] Auto-restart on crash
- [ ] Rate limiting

### 4.2 Migration Strategy

**Phase 4a: Parallel Deployment**
1. Deploy Go+Node stack alongside Python bot
2. Create test Telegram bot token for Go version
3. Test with separate user account
4. Compare behavior side-by-side

**Phase 4b: Data Migration**
```bash
# Export existing sessions from Python SQLite
sqlite3 data/omnik.db ".dump sessions" > sessions.sql
sqlite3 data/omnik.db ".dump messages" > messages.sql

# Import into Go GORM database
# (Schema will be compatible, GORM uses similar structure)
```

**Phase 4c: Cutover**
1. Announce maintenance window
2. Stop Python bot
3. Migrate database
4. Start Go bot with production token
5. Verify session recovery works
6. Monitor logs for 24 hours

### 4.3 Rollback Plan

**If issues arise:**
1. Stop Go bot: `docker compose stop omnik-bot claude-bridge`
2. Restore Python bot: `docker compose -f docker-compose.old.yml up -d omnik`
3. Restore database backup
4. Investigate issues in staging

**Backup strategy:**
```bash
# Before cutover
cp -r data/ data.backup/
cp docker-compose.yml docker-compose.old.yml
```

---

## 🐛 Phase 5: Bug Fixes from Current Implementation (Week 4)

### 5.1 Prompt Parser Issues

**Problem:** Current Python implementation fails to detect theme selection prompts

**Root cause:** Hardcoded indicators only match trust/confirmation prompts

**Solution in Go/Node:**
- Node.js bridge uses Claude SDK → structured JSON output
- No regex parsing needed!
- SDK provides `message.toolApproval` for permission requests
- Go bot receives structured data, creates buttons directly

**Implementation:**
```typescript
// node-bridge/src/claude.ts
for await (const sdkMessage of query({ ... })) {
  if (sdkMessage.type === 'assistant') {
    // Structured content, no parsing needed
    yield {
      type: 'assistant',
      content: sdkMessage.message.content,
      toolApproval: sdkMessage.message.toolApproval // ← Built-in!
    };
  }
}
```

### 5.2 Tini PID 1 Conflict

**Problem:** Docker compose `init: true` conflicts with tini in Dockerfile

**Solution:**
- Remove `init: true` from docker-compose.yml
- Let Dockerfile handle tini as PID 1
- Proper zombie process reaping

### 5.3 Session Recovery

**Problem:** Container restart loses all active Claude Code processes

**Solution:**
```go
// On startup, recover active sessions
func (sm *SessionManager) RecoverSessions() error {
    sessions, err := sm.db.ListSessions(SessionStatus.Active)

    for _, session := range sessions {
        log.Printf("Recovering session %s", session.ID)

        // Create new Claude session via bridge
        // Don't need to restart process - bridge handles it
        // Session continuity via SDK's resume feature
    }
}
```

**Key insight:** With SDK approach, sessions are managed by Claude Code itself (via `resume` parameter), not by long-running processes we manage!

### 5.4 Missing Commands

Implement from PRD:
- `/cd` - Change Claude's working directory (pass via SDK `cwd` option)
- `/download` - Send file from workspace as Telegram document
- `/export` - Export conversation as Markdown/JSON from database

---

## 📊 Phase 6: Monitoring & Observability (Week 5)

### 6.1 Structured Logging

**Go bot:**
```go
import "github.com/sirupsen/logrus"

log.WithFields(logrus.Fields{
    "user_id": userID,
    "session_id": sessionID,
    "command": "new_session",
}).Info("Session created")
```

**Node bridge:**
```typescript
import pino from 'pino';

const logger = pino();
logger.info({ sessionId, prompt: message }, 'Claude query started');
```

### 6.2 Health Checks

**Go bot:**
```go
router.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{
        "status": "ok",
        "claude_bridge": checkClaudeBridge(),
        "database": checkDatabase(),
    })
})
```

**Node bridge:**
```typescript
app.get('/health', (req, res) => {
    res.json({ status: 'ok', version: '1.0.0' });
});
```

### 6.3 Metrics (Optional)

**Future enhancements:**
- Prometheus metrics export
- Grafana dashboard
- Token usage tracking per user
- Session duration metrics
- Error rates

---

## ✅ Acceptance Criteria

### Minimum Viable Product (MVP)
- [ ] Can create Telegram bot session
- [ ] Can send message to Claude via Telegram
- [ ] Receives streaming response in Telegram
- [ ] Interactive prompts show as inline buttons
- [ ] Button clicks work (send choice to Claude)
- [ ] Sessions persist across container restarts
- [ ] File upload to workspace works
- [ ] User authorization working
- [ ] Audit logging operational

### Production Ready
- [ ] All PRD commands implemented
- [ ] Session recovery tested
- [ ] Performance better than Python version
- [ ] Error handling comprehensive
- [ ] Logging structured and searchable
- [ ] Docker health checks passing
- [ ] Documentation updated
- [ ] Migration tested in staging

---

## 📚 Documentation Tasks

### Files to Create/Update
- [ ] `go-bot/README.md` - Go service documentation
- [ ] `node-bridge/README.md` - Node bridge documentation
- [ ] `docs/API.md` - Bridge API specification
- [ ] `docs/DEPLOYMENT.md` - Deployment guide
- [ ] `docs/MIGRATION.md` - Python → Go migration guide
- [ ] Update root `README.md` with new architecture

---

## 🚀 Timeline Summary

| Week | Phase | Deliverable |
|------|-------|-------------|
| 1 | Node.js Bridge | Working Claude SDK wrapper with SSE streaming |
| 2 | Go Bot Core | Database, models, session management |
| 3 | Go Bot Handlers | Telegram commands, Docker integration |
| 4 | Testing & Migration | Feature parity, data migration, cutover |
| 5 | Polish | Monitoring, docs, production hardening |

**Total estimated time:** 4-5 weeks

---

## 🎯 Success Metrics

**Performance:**
- [ ] <100ms response time for simple commands
- [ ] <2s latency for first Claude response token
- [ ] Support 10+ concurrent sessions smoothly
- [ ] Memory usage <512MB baseline

**Reliability:**
- [ ] 99% uptime over 30 days
- [ ] Zero message loss
- [ ] Successful session recovery after restart
- [ ] No zombie processes

**User Experience:**
- [ ] All interactive prompts show buttons
- [ ] Streaming feels instantaneous
- [ ] Error messages clear and actionable
- [ ] Commands respond within 1 second

---

## 🔄 Next Steps

1. **Get approval** on this plan
2. **Start Phase 1:** Create node-bridge skeleton
3. **Verify Claude SDK** works with subscription auth in Docker
4. **Build Go bot incrementally** alongside working Python version
5. **Test in parallel** before switching production traffic

---

**Last Updated:** 2025-10-24
**Status:** Planning → Implementation
**Lead:** Drew + Claude Code
