// Package agent contains mock implementations for testing
package agent

import (
	"context"
	"io"
	"time"
)

// MockAgent implements Agent interface for testing
type MockAgent struct {
	MockID           string
	MockName         string
	MockType         string
	MockStatus       Status
	MockDirectory    string
	MockProjectID    string
	MockCurrentTask  string
	MockStartTime    time.Time
	MockLastActivity time.Time
	MockMetrics      Metrics
	MockLastError    error
	MockOutput       []byte

	// Track method calls
	TerminateCalled bool
	SendInputCalled bool
	LastInput       string
}

// NewMockAgent creates a new mock agent with default values
func NewMockAgent(id, name string) *MockAgent {
	now := time.Now()
	return &MockAgent{
		MockID:           id,
		MockName:         name,
		MockType:         "mock",
		MockStatus:       StatusRunning,
		MockDirectory:    "/test/dir",
		MockProjectID:    "test-project",
		MockCurrentTask:  "test task",
		MockStartTime:    now,
		MockLastActivity: now,
		MockMetrics: Metrics{
			TokensIn:      1000,
			TokensOut:     500,
			EstimatedCost: 0.01,
			ToolCalls:     5,
			ErrorCount:    0,
		},
	}
}

// Interface implementation
func (m *MockAgent) ID() string              { return m.MockID }
func (m *MockAgent) Name() string            { return m.MockName }
func (m *MockAgent) Type() string            { return m.MockType }
func (m *MockAgent) Status() Status          { return m.MockStatus }
func (m *MockAgent) Directory() string       { return m.MockDirectory }
func (m *MockAgent) ProjectID() string       { return m.MockProjectID }
func (m *MockAgent) CurrentTask() string     { return m.MockCurrentTask }
func (m *MockAgent) StartTime() time.Time    { return m.MockStartTime }
func (m *MockAgent) LastActivity() time.Time { return m.MockLastActivity }
func (m *MockAgent) Metrics() Metrics        { return m.MockMetrics }
func (m *MockAgent) LastError() error        { return m.MockLastError }
func (m *MockAgent) Output() io.Reader       { return nil }

func (m *MockAgent) SendInput(input string) error {
	m.SendInputCalled = true
	m.LastInput = input
	return nil
}

func (m *MockAgent) Terminate() error {
	m.TerminateCalled = true
	m.MockStatus = StatusCancelled
	return nil
}

func (m *MockAgent) Pause() error {
	m.MockStatus = StatusIdle
	return nil
}

func (m *MockAgent) Resume() error {
	m.MockStatus = StatusRunning
	return nil
}

func (m *MockAgent) Refresh() error {
	m.MockLastActivity = time.Now()
	return nil
}

// MockProvider implements Provider interface for testing
type MockProvider struct {
	MockAgents   []Agent
	MockEvents   chan Event
	SpawnedAgent *MockAgent
}

// NewMockProvider creates a new mock provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		MockAgents: make([]Agent, 0),
		MockEvents: make(chan Event, 100),
	}
}

func (p *MockProvider) Type() string {
	return "mock"
}

func (p *MockProvider) Discover(ctx context.Context) ([]Agent, error) {
	return p.MockAgents, nil
}

func (p *MockProvider) Watch(ctx context.Context) (<-chan Event, error) {
	return p.MockEvents, nil
}

func (p *MockProvider) Spawn(ctx context.Context, config SpawnConfig) (Agent, error) {
	agent := NewMockAgent(config.Name+"-id", config.Name)
	agent.MockDirectory = config.Directory
	p.SpawnedAgent = agent
	p.MockAgents = append(p.MockAgents, agent)
	return agent, nil
}

// AddAgent adds an agent to the mock provider
func (p *MockProvider) AddAgent(agent Agent) {
	p.MockAgents = append(p.MockAgents, agent)
}

// SendEvent sends a mock event
func (p *MockProvider) SendEvent(event Event) {
	p.MockEvents <- event
}

// Close closes the event channel
func (p *MockProvider) Close() {
	close(p.MockEvents)
}
