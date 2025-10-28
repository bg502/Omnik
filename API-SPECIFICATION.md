# Omnik API Specification

## Overview

This document describes the API support implementation for the Omnik bot system, enabling programmatic control of Claude AI through both HTTP API and Telegram messaging.

## Architecture

### Dual-Bot Setup

The system uses two independent bot instances sharing workspace and session storage:

#### 1. Omnik Bot (Main)
- **Bot Token**: `7490276912:AAH3orrFvgjWiM4--QVfbt8ywrlebhpRPZ4`
- **Username**: `@omnikai_bot`
- **Purpose**: Personal chat and group chat interactions
- **Authorized Chat**: `-1003048532828` (OmnikAI Chat)
- **Container**: `omnik`

#### 2. Memnikai Bot (API)
- **Bot Token**: `8329011908:AAE6Vv-Fx5ZheexoupguXBtDVWB1ItQffqk`
- **Username**: `@Memnkai_bot`
- **Purpose**: Dedicated API access and programmatic control
- **Authorized Chat**: `-4958242815` (Memnikai_chat)
- **Container**: `omnik-api`
- **HTTP API Port**: `8081` (external) → `8080` (internal)

### Shared Resources

Both bots share:
- **Workspace Volume**: `/workspace` (project directories)
- **Claude Home Volume**: `/home/node/.claude` (sessions and configuration)
- **Session Storage**: Sessions are accessible by both bots
- **Docker Socket**: `/var/run/docker.sock` (for container management)

### Per-Chat Context Isolation

Each chat maintains independent context:
- **Session**: Current active session name
- **Working Directory**: Current workspace path
- **Thread-Safe**: Uses `sync.RWMutex` for concurrent access

Implementation in `go-bot/internal/bot/bot.go:105-167`:
```go
type ChatContext struct {
    ChatID         int64
    CurrentSession string
    WorkingDir     string
}

type Bot struct {
    chatContexts   map[int64]*ChatContext
    contextMutex   sync.RWMutex
    // ...
}
```

## HTTP API

### Base URL
```
http://localhost:8081
```

### Endpoints

#### POST /api/query
Send a query to Claude AI and receive processed response.

**Request:**
```json
{
  "message": "Your query text",
  "session_id": "optional-session-name"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Query accepted and being processed"
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "Error description"
}
```

**Status Codes:**
- `200 OK`: Query accepted and being processed
- `400 Bad Request`: Invalid JSON or missing message field
- `405 Method Not Allowed`: Wrong HTTP method
- `500 Internal Server Error`: Processing error

**Notes:**
- Response is sent to Telegram chat asynchronously
- Messages appear in Memnikai_chat (`-4958242815`)
- Includes Stop button for canceling queries
- Supports real-time streaming updates

#### GET /api/health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy"
}
```

### Implementation Details

#### Request Flow
1. HTTP request received by `/api/query` endpoint
2. Request validated and parsed
3. Message passed to `ProcessAPIMessage()` function
4. "🔄 API Query" message sent to Telegram with Stop button
5. Claude CLI invoked with query
6. Responses streamed back to Telegram chat
7. Stop button removed when complete

#### Response Handling (go-bot/internal/bot/bot.go:1364-1622)

The `ProcessAPIMessage()` function uses the same proven pattern as `forwardToClaude()`:

**Content Event Tracking:**
```go
type contentEvent struct {
    eventType string // "text" or "tool"
    content   string
}
var contentHistory []contentEvent
```

**Features:**
- Chronological content tracking (text and tool usage)
- Progressive message updates with rate limiting (1 second intervals)
- Message splitting for long responses
- Stop button support via channel signaling
- Comprehensive logging for debugging

**Message Types Handled:**
- `system`: Session initialization
- `assistant`: Text content and tool usage
- `result`: Tool execution results
- `done`: Query completion
- `error`: Error handling

## Scripts

### api-query.sh
Primary method for programmatic API access via HTTP.

**Location**: `scripts/api-query.sh`

**Usage:**
```bash
./scripts/api-query.sh "Your message" [session_id]
```

**Examples:**
```bash
# Simple query
./scripts/api-query.sh "What is 2+2?"

