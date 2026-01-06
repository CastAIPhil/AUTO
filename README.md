# AUTO - Agent Unified Terminal Orchestrator

A terminal user interface for monitoring and controlling dozens of AI agent sessions (opencode and other pluggable agents) from a single unified dashboard.

## Vision

AUTO provides at-a-glance visibility into all active AI agent sessions, enabling operators to:
- Monitor real-time status of dozens of concurrent agent sessions
- Receive alerts when agents error, hit context limits, or complete tasks
- Switch seamlessly between agent sessions
- View aggregated statistics across all agents
- Control agent lifecycle (start, pause, resume, terminate)

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         AUTO TUI                                │
├─────────────┬─────────────────────────────┬────────────────────┤
│  Agent List │      Session Viewport       │   Stats/Alerts     │
│             │                             │                    │
│  ● agent-1  │  [Active session output]    │  Total: 42         │
│  ○ agent-2  │                             │  Active: 38        │
│  ◉ agent-3  │                             │  Errored: 2        │
│  ● agent-4  │                             │  Complete: 2       │
│  ...        │                             │                    │
└─────────────┴─────────────────────────────┴────────────────────┘
```

## Core Features

### Phase 1: Foundation
- [ ] Session discovery and monitoring for opencode
- [ ] Real-time status updates
- [ ] Basic TUI with agent list and session view

### Phase 2: Control
- [ ] Session switching
- [ ] Alert system for errors/completions
- [ ] Input forwarding to active session

### Phase 3: Analytics
- [ ] Statistics dashboard
- [ ] Session history and logs
- [ ] Performance metrics

### Phase 4: Extensibility
- [ ] Plugin system for other agent types
- [ ] Web interface
- [ ] API for external integrations

## Tech Stack

- **Language**: Go
- **TUI Framework**: Bubbletea (Charm.sh ecosystem)
- **Styling**: Lipgloss
- **Components**: Bubbles (tables, lists, viewports)

## Project Structure

```
auto/
├── cmd/
│   └── auto/           # Main entrypoint
├── internal/
│   ├── tui/            # Terminal UI components
│   │   ├── app.go      # Main application model
│   │   ├── views/      # Different screens/views
│   │   └── components/ # Reusable UI components
│   ├── agent/          # Agent abstraction layer
│   │   ├── agent.go    # Agent interface
│   │   ├── registry.go # Agent type registry
│   │   └── providers/  # Concrete agent implementations
│   │       └── opencode/
│   ├── session/        # Session management
│   │   ├── session.go  # Session model
│   │   ├── manager.go  # Session lifecycle
│   │   └── store.go    # Session persistence
│   ├── monitor/        # Status monitoring
│   │   ├── monitor.go  # Monitor coordinator
│   │   ├── watcher.go  # File/process watchers
│   │   └── alerts.go   # Alert generation
│   └── config/         # Configuration
├── pkg/
│   └── api/            # Public API (future web interface)
├── configs/            # Default configurations
└── docs/               # Documentation
```

## Getting Started

```bash
# Build
go build -o auto ./cmd/auto

# Run
./auto

# Run with config
./auto --config ~/.config/auto/config.yaml
```

## License

Apache 2.0
