package components

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/CastAIPhil/AUTO/internal/agent"
	"github.com/CastAIPhil/AUTO/internal/session"
)

func newBenchMockAgent(id, name string, status agent.Status) *mockAgent {
	return &mockAgent{
		id:           id,
		name:         name,
		agentType:    "opencode",
		directory:    "/home/user/project",
		projectID:    "proj-1",
		status:       status,
		startTime:    time.Now().Add(-time.Hour),
		lastActivity: time.Now().Add(-time.Duration(id[len(id)-1]) * time.Minute),
		currentTask:  fmt.Sprintf("Working on task for %s", name),
		metrics:      agent.Metrics{TokensIn: 1000, TokensOut: 500},
	}
}

func createMockAgents(n int) []agent.Agent {
	agents := make([]agent.Agent, n)
	for i := 0; i < n; i++ {
		status := agent.Status(i % 5)
		agents[i] = newBenchMockAgent(fmt.Sprintf("agent-%d", i), fmt.Sprintf("Agent %d", i), status)
	}
	return agents
}

func BenchmarkFilterApply(b *testing.B) {
	sizes := []int{10, 50, 100, 500}

	for _, size := range sizes {
		agents := createMockAgents(size)

		b.Run(fmt.Sprintf("agents=%d/no_filter", size), func(b *testing.B) {
			f := NewFilter()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.Apply(agents)
			}
		})

		b.Run(fmt.Sprintf("agents=%d/status_filter", size), func(b *testing.B) {
			f := NewFilter()
			f.statusFilter = agent.StatusRunning
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.Apply(agents)
			}
		})

		b.Run(fmt.Sprintf("agents=%d/search_filter", size), func(b *testing.B) {
			f := NewFilter()
			f.searchInput.SetValue("Agent")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = f.Apply(agents)
			}
		})
	}
}

func BenchmarkAgentItemTitle(b *testing.B) {
	mockAg := newBenchMockAgent("agent-1", "Test Agent", agent.StatusRunning)

	b.Run("no_children", func(b *testing.B) {
		item := AgentItem{Agent: mockAg, ChildCount: 0}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = item.Title()
		}
	})

	b.Run("with_children", func(b *testing.B) {
		item := AgentItem{Agent: mockAg, ChildCount: 5}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = item.Title()
		}
	})
}

func BenchmarkAgentItemDescription(b *testing.B) {
	mockAg := newBenchMockAgent("agent-1", "Test Agent", agent.StatusRunning)
	item := AgentItem{Agent: mockAg, ChildCount: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = item.Description()
	}
}

func BenchmarkAgentItemFilterValue(b *testing.B) {
	mockAg := newBenchMockAgent("agent-1", "Test Agent", agent.StatusRunning)
	item := AgentItem{Agent: mockAg, ChildCount: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = item.FilterValue()
	}
}

func BenchmarkStatusIndicator(b *testing.B) {
	theme := DefaultDarkTheme()
	statuses := []agent.Status{
		agent.StatusRunning,
		agent.StatusIdle,
		agent.StatusCompleted,
		agent.StatusErrored,
		agent.StatusPending,
	}

	for _, status := range statuses {
		b.Run(status.String(), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = StatusIndicator(theme, status)
			}
		})
	}
}

func BenchmarkFormatAgentLine(b *testing.B) {
	theme := DefaultDarkTheme()
	mockAg := newBenchMockAgent("agent-1", "Test Agent with a longer name", agent.StatusRunning)

	widths := []int{40, 80, 120}
	for _, width := range widths {
		b.Run(fmt.Sprintf("width=%d", width), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = FormatAgentLine(theme, mockAg, width)
			}
		})
	}
}

func BenchmarkFormatDuration(b *testing.B) {
	durations := []time.Duration{
		30 * time.Second,
		5 * time.Minute,
		2 * time.Hour,
		48 * time.Hour,
	}

	for _, d := range durations {
		b.Run(d.String(), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = FormatDuration(d)
			}
		})
	}
}

func BenchmarkThemeStatusStyle(b *testing.B) {
	theme := DefaultDarkTheme()

	b.Run("known_status", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = theme.StatusStyle(agent.StatusRunning)
		}
	})

	b.Run("unknown_status", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = theme.StatusStyle(agent.Status(999))
		}
	})
}

