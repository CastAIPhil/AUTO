package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/localrivet/auto/internal/agent"
	"github.com/localrivet/auto/internal/alert"
	"github.com/localrivet/auto/internal/config"
	"github.com/localrivet/auto/internal/session"
	"github.com/localrivet/auto/internal/tui/components"
)

type Theme = components.Theme

// Pane represents the focused pane
type Pane int

const (
	PaneAgentList Pane = iota
	PaneViewport
	PaneStats
	PaneAlerts
)

// App is the main TUI application model
type App struct {
	cfg      *config.Config
	theme    *Theme
	manager  *session.Manager
	alertMgr *alert.Manager

	agentList *components.AgentList
	viewport  *components.SessionViewport
	stats     *components.StatsPanel
	alerts    *components.AlertsPanel
	input     *components.InputBar
	command   *components.CommandPalette
	help      *components.HelpScreen

	activePane  Pane
	showStats   bool
	showAlerts  bool
	inputActive bool
	width       int
	height      int
	ready       bool
	ctx         context.Context
	eventChan   chan agent.Event
}

func NewTheme(cfg *config.ThemeConfig) *Theme {
	return components.NewTheme(cfg)
}

func NewApp(cfg *config.Config, manager *session.Manager, alertMgr *alert.Manager) *App {
	theme := components.NewTheme(&cfg.Theme)

	return &App{
		cfg:      cfg,
		theme:    theme,
		manager:  manager,
		alertMgr: alertMgr,

		activePane: PaneAgentList,
		showStats:  cfg.UI.ShowMetrics,
		showAlerts: true,
		eventChan:  make(chan agent.Event, 100),
	}
}

// Init initializes the application
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.tickCmd(),
		a.waitForEvents(),
	)
}

