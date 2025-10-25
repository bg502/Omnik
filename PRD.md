# Product Requirements Document: omnik

## Executive Summary

**omnik is a Telegram-based conversational interface for Claude Code that enables developers to manage AI-powered coding sessions from anywhere.** The application runs as a containerized Python service that spawns and controls Claude Code subprocess sessions, providing a mobile-first development assistant with persistent conversation state, workspace isolation, and Docker integration.

Built on python-telegram-bot v22.5 and Pydantic AI frameworks, omnik bridges the gap between mobile accessibility and powerful AI coding assistance. The system manages Claude Code as a long-running subprocess (not PID 1) within a Docker container, handling real-time output streaming, session persistence, and secure workspace management.

**Key Innovation:** While existing solutions either provide Docker control OR AI assistance, omnik uniquely combines Claude Code's agentic capabilities with Telegram's ubiquitous interface and Docker's isolation, creating an end-to-end mobile development environment manager.

## Problem Statement

Developers need AI coding assistance but face three critical limitations:

**Access constraints**: Claude Code requires desktop terminal access, blocking mobile or remote development scenarios where developers need quick code reviews, bug fixes, or project assistance while away from their workstation.

**Environment management overhead**: Setting up, maintaining, and switching between development environments requires manual configuration, making it difficult to quickly spin up isolated coding sessions for different projects or experiments.

**Disconnected tooling**: AI assistants (Claude), infrastructure (Docker), and communication platforms (Telegram) operate independently, forcing developers to context-switch between tools and manually coordinate workflows.

omnik solves these by providing a unified, mobile-accessible interface that orchestrates Claude Code sessions within managed Docker environments, accessible through the ubiquitous Telegram messaging platform.

## Goals and Non-Goals

### Goals

**Primary objectives:**
- Enable full Claude Code session management through Telegram's conversational interface with sub-second response times for simple commands
- Provide real-time streaming of Claude Code output to Telegram with readable formatting and minimal latency (<2s for first token)
- Maintain persistent conversation context across sessions with SQLite-backed state management supporting workspace switching
- Ensure secure, isolated execution environment with non-root Docker containers and workspace sandboxing
- Support multiple concurrent workspaces with independent sessions and file isolation per workspace

**Success metrics:**
- 95% uptime for single-user deployment
- Stream Claude Code responses with <500ms update intervals
- Zero data loss on graceful shutdown
- Complete audit trail of all commands and file operations

### Non-Goals

- **Multi-user deployment**: Single bot token = single user. Multi-tenancy deferred to future versions.
- **GUI/web interface**: Telegram is the exclusive interface. No web dashboard in v1.
- **Code execution outside Docker**: All Claude Code sessions run in containerized environments only.
- **Real-time collaboration**: No simultaneous multi-user access to same workspace.
- **Built-in CI/CD pipelines**: Integration points provided, but orchestration is external.
- **Windows support**: Linux containers only, no native Windows runtime.

## User Stories

### Core Workflows

**As a developer commuting home**, I want to review a pull request via Telegram so I can provide feedback without opening my laptop.

**As an on-call engineer**, I want to debug production issues by asking Claude to analyze logs and suggest fixes, receiving responses in Telegram while I'm mobile.

**As a project maintainer**, I want to quickly prototype a code change in an isolated workspace and see the results without setting up a local environment.

**As a remote consultant**, I want to switch between multiple client projects, each with its own Claude Code session and workspace context.

**As a security-conscious developer**, I want all code execution isolated in Docker containers with audit logs of every command.

### Specific Interactions

**Session management:**
- `/start` - Initialize omnik bot and see available commands
- `/new [project-name]` - Create new Claude Code session in dedicated workspace
- `/list` - Show all active sessions with workspace paths and uptime
- `/switch [session-id]` - Change active workspace context
- Send message - Forward to active Claude Code session, stream response back

**File operations:**
- Upload file to Telegram - Save to active workspace, notify Claude
- `/upload [path]` - Specify destination path for uploaded file
- `/download [path]` - Retrieve file from workspace as Telegram document

