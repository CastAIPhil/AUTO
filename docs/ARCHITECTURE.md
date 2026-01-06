# AUTO Architecture Document

## Overview

AUTO (Agent Unified Terminal Orchestrator) is a terminal UI for monitoring and controlling dozens of AI agent sessions from a single dashboard.

## Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Excellent concurrency, fast compilation, single binary deployment |
| TUI Framework | Bubbletea | Elm architecture, active community, used by major tools |
| Styling | Lipgloss | CSS-like syntax, flexible layouts |
| Components | Bubbles | Pre-built tables, lists, viewports, spinners |
| First Agent Type | OpenCode | File-based session storage, well-documented structure |

## Core Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          AUTO Application                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │   TUI App   │  │   Session   │  │   Status    │  │   Alert    │ │
│  │   (Model)   │  │   Manager   │  │   Monitor   │  │   Manager  │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬──────┘ │
│         │                │                │                │        │
│         └────────────────┴────────────────┴────────────────┘        │
│                                    │                                 │
│                         ┌──────────┴──────────┐                     │
│                         │   Agent Registry    │                     │
│                         │    (Interface)      │                     │
│                         └──────────┬──────────┘                     │
│                                    │                                 │
│              ┌─────────────────────┼─────────────────────┐          │
│              │                     │                     │          │
│        ┌─────┴─────┐        ┌──────┴──────┐       ┌──────┴──────┐   │
│        │ OpenCode  │        │   Claude    │       │   Future    │   │
│        │ Provider  │        │  Provider   │       │  Providers  │   │
│        └───────────┘        └─────────────┘       └─────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Agent Interface (Plugin System)

```go
// Agent represents a single AI agent instance
type Agent interface {
    // Identity
    ID() string
    Name() string
    Type() string  // "opencode", "claude", etc.
    
    // Status
    Status() AgentStatus
    StartTime() time.Time
    LastActivity() time.Time
    
    // Session data
    Output() io.Reader          // Stream of output
    CurrentTask() string        // What the agent is working on
    Metrics() AgentMetrics      // Token usage, duration, etc.
}

// AgentProvider discovers and manages agents of a specific type
type AgentProvider interface {
    // Discovery
    Name() string
    Discover() ([]Agent, error)
    Watch(ctx context.Context) (<-chan AgentEvent, error)
    
    // Control
    Spawn(config AgentConfig) (Agent, error)
    Terminate(agentID string) error
    SendInput(agentID string, input string) error
}

// AgentStatus represents the current state of an agent
type AgentStatus int

const (
    StatusPending AgentStatus = iota
    StatusRunning
    StatusIdle
    StatusCompleted
    StatusErrored
    StatusContextLimit
    StatusCancelled
)
```

## OpenCode Provider Implementation

### Session Discovery

OpenCode stores sessions in `~/.local/share/opencode/storage/`:

```
storage/
├── session/
│   ├── global/           # Global sessions
│   │   └── ses_xxx.json  # Session metadata
│   └── <project-hash>/   # Project-specific sessions
├── message/
│   └── <session-id>/     # Messages per session
└── part/
    └── <message-id>/     # Parts per message
```

### Monitoring Strategy

1. **File System Watching**: Use `fsnotify` to watch session directories
2. **Polling Fallback**: Poll every 5 seconds as backup
3. **Status Detection**:
   - Active: `time.updated` < 5 minutes ago
   - Idle: `time.updated` 5-30 minutes ago
   - Complete: No updates > 30 minutes
   - Error: Check message parts for `error` field

## TUI Layout

```
┌─────────────────────────────────────────────────────────────────────┐
│ AUTO v0.1.0    Agents: 42 (38●, 2◉, 2○)    CPU: 12%    14:32:45   │
├──────────────────┬──────────────────────────────────────────────────┤
│                  │                                                  │
│  Agent List      │  Session Viewport                                │
│  ─────────────   │  ─────────────────                               │
│  ● agent-1 [5m]  │  > Creating feature branch...                    │
│  ○ agent-2 [✓]   │  > Running tests...                              │
│  ◉ agent-3 [ERR] │  > Test failed: expected 3, got 2                │
│  ● agent-4 [2m]  │                                                  │
│  ● agent-5 [8m]  │  [Agent: opencode-1]                             │
│  ◌ agent-6 [--]  │  [Status: Running]                               │
│                  │  [Task: Implementing login feature]              │
│                  │  [Duration: 5m 32s]                              │
│                  │                                                  │
├──────────────────┴──────────────────────────────────────────────────┤
│ [!] agent-3: Context limit reached (200k tokens)                   │
├─────────────────────────────────────────────────────────────────────┤
│ j/k: navigate  Enter: focus  Space: pause  d: details  ?: help     │
└─────────────────────────────────────────────────────────────────────┘
```

### Status Icons