func BenchmarkCommandPaletteFilter(b *testing.B) {
	theme := DefaultDarkTheme()
	commands := make([]Command, 50)
	for i := 0; i < 50; i++ {
		commands[i] = Command{
			Name:        fmt.Sprintf("Command%d", i),
			Description: fmt.Sprintf("Description for command %d", i),
			Keys:        fmt.Sprintf("c%d", i),
		}
	}

	queries := []string{"", "com", "command1", "nonexistent"}

	for _, q := range queries {
		b.Run(fmt.Sprintf("query=%s", q), func(b *testing.B) {
			cp := NewCommandPalette(theme, commands)
			cp.input.SetValue(q)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cp.filter()
			}
		})
	}
}

func BenchmarkHelpScreenView(b *testing.B) {
	theme := DefaultDarkTheme()
	h := NewHelpScreen(theme)
	h.SetSize(80, 40)
	h.Show()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.View()
	}
}

func BenchmarkShortHelp(b *testing.B) {
	theme := DefaultDarkTheme()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ShortHelp(theme)
	}
}

func BenchmarkSpawnDialogView(b *testing.B) {
	d := NewSpawnDialog()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = d.View()
	}
}

func BenchmarkMiniStats(b *testing.B) {
	theme := DefaultDarkTheme()

	stats := &session.Stats{
		Total: 42,
		ByStatus: map[agent.Status]int{
			agent.StatusRunning:   10,
			agent.StatusIdle:      20,
			agent.StatusCompleted: 10,
			agent.StatusErrored:   2,
		},
		ByType:         map[string]int{"opencode": 42},
		ByProject:      map[string]int{"proj-1": 42},
		TotalTokensIn:  100000,
		TotalTokensOut: 50000,
		TotalCost:      5.50,
		TotalToolCalls: 200,
		TotalErrors:    5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = MiniStats(theme, stats)
	}
}

func BenchmarkInputBarView(b *testing.B) {
	theme := DefaultDarkTheme()
	input := NewInputBar(theme, 80)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = input.View()
	}
}

func BenchmarkViewportView(b *testing.B) {
	theme := DefaultDarkTheme()
	vp := NewSessionViewport(theme, 80, 40)

	mockAg := newBenchMockAgent("agent-1", "Test Agent", agent.StatusRunning)
	vp.SetAgent(mockAg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = vp.View()
	}
}

func BenchmarkViewportSetAgent(b *testing.B) {
	theme := DefaultDarkTheme()
	vp := NewSessionViewport(theme, 80, 40)

	mockAg := newBenchMockAgent("agent-1", "Test Agent", agent.StatusRunning)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vp.SetAgent(mockAg)
	}
}

func BenchmarkViewportWithLargeContent(b *testing.B) {
	theme := DefaultDarkTheme()

	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("lines=%d", size), func(b *testing.B) {
			vp := NewSessionViewport(theme, 80, 40)
			mockAg := &mockAgentWithOutput{
				mockAgent: newBenchMockAgent("agent-1", "Test Agent", agent.StatusRunning),
				output:    strings.Repeat("Line of output content here\n", size),
			}
			vp.SetAgent(mockAg)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = vp.View()
			}
		})
	}
}

type mockAgentWithOutput struct {
	*mockAgent
	output string
}

func (m *mockAgentWithOutput) Output() io.Reader {
	return strings.NewReader(m.output)
}

func BenchmarkRenderGrouped(b *testing.B) {
	theme := DefaultDarkTheme()

	groups := []session.Group{
		{
			Name:   "opencode",
			Agents: createMockAgents(20),
		},
		{
			Name:   "claude",
			Agents: createMockAgents(15),
		},
		{
			Name:   "other",
			Agents: createMockAgents(10),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RenderGrouped(theme, groups, 80)
	}
}

func BenchmarkFormatNumber(b *testing.B) {
	numbers := []int64{100, 5000, 50000, 1000000, 50000000}

	for _, n := range numbers {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = formatNumber(n)
			}
		})
	}
}
