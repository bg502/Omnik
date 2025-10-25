# omnik Deployment Guide

## ✅ Implementation Status

The omnik Telegram bot for Claude Code has been successfully implemented and deployed!

### What's Been Built

1. **Complete Python Application**
   - Telegram bot with all command handlers (`/start`, `/new`, `/list`, `/switch`, `/status`, `/kill`, `/restart`, `/pwd`, `/ls`)
   - Session manager for Claude Code subprocess control
   - SQLite database with full schema (sessions, messages, audit logs)
   - Pydantic models for type safety
   - Structured logging with structlog
   - Configuration management with environment variables and Docker secrets

2. **Docker Infrastructure**
   - Dockerfile with Python 3.11, Node.js 20, and Claude Code CLI
   - Docker Compose configuration with:
     - Named volumes for workspace and database persistence
     - Network isolation
     - Resource limits (2 CPU, 4GB RAM)
     - Health checks
     - Proper user permissions (UID 1000)

3. **Security Features**
   - Non-root container user
   - Whitelist-based authentication
   - Docker secrets for sensitive data
   - Audit logging for all operations
   - Workspace isolation per session

## 🚀 Current Status

```bash
$ docker compose ps
NAME      IMAGE         COMMAND                  SERVICE   CREATED        STATUS
omnik     omnik-omnik   "tini -g -- python -…"   omnik     Running        Up (healthy)
```

The bot is **running successfully** and ready to accept connections!

## 📋 Next Steps to Use the Bot

### 1. Get Your Telegram User ID

Message [@userinfobot](https://t.me/userinfobot) on Telegram to get your user ID.

### 2. Configure Environment Variables

Edit `.env` file:

```bash
# Required Configuration
AUTHORIZED_USER_ID=123456789  # Your Telegram user ID from step 1
ANTHROPIC_API_KEY=sk-ant-...  # Your Anthropic API key

# Optional (defaults shown)
LOG_LEVEL=INFO
MAX_SESSIONS=10
SESSION_TIMEOUT_HOURS=24
```

### 3. Restart the Container

```bash
docker compose down
docker compose up -d omnik
```

### 4. Start Using the Bot

1. Open Telegram and search for your bot (the one you created with @BotFather)
2. Send `/start` to begin
3. Send `/new my-project` to create a new coding session
4. Start chatting with Claude Code!

## 📱 Available Commands

| Command | Description |
|---------|-------------|
| `/start` | Welcome message and command list |
| `/new [name]` | Create new Claude Code session |
| `/list` | Show all your sessions |
| `/switch <id>` | Switch to different session |
| `/status` | Current session details |
| `/kill` | Terminate active session |
| `/restart` | Restart Claude Code process |
| `/pwd` | Show current directory |
| `/ls [path]` | List files in workspace |
| `/help` | Full command reference |

**Pro tip:** Just send any message to interact with Claude Code in your active session!

## 🔍 Monitoring

### View Logs

```bash
# Follow logs in real-time
docker compose logs -f omnik

# View last 50 lines
docker compose logs --tail=50 omnik
```

### Check Container Status

```bash
docker compose ps
docker compose top omnik
```

### Database Location

- **Database**: Docker volume `omnik_omnik-data` (contains SQLite database)
- **Workspaces**: Docker volume `omnik_workspace` (contains all session workspaces)
- **Logs**: `./logs/` directory (bind mounted for easy access)

## 🛠 Troubleshooting

### Bot Not Responding

1. Check logs: `docker compose logs omnik`
2. Verify Telegram token is correct
3. Ensure your user ID is authorized
4. Check bot status on Telegram with @BotFather

### Claude Code Not Starting

1. Verify Anthropic API key is set
2. Check workspace permissions
3. Review session logs: `/logs` command in Telegram

### Database Issues

```bash
# Reset database (WARNING: This deletes all data)
docker compose down -v
docker compose up -d omnik
```

## 📂 Project Structure

```
omnik/
├── src/
│   ├── bot/
│   │   ├── handlers.py          # Telegram command handlers
│   │   └── session_manager.py   # Claude Code subprocess management
│   ├── database/
│   │   ├── manager.py           # Database CRUD operations
│   │   └── schema.py            # SQLAlchemy models
│   ├── models/
│   │   ├── session.py           # Session Pydantic model
│   │   ├── message.py           # Message Pydantic model
│   │   ├── audit.py             # Audit log model
│   │   └── workspace.py         # Workspace info model
│   ├── utils/
│   │   ├── config.py            # Configuration management
│   │   └── logging.py           # Logging setup
│   └── main.py                  # Application entry point
├── Dockerfile                   # Container definition
├── docker-compose.yml           # Service orchestration
├── requirements.txt             # Python dependencies
├── .env                         # Environment configuration
└── README.md                    # Project documentation
```

## 🔐 Security Notes

1. **Never commit secrets**: `.env` and `secrets/` are gitignored
2. **User authorization**: Only your Telegram user ID can access the bot
3. **Workspace isolation**: Each session has its own isolated directory
4. **Audit logging**: All commands are logged for security review
5. **No Docker socket**: Claude Code runs as subprocess, not separate container

## 🔄 Updating the Application

```bash
# Pull latest changes
git pull

# Rebuild and restart
docker compose build omnik
docker compose up -d omnik
```

## 📊 Resource Usage

**Expected resource usage:**
- **CPU**: ~0.5 cores idle, up to 2 cores during active sessions
- **Memory**: ~512MB baseline, up to 4GB peak
- **Disk**: Varies based on workspace files and database size

## 🎯 Example Workflow

```
You: /new my-fastapi-app
Bot: ✅ Created session abc12345
     Name: my-fastapi-app
     Send a message to start coding!

You: Create a FastAPI hello world application with a /health endpoint
Bot: 🤔 Processing...
     [Claude Code responds with code and creates files]

You: /ls
Bot: 📁 ./
     📄 main.py (245 bytes)
     📄 requirements.txt (15 bytes)

You: Run the application
Bot: [Claude Code executes and shows output]
```

## 🐛 Known Issues

1. **Tini Warning**: The tini PID 1 warning can be ignored - it's cosmetic and doesn't affect functionality
2. **Long Messages**: Telegram messages over 4096 characters are automatically truncated with "... (truncated)" indicator

## 🚀 Future Enhancements

See [PRD.md](PRD.md) for the complete roadmap, including:
- Multi-user support
- File upload/download
- Git integration
- Workspace templates
- RAG and context enhancement
- Monitoring dashboards

## 📞 Support

- **Issues**: Create an issue in the repository
- **Documentation**: See [README.md](README.md) for detailed information
- **PRD**: See [PRD.md](PRD.md) for product requirements

---

**Status**: ✅ Ready for use!
**Version**: 0.1.0
**Last Updated**: 2025-10-24
