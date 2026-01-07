package components

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/CastAIPhil/AUTO/internal/agent"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// SessionViewport displays agent session output
type SessionViewport struct {
	viewport      viewport.Model
	theme         *Theme
	agent         agent.Agent
	content       string
	streamContent strings.Builder
	focused       bool
	width         int
	height        int
	autoScroll    bool
	isStreaming   bool
	mdRenderer    *glamour.TermRenderer

	lastContentLen   int
	formattedContent string
	contentDirty     bool
	headerDirty      bool
	cachedHeader     string
}

// NewSessionViewport creates a new session viewport
func NewSessionViewport(theme *Theme, width, height int) *SessionViewport {
	vp := viewport.New(width-4, height-6)
	vp.Style = lipgloss.NewStyle()

	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-8),
	)

	return &SessionViewport{
		viewport:     vp,
		theme:        theme,
		width:        width,
		height:       height,
		autoScroll:   true,
		mdRenderer:   renderer,
		contentDirty: true,
		headerDirty:  true,
	}
}

// Init initializes the viewport
func (s *SessionViewport) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (s *SessionViewport) Update(msg tea.Msg) (*SessionViewport, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "G":
			// Go to bottom and enable auto-scroll
			s.viewport.GotoBottom()
			s.autoScroll = true
		case "g":
			// Go to top and disable auto-scroll
			s.viewport.GotoTop()
			s.autoScroll = false
		case "ctrl+c":
			// Copy content to clipboard (placeholder)
		}

	case AgentSelectedMsg:
		s.SetAgent(msg.Agent)
		return s, s.refreshContent()

	case ViewportRefreshMsg:
		s.updateContent()

	case StreamEventMsg:
		return s, s.handleStreamEvent(msg)
	}

	// Update viewport
	s.viewport, cmd = s.viewport.Update(msg)

	// Check if user scrolled manually
	if s.viewport.AtBottom() {
		s.autoScroll = true
	} else if msg != nil {
		if _, ok := msg.(tea.KeyMsg); ok {
			s.autoScroll = false
		}
	}

	return s, cmd
}

// ViewportRefreshMsg triggers a content refresh
type ViewportRefreshMsg struct{}

// refreshContent returns a command to refresh content
func (s *SessionViewport) refreshContent() tea.Cmd {
	return func() tea.Msg {
		return ViewportRefreshMsg{}
	}
}

// updateContent updates the viewport content from the agent
func (s *SessionViewport) updateContent() {
	if s.agent == nil {
		if s.content != "" || s.formattedContent == "" {
			s.content = ""
			s.formattedContent = s.renderEmptyState()
			s.viewport.SetContent(s.formattedContent)
		}
		return
	}

	reader := s.agent.Output()
	data, err := io.ReadAll(reader)
	if err != nil {
		s.content = fmt.Sprintf("Error reading output: %v", err)
		s.formattedContent = s.content
		s.viewport.SetContent(s.formattedContent)
		return
	}

	newLen := len(data)
	if newLen == s.lastContentLen && !s.contentDirty {
		return
	}

	s.content = string(data)
	s.lastContentLen = newLen
	s.formattedContent = s.formatContent(s.content)
	s.viewport.SetContent(s.formattedContent)
	s.contentDirty = false
	s.headerDirty = true

	if s.autoScroll {
		s.viewport.GotoBottom()
	}
}

// renderEmptyState renders the empty state
func (s *SessionViewport) renderEmptyState() string {
	return s.theme.Base.Faint(true).Render(`
  No agent selected.

  Select an agent from the list to view its output.

  Keys:
    j/k - Navigate agents
    Enter - Select agent
    ? - Help
`)
}

// formatContent formats the output content with markdown rendering
func (s *SessionViewport) formatContent(content string) string {
	if content == "" {
		return s.theme.Base.Faint(true).Render("(no output)")
	}

	if s.mdRenderer != nil {
		rendered, err := s.mdRenderer.Render(content)
		if err == nil {
			return strings.TrimSpace(rendered)
		}
	}

	lines := strings.Split(content, "\n")
	var formatted []string

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "error") {
			line = s.theme.StatusStyle(agent.StatusErrored).Render(line)
		} else if strings.Contains(strings.ToLower(line), "warning") {
			line = lipgloss.NewStyle().Foreground(s.theme.StatusIdle).Render(line)
		} else if strings.HasPrefix(line, ">>>") || strings.HasPrefix(line, "---") {
			line = s.theme.Title.Render(line)
		}
		formatted = append(formatted, line)
	}

	return strings.Join(formatted, "\n")
}

// View renders the viewport
func (s *SessionViewport) View() string {
	style := s.theme.ViewportStyle.Width(s.width).Height(s.height)
	if s.focused {
		style = s.theme.FocusedBorder(style)
	}

	if s.headerDirty || s.cachedHeader == "" {
		s.cachedHeader = s.renderHeader()
		s.headerDirty = false
	}

	footer := s.renderFooter()

	viewportHeight := s.height - 4 - lipgloss.Height(s.cachedHeader) - lipgloss.Height(footer)
	s.viewport.Height = viewportHeight

	return style.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			s.cachedHeader,
			s.viewport.View(),
			footer,
		),
	)
}

