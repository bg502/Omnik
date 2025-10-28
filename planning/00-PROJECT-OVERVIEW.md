# OMNI Project Overview

## Project Name
OMNI - Omnik Automation & API System

## Goal
Enable programmatic command execution in Omnik bot through self-messaging or API, allowing automated workflows that leverage Claude AI, session management, and file operations without manual Telegram interaction.

## Core Use Case
**"Idea to Implementation" Workflow**:
1. User sends message via cURL/API to bot's authorized chat
2. Bot receives and processes the message as if from authorized user
3. Bot creates session, sets up directories, documents idea
4. Bot uses Archon MCP to create project plan with tasks
5. Bot implements first task using Claude AI with knowledge base context

## Current State Analysis

### What Works
- ✅ Telegram bot with full command set (sessions, filesystem, MCP, Claude queries)
- ✅ User ID authentication (`OMNI_AUTHORIZED_USER_ID=55340979`)
- ✅ Chat ID authentication implemented (`OMNI_TG_AUTH_CHAT_ID`)
- ✅ Session management with working directory persistence
- ✅ Claude CLI integration with stream responses
- ✅ MCP integration (Archon available at `http://archon-mcp:8051/mcp`)
- ✅ Can send messages AS the bot using Telegram API

### Current Issues
- ❌ Bot not receiving messages from group chat (`-1003048532828`)
- ❌ Group messages sent via API don't trigger bot processing
- ❌ No REST API for programmatic command execution
- ❌ No workflow/automation system

### Why Group Messages Aren't Working
**Hypothesis**: When bot sends message to group using `sendMessage` API:
- Message `from.id` = bot's own ID (7490276912)
- Message `chat.id` = group chat ID (-1003048532828)
- Current auth logic: `msg.From.ID == b.authorizedUID` (55340979) → FAILS
- Chat-based auth: `msg.Chat.ID == b.authChatID` (-1003048532828) → SHOULD PASS

**Possible Root Causes**:
1. Bot not subscribed to group updates (Privacy Mode enabled?)
2. Bot not admin in group or lacks permissions
3. Forum/topic-based group requires additional handling
4. Bot API endpoint not configured for group updates

## Architecture Decision: Single Bot vs. Dual Bot

### Option A: Single Bot (Self-Processing)
**Pros**:
- Simpler architecture
- No additional bot management
- Uses existing authentication system

**Cons**:
- Bot processing its own messages (unusual pattern)
- May conflict with Telegram's bot design
- Privacy mode complications

### Option B: Dual Bot (Sender + Processor)
**Pros**:
- Clean separation: Bot A sends, Bot B processes
- Natural authentication flow
- Aligns with Telegram's design patterns
- More scalable for multiple automation sources

**Cons**:
- Two bots to manage
- Additional token/configuration
- Slightly more complex setup

### Recommendation: **Option B - Dual Bot**
Creates cleaner architecture and avoids edge cases with self-processing.

## Alternative: REST API (Recommended)

### Option C: HTTP API Layer
**Approach**: Add REST API server alongside Telegram bot

**Pros**:
- ✅ Direct programmatic access (no Telegram intermediary)
- ✅ Standard HTTP authentication (API keys)
- ✅ Better for automation/workflows
- ✅ Can use existing command handlers
- ✅ Supports both sync and async operations
- ✅ Easy to test and debug

**Cons**:
- Additional server component
- Need to expose port (8080)
- Slightly more complex Docker setup

**Verdict**: **BEST OPTION** - Most flexible and standard approach

## Proposed Solution: Hybrid Approach

### Architecture Components

1. **REST API Server** (Primary automation interface)
   - HTTP endpoints for all commands
   - API key authentication
   - JSON request/response
   - Runs on port 8080 alongside Telegram bot

2. **Telegram Bot** (Human interface)
   - Remains unchanged for human users
   - Commands via Telegram messages
   - Real-time streaming responses

3. **Shared Command Layer**
   - Command handlers used by both API and Bot
   - Session manager
   - Claude integration
   - File system operations

4. **Workflow Engine**
   - JSON-based workflow definitions
   - Multi-step execution with dependencies
   - Variable interpolation
   - Progress tracking

5. **Archon Integration**
   - Project creation and management
   - Knowledge base queries
   - Task breakdown generation

## Success Criteria

1. ✅ Can execute any bot command programmatically via REST API
2. ✅ API authenticated with API keys
3. ✅ Workflow engine executes multi-step operations
4. ✅ "Idea to Implementation" workflow works end-to-end
5. ✅ Archon integration creates project plans from ideas
6. ✅ Telegram bot functionality unchanged
7. ✅ Full documentation and examples
8. ✅ Comprehensive test coverage

## Technical Stack

- **Language**: Go 1.21+
- **HTTP Router**: `chi` (lightweight, stdlib-compatible)
- **API Auth**: API key (environment variable)
- **Workflow Format**: JSON
- **Container**: Docker (existing)
- **MCP**: Archon knowledge base

## Timeline Estimate

- **Phase 1**: Investigation & Design - 1 day
- **Phase 2**: REST API Foundation - 2-3 days
- **Phase 3**: Command Abstraction - 1-2 days
- **Phase 4**: Workflow Engine - 3-4 days
- **Phase 5**: Archon Integration - 2-3 days
- **Phase 6**: Testing & Docs - 2 days

**Total**: 11-15 days

## Risk Mitigation

- **Breaking Changes**: Keep Telegram bot fully functional
- **Concurrency**: Ensure thread-safe session access
- **MCP Reliability**: Add error handling and retries
- **Complexity**: Start simple, iterate based on feedback
