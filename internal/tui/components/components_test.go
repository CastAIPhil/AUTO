package components

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/CastAIPhil/AUTO/internal/agent"
	tea "github.com/charmbracelet/bubbletea"
)

type mockAgent struct {
	id           string
	name         string
	agentType    string
	directory    string
	projectID    string
	parentID     string
	status       agent.Status
	startTime    time.Time
	lastActivity time.Time
	currentTask  string
	metrics      agent.Metrics
	lastError    error
}

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Name() string            { return m.name }
func (m *mockAgent) Type() string            { return m.agentType }
func (m *mockAgent) Directory() string       { return m.directory }
func (m *mockAgent) ProjectID() string       { return m.projectID }
func (m *mockAgent) ParentID() string        { return m.parentID }
func (m *mockAgent) IsBackground() bool      { return m.parentID != "" }
func (m *mockAgent) Status() agent.Status    { return m.status }
func (m *mockAgent) StartTime() time.Time    { return m.startTime }
func (m *mockAgent) LastActivity() time.Time { return m.lastActivity }
func (m *mockAgent) Output() io.Reader {
	return strings.NewReader("test output")
}
func (m *mockAgent) CurrentTask() string          { return m.currentTask }
func (m *mockAgent) Metrics() agent.Metrics       { return m.metrics }
func (m *mockAgent) LastError() error             { return m.lastError }
func (m *mockAgent) SendInput(input string) error { return nil }
func (m *mockAgent) Terminate() error             { return nil }
func (m *mockAgent) Pause() error                 { return nil }
func (m *mockAgent) Resume() error                { return nil }

func newMockAgent(id, name string, status agent.Status) *mockAgent {
	return &mockAgent{
		id:           id,
		name:         name,
		agentType:    "test",
		directory:    "/test/dir",
		projectID:    "test-project",
		status:       status,
		startTime:    time.Now().Add(-time.Hour),
		lastActivity: time.Now(),
		currentTask:  "test task",
		metrics:      agent.Metrics{TokensIn: 100, TokensOut: 50},
	}
}

// =============================================================================
// Filter Tests
// =============================================================================

func TestNewFilter(t *testing.T) {
	f := NewFilter()

	if f.mode != FilterModeNone {
		t.Errorf("NewFilter mode = %v, want FilterModeNone", f.mode)
	}
	if f.statusFilter != -1 {
		t.Errorf("NewFilter statusFilter = %v, want -1", f.statusFilter)
	}
	if f.IsActive() {
		t.Error("NewFilter should not be active")
	}
}

func TestFilterApplyStatusFilter(t *testing.T) {
	tests := []struct {
		name         string
		statusFilter agent.Status
		agents       []agent.Agent
		wantCount    int
	}{
		{
			name:         "no filter returns all",
			statusFilter: -1,
			agents: []agent.Agent{
				newMockAgent("1", "agent1", agent.StatusRunning),
				newMockAgent("2", "agent2", agent.StatusIdle),
				newMockAgent("3", "agent3", agent.StatusErrored),
			},
			wantCount: 3,
		},
		{
			name:         "filter running only",
			statusFilter: agent.StatusRunning,
			agents: []agent.Agent{
				newMockAgent("1", "agent1", agent.StatusRunning),
				newMockAgent("2", "agent2", agent.StatusIdle),
				newMockAgent("3", "agent3", agent.StatusRunning),
			},
			wantCount: 2,
		},
		{
			name:         "filter errored only",
			statusFilter: agent.StatusErrored,
			agents: []agent.Agent{
				newMockAgent("1", "agent1", agent.StatusRunning),
				newMockAgent("2", "agent2", agent.StatusErrored),
			},
			wantCount: 1,
		},
		{
			name:         "filter with no matches",
			statusFilter: agent.StatusCompleted,
			agents: []agent.Agent{
				newMockAgent("1", "agent1", agent.StatusRunning),
				newMockAgent("2", "agent2", agent.StatusIdle),
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFilter()
			f.statusFilter = tt.statusFilter

			result := f.Apply(tt.agents)
			if len(result) != tt.wantCount {
				t.Errorf("Apply() returned %d agents, want %d", len(result), tt.wantCount)
			}
		})
	}
}

func TestFilterApplySearchFilter(t *testing.T) {
	agents := []agent.Agent{
		newMockAgent("1", "frontend-dev", agent.StatusRunning),
		newMockAgent("2", "backend-api", agent.StatusIdle),
		newMockAgent("3", "database-worker", agent.StatusRunning),
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{"empty query returns all", "", 3},
		{"search by name prefix", "front", 1},
		{"search by name substring", "end", 2}, // frontend, backend
		{"search case insensitive", "BACKEND", 1},
		{"search no matches", "xyz", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFilter()
			f.searchInput.SetValue(tt.query)

			result := f.Apply(agents)
			if len(result) != tt.wantCount {
				t.Errorf("Apply() with query %q returned %d agents, want %d",
					tt.query, len(result), tt.wantCount)
			}
		})
	}
}