// renderHeader renders the viewport header
func (s *SessionViewport) renderHeader() string {
	if s.agent == nil {
		return s.theme.Header.Render("Session Output")
	}

	status := StatusIndicator(s.theme, s.agent.Status())
	name := s.agent.Name()
	task := s.agent.CurrentTask()

	if len(task) > 50 {
		task = task[:50] + "..."
	}

	header := fmt.Sprintf("%s %s - %s", status, name, task)

	// Add streaming indicator
	if s.isStreaming {
		streamIndicator := lipgloss.NewStyle().
			Foreground(s.theme.StatusRunning).
			Bold(true).
			Render(" ● STREAMING")
		header = header + streamIndicator
	}

	return s.theme.Header.Render(header)
}

// renderFooter renders the viewport footer
func (s *SessionViewport) renderFooter() string {
	if s.agent == nil {
		return ""
	}

	metrics := s.agent.Metrics()
	scrollInfo := fmt.Sprintf("%d%%", int(s.viewport.ScrollPercent()*100))

	info := fmt.Sprintf("Tokens: %d/%d | Cost: $%.4f | Tools: %d | %s",
		metrics.TokensIn,
		metrics.TokensOut,
		metrics.EstimatedCost,
		metrics.ToolCalls,
		scrollInfo,
	)

	if s.autoScroll {
		info += " [auto-scroll]"
	}

	return s.theme.Footer.Render(info)
}

// SetSize sets the component size
func (s *SessionViewport) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.viewport.Width = width - 4
	s.viewport.Height = height - 8

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-8),
	)
	if err == nil {
		s.mdRenderer = renderer
	}
}

// SetFocused sets the focus state
func (s *SessionViewport) SetFocused(focused bool) {
	s.focused = focused
}

// IsFocused returns the focus state
func (s *SessionViewport) IsFocused() bool {
	return s.focused
}

// SetAgent sets the agent to display
func (s *SessionViewport) SetAgent(ag agent.Agent) {
	s.agent = ag
	s.lastContentLen = 0
	s.contentDirty = true
	s.headerDirty = true
	s.updateContent()
}

// Agent returns the current agent
func (s *SessionViewport) Agent() agent.Agent {
	return s.agent
}

// ScrollToTop scrolls to the top
func (s *SessionViewport) ScrollToTop() {
	s.viewport.GotoTop()
	s.autoScroll = false
}

// ScrollToBottom scrolls to the bottom
func (s *SessionViewport) ScrollToBottom() {
	s.viewport.GotoBottom()
	s.autoScroll = true
}

// SetAutoScroll sets the auto-scroll state
func (s *SessionViewport) SetAutoScroll(enabled bool) {
	s.autoScroll = enabled
}

// handleStreamEvent processes a streaming event
func (s *SessionViewport) handleStreamEvent(msg StreamEventMsg) tea.Cmd {
	event := msg.Event

	switch event.Type {
	case "text":
		s.isStreaming = true
		s.streamContent.WriteString(event.Text)
		combined := s.content + "\n" + s.streamContent.String()
		s.viewport.SetContent(s.formatContent(combined))
		if s.autoScroll {
			s.viewport.GotoBottom()
		}

	case "tool-start":
		s.isStreaming = true
		s.streamContent.WriteString(fmt.Sprintf("\n[Tool: %s]\n", event.ToolName))
		combined := s.content + "\n" + s.streamContent.String()
		s.viewport.SetContent(s.formatContent(combined))
		if s.autoScroll {
			s.viewport.GotoBottom()
		}

	case "tool-end":
		s.streamContent.WriteString(fmt.Sprintf("[/%s: %s]\n", event.ToolName, event.State))
		combined := s.content + "\n" + s.streamContent.String()
		s.viewport.SetContent(s.formatContent(combined))
		if s.autoScroll {
			s.viewport.GotoBottom()
		}

	case "done", "error":
		s.isStreaming = false
		if s.streamContent.Len() > 0 {
			s.content = s.content + "\n" + s.streamContent.String()
			s.streamContent.Reset()
		}
		s.viewport.SetContent(s.formatContent(s.content))
		if s.autoScroll {
			s.viewport.GotoBottom()
		}
	}

	return nil
}

// ClearStreamContent clears the streaming buffer
func (s *SessionViewport) ClearStreamContent() {
	s.streamContent.Reset()
	s.isStreaming = false
}

func (s *SessionViewport) AppendUserInput(input string) {
	userPrefix := lipgloss.NewStyle().Foreground(s.theme.Secondary).Bold(true).Render(">>> You: ")
	s.content += "\n\n" + userPrefix + input + "\n"
	s.viewport.SetContent(s.formatContent(s.content))
	if s.autoScroll {
		s.viewport.GotoBottom()
	}
}

// IsStreaming returns whether the viewport is currently streaming
func (s *SessionViewport) IsStreaming() bool {
	return s.isStreaming
}

func (s *SessionViewport) MarkDirty() {
	s.contentDirty = true
	s.headerDirty = true
}

// FormatDuration formats a duration for display
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
