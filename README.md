# Omnik

**Telegram Bot for Claude Code - Mobile AI Coding Assistant**

Omnik enables you to interact with Claude Code AI assistant directly from Telegram, allowing you to code, debug, and manage projects from anywhere using your phone or any Telegram client.

## Features

- 🤖 **Full Claude Code Integration** - Access Claude's AI coding capabilities via Telegram
- 📱 **Mobile-First** - Code from your phone, tablet, or any device with Telegram
- 💬 **Multi-Turn Conversations** - Maintain context across messages with full conversation history
- 🗂️ **Session Management** - Create, switch between, and manage multiple independent coding sessions
- 📂 **Workspace Persistence** - Each session remembers its working directory
- 🔧 **Direct File Navigation** - Browse, read, and execute commands directly in the workspace
- 🔒 **Secure** - Whitelist authentication, containerized execution
- ⚡ **Real-Time Streaming** - Watch Claude's responses stream in real-time

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Telegram Bot Token (get from [@BotFather](https://t.me/botfather))
- Anthropic API Key (get from [Anthropic Console](https://console.anthropic.com/))
- Your Telegram User ID (get from [@userinfobot](https://t.me/userinfobot))

### Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yourusername/omnik.git
   cd omnik
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env and set:
   # - TELEGRAM_BOT_TOKEN
   # - AUTHORIZED_USER_ID
   # - ANTHROPIC_API_KEY
   ```

3. **Authenticate Claude CLI** (one-time setup):
   ```bash
   docker compose build omnik-unified
   docker compose run --rm omnik-unified claude setup-token
   # Follow the prompts to authenticate
   ```

4. **Start the bot:**
   ```bash
   docker compose up -d
   ```

5. **Verify it's running:**
   ```bash
   docker compose logs -f omnik-unified
   ```

## Usage

### Bot Commands

**Session Management:**
- `/sessions` - List all sessions
- `/newsession <name> [description]` - Create a new session
- `/switch <name>` - Switch to a different session
- `/delsession <name>` - Delete a session
- `/status` - Show current session details

**File Navigation:**
- `/pwd` - Show current working directory
- `/ls` - List files in current directory
- `/cd <path>` - Change directory (saved per session!)
- `/cat <file>` - View file contents
- `/exec <command>` - Execute bash command

**Help:**
- `/start` - Show welcome message and commands

### Example Workflow

```
You: /newsession myproject Building a web scraper
Bot: Created and switched to session: myproject

You: Hi! Can you help me create a Python web scraper?
Claude: Hi! Yes, I can help you create a Python web scraper...

You: Let's use BeautifulSoup to scrape news headlines
Claude: [Creates scraper.py with BeautifulSoup code]

You: /ls
Bot: total 8K
     -rw-r--r-- 1 node node 1.2K scraper.py

You: /newsession backend Working on API
Bot: Created and switched to session: backend

You: Create a FastAPI hello world
Claude: [Creates FastAPI app]

You: /switch myproject
Bot: Switched to session: myproject
     Working directory: /workspace

# Your scraper code is still there!
You: Can you add error handling to the scraper?
Claude: [Continues the conversation from before]
```

## Architecture

Omnik runs as a unified Docker container combining:
- **Go Telegram Bot** - Handles Telegram API, commands, and session management
- **Claude CLI** - Official Claude Code CLI for AI interactions
- **Shared Filesystem** - `/workspace` volume for persistent file storage

```
┌─────────────────────────────────┐
│     Telegram Platform           │
└────────────┬────────────────────┘
             │
             ▼
     ┌───────────────────┐
     │ omnik-unified     │
     │                   │
     │  ┌─────────────┐  │
     │  │   Go Bot    │  │
     │  │  (Telegram) │  │
     │  └──────┬──────┘  │
     │         │         │
     │  ┌──────▼──────┐  │
     │  │  Session    │  │
     │  │  Manager    │  │
     │  └──────┬──────┘  │
     │         │         │
     │  ┌──────▼──────┐  │
     │  │ Claude CLI  │  │
     │  │    (AI)     │  │
     │  └──────┬──────┘  │
     │         │         │
     │    /workspace     │
     │                   │
     └───────────────────┘
```

### Session Persistence

- Sessions are stored in `/workspace/.omnik-sessions.json`
- Each session maintains:
  - Name and description
  - Claude conversation ID
  - Current working directory
  - Creation and last-used timestamps
- Working directory persists when you switch sessions

## Configuration

Environment variables (`.env`):

| Variable | Description | Default |
|----------|-------------|---------|
| `TELEGRAM_BOT_TOKEN` | Telegram bot API token | Required |
| `AUTHORIZED_USER_ID` | Your Telegram user ID | Required |
| `ANTHROPIC_API_KEY` | Anthropic API key | Required |
| `CLAUDE_MODEL` | Claude model to use | `sonnet` |
| `LOG_LEVEL` | Logging verbosity | `INFO` |

## Development

### Project Structure

```
omnik/
├── go-bot/                    # Main application
│   ├── cmd/
│   │   └── main.go           # Application entry point
│   └── internal/
│       ├── bot/
│       │   └── bot.go        # Telegram bot logic
│       ├── claude/
│       │   ├── cli.go        # Claude CLI client
│       │   └── client.go     # HTTP client (legacy)
│       └── session/
│           └── manager.go    # Session management
├── Dockerfile.unified         # Production Dockerfile
├── docker-compose.yml         # Service definition
├── .env.example              # Environment template
└── README.md                 # This file
```

### Building Locally

```bash
# Build the container
docker compose build omnik-unified

# Run in development mode
docker compose up omnik-unified

# View logs
docker compose logs -f omnik-unified
```

## Security

- **Whitelist Authentication** - Only configured Telegram user ID can interact
- **Containerized Execution** - All code runs in isolated Docker container
- **No Sudo/Root** - Bot runs as non-privileged `node` user
- **Workspace Isolation** - Each session can have its own workspace directory
- **Permission Control** - Claude runs with `bypassPermissions` mode for autonomous operation within the secure sandbox

## Troubleshooting

### Bot not responding

1. Check if container is running:
   ```bash
   docker compose ps
   ```

2. Check logs:
   ```bash
   docker compose logs -f omnik-unified
   ```

3. Verify environment variables in `.env`

### Claude authentication issues

Re-authenticate Claude:
```bash
docker compose run --rm omnik-unified claude setup-token
```

### Session not persisting

- Ensure `/workspace` volume exists and is writable
- Check logs for session manager errors
- Sessions are stored in `/workspace/.omnik-sessions.json`

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

[MIT License](LICENSE)

## Acknowledgments

- [Claude Code](https://github.com/anthropics/claude-code) - AI coding assistant
- [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) - Telegram Bot API for Go
- Built with ❤️ for developers who code on the go
