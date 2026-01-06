// Package opencode provides an agent provider for opencode sessions using the CLI
package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/CastAIPhil/AUTO/internal/agent"
)

// CLISession represents the opencode session list output format
type CLISession struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Updated   int64  `json:"updated"` // Unix timestamp in milliseconds
	Created   int64  `json:"created"` // Unix timestamp in milliseconds
	ProjectID string `json:"projectId"`
	Directory string `json:"directory"`
}

// OpenCodeAgent implements the Agent interface for opencode sessions
type OpenCodeAgent struct {
	id           string
	name         string
	directory    string
	projectID    string
	parentID     string
	status       agent.Status
	startTime    time.Time
	lastActivity time.Time
	currentTask  string
	metrics      agent.Metrics
	lastError    error
	output       *bytes.Buffer
	mu           sync.RWMutex
}

// NewOpenCodeAgentFromCLI creates a new OpenCodeAgent from CLI session data
func NewOpenCodeAgentFromCLI(session CLISession) *OpenCodeAgent {
	createdAt := time.UnixMilli(session.Created)
	updatedAt := time.UnixMilli(session.Updated)

	a := &OpenCodeAgent{
		id:           session.ID,
		name:         session.Title,
		directory:    session.Directory,
		projectID:    session.ProjectID,
		status:       agent.StatusIdle,
		startTime:    createdAt,
		lastActivity: updatedAt,
		output:       bytes.NewBuffer(nil),
	}

	a.determineStatus()
	return a
}

// determineStatus determines status based on last activity time
func (a *OpenCodeAgent) determineStatus() {
	timeSinceUpdate := time.Since(a.lastActivity)

	if timeSinceUpdate < 60*time.Second {
		a.status = agent.StatusRunning
	} else if timeSinceUpdate < 30*time.Minute {
		a.status = agent.StatusIdle
	} else {
		a.status = agent.StatusCompleted
	}
}

// Update updates the agent from CLI session data
func (a *OpenCodeAgent) Update(session CLISession) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.name = session.Title
	a.lastActivity = time.UnixMilli(session.Updated)
	a.determineStatus()
}

// ID returns the agent's unique identifier
func (a *OpenCodeAgent) ID() string {
	return a.id
}

// Name returns the agent's display name
func (a *OpenCodeAgent) Name() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.name == "" {
		if len(a.id) > 8 {
			return a.id[:8]
		}
		return a.id
	}
	return a.name
}

// Type returns the agent type
func (a *OpenCodeAgent) Type() string {
	return "opencode"
}

// Directory returns the working directory
func (a *OpenCodeAgent) Directory() string {
	return a.directory
}

// ProjectID returns the project identifier
func (a *OpenCodeAgent) ProjectID() string {
	return a.projectID
}

// ParentID returns the parent session ID (empty for now - CLI doesn't expose this)
func (a *OpenCodeAgent) ParentID() string {
	return a.parentID
}

// IsBackground returns true if this is a background/child agent
func (a *OpenCodeAgent) IsBackground() bool {
	return a.parentID != ""
}

// Status returns the current status
func (a *OpenCodeAgent) Status() agent.Status {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

// StartTime returns when the agent started
func (a *OpenCodeAgent) StartTime() time.Time {
	return a.startTime
}

// LastActivity returns the last activity time
func (a *OpenCodeAgent) LastActivity() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastActivity
}

// Output returns the output stream
func (a *OpenCodeAgent) Output() io.Reader {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return bytes.NewReader(a.output.Bytes())
}

// CurrentTask returns the current task description
func (a *OpenCodeAgent) CurrentTask() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentTask
}

// Metrics returns the agent's metrics
func (a *OpenCodeAgent) Metrics() agent.Metrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.metrics
}

// LastError returns the last error
func (a *OpenCodeAgent) LastError() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastError
}

// SendInput sends input to the agent via CLI
func (a *OpenCodeAgent) SendInput(input string) error {
	if input == "" {
		return fmt.Errorf("empty input")
	}

	a.mu.RLock()
	sessionID := a.id
	workDir := a.directory
	a.mu.RUnlock()

	cmd := exec.Command("opencode", "run", "-s", sessionID, input)
	cmd.Dir = workDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("opencode run failed: %s", stderr.String())
		}
		return fmt.Errorf("opencode run failed: %w", err)
	}

	return nil
}