**Workspace control:**
- `/pwd` - Show current working directory in active session
- `/ls [path]` - List files in workspace directory
- `/cd [path]` - Change Claude Code's working directory

**Session monitoring:**
- `/status` - Session health, token usage, uptime
- `/logs [n]` - Last n lines of Claude Code output
- `/kill` - Gracefully terminate active session
- `/restart` - Stop and restart Claude Code subprocess

## Technical Architecture

### High-Level System Design

```
┌─────────────────────────────────────────────────────────┐
│                    Telegram Platform                     │
│                  (Bot API, Message Queue)                │
└────────────────────┬────────────────────────────────────┘
                     │ HTTPS
                     ▼
         ┌───────────────────────┐
         │   omnik Container     │
         │  (Python 3.11, tini)  │
         │                       │
         │  ┌─────────────────┐  │
         │  │  Bot Manager    │  │◀─── python-telegram-bot
         │  │  (Async Queue)  │  │     Pydantic AI agents
         │  └────────┬────────┘  │
         │           │           │
         │  ┌────────▼────────┐  │
         │  │  Session Mgr    │  │
         │  │  ┌───────────┐  │  │
         │  │  │ Session 1 │  │  │
         │  │  │ (State)   │  │  │
         │  │  └─────┬─────┘  │  │
         │  │        │        │  │
         │  │  ┌─────▼──────┐ │  │
         │  │  │ Claude     │ │  │◀─── asyncio subprocess
         │  │  │ Code CLI   │ │  │     Real-time streaming
         │  │  │ (Node.js)  │ │  │
         │  │  └────────────┘ │  │
         │  └─────────────────┘  │
         │           │           │
         │  ┌────────▼────────┐  │
         │  │  SQLite DB      │  │◀─── Session state
         │  │  - Messages     │  │     Audit logs
         │  │  - Sessions     │  │     User preferences
         │  └─────────────────┘  │
         │           │           │
         └───────────┼───────────┘
                     │ Volume Mount
                     ▼
         ┌───────────────────────┐
         │   Workspace Volume    │
         │  /workspace/          │
         │   ├── session-1/      │
         │   ├── session-2/      │
         │   └── session-n/      │
         └───────────────────────┘
```

### Component Responsibilities

**Bot Manager**: Telegram webhook/polling handler, message routing, user authentication, rate limiting (1 req/sec per chat), command parsing and dispatch.

**Session Manager**: Claude Code subprocess lifecycle (start/stop/restart), session state persistence, workspace isolation, multi-session coordination, health monitoring with auto-restart.

**Claude Code Subprocess**: Managed as asyncio subprocess, stdout/stderr streaming, JSON-formatted I/O, working directory control per session, graceful SIGTERM shutdown (10s timeout).

**State Persistence**: SQLite database with tables for sessions (id, user, workspace, pid, created_at), messages (session_id, role, content, timestamp, tokens), audit_logs (user, action, workspace, timestamp, details).

**Workspace Manager**: Volume mounting to /workspace, per-session directories with isolation, file upload/download handling, permission management (UID 1000:1000), temp file cleanup on session end.

### Technology Stack

**Core frameworks:**
- Python 3.11+ (async/await throughout)
- python-telegram-bot v22.5 (async, Telegram Bot API 9.2)
- Pydantic AI (type-safe agent framework, streaming support)
- claude-agent-sdk (official Python SDK for Claude Code control)

**Infrastructure:**
- Docker with tini as PID 1 init system
- docker-compose for orchestration
- SQLite for persistence (single-user appropriate)
- Node.js 18+ (for Claude Code CLI dependency)

**Process management:**
- asyncio.create_subprocess_exec for Claude Code
- Real-time stdout/stderr streaming
- Signal handling for graceful shutdown
- Health monitoring with exponential backoff restart

## Core Features and Requirements

### F1: Session Management

