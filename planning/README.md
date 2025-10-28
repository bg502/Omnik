# OMNI Project Planning

This directory contains comprehensive task breakdowns for implementing the OMNI automation system for Omnik bot.

## Project Goal

Enable programmatic command execution in Omnik through REST API and JSON-based workflows, with intelligent project planning via Archon knowledge base integration.

## Core Use Case: "Idea to Implementation"

Send an idea via API → Bot creates session → Documents idea → Uses Archon for planning → Claude implements first task automatically.

## Planning Documents

### [00-PROJECT-OVERVIEW.md](./00-PROJECT-OVERVIEW.md)
- Project goals and success criteria
- Current state analysis
- Architecture decisions (Single Bot vs Dual Bot vs REST API)
- Recommendation: **REST API + Workflow Engine** (best solution)
- Technical stack and timeline

### [01-INVESTIGATION-TASKS.md](./01-INVESTIGATION-TASKS.md)
**Phase 1: Investigation & Design** (7 tasks)
- Investigate Telegram group chat issues
- Test message delivery approaches
- Evaluate architectures
- Design REST API specification
- Research Go HTTP frameworks
- Design workflow JSON format
- Create architecture diagrams

**Estimated Effort**: 20-30 hours

### [02-CORE-AUTOMATION-TASKS.md](./02-CORE-AUTOMATION-TASKS.md)
**Phase 2: REST API & Command Abstraction** (17 tasks)
- Create API server foundation
- Implement authentication middleware
- Create command handler interface
- Refactor existing commands to handlers
- Implement API endpoints for:
  - Session management
  - Filesystem operations
  - Claude AI queries
  - MCP server management
- Update Docker configuration
- Write tests and documentation

**Estimated Effort**: 60-80 hours

### [03-ARCHON-INTEGRATION-TASKS.md](./03-ARCHON-INTEGRATION-TASKS.md)
**Phase 3: Workflow Engine & Archon Integration** (11 tasks)
- Implement workflow type definitions
- Create workflow parser
- Implement variable interpolation
- Build workflow executor
- Create Archon MCP client
- Implement Archon workflow step executors
- Create "Idea to Implementation" template
- Build workflow API endpoints
- Add workflow storage
- Implement progress streaming
- Add Telegram workflow command

**Estimated Effort**: 45-60 hours

### [04-TESTING-DEPLOYMENT.md](./04-TESTING-DEPLOYMENT.md)
**Phase 4: Testing, Documentation & Deployment** (17 tasks)
- End-to-end testing
- Load testing
- Health checks and monitoring
- API documentation
- Workflow guide
- Usage examples
- Deployment guide
- Security audit
- Performance optimization
- Release preparation

**Estimated Effort**: 50-70 hours

## Total Effort Estimate

**175-240 hours** (approximately 4-6 weeks for 1 developer)

## Task Format

Each task includes:
- **Title**: Clear task name
- **Description**: What needs to be done
- **Implementation Details**: Code examples, file locations
- **Acceptance Criteria**: Definition of done
- **Estimated Effort**: Time estimate
- **Priority**: HIGH/MEDIUM/LOW
- **Dependencies**: Which tasks must complete first

**Note**: Task IDs intentionally omitted - Archon will assign these when creating tasks.

## Key Design Decisions

### Architecture: REST API + Workflow Engine ✅

**Why not Telegram group chat approach?**
- Bot doesn't reliably receive group messages
- Complex permission management
- Telegram API limitations for self-messaging

**Why not dual bot approach?**
- Additional complexity
- Two bots to manage
- Still uses Telegram as intermediary

**Why REST API is best?**
- Direct programmatic access
- Standard HTTP authentication
- Better for automation
- Easy to test and debug
- Language-agnostic clients
- No Telegram intermediary

### Workflow System

**JSON-based workflows with:**
- Variable interpolation: `{{variable_name}}`
- Dependency management: `depends_on`
- Parallel execution: Independent steps run concurrently
- Error handling: Stop on error or continue
- Step types: session, fs, claude, archon
- Progress tracking: Real-time updates via SSE

### Archon Integration

**Capabilities:**
- Create projects from ideas
- Generate task breakdowns
- Search knowledge base
- Provide implementation context
- Guide Claude's implementation

## Technology Stack

- **Language**: Go 1.21+
- **HTTP Router**: chi (lightweight, stdlib-compatible)
- **API Auth**: API key (environment variable)
- **Workflow Format**: JSON
- **MCP**: Archon at `http://archon-mcp:8051/mcp`
- **Container**: Docker (existing)

## API Endpoint Overview

```
# Session Management
POST   /api/v1/session/create
POST   /api/v1/session/switch
GET    /api/v1/session/list
GET    /api/v1/session/current
DELETE /api/v1/session/:name

# Filesystem
POST /api/v1/fs/cd
GET  /api/v1/fs/pwd
GET  /api/v1/fs/ls
GET  /api/v1/fs/cat
POST /api/v1/fs/exec
POST /api/v1/fs/write

# Claude AI
POST /api/v1/claude/query
POST /api/v1/claude/query/stream  # SSE streaming

# MCP
GET  /api/v1/mcp/list
POST /api/v1/mcp/add

# Workflows
POST /api/v1/workflow/execute
POST /api/v1/workflow/execute/:template
GET  /api/v1/workflow/execution/:id
GET  /api/v1/workflow/execution/:id/stream  # SSE
POST /api/v1/workflow/execution/:id/cancel
GET  /api/v1/workflow/templates

# System
GET /api/v1/health
GET /api/v1/metrics  # Prometheus format
```

## Example Workflow: "Idea to Implementation"

```json
{
  "name": "idea-to-implementation",
  "variables": {
    "project_name": "required",
    "project_description": "required",
    "idea_content": "required"
  },
  "steps": [
    {"id": "create_session", "type": "session.create", "params": {...}},
    {"id": "create_dirs", "type": "fs.exec", "params": {...}, "depends_on": ["create_session"]},
    {"id": "write_idea", "type": "fs.write", "params": {...}, "depends_on": ["create_dirs"]},
    {"id": "create_archon_project", "type": "archon.create_project", "params": {...}, "depends_on": ["write_idea"]},
    {"id": "implement_with_claude", "type": "claude.query", "params": {...}, "depends_on": ["create_archon_project"]}
  ]
}
```

## Success Criteria

1. ✅ All bot commands accessible via REST API
2. ✅ API authenticated with API keys
3. ✅ Workflow engine executes multi-step operations
4. ✅ "Idea to Implementation" workflow works end-to-end
5. ✅ Archon integration creates project plans
6. ✅ Telegram bot functionality unchanged (backward compatible)
7. ✅ Full documentation and examples
8. ✅ Comprehensive test coverage

## Migration to Archon

These planning documents are designed to be easily migrated to Archon tasks:

1. Each task has clear title and description
2. Dependencies reference other task titles
3. Acceptance criteria define completion
4. Effort estimates provided
5. Priorities assigned
6. No task IDs (Archon assigns these)

## Next Steps

1. Review planning documents
2. Create OMNI project in Archon
3. Import tasks from planning docs
4. Begin implementation with Phase 1 (Investigation)
5. Iterate through phases sequentially

## Questions or Issues?

- Review specific phase planning documents for details
- Check [00-PROJECT-OVERVIEW.md](./00-PROJECT-OVERVIEW.md) for architecture decisions
- Consult task descriptions for implementation guidance
