package session

import (
	"context"
	"fmt"
	"testing"

	"github.com/CastAIPhil/AUTO/internal/agent"
	"github.com/CastAIPhil/AUTO/internal/config"
)

func setupManagerWithAgents(n int) *Manager {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	m.mu.Lock()
	for i := 0; i < n; i++ {
		a := agent.NewMockAgent(fmt.Sprintf("agent-%d", i), fmt.Sprintf("Agent %d", i))
		a.MockType = "opencode"
		a.MockProjectID = fmt.Sprintf("proj-%d", i%10)
		a.MockStatus = agent.Status(i % 5)
		a.MockDirectory = fmt.Sprintf("/home/user/project-%d", i)
		a.MockCurrentTask = fmt.Sprintf("Working on task %d", i)
		a.MockMetrics.TokensIn = int64(i * 100)
		a.MockMetrics.TokensOut = int64(i * 50)
		m.agents[a.ID()] = a
	}
	m.mu.Unlock()

	return m
}

func BenchmarkManagerList(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("agents=%d", size), func(b *testing.B) {
			m := setupManagerWithAgents(size)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = m.List()
			}
		})
	}
}

func BenchmarkManagerGet(b *testing.B) {
	m := setupManagerWithAgents(100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = m.Get(fmt.Sprintf("agent-%d", i%100))
	}
}

func BenchmarkManagerStats(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("agents=%d", size), func(b *testing.B) {
			m := setupManagerWithAgents(size)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = m.Stats()
			}
		})
	}
}

func BenchmarkManagerGroupBy(b *testing.B) {
	m := setupManagerWithAgents(100)

	modes := []struct {
		name string
		mode GroupMode
	}{
		{"flat", GroupModeFlat},
		{"type", GroupModeType},
		{"project", GroupModeProject},
		{"status", GroupModeStatus},
	}

	for _, mode := range modes {
		b.Run(mode.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = m.GroupBy(mode.mode)
			}
		})
	}
}

func BenchmarkManagerFilterByStatus(b *testing.B) {
	m := setupManagerWithAgents(100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = m.FilterByStatus(agent.StatusRunning)
	}
}

func BenchmarkManagerSearch(b *testing.B) {
	m := setupManagerWithAgents(100)

	queries := []string{"Agent", "task", "project-5", "nonexistent"}

	for _, q := range queries {
		b.Run(fmt.Sprintf("query=%s", q), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = m.Search(q)
			}
		})
	}
}

func BenchmarkManagerActiveCount(b *testing.B) {
	m := setupManagerWithAgents(100)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = m.ActiveCount()
	}
}

func BenchmarkManagerRefresh(b *testing.B) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	m := NewManager(cfg, nil, registry, nil)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = m.Refresh(ctx)
	}
}

func BenchmarkContainsIgnoreCase(b *testing.B) {
	cases := []struct {
		name string
		s    string
		sub  string
	}{
		{"short_match", "Hello World", "world"},
		{"short_nomatch", "Hello World", "foo"},
		{"long_match", "The quick brown fox jumps over the lazy dog", "lazy"},
		{"long_nomatch", "The quick brown fox jumps over the lazy dog", "cat"},
	}

	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = containsIgnoreCase(c.s, c.sub)
			}
		})
	}
}
