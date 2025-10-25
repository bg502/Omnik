# Product Requirements Document: Omnik Telegram Bot Agent

## 1. Project Overview

**Project Name:** Omnik Telegram Bot  
**Version:** 1.0  
**Date:** October 24, 2025  
**Status:** Initial Release

### 1.1 Purpose
Create a Telegram bot interface for the Omnik containerized coding agent that allows users to interact with Claude Code Agent SDK via Telegram messages, with integration to Archon MCP for web-based task management and execution.

### 1.2 Background
Omnik is a container-based coding agent system. This implementation extends the system by providing a Telegram bot interface built with Python and Claude Code Agent SDK. The bot runs in a Docker container with a process manager that can restart Claude Code as needed (e.g., to drop context). The system integrates with Archon MCP service for web UI capabilities, allowing users to define tasks in the web interface and trigger execution from Telegram.

### 1.3 Key Differences from Web UI Approach
- Telegram Bot API instead of custom web server
- Python-based using Claude Code Agent SDK
- Process manager (supervisord) manages Claude Code lifecycle
- Integration with Archon MCP for web UI functionality
- Shared state and file system between services
- Asynchronous message handling

---

## 2. Objectives

### 2.1 Primary Goals
- Provide Telegram-based access to Claude Code agent
- Enable file system operations (list, navigate, reference files)
- Allow agent context reset via process restart
- Integrate with Archon MCP for task definition and execution
- Support both interactive conversations and task execution
- Maintain conversation context across messages
- Handle long-running tasks with progress updates

### 2.2 Success Criteria
- Users can successfully send commands via Telegram and receive responses
- Agent can perform coding tasks and report results
- File operations work correctly (list, cd, reference files)
- Process restart successfully drops context without data loss
- Integration with Archon MCP enables web-based task triggering
- Response time for simple commands < 3 seconds
- Long-running tasks provide status updates
- System handles concurrent users (if multiple users authorized)

---

## 3. Scope

### 3.1 In Scope
- Python-based Telegram bot using python-telegram-bot library
- Claude Code Agent SDK integration
- Process manager (supervisord) for Claude Code lifecycle
- File system navigation and operations
- Conversation context management
- Task execution from Telegram commands
- Integration with Archon MCP via shared network
- Task definition in Archon, execution trigger from Telegram
- Progress updates for long-running tasks
- Error handling and retry logic
- User authentication (Telegram user ID whitelist)
- Docker compose setup with both services
- Shared volumes for workdir and state
- Command parsing and validation
- State persistence across restarts

### 3.2 Out of Scope (Future Iterations)
- Multiple agent instances per user
- Voice message support
- Image/diagram generation
- Real-time collaboration between users
- Complex workflow orchestration
- Advanced scheduling/cron jobs
- Web dashboard for bot management
- Metrics and analytics dashboard
- Multi-language support (English only in v1)

---

## 4. User Stories

### 4.1 As a Developer Using Telegram
- I want to send a coding task via Telegram so that the agent starts working on it
- I want to check the status of my current task so that I know what's happening
- I want to list files in my workspace so that I can see what the agent created
- I want to navigate between directories so that I can reference specific files
- I want to ask the agent to explain or modify code in a specific file
- I want to reset the agent's context so that I can start fresh if needed
- I want to receive progress updates so that I know the agent is still working
- I want to cancel a running task if I change my mind

### 4.2 As a Developer Using Both Telegram and Archon
- I want to define complex tasks in Archon web UI so that I can use a better interface for planning
- I want to trigger task execution from Telegram so that I don't need to open the web browser
- I want to check task status in either Telegram or Archon so that I have flexibility
- I want to review task results in Archon so that I can see formatted output

### 4.3 As a System Administrator
- I want to whitelist specific Telegram users so that only authorized people can use the bot
- I want to monitor bot health so that I can ensure it's running properly
- I want to restart the agent process so that I can recover from errors
- I want to view logs so that I can debug issues

---

## 5. Functional Requirements

### 5.1 Telegram Bot Commands

#### FR-1: Basic Commands
- **Priority:** P0 (Critical)

| Command | Description | Example |
|---------|-------------|---------|
| `/start` | Initialize bot and show help | `/start` |
| `/help` | Display available commands | `/help` |
| `/status` | Show agent status and current task | `/status` |
| `/task <description>` | Execute a coding task | `/task create a REST API with FastAPI` |
| `/cancel` | Cancel current task | `/cancel` |
| `/reset` | Reset agent context (restart process) | `/reset` |

#### FR-2: File System Commands
- **Priority:** P0 (Critical)

| Command | Description | Example |
|---------|-------------|---------|
| `/ls [path]` | List files in directory | `/ls` or `/ls src/` |
| `/cd <path>` | Change working directory | `/cd projects/myapp` |
| `/pwd` | Show current directory | `/pwd` |
| `/cat <file>` | Show file contents | `/cat main.py` |
| `/tree [depth]` | Show directory tree | `/tree 2` |

#### FR-3: Agent Management Commands
- **Priority:** P1 (High)

| Command | Description | Example |
|---------|-------------|---------|
| `/context` | Show current conversation context | `/context` |
| `/workdir` | Show workdir root path | `/workdir` |
| `/projects` | List all project directories | `/projects` |
| `/health` | Check system health | `/health` |

#### FR-4: Archon Integration Commands
- **Priority:** P1 (High)

| Command | Description | Example |
|---------|-------------|---------|
| `/tasks` | List tasks from Archon | `/tasks` |
| `/run <task_id>` | Execute task from Archon | `/run task-123` |
| `/result <task_id>` | Get task result | `/result task-123` |

### 5.2 Message Handling

#### FR-5: Interactive Conversation
- **Priority:** P0 (Critical)
- Bot should accept free-form messages (not just commands)
- Maintain conversation context with Claude Code agent
- Support multi-turn conversations
- Handle follow-up questions and clarifications
- Preserve context until `/reset` or timeout

#### FR-6: Long Message Support
- **Priority:** P1 (High)
- Handle messages up to Telegram limit (4096 chars)
- Support message splitting for long responses
- Provide "Continue..." option for truncated responses
- Allow file upload for long input (e.g., requirements doc)

#### FR-7: Progress Updates
- **Priority:** P1 (High)
- Show typing indicator while processing
- Send progress messages for long-running tasks (>10 seconds)
- Update messages in-place when possible
- Notify when task completes or fails

### 5.3 File Operations

#### FR-8: File System Navigation
- **Priority:** P0 (Critical)
- Maintain current working directory per user session
- Support absolute and relative paths
- Validate paths to prevent directory traversal attacks
- Show current directory in status command

#### FR-9: File Reading
- **Priority:** P1 (High)
- Read and display text files
- Syntax highlighting for code files (using markdown)
- Handle binary files gracefully (show file info, not contents)
- Limit file size for display (max 50KB direct, larger files via link)