func TestFilterKeyHandling(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		initialActive  bool
		wantActive     bool
		wantSearchOpen bool
	}{
		{
			name:           "slash opens search",
			key:            "/",
			initialActive:  false,
			wantActive:     true,
			wantSearchOpen: true,
		},
		{
			name:          "f cycles status forward",
			key:           "f",
			initialActive: false,
			wantActive:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFilter()

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			f, _ = f.Update(msg)

			if f.IsActive() != tt.wantActive {
				t.Errorf("IsActive() = %v, want %v", f.IsActive(), tt.wantActive)
			}
			if tt.wantSearchOpen && !f.IsSearchActive() {
				t.Error("search should be active after /")
			}
		})
	}
}

func TestFilterIsActive(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*Filter)
		wantActive bool
	}{
		{
			name:       "inactive by default",
			setup:      func(f *Filter) {},
			wantActive: false,
		},
		{
			name: "active with status filter",
			setup: func(f *Filter) {
				f.statusFilter = agent.StatusRunning
			},
			wantActive: true,
		},
		{
			name: "active with search query",
			setup: func(f *Filter) {
				f.searchInput.SetValue("test")
			},
			wantActive: true,
		},
		{
			name: "active when search input focused",
			setup: func(f *Filter) {
				f.searchActive = true
			},
			wantActive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFilter()
			tt.setup(&f)

			if f.IsActive() != tt.wantActive {
				t.Errorf("IsActive() = %v, want %v", f.IsActive(), tt.wantActive)
			}
		})
	}
}

func TestFilterSetCounts(t *testing.T) {
	f := NewFilter()
	f.SetCounts(5, 10)

	if f.filteredCount != 5 {
		t.Errorf("filteredCount = %d, want 5", f.filteredCount)
	}
	if f.totalCount != 10 {
		t.Errorf("totalCount = %d, want 10", f.totalCount)
	}
}

// =============================================================================
// SpawnDialog Tests
// =============================================================================

func TestNewSpawnDialog(t *testing.T) {
	theme := DefaultDarkTheme()
	d := NewSpawnDialog(theme, 80, 40)

	if d.state != SpawnStateDirectory {
		t.Errorf("initial state = %v, want SpawnStateDirectory", d.state)
	}
	if d.providerType != "opencode" {
		t.Errorf("providerType = %q, want opencode", d.providerType)
	}
	if d.IsComplete() {
		t.Error("dialog should not be complete initially")
	}
	if d.IsCancelled() {
		t.Error("dialog should not be cancelled initially")
	}
}

func TestSpawnDialogEscCancels(t *testing.T) {
	theme := DefaultDarkTheme()
	d := NewSpawnDialog(theme, 80, 40)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	d, _ = d.Update(msg)

	if !d.IsCancelled() {
		t.Error("dialog should be cancelled after esc")
	}
	if !d.IsComplete() {
		t.Error("dialog should be complete after cancellation")
	}
}

func TestSpawnDialogResult(t *testing.T) {
	tests := []struct {
		name          string
		directory     string
		sessionName   string
		cancelled     bool
		wantName      string
		wantCancelled bool
	}{
		{
			name:          "cancelled result",
			cancelled:     true,
			wantCancelled: true,
		},
		{
			name:        "default session name",
			directory:   "/test/dir",
			sessionName: "",
			wantName:    "new-session",
		},
		{
			name:        "custom session name",
			directory:   "/test/dir",
			sessionName: "my-session",
			wantName:    "my-session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := DefaultDarkTheme()
			d := NewSpawnDialog(theme, 80, 40)
			d.selectedDir = tt.directory
			d.nameInput.SetValue(tt.sessionName)
			d.cancelled = tt.cancelled

			result := d.Result()

			if result.Cancelled != tt.wantCancelled {
				t.Errorf("Result.Cancelled = %v, want %v", result.Cancelled, tt.wantCancelled)
			}
			if !tt.wantCancelled && result.Config.Name != tt.wantName {
				t.Errorf("Result.Config.Name = %q, want %q", result.Config.Name, tt.wantName)
			}
			if !tt.wantCancelled && result.Config.Directory != tt.directory {
				t.Errorf("Result.Config.Directory = %q, want %q", result.Config.Directory, tt.directory)
			}
		})
	}
}

