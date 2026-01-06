package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/CastAIPhil/AUTO/internal/alert"
)

// AlertsPanel displays alerts and notifications
type AlertsPanel struct {
	theme    *Theme
	alertMgr *alert.Manager
	alerts   []*alert.Alert
	focused  bool
	width    int
	height   int
	cursor   int
	offset   int
}

// NewAlertsPanel creates a new alerts panel
func NewAlertsPanel(theme *Theme, alertMgr *alert.Manager, width, height int) *AlertsPanel {
	return &AlertsPanel{
		theme:    theme,
		alertMgr: alertMgr,
		width:    width,
		height:   height,
	}
}

// Init initializes the alerts panel
func (a *AlertsPanel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (a *AlertsPanel) Update(msg tea.Msg) (*AlertsPanel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if a.cursor < len(a.alerts)-1 {
				a.cursor++
				a.ensureVisible()
			}
		case "k", "up":
			if a.cursor > 0 {
				a.cursor--
				a.ensureVisible()
			}
		case "enter", " ":
			if a.cursor < len(a.alerts) {
				a.alertMgr.MarkRead(a.alerts[a.cursor].ID)
				a.refresh()
			}
		case "R":
			a.alertMgr.MarkAllRead()
			a.refresh()
		}

	case AlertRefreshMsg:
		a.refresh()

	case *alert.Alert:
		a.refresh()
	}

	return a, nil
}

// AlertRefreshMsg triggers a refresh
type AlertRefreshMsg struct{}

// refresh refreshes the alerts list
func (a *AlertsPanel) refresh() {
	a.alerts = a.alertMgr.List(50, false)
}

// ensureVisible ensures the cursor is visible
func (a *AlertsPanel) ensureVisible() {
	visibleLines := a.height - 6
	if visibleLines < 1 {
		visibleLines = 1
	}

	if a.cursor < a.offset {
		a.offset = a.cursor
	}
	if a.cursor >= a.offset+visibleLines {
		a.offset = a.cursor - visibleLines + 1
	}
}

// View renders the alerts panel
func (a *AlertsPanel) View() string {
	style := a.theme.AlertsStyle.Width(a.width).Height(a.height)
	if a.focused {
		style = a.theme.FocusedBorder(style)
	}

	var b strings.Builder

	unreadCount := a.alertMgr.UnreadCount()
	title := "Alerts"
	if unreadCount > 0 {
		title = fmt.Sprintf("Alerts (%d unread)", unreadCount)
	}
	b.WriteString(a.theme.Title.Render(title))
	b.WriteString("\n\n")

	if len(a.alerts) == 0 {
		b.WriteString(a.theme.Base.Faint(true).Render("  No alerts"))
	} else {
		visibleLines := a.height - 6
		if visibleLines < 1 {
			visibleLines = 1
		}

		end := a.offset + visibleLines
		if end > len(a.alerts) {
			end = len(a.alerts)
		}

		for i := a.offset; i < end; i++ {
			al := a.alerts[i]
			line := a.renderAlert(al, i == a.cursor)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	return style.Render(b.String())
}

// renderAlert renders a single alert
func (a *AlertsPanel) renderAlert(al *alert.Alert, selected bool) string {
	icon := "  "
	switch al.Level {
	case alert.LevelError:
		icon = a.theme.StatusStyle(6).Render("! ")
	case alert.LevelWarning:
		icon = a.theme.Base.Foreground(a.theme.StatusIdle).Render("! ")
	case alert.LevelSuccess:
		icon = a.theme.StatusStyle(4).Render("* ")
	case alert.LevelInfo:
		icon = a.theme.Base.Foreground(a.theme.Secondary).Render("i ")
	}

	text := al.Title
	if len(text) > a.width-10 {
		text = text[:a.width-13] + "..."
	}

	timeStr := formatRelativeTime(al.Timestamp)

	line := fmt.Sprintf("%s%s %s", icon, text, a.theme.Base.Faint(true).Render(timeStr))

	if selected {
		return a.theme.SelectedItemStyle.Render(line)
	}

	if !al.Read {
		return a.theme.Base.Bold(true).Render(line)
	}

	return a.theme.Base.Faint(true).Render(line)
}

// SetSize sets the component size
func (a *AlertsPanel) SetSize(width, height int) {
	a.width = width
	a.height = height
}

// SetFocused sets the focus state
func (a *AlertsPanel) SetFocused(focused bool) {
	a.focused = focused
}

// IsFocused returns the focus state
func (a *AlertsPanel) IsFocused() bool {
	return a.focused
}

// UnreadCount returns the number of unread alerts
func (a *AlertsPanel) UnreadCount() int {
	return a.alertMgr.UnreadCount()
}

// formatRelativeTime formats a time as a relative string
func formatRelativeTime(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2")
}
