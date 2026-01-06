// Package agent defines the core agent interface and types
package agent

import (
	"context"
	"io"
	"time"
)

// Status represents the current state of an agent
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusIdle
	StatusCompleted
	StatusErrored
	StatusContextLimit
	StatusCancelled
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusIdle:
		return "idle"
	case StatusCompleted:
		return "completed"
	case StatusErrored:
		return "errored"
	case StatusContextLimit:
		return "context_limit"
	case StatusCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// StatusIcon returns the icon for the status
func (s Status) Icon() string {
	switch s {
	case StatusPending:
		return "○"
	case StatusRunning:
		return "●"
	case StatusIdle:
		return "◌"
	case StatusCompleted:
		return "✓"
	case StatusErrored:
		return "✗"
	case StatusContextLimit:
		return "⚠"
	case StatusCancelled:
		return "⊘"
	default:
		return "?"
	}
}

// Metrics holds agent performance metrics
type Metrics struct {
	TokensIn           int64         `json:"tokens_in"`
	TokensOut          int64         `json:"tokens_out"`
	EstimatedCost      float64       `json:"estimated_cost"`
	Duration           time.Duration `json:"duration"`
	ActiveTime         time.Duration `json:"active_time"`
	IdleTime           time.Duration `json:"idle_time"`
	ToolCalls          int           `json:"tool_calls"`
	ErrorCount         int           `json:"error_count"`
	TasksCompleted     int           `json:"tasks_completed"`
	TasksFailed        int           `json:"tasks_failed"`
	ContextUtilization float64       `json:"context_utilization"` // 0.0 - 1.0
}

// Agent represents a single AI agent instance
type Agent interface {
	// Identity
	ID() string
	Name() string
	Type() string // "opencode", "claude", etc.

	// Location
	Directory() string
	ProjectID() string

	// Hierarchy
	ParentID() string
	IsBackground() bool

	// Status
	Status() Status
	StartTime() time.Time
	LastActivity() time.Time

	// Session data
	Output() io.Reader   // Stream of output
	CurrentTask() string // What the agent is working on
	Metrics() Metrics    // Performance metrics
	LastError() error    // Last error if any

	// Control
	SendInput(input string) error
	Terminate() error
	Pause() error
	Resume() error
}

// EventType represents the type of agent event
type EventType int

const (
	EventAgentDiscovered EventType = iota
	EventAgentUpdated
	EventAgentStarted
	EventAgentCompleted
	EventAgentErrored
	EventAgentContextLimit
	EventAgentTerminated
	EventAgentPaused
	EventAgentResumed
	EventAgentInput
	EventAgentOutput
)

func (e EventType) String() string {
	switch e {
	case EventAgentDiscovered:
		return "discovered"
	case EventAgentUpdated:
		return "updated"
	case EventAgentStarted:
		return "started"
	case EventAgentCompleted:
		return "completed"
	case EventAgentErrored:
		return "errored"
	case EventAgentContextLimit:
		return "context_limit"
	case EventAgentTerminated:
		return "terminated"
	case EventAgentPaused:
		return "paused"
	case EventAgentResumed:
		return "resumed"
	case EventAgentInput:
		return "input"
	case EventAgentOutput:
		return "output"
	default:
		return "unknown"
	}
}

// Event represents an agent lifecycle event
type Event struct {
	Type      EventType
	AgentID   string
	Agent     Agent
	Timestamp time.Time
	Data      interface{}
	Error     error
}

type StreamEvent struct {
	Type      string
	AgentID   string
	SessionID string
	MessageID string
	Text      string
	ToolName  string
	State     string
	Error     string
	Timestamp time.Time
}

type StreamingAgent interface {
	Agent
	SendInputAsync(ctx context.Context, input string) (<-chan StreamEvent, error)
	IsExecuting() bool
	CancelExecution()
}

// SpawnConfig holds configuration for spawning a new agent
type SpawnConfig struct {
	Type      string            `json:"type"`
	Name      string            `json:"name"`
	Directory string            `json:"directory"`
	Prompt    string            `json:"prompt"`
	Env       map[string]string `json:"env"`
}

// Provider discovers and manages agents of a specific type
type Provider interface {
	// Identity
	Name() string
	Type() string

	// Discovery
	Discover(ctx context.Context) ([]Agent, error)
	Watch(ctx context.Context) (<-chan Event, error)

	// Control
	Spawn(ctx context.Context, config SpawnConfig) (Agent, error)
	Get(id string) (Agent, error)
	List() []Agent
	Terminate(id string) error
	SendInput(id string, input string) error
}

// Registry manages multiple agent providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Type()] = provider
}

// Get returns a provider by type
func (r *Registry) Get(providerType string) (Provider, bool) {
	p, ok := r.providers[providerType]
	return p, ok
}

// List returns all registered providers
func (r *Registry) List() []Provider {
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// DiscoverAll discovers agents from all providers
func (r *Registry) DiscoverAll(ctx context.Context) ([]Agent, error) {
	var allAgents []Agent
	for _, p := range r.providers {
		agents, err := p.Discover(ctx)
		if err != nil {
			continue // Log but don't fail
		}
		allAgents = append(allAgents, agents...)
	}
	return allAgents, nil
}

// WatchAll watches for events from all providers
func (r *Registry) WatchAll(ctx context.Context) (<-chan Event, error) {
	merged := make(chan Event, 100)

	for _, p := range r.providers {
		ch, err := p.Watch(ctx)
		if err != nil {
			continue
		}
		go func(ch <-chan Event) {
			for event := range ch {
				select {
				case merged <- event:
				case <-ctx.Done():
					return
				}
			}
		}(ch)
	}

	return merged, nil
}