func TestSpawnDialogConfirmState(t *testing.T) {
	theme := DefaultDarkTheme()
	d := NewSpawnDialog(theme, 80, 40)
	d.selectedDir = "/test"
	d.state = SpawnStateConfirm

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	d, _ = d.Update(msg)

	if !d.submitted {
		t.Error("'y' should submit in confirm state")
	}
	if !d.IsComplete() {
		t.Error("dialog should be complete after submission")
	}

	d2 := NewSpawnDialog(theme, 80, 40)
	d2.state = SpawnStateConfirm
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	d2, _ = d2.Update(msg)

	if !d2.IsCancelled() {
		t.Error("'n' should cancel in confirm state")
	}
}

// =============================================================================
// HelpScreen Tests
// =============================================================================

func TestNewHelpScreen(t *testing.T) {
	theme := DefaultDarkTheme()
	h := NewHelpScreen(theme)

	if h.IsVisible() {
		t.Error("help screen should be hidden initially")
	}
}

func TestHelpScreenToggle(t *testing.T) {
	theme := DefaultDarkTheme()
	h := NewHelpScreen(theme)

	h.Toggle()
	if !h.IsVisible() {
		t.Error("Toggle should show help screen")
	}

	h.Toggle()
	if h.IsVisible() {
		t.Error("Toggle should hide help screen")
	}
}

func TestHelpScreenShowHide(t *testing.T) {
	theme := DefaultDarkTheme()
	h := NewHelpScreen(theme)

	h.Show()
	if !h.IsVisible() {
		t.Error("Show should make help visible")
	}

	h.Hide()
	if h.IsVisible() {
		t.Error("Hide should make help invisible")
	}
}

func TestHelpScreenKeyHandling(t *testing.T) {
	theme := DefaultDarkTheme()

	closeKeys := []string{"esc", "q", "?"}

	for _, key := range closeKeys {
		t.Run("close with "+key, func(t *testing.T) {
			h := NewHelpScreen(theme)
			h.Show()

			var msg tea.KeyMsg
			if key == "esc" {
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
			}

			h, _ = h.Update(msg)

			if h.IsVisible() {
				t.Errorf("help should hide after pressing %q", key)
			}
		})
	}
}

func TestHelpScreenViewWhenHidden(t *testing.T) {
	theme := DefaultDarkTheme()
	h := NewHelpScreen(theme)

	view := h.View()
	if view != "" {
		t.Error("View should return empty string when hidden")
	}
}

func TestHelpScreenViewWhenVisible(t *testing.T) {
	theme := DefaultDarkTheme()
	h := NewHelpScreen(theme)
	h.SetSize(80, 40)
	h.Show()

	view := h.View()
	if view == "" {
		t.Error("View should return content when visible")
	}

	// Check for expected content
	expectedStrings := []string{
		"AUTO",
		"Navigation",
		"Actions",
		"j / down",
		"enter",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(view, s) {
			t.Errorf("View should contain %q", s)
		}
	}
}

func TestHelpScreenSetSize(t *testing.T) {
	theme := DefaultDarkTheme()
	h := NewHelpScreen(theme)

	h.SetSize(100, 50)

	if h.width != 100 {
		t.Errorf("width = %d, want 100", h.width)
	}
	if h.height != 50 {
		t.Errorf("height = %d, want 50", h.height)
	}
}

// =============================================================================
// CommandPalette Tests
// =============================================================================

func TestNewCommandPalette(t *testing.T) {
	theme := DefaultDarkTheme()
	commands := DefaultCommands()
	cp := NewCommandPalette(theme, commands)

	if cp.IsVisible() {
		t.Error("command palette should be hidden initially")
	}
	if len(cp.commands) != len(commands) {
		t.Errorf("commands count = %d, want %d", len(cp.commands), len(commands))
	}
}

func TestCommandPaletteShowHide(t *testing.T) {
	theme := DefaultDarkTheme()
	cp := NewCommandPalette(theme, DefaultCommands())

	cp.Show()
	if !cp.IsVisible() {
		t.Error("Show should make palette visible")
	}

	cp.Hide()
	if cp.IsVisible() {
		t.Error("Hide should make palette invisible")
	}
}

func TestCommandPaletteShowResetsState(t *testing.T) {
	theme := DefaultDarkTheme()
	cp := NewCommandPalette(theme, DefaultCommands())

	cp.cursor = 5
	cp.input.SetValue("test")

	cp.Show()

	if cp.cursor != 0 {
		t.Error("Show should reset cursor to 0")
	}
	if cp.input.Value() != "" {
		t.Error("Show should clear input value")
	}
}