**Requirements:**
- Create new Claude Code session with unique workspace directory within 5 seconds
- Support minimum 5 concurrent sessions per user
- Persist session state (messages, context, workspace path) to SQLite on every message
- Resume sessions after omnik container restart by reloading from database
- Terminate sessions gracefully with SIGTERM (10s timeout), fallback to SIGKILL
- Auto-restart crashed sessions with exponential backoff (1s, 2s, 4s, 8s, max 5 retries)

**Implementation:**
```python
class SessionManager:
    async def create_session(workspace_id: str) -> Session:
        # Create workspace directory
        # Start Claude Code subprocess with claude-agent-sdk
        # Register in database
        # Return Session handle
    
    async def terminate_session(session_id: str):
        # Send SIGTERM to subprocess
        # Wait 10s for graceful shutdown
        # Force kill if timeout
        # Cleanup workspace (optional)
        # Update database
```

### F2: Real-Time Output Streaming

**Requirements:**
- Stream Claude Code stdout to Telegram with 500ms buffer intervals
- Handle Telegram message length limit (4096 chars) by splitting into multiple messages
- Support Markdown formatting for code blocks and syntax highlighting
- Display typing indicator during Claude Code processing
- Edit messages progressively during streaming (max 1 edit per 500ms per Telegram rate limits)

**Implementation:**
```python
async def stream_claude_output(process, chat_id):
    buffer = ""
    last_edit = time.time()
    sent_message = None
    
    async for line in process.stdout:
        buffer += line.decode()
        
        if time.time() - last_edit > 0.5 and len(buffer) > 20:
            if sent_message is None:
                sent_message = await bot.send_message(chat_id, buffer)
            else:
                await sent_message.edit_text(buffer[:4096])
            last_edit = time.time()
    
    # Final edit with complete output
    await sent_message.edit_text(buffer[:4096])
```

### F3: Persistent Conversation Context

**Requirements:**
- Store complete message history in SQLite with fields: session_id, role (user/assistant/system), content, timestamp, token_count
- Maintain separate context per session/workspace
- Support exporting conversation history as Markdown or JSON
- Implement context window management: keep last 50 messages, summarize older content
- Track token usage and cost per session

**Database schema:**
```sql
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL, -- 'user', 'assistant', 'system'
    content TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    token_count INTEGER,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
```

### F4: Workspace Isolation

**Requirements:**
- Each session has dedicated directory: `/workspace/{session_id}/`
- Non-root user (UID 1000:1000) owns all workspace files
- Claude Code subprocess runs with cwd set to session workspace
- File uploads saved to active session workspace with size limit 25MB
- Prevent path traversal attacks with input sanitization
- Optional: Workspace directory size quota (1GB per session)

**Security validation:**
```python
def validate_path(user_path: str, workspace_root: Path) -> Path:
    # Resolve to absolute path
    abs_path = (workspace_root / user_path).resolve()
    
    # Ensure path is within workspace
    if not str(abs_path).startswith(str(workspace_root)):
        raise SecurityError("Path traversal detected")
    
    return abs_path
```

### F5: Command Interface

**Bot commands:**
- `/start` - Welcome message, authentication check, usage instructions
- `/new [name]` - Create session with optional name/project identifier
- `/list` - Show all sessions with ID, name, status, uptime, last activity
- `/switch <id>` - Change active session context
- `/status` - Current session info (workspace, PID, memory, tokens used)
- `/pwd`, `/ls [path]`, `/cd <path>` - Directory navigation in active workspace
- `/upload` - Prompt for file upload to current directory
- `/download <path>` - Send file from workspace as Telegram document
- `/kill` - Terminate active session
- `/restart` - Restart Claude Code subprocess for active session
- `/export [format]` - Export conversation (markdown/json)
- `/help` - Command reference

**Message handling:**
- Non-command messages forwarded to active Claude Code session
- If no active session, prompt user to create one with `/new`
- Show typing indicator while Claude processes
- Stream response back with progressive message editing

### F6: Error Handling and Recovery

