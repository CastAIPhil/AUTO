# AUTO API Reference

This document provides a reference for the AUTO (Agent Unified Terminal Orchestrator) API, including both the internal Go interfaces and the external HTTP API.

## Go API

### Agent Interface
Defined in `internal/agent/agent.go`. Any new agent type must implement this interface.

```go
type Agent interface {
    // Identity
    ID() string        // Unique identifier for the session
    Name() string      // Human-readable name
    Type() string      // Agent type (e.g., "opencode")

    // Location
    Directory() string // Working directory of the agent
    ProjectID() string // Project identifier if applicable

    // Status
    Status() Status           // Current status (Running, Idle, etc.)
    StartTime() time.Time     // When the session started
    LastActivity() time.Time  // Last time the agent did something

    // Session data
    Output() io.Reader   // Stream of output from the agent
    CurrentTask() string // Brief description of current work
    Metrics() Metrics    // Token usage, costs, etc.
    LastError() error    // Most recent error if any

    // Control
    SendInput(input string) error // Send text input to the agent
    Terminate() error             // Kill the agent session
    Pause() error                 // Temporarily stop the agent
    Resume() error                // Resume a paused agent
}
```

### Provider Interface
Defined in `internal/agent/agent.go`. Handles discovery and lifecycle of agents for a specific platform.

```go
type Provider interface {
    Name() string // Name of the provider
    Type() string // Type of agent it provides

    // Discovery
    Discover(ctx context.Context) ([]Agent, error)
    Watch(ctx context.Context) (<-chan Event, error)

    // Control
    Spawn(ctx context.Context, config SpawnConfig) (Agent, error)
    Get(id string) (Agent, error)
    List() []Agent
    Terminate(id string) error
    SendInput(id string, input string) error
}
```

### Data Structures

#### Agent Status
```go
const (
    StatusPending Status = iota
    StatusRunning
    StatusIdle
    StatusCompleted
    StatusErrored
    StatusContextLimit
    StatusCancelled
)
```

#### Metrics
```go
type Metrics struct {
    TokensIn           int64
    TokensOut          int64
    EstimatedCost      float64
    Duration           time.Duration
    ActiveTime         time.Duration
    IdleTime           time.Duration
    ToolCalls          int
    ErrorCount         int
    TasksCompleted     int
    TasksFailed        int
    ContextUtilization float64 // 0.0 to 1.0
}
```

## HTTP API

If enabled in configuration, AUTO provides an HTTP API for remote monitoring.

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Check server health |
| `GET` | `/api/agents` | List all active agents |
| `GET` | `/api/agents/{id}` | Get details for a specific agent |
| `POST` | `/api/agents/{id}/terminate` | Terminate an agent session |
| `GET` | `/api/stats` | Get aggregate statistics |

### Examples

#### Get Agent Details
**Request:**
```bash
curl http://localhost:8080/api/agents/ses_abc123
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "ses_abc123",
    "name": "refactor-login",
    "type": "opencode",
    "status": "running",
    "directory": "/Users/dev/project",
    "project_id": "proj_xyz",
    "current_task": "Implementing JWT validation",
    "start_time": "2024-01-06T10:00:00Z",
    "last_activity": "2024-01-06T10:05:00Z",
    "tokens_in": 12500,
    "tokens_out": 4500
  }
}
```

#### Terminate Agent
**Request:**
```bash
curl -X POST http://localhost:8080/api/agents/ses_abc123/terminate
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "terminated"
  }
}
```

## Usage Example (Go)

Integrating the `Session Manager` into your own Go application:

```go
import (
    "context"
    "github.com/localrivet/auto/internal/session"
    "github.com/localrivet/auto/internal/config"
    "github.com/localrivet/auto/internal/agent/providers/opencode"
)

func main() {
    cfg, _ := config.Load("config.yaml")
    
    // Setup registry and providers
    registry := agent.NewRegistry()
    ocProvider := opencode.NewProvider(cfg.Providers.OpenCode)
    registry.Register(ocProvider)
    
    // Initialize manager
    manager := session.NewManager(cfg, nil, registry, nil)
    
    // Start discovery and event loop
    ctx := context.Background()
    manager.Start(ctx)
    
    // List agents
    agents := manager.List()
    for _, a := range agents {
        fmt.Printf("Found agent: %s (%s)\n", a.Name(), a.Status())
    }
}
```
