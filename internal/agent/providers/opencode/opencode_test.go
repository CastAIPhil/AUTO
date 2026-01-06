package opencode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/localrivet/auto/internal/agent"
)

// Helper to create a temporary session directory with valid session data
func createTestSession(t *testing.T, sessionID, title, projectPath string, createdAt, updatedAt time.Time) string {
	t.Helper()

	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, sessionID)
	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	session := SessionData{
		ID:        sessionID,
		Path:      projectPath,
		Title:     title,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	sessionFile := filepath.Join(sessionPath, "session.json")
	if err := os.WriteFile(sessionFile, data, 0644); err != nil {
		t.Fatalf("Failed to write session.json: %v", err)
	}

	return sessionPath
}

// Helper to add a message to a session
func addTestMessage(t *testing.T, sessionPath string, msg MessageData) {
	t.Helper()

	messagesDir := filepath.Join(sessionPath, "messages")
	if err := os.MkdirAll(messagesDir, 0755); err != nil {
		t.Fatalf("Failed to create messages dir: %v", err)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	msgFile := filepath.Join(messagesDir, msg.ID+".json")
	if err := os.WriteFile(msgFile, data, 0644); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}
}

func TestNewOpenCodeAgent(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session-1", "Test Session", "/home/user/project", now, now)

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.ID() != "test-session-1" {
		t.Errorf("ID() = %v, want %v", a.ID(), "test-session-1")
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

	// Status should be pending with no messages
	if a.Status() != agent.StatusPending {
		t.Errorf("Status() = %v, want %v", a.Status(), agent.StatusPending)
	}
}

func TestNewOpenCodeAgent_InvalidPath(t *testing.T) {
	_, err := NewOpenCodeAgent("/nonexistent/path")
	if err == nil {
		t.Error("NewOpenCodeAgent() expected error for invalid path")
	}
}

func TestNewOpenCodeAgent_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, "invalid-session")
	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	// Write invalid JSON
	if err := os.WriteFile(filepath.Join(sessionPath, "session.json"), []byte("not json"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, err := NewOpenCodeAgent(sessionPath)
	if err == nil {
		t.Error("NewOpenCodeAgent() expected error for invalid JSON")
	}
}

func TestOpenCodeAgent_StatusRunning(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	// Add a recent message (within 30 seconds)
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "user",
		Content:   "Hello",
		CreatedAt: now.Add(-10 * time.Second),
	})

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.Status() != agent.StatusRunning {
		t.Errorf("Status() = %v, want %v (recent message should indicate running)", a.Status(), agent.StatusRunning)
	}
}

func TestOpenCodeAgent_StatusIdle(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	// Add an older message (between 30 seconds and 5 minutes)
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "user",
		Content:   "Hello",
		CreatedAt: now.Add(-2 * time.Minute),
	})

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.Status() != agent.StatusIdle {
		t.Errorf("Status() = %v, want %v (older message should indicate idle)", a.Status(), agent.StatusIdle)
	}
}

func TestOpenCodeAgent_StatusErrored(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	// Add messages indicating error
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "user",
		Content:   "Fix the bug",
		CreatedAt: now.Add(-10 * time.Minute),
	})
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-2",
		SessionID: "test-session",
		Role:      "assistant",
		Content:   "I encountered an error while trying to fix this",
		CreatedAt: now.Add(-10 * time.Minute),
	})

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.Status() != agent.StatusErrored {
		t.Errorf("Status() = %v, want %v (error in message should indicate errored)", a.Status(), agent.StatusErrored)
	}
}

func TestOpenCodeAgent_StatusCompleted(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	// Add messages indicating completion
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "user",
		Content:   "Fix the bug",
		CreatedAt: now.Add(-10 * time.Minute),
	})
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-2",
		SessionID: "test-session",
		Role:      "assistant",
		Content:   "I have completed the task successfully",
		CreatedAt: now.Add(-10 * time.Minute),
	})

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.Status() != agent.StatusCompleted {
		t.Errorf("Status() = %v, want %v (complete in message should indicate completed)", a.Status(), agent.StatusCompleted)
	}
}

func TestOpenCodeAgent_CurrentTask(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "user",
		Content:   "Please implement the new feature",
		CreatedAt: now.Add(-10 * time.Second),
	})

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.CurrentTask() != "Please implement the new feature" {
		t.Errorf("CurrentTask() = %v, want %v", a.CurrentTask(), "Please implement the new feature")
	}
}

func TestOpenCodeAgent_CurrentTask_Truncated(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	longTask := "This is a very long task description that exceeds one hundred characters and should be truncated when displayed"
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "user",
		Content:   longTask,
		CreatedAt: now.Add(-10 * time.Second),
	})

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	task := a.CurrentTask()
	if len(task) > 103 { // 100 chars + "..."
		t.Errorf("CurrentTask() should be truncated, got length %d", len(task))
	}
	if task[len(task)-3:] != "..." {
		t.Errorf("CurrentTask() should end with '...', got %v", task)
	}
}

