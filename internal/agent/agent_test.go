package agent

import (
	"testing"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusIdle, "idle"},
		{StatusCompleted, "completed"},
		{StatusErrored, "errored"},
		{StatusContextLimit, "context_limit"},
		{StatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("Status.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status   Status
		expected string
	}{
		{StatusPending, "○"},
		{StatusRunning, "●"},
		{StatusIdle, "◌"},
		{StatusCompleted, "✓"},
		{StatusErrored, "✗"},
		{StatusContextLimit, "⚠"},
		{StatusCancelled, "⊘"},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			if got := tt.status.Icon(); got != tt.expected {
				t.Errorf("Status.Icon() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEventTypeString(t *testing.T) {
	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventAgentDiscovered, "discovered"},
		{EventAgentUpdated, "updated"},
		{EventAgentStarted, "started"},
		{EventAgentCompleted, "completed"},
		{EventAgentErrored, "errored"},
		{EventAgentContextLimit, "context_limit"},
		{EventAgentTerminated, "terminated"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.eventType.String(); got != tt.expected {
				t.Errorf("EventType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRegistryBasicOperations(t *testing.T) {
	registry := NewRegistry()

	if len(registry.List()) != 0 {
		t.Error("New registry should be empty")
	}

	_, found := registry.Get("nonexistent")
	if found {
		t.Error("Get should return false for nonexistent provider")
	}
}
