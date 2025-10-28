# Omnik Bot API Control Scripts

This directory contains bash scripts for controlling the **Memnikai API bot** via the Telegram API.

## Dual-Bot Setup

The system now runs two independent bots:

- **Omnik** (main bot): Personal chat + group chat interactions
- **Memnikai** (API bot): Dedicated API access and automation

Both bots:
- Share the same workspace and session storage
- Use per-chat context isolation
- Can work simultaneously without interfering

### How Scripts Work

These scripts send messages to **Memnikai's chat** (`-4958242815`):

1. **Scripts use Omnik bot's token** to send messages
2. Messages appear in Memnikai's chat **from Omnik bot**
3. **Memnikai bot** sees these messages and processes them
4. Responses come from Memnikai bot

**Why this approach?**
Telegram bots cannot see their own messages. If we sent messages using Memnikai's token, Memnikai would never receive them. By sending messages from Omnik bot, Memnikai can see and process them.

## Configuration

Before using any scripts, edit `config.sh` to set your bot token and chat IDs:

```bash
# Omnik bot token (used to send messages that Memnikai will process)
export OMNI_AUTH_API_TOKEN="7490276912:AAH3orrFvgjWiM4--QVfbt8ywrlebhpRPZ4"
# Memnikai's chat ID (where messages are sent)
export CHAT_ID="-4958242815"
export PERSONAL_CHAT_ID=""
```

**Note:** The token is from the **Omnik bot**, but messages go to **Memnikai's chat** where the Memnikai bot processes them.

## Usage

Make scripts executable (first time only):

```bash
chmod +x scripts/*.sh
```

## Available Scripts

### General Commands

**`send-command.sh`** - Send any command or message to the bot
```bash
./scripts/send-command.sh "/status"
./scripts/send-command.sh "Hello, Claude!"
```

**`query.sh`** - Send a query to Claude
```bash
./scripts/query.sh "What is the current date?"
./scripts/query.sh "Create a Python hello world script"
```

### Session Management

**`status.sh`** - Get current session status
```bash
./scripts/status.sh
```

**`sessions.sh`** - List all sessions
```bash
./scripts/sessions.sh
```

**`newsession.sh`** - Create a new session
```bash
./scripts/newsession.sh automation
./scripts/newsession.sh project1 "Working on project 1"
```

### File Navigation

**`pwd.sh`** - Show current working directory
```bash
./scripts/pwd.sh
```

**`ls.sh`** - List files in current directory
```bash
./scripts/ls.sh
```

## Examples

### Create a new session and query Claude:

```bash
# Create automation session
./scripts/newsession.sh automation "Automation testing session"

# Check status
./scripts/status.sh

# Send a query
./scripts/query.sh "List all Python files in the current directory"
```

### Send custom commands:

```bash
# Change directory
./scripts/send-command.sh "/cd /workspace/myproject"

# Execute bash command
./scripts/send-command.sh "/exec ls -la"

# View file
./scripts/send-command.sh "/cat README.md"
```

## Chat Context Isolation

The bot now supports per-chat session isolation:

- **Personal chat** (`PERSONAL_CHAT_ID`): Your personal interactions with the bot
- **Group chat** (`CHAT_ID`): Automation and programmatic access

Each chat maintains its own:
- Current session
- Working directory
- Session history

You can work on different tasks simultaneously in both chats without interference.

## Switching Between Chats

To send commands to your personal chat instead of the group chat, modify the script or set `CHAT_ID`:

```bash
# Send to personal chat
CHAT_ID="55340979" ./scripts/status.sh

# Or edit the script temporarily
```

## Advanced Usage

### Using the Telegram API directly:

```bash
source scripts/config.sh

curl -X POST "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/sendMessage" \
  -H "Content-Type: application/json" \
  -d "{\"chat_id\": \"${CHAT_ID}\", \"text\": \"/status\"}"
```

### Getting bot updates (see recent messages):

```bash
curl "https://api.telegram.org/bot${OMNI_AUTH_API_TOKEN}/getUpdates"
```

## Notes

- All scripts source `config.sh` automatically
- The bot responds in the same chat where the command was sent
- Commands sent via API work exactly like typing them in Telegram
- Use quotes around commands/queries that contain spaces
