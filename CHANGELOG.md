# Changelog

All notable changes to Omnik will be documented in this file.

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