#### FR-10: File Listing
- **Priority:** P1 (High)
- List files with size and modification time
- Support sorting (name, date, size)
- Hide system files by default (option to show)
- Display git status if in git repository

### 5.4 Agent Process Management

#### FR-11: Process Lifecycle
- **Priority:** P0 (Critical)
- Run Claude Code agent as child process
- Use supervisord or similar to manage process
- Bot (Python) should be PID 1 (or supervisord as PID 1)
- Claude Code should be restartable without container restart
- Capture stdout/stderr from Claude Code process
- Handle process crashes gracefully

#### FR-12: Context Reset
- **Priority:** P1 (High)
- `/reset` command triggers process restart
- Save current state before restart
- Clear conversation history
- Preserve file system and workdir
- Notify user when reset complete
- Allow optional reason parameter: `/reset "starting new project"`

#### FR-13: Process Health Monitoring
- **Priority:** P1 (High)
- Monitor Claude Code process health
- Auto-restart if process dies unexpectedly
- Log all restarts with reason
- Alert user if auto-restart occurs
- Implement circuit breaker for repeated crashes

### 5.5 Archon MCP Integration

#### FR-14: Task Synchronization
- **Priority:** P1 (High)
- Read tasks from Archon via REST API or shared database
- Support task creation from Telegram
- Sync task status bidirectionally
- Update Archon when task execution completes

#### FR-15: Shared Resources
- **Priority:** P1 (High)
- Share workdir volume between bot and Archon
- Share state database or file system
- Use shared network for inter-service communication
- Coordinate agent access (prevent concurrent execution conflicts)

#### FR-16: Web UI Integration
- **Priority:** P2 (Medium)
- Provide link to Archon web UI in messages
- Support deep linking to specific tasks
- Show task details from Archon in Telegram
- Allow task triggering from Telegram with Archon-defined parameters

### 5.6 Authentication and Authorization

#### FR-17: User Authentication
- **Priority:** P0 (Critical)
- Whitelist Telegram user IDs in configuration
- Reject messages from unauthorized users
- Support admin users with elevated permissions
- Allow multiple authorized users (optional)

#### FR-18: Rate Limiting
- **Priority:** P1 (High)
- Limit messages per user (e.g., 30/minute)
- Limit concurrent tasks per user (1 by default)
- Queue additional requests
- Notify user when rate limit hit

### 5.7 Error Handling

#### FR-19: Error Responses
- **Priority:** P1 (High)
- User-friendly error messages
- Specific errors for common issues:
  - Agent busy with another task
  - File not found
  - Permission denied
  - Invalid command syntax
  - Agent process crashed
- Provide recovery suggestions
- Log detailed errors for debugging

#### FR-20: Retry Logic
- **Priority:** P2 (Medium)
- Auto-retry on transient failures
- Exponential backoff for retries
- Max 3 retry attempts
- Notify user if all retries fail

---

## 6. Technical Requirements

### 6.1 Technology Stack

#### TR-1: Core Components
- **Python 3.11+** - Main programming language
- **python-telegram-bot 20.0+** - Telegram Bot API wrapper
- **Claude Code Agent SDK** - Agent integration (via subprocess or SDK if available)
- **supervisord** - Process manager for Claude Code
- **asyncio** - Asynchronous message handling

#### TR-2: Additional Libraries
```python
# Core
python-telegram-bot==20.7
python-dotenv==1.0.0
pydantic==2.5.0

# Process management
supervisor==4.2.5

# File system
pathlib (built-in)
aiofiles==23.2.1

# HTTP client (for Archon API)
httpx==0.25.2

# Database (optional, for state)
sqlalchemy==2.0.23
aiosqlite==0.19.0

# Logging
structlog==23.2.0

# Utilities
pyyaml==6.0.1
```

#### TR-3: Claude Code Agent SDK Integration
- Use Claude Code Agent SDK if available as Python package
- Otherwise, use subprocess to execute `claude` CLI
- Capture stdout/stderr in real-time
- Parse agent output for structured responses
- Handle streaming responses

### 6.2 Architecture

#### TR-4: System Architecture

```
┌──────────────────┐
│  Telegram User   │
└────────┬─────────┘
         │ HTTPS (Telegram API)
         ↓
┌──────────────────────────────────────┐
│         Docker Host (Cloud VM)       │
│                                      │
│  ┌────────────────────────────────┐ │
│  │   docker-compose services      │ │
│  │                                │ │
│  │  ┌──────────────────────────┐ │ │
│  │  │   omnik-telegram-bot     │ │ │
│  │  │  ┌────────────────────┐  │ │ │
│  │  │  │   supervisord      │  │ │ │
│  │  │  │   (PID 1)          │  │ │ │
│  │  │  │                    │  │ │ │
│  │  │  │  ├─ Python Bot     │  │ │ │
│  │  │  │  │  (telegram)     │  │ │ │
│  │  │  │  │                 │  │ │ │
│  │  │  │  └─ Claude Code    │  │ │ │
│  │  │  │     (agent)        │  │ │ │
│  │  │  └────────────────────┘  │ │ │
│  │  │                          │ │ │
│  │  │  Volumes:                │ │ │
│  │  │  - workdir:/workspace   │ │ │
│  │  │  - state:/state         │ │ │
│  │  └──────────┬───────────────┘ │ │
│  │             │ shared_network   │ │
│  │             │                  │ │
│  │  ┌──────────┴───────────────┐ │ │
│  │  │   archon-mcp             │ │ │
│  │  │   (Web UI)               │ │ │
│  │  │                          │ │ │
│  │  │  - HTTP Server           │ │ │
│  │  │  - Task Manager          │ │ │
│  │  │  - API Endpoints         │ │ │
│  │  │                          │ │ │
│  │  │  Volumes:                │ │ │
│  │  │  - workdir:/workspace   │ │ │
│  │  │  - state:/state         │ │ │
│  │  └──────────────────────────┘ │ │
│  │                                │ │
│  └────────────────────────────────┘ │
│                                      │
└──────────────────────────────────────┘
```

#### TR-5: Component Structure

