// Package opencode provides an agent provider for opencode sessions
package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/CastAIPhil/AUTO/internal/agent"
)

// SessionData represents the opencode session.json structure
type SessionData struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	ProjectID string `json:"projectID"`
	Directory string `json:"directory"`
	ParentID  string `json:"parentID,omitempty"`
	Title     string `json:"title"`
	Time      struct {
		Created int64 `json:"created"` // Unix timestamp in milliseconds
		Updated int64 `json:"updated"` // Unix timestamp in milliseconds
	} `json:"time"`
	Summary struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		Files     int `json:"files"`
	} `json:"summary"`
}

// MessageData represents a message in the session
type MessageData struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
	Time      struct {
		Created int64 `json:"created"` // Unix timestamp in milliseconds
	} `json:"time"`
	Summary struct {
		Title string `json:"title"`
	} `json:"summary"`
	Agent string `json:"agent,omitempty"`
	Model struct {
		ProviderID string `json:"providerID"`
		ModelID    string `json:"modelID"`
	} `json:"model,omitempty"`
}

// PartData represents a message part (content, tool calls, etc.)
type PartData struct {
	ID        string `json:"id"`
	MessageID string `json:"messageID"`
	SessionID string `json:"sessionID"`
	Type      string `json:"type"` // "text", "tool-invocation", etc.
	Time      struct {
		Created int64 `json:"created"`
	} `json:"time"`
	Text       string `json:"text,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
	State      string `json:"state,omitempty"` // "running", "success", "error"
}

// OpenCodeAgent implements the Agent interface for opencode sessions
type OpenCodeAgent struct {
	id           string
	name         string
	directory    string
	projectID    string
	parentID     string
	storagePath  string
	sessionFile  string
	status       agent.Status
	startTime    time.Time
	lastActivity time.Time
	currentTask  string
	metrics      agent.Metrics
	lastError    error
	output       *bytes.Buffer
	mu           sync.RWMutex
	sessionData  *SessionData
	messages     []MessageData
	loaded       bool
}

// NewOpenCodeAgent creates a new OpenCodeAgent from a session JSON file
func NewOpenCodeAgent(storagePath, sessionFilePath string) (*OpenCodeAgent, error) {
	data, err := os.ReadFile(sessionFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to parse session file: %w", err)
	}

	// Convert millisecond timestamps to time.Time
	createdAt := time.UnixMilli(session.Time.Created)
	updatedAt := time.UnixMilli(session.Time.Updated)

	a := &OpenCodeAgent{
		id:           session.ID,
		name:         session.Title,
		directory:    session.Directory,
		projectID:    session.ProjectID,
		parentID:     session.ParentID,
		storagePath:  storagePath,
		sessionFile:  sessionFilePath,
		status:       agent.StatusIdle,
		startTime:    createdAt,
		lastActivity: updatedAt,
		output:       bytes.NewBuffer(nil),
		sessionData:  &session,
		loaded:       false,
	}

	a.determineStatusFast()

	return a, nil
}

// loadMessages loads messages from the messages directory
func (a *OpenCodeAgent) loadMessages() error {
	// Messages are stored in storage/message/<session-id>/
	messagesPath := filepath.Join(a.storagePath, "message", a.id)
	entries, err := os.ReadDir(messagesPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	a.messages = nil
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(messagesPath, entry.Name()))
		if err != nil {
			continue
		}

		var msg MessageData
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		a.messages = append(a.messages, msg)
	}

	// Sort messages by creation time
	sort.Slice(a.messages, func(i, j int) bool {
		return a.messages[i].Time.Created < a.messages[j].Time.Created
	})

	// Load parts to get actual content and token metrics
	a.loadParts()

	return nil
}

func (a *OpenCodeAgent) loadParts() {
	type partWithTime struct {
		part PartData
		time int64
	}
	var allParts []partWithTime

	for _, msg := range a.messages {
		partsPath := filepath.Join(a.storagePath, "part", msg.ID)
		entries, err := os.ReadDir(partsPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(partsPath, entry.Name()))
			if err != nil {
				continue
			}

			var part PartData
			if err := json.Unmarshal(data, &part); err != nil {
				continue
			}

			allParts = append(allParts, partWithTime{part: part, time: part.Time.Created})
		}
	}

	sort.Slice(allParts, func(i, j int) bool {
		return allParts[i].time < allParts[j].time
	})

	for _, p := range allParts {
		if p.part.Type == "text" && p.part.Text != "" {
			a.output.WriteString(p.part.Text)
			a.output.WriteString("\n")
		}
	}
}

func (a *OpenCodeAgent) determineStatusFast() {
	timeSinceUpdate := time.Since(a.lastActivity)

	if timeSinceUpdate < 60*time.Second {
		a.status = agent.StatusRunning
	} else if timeSinceUpdate < 30*time.Minute {
		a.status = agent.StatusIdle
	} else {
		a.status = agent.StatusCompleted
	}
}

func (a *OpenCodeAgent) LoadFullHistory() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.loaded {
		return
	}

	a.loadMessages()
	a.determineStatus()
	a.extractCurrentTask()
	a.loaded = true
}

func (a *OpenCodeAgent) determineStatus() {
	if len(a.messages) == 0 {
		a.status = agent.StatusPending
		return
	}

	lastMsg := a.messages[len(a.messages)-1]
	lastMsgTime := time.UnixMilli(lastMsg.Time.Created)
	timeSinceLastActivity := time.Since(lastMsgTime)

	for _, msg := range a.messages {
		partsPath := filepath.Join(a.storagePath, "part", msg.ID)
		entries, _ := os.ReadDir(partsPath)
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(partsPath, entry.Name()))
			if err != nil {
				continue
			}
			var part PartData
			if err := json.Unmarshal(data, &part); err != nil {
				continue
			}
			if part.State == "running" {
				a.status = agent.StatusRunning
				a.lastActivity = time.Now()
				return
			}
			if part.State == "error" && timeSinceLastActivity < 5*time.Minute {
				a.status = agent.StatusErrored
				a.lastActivity = time.UnixMilli(part.Time.Created)
				return
			}
		}
	}

	if timeSinceLastActivity < 60*time.Second {
		a.status = agent.StatusRunning
	} else if timeSinceLastActivity < 30*time.Minute {
		a.status = agent.StatusIdle
	} else {
		a.status = agent.StatusCompleted
	}

	a.lastActivity = lastMsgTime
}

// extractCurrentTask extracts the current task from messages
func (a *OpenCodeAgent) extractCurrentTask() {
	// Look for the last user message as the current task
	for i := len(a.messages) - 1; i >= 0; i-- {
		if a.messages[i].Role == "user" {
			task := a.messages[i].Summary.Title
			if task == "" {
				task = "Working..."
			}
			if len(task) > 100 {
				task = task[:100] + "..."
			}
			a.currentTask = strings.TrimSpace(task)
			return
		}
	}
}

// Refresh refreshes the agent's state from disk
func (a *OpenCodeAgent) Refresh() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Re-read session file
	data, err := os.ReadFile(a.sessionFile)
	if err != nil {
		return err
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}
	a.sessionData = &session
	a.name = session.Title
	a.lastActivity = time.UnixMilli(session.Time.Updated)

	// Clear and reload
	a.output.Reset()
	if err := a.loadMessages(); err != nil {
		return err
	}

	a.determineStatus()
	a.extractCurrentTask()

	return nil
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

func (a *OpenCodeAgent) ParentID() string {
	return a.parentID
}

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

// Output returns the output stream (lazy-loads history on first access)
func (a *OpenCodeAgent) Output() io.Reader {
	a.mu.RLock()
	if !a.loaded {
		a.mu.RUnlock()
		a.LoadFullHistory()
		a.mu.RLock()
	}
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

// SendInput sends input to the agent by appending to the session via CLI
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
	// OpenCode sessions can be terminated by killing the process
	// For now, we just mark it as cancelled
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

// Provider implements the agent.Provider interface for opencode
type Provider struct {
	storagePath   string
	watchInterval time.Duration
	maxAge        time.Duration
	agents        map[string]*OpenCodeAgent
	mu            sync.RWMutex
	watcher       *fsnotify.Watcher
}

// NewProvider creates a new opencode provider
func NewProvider(storagePath string, watchInterval time.Duration, maxAge time.Duration) *Provider {
	return &Provider{
		storagePath:   storagePath,
		watchInterval: watchInterval,
		maxAge:        maxAge,
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

// Discover discovers all opencode sessions
func (p *Provider) Discover(ctx context.Context) ([]agent.Agent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear existing agents
	p.agents = make(map[string]*OpenCodeAgent)

	// Sessions are stored in storage/session/<project-id>/*.json
	sessionBasePath := filepath.Join(p.storagePath, "session")
	projectDirs, err := os.ReadDir(sessionBasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No sessions yet
		}
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var agents []agent.Agent

	for _, projectDir := range projectDirs {
		if !projectDir.IsDir() {
			continue
		}

		projectPath := filepath.Join(sessionBasePath, projectDir.Name())
		sessionFiles, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, sessionFile := range sessionFiles {
			if sessionFile.IsDir() || !strings.HasSuffix(sessionFile.Name(), ".json") {
				continue
			}

			sessionFilePath := filepath.Join(projectPath, sessionFile.Name())
			a, err := NewOpenCodeAgent(p.storagePath, sessionFilePath)
			if err != nil {
				continue // Skip invalid sessions
			}

			if p.maxAge > 0 && time.Since(a.LastActivity()) > p.maxAge {
				continue
			}

			p.agents[a.ID()] = a
			agents = append(agents, a)
		}
	}

	return agents, nil
}

// Watch watches for changes in opencode sessions
func (p *Provider) Watch(ctx context.Context) (<-chan agent.Event, error) {
	events := make(chan agent.Event, 100)

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	p.watcher = watcher

	// Watch the session directories
	sessionBasePath := filepath.Join(p.storagePath, "session")
	if err := watcher.Add(sessionBasePath); err != nil {
		// Directory might not exist yet
	}

	// Watch project directories
	projectDirs, _ := os.ReadDir(sessionBasePath)
	for _, projectDir := range projectDirs {
		if projectDir.IsDir() {
			watcher.Add(filepath.Join(sessionBasePath, projectDir.Name()))
		}
	}

	// Watch message directories for existing sessions
	p.mu.RLock()
	for _, a := range p.agents {
		msgPath := filepath.Join(p.storagePath, "message", a.ID())
		watcher.Add(msgPath)
		partPath := filepath.Join(p.storagePath, "part", a.ID())
		watcher.Add(partPath)
	}
	p.mu.RUnlock()

	go func() {
		defer close(events)
		defer watcher.Close()

		ticker := time.NewTicker(p.watchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return

			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
					p.handleFileChange(event.Name, events)
				}

			case <-ticker.C:
				// Periodic refresh of all agents
				p.refreshAll(events)

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				// Log error but continue
				_ = err
			}
		}
	}()

	return events, nil
}

// handleFileChange handles a file change event
func (p *Provider) handleFileChange(path string, events chan<- agent.Event) {
	// Check if this is a new session file
	if strings.Contains(path, "/session/") && strings.HasSuffix(path, ".json") {
		a, err := NewOpenCodeAgent(p.storagePath, path)
		if err != nil {
			return
		}

		p.mu.Lock()
		existing, exists := p.agents[a.ID()]
		if !exists {
			p.agents[a.ID()] = a
			p.mu.Unlock()

			// Watch the new session's messages and parts
			if p.watcher != nil {
				msgPath := filepath.Join(p.storagePath, "message", a.ID())
				p.watcher.Add(msgPath)
				partPath := filepath.Join(p.storagePath, "part", a.ID())
				p.watcher.Add(partPath)
			}

			events <- agent.Event{
				Type:      agent.EventAgentDiscovered,
				AgentID:   a.ID(),
				Agent:     a,
				Timestamp: time.Now(),
			}
		} else {
			p.mu.Unlock()
			// Refresh existing agent
			existing.Refresh()
			events <- agent.Event{
				Type:      agent.EventAgentUpdated,
				AgentID:   existing.ID(),
				Agent:     existing,
				Timestamp: time.Now(),
			}
		}
		return
	}

	// Check if this is a message or part update
	if (strings.Contains(path, "/message/") || strings.Contains(path, "/part/")) && strings.HasSuffix(path, ".json") {
		// Extract session ID from path
		parts := strings.Split(path, "/")
		var sessionID string
		for i, part := range parts {
			if (part == "message" || part == "part") && i+1 < len(parts) {
				sessionID = parts[i+1]
				break
			}
		}

		if sessionID != "" {
			p.mu.RLock()
			a, ok := p.agents[sessionID]
			p.mu.RUnlock()

			if ok {
				oldStatus := a.Status()
				a.Refresh()
				newStatus := a.Status()

				eventType := agent.EventAgentUpdated
				if oldStatus != newStatus {
					switch newStatus {
					case agent.StatusRunning:
						eventType = agent.EventAgentStarted
					case agent.StatusCompleted:
						eventType = agent.EventAgentCompleted
					case agent.StatusErrored:
						eventType = agent.EventAgentErrored
					}
				}

				events <- agent.Event{
					Type:      eventType,
					AgentID:   a.ID(),
					Agent:     a,
					Timestamp: time.Now(),
				}
			}
		}
	}
}

// refreshAll refreshes all agents
func (p *Provider) refreshAll(events chan<- agent.Event) {
	p.mu.RLock()
	agents := make([]*OpenCodeAgent, 0, len(p.agents))
	for _, a := range p.agents {
		agents = append(agents, a)
	}
	p.mu.RUnlock()

	for _, a := range agents {
		oldStatus := a.Status()
		a.Refresh()
		newStatus := a.Status()

		if oldStatus != newStatus {
			eventType := agent.EventAgentUpdated
			switch newStatus {
			case agent.StatusRunning:
				eventType = agent.EventAgentStarted
			case agent.StatusCompleted:
				eventType = agent.EventAgentCompleted
			case agent.StatusErrored:
				eventType = agent.EventAgentErrored
			}

			events <- agent.Event{
				Type:      eventType,
				AgentID:   a.ID(),
				Agent:     a,
				Timestamp: time.Now(),
			}
		}
	}
}

// Spawn spawns a new opencode session
func (p *Provider) Spawn(ctx context.Context, config agent.SpawnConfig) (agent.Agent, error) {
	// Create a new opencode session by running opencode in the specified directory
	cmd := exec.CommandContext(ctx, "opencode")
	cmd.Dir = config.Directory

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range config.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Start the process in background
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start opencode: %w", err)
	}

	// Wait a moment for the session to be created
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
