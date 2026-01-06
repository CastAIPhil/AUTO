package components

import (
	"github.com/CastAIPhil/AUTO/internal/agent"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputBar is a text input component
type InputBar struct {
	input    textinput.Model
	theme    *Theme
	active   bool
	width    int
	onSubmit func(string)
}

// NewInputBar creates a new input bar
func NewInputBar(theme *Theme, width int) *InputBar {
	ti := textinput.New()
	ti.Placeholder = "Send message to agent..."
	ti.CharLimit = 1000
	ti.Width = width - 4

	return &InputBar{
		input: ti,
		theme: theme,
		width: width,
	}
}

// Init initializes the input bar
func (i *InputBar) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (i *InputBar) Update(msg tea.Msg) (*InputBar, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if i.active && i.input.Value() != "" {
				value := i.input.Value()
				i.input.SetValue("")
				if i.onSubmit != nil {
					i.onSubmit(value)
				}
				return i, func() tea.Msg {
					return InputSubmitMsg{Value: value}
				}
			}
		case "esc":
			i.active = false
			i.input.Blur()
			return i, nil
		}
	}

	if i.active {
		i.input, cmd = i.input.Update(msg)
	}

	return i, cmd
}

// InputSubmitMsg is sent when input is submitted
type InputSubmitMsg struct {
	Value string
}

type StreamEventMsg struct {
	Event   agent.StreamEvent
	AgentID string
}

// View renders the input bar
func (i *InputBar) View() string {
	style := i.theme.InputStyle.Width(i.width)
	return style.Render(i.input.View())
}

// Focus focuses the input
func (i *InputBar) Focus() tea.Cmd {
	i.active = true
	return i.input.Focus()
}

// Blur unfocuses the input
func (i *InputBar) Blur() {
	i.active = false
	i.input.Blur()
}

// IsActive returns whether the input is active
func (i *InputBar) IsActive() bool {
	return i.active
}

// SetWidth sets the input width
func (i *InputBar) SetWidth(width int) {
	i.width = width
	i.input.Width = width - 4
}

// SetPlaceholder sets the placeholder text
func (i *InputBar) SetPlaceholder(placeholder string) {
	i.input.Placeholder = placeholder
}

// OnSubmit sets the submit callback
func (i *InputBar) OnSubmit(fn func(string)) {
	i.onSubmit = fn
}

// Value returns the current input value
func (i *InputBar) Value() string {
	return i.input.Value()
}

// SetValue sets the input value
func (i *InputBar) SetValue(value string) {
	i.input.SetValue(value)
}
