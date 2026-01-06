package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpScreen displays keybindings and help information
type HelpScreen struct {
	theme   *Theme
	visible bool
	width   int
	height  int
}

// NewHelpScreen creates a new help screen
func NewHelpScreen(theme *Theme) *HelpScreen {
	return &HelpScreen{
		theme: theme,
	}
}

// Init initializes the help screen
func (h *HelpScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (h *HelpScreen) Update(msg tea.Msg) (*HelpScreen, tea.Cmd) {
	if !h.visible {
		return h, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "?":
			h.Hide()
		}
	}

	return h, nil
}

// View renders the help screen
func (h *HelpScreen) View() string {
	if !h.visible {
		return ""
	}

	var b strings.Builder

	b.WriteString(h.theme.Title.Render("AUTO - Agent Unified Terminal Orchestrator"))
	b.WriteString("\n\n")

	sections := []struct {
		title string
		keys  [][2]string
	}{
		{
			title: "Navigation",
			keys: [][2]string{
				{"j / down", "Move down"},
				{"k / up", "Move up"},
				{"tab", "Switch pane"},
				{"enter", "Select agent"},
				{"g", "Cycle grouping mode"},
			},
		},
		{
			title: "Actions",
			keys: [][2]string{
				{"i", "Send input to agent"},
				{"x", "Terminate agent"},
				{"space", "Pause/resume agent"},
				{"r", "Refresh"},
				{"R", "Mark all alerts read"},
			},
		},
		{
			title: "Views",
			keys: [][2]string{
				{"s", "Toggle stats panel"},
				{"a", "Toggle alerts panel"},
				{"G", "Scroll to bottom"},
				{"g g", "Scroll to top"},
			},
		},
		{
			title: "Search & Commands",
			keys: [][2]string{
				{"/", "Filter agents"},
				{":", "Command palette"},
				{"esc", "Clear filter / close"},
			},
		},
		{
			title: "General",
			keys: [][2]string{
				{"?", "Toggle help"},
				{"q", "Quit"},
			},
		},
	}

	keyWidth := 15
	for _, section := range sections {
		b.WriteString(h.theme.Subtitle.Render(section.title))
		b.WriteString("\n")

		for _, key := range section.keys {
			keyStyle := lipgloss.NewStyle().
				Width(keyWidth).
				Foreground(h.theme.Primary).
				Bold(true)
			descStyle := lipgloss.NewStyle().
				Foreground(h.theme.Foreground)

			b.WriteString("  ")
			b.WriteString(keyStyle.Render(key[0]))
			b.WriteString(descStyle.Render(key[1]))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(h.theme.Base.Faint(true).Render("Press ? or esc to close"))

	return h.theme.HelpStyle.Width(h.width).Render(b.String())
}

// Show shows the help screen
func (h *HelpScreen) Show() {
	h.visible = true
}

// Hide hides the help screen
func (h *HelpScreen) Hide() {
	h.visible = false
}

// Toggle toggles the help screen
func (h *HelpScreen) Toggle() {
	h.visible = !h.visible
}

// IsVisible returns whether the help screen is visible
func (h *HelpScreen) IsVisible() bool {
	return h.visible
}

// SetSize sets the component size
func (h *HelpScreen) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// ShortHelp returns a short help string for the footer
func ShortHelp(theme *Theme) string {
	keys := []string{
		"j/k: nav",
		"enter: select",
		"i: input",
		"?: help",
		"q: quit",
	}

	style := lipgloss.NewStyle().
		Foreground(theme.Secondary)

	return style.Render(strings.Join(keys, " | "))
}
