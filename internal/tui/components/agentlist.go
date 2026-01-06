// Package components provides TUI components
package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/localrivet/auto/internal/agent"
	"github.com/localrivet/auto/internal/session"
)

// AgentItem represents an agent in the list
type AgentItem struct {
	Agent agent.Agent
}

func (i AgentItem) Title() string {
	return i.Agent.Name()
}

func (i AgentItem) Description() string {
	task := i.Agent.CurrentTask()
	if task == "" {
		task = i.Agent.Directory()
	}

	ago := time.Since(i.Agent.LastActivity())
	var timeStr string
	if ago < time.Minute {
		timeStr = "just now"
	} else if ago < time.Hour {
		timeStr = fmt.Sprintf("%dm ago", int(ago.Minutes()))
	} else if ago < 24*time.Hour {
		timeStr = fmt.Sprintf("%dh ago", int(ago.Hours()))
	} else {
		timeStr = fmt.Sprintf("%dd ago", int(ago.Hours()/24))
	}

	return fmt.Sprintf("%s %s · %s", i.Agent.Status().Icon(), task, timeStr)
}

func (i AgentItem) FilterValue() string {
	return i.Agent.Name() + " " + i.Agent.CurrentTask()
}

// AgentList is the agent list component
type AgentList struct {
	list      list.Model
	theme     *Theme
	manager   *session.Manager
	groupMode session.GroupMode
	focused   bool
	width     int
	height    int
	selected  agent.Agent
}

// AgentListKeyMap defines keybindings for the agent list
type AgentListKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Select      key.Binding
	ToggleGroup key.Binding
	Terminate   key.Binding
	Pause       key.Binding
	Filter      key.Binding
	ClearFilter key.Binding
}

// DefaultAgentListKeyMap returns the default keybindings
func DefaultAgentListKeyMap() AgentListKeyMap {
	return AgentListKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		ToggleGroup: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "toggle grouping"),
		),
		Terminate: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "terminate"),
		),
		Pause: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "pause/resume"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
	}
}

// NewAgentList creates a new agent list component
func NewAgentList(theme *Theme, manager *session.Manager, width, height int) *AgentList {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = theme.SelectedItemStyle
	delegate.Styles.SelectedDesc = theme.SelectedItemStyle.Copy().Faint(true)
	delegate.Styles.NormalTitle = theme.NormalItemStyle
	delegate.Styles.NormalDesc = theme.NormalItemStyle.Copy().Faint(true)

	l := list.New([]list.Item{}, delegate, width, height)
	l.Title = "Agents"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = theme.Title
	l.Styles.FilterPrompt = theme.Base
	l.Styles.FilterCursor = theme.Base.Copy().Foreground(theme.Primary)

	return &AgentList{
		list:      l,
		theme:     theme,
		manager:   manager,
		groupMode: session.GroupModeFlat,
		width:     width,
		height:    height,
	}
}

// Init initializes the agent list
func (a *AgentList) Init() tea.Cmd {
	return a.refresh()
}

// refresh refreshes the agent list
func (a *AgentList) refresh() tea.Cmd {
	return func() tea.Msg {
		return AgentListRefreshMsg{}
	}
}

// AgentListRefreshMsg triggers a refresh
type AgentListRefreshMsg struct{}

// AgentSelectedMsg is sent when an agent is selected
type AgentSelectedMsg struct {
	Agent agent.Agent
}

// Update handles messages
func (a *AgentList) Update(msg tea.Msg) (*AgentList, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.list.FilterState() == list.Filtering {
			// Let the list handle filtering
			break
		}

		keys := DefaultAgentListKeyMap()
		switch {
		case key.Matches(msg, keys.ToggleGroup):
			a.cycleGroupMode()
			return a, a.refresh()

		case key.Matches(msg, keys.Terminate):
			if a.selected != nil {
				a.manager.Terminate(a.selected.ID())
				return a, a.refresh()
			}

		case key.Matches(msg, keys.Select):
			if item, ok := a.list.SelectedItem().(AgentItem); ok {
				a.selected = item.Agent
				return a, func() tea.Msg {
					return AgentSelectedMsg{Agent: item.Agent}
				}
			}
		}

	case AgentListRefreshMsg:
		a.updateItems()

	case agent.Event:
		a.updateItems()
		// If the event is for the selected agent, send an update
		if a.selected != nil && msg.AgentID == a.selected.ID() && msg.Agent != nil {
			a.selected = msg.Agent
			cmds = append(cmds, func() tea.Msg {
				return AgentSelectedMsg{Agent: msg.Agent}
			})
		}
	}

	var cmd tea.Cmd
	a.list, cmd = a.list.Update(msg)
	cmds = append(cmds, cmd)

	// Update selected agent from list
	if item, ok := a.list.SelectedItem().(AgentItem); ok {
		if a.selected == nil || a.selected.ID() != item.Agent.ID() {
			a.selected = item.Agent
		}
	}

	return a, tea.Batch(cmds...)
}

