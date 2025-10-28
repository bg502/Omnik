# Phase 3: Workflow Engine & Archon Integration

## Phase Goal
Implement JSON-based workflow system with Archon MCP integration for intelligent project planning and automated execution.

---

## Task: Implement Workflow Type Definitions

**Description**:
Create Go structs and types for workflow definition, parsing, and execution.

**File**: `go-bot/internal/workflow/types.go`

**Key Types**:
```go
// Workflow represents a complete workflow definition
type Workflow struct {
    Name        string              `json:"name"`
    Description string              `json:"description"`
    Steps       []Step              `json:"steps"`
    Variables   map[string]Variable `json:"variables"`
}

// Step represents a single workflow step
type Step struct {
    ID         string                 `json:"id"`
    Type       string                 `json:"type"`
    Params     map[string]interface{} `json:"params"`
    DependsOn  []string               `json:"depends_on,omitempty"`
    Condition  string                 `json:"condition,omitempty"`
    OnError    string                 `json:"on_error,omitempty"`
}

// Variable definition
type Variable struct {
    Required    bool        `json:"required"`
    Default     interface{} `json:"default,omitempty"`
    Description string      `json:"description,omitempty"`
}

// Execution represents a workflow execution instance
type Execution struct {
    ID          string
    WorkflowID  string
    Status      ExecutionStatus
    Steps       map[string]*StepExecution
    Variables   map[string]interface{}
    StartedAt   time.Time
    CompletedAt time.Time
    Error       string
}

// StepExecution tracks individual step execution
type StepExecution struct {
    Status      ExecutionStatus
    StartedAt   time.Time
    CompletedAt time.Time
    Result      interface{}
    Error       string
}

// ExecutionStatus enum
type ExecutionStatus string

const (
    StatusPending   ExecutionStatus = "pending"
    StatusRunning   ExecutionStatus = "running"
    StatusCompleted ExecutionStatus = "completed"
    StatusFailed    ExecutionStatus = "failed"
    StatusSkipped   ExecutionStatus = "skipped"
)
```

**Acceptance Criteria**:
- All workflow structures defined
- JSON marshaling/unmarshaling works
- Status types and constants defined
- Documentation for each type

**Estimated Effort**: 3-4 hours

**Priority**: HIGH

**Dependencies**: None

---

## Task: Implement Workflow Parser

**Description**:
Create workflow parser that validates JSON workflow definitions and resolves dependencies.

**File**: `go-bot/internal/workflow/parser.go`

**Functionality**:
1. **JSON Validation**:
   - Valid JSON structure
   - Required fields present
   - Type validation

2. **Dependency Resolution**:
   - Build dependency graph
   - Detect circular dependencies
   - Determine execution order

3. **Variable Validation**:
   - Check required variables provided
   - Apply default values
   - Type checking

**Key Functions**:
```go
func ParseWorkflow(jsonData []byte) (*Workflow, error)
func ValidateWorkflow(w *Workflow) error
func ResolveExecutionOrder(w *Workflow) ([][]string, error) // Returns layers for parallel execution
```

**Acceptance Criteria**:
- Parses valid workflow JSON
- Rejects invalid workflows with clear errors
- Detects circular dependencies
- Returns execution order (with parallelization opportunities)
- Validates variable requirements

**Estimated Effort**: 5-6 hours

**Priority**: HIGH

**Dependencies**: "Implement Workflow Type Definitions"

---

## Task: Implement Variable Interpolation

**Description**:
Create system for replacing variable placeholders in workflow step parameters.

**File**: `go-bot/internal/workflow/interpolation.go`

**Syntax**: `{{variable_name}}`

**Features**:
- Simple variable substitution: `{{project_name}}`
- Step output references: `{{step_id.output.field}}`
- Environment variables: `{{env.HOME}}`
- Built-in functions: `{{timestamp}}`, `{{uuid}}`

**Example**:
```json
{
  "params": {
    "name": "{{project_name}}",
    "path": "/workspace/{{project_name}}/{{timestamp}}",
    "content": "Created by {{env.USER}}"
  }
}
```

**Acceptance Criteria**:
- Variable substitution works
- Step output references work
- Environment variables accessible
- Built-in functions implemented
- Handles missing variables gracefully
- Escaping supported

