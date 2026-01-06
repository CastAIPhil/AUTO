package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/localrivet/auto/internal/agent"
	"github.com/localrivet/auto/internal/session"
)

// StatsPanel displays aggregate statistics
type StatsPanel struct {
	theme   *Theme
	manager *session.Manager
	focused bool
	width   int
	height  int
}

// NewStatsPanel creates a new stats panel
func NewStatsPanel(theme *Theme, manager *session.Manager, width, height int) *StatsPanel {
	return &StatsPanel{
		theme:   theme,
		manager: manager,
		width:   width,
		height:  height,
	}
}

// Init initializes the stats panel
func (s *StatsPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (s *StatsPanel) Update(msg tea.Msg) (*StatsPanel, tea.Cmd) {
	// Stats panel doesn't handle input directly
	return s, nil
}

// View renders the stats panel
func (s *StatsPanel) View() string {
	style := s.theme.StatsStyle.Width(s.width).Height(s.height)
	if s.focused {
		style = s.theme.FocusedBorder(style)
	}

	stats := s.manager.Stats()

	var b strings.Builder

	// Title
	b.WriteString(s.theme.Title.Render("Statistics"))
	b.WriteString("\n\n")

	// Status breakdown
	b.WriteString(s.theme.Subtitle.Render("Agents"))
	b.WriteString("\n")
	b.WriteString(s.renderStatusBar(stats))
	b.WriteString("\n\n")

	// Counts
	b.WriteString(fmt.Sprintf("  Total:     %d\n", stats.Total))
	b.WriteString(fmt.Sprintf("  %s Running:  %d\n",
		s.theme.StatusStyle(agent.StatusRunning).Render("●"),
		stats.ByStatus[agent.StatusRunning]))
	b.WriteString(fmt.Sprintf("  %s Idle:     %d\n",
		s.theme.StatusStyle(agent.StatusIdle).Render("●"),
		stats.ByStatus[agent.StatusIdle]))
	b.WriteString(fmt.Sprintf("  %s Done:     %d\n",
		s.theme.StatusStyle(agent.StatusCompleted).Render("●"),
		stats.ByStatus[agent.StatusCompleted]))
	b.WriteString(fmt.Sprintf("  %s Errors:   %d\n",
		s.theme.StatusStyle(agent.StatusErrored).Render("●"),
		stats.ByStatus[agent.StatusErrored]))
	b.WriteString("\n")

	// Token usage
	b.WriteString(s.theme.Subtitle.Render("Usage"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Tokens In:  %s\n", formatNumber(stats.TotalTokensIn)))
	b.WriteString(fmt.Sprintf("  Tokens Out: %s\n", formatNumber(stats.TotalTokensOut)))
	b.WriteString(fmt.Sprintf("  Est. Cost:  $%.2f\n", stats.TotalCost))
	b.WriteString(fmt.Sprintf("  Tool Calls: %d\n", stats.TotalToolCalls))
	b.WriteString(fmt.Sprintf("  Errors:     %d\n", stats.TotalErrors))

	// Types breakdown if multiple types
	if len(stats.ByType) > 1 {
		b.WriteString("\n")
		b.WriteString(s.theme.Subtitle.Render("By Type"))
		b.WriteString("\n")
		for t, count := range stats.ByType {
			b.WriteString(fmt.Sprintf("  %s: %d\n", t, count))
		}
	}

	// Projects breakdown if multiple projects
	if len(stats.ByProject) > 1 && len(stats.ByProject) <= 5 {
		b.WriteString("\n")
		b.WriteString(s.theme.Subtitle.Render("By Project"))
		b.WriteString("\n")
		for p, count := range stats.ByProject {
			name := p
			if len(name) > 15 {
				name = name[:15] + "..."
			}
			b.WriteString(fmt.Sprintf("  %s: %d\n", name, count))
		}
	}

	return style.Render(b.String())
}

// renderStatusBar renders a visual status bar
func (s *StatsPanel) renderStatusBar(stats *session.Stats) string {
	if stats.Total == 0 {
		return s.theme.Base.Faint(true).Render("  [no agents]")
	}

	barWidth := s.width - 6
	if barWidth < 10 {
		barWidth = 10
	}

	// Calculate widths
	running := stats.ByStatus[agent.StatusRunning]
	idle := stats.ByStatus[agent.StatusIdle]
	done := stats.ByStatus[agent.StatusCompleted]
	errored := stats.ByStatus[agent.StatusErrored]
	other := stats.Total - running - idle - done - errored

	runningWidth := (running * barWidth) / stats.Total
	idleWidth := (idle * barWidth) / stats.Total
	doneWidth := (done * barWidth) / stats.Total
	erroredWidth := (errored * barWidth) / stats.Total
	otherWidth := barWidth - runningWidth - idleWidth - doneWidth - erroredWidth
	_ = other // suppress unused warning

	var bar strings.Builder
	bar.WriteString("  ")

	if runningWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().
			Background(s.theme.StatusRunning).
			Render(strings.Repeat(" ", runningWidth)))
	}
	if idleWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().
			Background(s.theme.StatusIdle).
			Render(strings.Repeat(" ", idleWidth)))
	}
	if doneWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().
			Background(s.theme.StatusDone).
			Render(strings.Repeat(" ", doneWidth)))
	}
	if erroredWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().
			Background(s.theme.StatusError).
			Render(strings.Repeat(" ", erroredWidth)))
	}
	if otherWidth > 0 {
		bar.WriteString(lipgloss.NewStyle().
			Background(s.theme.Border).
			Render(strings.Repeat(" ", otherWidth)))
	}

	return bar.String()
}

// SetSize sets the component size
func (s *StatsPanel) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetFocused sets the focus state
func (s *StatsPanel) SetFocused(focused bool) {
	s.focused = focused
}

// IsFocused returns the focus state
func (s *StatsPanel) IsFocused() bool {
	return s.focused
}

// formatNumber formats a number with K/M suffix
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

// MiniStats renders a compact one-line stats display
func MiniStats(theme *Theme, stats *session.Stats) string {
	parts := []string{
		fmt.Sprintf("%d agents", stats.Total),
	}

	if stats.ByStatus[agent.StatusRunning] > 0 {
		parts = append(parts, fmt.Sprintf("%s%d running",
			theme.StatusStyle(agent.StatusRunning).Render("●"),
			stats.ByStatus[agent.StatusRunning]))
	}

	if stats.ByStatus[agent.StatusErrored] > 0 {
		parts = append(parts, fmt.Sprintf("%s%d errors",
			theme.StatusStyle(agent.StatusErrored).Render("●"),
			stats.ByStatus[agent.StatusErrored]))
	}

	if stats.TotalCost > 0 {
		parts = append(parts, fmt.Sprintf("$%.2f", stats.TotalCost))
	}

	return strings.Join(parts, " | ")
}
