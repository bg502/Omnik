# Phase 2: Core Automation Implementation

## Phase Goal
Implement REST API server and command abstraction layer to enable programmatic access to all bot features.

---

## Task: Create API Server Foundation

**Description**:
Implement HTTP server that runs concurrently with Telegram bot, providing REST API access to all commands.

**Implementation Details**:
- Create `go-bot/internal/api/server.go`
- Use chosen HTTP framework (chi recommended)
- Implement graceful shutdown
- Add CORS support
- Health check endpoint
- Request logging middleware

**File Structure**:
```
go-bot/internal/api/
├── server.go          # Main server setup
├── middleware.go      # Auth, logging, CORS
├── response.go        # Standard response helpers
└── handlers/          # Endpoint handlers (created in separate tasks)
```

**Acceptance Criteria**:
- HTTP server starts on configured port (default 8080)
- Server runs concurrently with Telegram bot
- Graceful shutdown on SIGTERM/SIGINT
- Health check endpoint returns 200 OK
- Request logging includes method, path, status, duration

**Estimated Effort**: 4-6 hours

**Priority**: HIGH

**Dependencies**: "Research Go HTTP Frameworks"

---

## Task: Implement API Authentication Middleware

**Description**:
Create middleware for API key-based authentication to secure REST endpoints.

**Implementation Details**:
- API key from environment variable `OMNI_API_KEY`
- Check `Authorization: Bearer <token>` header
- Return 401 for missing/invalid keys
- Allow health check endpoint without auth

**Error Responses**:
```json
{
  "success": false,
  "error": "Unauthorized: Invalid or missing API key",
  "timestamp": "2025-10-27T21:00:00Z"
}
```

**Acceptance Criteria**:
- Requests without API key return 401
- Requests with invalid API key return 401
- Requests with valid API key proceed to handler
- Health check endpoint accessible without auth
- Authentication errors logged

**Estimated Effort**: 2-3 hours

**Priority**: HIGH

**Dependencies**: "Create API Server Foundation"

---

## Task: Create Command Handler Interface

**Description**:
Define common interface for command execution that can be used by both Telegram bot and REST API.

**Interface Design**:
```go
// go-bot/internal/commands/interface.go

package commands

import "context"

// ExecutionContext holds context for command execution
type ExecutionContext struct {
    SessionManager *session.Manager
    ClaudeClient   claude.QueryClient
    WorkingDir     string
    UserID         int64  // For logging/audit
}

// Result represents command execution result
type Result struct {
    Success   bool
    Data      interface{}
    Error     string
    Metadata  map[string]interface{}
}

// Handler is the interface all command handlers implement
type Handler interface {
    Execute(ctx context.Context, params map[string]interface{}) (*Result, error)
}
```

**Acceptance Criteria**:
- Interface defined in `commands/interface.go`
- ExecutionContext struct includes all necessary dependencies
- Result struct supports various data types
- Documentation explains interface usage

**Estimated Effort**: 2-3 hours

**Priority**: HIGH

**Dependencies**: None

---

## Task: Refactor Session Commands to Command Handlers

**Description**:
Extract session management logic from `bot.go` into reusable command handlers.

**Handlers to Create**:
- `SessionCreateHandler` - Create new session
- `SessionSwitchHandler` - Switch to existing session
- `SessionListHandler` - List all sessions
- `SessionDeleteHandler` - Delete session
- `SessionStatusHandler` - Get current session info

**File**: `go-bot/internal/commands/session.go`

**Example**:
```go
type SessionCreateHandler struct {
    manager *session.Manager
}

func (h *SessionCreateHandler) Execute(ctx context.Context, params map[string]interface{}) (*Result, error) {
    name := params["name"].(string)
    description := params["description"].(string)
    workingDir := params["working_dir"].(string)

    sess, err := h.manager.Create(name, description, workingDir)
    if err != nil {
        return &Result{Success: false, Error: err.Error()}, nil
    }

    return &Result{Success: true, Data: sess}, nil
}
```

**Acceptance Criteria**:
- All session commands implemented as handlers
- Handlers follow common interface
- Error handling consistent across handlers
- Can be used from both bot and API
- Original bot functionality unchanged

**Estimated Effort**: 4-5 hours

**Priority**: HIGH

**Dependencies**: "Create Command Handler Interface"

---

## Task: Refactor Filesystem Commands to Command Handlers

**Description**:
Extract filesystem operation logic into reusable command handlers.

**Handlers to Create**:
- `FsChangeDirectoryHandler` - Change working directory
- `FsPrintWorkingDirHandler` - Get current directory
- `FsListFilesHandler` - List directory contents
- `FsReadFileHandler` - Read file contents
- `FsExecuteCommandHandler` - Execute bash command
- `FsWriteFileHandler` - Write content to file (new)

**File**: `go-bot/internal/commands/filesystem.go`

**Security Considerations**:
- Path validation (prevent directory traversal)
- Command whitelist or sanitization
- File size limits
- Permission checks

**Acceptance Criteria**:
- All filesystem commands implemented as handlers
- Path security validated
- Command execution properly sandboxed
- File operations safe and tested
- Original bot functionality unchanged