| Icon | Status | Description |
|------|--------|-------------|
| ● | Running | Agent actively processing |
| ◌ | Idle | Agent waiting for input |
| ○ | Complete | Agent finished task |
| ◉ | Error | Agent encountered error |
| ⚠ | Warning | Context limit approaching |

## Keyboard Navigation

### Global Keys
| Key | Action |
|-----|--------|
| `q` | Quit application |
| `?` | Show help |
| `/` | Search/filter agents |
| `:` | Command mode |
| `Esc` | Cancel/back |
| `Tab` | Switch pane focus |

### Agent List Keys
| Key | Action |
|-----|--------|
| `j/k` | Navigate up/down |
| `Enter` | Focus agent in viewport |
| `Space` | Pause/resume agent |
| `d` | Show agent details |
| `l` | Show full logs |
| `x` | Terminate agent |
| `r` | Restart agent |

### Viewport Keys
| Key | Action |
|-----|--------|
| `j/k` | Scroll up/down |
| `g/G` | Go to top/bottom |
| `Ctrl+U/D` | Page up/down |
| `y` | Copy output |

## Event System

```go
type AgentEvent struct {
    Type      EventType
    AgentID   string
    Timestamp time.Time
    Data      interface{}
}

type EventType int

const (
    EventAgentSpawned EventType = iota
    EventAgentUpdated
    EventAgentCompleted
    EventAgentErrored
    EventAgentContextLimit
    EventAgentTerminated
)
```

## Alert System

### Alert Levels
| Level | Visual | Behavior |
|-------|--------|----------|
| Info | Blue | Flash in status bar |
| Warning | Yellow | Flash + add to alerts pane |
| Error | Red | Flash + alert pane + sound (optional) |
| Critical | Red blink | All above + stays visible |

### Alert Triggers
- Agent errors
- Context limit reached (90%, 100%)
- Task completion
- Long-running agents (configurable threshold)
- Connection lost to agent

## Configuration

```yaml
# ~/.config/auto/config.yaml

# General settings
refresh_interval: 5s
theme: dark

# Agent providers
providers:
  opencode:
    enabled: true
    storage_path: ~/.local/share/opencode/storage
    watch_interval: 1s
  
  # Future providers
  claude:
    enabled: false
    
# Alerts
alerts:
  context_limit_warning: 90  # percent
  long_running_threshold: 30m
  sound_enabled: false
  
# UI
ui:
  show_header: true
  show_footer: true
  agent_list_width: 25
  
# Keybindings (override defaults)
keybindings:
  quit: q
  help: "?"
  search: /
```

## Project Structure

```
auto/
├── cmd/
│   └── auto/
│       └── main.go           # Entry point
├── internal/
│   ├── tui/
│   │   ├── app.go            # Main Bubbletea model
│   │   ├── keys.go           # Key bindings
│   │   ├── styles.go         # Lipgloss styles
│   │   ├── views/
│   │   │   ├── dashboard.go  # Main dashboard view
│   │   │   ├── details.go    # Agent detail view
│   │   │   ├── logs.go       # Full logs view
│   │   │   ├── stats.go      # Statistics view
│   │   │   └── help.go       # Help screen
│   │   └── components/
│   │       ├── agentlist.go  # Agent list sidebar
│   │       ├── viewport.go   # Session output viewport
│   │       ├── statusbar.go  # Status bar
│   │       ├── alerts.go     # Alerts panel
│   │       └── header.go     # Header component
│   ├── agent/
│   │   ├── agent.go          # Agent interface
│   │   ├── status.go         # Status types
│   │   ├── registry.go       # Provider registry
│   │   └── providers/
│   │       ├── opencode/
│   │       │   ├── provider.go
│   │       │   ├── session.go
│   │       │   └── watcher.go
│   │       └── claude/       # Future
│   ├── session/
│   │   ├── session.go        # Session model
│   │   ├── manager.go        # Session lifecycle
│   │   └── store.go          # Session persistence
│   ├── monitor/
│   │   ├── monitor.go        # Status coordinator
│   │   ├── watcher.go        # File system watcher
│   │   └── aggregator.go     # Metrics aggregation
│   ├── alert/
│   │   ├── alert.go          # Alert types
│   │   ├── manager.go        # Alert routing
│   │   └── rules.go          # Alert rules
│   └── config/
│       ├── config.go         # Config struct
│       └── loader.go         # Config loading
├── pkg/
│   └── api/                  # Future: HTTP/WS API
├── configs/
│   └── default.yaml          # Default configuration
├── docs/
│   ├── ARCHITECTURE.md       # This document
│   └── USER_GUIDE.md         # User documentation
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Future Extensibility

### Web Interface (Phase 4)
- HTTP API for remote monitoring
- WebSocket for real-time updates
- React/Vue frontend (separate repo)

### Additional Providers
- Claude CLI
- Cursor
- Aider
- Custom agents via plugin protocol

### Advanced Features
- Agent orchestration (spawn agents from AUTO)
- Task queuing
- Multi-machine monitoring
- Metrics export (Prometheus)
