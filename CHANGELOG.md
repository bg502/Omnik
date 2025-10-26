# Changelog

All notable changes to Omnik will be documented in this file.

## [1.0.003] - 2025-10-26

### 🚀 New Features

**MCP Server Integration:**
- Added support for Model Context Protocol (MCP) servers
- New `/mcp` command to list configured MCP servers
- New `/reload` command to reload MCP configuration
- MCP configuration persists globally across all sessions
- MCP servers remain available after container restarts

**Enhanced Keyboard Interface:**
- Added "🔧 MCP" button to main keyboard for quick MCP server listing
- Added "🔄 Reload" button to reload MCP configuration
- Reload includes confirmation dialog (Yes/No) to prevent accidental reloads
- Updated keyboard layout:
  ```
  [📂 Sessions] [📊 Status]
  [📁 pwd] [📋 ls]
  [🔧 MCP] [🔄 Reload]
  [ℹ️ Help]
  ```

**Improved Tool Display:**
- Increased Bash command display limit from 40 to 150 characters
- Better visibility of command parameters in tool usage messages
- More context when viewing Claude's tool execution

**Network Configuration:**
- Connected omnik container to external `kb_network`
- Maintains both internal (`omnik-net`) and external network connectivity
- Enables communication with other services on shared network

### 🔧 Technical Changes

**Volume Mount Updates:**
- Changed volume mount from `claude-auth:/home/node/.claude` to `claude-home:/home/node`
- Ensures MCP configuration file (`.claude.json`) persists across restarts
- MCP servers now available globally to all sessions

**Session Management:**
- Reload command deletes and recreates session with same name
- Session name preservation during reload (no "-reloaded" suffix)
- Clears conversation history while maintaining session identity
- Triggers MCP server reload for new sessions

**Updated Files:**
- `docker-compose.yml` - Volume mount and network configuration
- `go-bot/internal/bot/bot.go` - MCP commands, reload logic, keyboard layout
- Volume renamed: `claude-auth` → `claude-home`

### Why These Changes?

**MCP Integration:**
MCP (Model Context Protocol) allows Claude to connect to external services and tools, extending its capabilities beyond the built-in toolset. This enables integration with custom APIs, databases, and specialized services.

**Persistence:**
Previous implementation stored MCP configuration in a non-persistent location, requiring reconfiguration after each container restart. The new volume mount ensures MCP servers remain configured across restarts and are available in all sessions.

**Reload Functionality:**
MCP servers are loaded when Claude CLI starts a new session. The reload command provides a way to pick up newly added MCP servers without restarting the entire container, maintaining session name and workspace context.

### Migration Instructions

**After updating code:**
```bash
# Rebuild container with new volume configuration
docker compose down
docker compose build omnik
docker compose up -d omnik

# MCP servers will need to be re-added after first rebuild
# Example: Add archon MCP server
docker compose exec omnik claude mcp add --transport http archon http://archon-mcp:8051/mcp

# Verify MCP server is connected
docker compose exec omnik claude mcp list
```

**Using MCP Features:**
- Use `/mcp` command in Telegram to list configured MCP servers
- Use `/reload` command to reload MCP configuration after adding new servers
- MCP servers are available in all sessions (global configuration)
- Test MCP tools by asking Claude to use them in conversation

---

## [1.0.002] - 2025-10-25

### 🔧 Breaking Changes

**Environment Variable Renaming:**
- All environment variables now use `OMNI_` prefix to prevent conflicts with other projects
- **BREAKING:** You must update your `.env` file with new variable names

**Variable Mapping:**
- `TELEGRAM_BOT_TOKEN` → `OMNI_TELEGRAM_BOT_TOKEN`
- `AUTHORIZED_USER_ID` → `OMNI_AUTHORIZED_USER_ID`
- `ANTHROPIC_API_KEY` → `OMNI_ANTHROPIC_API_KEY`
- `CLAUDE_MODEL` → `OMNI_CLAUDE_MODEL`
- `LOG_LEVEL` → `OMNI_LOG_LEVEL`
- `USE_CLAUDE_SDK` → `OMNI_USE_CLAUDE_SDK`
- `CLAUDE_BRIDGE_URL` → `OMNI_CLAUDE_BRIDGE_URL` (legacy)

**GitHub variables kept without prefix** (standard names):
- `GITHUB_TOKEN` (unchanged)
- `GIT_USER_NAME` (unchanged)
- `GIT_USER_EMAIL` (unchanged)

**Updated Files:**
- `.env.example` - Updated all variable names
- `docker-compose.yml` - Updated environment section
- `go-bot/internal/bot/bot.go` - Updated env var reading
- `entrypoint.sh` - Updated git configuration
- `git-credential-helper.sh` - Updated GitHub token reference
- Documentation (README.md, ARCHITECTURE.md)

### Why This Change?

When working with other projects (e.g., Eventswipe), environment variables with common names like `TELEGRAM_BOT_TOKEN` would be substituted when running `docker compose` commands, causing conflicts and unexpected behavior. The `OMNI_` prefix ensures Omnik's configuration remains isolated.

