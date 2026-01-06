# AUTO User Guide

AUTO (Agent Unified Terminal Orchestrator) is a Go-based Terminal User Interface (TUI) designed for monitoring and controlling multiple AI agent sessions from a single dashboard.

## Installation

### Prerequisites

- Go 1.21 or higher

### Building from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/localrivet/auto.git
   cd auto
   ```

2. Build the binary:
   ```bash
   go build -o auto ./cmd/auto
   ```

3. (Optional) Install the binary to your path:
   ```bash
   mv auto /usr/local/bin/
   ```

## Configuration

AUTO looks for a configuration file at `~/.config/auto/config.yaml`. If it doesn't exist, it uses default settings.

### Config File Format

The configuration file uses YAML format. Below is the default configuration with explanations:

```yaml
general:
  refresh_interval: 5s       # How often to refresh the UI
  log_level: info            # info, debug, warn, error
  log_file: ""               # Path to log file (empty for none)

providers:
  opencode:
    enabled: true
    storage_path: ~/.local/share/opencode/storage
    watch_interval: 1s       # How often to check for session updates

alerts:
  context_limit_warning: 90  # Alert when context reaches X%
  long_running_threshold: 30m # Alert if agent runs longer than this
  sound_enabled: false
  desktop_notifications: true
  slack_enabled: false
  slack_webhook_url: ""
  discord_enabled: false
  discord_webhook_url: ""

ui:
  show_header: true
  show_footer: true
  agent_list_width: 30       # Width of the left sidebar
  default_grouping: type     # flat, type, project
  show_metrics: true         # Show token usage and cost metrics

theme:
  mode: dark                 # dark, light
  colors:
    primary: "#7C3AED"
    secondary: "#A78BFA"
    accent: "#10B981"
    background: "#1E1E2E"
    foreground: "#CDD6F4"
    border: "#45475A"
    border_focused: "#7C3AED"
    status_running: "#10B981"
    status_idle: "#F59E0B"
    status_error: "#EF4444"
    status_done: "#3B82F6"

storage:
  database_path: ~/.local/share/auto/auto.db
  max_history: 30            # Days of history to keep

metrics:
  token_cost_input: 0.003    # Cost per 1k input tokens ($)
  token_cost_output: 0.015   # Cost per 1k output tokens ($)
```

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down in agent list |
| `k` / `↑` | Move up in agent list |
| `tab` | Cycle focus between panes (List, Viewport, Stats, Alerts) |
| `enter` | Select and view focused agent session |
| `g` | Cycle grouping mode (flat, type, project) |
| `G` | Scroll viewport to bottom |
| `gg` | Scroll viewport to top |

### Agent Actions

| Key | Action |
|-----|--------|
| `i` | Open input bar to send message to active agent |
| `x` | Terminate the selected agent session |
| `space` | Pause or resume the selected agent |
| `r` | Manually refresh all agent statuses |
| `R` | Mark all active alerts as read |

### Views & Filters

| Key | Action |
|-----|--------|
| `/` | Filter agent list by name or ID |
| `:` | Open command palette |
| `s` | Toggle Statistics panel |
| `a` | Toggle Alerts panel |
| `?` | Toggle Help screen |
| `esc` | Clear filter or close overlays |
| `q` | Quit AUTO |

## Basic Usage Walkthrough

1. **Launch AUTO**: Run `./auto` in your terminal.
2. **Explore Agents**: Use `j` and `k` to navigate the agent list on the left.
3. **View a Session**: Press `enter` to view the full output of the selected agent in the center viewport.
4. **Interact**: Press `i` to type a command or input for the active agent, then press `enter` to send it.
5. **Manage Alerts**: When an agent hits an error or context limit, an alert will appear in the Alerts panel. Use `tab` to focus the Alerts panel and review them.
6. **Customize View**: Use `s` and `a` to show or hide panels based on your needs.

## Troubleshooting

### No agents appearing
- Check if your provider is enabled in `config.yaml`.
- Ensure the `storage_path` for the provider (e.g., OpenCode) is correct and contains session data.
- Run with `log_level: debug` and check the log file for errors.

### TUI rendering issues
- Ensure your terminal supports 256 colors or TrueColor.
- Check if your terminal window is too small (AUTO requires at least 80x24 characters).
- If colors look wrong, try switching `theme.mode` in your config.

### "Too many open files" error
- This can happen if AUTO is watching many agent sessions. Increase your system's `ulimit -n` or increase the `watch_interval` in the configuration.
