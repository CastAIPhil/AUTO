# AUTO Architecture

This document describes the internal structure and design of AUTO (Agent Unified Terminal Orchestrator).

## System Overview

AUTO is designed as a centralized monitoring system for AI agent sessions. It follows a provider-based architecture, allowing it to support multiple types of AI agents through a unified interface.

### Component Diagram

```text
┌─────────────────────────────────────────────────────────────┐
│                          AUTO TUI                           │
│     (Bubbletea, Lipgloss, AgentList, Viewport, Stats)       │
└──────────────┬───────────────────────────▲──────────────────┘
               │                           │
               │ (Commands)                │ (Events)
               │                           │
┌──────────────▼───────────────────────────┴──────────────────┐
│                      Session Manager                        │
│   (Coordinates discovery, events, storage, and alerts)      │
└──────────────┬──────────────┬─────────────┬─────────────────┘
               │              │             │
               ▼              ▼             ▼
      ┌────────────────┐ ┌────────────┐ ┌──────────────┐
      │  Alert Manager │ │   Store    │ │   Registry   │
      └──────┬─────────┘ └─────┬──────┘ └─────┬────────┘
             │                 │              │
             ▼                 ▼              ▼
      ┌───────────────┐ ┌────────────┐ ┌──────────────┐
      │ Channels      │ │ SQLite DB  │ │ Providers    │
      │ (Slack, Disc) │ │            │ │ (OpenCode,..)│
      └───────────────┘ └────────────┘ └──────────────┘
```

## Package Structure

- `cmd/auto`: Main entrypoint. Initializes components, loads configuration, and starts the TUI application.
- `internal/agent`: Core abstractions for agents and providers. Defines the `Agent` and `Provider` interfaces and the event system.
- `internal/agent/providers`: Concrete implementations of agent types.
    - `opencode`: Monitors OpenCode sessions by watching the local file system.
- `internal/session`: Orchestration logic. The `Manager` struct coordinates agent discovery, event processing, and lifecycle management.
- `internal/alert`: Multi-channel notification system. Handles desktop, Slack, and Discord alerts.
- `internal/store`: Persistence layer. Uses SQLite to store session history, metrics, and alert logs.
- `internal/tui`: Terminal UI implementation using the Charm.sh ecosystem (Bubbletea, Lipgloss, Bubbles).
- `internal/config`: Configuration management, YAML parsing, and default settings.
- `pkg/api`: Publicly accessible types and future API definitions.

## Key Interfaces

### Agent Interface
Located in `internal/agent/agent.go`, this interface abstracts a single AI agent instance.

```go
type Agent interface {
    ID() string
    Name() string
    Type() string
    Directory() string
    ProjectID() string
    Status() Status
    StartTime() time.Time
    LastActivity() time.Time
    Output() io.Reader
    CurrentTask() string
    Metrics() Metrics
    LastError() error
    SendInput(input string) error
    Terminate() error
    Pause() error
    Resume() error
}
```

### Provider Interface
Defines how AUTO discovers and manages groups of agents.

```go
type Provider interface {
    Name() string
    Type() string
    Discover(ctx context.Context) ([]Agent, error)
    Watch(ctx context.Context) (<-chan Event, error)
    Spawn(ctx context.Context, config SpawnConfig) (Agent, error)
    Get(id string) (Agent, error)
    List() []Agent
    Terminate(id string) error
    SendInput(id string, input string) error
}
```

## Data Flow

1. **Initialization**: On startup, `cmd/auto` loads the config, initializes the `Store`, `AlertManager`, `Registry`, and `SessionManager`.
2. **Discovery**: The `Session Manager` performs an initial discovery via the `Registry`, which queries all registered `Providers`.
3. **Monitoring**: `Providers` (like `opencode`) monitor their respective backends (e.g., file system, API) and emit `agent.Event` objects.
4. **Event Handling**:
    - The `Session Manager` receives events, updates its internal cache, and persists the data to the `Store`.
    - Events are passed to the `Alert Manager` to trigger notifications.
    - The `TUI` receives events via a Go channel and updates its state.
5. **User Interaction**: User input (key presses) in the `TUI` triggers commands that call methods on the `Session Manager`, which then interacts with the `Providers` and `Agents`.

## Extension Points

### New Agent Providers
Support for new agent platforms (e.g., Claude, AutoGPT) can be added by implementing the `Agent` and `Provider` interfaces in `internal/agent/providers`.

### New Alert Channels
Additional notification channels (e.g., Telegram, PagerDuty) can be added by implementing the `alert.Channel` interface and registering it in the `Alert Manager`.

### Custom Themes
The UI appearance is controlled by `internal/tui/components/theme.go`, which is driven by the `theme` section in `config.yaml`.