```
omnik-telegram-bot/
├── src/
│   ├── bot/
│   │   ├── __init__.py
│   │   ├── main.py                  # Bot entry point
│   │   ├── handlers.py              # Command and message handlers
│   │   ├── commands.py              # Command implementations
│   │   ├── middleware.py            # Auth, logging middleware
│   │   └── formatters.py            # Response formatting
│   ├── agent/
│   │   ├── __init__.py
│   │   ├── manager.py               # Agent process manager
│   │   ├── interface.py             # Claude Code SDK interface
│   │   ├── executor.py              # Task execution
│   │   └── context.py               # Context management
│   ├── filesystem/
│   │   ├── __init__.py
│   │   ├── navigator.py             # Directory navigation
│   │   ├── operations.py            # File operations
│   │   └── validators.py            # Path validation
│   ├── archon/
│   │   ├── __init__.py
│   │   ├── client.py                # Archon API client
│   │   ├── tasks.py                 # Task management
│   │   └── sync.py                  # State synchronization
│   ├── state/
│   │   ├── __init__.py
│   │   ├── session.py               # User session management
│   │   ├── storage.py               # State persistence
│   │   └── models.py                # Data models
│   ├── utils/
│   │   ├── __init__.py
│   │   ├── logger.py                # Logging setup
│   │   ├── config.py                # Configuration
│   │   └── helpers.py               # Utility functions
│   └── types/
│       ├── __init__.py
│       └── models.py                # Pydantic models
├── config/
│   ├── supervisord.conf             # Supervisor configuration
│   ├── bot.yaml                     # Bot configuration
│   └── authorized_users.yaml        # User whitelist
├── tests/
│   ├── unit/
│   ├── integration/
│   └── fixtures/
├── Dockerfile
├── docker-compose.yml
├── requirements.txt
├── README.md
└── .env.example
```

### 6.3 Data Models

#### TR-6: User Session Model
```python
from pydantic import BaseModel
from datetime import datetime
from typing import Optional, List

class UserSession(BaseModel):
    user_id: int
    chat_id: int
    username: Optional[str]
    current_directory: str
    conversation_context: List[dict]
    active_task_id: Optional[str]
    created_at: datetime
    last_activity: datetime
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }
```

#### TR-7: Task Model
```python
class Task(BaseModel):
    id: str
    user_id: int
    description: str
    status: str  # pending, running, completed, failed, cancelled
    result: Optional[str]
    created_at: datetime
    started_at: Optional[datetime]
    completed_at: Optional[datetime]
    error: Optional[str]
    source: str  # telegram, archon
    archon_task_id: Optional[str]
```

#### TR-8: Agent State Model
```python
class AgentState(BaseModel):
    process_id: Optional[int]
    status: str  # idle, busy, crashed, restarting
    current_task_id: Optional[str]
    last_restart: Optional[datetime]
    restart_count: int
    uptime_seconds: int
```

### 6.4 Process Management

#### TR-9: Supervisord Configuration
```ini
[supervisord]
nodaemon=true
user=root

[program:telegram_bot]
command=python -m src.bot.main
directory=/app
user=agent
autostart=true
autorestart=true
stderr_logfile=/var/log/supervisor/bot.err.log
stdout_logfile=/var/log/supervisor/bot.out.log
environment=PYTHONUNBUFFERED=1

[program:claude_code]
command=bash -c "exec claude code --session-file /state/claude_session.json"
directory=/workspace
user=agent
autostart=false
autorestart=false
stderr_logfile=/var/log/supervisor/claude.err.log
stdout_logfile=/var/log/supervisor/claude.out.log
stopwaitsecs=10

[supervisorctl]
serverurl=unix:///var/run/supervisor.sock

[unix_http_server]
file=/var/run/supervisor.sock
chmod=0700

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface
```

#### TR-10: Agent Process Manager
```python
class AgentManager:
    def __init__(self):
        self.supervisor = xmlrpc.client.ServerProxy(
            'http://localhost',
            transport=SupervisorTransport(
                None, None, 'unix:///var/run/supervisor.sock'
            )
        )
    
    async def start_agent(self) -> bool:
        """Start Claude Code process"""
        pass
    
    async def stop_agent(self) -> bool:
        """Stop Claude Code process"""
        pass
    
    async def restart_agent(self, reason: str = "") -> bool:
        """Restart Claude Code process"""
        pass
    
    async def get_status(self) -> AgentState:
        """Get current agent status"""
        pass
    
    async def is_healthy(self) -> bool:
        """Check if agent is responding"""
        pass
```

### 6.5 Telegram Bot Implementation

#### TR-11: Bot Setup
```python
from telegram.ext import (
    Application,
    CommandHandler,
    MessageHandler,
    filters
)

def create_app() -> Application:
    """Create and configure bot application"""
    app = Application.builder().token(config.TELEGRAM_TOKEN).build()
    
    # Command handlers
    app.add_handler(CommandHandler("start", start_command))
    app.add_handler(CommandHandler("help", help_command))
    app.add_handler(CommandHandler("status", status_command))
    app.add_handler(CommandHandler("task", task_command))
    app.add_handler(CommandHandler("cancel", cancel_command))
    app.add_handler(CommandHandler("reset", reset_command))
    
    # File system commands
    app.add_handler(CommandHandler("ls", ls_command))
    app.add_handler(CommandHandler("cd", cd_command))
    app.add_handler(CommandHandler("pwd", pwd_command))
    app.add_handler(CommandHandler("cat", cat_command))
    app.add_handler(CommandHandler("tree", tree_command))
    
    # Archon commands
    app.add_handler(CommandHandler("tasks", tasks_command))
    app.add_handler(CommandHandler("run", run_command))
    
    # Message handler for conversation
    app.add_handler(MessageHandler(
        filters.TEXT & ~filters.COMMAND,
        message_handler
    ))
    
    # Middleware
    app.add_handler(AuthMiddleware())
    
    return app
```

#### TR-12: Command Handler Pattern
```python
async def task_command(update: Update, context: ContextTypes.DEFAULT_TYPE):
    """Handle /task command"""
    user_id = update.effective_user.id
    
    # Validate authorization
    if not is_authorized(user_id):
        await update.message.reply_text("Unauthorized")
        return
    
    # Get task description
    task_description = " ".join(context.args)
    if not task_description:
        await update.message.reply_text(
            "Usage: /task <description>\n"
            "Example: /task create a FastAPI app with PostgreSQL"
        )
        return
    
    # Check if agent is busy
    agent_state = await agent_manager.get_status()
    if agent_state.status == "busy":
        await update.message.reply_text(
            f"⏳ Agent is busy with task: {agent_state.current_task_id}\n"
            "Use /cancel to stop current task or wait for completion."
        )
        return
    
    # Create and execute task
    task = Task(
        id=generate_task_id(),
        user_id=user_id,
        description=task_description,
        status="pending",
        created_at=datetime.now(),
        source="telegram"
    )
    
    # Send initial response
    message = await update.message.reply_text(
        f"🚀 Starting task: {task.description}\n"
        f"Task ID: {task.id}"
    )
    
    # Execute task asynchronously
    asyncio.create_task(
        execute_task_with_updates(task, update.effective_chat.id, message.message_id)
    )
```

### 6.6 Archon Integration

