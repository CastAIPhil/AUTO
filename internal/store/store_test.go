package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	if store.db == nil {
		t.Error("Store db should not be nil")
	}
}

func TestSessionOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	session := &SessionRecord{
		ID:           "test-session-1",
		AgentID:      "agent-1",
		AgentType:    "opencode",
		AgentName:    "Test Agent",
		Directory:    "/tmp/test",
		ProjectID:    "project-1",
		Status:       "running",
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		TokensIn:     100,
		TokensOut:    200,
	}

	if err := store.SaveSession(session); err != nil {
		t.Fatalf("Failed to save session: %v", err)
	}

	loaded, err := store.GetSession("test-session-1")
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}

	if loaded.AgentName != "Test Agent" {
		t.Errorf("Loaded agent name should be 'Test Agent', got %v", loaded.AgentName)
	}

	if loaded.TokensIn != 100 {
		t.Errorf("Loaded tokens in should be 100, got %v", loaded.TokensIn)
	}

	sessions, err := store.ListSessions(10, "")
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}

	if len(sessions) != 1 {
		t.Errorf("Should have 1 session, got %d", len(sessions))
	}
}

func TestAlertOperations(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	alert := &AlertRecord{
		ID:        "alert-1",
		AgentID:   "agent-1",
		Level:     "error",
		Message:   "Test alert",
		Timestamp: time.Now(),
		Read:      false,
	}

	if err := store.SaveAlert(alert); err != nil {
		t.Fatalf("Failed to save alert: %v", err)
	}

	alerts, err := store.ListAlerts(10, false)
	if err != nil {
		t.Fatalf("Failed to list alerts: %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Should have 1 alert, got %d", len(alerts))
	}

	if err := store.MarkAlertRead("alert-1"); err != nil {
		t.Fatalf("Failed to mark alert read: %v", err)
	}

	unreadAlerts, err := store.ListAlerts(10, true)
	if err != nil {
		t.Fatalf("Failed to list unread alerts: %v", err)
	}

	if len(unreadAlerts) != 0 {
		t.Errorf("Should have 0 unread alerts, got %d", len(unreadAlerts))
	}
}

func TestGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	stats, err := store.GetStats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["total_sessions"].(int) != 0 {
		t.Errorf("Initial total sessions should be 0")
	}
}
