// Package session manages agent sessions
package session

import (
	"context"
	"sync"
	"time"

	"github.com/localrivet/auto/internal/agent"
	"github.com/localrivet/auto/internal/alert"
	"github.com/localrivet/auto/internal/config"
	"github.com/localrivet/auto/internal/store"
)

// Manager coordinates session discovery, monitoring, and lifecycle
type Manager struct {
	cfg      *config.Config
	store    *store.Store
	registry *agent.Registry
	alertMgr *alert.Manager
	agents   map[string]agent.Agent
	mu       sync.RWMutex
	onEvent  func(agent.Event)
	cancel   context.CancelFunc
}

// NewManager creates a new session manager
func NewManager(cfg *config.Config, st *store.Store, registry *agent.Registry, alertMgr *alert.Manager) *Manager {
	return &Manager{
		cfg:      cfg,
		store:    st,
		registry: registry,
		alertMgr: alertMgr,
		agents:   make(map[string]agent.Agent),
	}
}

// OnEvent sets the callback for agent events
func (m *Manager) OnEvent(fn func(agent.Event)) {
	m.onEvent = fn
}

// Start starts the session manager
func (m *Manager) Start(ctx context.Context) error {
	ctx, m.cancel = context.WithCancel(ctx)

	// Initial discovery
	agents, err := m.registry.DiscoverAll(ctx)
	if err != nil {
		return err
	}

	m.mu.Lock()
	for _, a := range agents {
		m.agents[a.ID()] = a
		// Persist to store
		if m.store != nil {
			m.store.SaveSession(&store.SessionRecord{
				ID:           a.ID(),
				AgentID:      a.ID(),
				AgentType:    a.Type(),
				AgentName:    a.Name(),
				Directory:    a.Directory(),
				ProjectID:    a.ProjectID(),
				Status:       a.Status().String(),
				StartTime:    a.StartTime(),
				LastActivity: a.LastActivity(),
				TokensIn:     a.Metrics().TokensIn,
				TokensOut:    a.Metrics().TokensOut,
			})
		}
	}
	m.mu.Unlock()

	// Start watching for events
	events, err := m.registry.WatchAll(ctx)
	if err != nil {
		return err
	}

	go m.processEvents(ctx, events)

	return nil
}

// Stop stops the session manager
func (m *Manager) Stop() {
	if m.cancel != nil {
		m.cancel()
	}
}

// processEvents processes agent events
func (m *Manager) processEvents(ctx context.Context, events <-chan agent.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}

			m.handleEvent(ctx, event)
		}
	}
}

// handleEvent handles a single agent event
func (m *Manager) handleEvent(ctx context.Context, event agent.Event) {
	m.mu.Lock()
	switch event.Type {
	case agent.EventAgentDiscovered:
		if event.Agent != nil {
			m.agents[event.AgentID] = event.Agent
		}
	case agent.EventAgentTerminated:
		delete(m.agents, event.AgentID)
	default:
		if event.Agent != nil {
			m.agents[event.AgentID] = event.Agent
		}
	}
	m.mu.Unlock()

	// Update store
	if m.store != nil && event.Agent != nil {
		a := event.Agent
		m.store.SaveSession(&store.SessionRecord{
			ID:           a.ID(),
			AgentID:      a.ID(),
			AgentType:    a.Type(),
			AgentName:    a.Name(),
			Directory:    a.Directory(),
			ProjectID:    a.ProjectID(),
			Status:       a.Status().String(),
			StartTime:    a.StartTime(),
			LastActivity: a.LastActivity(),
			TokensIn:     a.Metrics().TokensIn,
			TokensOut:    a.Metrics().TokensOut,
		})
	}

	// Send alerts for important events
	if m.alertMgr != nil {
		m.alertMgr.SendAgentEvent(ctx, event)
	}

	// Notify callback
	if m.onEvent != nil {
		m.onEvent(event)
	}
}

// List returns all agents
func (m *Manager) List() []agent.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]agent.Agent, 0, len(m.agents))
	for _, a := range m.agents {
		agents = append(agents, a)
	}
	return agents
}

// Get returns an agent by ID
func (m *Manager) Get(id string) (agent.Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	a, ok := m.agents[id]
	return a, ok
}

// Spawn spawns a new agent session
func (m *Manager) Spawn(ctx context.Context, config agent.SpawnConfig) (agent.Agent, error) {
	provider, ok := m.registry.Get(config.Type)
	if !ok {
		provider = m.registry.List()[0] // Use first available provider
	}

	a, err := provider.Spawn(ctx, config)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.agents[a.ID()] = a
	m.mu.Unlock()

	return a, nil
}

// Terminate terminates an agent
func (m *Manager) Terminate(id string) error {
	m.mu.RLock()
	a, ok := m.agents[id]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	return a.Terminate()
}