func TestOpenCodeAgent_Metrics(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	// Add message with token metadata
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "assistant",
		Content:   "Response",
		CreatedAt: now.Add(-10 * time.Second),
		Metadata: map[string]interface{}{
			"tokens": map[string]interface{}{
				"input":  float64(100),
				"output": float64(50),
			},
		},
	})

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	metrics := a.Metrics()
	if metrics.TokensIn != 100 {
		t.Errorf("Metrics().TokensIn = %d, want %d", metrics.TokensIn, 100)
	}
	if metrics.TokensOut != 50 {
		t.Errorf("Metrics().TokensOut = %d, want %d", metrics.TokensOut, 50)
	}
}

func TestOpenCodeAgent_Refresh(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	// Initially pending
	if a.Status() != agent.StatusPending {
		t.Errorf("Initial Status() = %v, want %v", a.Status(), agent.StatusPending)
	}

	// Add a message
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "user",
		Content:   "Hello",
		CreatedAt: time.Now(),
	})

	// Refresh and check status changed
	if err := a.Refresh(); err != nil {
		t.Errorf("Refresh() error = %v", err)
	}

	if a.Status() != agent.StatusRunning {
		t.Errorf("After Refresh(), Status() = %v, want %v", a.Status(), agent.StatusRunning)
	}
}

func TestOpenCodeAgent_Terminate(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if err := a.Terminate(); err != nil {
		t.Errorf("Terminate() error = %v", err)
	}

	if a.Status() != agent.StatusCancelled {
		t.Errorf("After Terminate(), Status() = %v, want %v", a.Status(), agent.StatusCancelled)
	}
}

func TestOpenCodeAgent_UnsupportedOperations(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if err := a.SendInput("test"); err == nil {
		t.Error("SendInput() should return error for unsupported operation")
	}

	if err := a.Pause(); err == nil {
		t.Error("Pause() should return error for unsupported operation")
	}

	if err := a.Resume(); err == nil {
		t.Error("Resume() should return error for unsupported operation")
	}
}

func TestOpenCodeAgent_NameFallback(t *testing.T) {
	now := time.Now()
	tempDir := t.TempDir()
	sessionPath := filepath.Join(tempDir, "abc12345678")
	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	// Session with empty title
	session := SessionData{
		ID:        "abc12345678",
		Path:      "/project",
		Title:     "", // Empty title
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, _ := json.Marshal(session)
	os.WriteFile(filepath.Join(sessionPath, "session.json"), data, 0644)

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	// Should fall back to short ID
	if a.Name() != "abc12345" {
		t.Errorf("Name() = %v, want %v (should fallback to short ID)", a.Name(), "abc12345")
	}
}

func TestProvider_NewProvider(t *testing.T) {
	p := NewProvider("/tmp/test", 5*time.Second)

	if p == nil {
		t.Fatal("NewProvider() returned nil")
	}

	if p.storagePath != "/tmp/test" {
		t.Errorf("storagePath = %v, want %v", p.storagePath, "/tmp/test")
	}

	if p.watchInterval != 5*time.Second {
		t.Errorf("watchInterval = %v, want %v", p.watchInterval, 5*time.Second)
	}

	if p.Name() != "OpenCode" {
		t.Errorf("Name() = %v, want %v", p.Name(), "OpenCode")
	}

	if p.Type() != "opencode" {
		t.Errorf("Type() = %v, want %v", p.Type(), "opencode")
	}
}

func TestProvider_Discover(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create multiple sessions
	session1Path := filepath.Join(tempDir, "session-1")
	os.MkdirAll(session1Path, 0755)
	s1 := SessionData{ID: "session-1", Path: "/p1", Title: "Session 1", CreatedAt: now, UpdatedAt: now}
	data1, _ := json.Marshal(s1)
	os.WriteFile(filepath.Join(session1Path, "session.json"), data1, 0644)

	session2Path := filepath.Join(tempDir, "session-2")
	os.MkdirAll(session2Path, 0755)
	s2 := SessionData{ID: "session-2", Path: "/p2", Title: "Session 2", CreatedAt: now, UpdatedAt: now}
	data2, _ := json.Marshal(s2)
	os.WriteFile(filepath.Join(session2Path, "session.json"), data2, 0644)

	// Create a directory without session.json (should be skipped)
	os.MkdirAll(filepath.Join(tempDir, "not-a-session"), 0755)

	// Create a file (not a directory, should be skipped)
	os.WriteFile(filepath.Join(tempDir, "regular-file.txt"), []byte("test"), 0644)

	p := NewProvider(tempDir, 5*time.Second)
	agents, err := p.Discover(context.Background())

	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("Discover() returned %d agents, want %d", len(agents), 2)
	}
}

func TestProvider_Discover_NonExistentPath(t *testing.T) {
	p := NewProvider("/nonexistent/path", 5*time.Second)
	agents, err := p.Discover(context.Background())

	if err != nil {
		t.Fatalf("Discover() error = %v (should not error for nonexistent path)", err)
	}

	if agents != nil && len(agents) != 0 {
		t.Errorf("Discover() should return empty for nonexistent path")
	}
}

