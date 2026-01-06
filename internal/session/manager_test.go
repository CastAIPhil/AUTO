package session

import (
	"context"
	"testing"
	"time"

	"github.com/localrivet/auto/internal/agent"
	"github.com/localrivet/auto/internal/alert"
	"github.com/localrivet/auto/internal/config"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	alertMgr := alert.NewManager(&config.AlertsConfig{}, nil)

	m := NewManager(cfg, nil, registry, alertMgr)
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if m.cfg != cfg {
		t.Error("NewManager() did not set config")
	}
	if m.registry != registry {
		t.Error("NewManager() did not set registry")
	}
}

func TestManagerListAndGet(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	// Add mock agents directly
	mockAgent1 := agent.NewMockAgent("agent-1", "Agent 1")
	mockAgent2 := agent.NewMockAgent("agent-2", "Agent 2")

	m.mu.Lock()
	m.agents["agent-1"] = mockAgent1
	m.agents["agent-2"] = mockAgent2
	m.mu.Unlock()

	// Test List
	list := m.List()
	if len(list) != 2 {
		t.Errorf("List() = %d agents, want 2", len(list))
	}

	// Test Get
	got, ok := m.Get("agent-1")
	if !ok {
		t.Error("Get() returned false for existing agent")
	}
	if got.ID() != "agent-1" {
		t.Errorf("Get() returned wrong agent: %s", got.ID())
	}

	// Test Get non-existent
	_, ok = m.Get("non-existent")
	if ok {
		t.Error("Get() returned true for non-existent agent")
	}
}

func TestManagerStats(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	// Add mock agents with different statuses
	agent1 := agent.NewMockAgent("agent-1", "Agent 1")
	agent1.MockStatus = agent.StatusRunning
	agent1.MockType = "opencode"
	agent1.MockProjectID = "proj-1"
	agent1.MockMetrics.TokensIn = 1000
	agent1.MockMetrics.TokensOut = 500

	agent2 := agent.NewMockAgent("agent-2", "Agent 2")
	agent2.MockStatus = agent.StatusIdle
	agent2.MockType = "opencode"
	agent2.MockProjectID = "proj-1"
	agent2.MockMetrics.TokensIn = 2000
	agent2.MockMetrics.TokensOut = 1000

	agent3 := agent.NewMockAgent("agent-3", "Agent 3")
	agent3.MockStatus = agent.StatusErrored
	agent3.MockType = "claude"
	agent3.MockProjectID = "proj-2"
	agent3.MockMetrics.TokensIn = 500
	agent3.MockMetrics.ErrorCount = 1

	m.mu.Lock()
	m.agents["agent-1"] = agent1
	m.agents["agent-2"] = agent2
	m.agents["agent-3"] = agent3
	m.mu.Unlock()

	stats := m.Stats()

	if stats.Total != 3 {
		t.Errorf("Stats().Total = %d, want 3", stats.Total)
	}
	if stats.ByStatus[agent.StatusRunning] != 1 {
		t.Errorf("Stats().ByStatus[Running] = %d, want 1", stats.ByStatus[agent.StatusRunning])
	}
	if stats.ByStatus[agent.StatusIdle] != 1 {
		t.Errorf("Stats().ByStatus[Idle] = %d, want 1", stats.ByStatus[agent.StatusIdle])
	}
	if stats.ByType["opencode"] != 2 {
		t.Errorf("Stats().ByType[opencode] = %d, want 2", stats.ByType["opencode"])
	}
	if stats.ByProject["proj-1"] != 2 {
		t.Errorf("Stats().ByProject[proj-1] = %d, want 2", stats.ByProject["proj-1"])
	}
	if stats.TotalTokensIn != 3500 {
		t.Errorf("Stats().TotalTokensIn = %d, want 3500", stats.TotalTokensIn)
	}
}

func TestManagerGroupBy(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	agent1 := agent.NewMockAgent("agent-1", "Agent 1")
	agent1.MockType = "opencode"
	agent1.MockStatus = agent.StatusRunning

	agent2 := agent.NewMockAgent("agent-2", "Agent 2")
	agent2.MockType = "opencode"
	agent2.MockStatus = agent.StatusIdle

	agent3 := agent.NewMockAgent("agent-3", "Agent 3")
	agent3.MockType = "claude"
	agent3.MockStatus = agent.StatusRunning

	m.mu.Lock()
	m.agents["agent-1"] = agent1
	m.agents["agent-2"] = agent2
	m.agents["agent-3"] = agent3
	m.mu.Unlock()

	// Test group by type
	byType := m.GroupBy(GroupModeType)
	if len(byType) != 2 {
		t.Errorf("GroupBy(Type) = %d groups, want 2", len(byType))
	}

	// Test group by status
	byStatus := m.GroupBy(GroupModeStatus)
	if len(byStatus) != 2 {
		t.Errorf("GroupBy(Status) = %d groups, want 2", len(byStatus))
	}

	// Test group flat
	flat := m.GroupBy(GroupModeFlat)
	if len(flat) != 1 {
		t.Errorf("GroupBy(Flat) = %d groups, want 1", len(flat))
	}
	if len(flat[0].Agents) != 3 {
		t.Errorf("GroupBy(Flat) group size = %d, want 3", len(flat[0].Agents))
	}
}

