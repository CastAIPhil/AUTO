package store

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func setupBenchStore(b *testing.B) *Store {
	b.Helper()
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	store, err := New(dbPath)
	if err != nil {
		b.Fatalf("Failed to create store: %v", err)
	}
	b.Cleanup(func() { store.Close() })

	return store
}

func setupBenchStoreWithData(b *testing.B, numSessions, numAlerts int) *Store {
	b.Helper()
	store := setupBenchStore(b)

	for i := 0; i < numSessions; i++ {
		store.SaveSession(&SessionRecord{
			ID:           fmt.Sprintf("session-%d", i),
			AgentID:      fmt.Sprintf("agent-%d", i%10),
			AgentType:    "opencode",
			AgentName:    fmt.Sprintf("Agent %d", i),
			Directory:    fmt.Sprintf("/home/user/project-%d", i),
			ProjectID:    fmt.Sprintf("proj-%d", i%5),
			Status:       "running",
			StartTime:    time.Now().Add(-time.Duration(i) * time.Hour),
			LastActivity: time.Now(),
			TokensIn:     int64(i * 100),
			TokensOut:    int64(i * 50),
		})
	}

	for i := 0; i < numAlerts; i++ {
		store.SaveAlert(&AlertRecord{
			ID:        fmt.Sprintf("alert-%d", i),
			AgentID:   fmt.Sprintf("agent-%d", i%10),
			Level:     "info",
			Message:   fmt.Sprintf("Alert message %d", i),
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			Read:      i%2 == 0,
		})
	}

	return store
}

func BenchmarkStoreSaveSession(b *testing.B) {
	store := setupBenchStore(b)
	session := &SessionRecord{
		ID:           "bench-session",
		AgentID:      "agent-1",
		AgentType:    "opencode",
		AgentName:    "Benchmark Agent",
		Directory:    "/home/user/project",
		ProjectID:    "proj-1",
		Status:       "running",
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		TokensIn:     1000,
		TokensOut:    500,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.ID = fmt.Sprintf("session-%d", i)
		_ = store.SaveSession(session)
	}
}

func BenchmarkStoreGetSession(b *testing.B) {
	store := setupBenchStoreWithData(b, 100, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetSession(fmt.Sprintf("session-%d", i%100))
	}
}

func BenchmarkStoreListSessions(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("sessions=%d", size), func(b *testing.B) {
			store := setupBenchStoreWithData(b, size, 0)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = store.ListSessions(0, "")
			}
		})
	}
}

func BenchmarkStoreListSessionsWithLimit(b *testing.B) {
	store := setupBenchStoreWithData(b, 100, 0)

	limits := []int{10, 25, 50}

	for _, limit := range limits {
		b.Run(fmt.Sprintf("limit=%d", limit), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = store.ListSessions(limit, "")
			}
		})
	}
}

func BenchmarkStoreListSessionsWithStatus(b *testing.B) {
	store := setupBenchStoreWithData(b, 100, 0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListSessions(0, "running")
	}
}

func BenchmarkStoreSaveAlert(b *testing.B) {
	store := setupBenchStore(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.SaveAlert(&AlertRecord{
			ID:        fmt.Sprintf("alert-%d", i),
			AgentID:   "agent-1",
			Level:     "info",
			Message:   "Benchmark alert",
			Timestamp: time.Now(),
			Read:      false,
		})
	}
}

func BenchmarkStoreListAlerts(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("alerts=%d", size), func(b *testing.B) {
			store := setupBenchStoreWithData(b, 0, size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = store.ListAlerts(0, false)
			}
		})
	}
}

func BenchmarkStoreListAlertsUnreadOnly(b *testing.B) {
	store := setupBenchStoreWithData(b, 0, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListAlerts(0, true)
	}
}

func BenchmarkStoreGetStats(b *testing.B) {
	store := setupBenchStoreWithData(b, 100, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetStats()
	}
}

func BenchmarkStoreAppendOutput(b *testing.B) {
	store := setupBenchStore(b)
	store.SaveSession(&SessionRecord{
		ID:        "output-session",
		AgentID:   "agent-1",
		AgentType: "opencode",
		AgentName: "Output Agent",
		Status:    "running",
		StartTime: time.Now(),
	})

	chunk := "This is a sample output chunk that might be written frequently. "

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.AppendOutput("output-session", chunk)
	}
}

func BenchmarkStoreGetOutput(b *testing.B) {
	store := setupBenchStore(b)
	store.SaveSession(&SessionRecord{
		ID:        "output-session",
		AgentID:   "agent-1",
		AgentType: "opencode",
		AgentName: "Output Agent",
		Status:    "running",
		StartTime: time.Now(),
	})

	chunk := "This is a sample output chunk. "
	for i := 0; i < 100; i++ {
		store.AppendOutput("output-session", chunk)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetOutput("output-session")
	}
}

func BenchmarkStoreMarkAlertRead(b *testing.B) {
	store := setupBenchStoreWithData(b, 0, b.N+100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.MarkAlertRead(fmt.Sprintf("alert-%d", i%100))
	}
}