func TestCommandPaletteFiltering(t *testing.T) {
	theme := DefaultDarkTheme()
	commands := []Command{
		{Name: "Quit", Description: "Exit", Keys: "q"},
		{Name: "Help", Description: "Show help", Keys: "?"},
		{Name: "Search", Description: "Search agents", Keys: "/"},
	}
	cp := NewCommandPalette(theme, commands)
	cp.Show()

	// Initially all commands visible
	if len(cp.filtered) != 3 {
		t.Errorf("initial filtered = %d, want 3", len(cp.filtered))
	}

	// Type 'h' to filter
	cp.input.SetValue("h")
	cp.filter()

	// Should match "Help" and maybe "Search" depending on fuzzy
	if len(cp.filtered) == 0 {
		t.Error("filtering should find at least one match for 'h'")
	}

	// Type 'quit' for exact match
	cp.input.SetValue("quit")
	cp.filter()

	if len(cp.filtered) != 1 || cp.filtered[0].Name != "Quit" {
		t.Error("filtering for 'quit' should return only Quit command")
	}
}

func TestCommandPaletteNavigation(t *testing.T) {
	theme := DefaultDarkTheme()
	commands := []Command{
		{Name: "A", Description: "First"},
		{Name: "B", Description: "Second"},
		{Name: "C", Description: "Third"},
	}
	cp := NewCommandPalette(theme, commands)
	cp.Show()

	// Test down navigation
	msg := tea.KeyMsg{Type: tea.KeyDown}
	cp, _ = cp.Update(msg)
	if cp.cursor != 1 {
		t.Errorf("after down, cursor = %d, want 1", cp.cursor)
	}

	// Test up navigation
	msg = tea.KeyMsg{Type: tea.KeyUp}
	cp, _ = cp.Update(msg)
	if cp.cursor != 0 {
		t.Errorf("after up, cursor = %d, want 0", cp.cursor)
	}

	// Test tab wraps
	cp.cursor = 2
	msg = tea.KeyMsg{Type: tea.KeyTab}
	cp, _ = cp.Update(msg)
	if cp.cursor != 0 {
		t.Errorf("tab at end should wrap to 0, got %d", cp.cursor)
	}
}

func TestCommandPaletteEscCloses(t *testing.T) {
	theme := DefaultDarkTheme()
	cp := NewCommandPalette(theme, DefaultCommands())
	cp.Show()

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cp, _ = cp.Update(msg)

	if cp.IsVisible() {
		t.Error("esc should close palette")
	}
}

func TestCommandPaletteEnterExecutes(t *testing.T) {
	theme := DefaultDarkTheme()
	executed := false
	commands := []Command{
		{
			Name:   "Test",
			Action: func() tea.Msg { executed = true; return nil },
		},
	}
	cp := NewCommandPalette(theme, commands)
	cp.Show()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	cp, cmd := cp.Update(msg)

	if cp.IsVisible() {
		t.Error("enter should close palette")
	}
	if cmd == nil {
		t.Error("enter should return command action")
	}

	// Execute the command
	if cmd != nil {
		cmd()
	}
	if !executed {
		t.Error("command action should have been executed")
	}
}

func TestCommandPaletteSetSize(t *testing.T) {
	theme := DefaultDarkTheme()
	cp := NewCommandPalette(theme, DefaultCommands())

	cp.SetSize(60, 30)

	if cp.width != 60 {
		t.Errorf("width = %d, want 60", cp.width)
	}
	if cp.height != 30 {
		t.Errorf("height = %d, want 30", cp.height)
	}
}

func TestCommandPaletteAddCommand(t *testing.T) {
	theme := DefaultDarkTheme()
	cp := NewCommandPalette(theme, []Command{})

	cp.AddCommand(Command{Name: "Zebra", Description: "Last"})
	cp.AddCommand(Command{Name: "Alpha", Description: "First"})

	if len(cp.commands) != 2 {
		t.Errorf("commands count = %d, want 2", len(cp.commands))
	}

	// Should be sorted alphabetically
	if cp.commands[0].Name != "Alpha" {
		t.Errorf("first command = %q, want Alpha", cp.commands[0].Name)
	}
}

// =============================================================================
// DefaultCommands Tests
// =============================================================================

func TestDefaultCommands(t *testing.T) {
	commands := DefaultCommands()

	if len(commands) == 0 {
		t.Fatal("DefaultCommands should return commands")
	}

	// Check for expected commands
	expectedNames := []string{"Quit", "Help", "Search", "Refresh"}
	for _, name := range expectedNames {
		found := false
		for _, cmd := range commands {
			if cmd.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultCommands should include %q", name)
		}
	}

	// Verify all commands have actions
	for _, cmd := range commands {
		if cmd.Action == nil {
			t.Errorf("command %q has nil action", cmd.Name)
		}
	}
}

