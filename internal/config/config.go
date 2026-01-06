// Package config handles application configuration
package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	General   GeneralConfig   `yaml:"general"`
	Providers ProvidersConfig `yaml:"providers"`
	Alerts    AlertsConfig    `yaml:"alerts"`
	UI        UIConfig        `yaml:"ui"`
	Theme     ThemeConfig     `yaml:"theme"`
	Keys      KeysConfig      `yaml:"keys"`
	Storage   StorageConfig   `yaml:"storage"`
	Metrics   MetricsConfig   `yaml:"metrics"`
}

// GeneralConfig holds general settings
type GeneralConfig struct {
	RefreshInterval time.Duration `yaml:"refresh_interval"`
	LogLevel        string        `yaml:"log_level"`
	LogFile         string        `yaml:"log_file"`
}

// ProvidersConfig holds provider-specific settings
type ProvidersConfig struct {
	OpenCode OpenCodeConfig `yaml:"opencode"`
}

// OpenCodeConfig holds opencode provider settings
type OpenCodeConfig struct {
	Enabled       bool          `yaml:"enabled"`
	StoragePath   string        `yaml:"storage_path"`
	WatchInterval time.Duration `yaml:"watch_interval"`
}

// AlertsConfig holds alert settings
type AlertsConfig struct {
	ContextLimitWarning  int           `yaml:"context_limit_warning"` // percentage
	LongRunningThreshold time.Duration `yaml:"long_running_threshold"`
	SoundEnabled         bool          `yaml:"sound_enabled"`
	DesktopNotifications bool          `yaml:"desktop_notifications"`
	SlackEnabled         bool          `yaml:"slack_enabled"`
	SlackWebhookURL      string        `yaml:"slack_webhook_url"`
	SlackChannel         string        `yaml:"slack_channel"`
	DiscordEnabled       bool          `yaml:"discord_enabled"`
	DiscordWebhookURL    string        `yaml:"discord_webhook_url"`
}

// UIConfig holds UI settings
type UIConfig struct {
	ShowHeader      bool   `yaml:"show_header"`
	ShowFooter      bool   `yaml:"show_footer"`
	AgentListWidth  int    `yaml:"agent_list_width"`
	DefaultGrouping string `yaml:"default_grouping"` // "flat", "type", "project"
	ShowMetrics     bool   `yaml:"show_metrics"`
}

// ThemeConfig holds theme settings
type ThemeConfig struct {
	Mode   string       `yaml:"mode"` // "dark", "light"
	Colors ColorsConfig `yaml:"colors"`
}

// ColorsConfig holds color customizations
type ColorsConfig struct {
	Primary       string `yaml:"primary"`
	Secondary     string `yaml:"secondary"`
	Accent        string `yaml:"accent"`
	Background    string `yaml:"background"`
	Foreground    string `yaml:"foreground"`
	Border        string `yaml:"border"`
	BorderFocused string `yaml:"border_focused"`
	StatusRunning string `yaml:"status_running"`
	StatusIdle    string `yaml:"status_idle"`
	StatusError   string `yaml:"status_error"`
	StatusDone    string `yaml:"status_done"`
}

// KeysConfig holds keybinding customizations
type KeysConfig struct {
	Quit           string `yaml:"quit"`
	Help           string `yaml:"help"`
	Search         string `yaml:"search"`
	Command        string `yaml:"command"`
	NextAgent      string `yaml:"next_agent"`
	PrevAgent      string `yaml:"prev_agent"`
	FocusAgent     string `yaml:"focus_agent"`
	TerminateAgent string `yaml:"terminate_agent"`
	PauseAgent     string `yaml:"pause_agent"`
	SendInput      string `yaml:"send_input"`
	ToggleGrouping string `yaml:"toggle_grouping"`
	SwitchPane     string `yaml:"switch_pane"`
}

// StorageConfig holds storage settings
type StorageConfig struct {
	DatabasePath string `yaml:"database_path"`
	MaxHistory   int    `yaml:"max_history"` // days to keep
}

// MetricsConfig holds metrics settings
type MetricsConfig struct {
	TokenCostInput  float64 `yaml:"token_cost_input"`  // cost per 1k input tokens
	TokenCostOutput float64 `yaml:"token_cost_output"` // cost per 1k output tokens
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		General: GeneralConfig{
			RefreshInterval: 5 * time.Second,
			LogLevel:        "info",
			LogFile:         "",
		},
		Providers: ProvidersConfig{
			OpenCode: OpenCodeConfig{
				Enabled:       true,
				StoragePath:   filepath.Join(homeDir, ".local", "share", "opencode", "storage"),
				WatchInterval: 1 * time.Second,
			},
		},
		Alerts: AlertsConfig{
			ContextLimitWarning:  90,
			LongRunningThreshold: 30 * time.Minute,
			SoundEnabled:         false,
			DesktopNotifications: true,
			SlackEnabled:         false,
			DiscordEnabled:       false,
		},
		UI: UIConfig{
			ShowHeader:      true,
			ShowFooter:      true,
			AgentListWidth:  30,
			DefaultGrouping: "type",
			ShowMetrics:     true,
		},
		Theme: ThemeConfig{
			Mode: "dark",
			Colors: ColorsConfig{
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
		},
		Keys: KeysConfig{
			Quit:           "q",
			Help:           "?",
			Search:         "/",
			Command:        ":",
			NextAgent:      "j",
			PrevAgent:      "k",
			FocusAgent:     "enter",
			TerminateAgent: "x",
			PauseAgent:     "space",
			SendInput:      "i",
			ToggleGrouping: "g",
			SwitchPane:     "tab",
		},
		Storage: StorageConfig{
			DatabasePath: filepath.Join(homeDir, ".local", "share", "auto", "auto.db"),
			MaxHistory:   30,
		},
		Metrics: MetricsConfig{
			TokenCostInput:  0.003, // $3 per 1M tokens
			TokenCostOutput: 0.015, // $15 per 1M tokens
		},
	}
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Return defaults if no config file
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves configuration to a file
func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ConfigPath returns the default config file path
func ConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "auto", "config.yaml")
}
