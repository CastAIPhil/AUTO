// Package components provides TUI components
package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/localrivet/auto/internal/agent"
	"github.com/localrivet/auto/internal/config"
)

// Theme holds all the styles for the TUI
type Theme struct {
	// Colors
	Primary       lipgloss.Color
	Secondary     lipgloss.Color
	Accent        lipgloss.Color
	Background    lipgloss.Color
	Foreground    lipgloss.Color
	Border        lipgloss.Color
	BorderFocused lipgloss.Color
	StatusRunning lipgloss.Color
	StatusIdle    lipgloss.Color
	StatusError   lipgloss.Color
	StatusDone    lipgloss.Color

	// Base styles
	Base        lipgloss.Style
	Title       lipgloss.Style
	Subtitle    lipgloss.Style
	Header      lipgloss.Style
	Footer      lipgloss.Style
	BorderStyle lipgloss.Style

	// Component styles
	AgentListStyle    lipgloss.Style
	ViewportStyle     lipgloss.Style
	StatsStyle        lipgloss.Style
	AlertsStyle       lipgloss.Style
	InputStyle        lipgloss.Style
	CommandStyle      lipgloss.Style
	HelpStyle         lipgloss.Style
	SelectedItemStyle lipgloss.Style
	NormalItemStyle   lipgloss.Style

	// Status styles
	StatusStyles map[agent.Status]lipgloss.Style
}

// NewTheme creates a new theme from config
func NewTheme(cfg *config.ThemeConfig) *Theme {
	t := &Theme{
		Primary:       lipgloss.Color(cfg.Colors.Primary),
		Secondary:     lipgloss.Color(cfg.Colors.Secondary),
		Accent:        lipgloss.Color(cfg.Colors.Accent),
		Background:    lipgloss.Color(cfg.Colors.Background),
		Foreground:    lipgloss.Color(cfg.Colors.Foreground),
		Border:        lipgloss.Color(cfg.Colors.Border),
		BorderFocused: lipgloss.Color(cfg.Colors.BorderFocused),
		StatusRunning: lipgloss.Color(cfg.Colors.StatusRunning),
		StatusIdle:    lipgloss.Color(cfg.Colors.StatusIdle),
		StatusError:   lipgloss.Color(cfg.Colors.StatusError),
		StatusDone:    lipgloss.Color(cfg.Colors.StatusDone),
	}

	// Initialize styles
	t.initStyles()

	return t
}

// initStyles initializes all the lipgloss styles
func (t *Theme) initStyles() {
	t.Base = lipgloss.NewStyle().
		Foreground(t.Foreground)

	t.Title = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1)

	t.Subtitle = lipgloss.NewStyle().
		Foreground(t.Secondary).
		Padding(0, 1)

	t.Header = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Border).
		Padding(0, 1)

	t.Footer = lipgloss.NewStyle().
		Foreground(t.Secondary).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(t.Border).
		Padding(0, 1)

	t.BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)

	t.AgentListStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)

	t.ViewportStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)

	t.StatsStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)

	t.AlertsStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1)

	t.InputStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderFocused).
		Padding(0, 1)

	t.CommandStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2)

	t.HelpStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2)

	t.SelectedItemStyle = lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Primary).
		Bold(true).
		Padding(0, 1)

	t.NormalItemStyle = lipgloss.NewStyle().
		Foreground(t.Foreground).
		Padding(0, 1)

	// Status styles
	t.StatusStyles = map[agent.Status]lipgloss.Style{
		agent.StatusPending: lipgloss.NewStyle().
			Foreground(t.Secondary),
		agent.StatusRunning: lipgloss.NewStyle().
			Foreground(t.StatusRunning).
			Bold(true),
		agent.StatusIdle: lipgloss.NewStyle().
			Foreground(t.StatusIdle),
		agent.StatusCompleted: lipgloss.NewStyle().
			Foreground(t.StatusDone),
		agent.StatusErrored: lipgloss.NewStyle().
			Foreground(t.StatusError).
			Bold(true),
		agent.StatusContextLimit: lipgloss.NewStyle().
			Foreground(t.StatusError),
		agent.StatusCancelled: lipgloss.NewStyle().
			Foreground(t.Secondary),
	}
}

// StatusStyle returns the style for a status
func (t *Theme) StatusStyle(status agent.Status) lipgloss.Style {
	if style, ok := t.StatusStyles[status]; ok {
		return style
	}
	return t.Base
}

// FocusedBorder returns a style with focused border
func (t *Theme) FocusedBorder(base lipgloss.Style) lipgloss.Style {
	return base.Copy().BorderForeground(t.BorderFocused)
}

// DefaultDarkTheme returns the default dark theme
func DefaultDarkTheme() *Theme {
	return NewTheme(&config.ThemeConfig{
		Mode: "dark",
		Colors: config.ColorsConfig{
			Primary:       "#7C3AED",
			Secondary:     "#A78BFA",
			Accent:        "#10B981",
			Background:    "#1E1E2E",
			Foreground:    "#CDD6F4",
			Border:        "#45475A",
			BorderFocused: "#7C3AED",
			StatusRunning: "#10B981",
			StatusIdle:    "#F59E0B",
			StatusError:   "#EF4444",
			StatusDone:    "#3B82F6",
		},
	})
}

// DefaultLightTheme returns the default light theme
func DefaultLightTheme() *Theme {
	return NewTheme(&config.ThemeConfig{
		Mode: "light",
		Colors: config.ColorsConfig{
			Primary:       "#7C3AED",
			Secondary:     "#8B5CF6",
			Accent:        "#059669",
			Background:    "#FFFFFF",
			Foreground:    "#1F2937",
			Border:        "#E5E7EB",
			BorderFocused: "#7C3AED",
			StatusRunning: "#059669",
			StatusIdle:    "#D97706",
			StatusError:   "#DC2626",
			StatusDone:    "#2563EB",
		},
	})
}