# Query with specific session
./scripts/api-query.sh "List files" "my-session"
```

**Implementation:**
```bash
API_URL="http://localhost:8081/api/query"

# Build JSON payload
if [ -n "$SESSION_ID" ]; then
    JSON_PAYLOAD=$(jq -n \
        --arg msg "$MESSAGE" \
        --arg sid "$SESSION_ID" \
        '{message: $msg, session_id: $sid}')
else
    JSON_PAYLOAD=$(jq -n \
        --arg msg "$MESSAGE" \
        '{message: $msg}')
fi

# Send request
curl -X POST "$API_URL" \
  -H "Content-Type: application/json" \
  -d "$JSON_PAYLOAD"
```

### query.sh (Deprecated)
Sends messages via Telegram API.

**Location**: `scripts/query.sh`

**Status**: ⚠️ **Known Limitation** - Currently uses Memnikai token, but Memnikai cannot see its own messages (Telegram platform limitation).

**Usage:**
```bash
./scripts/query.sh "Your message"
```

**Implementation:**
```bash
curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"$QUERY\"}"
```

### send-to-topic.sh
Sends messages to specific Telegram forum topics.

**Location**: `scripts/send-to-topic.sh`

**Status**: ⚠️ **Known Limitation** - Same issue as query.sh (bot cannot see own messages).

**Usage:**
```bash
./scripts/send-to-topic.sh <topic_id> <message>
```

**Implementation:**
```bash
curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"message_thread_id\": ${TOPIC_ID}, \"text\": \"$MESSAGE\"}"
```

### status.sh
Checks status of both bot containers.

**Location**: `scripts/status.sh`

**Usage:**
```bash
./scripts/status.sh
```

## Configuration

### Environment Variables (.env)

```bash
# ============================================
# Main Bot (Omnik) - Personal & Group Chat
# ============================================
OMNI_TELEGRAM_BOT_TOKEN=7490276912:AAH3orrFvgjWiM4--QVfbt8ywrlebhpRPZ4
OMNI_AUTHORIZED_USER_ID=55340979
OMNI_TG_AUTH_CHAT_ID=-1003048532828

# ============================================
# API Bot (Memnikai) - Dedicated API Access
# ============================================
OMNI_API_BOT_TOKEN=8329011908:AAE6Vv-Fx5ZheexoupguXBtDVWB1ItQffqk
OMNI_API_AUTHORIZED_USER_ID=55340979
OMNI_API_AUTH_CHAT_ID=-4958242815

# ============================================
# Script Configuration (for API bot control)
# ============================================
# Note: Uses Memnikai bot token to send messages to its own chat
OMNI_AUTH_API_TOKEN=8329011908:AAE6Vv-Fx5ZheexoupguXBtDVWB1ItQffqk
CHAT_ID=-4958242815

# ============================================
# Claude Configuration (shared by both bots)
# ============================================
OMNI_CLAUDE_MODEL=sonnet
OMNI_ANTHROPIC_API_KEY=<your-key>

# ============================================
# Application Settings
# ============================================
OMNI_LOG_LEVEL=DEBUG
```

### Docker Compose Configuration

```yaml
services:
  # Main Bot - Personal & Group Chat
  omnik:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: omnik
    environment:
      - OMNI_TELEGRAM_BOT_TOKEN=${OMNI_TELEGRAM_BOT_TOKEN}
      - OMNI_AUTHORIZED_USER_ID=${OMNI_AUTHORIZED_USER_ID}
      - OMNI_TG_AUTH_CHAT_ID=${OMNI_TG_AUTH_CHAT_ID:-}
      - OMNI_ANTHROPIC_API_KEY=${OMNI_ANTHROPIC_API_KEY}
      - OMNI_USE_CLAUDE_SDK=true
      - OMNI_CLAUDE_MODEL=${OMNI_CLAUDE_MODEL:-sonnet}
      - OMNI_LOG_LEVEL=${OMNI_LOG_LEVEL:-INFO}
    volumes:
      - workspace:/workspace
      - claude-home:/home/node
      - /var/run/docker.sock:/var/run/docker.sock

  # API Bot - Dedicated API Access
  omnik-api:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: omnik-api
    environment:
      - OMNI_TELEGRAM_BOT_TOKEN=${OMNI_API_BOT_TOKEN}
      - OMNI_AUTHORIZED_USER_ID=${OMNI_API_AUTHORIZED_USER_ID}
      - OMNI_TG_AUTH_CHAT_ID=${OMNI_API_AUTH_CHAT_ID:-}
      - OMNI_ANTHROPIC_API_KEY=${OMNI_ANTHROPIC_API_KEY}
      - OMNI_USE_CLAUDE_SDK=true
      - OMNI_CLAUDE_MODEL=${OMNI_CLAUDE_MODEL:-sonnet}
      - OMNI_LOG_LEVEL=${OMNI_LOG_LEVEL:-INFO}
      - OMNI_API_PORT=8080  # Enable HTTP API
    ports:
      - "8081:8080"
    volumes:
      - workspace:/workspace        # SHARED
      - claude-home:/home/node      # SHARED
      - /var/run/docker.sock:/var/run/docker.sock

