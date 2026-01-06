package alert

import (
	"context"
	"testing"
	"time"

	"github.com/CastAIPhil/AUTO/internal/agent"
	"github.com/CastAIPhil/AUTO/internal/config"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.AlertsConfig
		wantChannels int
	}{
		{
			name:         "no channels",
			cfg:          &config.AlertsConfig{},
			wantChannels: 0,
		},
		{
			name: "desktop only",
			cfg: &config.AlertsConfig{
				DesktopNotifications: true,
			},
			wantChannels: 1,
		},
		{
			name: "slack enabled but no webhook",
			cfg: &config.AlertsConfig{
				SlackEnabled: true,
			},
			wantChannels: 0,
		},
		{
			name: "slack with webhook",
			cfg: &config.AlertsConfig{
				SlackEnabled:    true,
				SlackWebhookURL: "https://hooks.slack.com/test",
			},
			wantChannels: 1,
		},
		{
			name: "discord with webhook",
			cfg: &config.AlertsConfig{
				DiscordEnabled:    true,
				DiscordWebhookURL: "https://discord.com/api/webhooks/test",
			},
			wantChannels: 1,
		},
		{
			name: "all channels",
			cfg: &config.AlertsConfig{
				DesktopNotifications: true,
				SlackEnabled:         true,
				SlackWebhookURL:      "https://hooks.slack.com/test",
				DiscordEnabled:       true,
				DiscordWebhookURL:    "https://discord.com/api/webhooks/test",
			},
			wantChannels: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.cfg, nil)
			if len(m.channels) != tt.wantChannels {
				t.Errorf("NewManager() channels = %d, want %d", len(m.channels), tt.wantChannels)
			}
		})
	}
}

func TestManagerSend(t *testing.T) {
	cfg := &config.AlertsConfig{}
	m := NewManager(cfg, nil)

	// Track callback
	var received *Alert
	m.OnAlert(func(a *Alert) {
		received = a
	})

	alert := &Alert{
		Level:   LevelInfo,
		Title:   "Test Alert",
		Message: "This is a test",
	}

	err := m.Send(context.Background(), alert)
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Check ID was generated
	if alert.ID == "" {
		t.Error("Send() did not generate alert ID")
	}

	// Check timestamp was set
	if alert.Timestamp.IsZero() {
		t.Error("Send() did not set timestamp")
	}

	// Check callback was called
	if received == nil {
		t.Error("Send() did not trigger callback")
	}
	if received.ID != alert.ID {
		t.Error("Send() callback received different alert")
	}
}

func TestManagerList(t *testing.T) {
	cfg := &config.AlertsConfig{}
	m := NewManager(cfg, nil)

	// Add some alerts
	for i := 0; i < 5; i++ {
		m.Send(context.Background(), &Alert{
			Level:   LevelInfo,
			Title:   "Test",
			Message: "Test",
		})
	}

	// Mark some as read
	alerts := m.List(0, false)
	if len(alerts) > 0 {
		m.MarkRead(alerts[0].ID)
	}

	// Test list all
	all := m.List(0, false)
	if len(all) != 5 {
		t.Errorf("List(0, false) = %d, want 5", len(all))
	}

	// Test list with limit
	limited := m.List(3, false)
	if len(limited) != 3 {
		t.Errorf("List(3, false) = %d, want 3", len(limited))
	}

	// Test list unread only
	unread := m.List(0, true)
	if len(unread) != 4 {
		t.Errorf("List(0, true) = %d, want 4", len(unread))
	}
}

func TestManagerMarkRead(t *testing.T) {
	cfg := &config.AlertsConfig{}
	m := NewManager(cfg, nil)

	// Add alert
	alert := &Alert{Level: LevelInfo, Title: "Test", Message: "Test"}
	m.Send(context.Background(), alert)

	// Check unread count
	if m.UnreadCount() != 1 {
		t.Errorf("UnreadCount() = %d, want 1", m.UnreadCount())
	}

	// Mark read
	m.MarkRead(alert.ID)

	// Check unread count again
	if m.UnreadCount() != 0 {
		t.Errorf("UnreadCount() after MarkRead = %d, want 0", m.UnreadCount())
	}
}