// Terminate terminates the agent
func (a *OpenCodeAgent) Terminate() error {
	a.mu.Lock()
	a.status = agent.StatusCancelled
	a.mu.Unlock()
	return nil
}

// Pause pauses the agent (not supported)
func (a *OpenCodeAgent) Pause() error {
	return fmt.Errorf("pause not supported for opencode sessions")
}

// Resume resumes the agent (not supported)
func (a *OpenCodeAgent) Resume() error {
	return fmt.Errorf("resume not supported for opencode sessions")
}

// Refresh refreshes the agent state (no-op for CLI-based approach, state comes from CLI)
func (a *OpenCodeAgent) Refresh() error {
	return nil
}

// LoadFullHistory loads full history (no-op for now - can be implemented with opencode export)
func (a *OpenCodeAgent) LoadFullHistory() {
	// Future: could call `opencode export <sessionID>` to get full history
}

// Provider implements the agent.Provider interface using opencode CLI
type Provider struct {
	watchInterval time.Duration
	maxAge        time.Duration
	maxSessions   int
	agents        map[string]*OpenCodeAgent
	mu            sync.RWMutex
}

// NewProvider creates a new Provider
func NewProvider(storagePath string, watchInterval time.Duration, maxAge time.Duration) *Provider {
	return &Provider{
		watchInterval: watchInterval,
		maxAge:        maxAge,
		maxSessions:   100, // Default to 100 sessions
		agents:        make(map[string]*OpenCodeAgent),
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "OpenCode"
}

// Type returns the provider type
func (p *Provider) Type() string {
	return "opencode"
}

// fetchSessions calls opencode CLI to get sessions
func (p *Provider) fetchSessions(ctx context.Context) ([]CLISession, error) {
	cmd := exec.CommandContext(ctx, "opencode", "session", "list", "--format", "json", "-n", fmt.Sprintf("%d", p.maxSessions))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("opencode session list failed: %s", stderr.String())
		}
		return nil, fmt.Errorf("opencode session list failed: %w", err)
	}

	var sessions []CLISession
	if err := json.Unmarshal(stdout.Bytes(), &sessions); err != nil {
		return nil, fmt.Errorf("failed to parse session list: %w", err)
	}

	return sessions, nil
}

// Discover discovers opencode sessions using CLI
func (p *Provider) Discover(ctx context.Context) ([]agent.Agent, error) {
	sessions, err := p.fetchSessions(ctx)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	cutoff := time.Now().Add(-p.maxAge)
	var agents []agent.Agent

	for _, session := range sessions {
		updatedAt := time.UnixMilli(session.Updated)

		// Filter by maxAge
		if p.maxAge > 0 && updatedAt.Before(cutoff) {
			continue
		}

		// Update existing or create new
		if existing, ok := p.agents[session.ID]; ok {
			existing.Update(session)
			agents = append(agents, existing)
		} else {
			a := NewOpenCodeAgentFromCLI(session)
			p.agents[session.ID] = a
			agents = append(agents, a)
		}
	}

	return agents, nil
}

// Watch watches for changes in opencode sessions using polling
func (p *Provider) Watch(ctx context.Context) (<-chan agent.Event, error) {
	events := make(chan agent.Event, 100)

	go func() {
		defer close(events)

		ticker := time.NewTicker(p.watchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				p.pollForChanges(ctx, events)
			}
		}
	}()

	return events, nil
}

