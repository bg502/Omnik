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
- 🐙 **Git & Docker Integration** - Clone repos, run containers, full DevOps capabilities
- 🔒 **Secure** - Whitelist authentication, containerized execution, token-based GitHub auth
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
   # - OMNI_TELEGRAM_BOT_TOKEN
   # - OMNI_AUTHORIZED_USER_ID
   # - OMNI_ANTHROPIC_API_KEY
   # - GITHUB_TOKEN (optional, for git operations)
   # - GIT_USER_NAME (optional, for git commits)
   # - GIT_USER_EMAIL (optional, for git commits)
   ```

   **Optional: GitHub Authentication Setup**

   If you want the bot to work with GitHub repositories:

   a. Create a fine-grained Personal Access Token:
      - Go to https://github.com/settings/tokens?type=beta
      - Click "Generate new token"
      - Set token name (e.g., "Omnik Bot")
      - Set expiration (recommend 90 days or 1 year)
      - Select repository access (specific repos or all repos)
      - Grant permissions:
        - **Contents**: Read and Write (for clone, pull, push)
        - **Metadata**: Read (required, auto-selected)
        - **Pull requests**: Read and Write (optional, for PR creation)
      - Generate and copy the token

   b. Add to `.env`:
      ```
      GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxxx
      GIT_USER_NAME=Your Name
      GIT_USER_EMAIL=your.email@example.com
      ```

3. **Authenticate Claude CLI** (one-time setup):
   ```bash
   docker compose build omnik
   docker compose run --rm omnik claude setup-token
   # Follow the prompts to authenticate
   ```

4. **Start the bot:**
   ```bash
   docker compose up -d
   ```

5. **Verify it's running:**
   ```bash
   docker compose logs -f omnik
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
     │      omnik        │
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
| `OMNI_TELEGRAM_BOT_TOKEN` | Telegram bot API token | Required |
| `OMNI_AUTHORIZED_USER_ID` | Your Telegram user ID | Required |
| `OMNI_ANTHROPIC_API_KEY` | Anthropic API key | Required |
| `OMNI_CLAUDE_MODEL` | Claude model to use | `sonnet` |
| `GITHUB_TOKEN` | GitHub fine-grained PAT | Optional |
| `GIT_USER_NAME` | Git commit author name | Optional |
| `GIT_USER_EMAIL` | Git commit author email | Optional |
| `OMNI_LOG_LEVEL` | Logging verbosity | `INFO` |

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
│       │   └── cli.go        # Claude CLI client
│       └── session/
│           └── manager.go    # Session management
├── Dockerfile                 # Production Dockerfile
├── docker-compose.yml         # Service definition
├── .env.example              # Environment template
└── README.md                 # This file
```

### Building Locally

```bash
# Build the container
docker compose build omnik

# Run in development mode
docker compose up omnik

# View logs
docker compose logs -f omnik
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
   docker compose logs -f omnik
   ```

3. Verify environment variables in `.env`

### Claude authentication issues

Re-authenticate Claude:
```bash
docker compose run --rm omnik claude setup-token
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