func TestManagerMarkAllRead(t *testing.T) {
	cfg := &config.AlertsConfig{}
	m := NewManager(cfg, nil)

	// Add multiple alerts
	for i := 0; i < 5; i++ {
		m.Send(context.Background(), &Alert{Level: LevelInfo, Title: "Test", Message: "Test"})
	}

	if m.UnreadCount() != 5 {
		t.Errorf("UnreadCount() = %d, want 5", m.UnreadCount())
	}

	m.MarkAllRead()

	if m.UnreadCount() != 0 {
		t.Errorf("UnreadCount() after MarkAllRead = %d, want 0", m.UnreadCount())
	}
}

func TestManagerSendAgentEvent(t *testing.T) {
	cfg := &config.AlertsConfig{}
	m := NewManager(cfg, nil)

	mockAgent := agent.NewMockAgent("test-1", "Test Agent")

	tests := []struct {
		name      string
		eventType agent.EventType
		wantLevel Level
		wantAlert bool
	}{
		{
			name:      "error event",
			eventType: agent.EventAgentErrored,
			wantLevel: LevelError,
			wantAlert: true,
		},
		{
			name:      "completed event",
			eventType: agent.EventAgentCompleted,
			wantLevel: LevelSuccess,
			wantAlert: true,
		},
		{
			name:      "context limit event",
			eventType: agent.EventAgentContextLimit,
			wantLevel: LevelWarning,
			wantAlert: true,
		},
		{
			name:      "discovered event (no alert)",
			eventType: agent.EventAgentDiscovered,
			wantAlert: false,
		},
		{
			name:      "updated event (no alert)",
			eventType: agent.EventAgentUpdated,
			wantAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initialCount := len(m.List(0, false))

			event := agent.Event{
				Type:      tt.eventType,
				AgentID:   mockAgent.ID(),
				Agent:     mockAgent,
				Timestamp: time.Now(),
			}

			m.SendAgentEvent(context.Background(), event)

			newCount := len(m.List(0, false))

			if tt.wantAlert {
				if newCount != initialCount+1 {
					t.Errorf("SendAgentEvent() alert count = %d, want %d", newCount, initialCount+1)
				}
				// Check the level of the last alert
				alerts := m.List(1, false)
				if len(alerts) > 0 && alerts[0].Level != tt.wantLevel {
					t.Errorf("SendAgentEvent() alert level = %v, want %v", alerts[0].Level, tt.wantLevel)
				}
			} else {
				if newCount != initialCount {
					t.Errorf("SendAgentEvent() created alert when it shouldn't have")
				}
			}
		})
	}
}

func TestAlertMemoryLimit(t *testing.T) {
	cfg := &config.AlertsConfig{}
	m := NewManager(cfg, nil)

	// Add more than 1000 alerts
	for i := 0; i < 1100; i++ {
		m.Send(context.Background(), &Alert{Level: LevelInfo, Title: "Test", Message: "Test"})
	}

	// Should be capped at 1000
	all := m.List(0, false)
	if len(all) != 1000 {
		t.Errorf("Alert memory limit not enforced: got %d, want 1000", len(all))
	}
}

func TestDesktopChannelName(t *testing.T) {
	c := &DesktopChannel{}
	if c.Name() != "desktop" {
		t.Errorf("DesktopChannel.Name() = %s, want desktop", c.Name())
	}
}

func TestSlackChannelName(t *testing.T) {
	c := &SlackChannel{}
	if c.Name() != "slack" {
		t.Errorf("SlackChannel.Name() = %s, want slack", c.Name())
	}
}

func TestDiscordChannelName(t *testing.T) {
	c := &DiscordChannel{}
	if c.Name() != "discord" {
		t.Errorf("DiscordChannel.Name() = %s, want discord", c.Name())
	}
}