**Requirements:**
- Catch and log all exceptions with full traceback to application logs
- User-friendly error messages in Telegram ("Claude Code crashed. Restarting..." vs raw exceptions)
- Automatic session restart on subprocess crash (max 5 attempts with exponential backoff)
- Network error retry for Telegram API calls (3 retries, exponential backoff)
- Graceful degradation if SQLite locked (retry with 100ms delay, max 10 attempts)
- Send critical errors to admin via Telegram (configurable admin chat ID)

**Error categories:**
```python
class ClaudeCodeError(Exception): pass
class SessionNotFoundError(Exception): pass
class WorkspaceSecurityError(Exception): pass
class TelegramAPIError(Exception): pass
```

### F7: Security and Authentication

**Requirements:**
- Whitelist-based authentication: Only configured Telegram user ID can interact
- All subprocess execution confined to Docker container
- No Docker socket mounting (omnik manages containers from outside, or runs Claude Code directly)
- Secrets (Telegram token, Claude API key) via Docker secrets mounted at `/run/secrets/`
- Audit log every command: user_id, timestamp, command, workspace, result
- Input sanitization on all user-provided paths and command arguments
- Read-only container root filesystem with writable /tmp (tmpfs) and /workspace (volume)

**docker-compose security:**
```yaml
services:
  omnik:
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - SETUID
      - SETGID
    read_only: true
    tmpfs:
      - /tmp:size=100M
```

## Technical Specifications

### Environment Configuration

**Environment variables:**
```bash
# Required
TELEGRAM_BOT_TOKEN=<from Docker secret>
ANTHROPIC_API_KEY=<from Docker secret>
AUTHORIZED_USER_ID=123456789

# Optional
LOG_LEVEL=INFO  # DEBUG, INFO, WARNING, ERROR
MAX_SESSIONS=10
SESSION_TIMEOUT_HOURS=24
WORKSPACE_BASE=/workspace
ADMIN_CHAT_ID=123456789  # For critical error notifications
RATE_LIMIT_REQUESTS=60  # Per minute
```

**Docker Compose:**
```yaml
version: '3.8'

services:
  omnik:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        - UID=1000
        - GID=1000
    
    container_name: omnik
    init: true  # Use tini as PID 1
    restart: unless-stopped
    
    user: "1000:1000"
    
    secrets:
      - telegram_bot_token
      - anthropic_api_key
    
    environment:
      - PYTHONUNBUFFERED=1
      - AUTHORIZED_USER_ID=${AUTHORIZED_USER_ID}
      - LOG_LEVEL=${LOG_LEVEL:-INFO}
    
    volumes:
      - workspace:/workspace
      - ./logs:/app/logs
    
    networks:
      - omnik-net
    
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '0.5'
          memory: 512M
    
    security_opt:
      - no-new-privileges:true
    
    cap_drop:
      - ALL
    
    read_only: true
    tmpfs:
      - /tmp:size=100M

secrets:
  telegram_bot_token:
    file: ./secrets/telegram_token.txt
  anthropic_api_key:
    file: ./secrets/anthropic_key.txt

volumes:
  workspace:
    driver: local

networks:
  omnik-net:
    driver: bridge
```

### Dockerfile

```dockerfile
FROM python:3.11-slim

# Install system dependencies including Node.js for Claude Code
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        tini \
        curl \
        ca-certificates \
        gnupg \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_18.x nodistro main" | tee /etc/apt/sources.list.d/nodesource.list \
    && apt-get update \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI globally
RUN npm install -g @anthropic-ai/claude-code

# Create non-root user
ARG UID=1000
ARG GID=1000
RUN groupadd -g ${GID} appuser && \
    useradd -m -u ${UID} -g ${GID} -s /bin/bash appuser

# Set working directory
WORKDIR /app

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Create directories
RUN mkdir -p /workspace /app/logs /app/data && \
    chown -R appuser:appuser /app /workspace

# Copy application
COPY --chown=appuser:appuser src/ ./src/

# Switch to non-root user
USER appuser

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD python -c "import sys; sys.exit(0)"

# Use tini as PID 1
ENTRYPOINT ["tini", "-g", "--"]
CMD ["python", "-m", "src.main"]
```