#### TR-13: Archon API Client
```python
class ArchonClient:
    def __init__(self, base_url: str):
        self.base_url = base_url
        self.client = httpx.AsyncClient(base_url=base_url)
    
    async def list_tasks(self, user_id: int) -> List[ArchonTask]:
        """Get tasks from Archon"""
        response = await self.client.get(
            f"/api/tasks",
            params={"user_id": user_id}
        )
        return [ArchonTask(**t) for t in response.json()]
    
    async def get_task(self, task_id: str) -> ArchonTask:
        """Get specific task details"""
        response = await self.client.get(f"/api/tasks/{task_id}")
        return ArchonTask(**response.json())
    
    async def update_task_status(
        self, 
        task_id: str, 
        status: str,
        result: Optional[str] = None
    ):
        """Update task status in Archon"""
        await self.client.patch(
            f"/api/tasks/{task_id}",
            json={"status": status, "result": result}
        )
    
    async def create_task(self, task: Task) -> str:
        """Create task in Archon"""
        response = await self.client.post(
            "/api/tasks",
            json=task.dict()
        )
        return response.json()["id"]
```

#### TR-14: Task Synchronization
- Tasks created in Telegram are synced to Archon
- Tasks created in Archon can be executed from Telegram
- Status updates are bidirectional
- Results are stored in both systems
- Use webhooks or polling for sync

---

## 7. Docker Configuration

### 7.1 Dockerfile

#### TR-15: Bot Container Dockerfile
```dockerfile
FROM python:3.11-slim-bookworm

# Install system dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    supervisor \
    curl \
    git \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js for Claude Code (if needed)
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g @anthropic-ai/claude-code

# Create agent user
ARG USER=agent
ARG UID=1000
ARG GID=1000

RUN groupadd -g "${GID}" "${USER}" \
    && useradd -m -u "${UID}" -g "${GID}" -s /bin/bash "${USER}"

# Set working directory
WORKDIR /app

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application code
COPY src/ ./src/
COPY config/ ./config/

# Copy supervisor configuration
COPY config/supervisord.conf /etc/supervisor/conf.d/supervisord.conf

# Create necessary directories
RUN mkdir -p /workspace /state /var/log/supervisor \
    && chown -R agent:agent /app /workspace /state /var/log/supervisor

# Create volume mount points
VOLUME ["/workspace", "/state"]

# Expose supervisor HTTP API (optional)
EXPOSE 9001

# Use supervisor as PID 1
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisor/conf.d/supervisord.conf"]
```

### 7.2 Docker Compose Configuration

#### TR-16: Complete docker-compose.yml
```yaml
version: '3.8'

services:
  telegram-bot:
    build: 
      context: ./telegram-bot
      dockerfile: Dockerfile
    container_name: omnik-telegram-bot
    networks:
      - omnik-network
    volumes:
      - workdir:/workspace
      - state:/state
      - ./config/bot.yaml:/app/config/bot.yaml:ro
      - ./config/authorized_users.yaml:/app/config/authorized_users.yaml:ro
      - /var/run/docker.sock:/var/run/docker.sock  # If agent needs Docker access
    environment:
      - TELEGRAM_TOKEN=${TELEGRAM_BOT_TOKEN}
      - ARCHON_URL=http://archon-mcp:8080
      - ANTHROPIC_API_KEY=${ANTHROPIC_API_KEY}
      - LOG_LEVEL=INFO
      - WORKSPACE_DIR=/workspace
      - STATE_DIR=/state
    restart: unless-stopped
    depends_on:
      - archon-mcp

  archon-mcp:
    image: archon-mcp:latest  # Or build from source
    container_name: omnik-archon
    networks:
      - omnik-network
      - proxy  # External network for web access
    volumes:
      - workdir:/workspace
      - state:/state
      - archon-data:/app/data
    environment:
      - VIRTUAL_HOST=omnik.bg502.ru
      - VIRTUAL_PORT=8080
      - DATABASE_URL=sqlite:////state/archon.db
      - WORKSPACE_DIR=/workspace
    ports:
      - "8080:8080"  # Internal port, proxied via nginx
    restart: unless-stopped

networks:
  omnik-network:
    driver: bridge
  proxy:
    external: true

volumes:
  workdir:
    driver: local
  state:
    driver: local
  archon-data:
    driver: local
```

### 7.3 Environment Variables

#### TR-17: .env.example
```bash
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_bot_token_here
TELEGRAM_ADMIN_USER_ID=123456789

# Anthropic API
ANTHROPIC_API_KEY=your_anthropic_api_key_here

# Archon Configuration
ARCHON_URL=http://archon-mcp:8080
ARCHON_API_KEY=optional_api_key

# Application
LOG_LEVEL=INFO
WORKSPACE_DIR=/workspace
STATE_DIR=/state
MAX_CONCURRENT_TASKS=1
SESSION_TIMEOUT_MINUTES=30

# Optional: Docker access
DOCKER_HOST=unix:///var/run/docker.sock
```

---

## 8. File System and State Management

### 8.1 Directory Structure

#### TR-18: Workspace Layout
```
/workspace/                    # Shared workdir volume
├── projects/                  # User projects
│   ├── project1/
│   │   ├── .git/
│   │   └── src/
│   └── project2/
├── temp/                      # Temporary files
└── shared/                    # Shared resources

/state/                        # Shared state volume
├── sessions/                  # User sessions
│   ├── user_123456789.json
│   └── user_987654321.json
├── tasks/                     # Task data
│   ├── task-abc123.json
│   └── task-def456.json
├── claude_session.json        # Claude Code session
├── archon.db                  # Archon database
└── locks/                     # Coordination locks
```

### 8.2 State Persistence

#### TR-19: Session Storage
```python
class SessionStorage:
    def __init__(self, state_dir: Path):
        self.state_dir = state_dir / "sessions"
        self.state_dir.mkdir(parents=True, exist_ok=True)
    
    async def save_session(self, session: UserSession):
        """Save user session to disk"""
        path = self.state_dir / f"user_{session.user_id}.json"
        async with aiofiles.open(path, 'w') as f:
            await f.write(session.json(indent=2))
    
    async def load_session(self, user_id: int) -> Optional[UserSession]:
        """Load user session from disk"""
        path = self.state_dir / f"user_{user_id}.json"
        if not path.exists():
            return None
        
        async with aiofiles.open(path, 'r') as f:
            content = await f.read()
            return UserSession.parse_raw(content)
    
    async def delete_session(self, user_id: int):
        """Delete user session"""
        path = self.state_dir / f"user_{user_id}.json"
        if path.exists():
            path.unlink()
```

---

## 9. Security Requirements

### 9.1 Authentication

#### SEC-1: User Authorization
- Whitelist Telegram user IDs in `authorized_users.yaml`
- Environment variable for admin user ID
- Reject all unauthorized requests immediately
- Log unauthorized access attempts

```yaml
# authorized_users.yaml
authorized_users:
  - user_id: 123456789
    username: "john_doe"
    role: "admin"
    allowed_commands: ["all"]
  
  - user_id: 987654321
    username: "jane_smith"
    role: "user"
    allowed_commands: ["task", "status", "ls", "cd", "pwd"]
```

