package opencode

import (
	"context"
	"testing"
	"time"

	"github.com/CastAIPhil/AUTO/internal/agent"
)

func TestNewOpenCodeAgentFromCLI(t *testing.T) {
	now := time.Now()
	session := CLISession{
		ID:        "ses_test1",
		Title:     "Test Session",
		Updated:   now.UnixMilli(),
		Created:   now.Add(-1 * time.Hour).UnixMilli(),
		ProjectID: "proj123",
		Directory: "/home/user/project",
	}

	a := NewOpenCodeAgentFromCLI(session)

	if a.ID() != "ses_test1" {
		t.Errorf("ID() = %v, want %v", a.ID(), "ses_test1")
	}

	if a.Name() != "Test Session" {
		t.Errorf("Name() = %v, want %v", a.Name(), "Test Session")
	}

	if a.Type() != "opencode" {
		t.Errorf("Type() = %v, want %v", a.Type(), "opencode")
	}

	if a.Directory() != "/home/user/project" {
		t.Errorf("Directory() = %v, want %v", a.Directory(), "/home/user/project")
	}

	if a.ProjectID() != "proj123" {
		t.Errorf("ProjectID() = %v, want %v", a.ProjectID(), "proj123")
	}

	// Recently updated session should show as Running
	if a.Status() != agent.StatusRunning {
		t.Errorf("Status() = %v, want %v (recently updated)", a.Status(), agent.StatusRunning)
	}
}

func TestOpenCodeAgent_StatusRunning(t *testing.T) {
	now := time.Now()
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: now.Add(-10 * time.Second).UnixMilli(), // Updated 10 seconds ago
		Created: now.Add(-1 * time.Hour).UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	if a.Status() != agent.StatusRunning {
		t.Errorf("Status() = %v, want %v (recent activity)", a.Status(), agent.StatusRunning)
	}
}

func TestOpenCodeAgent_StatusIdle(t *testing.T) {
	now := time.Now()
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: now.Add(-5 * time.Minute).UnixMilli(), // Updated 5 minutes ago
		Created: now.Add(-1 * time.Hour).UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	if a.Status() != agent.StatusIdle {
		t.Errorf("Status() = %v, want %v", a.Status(), agent.StatusIdle)
	}
}

func TestOpenCodeAgent_StatusCompleted(t *testing.T) {
	now := time.Now()
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: now.Add(-1 * time.Hour).UnixMilli(), // Updated 1 hour ago
		Created: now.Add(-2 * time.Hour).UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	if a.Status() != agent.StatusCompleted {
		t.Errorf("Status() = %v, want %v", a.Status(), agent.StatusCompleted)
	}
}