volumes:
  workspace:
    driver: local
  claude-home:
    driver: local
```

## Telegram Platform Limitations

### Bot-to-Bot Messaging
❌ **Limitation**: Telegram bots cannot see messages from other bots in groups.

**Impact**:
- Initially planned to use Omnik bot to send messages that Memnikai would process
- This approach doesn't work - messages are filtered by Telegram platform
- Privacy mode settings don't affect this limitation

**Workaround**: Implemented HTTP API as alternative.

### Bots Cannot See Own Messages
❌ **Limitation**: A bot cannot see messages it sends via Telegram API.

**Impact**:
- Scripts using `query.sh` and `send-to-topic.sh` won't work if using Memnikai's token
- Messages sent by Memnikai via sendMessage API are invisible to Memnikai

**Solutions Evaluated**:
1. **Option A**: Revert scripts to Omnik token (mixed bot appearance)
2. **Option B**: Use HTTP API exclusively (recommended - consistent experience)
3. **Option C**: Hybrid approach (flexible but complex)

**Current Status**: Known limitation documented, solution pending user decision.

## Logging

### HTTP API Logs

Detailed logging implemented matching Telegram bot style:

```
[API] Received query: What is 2+2? (session: )
[API] Processing message: What is 2+2? (session: )
[ChatContext] Created context for chat -4958242815: session="RAG" workingDir="/workspace/rag"
[Claude CLI] Executing: claude [--print --output-format stream-json ...]
[API] Received message type: system
[API] Received message type: assistant
[API] Extracted text content (length: 1): 4...
[API] Updating message. History items: 1, Display length: 1
[API] Received message type: result
[API] ← Received 4 messages from Claude
[API] Sending final response (length: 1)
```

### Log Levels

- **System messages**: Session initialization, session ID updates
- **Assistant messages**: Text extraction, tool usage
- **Result messages**: Tool execution results
- **Debug messages**: Message type, content length, update frequency
- **Error messages**: Parsing errors, query errors

## Session Management

### Session Switching

Both HTTP API and Telegram commands support session switching:

**Via HTTP API:**
```bash
./scripts/api-query.sh "list files" "my-session"
```

**Via Telegram:**
```
/session my-session
```

### Session Storage

Sessions are stored in:
```
/home/node/.claude/sessions/
```

Session data includes:
- **Name**: Human-readable session identifier
- **ID**: Claude conversation UUID
- **Working Directory**: Project workspace path
- **Created/Updated timestamps**

### Session Isolation

Each chat maintains its own active session:
- Personal chat with Omnik: Independent session
- OmnikAI Chat group: Independent session
- Memnikai_chat: Independent session
- HTTP API requests: Use Memnikai_chat session

## Features

### Stop Button Support

Both Telegram and HTTP API requests include Stop button:

**Implementation:**
1. Stop channel created and registered per chat
2. Inline keyboard with "⏹️ Stop" button attached to processing message
3. User clicks Stop → signal sent to stop channel
4. Query context canceled, Claude execution terminated
5. Button removed, stop notification sent

**Code Location**: `go-bot/internal/bot/bot.go:1415-1461`

### Progressive Message Updates

Responses update in real-time with rate limiting:

**Update Strategy:**
- Every 3 messages OR
- Every 1000 milliseconds (1 second)

**Benefits:**
- Real-time feedback without API rate limiting issues
- Smooth user experience
- Reduced Telegram API calls

### Message Splitting

Long responses automatically split across multiple messages:

**Behavior:**
- Primary message: Initial response (up to Telegram limit)
- Continuation messages: Subsequent parts
- Each part numbered (e.g., "Part 2/3")

**Code Location**: `go-bot/internal/bot/bot.go:905-965`

### Tool Usage Display

Tool executions formatted and displayed:

**Format:**
```
🛠️ Tool: Read
  file_path: "/workspace/example.txt"
