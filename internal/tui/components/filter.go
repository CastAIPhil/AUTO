package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/CastAIPhil/AUTO/internal/agent"
)

type FilterMode int

const (
	FilterModeNone FilterMode = iota
	FilterModeSearch
	FilterModeStatus
)

type Filter struct {
	searchInput   textinput.Model
	mode          FilterMode
	statusFilter  agent.Status
	statusIndex   int
	width         int
	searchActive  bool
	totalCount    int
	filteredCount int
}

var statusFilters = []agent.Status{
	-1, // All (no filter)
	agent.StatusRunning,
	agent.StatusIdle,
	agent.StatusCompleted,
	agent.StatusErrored,
	agent.StatusContextLimit,
	agent.StatusPending,
}

func NewFilter() Filter {
	ti := textinput.New()
	ti.Placeholder = "Search agents..."
	ti.Width = 30

	return Filter{
		searchInput:  ti,
		mode:         FilterModeNone,
		statusFilter: -1,
		statusIndex:  0,
	}
}

func (f Filter) Init() tea.Cmd {
	return nil
}

func (f Filter) Update(msg tea.Msg) (Filter, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if f.searchActive {
			switch msg.String() {
			case "esc":
				f.searchActive = false
				f.searchInput.Blur()
				f.searchInput.SetValue("")
				return f, nil
			case "enter":
				f.searchActive = false
				f.searchInput.Blur()
				return f, nil
			default:
				f.searchInput, cmd = f.searchInput.Update(msg)
				return f, cmd
			}
		}

		switch msg.String() {
		case "/":
			f.searchActive = true
			f.searchInput.Focus()
			return f, textinput.Blink
		case "f":
			f.statusIndex = (f.statusIndex + 1) % len(statusFilters)
			f.statusFilter = statusFilters[f.statusIndex]
			return f, nil
		case "F":
			f.statusIndex = (f.statusIndex - 1 + len(statusFilters)) % len(statusFilters)
			f.statusFilter = statusFilters[f.statusIndex]
			return f, nil
		case "esc":
			f.statusFilter = -1
			f.statusIndex = 0
			f.searchInput.SetValue("")
			return f, nil
		}
	}

	if f.searchActive {
		f.searchInput, cmd = f.searchInput.Update(msg)
	}

	return f, cmd
}

func (f Filter) View() string {
	if !f.IsActive() && f.searchInput.Value() == "" {
		return ""
	}

	var parts []string

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	if f.searchActive {
		parts = append(parts, f.searchInput.View())
	} else if f.searchInput.Value() != "" {
		parts = append(parts, infoStyle.Render("Search: ")+f.searchInput.Value())
	}

	if f.statusFilter >= 0 {
		parts = append(parts, activeStyle.Render("Status: ")+f.statusFilter.String())
	}

	if f.totalCount > 0 && f.filteredCount != f.totalCount {
		parts = append(parts, infoStyle.Render(fmt.Sprintf("(%d/%d)", f.filteredCount, f.totalCount)))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " | ")
}

func (f Filter) IsActive() bool {
	return f.searchActive || f.statusFilter >= 0 || f.searchInput.Value() != ""
}

func (f Filter) SearchQuery() string {
	return f.searchInput.Value()
}

func (f Filter) StatusFilter() agent.Status {
	return f.statusFilter
}

func (f Filter) IsSearchActive() bool {
	return f.searchActive
}

func (f *Filter) SetCounts(filtered, total int) {
	f.filteredCount = filtered
	f.totalCount = total
}

func (f Filter) Apply(agents []agent.Agent) []agent.Agent {
	result := agents

	// Apply status filter
	if f.statusFilter >= 0 {
		var filtered []agent.Agent
		for _, a := range result {
			if a.Status() == f.statusFilter {
				filtered = append(filtered, a)
			}
		}
		result = filtered
	}

	// Apply search filter
	query := strings.ToLower(f.searchInput.Value())
	if query != "" {
		var filtered []agent.Agent
		for _, a := range result {
			if strings.Contains(strings.ToLower(a.Name()), query) ||
				strings.Contains(strings.ToLower(a.ID()), query) ||
				strings.Contains(strings.ToLower(a.Directory()), query) ||
				strings.Contains(strings.ToLower(a.CurrentTask()), query) {
				filtered = append(filtered, a)
			}
		}
		result = filtered
	}

	return result
}

func (f Filter) HelpText() string {
	if f.searchActive {
		return "Type to search | Enter: confirm | Esc: cancel"
	}
	return "/ search | f/F cycle status | Esc: clear filters"
}