// =============================================================================
// ShortHelp Tests
// =============================================================================

func TestShortHelp(t *testing.T) {
	theme := DefaultDarkTheme()
	help := ShortHelp(theme)

	if help == "" {
		t.Error("ShortHelp should return non-empty string")
	}

	expectedKeys := []string{"j/k", "enter", "children", "back", "?", "q"}
	for _, key := range expectedKeys {
		if !strings.Contains(help, key) {
			t.Errorf("ShortHelp should contain %q", key)
		}
	}
}

// =============================================================================
// Theme Tests
// =============================================================================

func TestDefaultDarkTheme(t *testing.T) {
	theme := DefaultDarkTheme()

	if theme == nil {
		t.Fatal("DefaultDarkTheme returned nil")
	}

	// Check that status styles are initialized
	if len(theme.StatusStyles) == 0 {
		t.Error("StatusStyles should be initialized")
	}

	// Check that we have styles for all statuses
	statuses := []agent.Status{
		agent.StatusPending,
		agent.StatusRunning,
		agent.StatusIdle,
		agent.StatusCompleted,
		agent.StatusErrored,
	}
	for _, s := range statuses {
		if _, ok := theme.StatusStyles[s]; !ok {
			t.Errorf("missing style for status %v", s)
		}
	}
}

func TestDefaultLightTheme(t *testing.T) {
	theme := DefaultLightTheme()

	if theme == nil {
		t.Fatal("DefaultLightTheme returned nil")
	}
}

func TestThemeStatusStyle(t *testing.T) {
	theme := DefaultDarkTheme()

	// Test known status
	style := theme.StatusStyle(agent.StatusRunning)
	if style.GetBold() != true {
		t.Error("Running status should be bold")
	}

	// Test unknown status returns base
	unknownStyle := theme.StatusStyle(agent.Status(999))
	if unknownStyle.String() != theme.Base.String() {
		t.Error("Unknown status should return base style")
	}
}

// =============================================================================
// Message Type Tests
// =============================================================================

func TestMessageTypes(t *testing.T) {
	// These should all compile and be distinct types
	_ = ShowHelpMsg{}
	_ = ToggleGroupingMsg{}
	_ = StartSearchMsg{}
	_ = StartInputMsg{}
	_ = TerminateAgentMsg{}
	_ = RefreshMsg{}
	_ = ToggleStatsMsg{}
	_ = ToggleAlertsMsg{}
	_ = AgentSelectedMsg{Agent: nil}
	_ = AgentListRefreshMsg{}
	_ = ViewportRefreshMsg{}
}

// =============================================================================
// AgentItem Tests
// =============================================================================

func TestAgentItem(t *testing.T) {
	mockAg := newMockAgent("test-id", "test-agent", agent.StatusRunning)
	item := AgentItem{Agent: mockAg, ChildCount: 0}

	if item.Title() != "test-agent" {
		t.Errorf("Title() = %q, want test-agent", item.Title())
	}

	desc := item.Description()
	if !strings.Contains(desc, "●") { // Running icon
		t.Error("Description should contain running status icon")
	}

	filterValue := item.FilterValue()
	if !strings.Contains(filterValue, "test-agent") {
		t.Error("FilterValue should contain agent name")
	}

	itemWithChildren := AgentItem{Agent: mockAg, ChildCount: 3}
	titleWithChildren := itemWithChildren.Title()
	if !strings.Contains(titleWithChildren, "[3]") {
		t.Errorf("Title with children should contain [3], got %q", titleWithChildren)
	}
}

// =============================================================================
// FormatDuration Tests
// =============================================================================

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{time.Hour + 30*time.Minute, "1h30m"},
		{2*time.Hour + 45*time.Minute, "2h45m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

// =============================================================================
// StatusIndicator Tests
// =============================================================================

func TestStatusIndicator(t *testing.T) {
	theme := DefaultDarkTheme()

	tests := []struct {
		status agent.Status
		icon   string
	}{
		{agent.StatusRunning, "●"},
		{agent.StatusIdle, "◌"},
		{agent.StatusCompleted, "✓"},
		{agent.StatusErrored, "✗"},
		{agent.StatusPending, "○"},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			indicator := StatusIndicator(theme, tt.status)
			if !strings.Contains(indicator, tt.icon) {
				t.Errorf("StatusIndicator(%v) should contain %q", tt.status, tt.icon)
			}
		})
	}
}