// tickCmd returns a command that sends tick messages
func (a *App) tickCmd() tea.Cmd {
	return tea.Tick(time.Second*time.Duration(a.cfg.General.RefreshInterval.Seconds()), func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

// waitForEvents waits for agent events
func (a *App) waitForEvents() tea.Cmd {
	return func() tea.Msg {
		select {
		case event := <-a.eventChan:
			return event
		case <-time.After(100 * time.Millisecond):
			return nil
		}
	}
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateSizes()
		a.ready = true
		return a, nil

	case tea.KeyMsg:
		if a.help.IsVisible() {
			a.help, _ = a.help.Update(msg)
			return a, nil
		}

		if a.command.IsVisible() {
			var cmd tea.Cmd
			a.command, cmd = a.command.Update(msg)
			return a, cmd
		}

		if a.inputActive {
			var cmd tea.Cmd
			a.input, cmd = a.input.Update(msg)
			return a, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit

		case "?":
			a.help.Toggle()
			return a, nil

		case ":":
			return a, a.command.Show()

		case "tab":
			a.cycleFocus()
			return a, nil

		case "s":
			a.showStats = !a.showStats
			a.updateSizes()
			return a, nil

		case "a":
			a.showAlerts = !a.showAlerts
			a.updateSizes()
			return a, nil

		case "i":
			a.inputActive = true
			return a, a.input.Focus()

		case "r":
			return a, func() tea.Msg {
				a.manager.Refresh(a.ctx)
				return components.AgentListRefreshMsg{}
			}

		case "g":
			return a, func() tea.Msg {
				return components.ToggleGroupingMsg{}
			}

		case "x":
			if selected := a.agentList.Selected(); selected != nil {
				a.manager.Terminate(selected.ID())
			}
			return a, nil
		}

	case tickMsg:
		cmds = append(cmds, a.tickCmd())
		cmds = append(cmds, a.waitForEvents())

	case agent.Event:
		if a.agentList != nil {
			var cmd tea.Cmd
			a.agentList, cmd = a.agentList.Update(msg)
			cmds = append(cmds, cmd)
		}
		if a.viewport != nil && msg.Agent != nil {
			if a.viewport.Agent() != nil && a.viewport.Agent().ID() == msg.AgentID {
				a.viewport, _ = a.viewport.Update(components.AgentSelectedMsg{Agent: msg.Agent})
			}
		}
		cmds = append(cmds, a.waitForEvents())

	case components.AgentSelectedMsg:
		if a.viewport != nil {
			var cmd tea.Cmd
			a.viewport, cmd = a.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case components.InputSubmitMsg:
		a.inputActive = false
		a.input.Blur()
		if selected := a.agentList.Selected(); selected != nil {
			a.manager.SendInput(selected.ID(), msg.Value)
		}

	case components.ShowHelpMsg:
		a.help.Show()

	case components.ToggleGroupingMsg:
		if a.agentList != nil {
			a.agentList, _ = a.agentList.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
		}

	case components.StartSearchMsg:
		if a.agentList != nil {
			a.activePane = PaneAgentList
			a.updateFocus()
		}

	case components.StartInputMsg:
		a.inputActive = true
		return a, a.input.Focus()

	case components.TerminateAgentMsg:
		if selected := a.agentList.Selected(); selected != nil {
			a.manager.Terminate(selected.ID())
		}

	case components.RefreshMsg:
		return a, func() tea.Msg {
			a.manager.Refresh(a.ctx)
			return components.AgentListRefreshMsg{}
		}

	case components.ToggleStatsMsg:
		a.showStats = !a.showStats
		a.updateSizes()

	case components.ToggleAlertsMsg:
		a.showAlerts = !a.showAlerts
		a.updateSizes()

	case *alert.Alert:
		if a.alerts != nil {
			a.alerts, _ = a.alerts.Update(msg)
		}
	}

	if a.agentList != nil && a.activePane == PaneAgentList {
		var cmd tea.Cmd
		a.agentList, cmd = a.agentList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if a.viewport != nil && a.activePane == PaneViewport {
		var cmd tea.Cmd
		a.viewport, cmd = a.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	if a.stats != nil && a.activePane == PaneStats {
		var cmd tea.Cmd
		a.stats, cmd = a.stats.Update(msg)
		cmds = append(cmds, cmd)
	}

	if a.alerts != nil && a.activePane == PaneAlerts {
		var cmd tea.Cmd
		a.alerts, cmd = a.alerts.Update(msg)
		cmds = append(cmds, cmd)
	}

	return a, tea.Batch(cmds...)
}

// updateSizes recalculates component sizes
func (a *App) updateSizes() {
	if a.width == 0 || a.height == 0 {
		return
	}

	headerHeight := 3
	footerHeight := 3
	availHeight := a.height - headerHeight - footerHeight

	listWidth := a.cfg.UI.AgentListWidth
	if listWidth > a.width/3 {
		listWidth = a.width / 3
	}

	rightPanelWidth := 0
	if a.showStats || a.showAlerts {
		rightPanelWidth = a.width / 4
		if rightPanelWidth < 25 {
			rightPanelWidth = 25
		}
	}

	viewportWidth := a.width - listWidth - rightPanelWidth - 4

	if a.agentList == nil {
		a.agentList = components.NewAgentList(a.theme, a.manager, listWidth, availHeight)
	} else {
		a.agentList.SetSize(listWidth, availHeight)
	}

	if a.viewport == nil {
		a.viewport = components.NewSessionViewport(a.theme, viewportWidth, availHeight)
	} else {
		a.viewport.SetSize(viewportWidth, availHeight)
	}

	if a.showStats {
		statsHeight := availHeight / 2
		if a.stats == nil {
			a.stats = components.NewStatsPanel(a.theme, a.manager, rightPanelWidth, statsHeight)
		} else {
			a.stats.SetSize(rightPanelWidth, statsHeight)
		}
	}

	if a.showAlerts {
		alertsHeight := availHeight / 2
		if a.stats != nil {
			alertsHeight = availHeight - availHeight/2
		}
		if a.alerts == nil {
			a.alerts = components.NewAlertsPanel(a.theme, a.alertMgr, rightPanelWidth, alertsHeight)
		} else {
			a.alerts.SetSize(rightPanelWidth, alertsHeight)
		}
	}

	if a.input == nil {
		a.input = components.NewInputBar(a.theme, a.width-4)
	} else {
		a.input.SetWidth(a.width - 4)
	}

	if a.command == nil {
		a.command = components.NewCommandPalette(a.theme, components.DefaultCommands())
	}
	a.command.SetSize(a.width/2, a.height/2)

	if a.help == nil {
		a.help = components.NewHelpScreen(a.theme)
	}
	a.help.SetSize(a.width*2/3, a.height*2/3)

	a.updateFocus()
}

// cycleFocus cycles through panes
func (a *App) cycleFocus() {
	switch a.activePane {
	case PaneAgentList:
		a.activePane = PaneViewport
	case PaneViewport:
		if a.showStats {
			a.activePane = PaneStats
		} else if a.showAlerts {
			a.activePane = PaneAlerts
		} else {
			a.activePane = PaneAgentList
		}
	case PaneStats:
		if a.showAlerts {
			a.activePane = PaneAlerts
		} else {
			a.activePane = PaneAgentList
		}
	case PaneAlerts:
		a.activePane = PaneAgentList
	}
	a.updateFocus()
}

// updateFocus updates component focus states
func (a *App) updateFocus() {
	if a.agentList != nil {
		a.agentList.SetFocused(a.activePane == PaneAgentList)
	}
	if a.viewport != nil {
		a.viewport.SetFocused(a.activePane == PaneViewport)
	}
	if a.stats != nil {
		a.stats.SetFocused(a.activePane == PaneStats)
	}
	if a.alerts != nil {
		a.alerts.SetFocused(a.activePane == PaneAlerts)
	}
}

// View renders the application
func (a *App) View() string {
	if !a.ready {
		return "Loading..."
	}

	if a.help.IsVisible() {
		return a.renderCentered(a.help.View())
	}

	header := a.renderHeader()
	body := a.renderBody()
	footer := a.renderFooter()

	view := lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		footer,
	)

	if a.command.IsVisible() {
		view = a.overlay(view, a.command.View())
	}

	return view
}

// renderHeader renders the header
func (a *App) renderHeader() string {
	title := a.theme.Title.Render("AUTO")
	stats := components.MiniStats(a.theme, a.manager.Stats())

	width := a.width - lipgloss.Width(title) - 4
	statsRight := lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(stats)

	return a.theme.Header.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, title, statsRight),
	)
}

// renderBody renders the main body
func (a *App) renderBody() string {
	var leftPanel, centerPanel, rightPanel string

	if a.agentList != nil {
		leftPanel = a.agentList.View()
	}

	if a.viewport != nil {
		centerPanel = a.viewport.View()
	}

	if a.showStats && a.stats != nil {
		rightPanel = a.stats.View()
	}
	if a.showAlerts && a.alerts != nil {
		if rightPanel != "" {
			rightPanel = lipgloss.JoinVertical(lipgloss.Left, rightPanel, a.alerts.View())
		} else {
			rightPanel = a.alerts.View()
		}
	}

	panels := []string{leftPanel, centerPanel}
	if rightPanel != "" {
		panels = append(panels, rightPanel)
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, panels...)

	if a.inputActive {
		body = lipgloss.JoinVertical(lipgloss.Left, body, a.input.View())
	}

	return body
}

// renderFooter renders the footer
func (a *App) renderFooter() string {
	help := components.ShortHelp(a.theme)

	unread := a.alertMgr.UnreadCount()
	var alertInfo string
	if unread > 0 {
		alertInfo = a.theme.StatusStyle(agent.StatusErrored).Render(fmt.Sprintf(" %d alerts", unread))
	}

	lastActivity := a.manager.LastActivityTime()
	var timeInfo string
	if !lastActivity.IsZero() {
		timeInfo = a.theme.Base.Faint(true).Render(fmt.Sprintf(" Last: %s", lastActivity.Format("15:04:05")))
	}

	width := a.width - lipgloss.Width(help) - lipgloss.Width(alertInfo) - lipgloss.Width(timeInfo) - 4
	spacer := lipgloss.NewStyle().Width(width).Render("")

	return a.theme.Footer.Width(a.width).Render(
		lipgloss.JoinHorizontal(lipgloss.Top, help, spacer, alertInfo, timeInfo),
	)
}

// renderCentered renders content centered on screen
func (a *App) renderCentered(content string) string {
	contentHeight := lipgloss.Height(content)
	contentWidth := lipgloss.Width(content)

	paddingTop := (a.height - contentHeight) / 2
	paddingLeft := (a.width - contentWidth) / 2

	if paddingTop < 0 {
		paddingTop = 0
	}
	if paddingLeft < 0 {
		paddingLeft = 0
	}

	style := lipgloss.NewStyle().
		PaddingTop(paddingTop).
		PaddingLeft(paddingLeft)

	return style.Render(content)
}

// overlay renders content as an overlay
func (a *App) overlay(base, overlay string) string {
	overlayHeight := lipgloss.Height(overlay)
	overlayWidth := lipgloss.Width(overlay)

	startY := (a.height - overlayHeight) / 2
	startX := (a.width - overlayWidth) / 2

	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	baseLines := splitLines(base)
	overlayLines := splitLines(overlay)

	for i, line := range overlayLines {
		y := startY + i
		if y < len(baseLines) {
			before := ""
			if startX > 0 && startX < len(baseLines[y]) {
				before = baseLines[y][:startX]
			}
			after := ""
			endX := startX + overlayWidth
			if endX < len(baseLines[y]) {
				after = baseLines[y][endX:]
			}
			baseLines[y] = before + line + after
		}
	}

	return joinLines(baseLines)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

// SetContext sets the context for the app
func (a *App) SetContext(ctx context.Context) {
	a.ctx = ctx
}

// EventChannel returns the event channel for pushing events
func (a *App) EventChannel() chan<- agent.Event {
	return a.eventChan
}
