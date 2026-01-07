package components

import (
	"github.com/CastAIPhil/AUTO/internal/agent"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SpawnState int

const (
	SpawnStateDirectory SpawnState = iota
	SpawnStateName
	SpawnStateConfirm
)

type SpawnDialog struct {
	theme        *Theme
	dirPicker    *DirectoryPicker
	nameInput    textinput.Model
	state        SpawnState
	providerType string
	width        int
	height       int
	cancelled    bool
	submitted    bool
	selectedDir  string
}

type SpawnResult struct {
	Config    agent.SpawnConfig
	Cancelled bool
}

func NewSpawnDialog(theme *Theme, width, height int) *SpawnDialog {
	nameInput := textinput.New()
	nameInput.Placeholder = "Session name (optional)"
	nameInput.Width = 50

	pickerWidth := width - 4
	pickerHeight := height - 8
	if pickerWidth < 40 {
		pickerWidth = 40
	}
	if pickerHeight < 15 {
		pickerHeight = 15
	}

	return &SpawnDialog{
		theme:        theme,
		dirPicker:    NewDirectoryPicker(theme, "", pickerWidth, pickerHeight),
		nameInput:    nameInput,
		state:        SpawnStateDirectory,
		providerType: "opencode",
		width:        width,
		height:       height,
	}
}

func (d *SpawnDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (d *SpawnDialog) Update(msg tea.Msg) (*SpawnDialog, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch d.state {
		case SpawnStateDirectory:
			switch msg.String() {
			case "esc":
				d.cancelled = true
				return d, nil
			default:
				d.dirPicker, cmd = d.dirPicker.Update(msg)
				if d.dirPicker.IsDone() {
					if d.dirPicker.IsCancelled() {
						d.cancelled = true
					} else {
						d.selectedDir = d.dirPicker.Selected()
						d.state = SpawnStateName
						d.nameInput.Focus()
						return d, textinput.Blink
					}
				}
				return d, cmd
			}

		case SpawnStateName:
			switch msg.String() {
			case "esc":
				d.state = SpawnStateDirectory
				d.dirPicker.Reset(d.selectedDir)
				d.nameInput.Blur()
				return d, nil
			case "enter":
				d.state = SpawnStateConfirm
				d.nameInput.Blur()
				return d, nil
			default:
				d.nameInput, cmd = d.nameInput.Update(msg)
				return d, cmd
			}

		case SpawnStateConfirm:
			switch msg.String() {
			case "esc":
				d.state = SpawnStateName
				d.nameInput.Focus()
				return d, textinput.Blink
			case "y", "Y", "enter":
				d.submitted = true
				return d, nil
			case "n", "N":
				d.cancelled = true
				return d, nil
			}
		}
	}

	return d, cmd
}

func (d *SpawnDialog) View() string {
	switch d.state {
	case SpawnStateDirectory:
		return d.dirPicker.View()
	case SpawnStateName, SpawnStateConfirm:
		return d.renderNameAndConfirm()
	}
	return ""
}

func (d *SpawnDialog) renderNameAndConfirm() string {
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

	content += labelStyle.Render("Directory:") + "\n"
	content += activeLabel.Render(d.selectedDir) + "\n\n"

	if d.state == SpawnStateName {
		content += activeLabel.Render("Name (optional):") + "\n"
	} else {
		content += labelStyle.Render("Name (optional):") + "\n"
	}
	content += d.nameInput.View() + "\n\n"

	content += labelStyle.Render("Provider: ") + d.providerType + "\n\n"

	if d.state == SpawnStateConfirm {
		content += activeLabel.Render("Spawn session? (y/n)") + "\n"
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginTop(1)
	content += helpStyle.Render("Enter: confirm | Esc: back")

	return boxStyle.Render(content)
}

func (d *SpawnDialog) Result() SpawnResult {
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
			Directory: d.selectedDir,
		},
		Cancelled: false,
	}
}

func (d *SpawnDialog) IsComplete() bool {
	return d.cancelled || d.submitted
}

func (d *SpawnDialog) IsCancelled() bool {
	return d.cancelled
}

func (d *SpawnDialog) SetSize(width, height int) {
	d.width = width
	d.height = height
	d.dirPicker.SetSize(width-4, height-8)
	d.nameInput.Width = width - 10
}

func (d *SpawnDialog) Reset() {
	d.state = SpawnStateDirectory
	d.cancelled = false
	d.submitted = false
	d.selectedDir = ""
	d.nameInput.SetValue("")
	d.dirPicker.Reset("")
}

type SpawnSessionMsg struct{}
type SpawnCompleteMsg struct {
	Config agent.SpawnConfig
}
type SpawnCancelledMsg struct{}