### Python Dependencies (requirements.txt)

```
# Core frameworks
python-telegram-bot==22.5
pydantic-ai==0.1.0
anthropic==0.39.0
claude-agent-sdk==0.1.0

# Async support
aiofiles==24.1.0
asyncio==3.4.3

# Data and persistence
sqlalchemy==2.0.35
aiosqlite==0.20.0

# Utilities
python-dotenv==1.0.1
pydantic==2.10.0
structlog==24.4.0
```

## Data Models

### Pydantic Models

```python
from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, Literal

class Session(BaseModel):
    id: str = Field(description="Unique session identifier (UUID)")
    user_id: int = Field(description="Telegram user ID")
    workspace_path: str = Field(description="Absolute path to workspace directory")
    name: Optional[str] = Field(default=None, description="User-provided session name")
    status: Literal["active", "paused", "crashed", "terminated"] = "active"
    pid: Optional[int] = Field(default=None, description="Claude Code subprocess PID")
    created_at: datetime = Field(default_factory=datetime.utcnow)
    last_activity: datetime = Field(default_factory=datetime.utcnow)
    token_usage: int = Field(default=0, description="Total tokens used")
    cost_usd: float = Field(default=0.0, description="Total cost in USD")

class Message(BaseModel):
    id: Optional[int] = None
    session_id: str = Field(description="Parent session ID")
    role: Literal["user", "assistant", "system"] = Field(description="Message sender")
    content: str = Field(description="Message text content")
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    token_count: Optional[int] = Field(default=None, description="Tokens in this message")

class AuditLog(BaseModel):
    id: Optional[int] = None
    user_id: int
    action: str = Field(description="Command or action performed")
    workspace: Optional[str] = Field(default=None)
    timestamp: datetime = Field(default_factory=datetime.utcnow)
    details: Optional[str] = Field(default=None, description="Additional context as JSON")
    success: bool = Field(default=True)

class WorkspaceInfo(BaseModel):
    session_id: str
    path: str
    size_bytes: int = Field(description="Total workspace size")
    file_count: int = Field(description="Number of files in workspace")
    last_modified: datetime
```

### SQLite Schema

```sql
-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    workspace_path TEXT NOT NULL,
    name TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    pid INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    token_usage INTEGER DEFAULT 0,
    cost_usd REAL DEFAULT 0.0
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_status ON sessions(status);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    token_count INTEGER,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_session_id ON messages(session_id);
CREATE INDEX idx_messages_timestamp ON messages(timestamp);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    workspace TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    details TEXT,
    success BOOLEAN DEFAULT TRUE
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
```

## API/Interface Design

### Internal Python API

**SessionManager class:**
```python
class SessionManager:
    def __init__(self, db: Database, workspace_base: Path):
        self.sessions: Dict[str, Session] = {}
        self.db = db
        self.workspace_base = workspace_base
    
    async def create_session(
        self, 
        user_id: int, 
        name: Optional[str] = None
    ) -> Session:
        """Create new Claude Code session with isolated workspace."""
        ...
    
    async def get_session(self, session_id: str) -> Optional[Session]:
        """Retrieve session by ID."""
        ...
    
    async def list_sessions(self, user_id: int) -> List[Session]:
        """List all sessions for user."""
        ...
    
    async def terminate_session(self, session_id: str) -> bool:
        """Gracefully stop Claude Code subprocess and cleanup."""
        ...
    
    async def restart_session(self, session_id: str) -> bool:
        """Restart crashed or hung session."""
        ...
    
    async def send_message(
        self, 
        session_id: str, 
        message: str
    ) -> AsyncIterator[str]:
        """Send message to Claude Code and stream response."""
        ...
```