**Estimated Effort**: 4-5 hours

**Priority**: HIGH

**Dependencies**: "Implement Workflow Type Definitions"

---

## Task: Implement Workflow Executor

**Description**:
Create workflow executor that runs steps in correct order with dependency management.

**File**: `go-bot/internal/workflow/executor.go`

**Key Features**:
1. **Sequential Execution**: Steps run in dependency order
2. **Parallel Execution**: Independent steps run concurrently
3. **Error Handling**: Stop on error or continue based on config
4. **Progress Tracking**: Real-time status updates
5. **Context Cancellation**: Support for stopping execution

**Key Functions**:
```go
func NewExecutor(ctx ExecutionContext) *Executor
func (e *Executor) Execute(w *Workflow, variables map[string]interface{}) (*Execution, error)
func (e *Executor) ExecuteAsync(w *Workflow, variables map[string]interface{}) (string, error) // Returns execution ID
func (e *Executor) GetStatus(executionID string) (*Execution, error)
func (e *Executor) Cancel(executionID string) error
```

**Step Type Registry**:
```go
var stepExecutors = map[string]StepExecutor{
    "session.create":        &SessionCreateExecutor{},
    "session.switch":        &SessionSwitchExecutor{},
    "fs.cd":                 &FsCdExecutor{},
    "fs.exec":               &FsExecExecutor{},
    "fs.write":              &FsWriteExecutor{},
    "claude.query":          &ClaudeQueryExecutor{},
    "archon.create_project": &ArchonCreateProjectExecutor{},
    // ... more executors
}
```

**Acceptance Criteria**:
- Executes steps in correct dependency order
- Parallel execution where possible
- Error handling works (stop/continue)
- Progress tracking accurate
- Can cancel running executions
- Step results accessible to dependent steps

**Estimated Effort**: 8-10 hours

**Priority**: HIGH

**Dependencies**: "Implement Workflow Parser", "Implement Variable Interpolation"

---

## Task: Create Archon MCP Client

**Description**:
Implement Go client for Archon MCP server to interact with knowledge base.

**File**: `go-bot/internal/archon/client.go`

**MCP Server**: `http://archon-mcp:8051/mcp`

**Methods to Implement**:
```go
type Client struct {
    baseURL    string
    httpClient *http.Client
}

// Project Management
func (c *Client) CreateProject(name, description string, context map[string]interface{}) (string, error)
func (c *Client) GetProject(projectID string) (*Project, error)
func (c *Client) ListProjects() ([]*Project, error)

// Task Management
func (c *Client) AddTask(projectID string, task *Task) (string, error)
func (c *Client) GetTask(projectID, taskID string) (*Task, error)
func (c *Client) UpdateTask(projectID, taskID string, updates *TaskUpdate) error

// Knowledge Base
func (c *Client) SearchKnowledge(query string, filters map[string]interface{}) ([]*KnowledgeItem, error)
func (c *Client) GetContext(projectID string) (*ProjectContext, error)
```

**Data Structures**:
```go
type Project struct {
    ID          string
    Name        string
    Description string
    CreatedAt   time.Time
    Tasks       []*Task
}

type Task struct {
    ID          string
    Title       string
    Description string
    Status      string
    Priority    string
    Dependencies []string
    Effort      string
}
```

**Acceptance Criteria**:
- All MCP endpoints accessible
- Project CRUD operations work
- Task management functional
- Knowledge base search works
- Error handling robust
- Connection pooling for performance

**Estimated Effort**: 6-8 hours

**Priority**: HIGH

**Dependencies**: None (can start early)

---

## Task: Implement Archon Workflow Step Executors

**Description**:
Create workflow step executors that use Archon MCP client.

**File**: `go-bot/internal/workflow/executors/archon.go`

**Step Types**:
1. **archon.create_project**
   ```json
   {
     "type": "archon.create_project",
     "params": {
       "name": "{{project_name}}",
       "description": "{{project_description}}",
       "context_file": "{{workspace}}/docs/IDEA.md"
     }
   }
   ```

2. **archon.add_task**
   ```json
   {
     "type": "archon.add_task",
     "params": {
       "project_id": "{{create_project.output.id}}",
       "title": "Setup project structure",
       "description": "Create directories and files",
       "priority": "HIGH"
     }
   }
   ```