// updateItems updates the list items from the manager
func (a *AgentList) updateItems() {
	agents := a.manager.List()

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].LastActivity().After(agents[j].LastActivity())
	})

	items := make([]list.Item, len(agents))
	for i, ag := range agents {
		items[i] = AgentItem{Agent: ag}
	}

	a.list.SetItems(items)
}

// cycleGroupMode cycles through group modes
func (a *AgentList) cycleGroupMode() {
	switch a.groupMode {
	case session.GroupModeFlat:
		a.groupMode = session.GroupModeType
	case session.GroupModeType:
		a.groupMode = session.GroupModeProject
	case session.GroupModeProject:
		a.groupMode = session.GroupModeStatus
	default:
		a.groupMode = session.GroupModeFlat
	}
}

// View renders the agent list
func (a *AgentList) View() string {
	style := a.theme.AgentListStyle.Width(a.width).Height(a.height)
	if a.focused {
		style = a.theme.FocusedBorder(style)
	}

	// Build header with stats
	stats := a.manager.Stats()
	header := fmt.Sprintf("Agents (%d)", stats.Total)
	if stats.ByStatus[agent.StatusRunning] > 0 {
		header += fmt.Sprintf(" | %s %d running",
			a.theme.StatusStyle(agent.StatusRunning).Render("●"),
			stats.ByStatus[agent.StatusRunning])
	}
	if stats.ByStatus[agent.StatusErrored] > 0 {
		header += fmt.Sprintf(" | %s %d errors",
			a.theme.StatusStyle(agent.StatusErrored).Render("●"),
			stats.ByStatus[agent.StatusErrored])
	}

	a.list.Title = header

	return style.Render(a.list.View())
}

// SetSize sets the component size
func (a *AgentList) SetSize(width, height int) {
	a.width = width
	a.height = height
	a.list.SetSize(width-4, height-4) // Account for borders
}

// SetFocused sets the focus state
func (a *AgentList) SetFocused(focused bool) {
	a.focused = focused
}

// IsFocused returns the focus state
func (a *AgentList) IsFocused() bool {
	return a.focused
}

// Selected returns the currently selected agent
func (a *AgentList) Selected() agent.Agent {
	return a.selected
}

// SetSelected sets the selected agent by ID
func (a *AgentList) SetSelected(id string) {
	for i, item := range a.list.Items() {
		if ai, ok := item.(AgentItem); ok && ai.Agent.ID() == id {
			a.list.Select(i)
			a.selected = ai.Agent
			return
		}
	}
}

// FilterValue returns the current filter value
func (a *AgentList) FilterValue() string {
	return a.list.FilterValue()
}

// GroupMode returns the current group mode
func (a *AgentList) GroupMode() session.GroupMode {
	return a.groupMode
}

// SetGroupMode sets the group mode
func (a *AgentList) SetGroupMode(mode session.GroupMode) {
	a.groupMode = mode
}

// StatusIndicator returns a styled status indicator
func StatusIndicator(theme *Theme, status agent.Status) string {
	icon := status.Icon()
	style := theme.StatusStyle(status)
	return style.Render(icon)
}

// FormatAgentLine formats an agent for display
func FormatAgentLine(theme *Theme, ag agent.Agent, width int) string {
	status := StatusIndicator(theme, ag.Status())
	name := ag.Name()
	task := ag.CurrentTask()

	// Truncate if needed
	maxNameLen := width / 3
	if len(name) > maxNameLen {
		name = name[:maxNameLen-3] + "..."
	}

	maxTaskLen := width - len(name) - 5
	if len(task) > maxTaskLen && maxTaskLen > 3 {
		task = task[:maxTaskLen-3] + "..."
	}

	return fmt.Sprintf("%s %s %s",
		status,
		theme.Base.Bold(true).Render(name),
		theme.Base.Faint(true).Render(task))
}

// RenderGrouped renders agents in groups
func RenderGrouped(theme *Theme, groups []session.Group, width int) string {
	var b strings.Builder

	for _, group := range groups {
		// Group header
		b.WriteString(theme.Title.Render(fmt.Sprintf("─── %s (%d) ───", group.Name, len(group.Agents))))
		b.WriteString("\n")

		for _, ag := range group.Agents {
			b.WriteString(FormatAgentLine(theme, ag, width))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	return b.String()
}