// pollForChanges polls CLI and emits events for changes
func (p *Provider) pollForChanges(ctx context.Context, events chan<- agent.Event) {
	sessions, err := p.fetchSessions(ctx)
	if err != nil {
		return // Silently ignore errors during polling
	}

	cutoff := time.Now().Add(-p.maxAge)
	seenIDs := make(map[string]bool)

	p.mu.Lock()
	for _, session := range sessions {
		updatedAt := time.UnixMilli(session.Updated)

		// Filter by maxAge
		if p.maxAge > 0 && updatedAt.Before(cutoff) {
			continue
		}

		seenIDs[session.ID] = true

		if existing, ok := p.agents[session.ID]; ok {
			// Check for updates
			oldStatus := existing.Status()
			oldActivity := existing.LastActivity()
			existing.Update(session)
			newStatus := existing.Status()
			newActivity := existing.LastActivity()

			// Emit event if something changed
			if oldStatus != newStatus || !oldActivity.Equal(newActivity) {
				eventType := agent.EventAgentUpdated
				switch {
				case oldStatus != agent.StatusRunning && newStatus == agent.StatusRunning:
					eventType = agent.EventAgentStarted
				case oldStatus != agent.StatusCompleted && newStatus == agent.StatusCompleted:
					eventType = agent.EventAgentCompleted
				case oldStatus != agent.StatusErrored && newStatus == agent.StatusErrored:
					eventType = agent.EventAgentErrored
				}

				events <- agent.Event{
					Type:      eventType,
					AgentID:   existing.ID(),
					Agent:     existing,
					Timestamp: time.Now(),
				}
			}
		} else {
			// New session discovered
			a := NewOpenCodeAgentFromCLI(session)
			p.agents[session.ID] = a

			events <- agent.Event{
				Type:      agent.EventAgentDiscovered,
				AgentID:   a.ID(),
				Agent:     a,
				Timestamp: time.Now(),
			}
		}
	}
	p.mu.Unlock()

	// Note: We don't remove agents that are no longer in the list
	// They may just be older than maxAge but still valid for viewing
}

// Spawn spawns a new opencode session
func (p *Provider) Spawn(ctx context.Context, config agent.SpawnConfig) (agent.Agent, error) {
	cmd := exec.CommandContext(ctx, "opencode")
	cmd.Dir = config.Directory

	cmd.Env = os.Environ()
	for k, v := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	// Wait for session to be created
	time.Sleep(500 * time.Millisecond)

	// Re-discover to find the new session
	agents, err := p.Discover(ctx)
	if err != nil {
		return nil, err
	}

	// Return the most recently created agent
	var newest agent.Agent
	var newestTime time.Time
	for _, a := range agents {
		if a.StartTime().After(newestTime) {
			newestTime = a.StartTime()
			newest = a
		}
	}

	if newest == nil {
		return nil, fmt.Errorf("failed to find newly spawned session")
	}

	return newest, nil
}

// Get returns an agent by ID
func (p *Provider) Get(id string) (agent.Agent, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if a, ok := p.agents[id]; ok {
		return a, nil
	}
	return nil, fmt.Errorf("agent not found: %s", id)
}

// List returns all agents
func (p *Provider) List() []agent.Agent {
	p.mu.RLock()
	defer p.mu.RUnlock()

	agents := make([]agent.Agent, 0, len(p.agents))
	for _, a := range p.agents {
		agents = append(agents, a)
	}
	return agents
}

// Terminate terminates an agent
func (p *Provider) Terminate(id string) error {
	p.mu.RLock()
	a, ok := p.agents[id]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	return a.Terminate()
}

// SendInput sends input to an agent
func (p *Provider) SendInput(id string, input string) error {
	p.mu.RLock()
	a, ok := p.agents[id]
	p.mu.RUnlock()

	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	return a.SendInput(input)
}

// ListPrimary returns only primary agents (agents without a parent)
func (p *Provider) ListPrimary() []agent.Agent {
	p.mu.RLock()
	defer p.mu.RUnlock()

	agents := make([]agent.Agent, 0)
	for _, a := range p.agents {
		if !a.IsBackground() {
			agents = append(agents, a)
		}
	}
	return agents
}

// GetChildren returns all child agents for a given parent ID
func (p *Provider) GetChildren(parentID string) []agent.Agent {
	p.mu.RLock()
	defer p.mu.RUnlock()

	children := make([]agent.Agent, 0)
	for _, a := range p.agents {
		if a.ParentID() == parentID {
			children = append(children, a)
		}
	}
	return children
}

// ChildCount returns the number of child agents for a given parent ID
func (p *Provider) ChildCount(parentID string) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, a := range p.agents {
		if a.ParentID() == parentID {
			count++
		}
	}
	return count
}
