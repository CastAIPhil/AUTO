package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

// Command represents a command palette command
type Command struct {
	Name        string
	Description string
	Keys        string
	Action      func() tea.Msg
}

// CommandPalette is a fuzzy command palette
type CommandPalette struct {
	input    textinput.Model
	theme    *Theme
	commands []Command
	filtered []Command
	cursor   int
	visible  bool
	width    int
	height   int
}

// NewCommandPalette creates a new command palette
func NewCommandPalette(theme *Theme, commands []Command) *CommandPalette {
	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.CharLimit = 100

	return &CommandPalette{
		input:    ti,
		theme:    theme,
		commands: commands,
		filtered: commands,
	}
}

// Init initializes the command palette
func (c *CommandPalette) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (c *CommandPalette) Update(msg tea.Msg) (*CommandPalette, tea.Cmd) {
	if !c.visible {
		return c, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			c.Hide()
			return c, nil
		case "enter":
			if c.cursor < len(c.filtered) {
				command := c.filtered[c.cursor]
				c.Hide()
				if command.Action != nil {
					return c, command.Action
				}
			}
			return c, nil
		case "up", "ctrl+p":
			if c.cursor > 0 {
				c.cursor--
			}
			return c, nil
		case "down", "ctrl+n":
			if c.cursor < len(c.filtered)-1 {
				c.cursor++
			}
			return c, nil
		case "tab":
			if c.cursor < len(c.filtered)-1 {
				c.cursor++
			} else {
				c.cursor = 0
			}
			return c, nil
		}
	}

	c.input, cmd = c.input.Update(msg)
	c.filter()

	return c, cmd
}

// filter filters commands based on input
func (c *CommandPalette) filter() {
	query := c.input.Value()
	if query == "" {
		c.filtered = c.commands
		c.cursor = 0
		return
	}

	names := make([]string, len(c.commands))
	for i, cmd := range c.commands {
		names[i] = cmd.Name + " " + cmd.Description
	}

	matches := fuzzy.Find(query, names)
	c.filtered = make([]Command, len(matches))
	for i, match := range matches {
		c.filtered[i] = c.commands[match.Index]
	}

	if c.cursor >= len(c.filtered) {
		c.cursor = 0
	}
}

// View renders the command palette
func (c *CommandPalette) View() string {
	if !c.visible {
		return ""
	}

	var b strings.Builder

	b.WriteString(c.theme.Title.Render("Command Palette"))
	b.WriteString("\n\n")

	b.WriteString(c.theme.InputStyle.Render(c.input.View()))
	b.WriteString("\n\n")

	maxVisible := 10
	if len(c.filtered) < maxVisible {
		maxVisible = len(c.filtered)
	}

	for i := 0; i < maxVisible; i++ {
		cmd := c.filtered[i]
		line := fmt.Sprintf("%-30s %s", cmd.Name, c.theme.Base.Faint(true).Render(cmd.Keys))
		if i == c.cursor {
			b.WriteString(c.theme.SelectedItemStyle.Render(line))
		} else {
			b.WriteString(c.theme.NormalItemStyle.Render(line))
		}
		b.WriteString("\n")
	}

	if len(c.filtered) == 0 {
		b.WriteString(c.theme.Base.Faint(true).Render("  No matching commands"))
	}

	return c.theme.CommandStyle.Width(c.width).Render(b.String())
}

// Show shows the command palette
func (c *CommandPalette) Show() tea.Cmd {
	c.visible = true
	c.input.SetValue("")
	c.filtered = c.commands
	c.cursor = 0
	return c.input.Focus()
}

// Hide hides the command palette
func (c *CommandPalette) Hide() {
	c.visible = false
	c.input.Blur()
}

// IsVisible returns whether the palette is visible
func (c *CommandPalette) IsVisible() bool {
	return c.visible
}

// SetSize sets the component size
func (c *CommandPalette) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.input.Width = width - 8
}

// SetCommands sets the available commands
func (c *CommandPalette) SetCommands(commands []Command) {
	c.commands = commands
	c.filter()
}

// AddCommand adds a command
func (c *CommandPalette) AddCommand(cmd Command) {
	c.commands = append(c.commands, cmd)
	sort.Slice(c.commands, func(i, j int) bool {
		return c.commands[i].Name < c.commands[j].Name
	})
	c.filter()
}

// DefaultCommands returns the default command set
func DefaultCommands() []Command {
	return []Command{
		{
			Name:        "Quit",
			Description: "Exit AUTO",
			Keys:        "q",
			Action:      func() tea.Msg { return tea.Quit() },
		},
		{
			Name:        "Help",
			Description: "Show help",
			Keys:        "?",
			Action:      func() tea.Msg { return ShowHelpMsg{} },
		},
		{
			Name:        "Toggle Grouping",
			Description: "Cycle through grouping modes",
			Keys:        "g",
			Action:      func() tea.Msg { return ToggleGroupingMsg{} },
		},
		{
			Name:        "Search",
			Description: "Search agents",
			Keys:        "/",
			Action:      func() tea.Msg { return StartSearchMsg{} },
		},
		{
			Name:        "Send Input",
			Description: "Send input to selected agent",
			Keys:        "i",
			Action:      func() tea.Msg { return StartInputMsg{} },
		},
		{
			Name:        "Terminate Agent",
			Description: "Terminate selected agent",
			Keys:        "x",
			Action:      func() tea.Msg { return TerminateAgentMsg{} },
		},
		{
			Name:        "Refresh",
			Description: "Refresh agent list",
			Keys:        "r",
			Action:      func() tea.Msg { return RefreshMsg{} },
		},
		{
			Name:        "Stats",
			Description: "Toggle statistics panel",
			Keys:        "s",
			Action:      func() tea.Msg { return ToggleStatsMsg{} },
		},
		{
			Name:        "Alerts",
			Description: "Toggle alerts panel",
			Keys:        "a",
			Action:      func() tea.Msg { return ToggleAlertsMsg{} },
		},
		{
			Name:        "New Session",
			Description: "Spawn a new agent session",
			Keys:        "n",
			Action:      func() tea.Msg { return SpawnSessionMsg{} },
		},
	}
}

// Command message types
type ShowHelpMsg struct{}
type ToggleGroupingMsg struct{}
type StartSearchMsg struct{}
type StartInputMsg struct{}
type TerminateAgentMsg struct{}
type RefreshMsg struct{}
type ToggleStatsMsg struct{}
type ToggleAlertsMsg struct{}