3. **archon.search_knowledge**
   ```json
   {
     "type": "archon.search_knowledge",
     "params": {
       "query": "{{search_query}}",
       "filters": {"domain": "backend"}
     }
   }
   ```

4. **archon.get_context**
   ```json
   {
     "type": "archon.get_context",
     "params": {
       "project_id": "{{create_project.output.id}}"
     }
   }
   ```

**Acceptance Criteria**:
- All Archon step types implemented
- Step outputs accessible to other steps
- Error handling for MCP failures
- Documentation for each step type

**Estimated Effort**: 4-5 hours

**Priority**: HIGH

**Dependencies**: "Create Archon MCP Client", "Implement Workflow Executor"

---

## Task: Create "Idea to Implementation" Workflow Template

**Description**:
Create predefined workflow template for the core use case: turning an idea into a working implementation.

**File**: `go-bot/internal/workflow/templates/idea-to-implementation.json`

**Workflow Steps**:
1. Create session with project name
2. Create project directory structure
3. Write idea document to `docs/IDEA.md`
4. Create Archon project with idea context
5. Get first task from Archon
6. Ask Claude to implement first task with context

**Template**:
```json
{
  "name": "idea-to-implementation",
  "description": "Transform an idea into initial implementation using Archon planning and Claude coding",
  "variables": {
    "project_name": {"required": true, "description": "Name of the project"},
    "project_description": {"required": true, "description": "Brief project description"},
    "idea_content": {"required": true, "description": "Detailed idea description"}
  },
  "steps": [
    {
      "id": "create_session",
      "type": "session.create",
      "params": {
        "name": "{{project_name}}",
        "description": "{{project_description}}",
        "working_dir": "/workspace"
      }
    },
    {
      "id": "create_dirs",
      "type": "fs.exec",
      "params": {
        "command": "mkdir -p {{project_name}}/{docs,src,tests}"
      },
      "depends_on": ["create_session"]
    },
    {
      "id": "write_idea",
      "type": "fs.write",
      "params": {
        "path": "{{project_name}}/docs/IDEA.md",
        "content": "# {{project_name}}\n\n{{idea_content}}"
      },
      "depends_on": ["create_dirs"]
    },
    {
      "id": "create_archon_project",
      "type": "archon.create_project",
      "params": {
        "name": "{{project_name}}",
        "description": "{{project_description}}",
        "context_file": "{{project_name}}/docs/IDEA.md"
      },
      "depends_on": ["write_idea"]
    },
    {
      "id": "get_project_context",
      "type": "archon.get_context",
      "params": {
        "project_id": "{{create_archon_project.output.id}}"
      },
      "depends_on": ["create_archon_project"]
    },
    {
      "id": "implement_with_claude",
      "type": "claude.query",
      "params": {
        "prompt": "I have created a new project called {{project_name}}. The idea is documented in docs/IDEA.md. Archon has created a project plan. Please review the plan and implement the first task. Use the knowledge base context to inform your implementation.",
        "workspace": "/workspace/{{project_name}}"
      },
      "depends_on": ["get_project_context"]
    }
  ]
}
```

**Acceptance Criteria**:
- Template executes successfully end-to-end
- All steps complete in correct order
- Archon project created with tasks
- Claude receives proper context
- Implementation begins automatically

**Estimated Effort**: 3-4 hours

**Priority**: HIGH

**Dependencies**: "Implement Workflow Executor", "Implement Archon Workflow Step Executors"

---

## Task: Create Workflow API Endpoints

**Description**:
Implement REST API endpoints for workflow execution and management.

**File**: `go-bot/internal/api/handlers/workflow.go`

**Endpoints**:
```
POST /api/v1/workflow/execute       # Execute workflow from JSON
POST /api/v1/workflow/execute/:template # Execute predefined template
GET  /api/v1/workflow/execution/:id # Get execution status
POST /api/v1/workflow/execution/:id/cancel # Cancel execution
GET  /api/v1/workflow/templates     # List available templates
GET  /api/v1/workflow/templates/:name # Get template definition
```

