# Phase 1: Investigation Tasks

## Phase Goal
Understand why group chat messages aren't being received and determine the best architecture for programmatic automation.

---

## Task: Investigate Telegram Bot Group Chat Configuration

**Description**:
Determine why bot is not receiving messages sent to group chat `-1003048532828` via Telegram API, even though authentication logic should allow it.

**Steps**:
1. Check bot's Privacy Mode setting via BotFather
2. Verify bot is admin in the group with correct permissions
3. Review group type (supergroup vs. regular group)
4. Check if forum/topics mode is enabled
5. Test with `/getBotUpdates` to see if group messages appear
6. Review Telegram Bot API documentation for group message handling

**Acceptance Criteria**:
- Root cause identified for why group messages aren't received
- Documentation of required bot permissions and settings
- Clear understanding of Telegram's group message delivery rules

**Estimated Effort**: 2-4 hours

**Priority**: HIGH

**Dependencies**: None

---

## Task: Test Alternative Message Delivery Approaches

**Description**:
Test different methods of delivering programmatic messages to the bot to find the most reliable approach.

**Test Scenarios**:
1. **Private Message**: Send via API to bot's private chat
2. **Group Message**: Fix group permissions and retry
3. **Channel Post**: Create private channel, add bot, test
4. **Webhook**: Configure webhook endpoint for receiving updates

**Acceptance Criteria**:
- Document which methods successfully trigger bot processing
- Measure latency for each approach
- Identify most reliable method for automation

**Estimated Effort**: 3-4 hours

**Priority**: HIGH

**Dependencies**: "Investigate Telegram Bot Group Chat Configuration"

---

## Task: Evaluate Single Bot vs Dual Bot Architecture

**Description**:
Analyze pros/cons of having bot process its own messages vs. using a second bot to send commands.

**Analysis Points**:
1. **Single Bot Approach**:
   - Can bot reliably receive its own messages?
   - Are there Telegram API limitations?
   - What edge cases exist?

2. **Dual Bot Approach**:
   - Setup complexity
   - Cost (additional bot token)
   - Maintenance overhead
   - Security considerations

3. **REST API Approach**:
   - Implementation effort
   - Deployment changes (port exposure)
   - Authentication mechanisms
   - Integration with existing code

**Deliverables**:
- Comparison matrix with pros/cons
- Recommendation with justification
- Architecture diagram for chosen approach

**Acceptance Criteria**:
- Clear recommendation documented
- Team consensus on approach
- Architecture diagram created

**Estimated Effort**: 2-3 hours

**Priority**: HIGH

**Dependencies**: "Test Alternative Message Delivery Approaches"

---

## Task: Design REST API Specification

**Description**:
Create detailed API specification for all bot commands accessible via HTTP.

**Endpoints to Design**:
```
POST /api/v1/session/create
POST /api/v1/session/switch
GET  /api/v1/session/list
GET  /api/v1/session/current
DELETE /api/v1/session/:name

POST /api/v1/fs/cd
GET  /api/v1/fs/pwd
GET  /api/v1/fs/ls
GET  /api/v1/fs/cat
POST /api/v1/fs/exec

GET  /api/v1/mcp/list
POST /api/v1/mcp/add

POST /api/v1/claude/query

GET  /api/v1/health
```

**Specification Elements**:
- Request/response JSON schemas
- Authentication headers
- Error response formats
- Status codes
- Rate limiting considerations

**Deliverables**:
- API specification document (OpenAPI/Swagger format)
- Example requests and responses
- Error handling documentation

**Acceptance Criteria**:
- All existing bot commands mapped to API endpoints
- Request/response schemas defined
- Authentication mechanism specified
- Example cURL commands for each endpoint

**Estimated Effort**: 4-6 hours

**Priority**: MEDIUM

**Dependencies**: "Evaluate Single Bot vs Dual Bot Architecture"

---

## Task: Research Go HTTP Frameworks

**Description**:
Select appropriate HTTP router/framework for API server component.

**Options to Evaluate**:
1. **stdlib net/http + chi**
   - Lightweight
   - Middleware support
   - Context-aware

2. **gin**
   - Fast
   - Popular
   - Built-in features

3. **echo**
   - Simple
   - Good middleware
   - Performance

4. **fiber**
   - Very fast
   - Express-like API
   - Modern features

**Evaluation Criteria**:
- Performance
- Middleware ecosystem
- Learning curve
- Community support
- Compatibility with existing code
- Binary size impact

**Deliverables**:
- Comparison table
- Recommendation with justification
- Sample implementation (hello world)

**Acceptance Criteria**:
- Framework selected
- Performance benchmarks documented
- Sample code demonstrates integration

**Estimated Effort**: 2-3 hours

**Priority**: MEDIUM

**Dependencies**: None

---

## Task: Design Workflow JSON Format

**Description**:
Create specification for JSON-based workflow format that supports multi-step operations with dependencies.

**Format Requirements**:
- Step definitions with types
- Variable interpolation
- Dependency management (depends_on)
- Error handling
- Conditional execution
- Parallel execution support

**Example Workflow Types**:
- `session.create`
- `session.switch`
- `fs.cd`, `fs.exec`, `fs.write`
- `claude.query`
- `archon.create_project`
- `archon.add_task`

**Deliverables**:
- JSON schema for workflow format
- Example workflows for common use cases
- Documentation of all supported step types

**Acceptance Criteria**:
- JSON schema validates workflow definitions
- At least 3 example workflows created
- Variable interpolation syntax documented
- Dependency resolution algorithm defined

**Estimated Effort**: 4-5 hours

**Priority**: MEDIUM

**Dependencies**: "Design REST API Specification"

---

## Task: Create Architecture Diagrams

**Description**:
Create visual diagrams showing system architecture, data flow, and component interactions.

**Diagrams to Create**:
1. **Overall System Architecture**
   - Telegram Bot
   - REST API Server
   - Shared Command Layer
   - Session Manager
   - Claude Integration
   - Archon MCP

2. **API Request Flow**
   - Client → API → Command Handler → Response

3. **Workflow Execution Flow**
   - Workflow submission → Parsing → Execution → Status updates

4. **Authentication Flow**
   - API key validation
   - Session management

**Deliverables**:
- Architecture diagrams (mermaid or draw.io)
- Sequence diagrams for key flows
- Component interaction diagrams

**Acceptance Criteria**:
- All major components represented
- Data flow clearly illustrated
- Diagrams included in documentation

**Estimated Effort**: 3-4 hours

**Priority**: LOW

**Dependencies**: "Evaluate Single Bot vs Dual Bot Architecture", "Design REST API Specification"