func TestOpenCodeAgent_Update(t *testing.T) {
	now := time.Now()
	session := CLISession{
		ID:      "ses_test",
		Title:   "Original Title",
		Updated: now.Add(-1 * time.Hour).UnixMilli(),
		Created: now.Add(-2 * time.Hour).UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	if a.Status() != agent.StatusCompleted {
		t.Errorf("Initial Status() = %v, want %v", a.Status(), agent.StatusCompleted)
	}

	// Update with recent activity
	updatedSession := CLISession{
		ID:      "ses_test",
		Title:   "Updated Title",
		Updated: now.UnixMilli(), // Just updated
		Created: now.Add(-2 * time.Hour).UnixMilli(),
	}

	a.Update(updatedSession)

	if a.Name() != "Updated Title" {
		t.Errorf("Name() after update = %v, want %v", a.Name(), "Updated Title")
	}

	if a.Status() != agent.StatusRunning {
		t.Errorf("Status() after update = %v, want %v", a.Status(), agent.StatusRunning)
	}
}

func TestOpenCodeAgent_Terminate(t *testing.T) {
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	if err := a.Terminate(); err != nil {
		t.Errorf("Terminate() error = %v", err)
	}

	if a.Status() != agent.StatusCancelled {
		t.Errorf("After Terminate(), Status() = %v, want %v", a.Status(), agent.StatusCancelled)
	}
}

func TestOpenCodeAgent_UnsupportedOperations(t *testing.T) {
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	if err := a.Pause(); err == nil {
		t.Error("Pause() should return error for unsupported operation")
	}

	if err := a.Resume(); err == nil {
		t.Error("Resume() should return error for unsupported operation")
	}
}

func TestOpenCodeAgent_NameFallback(t *testing.T) {
	session := CLISession{
		ID:      "ses_abc12345678",
		Title:   "", // Empty title
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	// Should fall back to short ID
	if a.Name() != "ses_abc1" {
		t.Errorf("Name() = %v, want %v (should fallback to short ID)", a.Name(), "ses_abc1")
	}
}

func TestOpenCodeAgent_IsBackground(t *testing.T) {
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	// Default agents don't have parent IDs (CLI doesn't expose this currently)
	if a.IsBackground() {
		t.Error("IsBackground() should be false for agents without parent")
	}

	if a.ParentID() != "" {
		t.Errorf("ParentID() = %v, want empty string", a.ParentID())
	}
}

func TestProvider_NewProvider(t *testing.T) {
	p := NewProvider("/tmp/test", 5*time.Second, 24*time.Hour)

	if p == nil {
		t.Fatal("NewProvider() returned nil")
	}

	if p.watchInterval != 5*time.Second {
		t.Errorf("watchInterval = %v, want %v", p.watchInterval, 5*time.Second)
	}

	if p.maxAge != 24*time.Hour {
		t.Errorf("maxAge = %v, want %v", p.maxAge, 24*time.Hour)
	}

	if p.Name() != "OpenCode" {
		t.Errorf("Name() = %v, want %v", p.Name(), "OpenCode")
	}

	if p.Type() != "opencode" {
		t.Errorf("Type() = %v, want %v", p.Type(), "opencode")
	}
}

func TestProvider_GetAndList(t *testing.T) {
	p := NewProvider("/tmp", 5*time.Second, 0)

	// Manually add an agent for testing
	session := CLISession{
		ID:        "ses_1",
		Title:     "Session 1",
		Updated:   time.Now().UnixMilli(),
		Created:   time.Now().UnixMilli(),
		Directory: "/p1",
	}
	p.agents["ses_1"] = NewOpenCodeAgentFromCLI(session)

	// Test Get
	a, err := p.Get("ses_1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if a.ID() != "ses_1" {
		t.Errorf("Get() returned wrong agent")
	}

	// Test Get non-existent
	_, err = p.Get("non-existent")
	if err == nil {
		t.Error("Get() should error for non-existent agent")
	}

	// Test List
	list := p.List()
	if len(list) != 1 {
		t.Errorf("List() returned %d agents, want %d", len(list), 1)
	}
}

func TestProvider_Terminate(t *testing.T) {
	p := NewProvider("/tmp", 5*time.Second, 0)

	session := CLISession{
		ID:        "ses_1",
		Title:     "Session 1",
		Updated:   time.Now().UnixMilli(),
		Created:   time.Now().UnixMilli(),
		Directory: "/p1",
	}
	p.agents["ses_1"] = NewOpenCodeAgentFromCLI(session)

	// Terminate existing agent
	if err := p.Terminate("ses_1"); err != nil {
		t.Errorf("Terminate() error = %v", err)
	}

	// Check status changed
	a, _ := p.Get("ses_1")
	if a.Status() != agent.StatusCancelled {
		t.Errorf("After Terminate(), Status() = %v, want %v", a.Status(), agent.StatusCancelled)
	}

	// Terminate non-existent
	if err := p.Terminate("non-existent"); err == nil {
		t.Error("Terminate() should error for non-existent agent")
	}
}

func TestProvider_SendInput_Errors(t *testing.T) {
	p := NewProvider("/tmp", 5*time.Second, 0)

	session := CLISession{
		ID:        "ses_1",
		Title:     "Session 1",
		Updated:   time.Now().UnixMilli(),
		Created:   time.Now().UnixMilli(),
		Directory: "/tmp",
	}
	p.agents["ses_1"] = NewOpenCodeAgentFromCLI(session)

	// Empty input should error
	if err := p.SendInput("ses_1", ""); err == nil {
		t.Error("SendInput() should error for empty input")
	}

	// Non-existent agent should error
	if err := p.SendInput("non-existent", "test"); err == nil {
		t.Error("SendInput() should error for non-existent agent")
	}
}

func TestProvider_ListPrimary(t *testing.T) {
	p := NewProvider("/tmp", 5*time.Second, 0)

	session := CLISession{
		ID:        "ses_1",
		Title:     "Session 1",
		Updated:   time.Now().UnixMilli(),
		Created:   time.Now().UnixMilli(),
		Directory: "/p1",
	}
	p.agents["ses_1"] = NewOpenCodeAgentFromCLI(session)

	list := p.ListPrimary()
	if len(list) != 1 {
		t.Errorf("ListPrimary() returned %d agents, want %d", len(list), 1)
	}
}

func TestProvider_GetChildren(t *testing.T) {
	p := NewProvider("/tmp", 5*time.Second, 0)

	session := CLISession{
		ID:        "ses_1",
		Title:     "Session 1",
		Updated:   time.Now().UnixMilli(),
		Created:   time.Now().UnixMilli(),
		Directory: "/p1",
	}
	p.agents["ses_1"] = NewOpenCodeAgentFromCLI(session)

	// No children since we don't have parent/child relationships in CLI output
	children := p.GetChildren("ses_1")
	if len(children) != 0 {
		t.Errorf("GetChildren() returned %d children, want 0", len(children))
	}

	count := p.ChildCount("ses_1")
	if count != 0 {
		t.Errorf("ChildCount() = %d, want 0", count)
	}
}

func TestProvider_Watch(t *testing.T) {
	p := NewProvider("/tmp", 100*time.Millisecond, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	events, err := p.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Just verify the channel is returned and closes on context cancel
	<-ctx.Done()
	// Channel should close after context is done
	select {
	case _, ok := <-events:
		if ok {
			// Got an event, that's fine
		}
	case <-time.After(500 * time.Millisecond):
		// Timeout is acceptable
	}
}

func TestOpenCodeAgent_Output(t *testing.T) {
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	output := a.Output()
	if output == nil {
		t.Error("Output() returned nil")
	}
}

func TestOpenCodeAgent_StartTime(t *testing.T) {
	createdAt := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: createdAt.UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	// Check within 1 second tolerance (millisecond precision)
	diff := a.StartTime().Sub(createdAt)
	if diff > time.Second || diff < -time.Second {
		t.Errorf("StartTime() = %v, want approximately %v", a.StartTime(), createdAt)
	}
}

func TestOpenCodeAgent_LastError(t *testing.T) {
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	// Initially no error
	if a.LastError() != nil {
		t.Errorf("LastError() = %v, want nil", a.LastError())
	}
}

func TestOpenCodeAgent_Metrics(t *testing.T) {
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	// Metrics should be initialized
	metrics := a.Metrics()
	if metrics.TokensIn < 0 || metrics.TokensOut < 0 {
		t.Error("Metrics should have non-negative values")
	}
}

func TestOpenCodeAgent_Refresh(t *testing.T) {
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	// Refresh is a no-op for CLI-based agents
	if err := a.Refresh(); err != nil {
		t.Errorf("Refresh() error = %v", err)
	}
}

func TestOpenCodeAgent_LoadFullHistory(t *testing.T) {
	session := CLISession{
		ID:      "ses_test",
		Title:   "Test",
		Updated: time.Now().UnixMilli(),
		Created: time.Now().UnixMilli(),
	}

	a := NewOpenCodeAgentFromCLI(session)

	// LoadFullHistory is a no-op for CLI-based agents (can be implemented with opencode export)
	a.LoadFullHistory() // Should not panic
}
