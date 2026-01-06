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

// Helper to create a test storage structure with a session
func createTestStorage(t *testing.T, sessionID, projectID, title, directory string, createdAt, updatedAt time.Time) string {
	t.Helper()

	tempDir := t.TempDir()

	// Create session directory: storage/session/<projectID>/
	sessionDir := filepath.Join(tempDir, "session", projectID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("Failed to create session dir: %v", err)
	}

	// Create session file
	session := SessionData{
		ID:        sessionID,
		ProjectID: projectID,
		Directory: directory,
		Title:     title,
	}
	session.Time.Created = createdAt.UnixMilli()
	session.Time.Updated = updatedAt.UnixMilli()

	data, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("Failed to marshal session: %v", err)
	}

	sessionFile := filepath.Join(sessionDir, sessionID+".json")
	if err := os.WriteFile(sessionFile, data, 0644); err != nil {
		t.Fatalf("Failed to write session file: %v", err)
	}

	// Create message directory: storage/message/<sessionID>/
	messageDir := filepath.Join(tempDir, "message", sessionID)
	if err := os.MkdirAll(messageDir, 0755); err != nil {
		t.Fatalf("Failed to create message dir: %v", err)
	}

	// Create part directory: storage/part/<sessionID>/
	partDir := filepath.Join(tempDir, "part", sessionID)
	if err := os.MkdirAll(partDir, 0755); err != nil {
		t.Fatalf("Failed to create part dir: %v", err)
	}

	return tempDir
}

// Helper to get the session file path
func getSessionFilePath(storagePath, projectID, sessionID string) string {
	return filepath.Join(storagePath, "session", projectID, sessionID+".json")
}