### 9.2 API Security

#### SEC-2: Telegram Token Security
- Store token in environment variable (never in code)
- Use `.env` file for local development
- Use secrets management for production (Docker secrets, Vault)
- Rotate token if compromised

#### SEC-3: Archon API Security
- Authenticate requests to Archon with API key or JWT
- Validate responses from Archon
- Use HTTPS for Archon communication if exposed externally

### 9.3 File System Security

#### SEC-4: Path Validation
- Prevent directory traversal (`../../../etc/passwd`)
- Restrict access to `/workspace` directory only
- Validate all user-provided paths
- Sanitize file names

```python
def validate_path(requested_path: str, workspace_root: Path) -> Path:
    """Validate and resolve path safely"""
    # Resolve to absolute path
    abs_path = (workspace_root / requested_path).resolve()
    
    # Ensure path is within workspace
    if not str(abs_path).startswith(str(workspace_root)):
        raise ValueError("Path outside workspace")
    
    return abs_path
```

#### SEC-5: Command Injection Prevention
- Never use `os.system()` or `shell=True`
- Use subprocess with argument list
- Validate all command parameters
- Sanitize agent input/output

### 9.4 Rate Limiting

#### SEC-6: Request Rate Limits
- 30 messages per minute per user
- 5 task executions per hour per user
- 3 context resets per hour per user
- Exponential backoff for violations

### 9.5 Logging and Monitoring

#### SEC-7: Security Logging
- Log all authentication attempts
- Log all command executions
- Log file access operations
- Log agent restarts and crashes
- Sanitize sensitive data in logs (tokens, API keys)

---

## 10. Error Handling and Resilience

### 10.1 Agent Process Failures

#### ERR-1: Process Crash Handling
- Detect when Claude Code process dies
- Auto-restart with exponential backoff
- Maximum 5 restart attempts in 5 minutes
- Circuit breaker opens after max attempts
- Notify user of crash and recovery

#### ERR-2: Timeout Handling
- Task timeout: 10 minutes default
- Configurable per task type
- Send progress updates every 30 seconds
- Cancel task if timeout exceeded
- Save partial results

### 10.2 Network Failures