```

**Code Location**: `go-bot/internal/bot/bot.go:841-903`

## Development

### Building Containers

```bash
# Build both containers
docker compose build

# Build specific container
docker compose build omnik-api
```

### Running Containers

```bash
# Start all services
docker compose up -d

# Start specific service
docker compose up -d omnik-api

# View logs
docker compose logs -f omnik-api
```

### Testing HTTP API

```bash
# Health check
curl http://localhost:8081/api/health

# Send query
./scripts/api-query.sh "test message"

# Send query with session
./scripts/api-query.sh "test message" "my-session"
```

### Rebuilding After Code Changes

```bash
# Rebuild and restart API bot
docker compose build omnik-api && docker compose up -d omnik-api

# Check logs
docker compose logs -f omnik-api --tail=50
```

## Troubleshooting

### API Not Responding

**Check if container is running:**
```bash
docker compose ps
```

**Check logs:**
```bash
docker compose logs omnik-api --tail=50
```

**Verify API is listening:**
```bash
curl http://localhost:8081/api/health
```

### Messages Not Processed

**For HTTP API:**
- Check if query was accepted (`{"success":true}`)
- Check omnik-api logs for processing messages
- Verify Claude CLI is authenticated

**For Telegram Scripts:**
- ⚠️ Known issue: Bot cannot see own messages
- Use HTTP API (`api-query.sh`) instead

### Session Errors

**"No active session" error:**
```bash
# Create new session via Telegram
/newsession session-name

# Or switch to existing session
/session existing-session
```

**Session ID errors:**
- Restart Claude CLI authentication in container
- Check `/home/node/.claude/sessions/` for valid sessions

### Stop Button Not Working

**Check logs for:**
- Stop signal registration: `Sent stop signal for chat -4958242815`
- Stop request: `[API] Stop requested by user`
- Proper cleanup: Stop channel deleted after request

## Future Improvements

### Potential Enhancements

1. **Async Response Webhook**
   - Return response via webhook URL instead of Telegram
   - Enable true REST API without Telegram dependency

2. **Authentication System**
   - API key authentication
   - Rate limiting per API key
   - Usage tracking

3. **Batch Query Support**
   - Process multiple queries in single request
   - Return all responses together

4. **WebSocket Support**
   - Real-time streaming responses
   - Better for long-running queries

5. **Response Format Options**
   - JSON-only response mode (no Telegram)
   - Markdown formatting control
   - Custom response templates

### Known Issues

1. **Telegram Script Limitation**
   - `query.sh` and `send-to-topic.sh` don't work with Memnikai token
   - Bot cannot see own messages (platform limitation)
   - Solution pending: Use HTTP API exclusively or mixed token approach

2. **Session Synchronization**
   - No conflict resolution if both bots access same session
   - Relies on file-based session storage
   - Could benefit from proper session locking

3. **Error Recovery**
   - Claude CLI errors not always gracefully handled
   - Session ID mismatches can cause "Done (no output)"
   - Need better error recovery and auto-retry logic

## Version History

### v1.1.002-additional-buttons (Current)
- Implemented HTTP API with comprehensive logging
- Added Stop button support for API requests
- Fixed response streaming and display
- Updated scripts to use Memnikai token
- Documented Telegram platform limitations

### v0.2.000-API
- Initial dual-bot architecture
- Per-chat context isolation
- Basic HTTP API implementation
- Session management across bots

### v1.0.x
- Single bot implementation
- Basic Telegram commands
- MCP support
- Initial session management

---

**Last Updated**: 2025-10-28
**Maintainer**: bg502
**Repository**: https://github.com/bg502/omnik
