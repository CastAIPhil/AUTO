# Clarification Questions for AUTO

Before proceeding with implementation, I need clarification on several design decisions.

## 1. Session Control Scope

**Question**: What level of control do you want over agent sessions?

**Options**:
1. **Read-only monitoring** - Only observe agents, no control
2. **Basic control** - Pause/resume, terminate existing sessions
3. **Full control** - Spawn new agents, send input, terminate, restart

**Current assumption**: Option 2 (Basic control) for MVP

---

## 2. Agent Discovery Method

**Question**: How should AUTO discover running agents?

**Options**:
1. **Automatic discovery** - Scan known storage locations (opencode's ~/.local/share/opencode)
2. **Manual registration** - User explicitly adds agent paths/connections
3. **Hybrid** - Auto-discover known types + manual for custom

**Current assumption**: Option 3 (Hybrid)

---

## 3. Multi-Machine Support

**Question**: Should AUTO support monitoring agents on remote machines?

**Options**:
1. **Local only** - Single machine monitoring
2. **SSH-based** - Monitor remote machines via SSH
3. **Agent-based** - Deploy AUTO agents on remote machines that report to central AUTO

**Current assumption**: Option 1 for MVP, designed for future Option 2

---

## 4. Notification Channels

**Question**: Besides the TUI, where should alerts go?

**Options**:
1. **TUI only** - Alerts visible only in the terminal
2. **Desktop notifications** - OS-level notifications (macOS/Linux)
3. **External integrations** - Slack, Discord, webhooks
4. **All of the above** - Configurable per-alert-level

**Current assumption**: Option 2 for MVP

---

## 5. Session History Persistence

**Question**: How much history should AUTO maintain?

**Options**:
1. **Session-only** - Data lost when AUTO exits
2. **Light persistence** - Recent session list, alert history (SQLite)
3. **Full persistence** - All session outputs, metrics, searchable history

**Current assumption**: Option 2

---

## 6. Input Forwarding

**Question**: Should users be able to send input to agents through AUTO?

**Context**: Some agents (like opencode) allow user input during execution.

**Options**:
1. **No forwarding** - View-only mode
2. **Basic input** - Send text to focused agent
3. **Full terminal** - Full PTY passthrough (like tmux)

**Current assumption**: Option 1 for MVP, designed for Option 2

---

## 7. Agent Grouping

**Question**: How should agents be organized in the UI?

**Options**:
1. **Flat list** - All agents in single list
2. **By type** - Grouped by provider (opencode, claude, etc.)
3. **By project** - Grouped by working directory
4. **Custom groups** - User-defined tags/groups

**Current assumption**: Option 2 with Option 3 as alternative view

---

## 8. Metrics & Statistics

**Question**: What metrics are most important to display?

**Options** (select all that apply):
- [ ] Token usage (input/output)
- [ ] Cost estimation
- [ ] Session duration
- [ ] Task completion rate
- [ ] Error rate
- [ ] Active time vs idle time
- [ ] Tool call statistics
- [ ] Context window utilization

**Current assumption**: Session duration, error rate, context utilization

---

## 9. Theme & Appearance

**Question**: Should themes be customizable?

**Options**:
1. **Fixed theme** - One well-designed theme
2. **Light/Dark** - Two preset themes
3. **Configurable** - User can customize all colors
4. **Theme files** - Support external theme files

**Current assumption**: Option 2 for MVP

---

## 10. Priority: First Features to Ship

**Question**: What's the absolute minimum for a useful v0.1?

**My proposed MVP**:
1. Discover and list opencode sessions
2. Show real-time status (running/complete/error)
3. View session output (read-only)
4. Alert on errors and completion
5. Basic keyboard navigation

**What would you add or remove?**

---

## 11. Naming & Branding

**Question**: Is "AUTO" the final name?

**Alternatives considered**:
- SWARM (Session Watch And Resource Monitor)
- HIVE (Hierarchical Interactive View for Entities)
- LENS (Live Entity Notification System)
- GRID (General Resource Interface Dashboard)

**Current assumption**: AUTO is final

---

## 12. Installation & Distribution

**Question**: How should users install AUTO?

**Options**:
1. **Go install** - `go install github.com/user/auto@latest`
2. **Homebrew** - `brew install auto`
3. **Binary releases** - GitHub releases with pre-built binaries
4. **All of the above**

**Current assumption**: Option 3 for MVP, Option 4 long-term

---

## Please Respond

For each question, please indicate:
1. Your preferred option
2. Any modifications to the options
3. Any additional considerations I haven't mentioned

This will help ensure we build exactly what you need.