### Migration Instructions

**After updating code, update your `.env` file:**
```bash
# Rename Omnik-specific variables to use OMNI_ prefix
sed -i 's/^TELEGRAM_BOT_TOKEN=/OMNI_TELEGRAM_BOT_TOKEN=/' .env
sed -i 's/^AUTHORIZED_USER_ID=/OMNI_AUTHORIZED_USER_ID=/' .env
sed -i 's/^ANTHROPIC_API_KEY=/OMNI_ANTHROPIC_API_KEY=/' .env
sed -i 's/^CLAUDE_MODEL=/OMNI_CLAUDE_MODEL=/' .env
sed -i 's/^LOG_LEVEL=/OMNI_LOG_LEVEL=/' .env

# GitHub variables (GITHUB_TOKEN, GIT_USER_NAME, GIT_USER_EMAIL) remain unchanged

# Rebuild and restart
docker compose build omnik
docker compose up -d omnik
```

---

## [1.0.001] - 2025-10-25

### 🔧 Enhancements

**Container Improvements:**
- Renamed `Dockerfile.unified` → `Dockerfile` (simplified naming)
- Service name: `omnik-unified` → `omnik` across all files
- Added Docker CLI and Docker Compose v2 to container
- Added `node` user to docker group for socket access
- Mounted `/var/run/docker.sock` for Docker-in-Docker capabilities

**GitHub Integration:**
- Added GitHub authentication via fine-grained Personal Access Tokens
- Automatic git credential management using environment variables
- Custom git credential helper script (`/app/git-credential-helper.sh`)
- Runtime git configuration via entrypoint script
- Support for `GITHUB_TOKEN`, `GIT_USER_NAME`, `GIT_USER_EMAIL` env vars
- Secure token handling (never stored in files, only in environment)

**Security Updates:**
- Removed `Bash(docker rm:*)` from auto-approved commands list
- Maintained bypass permissions for safe development tools
- Token-based GitHub authentication (no SSH keys required)
- Git already included and updated in container

**Documentation:**
- Updated all references from `omnik-unified` to `omnik`
- Updated architecture diagrams with Docker capabilities
- Added comprehensive GitHub authentication setup guide
- Simplified build and deployment commands
- Added configuration table with new GitHub variables

### Technical Details

This release enables Claude to:
- Work with Docker containers directly (build, run, compose)
- Clone, pull, and push to GitHub repositories
- Create commits and potentially pull requests
- All within a secure containerized environment

Authentication is handled via GitHub fine-grained Personal Access Tokens, allowing repository-specific access with granular permissions. The custom credential helper reads from environment variables, ensuring tokens are never persisted to disk.

---

## [1.0.0] - 2025-10-25

### 🎉 Initial Release

**Omnik v1.0** - Telegram Bot for Claude Code

### Features

- **Full Claude Code Integration**
  - Direct Claude CLI execution for AI interactions
  - Real-time streaming responses
  - Multi-turn conversations with full context

- **Session Management**
  - Create, list, switch, and delete sessions
  - Each session maintains independent conversation history
  - Automatic session ID management
  - Session persistence across container restarts

- **Workspace Management**
  - Per-session working directory persistence
  - Direct file navigation commands (`/pwd`, `/ls`, `/cd`, `/cat`, `/exec`)
  - Shared `/workspace` volume for file storage

- **Telegram Bot**
  - Whitelist authentication (single authorized user)
  - Comprehensive command set
  - Real-time message streaming
  - Error handling and user feedback

### Architecture

- **Unified Container**: Single Docker container with Go bot + Node.js + Claude CLI
- **Go Backend**: High-performance Telegram bot implementation
- **Claude CLI Integration**: Direct execution of official Claude Code CLI
- **Session Persistence**: JSON-based session storage in workspace

### Technical Stack

- Go 1.21 (Telegram bot, session management)
- Node.js 20 (Claude CLI runtime)
- Claude Code CLI 2.0.27
- Docker multi-stage build

### Security

- Whitelist-based authentication
- Non-root container execution (node user)
- Containerized sandbox for all operations
- No external dependencies or services

### Commands

**Session Management:**
- `/sessions` - List all sessions
- `/newsession <name> [description]` - Create new session
- `/switch <name>` - Switch sessions
- `/delsession <name>` - Delete session
- `/status` - Current session status

**File Navigation:**
- `/pwd` - Print working directory
- `/ls` - List files
- `/cd <path>` - Change directory
- `/cat <file>` - View file
- `/exec <cmd>` - Execute command

### Known Limitations

- Single authorized user per bot instance
- Session IDs must be unique UUIDs (assigned by Claude CLI)
- Working directory must exist before `cd` command
- Maximum 2GB memory allocation per container

### Installation

See [README.md](README.md) for installation instructions.

---

## Future Releases

### Planned for v1.1
- Multiple user support
- Enhanced error messages
- File upload support via Telegram
- Session export/import

### Planned for v2.0
- Web UI for session management
- Multi-model support (Opus, Haiku)
- Advanced workspace features
- Collaborative sessions