func TestManagerFilterByStatus(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	agent1 := agent.NewMockAgent("agent-1", "Agent 1")
	agent1.MockStatus = agent.StatusRunning

	agent2 := agent.NewMockAgent("agent-2", "Agent 2")
	agent2.MockStatus = agent.StatusIdle

	agent3 := agent.NewMockAgent("agent-3", "Agent 3")
	agent3.MockStatus = agent.StatusRunning

	m.mu.Lock()
	m.agents["agent-1"] = agent1
	m.agents["agent-2"] = agent2
	m.agents["agent-3"] = agent3
	m.mu.Unlock()

	running := m.FilterByStatus(agent.StatusRunning)
	if len(running) != 2 {
		t.Errorf("FilterByStatus(Running) = %d, want 2", len(running))
	}

	idle := m.FilterByStatus(agent.StatusIdle)
	if len(idle) != 1 {
		t.Errorf("FilterByStatus(Idle) = %d, want 1", len(idle))
	}
}

func TestManagerFilterByType(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	agent1 := agent.NewMockAgent("agent-1", "Agent 1")
	agent1.MockType = "opencode"

	agent2 := agent.NewMockAgent("agent-2", "Agent 2")
	agent2.MockType = "claude"

	m.mu.Lock()
	m.agents["agent-1"] = agent1
	m.agents["agent-2"] = agent2
	m.mu.Unlock()

	opencode := m.FilterByType("opencode")
	if len(opencode) != 1 {
		t.Errorf("FilterByType(opencode) = %d, want 1", len(opencode))
	}
}

func TestManagerSearch(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	agent1 := agent.NewMockAgent("agent-1", "Project Alpha")
	agent1.MockDirectory = "/home/user/project-alpha"
	agent1.MockCurrentTask = "Building feature X"

	agent2 := agent.NewMockAgent("agent-2", "Project Beta")
	agent2.MockDirectory = "/home/user/project-beta"
	agent2.MockCurrentTask = "Testing"

	m.mu.Lock()
	m.agents["agent-1"] = agent1
	m.agents["agent-2"] = agent2
	m.mu.Unlock()

	// Search by name
	results := m.Search("Alpha")
	if len(results) != 1 {
		t.Errorf("Search(Alpha) = %d, want 1", len(results))
	}

	// Search by directory
	results = m.Search("beta")
	if len(results) != 1 {
		t.Errorf("Search(beta) = %d, want 1", len(results))
	}

	// Search by task
	results = m.Search("feature")
	if len(results) != 1 {
		t.Errorf("Search(feature) = %d, want 1", len(results))
	}

	// Search no match
	results = m.Search("gamma")
	if len(results) != 0 {
		t.Errorf("Search(gamma) = %d, want 0", len(results))
	}
}

func TestManagerActiveCount(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	agent1 := agent.NewMockAgent("agent-1", "Agent 1")
	agent1.MockStatus = agent.StatusRunning

	agent2 := agent.NewMockAgent("agent-2", "Agent 2")
	agent2.MockStatus = agent.StatusIdle

	agent3 := agent.NewMockAgent("agent-3", "Agent 3")
	agent3.MockStatus = agent.StatusRunning

	m.mu.Lock()
	m.agents["agent-1"] = agent1
	m.agents["agent-2"] = agent2
	m.agents["agent-3"] = agent3
	m.mu.Unlock()

	if m.ActiveCount() != 2 {
		t.Errorf("ActiveCount() = %d, want 2", m.ActiveCount())
	}
}

func TestManagerLastActivityTime(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	now := time.Now()
	earlier := now.Add(-1 * time.Hour)

	agent1 := agent.NewMockAgent("agent-1", "Agent 1")
	agent1.MockLastActivity = earlier

	agent2 := agent.NewMockAgent("agent-2", "Agent 2")
	agent2.MockLastActivity = now

	m.mu.Lock()
	m.agents["agent-1"] = agent1
	m.agents["agent-2"] = agent2
	m.mu.Unlock()

	lastActivity := m.LastActivityTime()
	if !lastActivity.Equal(now) {
		t.Errorf("LastActivityTime() = %v, want %v", lastActivity, now)
	}
}

func TestManagerSendInput(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	mockAgent := agent.NewMockAgent("agent-1", "Agent 1")
	m.mu.Lock()
	m.agents["agent-1"] = mockAgent
	m.mu.Unlock()

	err := m.SendInput("agent-1", "test input")
	if err != nil {
		t.Errorf("SendInput() error = %v", err)
	}

	if !mockAgent.SendInputCalled {
		t.Error("SendInput() did not call agent.SendInput")
	}
	if mockAgent.LastInput != "test input" {
		t.Errorf("SendInput() sent wrong input: %s", mockAgent.LastInput)
	}

	// Test non-existent agent
	err = m.SendInput("non-existent", "test")
	if err != nil {
		t.Errorf("SendInput() for non-existent agent should not error, got %v", err)
	}
}

func TestManagerTerminate(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	mockAgent := agent.NewMockAgent("agent-1", "Agent 1")
	m.mu.Lock()
	m.agents["agent-1"] = mockAgent
	m.mu.Unlock()

	err := m.Terminate("agent-1")
	if err != nil {
		t.Errorf("Terminate() error = %v", err)
	}

	if !mockAgent.TerminateCalled {
		t.Error("Terminate() did not call agent.Terminate")
	}
}

func TestManagerOnEvent(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	var received agent.Event
	m.OnEvent(func(e agent.Event) {
		received = e
	})

	if m.onEvent == nil {
		t.Error("OnEvent() did not set callback")
	}

	// Simulate event handling
	mockAgent := agent.NewMockAgent("agent-1", "Agent 1")
	event := agent.Event{
		Type:      agent.EventAgentUpdated,
		AgentID:   "agent-1",
		Agent:     mockAgent,
		Timestamp: time.Now(),
	}

	m.handleEvent(context.Background(), event)

	if received.AgentID != "agent-1" {
		t.Error("OnEvent callback was not called")
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Hello World", "hello", true},
		{"Hello World", "WORLD", true},
		{"Hello World", "foo", false},
		{"abc", "abcd", false},
		{"", "a", false},
		{"a", "", true},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}