**Request Example**:
```json
POST /api/v1/workflow/execute/idea-to-implementation
{
  "variables": {
    "project_name": "my-scraper",
    "project_description": "Web scraper for news headlines",
    "idea_content": "Build a Python-based web scraper that..."
  }
}

Response:
{
  "success": true,
  "data": {
    "execution_id": "exec-123abc",
    "status": "running",
    "workflow": "idea-to-implementation",
    "started_at": "2025-10-27T21:30:00Z"
  }
}
```

**Acceptance Criteria**:
- Can execute workflows from JSON
- Can execute predefined templates
- Status polling works
- Can cancel running executions
- Template listing works
- Error messages clear

**Estimated Effort**: 4-5 hours

**Priority**: HIGH

**Dependencies**: "Implement Workflow Executor", "Create Idea to Implementation Workflow Template"

---

## Task: Implement Workflow Storage

**Description**:
Add persistence for workflow definitions and execution history.

**File**: `go-bot/internal/workflow/storage.go`

**Storage Requirements**:
1. **Templates**: Store workflow templates
2. **Executions**: Store execution history
3. **Results**: Store step results for reference

**Storage Location**: `/workspace/.omnik-workflows/`

**File Structure**:
```
/workspace/.omnik-workflows/
├── templates/
│   ├── idea-to-implementation.json
│   ├── multi-project-setup.json
│   └── code-review.json
├── executions/
│   ├── exec-123abc.json
│   └── exec-456def.json
└── results/
    ├── exec-123abc/
    │   ├── step-1-result.json
    │   └── step-2-result.json
    └── exec-456def/
        └── step-1-result.json
```

**Key Functions**:
```go
func SaveTemplate(name string, workflow *Workflow) error
func LoadTemplate(name string) (*Workflow, error)
func ListTemplates() ([]*Workflow, error)

func SaveExecution(exec *Execution) error
func LoadExecution(id string) (*Execution, error)
func ListExecutions(filters map[string]interface{}) ([]*Execution, error)
```

**Acceptance Criteria**:
- Templates persist across restarts
- Execution history queryable
- Step results accessible
- Cleanup old executions (retention policy)
- Thread-safe file operations

**Estimated Effort**: 4-5 hours

**Priority**: MEDIUM

**Dependencies**: "Implement Workflow Executor"

---

## Task: Add Workflow Execution Progress Streaming

**Description**:
Implement Server-Sent Events (SSE) endpoint for real-time workflow execution progress.

**Endpoint**: `GET /api/v1/workflow/execution/:id/stream`

**Event Types**:
```javascript
// Step started
data: {"event":"step_started","step_id":"create_session","timestamp":"2025-10-27T21:30:00Z"}

// Step progress
data: {"event":"step_progress","step_id":"implement_with_claude","progress":0.5,"message":"Processing..."}

// Step completed
data: {"event":"step_completed","step_id":"create_session","result":{...},"timestamp":"2025-10-27T21:30:05Z"}

// Step failed
data: {"event":"step_failed","step_id":"create_dirs","error":"Directory exists","timestamp":"2025-10-27T21:30:06Z"}

// Workflow completed
data: {"event":"workflow_completed","status":"completed","timestamp":"2025-10-27T21:35:00Z"}
```

**Acceptance Criteria**:
- SSE stream provides real-time updates
- All step lifecycle events emitted
- Connection handles client disconnects
- Multiple clients can subscribe
- Events include timestamps

**Estimated Effort**: 3-4 hours

**Priority**: MEDIUM

**Dependencies**: "Implement Workflow Executor", "Create Workflow API Endpoints"

---

## Task: Create Workflow CLI Command

**Description**:
Add Telegram bot command for executing workflows.

**Command**: `/workflow <template_name> [variables_json]`

**Examples**:
```
/workflow idea-to-implementation {"project_name":"my-scraper","project_description":"News scraper","idea_content":"Build a scraper for..."}

/workflow list

/workflow status exec-123abc
```

**Acceptance Criteria**:
- Can execute workflows from Telegram
- Can list available templates
- Can check execution status
- Progress updates sent to Telegram chat
- Errors reported clearly

**Estimated Effort**: 3-4 hours

**Priority**: LOW

**Dependencies**: "Create Workflow API Endpoints"