**ClaudeCodeProcess class:**
```python
class ClaudeCodeProcess:
    def __init__(
        self, 
        session_id: str, 
        workspace: Path, 
        anthropic_key: str
    ):
        self.session_id = session_id
        self.workspace = workspace
        self.process: Optional[asyncio.subprocess.Process] = None
        self.anthropic_key = anthropic_key
    
    async def start(self) -> int:
        """Start Claude Code subprocess, return PID."""
        ...
    
    async def send_input(self, text: str):
        """Send input to Claude Code stdin."""
        ...
    
    async def read_output(self) -> AsyncIterator[str]:
        """Stream stdout/stderr from Claude Code."""
        ...
    
    async def terminate(self, timeout: int = 10):
        """Send SIGTERM, wait for graceful shutdown, fallback to SIGKILL."""
        ...
    
    def is_alive(self) -> bool:
        """Check if subprocess is running."""
        ...
```

### Telegram Bot Command Handlers

```python
from telegram import Update
from telegram.ext import ContextTypes

async def start_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /start command."""
    await update.message.reply_text(
        "👋 Welcome to omnik - Claude Code on Telegram\n\n"
        "Commands:\n"
        "/new [name] - Create new session\n"
        "/list - Show all sessions\n"
        "/help - Full command reference"
    )

async def new_session_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /new [name] command."""
    user_id = update.effective_user.id
    name = " ".join(context.args) if context.args else None
    
    session = await session_manager.create_session(user_id, name)
    
    await update.message.reply_text(
        f"✅ Created session `{session.id}`\n"
        f"Workspace: `{session.workspace_path}`\n\n"
        "Send a message to start coding!",
        parse_mode="Markdown"
    )

async def message_handler(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle non-command messages - forward to Claude Code."""
    user_id = update.effective_user.id
    message_text = update.message.text
    
    # Get active session for user
    active_session = await session_manager.get_active_session(user_id)
    
    if not active_session:
        await update.message.reply_text(
            "No active session. Create one with /new"
        )
        return
    
    # Show typing indicator
    await update.message.chat.send_action("typing")
    
    # Send to Claude Code and stream response
    sent_message = await update.message.reply_text("🤔 Processing...")
    
    buffer = ""
    last_edit = 0
    
    async for chunk in session_manager.send_message(active_session.id, message_text):
        buffer += chunk
        current_time = asyncio.get_event_loop().time()
        
        if current_time - last_edit > 0.5:
            try:
                await sent_message.edit_text(buffer[:4096])
                last_edit = current_time
            except Exception:
                pass  # Ignore "message not modified" errors
    
    # Final edit with complete response
    await sent_message.edit_text(buffer[:4096])
```

## Security Considerations

### Authentication and Authorization

**Whitelist enforcement:**
- Single authorized Telegram user ID configured via environment variable
- Reject all messages from unauthorized users with "Access denied" response
- Log unauthorized access attempts to audit log
- Optional: Support comma-separated list of authorized user IDs for team deployment

**Implementation:**
```python
def authorized_only(handler):
    async def wrapper(update: Update, context: ContextTypes.DEFAULT_TYPE):
        user_id = update.effective_user.id
        if user_id != config.AUTHORIZED_USER_ID:
            await update.message.reply_text("❌ Unauthorized")
            await audit_logger.log_unauthorized_access(user_id)
            return
        return await handler(update, context)
    return wrapper
```

### Container Security

**Multi-layer defense:**
- Non-root user (UID 1000) for all application processes
- Read-only root filesystem with specific writable mounts (/tmp, /workspace)
- Dropped capabilities (CAP_DROP ALL, add back only CHOWN, SETUID, SETGID)
- no-new-privileges security option prevents privilege escalation
- Resource limits (2 CPU cores, 4GB RAM max)
- Network isolation (no host network mode)

**No Docker socket mounting:**
- omnik manages Claude Code as subprocess, not separate containers
- Avoids Docker-in-Docker security risks
- If future container control needed, use docker-socket-proxy with minimal permissions

### Input Validation and Sanitization