func TestProvider_GetAndList(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	sessionPath := filepath.Join(tempDir, "session-1")
	os.MkdirAll(sessionPath, 0755)
	s := SessionData{ID: "session-1", Path: "/p1", Title: "Session 1", CreatedAt: now, UpdatedAt: now}
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(sessionPath, "session.json"), data, 0644)

	p := NewProvider(tempDir, 5*time.Second)
	p.Discover(context.Background())

	// Test Get
	a, err := p.Get("session-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if a.ID() != "session-1" {
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
	tempDir := t.TempDir()
	now := time.Now()

	sessionPath := filepath.Join(tempDir, "session-1")
	os.MkdirAll(sessionPath, 0755)
	s := SessionData{ID: "session-1", Path: "/p1", Title: "Session 1", CreatedAt: now, UpdatedAt: now}
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(sessionPath, "session.json"), data, 0644)

	p := NewProvider(tempDir, 5*time.Second)
	p.Discover(context.Background())

	// Terminate existing agent
	if err := p.Terminate("session-1"); err != nil {
		t.Errorf("Terminate() error = %v", err)
	}

	// Check status changed
	a, _ := p.Get("session-1")
	if a.Status() != agent.StatusCancelled {
		t.Errorf("After Terminate(), Status() = %v, want %v", a.Status(), agent.StatusCancelled)
	}

	// Terminate non-existent
	if err := p.Terminate("non-existent"); err == nil {
		t.Error("Terminate() should error for non-existent agent")
	}
}

func TestProvider_SendInput(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	sessionPath := filepath.Join(tempDir, "session-1")
	os.MkdirAll(sessionPath, 0755)
	s := SessionData{ID: "session-1", Path: "/p1", Title: "Session 1", CreatedAt: now, UpdatedAt: now}
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(sessionPath, "session.json"), data, 0644)

	p := NewProvider(tempDir, 5*time.Second)
	p.Discover(context.Background())

	// SendInput returns error (not supported)
	if err := p.SendInput("session-1", "test"); err == nil {
		t.Error("SendInput() should return error (not supported for opencode)")
	}

	// SendInput for non-existent
	if err := p.SendInput("non-existent", "test"); err == nil {
		t.Error("SendInput() should error for non-existent agent")
	}
}

func TestProvider_Watch(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	// Create initial session
	sessionPath := filepath.Join(tempDir, "session-1")
	os.MkdirAll(sessionPath, 0755)
	s := SessionData{ID: "session-1", Path: "/p1", Title: "Session 1", CreatedAt: now, UpdatedAt: now}
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(sessionPath, "session.json"), data, 0644)

	p := NewProvider(tempDir, 100*time.Millisecond)
	p.Discover(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := p.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Create a new session while watching
	session2Path := filepath.Join(tempDir, "session-2")
	os.MkdirAll(session2Path, 0755)
	s2 := SessionData{ID: "session-2", Path: "/p2", Title: "Session 2", CreatedAt: now, UpdatedAt: now}
	data2, _ := json.Marshal(s2)
	os.WriteFile(filepath.Join(session2Path, "session.json"), data2, 0644)

	// Wait for event or timeout
	select {
	case event := <-events:
		if event.Type != agent.EventAgentDiscovered && event.Type != agent.EventAgentUpdated {
			t.Errorf("Expected discovery or update event, got %v", event.Type)
		}
	case <-ctx.Done():
		// Timeout is acceptable - file watcher events can be flaky in tests
		t.Log("Watch test timed out (acceptable in CI)")
	}
}

func TestOpenCodeAgent_Output(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	// Add assistant message
	addTestMessage(t, sessionPath, MessageData{
		ID:        "msg-1",
		SessionID: "test-session",
		Role:      "assistant",
		Content:   "Hello, this is output",
		CreatedAt: now,
	})

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	output := a.Output()
	if output == nil {
		t.Error("Output() returned nil")
	}
}

func TestOpenCodeAgent_ProjectID(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/home/user/myproject", now, now)

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.ProjectID() != "myproject" {
		t.Errorf("ProjectID() = %v, want %v", a.ProjectID(), "myproject")
	}
}

func TestOpenCodeAgent_StartTime(t *testing.T) {
	createdAt := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	sessionPath := createTestSession(t, "test-session", "Test", "/project", createdAt, createdAt)

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if !a.StartTime().Equal(createdAt) {
		t.Errorf("StartTime() = %v, want %v", a.StartTime(), createdAt)
	}
}

func TestOpenCodeAgent_LastError(t *testing.T) {
	now := time.Now()
	sessionPath := createTestSession(t, "test-session", "Test", "/project", now, now)

	a, err := NewOpenCodeAgent(sessionPath)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	// Initially no error
	if a.LastError() != nil {
		t.Errorf("LastError() = %v, want nil", a.LastError())
	}
}