// SendInput sends input to an agent
func (m *Manager) SendInput(id string, input string) error {
	m.mu.RLock()
	a, ok := m.agents[id]
	m.mu.RUnlock()

	if !ok {
		return nil
	}

	return a.SendInput(input)
}

// Stats returns aggregate statistics
func (m *Manager) Stats() *Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &Stats{
		Total:     len(m.agents),
		ByStatus:  make(map[agent.Status]int),
		ByType:    make(map[string]int),
		ByProject: make(map[string]int),
	}

	for _, a := range m.agents {
		stats.ByStatus[a.Status()]++
		stats.ByType[a.Type()]++
		stats.ByProject[a.ProjectID()]++

		metrics := a.Metrics()
		stats.TotalTokensIn += metrics.TokensIn
		stats.TotalTokensOut += metrics.TokensOut
		stats.TotalCost += metrics.EstimatedCost
		stats.TotalToolCalls += metrics.ToolCalls
		stats.TotalErrors += metrics.ErrorCount
	}

	return stats
}

// Stats holds aggregate statistics
type Stats struct {
	Total          int
	ByStatus       map[agent.Status]int
	ByType         map[string]int
	ByProject      map[string]int
	TotalTokensIn  int64
	TotalTokensOut int64
	TotalCost      float64
	TotalToolCalls int
	TotalErrors    int
}

// GroupMode represents how agents are grouped
type GroupMode string

const (
	GroupModeFlat    GroupMode = "flat"
	GroupModeType    GroupMode = "type"
	GroupModeProject GroupMode = "project"
	GroupModeStatus  GroupMode = "status"
)

// Group represents a group of agents
type Group struct {
	Name   string
	Agents []agent.Agent
}

// GroupBy returns agents grouped by the specified mode
func (m *Manager) GroupBy(mode GroupMode) []Group {
	m.mu.RLock()
	defer m.mu.RUnlock()

	groups := make(map[string][]agent.Agent)

	for _, a := range m.agents {
		var key string
		switch mode {
		case GroupModeType:
			key = a.Type()
		case GroupModeProject:
			key = a.ProjectID()
		case GroupModeStatus:
			key = a.Status().String()
		default:
			key = "all"
		}
		groups[key] = append(groups[key], a)
	}

	result := make([]Group, 0, len(groups))
	for name, agents := range groups {
		result = append(result, Group{Name: name, Agents: agents})
	}

	return result
}

// FilterByStatus returns agents with the specified status
func (m *Manager) FilterByStatus(status agent.Status) []agent.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []agent.Agent
	for _, a := range m.agents {
		if a.Status() == status {
			result = append(result, a)
		}
	}
	return result
}

// FilterByType returns agents of the specified type
func (m *Manager) FilterByType(agentType string) []agent.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []agent.Agent
	for _, a := range m.agents {
		if a.Type() == agentType {
			result = append(result, a)
		}
	}
	return result
}

// Search searches agents by name
func (m *Manager) Search(query string) []agent.Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []agent.Agent
	for _, a := range m.agents {
		// Simple substring match for now
		if containsIgnoreCase(a.Name(), query) ||
			containsIgnoreCase(a.ID(), query) ||
			containsIgnoreCase(a.Directory(), query) ||
			containsIgnoreCase(a.CurrentTask(), query) {
			result = append(result, a)
		}
	}
	return result
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains
	sl := len(s)
	subl := len(substr)
	if subl > sl {
		return false
	}
	for i := 0; i <= sl-subl; i++ {
		if equalFold(s[i:i+subl], substr) {
			return true
		}
	}
	return false
}

func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		sr := s[i]
		tr := t[i]
		if sr >= 'A' && sr <= 'Z' {
			sr += 'a' - 'A'
		}
		if tr >= 'A' && tr <= 'Z' {
			tr += 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return true
}

// Refresh refreshes all agent data
func (m *Manager) Refresh(ctx context.Context) error {
	agents, err := m.registry.DiscoverAll(ctx)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update existing and add new
	seen := make(map[string]bool)
	for _, a := range agents {
		m.agents[a.ID()] = a
		seen[a.ID()] = true
	}

	// Remove agents that no longer exist
	for id := range m.agents {
		if !seen[id] {
			delete(m.agents, id)
		}
	}

	return nil
}

// ActiveCount returns the number of active (running) agents
func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, a := range m.agents {
		if a.Status() == agent.StatusRunning {
			count++
		}
	}
	return count
}

// LastActivityTime returns the most recent activity time across all agents
func (m *Manager) LastActivityTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest time.Time
	for _, a := range m.agents {
		if a.LastActivity().After(latest) {
			latest = a.LastActivity()
		}
	}
	return latest
}