**Path traversal prevention:**
```python
def sanitize_path(user_path: str, workspace_root: Path) -> Path:
    """Validate and sanitize user-provided paths."""
    # Remove dangerous sequences
    clean_path = user_path.replace("..", "").strip()
    
    # Resolve to absolute path
    abs_path = (workspace_root / clean_path).resolve()
    
    # Ensure within workspace
    if not abs_path.is_relative_to(workspace_root):
        raise SecurityError("Path outside workspace")
    
    return abs_path
```

**Command injection prevention:**
- Never use `shell=True` with subprocess
- Pass commands as list of arguments, not strings
- Validate all user inputs against allowlist patterns
- Claude Code CLI handles command execution in its own sandbox

### Secrets Management

**Docker Secrets pattern:**
```python
from pathlib import Path

def read_secret(secret_name: str) -> str:
    """Read Docker secret from /run/secrets/"""
    secret_path = Path("/run/secrets") / secret_name
    
    if secret_path.exists():
        return secret_path.read_text().strip()
    else:
        # Fallback to environment variable for development
        import os
        value = os.getenv(secret_name.upper())
        if value is None:
            raise ValueError(f"Secret {secret_name} not found")
        return value

# Usage
TELEGRAM_BOT_TOKEN = read_secret("telegram_bot_token")
ANTHROPIC_API_KEY = read_secret("anthropic_api_key")
```

### Audit Logging

**Comprehensive activity tracking:**
- Log every command execution with timestamp, user, workspace, and outcome
- Log file uploads/downloads with size and checksum
- Log session creation/termination events
- Log authentication failures
- Store logs in SQLite audit_logs table and optionally syslog
- Implement log rotation to prevent disk exhaustion (max 1GB logs)

**Sensitive data handling:**
- Never log API keys or tokens
- Redact sensitive file contents from logs
- Hash user messages before logging for privacy (optional)

## Future Enhancements

### Phase 2 (Post-MVP)

**Multi-user support:**
- User registration workflow in Telegram
- Per-user workspace isolation with quotas
- Admin panel for user management
- Shared workspaces with access control

**Advanced Claude Code integration:**
- Custom MCP (Model Context Protocol) tools via Python
- Project-specific AGENTS.md templates
- Git integration (commit, push, PR creation from Telegram)
- Automated testing and linting via quick actions

**Enhanced workspace features:**
- Workspace templates (Python, Node.js, Go projects)
- Automatic dependency installation detection
- Workspace snapshots and restore
- Collaboration mode (multiple users, same workspace)

### Phase 3 (Advanced)

**RAG and context enhancement:**
- Vector database integration (ChromaDB) for project indexing
- Semantic search across workspace files
- Automatic context augmentation from documentation
- Long-term memory store for preferences and patterns

**Monitoring and observability:**
- Prometheus metrics export (session count, token usage, latency)
- Grafana dashboards for usage visualization
- Alerting on error rates or resource exhaustion
- OpenTelemetry tracing for debugging

**AI model flexibility:**
- Support multiple LLM providers (OpenAI, Google, local models)
- Model switching per session or per query
- Cost optimization with model routing (cheap for simple, premium for complex)
- Fallback to alternative models on rate limits

**Integration ecosystem:**
- GitHub integration (open files from repo URLs)
- Linear/Jira ticket integration (create tasks from conversations)
- Slack/Discord mirroring for team notifications
- CI/CD webhooks (trigger deployments from omnik)

## Implementation Roadmap

### Milestone 1: Core Infrastructure (Weeks 1-2)
- Docker container setup with tini, non-root user, Node.js + Python
- SQLite database schema and migration system
- Basic Telegram bot with authentication and command routing
- Workspace directory creation and management

**Deliverable:** Bot responds to /start and /help, creates workspace directories

### Milestone 2: Claude Code Integration (Weeks 3-4)
- claude-agent-sdk integration for subprocess control
- Async subprocess management with real-time output streaming
- Session lifecycle (create, list, terminate)
- Message forwarding to Claude Code

**Deliverable:** Send messages to Claude, receive responses in Telegram

