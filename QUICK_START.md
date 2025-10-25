# Quick Start Guide

## ✅ Current Status

Your omnik bot is **running and ready** with Claude Pro account authentication!

```bash
$ docker compose ps
NAME      IMAGE         COMMAND                  SERVICE   CREATED        STATUS
omnik     omnik-omnik   "tini -g -- python -…"   omnik     1 min ago      Up (healthy)
```

## 🔐 Authentication Setup Needed

Before you can use the bot, Claude Code needs to be authenticated with your Pro account.

### Option 1: Authenticate via omnik Container (Recommended)

```bash
# Run authentication in the omnik container
docker compose exec -it omnik claude auth login

# Follow the prompts:
# 1. Choose authentication method (browser recommended)
# 2. Complete authentication in your browser
# 3. Verify success
```

### Option 2: Authenticate via code-agent Container

If you have the code-agent container running:

```bash
# Start code-agent if not running
docker compose up -d code-agent

# Authenticate
docker compose exec -it code-agent claude auth login

# The authentication is shared with omnik via the agent-home volume
```

### Verify Authentication

```bash
# Check authentication status
docker compose exec omnik claude auth status

# Should show: "Logged in as: your@email.com"
```

## 📱 Using the Bot

### 1. Find Your Bot

Open Telegram and search for your bot using the username you created with @BotFather.

### 2. Start Chatting

```
You: /start
Bot: 👋 Welcome to omnik - Claude Code on Telegram

     Commands:
     /new [name] - Create new session
     /list - Show all sessions
     ...
```

### 3. Create Your First Session

```
You: /new my-project
Bot: ✅ Created session abc12345
     Name: my-project
     Workspace: /workspace/abc12345

     Send a message to start coding!
```

### 4. Start Coding!

```
You: Create a Python script that says hello
Bot: 🤔 Processing...
     [Claude Code responds and creates files]
```

## 🔍 Monitoring

### View Logs

```bash
# Follow logs in real-time
docker compose logs -f omnik

# Check for authentication messages
docker compose logs omnik | grep -i "claude pro\|api key\|auth"
```

### Check Container Health

```bash
docker compose ps
docker compose top omnik
```

## 🐛 Troubleshooting

### Bot Starts But Claude Code Fails

**Check authentication**:
```bash
docker compose exec omnik claude auth status
```

**If not authenticated**:
```bash
docker compose exec -it omnik claude auth login
```

### "No active session" Error

Create a session first:
```
/new my-session
```

### Bot Not Responding in Telegram

1. Check bot is running: `docker compose ps`
2. Check logs: `docker compose logs omnik --tail=50`
3. Verify bot token is correct in `.env`
4. Ensure your user ID is authorized

## 📊 Your Configuration

```env
AUTHORIZED_USER_ID=55340979  ✅ Configured
TELEGRAM_BOT_TOKEN=7490276912:AAH...  ✅ Configured
ANTHROPIC_API_KEY=  ✅ Using Pro Account (Empty)
```

## 🚀 Next Steps

1. **Authenticate Claude Code**: Run `docker compose exec -it omnik claude auth login`
2. **Open Telegram**: Find your bot
3. **Send `/start`**: Initialize the bot
4. **Create session**: `/new test`
5. **Start coding**: Just send a message!

## 📚 More Information

- **Full Documentation**: See [README.md](README.md)
- **Authentication Details**: See [AUTHENTICATION.md](AUTHENTICATION.md)
- **Deployment Guide**: See [DEPLOYMENT.md](DEPLOYMENT.md)
- **Product Requirements**: See [PRD.md](PRD.md)

## 💡 Example Workflow

```
# Terminal
$ docker compose exec -it omnik claude auth login
✓ Authenticated successfully

# Telegram
You: /start
Bot: Welcome! 👋

You: /new fastapi-demo
Bot: ✅ Created session

You: Create a FastAPI app with a hello endpoint
Bot: [Creates main.py with FastAPI code]

You: /ls
Bot: 📁 ./
     📄 main.py (245 bytes)

You: Run it
Bot: [Executes and shows output]
```

---

**Status**: ✅ Bot running, needs Claude authentication
**Action Required**: Run `docker compose exec -it omnik claude auth login`
