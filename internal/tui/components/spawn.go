package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/localrivet/auto/internal/agent"
)

type SpawnState int

const (
	SpawnStateDirectory SpawnState = iota
	SpawnStateName
	SpawnStateConfirm
)

type SpawnDialog struct {
	directoryInput textinput.Model
	nameInput      textinput.Model
	state          SpawnState
	providerType   string
	width          int
	height         int
	cancelled      bool
	submitted      bool
}

type SpawnResult struct {
	Config    agent.SpawnConfig
	Cancelled bool
}

func NewSpawnDialog() SpawnDialog {
	dirInput := textinput.New()
	dirInput.Placeholder = "Working directory (e.g., ~/projects/myapp)"
	dirInput.Focus()
	dirInput.Width = 50

	nameInput := textinput.New()
	nameInput.Placeholder = "Session name (optional)"
	nameInput.Width = 50

	return SpawnDialog{
		directoryInput: dirInput,
		nameInput:      nameInput,
		state:          SpawnStateDirectory,
		providerType:   "opencode",
		width:          60,
		height:         15,
	}
}

func (d SpawnDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (d SpawnDialog) Update(msg tea.Msg) (SpawnDialog, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			d.cancelled = true
			return d, nil

		case "enter":
			switch d.state {
			case SpawnStateDirectory:
				if d.directoryInput.Value() != "" {
					d.state = SpawnStateName
					d.directoryInput.Blur()
					d.nameInput.Focus()
					return d, textinput.Blink
				}
			case SpawnStateName:
				d.state = SpawnStateConfirm
				d.nameInput.Blur()
				return d, nil
			case SpawnStateConfirm:
				d.submitted = true
				return d, nil
			}

		case "tab":
			switch d.state {
			case SpawnStateDirectory:
				d.state = SpawnStateName
				d.directoryInput.Blur()
				d.nameInput.Focus()
				return d, textinput.Blink
			case SpawnStateName:
				d.state = SpawnStateConfirm
				d.nameInput.Blur()
				return d, nil
			}

		case "shift+tab":
			switch d.state {
			case SpawnStateName:
				d.state = SpawnStateDirectory
				d.nameInput.Blur()
				d.directoryInput.Focus()
				return d, textinput.Blink
			case SpawnStateConfirm:
				d.state = SpawnStateName
				d.nameInput.Focus()
				return d, textinput.Blink
			}

		case "y", "Y":
			if d.state == SpawnStateConfirm {
				d.submitted = true
				return d, nil
			}

		case "n", "N":
			if d.state == SpawnStateConfirm {
				d.cancelled = true
				return d, nil
			}
		}
	}

	switch d.state {
	case SpawnStateDirectory:
		d.directoryInput, cmd = d.directoryInput.Update(msg)
	case SpawnStateName:
		d.nameInput, cmd = d.nameInput.Update(msg)
	}

	return d, cmd
}

func (d SpawnDialog) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	activeLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(1, 2).
		Width(d.width)

	var content string
	content += titleStyle.Render("Spawn New Agent Session") + "\n\n"

	// Directory input
	if d.state == SpawnStateDirectory {
		content += activeLabel.Render("Directory:") + "\n"
	} else {
		content += labelStyle.Render("Directory:") + "\n"
	}
	content += d.directoryInput.View() + "\n\n"

	// Name input
	if d.state == SpawnStateName {
		content += activeLabel.Render("Name (optional):") + "\n"
	} else {
		content += labelStyle.Render("Name (optional):") + "\n"
	}
	content += d.nameInput.View() + "\n\n"

	// Provider info
	content += labelStyle.Render("Provider: ") + d.providerType + "\n\n"

	// Confirm prompt
	if d.state == SpawnStateConfirm {
		content += activeLabel.Render("Spawn session? (y/n)") + "\n"
	}

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)
	content += helpStyle.Render("Tab: next field | Esc: cancel | Enter: confirm")

	return boxStyle.Render(content)
}

func (d SpawnDialog) Result() SpawnResult {
	if d.cancelled {
		return SpawnResult{Cancelled: true}
	}

	name := d.nameInput.Value()
	if name == "" {
		name = "new-session"
	}

	return SpawnResult{
		Config: agent.SpawnConfig{
			Type:      d.providerType,
			Name:      name,
			Directory: d.directoryInput.Value(),
		},
		Cancelled: false,
	}
}

func (d SpawnDialog) IsComplete() bool {
	return d.cancelled || d.submitted
}

func (d SpawnDialog) IsCancelled() bool {
	return d.cancelled
}
