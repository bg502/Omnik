# omnik

**Telegram-based conversational interface for Claude Code**

omnik enables developers to manage AI-powered coding sessions from anywhere using Telegram. The application runs as a containerized Python service that spawns and controls Claude Code subprocess sessions, providing a mobile-first development assistant with persistent conversation state, workspace isolation, and Docker integration.

## Features

- 🤖 **Claude Code Integration**: Full Claude Code session management through Telegram
- 📱 **Mobile-First**: Code from anywhere using just your phone
- 🔒 **Secure**: Whitelist authentication, isolated workspaces, containerized execution
- 💾 **Persistent**: SQLite-backed conversation history and session state
- 🚀 **Real-Time**: Live streaming of Claude Code output to Telegram
- 🔄 **Multi-Session**: Support for multiple concurrent workspaces

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Telegram Bot Token (get from [@BotFather](https://t.me/botfather))
- Anthropic API Key (get from [Anthropic Console](https://console.anthropic.com/))

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd omnik
```

2. Configure secrets:
```bash
# Create secrets directory
mkdir -p secrets

# Add your Telegram bot token
echo "TELEGRAM_BOT_TOKEN=your_token_here" > secrets/telegram_token.txt
```

3. Set environment variables:
```bash
# Copy example environment file
cp .env.example .env

# Edit .env and set:
# - AUTHORIZED_USER_ID (your Telegram user ID)
# - ANTHROPIC_API_KEY (your Anthropic API key)
vim .env
```

4. Build and start the service:
```bash
docker-compose up -d omnik
```

5. Check logs:
```bash
docker-compose logs -f omnik
```

## Usage

### Getting Your Telegram User ID

1. Message [@userinfobot](https://t.me/userinfobot) on Telegram
2. It will reply with your user ID
3. Set this in your `.env` file as `AUTHORIZED_USER_ID`

### Bot Commands

**Session Management:**
- `/start` - Initialize bot and see welcome message
- `/new [name]` - Create new Claude Code session
- `/list` - Show all your sessions
- `/switch <id>` - Switch to a different session
- `/kill` - Terminate active session
- `/restart` - Restart Claude Code process

**Session Info:**
- `/status` - Show current session details
- `/pwd` - Show current working directory
- `/ls [path]` - List files in workspace

**Interaction:**
- Send any message to chat with Claude Code
- Upload files to add them to your workspace

### Example Workflow

```
User: /new my-project
Bot:  ✅ Created session abc12345
      Name: my-project
      Send a message to start coding!

User: Create a Python hello world script
Bot:  🤔 Processing...
      [Claude Code responds with code and creates file]

User: /ls
Bot:  📁 ./
      📄 hello.py (85 bytes)

User: Run the script
Bot:  [Claude Code executes and shows output]
```

## Architecture

```
┌─────────────────────────────────┐
│     Telegram Platform           │
└────────────┬────────────────────┘
             │ HTTPS
             ▼
     ┌───────────────────┐
     │   omnik Container │
     │                   │
     │  ┌─────────────┐  │
     │  │  Bot Mgr    │  │
     │  └──────┬──────┘  │
     │         │         │
     │  ┌──────▼──────┐  │
     │  │ Session Mgr │  │
     │  │  Claude Code│  │
     │  └─────────────┘  │
     │         │         │
     │  ┌──────▼──────┐  │
     │  │  SQLite DB  │  │
     │  └─────────────┘  │
     └─────────┬─────────┘
               │
               ▼
     ┌─────────────────┐
     │ Workspace Volume│
     └─────────────────┘
```

## Configuration

Environment variables can be set in `.env`:

| Variable | Description | Default |
|----------|-------------|---------|
| `AUTHORIZED_USER_ID` | Your Telegram user ID | None (required) |
| `ANTHROPIC_API_KEY` | Anthropic API key | None (required) |
| `LOG_LEVEL` | Logging level | INFO |
| `MAX_SESSIONS` | Max concurrent sessions | 10 |
| `SESSION_TIMEOUT_HOURS` | Session timeout | 24 |
| `WORKSPACE_BASE` | Workspace directory | /workspace |
| `RATE_LIMIT_REQUESTS` | Rate limit per minute | 60 |

## Development

### Project Structure

```
omnik/
├── src/
│   ├── bot/
│   │   ├── handlers.py        # Telegram command handlers
│   │   └── session_manager.py # Claude Code subprocess management
│   ├── database/
│   │   ├── manager.py         # Database CRUD operations
│   │   └── schema.py          # SQLAlchemy models
│   ├── models/
│   │   ├── session.py         # Session Pydantic model
│   │   ├── message.py         # Message Pydantic model
│   │   └── audit.py           # Audit log Pydantic model
│   ├── utils/
│   │   ├── config.py          # Configuration management
│   │   └── logging.py         # Logging setup
│   └── main.py                # Application entry point
├── Dockerfile                 # Container definition
├── docker-compose.yml         # Service orchestration
└── requirements.txt           # Python dependencies
```

### Running Locally

```bash
# Install dependencies
pip install -r requirements.txt

# Set environment variables
export TELEGRAM_BOT_TOKEN="your_token"
export ANTHROPIC_API_KEY="your_key"
export AUTHORIZED_USER_ID="your_id"

# Run the bot
python -m src.main
```

## Security

- **Authentication**: Whitelist-based, only configured Telegram user ID can interact
- **Isolation**: Each session runs in isolated workspace directory
- **Container Security**: Non-root user, read-only filesystem, dropped capabilities
- **Audit Logging**: All commands and file operations logged
- **No Docker Socket**: Claude Code runs as subprocess, not separate container

## Troubleshooting

### Bot not responding

1. Check if container is running: `docker-compose ps`
2. Check logs: `docker-compose logs -f omnik`
3. Verify Telegram token is correct
4. Ensure your user ID is authorized

### Claude Code not starting

1. Check Anthropic API key is set correctly
2. Verify Claude Code is installed in container
3. Check workspace permissions
4. Review session logs in database

### Database errors

1. Ensure `/data` directory is writable
2. Check SQLite database file permissions
3. Try deleting database and restarting (will lose history)

## Roadmap

See [PRD.md](PRD.md) for detailed product requirements and roadmap.

**Milestone 1** (Current): Core infrastructure, basic bot commands
**Milestone 2**: Claude Code integration, session management
**Milestone 3**: Session persistence, message history
**Milestone 4**: File operations, multiple sessions
**Milestone 5**: Production hardening, security, documentation

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[License information here]

## Acknowledgments

Built with:
- [python-telegram-bot](https://github.com/python-telegram-bot/python-telegram-bot)
- [Claude Code](https://github.com/anthropics/claude-code)
- [Pydantic](https://github.com/pydantic/pydantic)
- [SQLAlchemy](https://www.sqlalchemy.org/)