**Estimated Effort**: 5-6 hours

**Priority**: HIGH

**Dependencies**: "Create Command Handler Interface"

---

## Task: Refactor Claude Commands to Command Handlers

**Description**:
Extract Claude AI query logic into command handler with streaming support.

**Handler**: `ClaudeQueryHandler`

**Special Requirements**:
- Support streaming responses (SSE for API)
- Handle session ID for conversation continuity
- Workspace context
- Permission mode configuration

**File**: `go-bot/internal/commands/claude.go`

**Streaming Support**:
```go
type StreamingResult struct {
    Channel  chan StreamResponse
    Error    chan error
    Complete chan bool
}

func (h *ClaudeQueryHandler) ExecuteStreaming(ctx context.Context, params map[string]interface{}) (*StreamingResult, error)
```

**Acceptance Criteria**:
- Query execution works for both bot and API
- Streaming responses supported
- Session continuity maintained
- Error handling for Claude CLI failures
- Original bot functionality unchanged

**Estimated Effort**: 4-5 hours

**Priority**: HIGH

**Dependencies**: "Create Command Handler Interface"

---

## Task: Refactor MCP Commands to Command Handlers

**Description**:
Extract MCP server management logic into command handlers.

**Handlers to Create**:
- `McpListHandler` - List configured MCP servers
- `McpAddHandler` - Add new MCP server
- `McpRemoveHandler` - Remove MCP server (new)

**File**: `go-bot/internal/commands/mcp.go`

**Acceptance Criteria**:
- MCP commands implemented as handlers
- Server health checks included
- Project-specific MCP configuration supported
- Original bot functionality unchanged

**Estimated Effort**: 2-3 hours

**Priority**: MEDIUM

**Dependencies**: "Create Command Handler Interface"

---

## Task: Create Session API Endpoints

**Description**:
Implement REST API endpoints for session management using command handlers.

**Endpoints**:
```
POST   /api/v1/session/create
POST   /api/v1/session/switch
GET    /api/v1/session/list
GET    /api/v1/session/current
DELETE /api/v1/session/:name
GET    /api/v1/session/:name/status
```

**File**: `go-bot/internal/api/handlers/session.go`

**Request/Response Examples**:
```json
POST /api/v1/session/create
{
  "name": "my-project",
  "description": "My new project",
  "working_dir": "/workspace"
}

Response:
{
  "success": true,
  "data": {
    "id": "",
    "name": "my-project",
    "description": "My new project",
    "working_dir": "/workspace",
    "created_at": "2025-10-27T21:00:00Z",
    "last_used_at": "2025-10-27T21:00:00Z"
  },
  "timestamp": "2025-10-27T21:00:00Z"
}
```

**Acceptance Criteria**:
- All session endpoints implemented
- Request validation with meaningful errors
- Successful responses include data
- Error responses include error message
- HTTP status codes appropriate (200, 201, 400, 404, 500)

**Estimated Effort**: 3-4 hours

**Priority**: HIGH

**Dependencies**: "Refactor Session Commands to Command Handlers", "Implement API Authentication Middleware"

---

## Task: Create Filesystem API Endpoints

**Description**:
Implement REST API endpoints for filesystem operations.

**Endpoints**:
```
POST /api/v1/fs/cd
GET  /api/v1/fs/pwd
GET  /api/v1/fs/ls
GET  /api/v1/fs/cat
POST /api/v1/fs/exec
POST /api/v1/fs/write
```

**File**: `go-bot/internal/api/handlers/filesystem.go`

**Security Headers**:
- `X-Working-Directory`: Current working directory in response
- `X-Execution-Time-Ms`: Command execution time

**Acceptance Criteria**:
- All filesystem endpoints implemented
- Path validation prevents directory traversal
- Command execution properly logged
- File read operations handle binary files
- Write operations create parent directories

**Estimated Effort**: 4-5 hours

**Priority**: HIGH

**Dependencies**: "Refactor Filesystem Commands to Command Handlers"

---

## Task: Create Claude Query API Endpoint

**Description**:
Implement REST API endpoint for Claude AI queries with Server-Sent Events (SSE) streaming.

**Endpoints**:
```
POST /api/v1/claude/query        # Regular query (buffered response)
POST /api/v1/claude/query/stream # Streaming query (SSE)
```

**File**: `go-bot/internal/api/handlers/claude.go`

**SSE Streaming Format**:
```
POST /api/v1/claude/query/stream
Content-Type: application/json

{
  "prompt": "Help me create a Python web scraper",
  "session_id": "optional-session-id",
  "workspace": "/workspace/my-project"
}

Response (SSE):
data: {"type":"claude_message","data":{...}}

data: {"type":"done"}
```

**Acceptance Criteria**:
- Both buffered and streaming endpoints work
- SSE streaming properly formatted
- Session continuity supported
- Workspace context applied
- Errors handled gracefully

**Estimated Effort**: 5-6 hours

**Priority**: HIGH

**Dependencies**: "Refactor Claude Commands to Command Handlers"

---

## Task: Create MCP API Endpoints

**Description**:
Implement REST API endpoints for MCP server management.