### Milestone 3: Session Persistence (Week 5)
- SQLite message storage and retrieval
- Session state persistence across restarts
- Context window management (50 message limit)
- Export conversations as Markdown

**Deliverable:** Conversations survive omnik restarts

### Milestone 4: Advanced Features (Week 6)
- File upload/download to workspace
- Directory navigation commands (/pwd, /ls, /cd)
- Multiple concurrent sessions
- Session switching

**Deliverable:** Full workspace interaction via Telegram

### Milestone 5: Polish and Production (Week 7-8)
- Comprehensive error handling and retry logic
- Health monitoring and auto-restart
- Audit logging for all operations
- Security hardening (read-only filesystem, capability dropping)
- Documentation and deployment guide

**Deliverable:** Production-ready deployment with docker-compose

## Success Criteria

**Technical metrics:**
- 99% uptime in single-user deployment
- <2s latency for first response token from Claude
- Zero message loss (all conversations persisted)
- Successful restart recovery for all sessions
- No security vulnerabilities in external audit

**User experience:**
- Natural conversation flow with Claude via Telegram
- Clear, actionable error messages (no raw stack traces)
- Intuitive command interface (10-minute onboarding for new users)
- Reliable file operations (upload/download success rate >99%)

**Performance:**
- Support 5 concurrent Claude Code sessions per container
- Handle 60 messages/minute burst load
- SQLite operations <100ms p95 latency
- Memory usage <512MB baseline, <4GB peak

## Appendix: Reference Architecture

### Comparison to Similar Projects

**claude-code-telegram (RichardAtCT):**
- Similarities: Claude Code subprocess management, SQLite persistence, Telegram interface
- Differences: omnik uses Pydantic AI, has stricter security (no Docker socket), better workspace isolation
- Learnings: Session export patterns, quick action buttons, audit logging design

**docker-controller-bot:**
- Similarities: Docker container deployment, multi-language support
- Differences: omnik focuses on coding assistance vs container ops
- Learnings: Label-based control, scheduled tasks, update notifications

**shell-bot (botgram):**
- Similarities: CLI tool control via Telegram
- Differences: omnik specializes in Claude Code vs general shell access
- Learnings: PTY emulation for live output, escape sequence handling

### Key Design Decisions

**Why Python over Node.js:**
- python-telegram-bot more mature than Node alternatives (Telegraf)
- Pydantic AI provides excellent type safety and agent framework
- Easier async subprocess management with asyncio
- Better data science/ML ecosystem for future enhancements

**Why SQLite over PostgreSQL:**
- Single-user deployment doesn't need distributed database
- Embedded database simplifies deployment (no external services)
- File-based backup and restore trivial
- Sufficient performance for conversational workload (<10 queries/sec)

**Why not Docker-in-Docker:**
- Security risk of mounting Docker socket
- Subprocess management simpler and more portable
- Avoids nested container complexity
- Claude Code designed to run directly, not in separate containers

**Why Pydantic AI over LangChain:**
- Lighter weight, less opinionated framework
- Better TypeScript-style type safety with Pydantic
- Simpler streaming API
- FastAPI-style developer experience

## Conclusion

omnik bridges the gap between powerful AI coding assistance (Claude Code) and ubiquitous mobile communication (Telegram), wrapped in a secure, containerized deployment. By focusing on single-user, Docker-based architecture with robust subprocess management and persistent state, omnik enables developers to code from anywhere with just a smartphone.

The architecture leverages proven patterns from similar open-source projects while adding unique innovations: Pydantic AI for type-safe agent orchestration, claude-agent-sdk for reliable subprocess control, and multi-workspace isolation for project switching. Security is paramount with whitelist authentication, read-only containers, and comprehensive audit logging.

With an 8-week development timeline and clear milestones, omnik can move from concept to production-ready deployment, providing immediate value to remote developers, on-call engineers, and mobile-first practitioners.

**Next steps:** Set up development environment, implement Milestone 1 (core infrastructure), and begin user testing with alpha deployment.