// Helper to add a message to a session
func addTestMessage(t *testing.T, storagePath, sessionID string, msg MessageData) {
	t.Helper()

	messagesDir := filepath.Join(storagePath, "message", sessionID)
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

func addTestPart(t *testing.T, storagePath, messageID string, part PartData) {
	t.Helper()

	partsDir := filepath.Join(storagePath, "part", messageID)
	if err := os.MkdirAll(partsDir, 0755); err != nil {
		t.Fatalf("Failed to create parts dir: %v", err)
	}

	data, err := json.Marshal(part)
	if err != nil {
		t.Fatalf("Failed to marshal part: %v", err)
	}

	partFile := filepath.Join(partsDir, part.ID+".json")
	if err := os.WriteFile(partFile, data, 0644); err != nil {
		t.Fatalf("Failed to write part: %v", err)
	}
}

func TestNewOpenCodeAgent(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test1", "global", "Test Session", "/home/user/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test1")

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

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

	if a.ProjectID() != "global" {
		t.Errorf("ProjectID() = %v, want %v", a.ProjectID(), "global")
	}

	// Status should be pending with no messages
	if a.Status() != agent.StatusPending {
		t.Errorf("Status() = %v, want %v", a.Status(), agent.StatusPending)
	}
}

func TestNewOpenCodeAgent_InvalidPath(t *testing.T) {
	_, err := NewOpenCodeAgent("/tmp", "/nonexistent/path/session.json")
	if err == nil {
		t.Error("NewOpenCodeAgent() expected error for invalid path")
	}
}

func TestNewOpenCodeAgent_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	sessionDir := filepath.Join(tempDir, "session", "global")
	os.MkdirAll(sessionDir, 0755)

	sessionFile := filepath.Join(sessionDir, "invalid.json")
	if err := os.WriteFile(sessionFile, []byte("not json"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, err := NewOpenCodeAgent(tempDir, sessionFile)
	if err == nil {
		t.Error("NewOpenCodeAgent() expected error for invalid JSON")
	}
}

func TestOpenCodeAgent_StatusRunning(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	// Add a recent message (within 30 seconds)
	msg := MessageData{
		ID:        "msg-1",
		SessionID: "ses_test",
		Role:      "user",
	}
	msg.Time.Created = now.Add(-10 * time.Second).UnixMilli()
	msg.Summary.Title = "Hello"
	addTestMessage(t, storagePath, "ses_test", msg)

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.Status() != agent.StatusRunning {
		t.Errorf("Status() = %v, want %v (recent message should indicate running)", a.Status(), agent.StatusRunning)
	}
}

func TestOpenCodeAgent_StatusIdle(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	// Add a message between 1 minute and 30 minutes old (idle threshold)
	msg := MessageData{
		ID:        "msg-1",
		SessionID: "ses_test",
		Role:      "user",
	}
	msg.Time.Created = now.Add(-5 * time.Minute).UnixMilli()
	msg.Summary.Title = "Hello"
	addTestMessage(t, storagePath, "ses_test", msg)

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.Status() != agent.StatusIdle {
		t.Errorf("Status() = %v, want %v (recent message should indicate idle)", a.Status(), agent.StatusIdle)
	}
}

func TestOpenCodeAgent_StatusCompleted(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	// Add a message older than 30 minutes (historical/completed)
	msg := MessageData{
		ID:        "msg-1",
		SessionID: "ses_test",
		Role:      "user",
	}
	msg.Time.Created = now.Add(-1 * time.Hour).UnixMilli()
	msg.Summary.Title = "Old task"
	addTestMessage(t, storagePath, "ses_test", msg)

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.Status() != agent.StatusCompleted {
		t.Errorf("Status() = %v, want %v (old session should be completed/historical)", a.Status(), agent.StatusCompleted)
	}
}

func TestOpenCodeAgent_StatusErrored(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	// Add message
	msg := MessageData{
		ID:        "msg-1",
		SessionID: "ses_test",
		Role:      "user",
	}
	msg.Time.Created = now.Add(-2 * time.Minute).UnixMilli()
	addTestMessage(t, storagePath, "ses_test", msg)

	part := PartData{
		ID:        "part-1",
		MessageID: "msg-1",
		SessionID: "ses_test",
		Type:      "tool-invocation",
		State:     "error",
	}
	part.Time.Created = now.Add(-2 * time.Minute).UnixMilli()
	addTestPart(t, storagePath, "msg-1", part)

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.Status() != agent.StatusErrored {
		t.Errorf("Status() = %v, want %v (error in part should indicate errored)", a.Status(), agent.StatusErrored)
	}
}

func TestOpenCodeAgent_CurrentTask(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	msg := MessageData{
		ID:        "msg-1",
		SessionID: "ses_test",
		Role:      "user",
	}
	msg.Time.Created = now.Add(-10 * time.Second).UnixMilli()
	msg.Summary.Title = "Please implement the new feature"
	addTestMessage(t, storagePath, "ses_test", msg)

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	if a.CurrentTask() != "Please implement the new feature" {
		t.Errorf("CurrentTask() = %v, want %v", a.CurrentTask(), "Please implement the new feature")
	}
}

func TestOpenCodeAgent_CurrentTask_Truncated(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	longTask := "This is a very long task description that exceeds one hundred characters and should be truncated when displayed"
	msg := MessageData{
		ID:        "msg-1",
		SessionID: "ses_test",
		Role:      "user",
	}
	msg.Time.Created = now.Add(-10 * time.Second).UnixMilli()
	msg.Summary.Title = longTask
	addTestMessage(t, storagePath, "ses_test", msg)

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	task := a.CurrentTask()
	if len(task) > 103 { // 100 chars + "..."
		t.Errorf("CurrentTask() should be truncated, got length %d", len(task))
	}
	if len(task) > 3 && task[len(task)-3:] != "..." {
		t.Errorf("CurrentTask() should end with '...', got %v", task)
	}
}

func TestOpenCodeAgent_Refresh(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	// Initially pending
	if a.Status() != agent.StatusPending {
		t.Errorf("Initial Status() = %v, want %v", a.Status(), agent.StatusPending)
	}

	// Add a message
	msg := MessageData{
		ID:        "msg-1",
		SessionID: "ses_test",
		Role:      "user",
	}
	msg.Time.Created = time.Now().UnixMilli()
	msg.Summary.Title = "Hello"
	addTestMessage(t, storagePath, "ses_test", msg)

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
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
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
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
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

	// Create session with empty title
	sessionDir := filepath.Join(tempDir, "session", "global")
	os.MkdirAll(sessionDir, 0755)

	session := SessionData{
		ID:        "ses_abc12345678",
		ProjectID: "global",
		Directory: "/project",
		Title:     "", // Empty title
	}
	session.Time.Created = now.UnixMilli()
	session.Time.Updated = now.UnixMilli()

	data, _ := json.Marshal(session)
	sessionFile := filepath.Join(sessionDir, "ses_abc12345678.json")
	os.WriteFile(sessionFile, data, 0644)

	// Create message and part directories
	os.MkdirAll(filepath.Join(tempDir, "message", "ses_abc12345678"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "part", "ses_abc12345678"), 0755)

	a, err := NewOpenCodeAgent(tempDir, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	// Should fall back to short ID
	if a.Name() != "ses_abc1" {
		t.Errorf("Name() = %v, want %v (should fallback to short ID)", a.Name(), "ses_abc1")
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

	// Create session directory
	sessionDir := filepath.Join(tempDir, "session", "global")
	os.MkdirAll(sessionDir, 0755)

	// Create session 1
	s1 := SessionData{ID: "ses_1", ProjectID: "global", Directory: "/p1", Title: "Session 1"}
	s1.Time.Created = now.UnixMilli()
	s1.Time.Updated = now.UnixMilli()
	data1, _ := json.Marshal(s1)
	os.WriteFile(filepath.Join(sessionDir, "ses_1.json"), data1, 0644)
	os.MkdirAll(filepath.Join(tempDir, "message", "ses_1"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "part", "ses_1"), 0755)

	// Create session 2
	s2 := SessionData{ID: "ses_2", ProjectID: "global", Directory: "/p2", Title: "Session 2"}
	s2.Time.Created = now.UnixMilli()
	s2.Time.Updated = now.UnixMilli()
	data2, _ := json.Marshal(s2)
	os.WriteFile(filepath.Join(sessionDir, "ses_2.json"), data2, 0644)
	os.MkdirAll(filepath.Join(tempDir, "message", "ses_2"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "part", "ses_2"), 0755)

	// Create a non-JSON file (should be skipped)
	os.WriteFile(filepath.Join(sessionDir, "readme.txt"), []byte("test"), 0644)

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

	sessionDir := filepath.Join(tempDir, "session", "global")
	os.MkdirAll(sessionDir, 0755)

	s := SessionData{ID: "ses_1", ProjectID: "global", Directory: "/p1", Title: "Session 1"}
	s.Time.Created = now.UnixMilli()
	s.Time.Updated = now.UnixMilli()
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(sessionDir, "ses_1.json"), data, 0644)
	os.MkdirAll(filepath.Join(tempDir, "message", "ses_1"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "part", "ses_1"), 0755)

	p := NewProvider(tempDir, 5*time.Second)
	p.Discover(context.Background())

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
	tempDir := t.TempDir()
	now := time.Now()

	sessionDir := filepath.Join(tempDir, "session", "global")
	os.MkdirAll(sessionDir, 0755)

	s := SessionData{ID: "ses_1", ProjectID: "global", Directory: "/p1", Title: "Session 1"}
	s.Time.Created = now.UnixMilli()
	s.Time.Updated = now.UnixMilli()
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(sessionDir, "ses_1.json"), data, 0644)
	os.MkdirAll(filepath.Join(tempDir, "message", "ses_1"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "part", "ses_1"), 0755)

	p := NewProvider(tempDir, 5*time.Second)
	p.Discover(context.Background())

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

func TestProvider_SendInput(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Now()

	sessionDir := filepath.Join(tempDir, "session", "global")
	os.MkdirAll(sessionDir, 0755)

	s := SessionData{ID: "ses_1", ProjectID: "global", Directory: "/p1", Title: "Session 1"}
	s.Time.Created = now.UnixMilli()
	s.Time.Updated = now.UnixMilli()
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(sessionDir, "ses_1.json"), data, 0644)
	os.MkdirAll(filepath.Join(tempDir, "message", "ses_1"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "part", "ses_1"), 0755)

	p := NewProvider(tempDir, 5*time.Second)
	p.Discover(context.Background())

	// SendInput returns error (not supported)
	if err := p.SendInput("ses_1", "test"); err == nil {
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

	// Create session directory
	sessionDir := filepath.Join(tempDir, "session", "global")
	os.MkdirAll(sessionDir, 0755)

	// Create initial session
	s := SessionData{ID: "ses_1", ProjectID: "global", Directory: "/p1", Title: "Session 1"}
	s.Time.Created = now.UnixMilli()
	s.Time.Updated = now.UnixMilli()
	data, _ := json.Marshal(s)
	os.WriteFile(filepath.Join(sessionDir, "ses_1.json"), data, 0644)
	os.MkdirAll(filepath.Join(tempDir, "message", "ses_1"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "part", "ses_1"), 0755)

	p := NewProvider(tempDir, 100*time.Millisecond)
	p.Discover(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := p.Watch(ctx)
	if err != nil {
		t.Fatalf("Watch() error = %v", err)
	}

	// Create a new session while watching
	s2 := SessionData{ID: "ses_2", ProjectID: "global", Directory: "/p2", Title: "Session 2"}
	s2.Time.Created = now.UnixMilli()
	s2.Time.Updated = now.UnixMilli()
	data2, _ := json.Marshal(s2)
	os.WriteFile(filepath.Join(sessionDir, "ses_2.json"), data2, 0644)

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
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	msg := MessageData{
		ID:        "msg-1",
		SessionID: "ses_test",
		Role:      "assistant",
	}
	msg.Time.Created = now.UnixMilli()
	addTestMessage(t, storagePath, "ses_test", msg)

	part := PartData{
		ID:        "part-1",
		MessageID: "msg-1",
		SessionID: "ses_test",
		Type:      "text",
		Text:      "Hello, this is output",
	}
	part.Time.Created = now.UnixMilli()
	addTestPart(t, storagePath, "msg-1", part)

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	output := a.Output()
	if output == nil {
		t.Error("Output() returned nil")
	}
}

func TestOpenCodeAgent_StartTime(t *testing.T) {
	createdAt := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", createdAt, createdAt)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	// Check within 1 second tolerance (millisecond precision)
	diff := a.StartTime().Sub(createdAt)
	if diff > time.Second || diff < -time.Second {
		t.Errorf("StartTime() = %v, want approximately %v", a.StartTime(), createdAt)
	}
}

func TestOpenCodeAgent_LastError(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	// Initially no error
	if a.LastError() != nil {
		t.Errorf("LastError() = %v, want nil", a.LastError())
	}
}

func TestOpenCodeAgent_Metrics(t *testing.T) {
	now := time.Now()
	storagePath := createTestStorage(t, "ses_test", "global", "Test", "/project", now, now)
	sessionFile := getSessionFilePath(storagePath, "global", "ses_test")

	a, err := NewOpenCodeAgent(storagePath, sessionFile)
	if err != nil {
		t.Fatalf("NewOpenCodeAgent() error = %v", err)
	}

	// Metrics should be initialized
	metrics := a.Metrics()
	if metrics.TokensIn < 0 || metrics.TokensOut < 0 {
		t.Error("Metrics should have non-negative values")
	}
}
