# Authentication Setup for omnik

## Overview

omnik supports two authentication methods for Claude Code:

1. **Claude Pro Account** (Recommended) - Uses your existing authenticated Claude Pro account
2. **API Key** - Uses an Anthropic API key

## Method 1: Claude Pro Account Authentication (Current Setup)

This method uses your existing Claude Pro account authentication, which is ideal if you're already logged in to Claude Code on your system.

### How It Works

- omnik shares the Claude authentication stored in the `agent-home` Docker volume
- Claude Code CLI automatically uses your existing session
- No API key needed in `.env`

### Setup

1. **Ensure Claude Code is authenticated** on your system or in the code-agent container:
   ```bash
   # If using the code-agent container
   docker compose exec code-agent claude auth login

   # Or if running locally
   claude auth login
   ```

2. **Configure omnik** with your Telegram credentials (`.env`):
   ```bash
   AUTHORIZED_USER_ID=55340979
   TELEGRAM_BOT_TOKEN=7490276912:AAH3orrFvgjWiM4--QVfbt8ywrlebhpRPZ4
   ANTHROPIC_API_KEY=  # Leave empty for Pro account auth
   ```

3. **Start the bot**:
   ```bash
   docker compose up -d omnik
   ```

4. **Verify authentication** in logs:
   ```bash
   docker compose logs omnik | grep "Claude Pro"
   ```

   You should see: `Using existing Claude Pro account authentication`

### Advantages

- ✅ No API key management
- ✅ Uses your existing Pro subscription
- ✅ Shared authentication with other Claude Code instances
- ✅ Automatic credential refresh

### Volume Sharing

The `agent-home` volume is shared between services:

```yaml
volumes:
  - agent-home:/home/appuser/.config  # Contains Claude authentication
```

This allows omnik to access the same Claude authentication as your code-agent container.

## Method 2: API Key Authentication

If you prefer to use an API key or don't have a Pro account, you can provide an Anthropic API key.

### Setup

1. **Get your API key** from [Anthropic Console](https://console.anthropic.com/)

2. **Configure `.env`**:
   ```bash
   AUTHORIZED_USER_ID=55340979
   TELEGRAM_BOT_TOKEN=7490276912:AAH3orrFvgjWiM4--QVfbt8ywrlebhpRPZ4
   ANTHROPIC_API_KEY=sk-ant-api03-...  # Your API key
   ```

3. **Restart omnik**:
   ```bash
   docker compose restart omnik
   ```

4. **Verify** in logs:
   ```bash
   docker compose logs omnik | grep "Using provided API key"
   ```

### Advantages

- ✅ Independent from Pro account
- ✅ Programmatic access
- ✅ Usage tracking per API key

## Troubleshooting

### Issue: "Authentication required" errors

**Solution**: Authenticate Claude Code in the shared volume:

```bash
# Option 1: Use code-agent container
docker compose up -d code-agent
docker compose exec code-agent claude auth login

# Option 2: Use omnik container directly
docker compose exec omnik claude auth login

# Follow the prompts to authenticate
```

### Issue: Bot starts but Claude Code fails

**Check logs**:
```bash
docker compose logs omnik --tail=50
```

**Common causes**:
1. No authentication available (neither API key nor Pro account)
2. Pro account session expired
3. API key invalid

**Solution**: Either provide a valid API key or re-authenticate:
```bash
docker compose exec omnik claude auth login
```

### Issue: "Using existing Claude Pro account authentication" but still getting auth errors

**Verify authentication**:
```bash
# Check if Claude is authenticated in the container
docker compose exec omnik claude auth status

# If not authenticated, login
docker compose exec omnik claude auth login
```

### Issue: Want to switch from Pro account to API key

1. Edit `.env` and add your API key:
   ```bash
   ANTHROPIC_API_KEY=sk-ant-api03-...
   ```

2. Restart:
   ```bash
   docker compose restart omnik
   ```

### Issue: Want to switch from API key to Pro account

1. Edit `.env` and remove the API key:
   ```bash
   ANTHROPIC_API_KEY=  # Leave empty
   ```

2. Ensure you're authenticated:
   ```bash
   docker compose exec omnik claude auth login
   ```

3. Restart:
   ```bash
   docker compose restart omnik
   ```

## Current Configuration

Based on your `.env`:

```bash
AUTHORIZED_USER_ID=55340979
TELEGRAM_BOT_TOKEN=7490276912:AAH3orrFvgjWiM4--QVfbt8ywrlebhpRPZ4
ANTHROPIC_API_KEY=  # Empty - using Pro account authentication
```

**Status**: ✅ Using Claude Pro Account Authentication

## Checking Authentication Status

### Via Container

```bash
# Check Claude authentication status
docker compose exec omnik claude auth status

# View current user
docker compose exec omnik claude auth whoami
```

### Via Logs

```bash
# See which authentication method is being used
docker compose logs omnik | grep -i "auth\|api key\|pro account"
```

## Security Notes

1. **API Keys**: Never commit API keys to git. Use `.env` file (already gitignored)
2. **Shared Volumes**: The `agent-home` volume contains authentication tokens - keep it secure
3. **Container Access**: Only authorized containers can access the shared authentication
4. **Audit Logging**: All authentication attempts are logged in the omnik audit log

## Advanced: Manual Authentication Setup

If you need to manually set up authentication:

```bash
# Enter the container
docker compose exec -u appuser omnik bash

# Authenticate
claude auth login

# Verify
claude auth status

# Test
claude --version
```

## Next Steps

1. ✅ Bot is running with Pro account authentication
2. 📱 Open Telegram and message your bot
3. 🧪 Test with `/start` command
4. 🚀 Create your first session with `/new test`

---

**Current Status**: ✅ Bot configured with Claude Pro account authentication
**Last Updated**: 2025-10-24
