# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AUTO (Agent Unified Terminal Orchestrator) is a terminal user interface (TUI) for monitoring and controlling multiple AI agent sessions from a unified dashboard. Built with Go and the Charm.sh Bubbletea ecosystem, it provides real-time visibility into dozens of concurrent agent sessions with alerts, statistics, and session control.

## Build and Development Commands

```bash
# Build the binary
make build           # Builds ./auto executable
go build -o auto ./cmd/auto

# Development
make dev             # Run without building (go run)
make run             # Build then run

# Testing
make test            # Run all tests with race detector
make test-coverage   # Generate coverage report (coverage.html)

# Quality checks
make lint            # Run golangci-lint
make check           # Run both lint and test

# Dependencies
make deps            # Download and tidy modules

# Release
make release         # Build release with goreleaser
make snapshot        # Build snapshot release locally
```

### Running Tests

```bash
# Run all tests
go test -v -race ./...

# Run specific package tests
go test -v ./internal/agent/...
go test -v ./internal/session/...

# Run single test
go test -v -run TestSessionManager ./internal/session/

# Benchmarks
go test -bench=. ./internal/session/
go test -bench=. ./internal/store/
```

## Architecture Overview

AUTO uses a layered architecture with clear separation of concerns:

### Core Layers

1. **TUI Layer** (`internal/tui/`): Bubbletea application with modular components
   - `app.go`: Main application model implementing tea.Model interface
   - `components/`: Reusable UI components (agent list, viewport, stats, alerts, etc.)
   - Uses Elm architecture: Model → Update → View cycle

2. **Session Management** (`internal/session/`): Coordinates agent lifecycle
   - `manager.go`: Session manager that orchestrates all agents
   - Handles discovery, monitoring, filtering, grouping, and statistics
   - Thread-safe with sync.RWMutex for concurrent access
   - Integrates with store for persistence and alert manager for notifications

3. **Agent Abstraction** (`internal/agent/`): Provider-agnostic agent interface
   - `agent.go`: Core Agent interface and types
   - `agent.Registry`: Manages multiple provider types
   - Status types: Pending, Running, Idle, Completed, Errored, ContextLimit, Cancelled
   - `StreamingAgent` interface for real-time streaming support

4. **Provider Layer** (`internal/agent/providers/`): Concrete agent implementations
   - `opencode/`: OpenCode agent provider
     - Discovers sessions from `~/.local/share/opencode/storage`
     - Parses session.json, messages/, and parts/ directories
     - `runner.go`: Executes opencode CLI for spawning new sessions
   - Extensible: new providers implement the Provider interface

5. **Store Layer** (`internal/store/`): SQLite persistence
   - Session history and metrics
   - Thread-safe operations with connection pooling

6. **Alert System** (`internal/alert/`): Multi-channel notifications
   - Desktop notifications, Slack, Discord integration
   - Alert on errors, context limits, completions

### Key Design Patterns

- **Provider Pattern**: Agent providers are pluggable via the Registry
- **Observer Pattern**: Session manager publishes events; TUI subscribes
- **Command Pattern**: TUI components use Bubbletea messages/commands
- **Repository Pattern**: Store abstracts persistence layer

### Data Flow

```
User Input → TUI (app.go) → Manager → Agent → Provider → External System
                ↓              ↓         ↓
              View ←───── Events ←─── Store/Alerts
```

### OpenCode Session Structure

OpenCode stores sessions in `~/.local/share/opencode/storage/<session-id>/`:
- `session.json`: Session metadata (id, project, directory, timestamps)
- `messages/`: Per-message JSON files
- `parts/`: Message parts (text, tool calls, results)

The opencode provider watches this directory and reconstructs agent state from these files.

## Configuration

Config file: `~/.config/auto/config.yaml` (default) or use `--config` flag

Key config sections:
- `providers.opencode.storage_path`: Where to find opencode sessions (default: `~/.local/share/opencode/storage`)
- `providers.opencode.max_age`: Only load sessions newer than this (e.g., `24h`)
- `ui.agent_list_width`: Width of left sidebar
- `ui.default_grouping`: `flat`, `type`, `project`, or `status`
- `alerts`: Configure desktop notifications, Slack, Discord webhooks
- `theme.colors`: Customize all colors (see `configs/default.yaml` for reference)
- `keys`: Rebind all keybindings

## Key Behaviors and Gotchas

### OpenCode Integration

- AUTO discovers opencode sessions by scanning `~/.local/share/opencode/storage`
- Sessions are identified by their directory name (UUID)
- Background agents (created with Task tool) have a `parentID` field
- Agent status is inferred from:
  - Last message role (assistant running → StatusRunning)
  - Presence of error messages → StatusErrored
  - Tool state (`running` → StatusRunning, `error` → StatusErrored)
  - Idle detection via `lastActivity` timestamp

### Streaming vs Non-Streaming

- `StreamingAgent` interface enables real-time output streaming
- OpenCode runner implements streaming via `SendInputAsync`
- TUI viewport displays streaming content as it arrives
- Press `q` during streaming to cancel (doesn't quit app)

### Performance Considerations

- Session discovery can be slow with many sessions; use `max_age` config to limit
- Viewport uses content caching to avoid re-reading files on every render
- Stats panel caches statistics and only recomputes on `statsDirty` flag
- Agent list uses efficient filtering/grouping with minimal allocations

### Thread Safety

- Session manager uses `sync.RWMutex` for all agent access
- Store uses SQLite with `_busy_timeout=5000` for concurrent writes
- Event channels are buffered (100) to prevent blocking

## Task Tracking

This project uses `bd` (Beads) for task tracking. Always use `bd` instead of internal todo tools:

```bash
bd ready                          # List tasks ready to work on
bd create "Task title" -p 0       # Create P0 (highest priority)
bd update <id> --status in_progress
bd update <id> --status done
bd show <id>
bd list
bd dep add <child> <parent>       # Add dependency
bd sync                           # Sync with git
```

Before starting multi-step work, create tasks with `bd create`. Update status as you progress.

## Session Completion Workflow

When ending a work session, complete ALL these steps:

1. Create issues for remaining work with `bd create`
2. Run quality gates: `make check` (lint + test)
3. Update issue status: `bd update <id> --status done`
4. Push to remote (MANDATORY):
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # Must show "up to date with origin"
   ```
5. Verify all changes are pushed

Work is NOT complete until `git push` succeeds. Never stop before pushing.