#### ERR-3: Telegram API Failures
- Retry failed message sends (3 attempts)
- Queue messages if Telegram unreachable
- Graceful degradation (log locally if can't send)

#### ERR-4: Archon Communication Failures
- Retry Archon API calls (3 attempts with backoff)
- Cache last known state
- Continue Telegram operation if Archon unavailable
- Sync when connection restored

### 10.3 File System Errors

#### ERR-5: File Operation Failures
- Handle permission denied errors
- Handle disk full errors
- Validate file existence before operations
- Provide clear error messages to user

---

## 11. Command Reference

### 11.1 Core Commands

#### `/start`
**Description:** Initialize bot and show welcome message  
**Usage:** `/start`  
**Response:**
```
👋 Welcome to Omnik Telegram Bot!

I'm your remote coding agent powered by Claude Code.

Available commands:
/help - Show this message
/task <description> - Execute a coding task
/status - Show current status
/reset - Reset agent context

Try /task create a Python hello world script
```

#### `/help`
**Description:** Display command reference  
**Usage:** `/help [command]`  
**Examples:**
- `/help` - Show all commands
- `/help task` - Show task command details

#### `/status`
**Description:** Show agent and task status  
**Usage:** `/status`  
**Response:**
```
📊 Agent Status

Agent: 🟢 Idle
Uptime: 2h 34m
Working Directory: /workspace/projects/myapp

Current Task: None
Last Task: #task-abc123 (completed 5m ago)

Archon: Connected
Tasks in Queue: 0
```

#### `/task <description>`
**Description:** Execute a coding task  
**Usage:** `/task <description>`  
**Examples:**
- `/task create a FastAPI REST API with CRUD operations`
- `/task add unit tests to the UserService class`
- `/task refactor the authentication module to use JWT`

**Response:**
```
🚀 Starting task: create a FastAPI REST API

Task ID: task-abc123
Status: Running...

[Progress updates sent as task executes]

✅ Task completed!

Created files:
- main.py
- models.py
- routes.py
- requirements.txt

Summary: Created a FastAPI application with...
```

#### `/cancel`
**Description:** Cancel currently running task  
**Usage:** `/cancel`  
**Response:**
```
🛑 Cancelling task: task-abc123

Task cancelled successfully.
Partial results saved to: /workspace/projects/myapp
```

#### `/reset [reason]`
**Description:** Reset agent context (restart process)  
**Usage:** `/reset [optional reason]`  
**Examples:**
- `/reset`
- `/reset starting new project`

**Response:**
```
🔄 Resetting agent context...

Reason: starting new project
Saving current state...
Restarting Claude Code process...

✅ Agent reset complete!

Context cleared.
Working directory preserved: /workspace/projects/myapp
Ready for new tasks.
```

### 11.2 File System Commands

#### `/ls [path]`
**Description:** List directory contents  
**Usage:** `/ls [path]`  
**Examples:**
- `/ls` - List current directory
- `/ls src/` - List src directory
- `/ls ../` - List parent directory

**Response:**
```
📁 /workspace/projects/myapp/

Directories:
  src/
  tests/
  .git/

Files:
  README.md (2.3 KB) - 2 hours ago
  requirements.txt (456 B) - 1 hour ago
  main.py (5.1 KB) - 30 minutes ago
  .gitignore (234 B) - 2 hours ago

Total: 3 directories, 4 files
```

#### `/cd <path>`
**Description:** Change working directory  
**Usage:** `/cd <path>`  
**Examples:**
- `/cd src/`
- `/cd ../tests/`
- `/cd /workspace/projects/project2`

**Response:**
```
✅ Changed directory to: /workspace/projects/myapp/src
```

#### `/pwd`
**Description:** Show current working directory  
**Usage:** `/pwd`  
**Response:**
```
📍 Current directory: /workspace/projects/myapp/src
```

#### `/cat <file>`
**Description:** Display file contents  
**Usage:** `/cat <file>`  
**Examples:**
- `/cat main.py`
- `/cat ../README.md`

**Response:**
````
📄 main.py

```python
from fastapi import FastAPI

app = FastAPI()

@app.get("/")
async def root():
    return {"message": "Hello World"}
```

Lines: 7 | Size: 156 B
````

#### `/tree [depth]`
**Description:** Show directory tree  
**Usage:** `/tree [depth]`  
**Examples:**
- `/tree` - Show tree (default depth 3)
- `/tree 2` - Show tree depth 2

**Response:**
```
🌲 Directory tree: /workspace/projects/myapp

myapp/
├── src/
│   ├── main.py
│   ├── models.py
│   └── routes/
│       ├── __init__.py
│       └── users.py
├── tests/
│   └── test_main.py
├── README.md
└── requirements.txt

2 directories, 6 files
```

### 11.3 Agent Management Commands

#### `/context`
**Description:** Show conversation context summary  
**Usage:** `/context`  
**Response:**
```
💭 Conversation Context

Messages: 12
Started: 45 minutes ago
Last activity: 2 minutes ago

Recent topics:
- FastAPI application structure
- Database models
- User authentication

Use /reset to clear context
```

#### `/workdir`
**Description:** Show workspace root  
**Usage:** `/workdir`  
**Response:**
```
🗂️ Workspace: /workspace
Current: /workspace/projects/myapp

Available space: 45.2 GB / 100 GB
```

#### `/projects`
**Description:** List all projects  
**Usage:** `/projects`  
**Response:**
```
📂 Projects in /workspace/projects/

1. myapp/ (modified 5m ago)
   FastAPI REST API project

2. data-pipeline/ (modified 2d ago)
   ETL pipeline with Apache Airflow

3. ml-model/ (modified 1w ago)
   Machine learning model training

Total: 3 projects
```

#### `/health`
**Description:** System health check  
**Usage:** `/health`  
**Response:**
```
🏥 System Health

Telegram Bot: 🟢 Healthy
Claude Code Agent: 🟢 Running
Archon MCP: 🟢 Connected
File System: 🟢 OK (45GB free)

CPU: 23% | Memory: 1.2GB / 4GB
Uptime: 2d 14h 32m

Last restart: 2 days ago (scheduled maintenance)
Restarts today: 0
```

### 11.4 Archon Integration Commands

#### `/tasks`
**Description:** List tasks from Archon  
**Usage:** `/tasks [filter]`  
**Examples:**
- `/tasks` - List all tasks
- `/tasks pending` - List pending tasks
- `/tasks completed` - List completed tasks

**Response:**
```
📋 Tasks from Archon

Pending:
  #arch-001: Setup CI/CD pipeline
  #arch-002: Add monitoring dashboard

Running:
  #arch-003: Implement caching layer (started 10m ago)

Completed Today:
  #arch-004: Fix authentication bug ✅
  #arch-005: Update dependencies ✅

Use /run <task_id> to execute a task
Use /result <task_id> to see task details
```

#### `/run <task_id>`
**Description:** Execute task from Archon  
**Usage:** `/run <task_id>`  
**Examples:**
- `/run arch-001`
- `/run #arch-002`

**Response:**
```
🚀 Running task from Archon: #arch-001

Task: Setup CI/CD pipeline
Source: Archon
Priority: High

Starting execution...

[Task executes with progress updates]

✅ Task completed!

View results in Archon: https://omnik.bg502.ru/tasks/arch-001
```

#### `/result <task_id>`
**Description:** Get task result/details  
**Usage:** `/result <task_id>`  
**Examples:**
- `/result arch-004`
- `/result task-abc123`

**Response:**
```
📊 Task Result: #arch-004

Title: Fix authentication bug
Status: ✅ Completed
Duration: 12 minutes

Description:
Fixed JWT token validation in auth middleware

Changes:
- Updated auth.py with proper error handling
- Added unit tests for token validation
- Fixed expiration time calculation

Files modified:
- src/auth.py (45 lines changed)
- tests/test_auth.py (23 lines added)

Commits:
- abc123d: Fix JWT validation
- def456e: Add auth tests

View in Archon: https://omnik.bg502.ru/tasks/arch-004
```

---

## 12. User Flows

### 12.1 Basic Task Execution Flow

```
User: /task create a FastAPI hello world app

Bot: 🚀 Starting task: create a FastAPI hello world app
     Task ID: task-001
     Status: Running...

[30 seconds pass]

Bot: 📝 Creating project structure...
     - Created main.py
     - Created requirements.txt

[1 minute passes]

Bot: ✅ Task completed!
     
     Created FastAPI application in /workspace/projects/hello-fastapi/
     
     Files created:
     - main.py (FastAPI app with root endpoint)
     - requirements.txt (dependencies)
     - README.md (setup instructions)
     
     To run: cd hello-fastapi && uvicorn main:app --reload
     
     Task ID: task-001
     Duration: 1m 42s
```

### 12.2 File Navigation Flow

```
User: /pwd

Bot: 📍 Current directory: /workspace

User: /projects

Bot: 📂 Projects in /workspace/projects/
     1. hello-fastapi/ (modified 2m ago)
     2. data-pipeline/ (modified 1d ago)

User: /cd projects/hello-fastapi

Bot: ✅ Changed directory to: /workspace/projects/hello-fastapi

User: /ls

Bot: 📁 /workspace/projects/hello-fastapi/
     
     Files:
       main.py (234 B) - 2 minutes ago
       requirements.txt (45 B) - 2 minutes ago
       README.md (512 B) - 2 minutes ago
     
     Total: 0 directories, 3 files

User: /cat main.py

Bot: 📄 main.py
     
     ```python
     from fastapi import FastAPI
     
     app = FastAPI()
     
     @app.get("/")
     async def root():
         return {"message": "Hello World"}
     ```
     
     Lines: 7 | Size: 234 B
```

### 12.3 Agent Reset Flow

```
User: /status

Bot: 📊 Agent Status
     
     Agent: 🟢 Busy
     Current Task: task-042 (refactoring authentication)
     Working Directory: /workspace/projects/myapp

User: /cancel

Bot: 🛑 Cancelling task: task-042
     Task cancelled successfully.

User: /reset starting fresh with new approach

Bot: 🔄 Resetting agent context...
     
     Reason: starting fresh with new approach
     Saving current state...
     Stopping Claude Code process...
     Clearing conversation history...
     Starting Claude Code process...
     
     ✅ Agent reset complete!
     
     Context cleared.
     Working directory preserved: /workspace/projects/myapp
     Ready for new tasks.

User: /status

Bot: 📊 Agent Status
     
     Agent: 🟢 Idle
     Uptime: 5s
     Working Directory: /workspace/projects/myapp
     Current Task: None
```

### 12.4 Archon Integration Flow

```
User: /tasks

Bot: 📋 Tasks from Archon
     
     Pending:
       #arch-101: Implement user profile API
       #arch-102: Add search functionality
     
     Use /run <task_id> to execute

User: /run arch-101

Bot: 🚀 Running task from Archon: #arch-101
     
     Task: Implement user profile API
     Description: Create REST endpoints for user profile CRUD
     Priority: High
     
     Starting execution...

[Task executes]

Bot: 📝 Progress update (1/3)
     - Created user model
     - Set up database schema

[2 minutes pass]

Bot: 📝 Progress update (2/3)
     - Implemented API endpoints
     - Added input validation

[2 minutes pass]

Bot: 📝 Progress update (3/3)
     - Added unit tests
     - Updated API documentation

Bot: ✅ Task completed!
     
     Implemented user profile API with:
     - GET /users/{id} - Get user profile
     - PUT /users/{id} - Update user profile
     - DELETE /users/{id} - Delete user profile
     
     Files:
     - src/models/user.py (new)
     - src/routes/users.py (new)
     - tests/test_users.py (new)
     
     View in Archon: https://omnik.bg502.ru/tasks/arch-101
     
     Task ID: arch-101
     Duration: 4m 23s

User: Opens Archon web UI to review code and approve changes
```

---

## 13. Testing Requirements

### 13.1 Unit Tests

#### TEST-1: Bot Command Tests
```python
# Test command parsing
async def test_task_command_parsing():
    """Test /task command parses arguments correctly"""
    pass

# Test authorization
async def test_unauthorized_user_rejected():
    """Test unauthorized users are rejected"""
    pass

# Test file operations
async def test_path_validation():
    """Test path validation prevents directory traversal"""
    pass
```

#### TEST-2: Agent Manager Tests
```python
async def test_agent_start():
    """Test agent process starts successfully"""
    pass

async def test_agent_restart():
    """Test agent can be restarted"""
    pass

async def test_agent_health_check():
    """Test health check detects crashed agent"""
    pass
```

#### TEST-3: File System Tests
```python
async def test_list_directory():
    """Test directory listing"""
    pass

async def test_change_directory():
    """Test directory navigation"""
    pass

async def test_read_file():
    """Test file reading"""
    pass

async def test_invalid_path_rejected():
    """Test invalid paths are rejected"""
    pass
```

### 13.2 Integration Tests

#### TEST-4: End-to-End Task Flow
```python
async def test_task_execution_flow():
    """Test complete task from Telegram to completion"""
    # Send /task command
    # Verify agent starts processing
    # Wait for completion
    # Verify response received
    # Verify files created
    pass
```

#### TEST-5: Archon Integration
```python
async def test_archon_task_sync():
    """Test task synchronization with Archon"""
    # Create task in Archon
    # Trigger from Telegram
    # Verify status updates in both systems
    pass
```

### 13.3 Manual Testing Checklist

- [ ] Bot responds to /start command
- [ ] Unauthorized users are rejected
- [ ] Task execution completes successfully
- [ ] Progress updates are sent during long tasks
- [ ] File listing works correctly
- [ ] Directory navigation works
- [ ] File reading displays content
- [ ] Agent restart clears context
- [ ] Agent auto-restarts after crash
- [ ] Multiple users can use bot (if enabled)
- [ ] Archon tasks are listed correctly
- [ ] Archon task execution works
- [ ] Task results sync to Archon
- [ ] Error messages are user-friendly
- [ ] Rate limiting works
- [ ] Session persistence works
- [ ] Bot recovers from Telegram API errors
- [ ] Bot recovers from Archon downtime

---

## 14. Deployment and Operations

### 14.1 Initial Setup

#### DEPLOY-1: Server Preparation
```bash
# 1. Clone repository
git clone <repository-url>
cd omnik-telegram-bot

# 2. Create .env file
cp .env.example .env
nano .env  # Edit with your credentials

# 3. Configure authorized users
cp config/authorized_users.yaml.example config/authorized_users.yaml
nano config/authorized_users.yaml  # Add your Telegram user IDs

# 4. Build and start services
docker-compose build
docker-compose up -d

# 5. Check logs
docker-compose logs -f telegram-bot
docker-compose logs -f archon-mcp

# 6. Test bot
# Send /start to your bot on Telegram
```

#### DEPLOY-2: Telegram Bot Setup
```bash
# 1. Create bot with BotFather
# - Open Telegram and search for @BotFather
# - Send /newbot
# - Follow prompts to create bot
# - Save token to .env as TELEGRAM_BOT_TOKEN

# 2. Get your user ID
# - Send /start to @userinfobot
# - Save your ID to .env as TELEGRAM_ADMIN_USER_ID

# 3. Configure bot settings with BotFather
# /setcommands - Set command list
# /setdescription - Set bot description
# /setabouttext - Set about text
```

### 14.2 Monitoring

#### DEPLOY-3: Health Monitoring
```bash
# Check service status
docker-compose ps

# View logs
docker-compose logs -f telegram-bot
docker-compose logs -f archon-mcp

# Check agent process
docker-compose exec telegram-bot supervisorctl status

# Check disk space
docker-compose exec telegram-bot df -h /workspace

# Check memory usage
docker stats omnik-telegram-bot omnik-archon
```

#### DEPLOY-4: Log Management
```yaml
# docker-compose.yml logging configuration
services:
  telegram-bot:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

### 14.3 Backup and Recovery

#### DEPLOY-5: Backup Strategy
```bash
# Backup workspace (projects)
docker run --rm \
  -v omnik_workdir:/workspace \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/workspace-$(date +%Y%m%d).tar.gz /workspace

# Backup state (sessions, tasks)
docker run --rm \
  -v omnik_state:/state \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/state-$(date +%Y%m%d).tar.gz /state

# Automated daily backup (add to crontab)
0 2 * * * cd /path/to/omnik && ./backup.sh
```

#### DEPLOY-6: Recovery
```bash
# Restore workspace
docker run --rm \
  -v omnik_workdir:/workspace \
  -v $(pwd)/backups:/backup \
  alpine tar xzf /backup/workspace-20251024.tar.gz -C /

# Restore state
docker run --rm \
  -v omnik_state:/state \
  -v $(pwd)/backups:/backup \
  alpine tar xzf /backup/state-20251024.tar.gz -C /
```

### 14.4 Troubleshooting

#### Common Issues

**Issue: Bot not responding**
```bash
# Check if container is running
docker ps | grep telegram-bot

# Check logs for errors
docker-compose logs --tail=100 telegram-bot

# Verify Telegram token
docker-compose exec telegram-bot env | grep TELEGRAM_TOKEN

# Restart bot
docker-compose restart telegram-bot
```

**Issue: Agent process crashed**
```bash
# Check supervisor status
docker-compose exec telegram-bot supervisorctl status

# View Claude Code logs
docker-compose exec telegram-bot tail -f /var/log/supervisor/claude.err.log

# Manually restart agent
docker-compose exec telegram-bot supervisorctl restart claude_code
```

**Issue: Can't access Archon**
```bash
# Check Archon container
docker ps | grep archon

# Test connectivity from bot container
docker-compose exec telegram-bot curl http://archon-mcp:8080/health

# Check network
docker network inspect omnik_omnik-network
```

**Issue: File operation permissions**
```bash
# Check file ownership
docker-compose exec telegram-bot ls -la /workspace

# Fix permissions
docker-compose exec telegram-bot chown -R agent:agent /workspace
```

---

## 15. Performance Requirements

### 15.1 Response Times

- Command response (non-agent): < 1 second
- Simple file operations: < 2 seconds
- Task initiation: < 3 seconds
- Progress update frequency: Every 30 seconds for long tasks
- Agent restart: < 10 seconds

### 15.2 Resource Limits

```yaml
# docker-compose.yml resource limits
services:
  telegram-bot:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '0.5'
          memory: 512M
```

### 15.3 Concurrency

- Single agent instance per container (no parallel tasks)
- Support multiple users (queued tasks)
- Maximum 10 concurrent Telegram requests
- Maximum 5 concurrent Archon API requests

---

## 16. Success Metrics

### 16.1 Functional Metrics
- [ ] Bot responds to all commands within SLA
- [ ] Task execution success rate > 95%
- [ ] File operations success rate > 99%
- [ ] Agent restart success rate > 99%
- [ ] Archon integration works reliably
- [ ] Zero unauthorized access

### 16.2 Performance Metrics
- [ ] Average command response time < 2 seconds
- [ ] Average task completion time acceptable (varies by task)
- [ ] Agent uptime > 99%
- [ ] Bot uptime > 99.5%
- [ ] Memory usage stable (no leaks)

### 16.3 User Experience Metrics
- [ ] Clear, helpful error messages
- [ ] Progress updates for long tasks
- [ ] Intuitive command interface
- [ ] File operations work as expected
- [ ] Context reset works reliably

---

## 17. Development Phases

### Phase 1: Core Bot (Week 1-2)
- [ ] Setup project structure
- [ ] Implement basic Telegram bot
- [ ] Add command handlers (/start, /help, /status)
- [ ] Implement user authorization
- [ ] Setup Docker container with supervisord
- [ ] Integrate Claude Code Agent SDK (subprocess)
- [ ] Basic task execution (/task)
- [ ] Implement agent process manager
- [ ] Add logging and error handling

### Phase 2: File System (Week 2-3)
- [ ] Implement file system commands (/ls, /cd, /pwd)
- [ ] Add file reading (/cat)
- [ ] Add directory tree (/tree)
- [ ] Implement path validation
- [ ] Add session persistence
- [ ] Improve error handling for file operations

### Phase 3: Agent Management (Week 3-4)
- [ ] Implement agent restart (/reset)
- [ ] Add health monitoring
- [ ] Implement auto-restart on crash
- [ ] Add circuit breaker
- [ ] Improve progress updates
- [ ] Add task cancellation (/cancel)
- [ ] Implement timeout handling

### Phase 4: Archon Integration (Week 4-5)
- [ ] Setup Archon MCP in docker-compose
- [ ] Configure shared volumes and network
- [ ] Implement Archon API client
- [ ] Add task listing (/tasks)
- [ ] Add task execution (/run)
- [ ] Add task result viewing (/result)
- [ ] Implement bidirectional sync
- [ ] Test integration flows

### Phase 5: Testing & Polish (Week 5-6)
- [ ] Write unit tests
- [ ] Write integration tests
- [ ] Manual testing
- [ ] Performance testing
- [ ] Security audit
- [ ] Documentation
- [ ] Deployment guide
- [ ] Production deployment

---

## 18. Future Enhancements (v2+)

- Voice message support (speech-to-text)
- Multi-agent support (multiple Claude Code instances)
- Advanced scheduling (cron-like task scheduling)
- File upload support (send files to agent)
- Image generation integration
- Webhook support for external triggers
- Web dashboard for bot management
- Metrics and analytics
- Multi-language support
- Custom command aliases
- Team collaboration features
- Git integration (commit, push via commands)
- Code review workflows
- Notification preferences
- Task templates
- Workflow automation
- Integration with other MCP servers
- Database query interface
- API testing interface

---

## 19. Dependencies and Risks

### 19.1 Dependencies
- Telegram Bot API availability
- Anthropic API availability
- Claude Code Agent SDK stability
- Archon MCP compatibility
- Docker infrastructure
- Network connectivity

### 19.2 Risks and Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Telegram API rate limits | High | Implement request queuing and caching |
| Claude Code crashes | High | Auto-restart with circuit breaker |
| Archon incompatibility | Medium | Version pinning, API contracts |
| Unauthorized access | High | Strong authentication, audit logs |
| Resource exhaustion | Medium | Resource limits, monitoring |
| Data loss | Medium | Regular backups, state persistence |
| Network issues | Low | Retry logic, graceful degradation |

### 19.3 Assumptions
- Single user or small team usage
- Trusted environment (VM in cloud)
- Stable network connectivity
- Anthropic API has sufficient quota
- Telegram Bot API is stable

---

## 20. Appendix

### 20.1 Glossary
- **Telegram Bot:** Python application handling Telegram messages
- **Agent:** Claude Code process managed by supervisord
- **Archon MCP:** Separate service providing web UI
- **Session:** User conversation state and context
- **Task:** Unit of work executed by the agent
- **Workspace:** Shared directory for projects (/workspace)
- **State:** Persistent data (sessions, tasks, config)

### 20.2 References
- Python Telegram Bot: https://python-telegram-bot.org/
- Claude Code Agent SDK: https://docs.claude.com/
- Supervisord: http://supervisord.org/
- Archon MCP: [documentation link]
- Docker Compose: https://docs.docker.com/compose/

### 20.3 Configuration Examples

**Bot Configuration (config/bot.yaml)**
```yaml
bot:
  name: "Omnik Telegram Bot"
  version: "1.0.0"
  
agent:
  workspace_dir: "/workspace"
  state_dir: "/state"
  restart_max_attempts: 5
  restart_backoff_seconds: 60
  task_timeout_seconds: 600
  
session:
  timeout_minutes: 30
  max_context_messages: 50
  
rate_limiting:
  messages_per_minute: 30
  tasks_per_hour: 5
  resets_per_hour: 3
  
archon:
  enabled: true
  url: "http://archon-mcp:8080"
  api_key: "${ARCHON_API_KEY}"
  sync_interval_seconds: 60
```

**Authorized Users (config/authorized_users.yaml)**
```yaml
authorized_users:
  - user_id: 123456789
    username: "admin_user"
    role: "admin"
    allowed_commands: ["all"]
    rate_limits:
      messages_per_minute: 60
      tasks_per_hour: 20
  
  - user_id: 987654321
    username: "developer"
    role: "user"
    allowed_commands:
      - "task"
      - "status"
      - "cancel"
      - "ls"
      - "cd"
      - "pwd"
      - "cat"
      - "tasks"
      - "run"
    rate_limits:
      messages_per_minute: 30
      tasks_per_hour: 5
```

### 20.4 Document History
| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2025-10-24 | Initial | Initial PRD draft |

---

## Document Approval

**Product Owner:** _________________  
**Technical Lead:** _________________  
**Date:** _________________