**Endpoints**:
```
GET  /api/v1/mcp/list
POST /api/v1/mcp/add
DELETE /api/v1/mcp/:name
```

**File**: `go-bot/internal/api/handlers/mcp.go`

**Acceptance Criteria**:
- List endpoint shows server health status
- Add endpoint validates transport type
- Remove endpoint handles non-existent servers
- Project-specific MCP configuration supported

**Estimated Effort**: 2-3 hours

**Priority**: MEDIUM

**Dependencies**: "Refactor MCP Commands to Command Handlers"

---

## Task: Update main.go to Run API Server

**Description**:
Modify application entry point to run both Telegram bot and API server concurrently.

**Implementation**:
```go
func main() {
    // ... existing bot setup ...

    // Setup API server
    if os.Getenv("OMNI_API_ENABLED") == "true" {
        apiServer := api.NewServer(apiConfig)

        // Run both concurrently
        go func() {
            if err := bot.Start(ctx); err != nil {
                log.Fatalf("Bot error: %v", err)
            }
        }()

        if err := apiServer.Start(ctx); err != nil {
            log.Fatalf("API error: %v", err)
        }
    } else {
        // Just run bot
        if err := bot.Start(ctx); err != nil {
            log.Fatalf("Bot error: %v", err)
        }
    }
}
```

**Acceptance Criteria**:
- Both bot and API run concurrently
- Either can be disabled via environment variable
- Graceful shutdown stops both components
- Errors from either component logged
- Backward compatible (API disabled by default)

**Estimated Effort**: 2-3 hours

**Priority**: HIGH

**Dependencies**: "Create API Server Foundation", "Create Session API Endpoints"

---

## Task: Update Docker Configuration for API

**Description**:
Modify Docker Compose and Dockerfile to expose API port and add new environment variables.

**Changes Needed**:

`docker-compose.yml`:
```yaml
services:
  omnik:
    # ... existing config ...
    ports:
      - "8080:8080"  # API port
    environment:
      # ... existing vars ...
      - OMNI_API_ENABLED=true
      - OMNI_API_PORT=8080
      - OMNI_API_KEY=${OMNI_API_KEY}
```

`.env.example`:
```bash
# API Configuration
OMNI_API_ENABLED=true
OMNI_API_PORT=8080
OMNI_API_KEY=your_secret_api_key_here
```

**Acceptance Criteria**:
- API port exposed in Docker Compose
- Environment variables documented
- API can be enabled/disabled
- Health check endpoint accessible from host
- No breaking changes to existing deployment

**Estimated Effort**: 1-2 hours

**Priority**: MEDIUM

**Dependencies**: "Update main.go to Run API Server"

---

## Task: Create API Documentation

**Description**:
Generate comprehensive API documentation with examples and OpenAPI specification.

**Deliverables**:
1. OpenAPI 3.0 specification (`docs/api-spec.yaml`)
2. Postman collection
3. cURL examples for all endpoints
4. API usage guide in README

**Documentation Sections**:
- Authentication
- Endpoints (all with examples)
- Error codes and messages
- Rate limiting (if implemented)
- Versioning strategy

**Acceptance Criteria**:
- OpenAPI spec validates
- All endpoints documented
- Request/response examples provided
- Error cases covered
- Postman collection importable

**Estimated Effort**: 4-5 hours

**Priority**: MEDIUM

**Dependencies**: All "Create * API Endpoints" tasks

---

## Task: Write Unit Tests for Command Handlers

**Description**:
Create comprehensive unit tests for all command handler implementations.

**Test Coverage Required**:
- Session handlers (create, switch, list, delete)
- Filesystem handlers (cd, ls, cat, exec, write)
- Claude handler (query, streaming)
- MCP handlers (list, add, remove)

**Test File Structure**:
```
go-bot/internal/commands/
├── session_test.go
├── filesystem_test.go
├── claude_test.go
└── mcp_test.go
```

**Test Requirements**:
- Mock dependencies (session manager, Claude client)
- Test success cases
- Test error cases
- Test input validation
- Test edge cases

**Acceptance Criteria**:
- Code coverage > 80%
- All success paths tested
- All error paths tested
- Tests run in CI
- No flaky tests

**Estimated Effort**: 6-8 hours

**Priority**: MEDIUM

**Dependencies**: All "Refactor * Commands" tasks

---

## Task: Write Integration Tests for API

**Description**:
Create end-to-end integration tests for REST API endpoints.

**Test Scenarios**:
1. Complete session workflow (create → switch → list → delete)
2. Filesystem operations in session context
3. Claude query with session continuity
4. MCP server management
5. Authentication failures
6. Error handling

**Test File**: `go-bot/internal/api/integration_test.go`

**Test Setup**:
- Test HTTP server
- In-memory session storage
- Mock Claude client
- Test workspace directory

**Acceptance Criteria**:
- All critical paths tested
- Tests use actual HTTP requests
- Cleanup between tests
- Tests pass consistently
- Run time < 30 seconds

**Estimated Effort**: 5-6 hours

**Priority**: MEDIUM

**Dependencies**: "Write Unit Tests for Command Handlers", all "Create * API Endpoints" tasks
