// Package alert handles multi-channel alerting
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/CastAIPhil/AUTO/internal/agent"
	"github.com/CastAIPhil/AUTO/internal/config"
	"github.com/CastAIPhil/AUTO/internal/store"
	"github.com/gen2brain/beeep"
	"github.com/slack-go/slack"
)

// Level represents alert severity
type Level string

const (
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
	LevelSuccess Level = "success"
)

// Alert represents an alert message
type Alert struct {
	ID        string
	Level     Level
	Title     string
	Message   string
	AgentID   string
	Agent     agent.Agent
	Timestamp time.Time
	Read      bool
}

// Channel represents an alert channel
type Channel interface {
	Send(ctx context.Context, alert *Alert) error
	Name() string
}

// Manager manages alert channels and distribution
type Manager struct {
	cfg      *config.AlertsConfig
	store    *store.Store
	channels []Channel
	alerts   []*Alert
	mu       sync.RWMutex
	onAlert  func(*Alert)
}

// NewManager creates a new alert manager
func NewManager(cfg *config.AlertsConfig, st *store.Store) *Manager {
	m := &Manager{
		cfg:    cfg,
		store:  st,
		alerts: make([]*Alert, 0),
	}

	// Initialize channels based on config
	if cfg.DesktopNotifications {
		m.channels = append(m.channels, &DesktopChannel{})
	}
	if cfg.SlackEnabled && cfg.SlackWebhookURL != "" {
		m.channels = append(m.channels, &SlackChannel{
			webhookURL: cfg.SlackWebhookURL,
			channel:    cfg.SlackChannel,
		})
	}
	if cfg.DiscordEnabled && cfg.DiscordWebhookURL != "" {
		m.channels = append(m.channels, &DiscordChannel{
			webhookURL: cfg.DiscordWebhookURL,
			httpClient: &http.Client{Timeout: 10 * time.Second},
		})
	}

	return m
}

// OnAlert sets the callback for new alerts (for TUI)
func (m *Manager) OnAlert(fn func(*Alert)) {
	m.onAlert = fn
}

// Send sends an alert to all configured channels
func (m *Manager) Send(ctx context.Context, alert *Alert) error {
	// Generate ID if not set
	if alert.ID == "" {
		alert.ID = fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}
	if alert.Timestamp.IsZero() {
		alert.Timestamp = time.Now()
	}

	// Store the alert
	m.mu.Lock()
	m.alerts = append(m.alerts, alert)
	// Keep only last 1000 alerts in memory
	if len(m.alerts) > 1000 {
		m.alerts = m.alerts[len(m.alerts)-1000:]
	}
	m.mu.Unlock()

	// Persist to database
	if m.store != nil {
		m.store.SaveAlert(&store.AlertRecord{
			ID:        alert.ID,
			AgentID:   alert.AgentID,
			Level:     string(alert.Level),
			Message:   fmt.Sprintf("%s: %s", alert.Title, alert.Message),
			Timestamp: alert.Timestamp,
			Read:      false,
		})
	}

	// Notify TUI callback
	if m.onAlert != nil {
		m.onAlert(alert)
	}

	// Send to all channels
	var lastErr error
	for _, ch := range m.channels {
		if err := ch.Send(ctx, alert); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// SendAgentEvent sends an alert for an agent event
func (m *Manager) SendAgentEvent(ctx context.Context, event agent.Event) error {
	var level Level
	var title string
	var message string

	switch event.Type {
	case agent.EventAgentErrored:
		level = LevelError
		title = "Agent Error"
		if event.Agent != nil {
			message = fmt.Sprintf("Agent %s encountered an error", event.Agent.Name())
			if event.Agent.LastError() != nil {
				message += ": " + event.Agent.LastError().Error()
			}
		}
	case agent.EventAgentCompleted:
		level = LevelSuccess
		title = "Agent Completed"
		if event.Agent != nil {
			message = fmt.Sprintf("Agent %s completed its task", event.Agent.Name())
		}
	case agent.EventAgentContextLimit:
		level = LevelWarning
		title = "Context Limit Warning"
		if event.Agent != nil {
			message = fmt.Sprintf("Agent %s is approaching context limit", event.Agent.Name())
		}
	default:
		// Don't alert for other events
		return nil
	}

	alert := &Alert{
		Level:     level,
		Title:     title,
		Message:   message,
		AgentID:   event.AgentID,
		Agent:     event.Agent,
		Timestamp: event.Timestamp,
	}

	return m.Send(ctx, alert)
}

// List returns all alerts
func (m *Manager) List(limit int, unreadOnly bool) []*Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Alert
	for i := len(m.alerts) - 1; i >= 0 && (limit <= 0 || len(result) < limit); i-- {
		if unreadOnly && m.alerts[i].Read {
			continue
		}
		result = append(result, m.alerts[i])
	}

	return result
}

// MarkRead marks an alert as read
func (m *Manager) MarkRead(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.alerts {
		if a.ID == id {
			a.Read = true
			break
		}
	}

	if m.store != nil {
		m.store.MarkAlertRead(id)
	}
}

// MarkAllRead marks all alerts as read
func (m *Manager) MarkAllRead() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, a := range m.alerts {
		a.Read = true
	}

	if m.store != nil {
		m.store.MarkAllAlertsRead()
	}
}

// UnreadCount returns the number of unread alerts
func (m *Manager) UnreadCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, a := range m.alerts {
		if !a.Read {
			count++
		}
	}
	return count
}

// DesktopChannel sends desktop notifications
type DesktopChannel struct{}

func (c *DesktopChannel) Name() string {
	return "desktop"
}

func (c *DesktopChannel) Send(ctx context.Context, alert *Alert) error {
	return beeep.Notify(alert.Title, alert.Message, "")
}

// SlackChannel sends Slack notifications
type SlackChannel struct {
	webhookURL string
	channel    string
}

func (c *SlackChannel) Name() string {
	return "slack"
}

func (c *SlackChannel) Send(ctx context.Context, alert *Alert) error {
	color := "#2196F3" // info blue
	switch alert.Level {
	case LevelWarning:
		color = "#FF9800" // warning orange
	case LevelError:
		color = "#F44336" // error red
	case LevelSuccess:
		color = "#4CAF50" // success green
	}

	attachment := slack.Attachment{
		Color:      color,
		Title:      alert.Title,
		Text:       alert.Message,
		Footer:     "AUTO",
		Ts:         json.Number(fmt.Sprintf("%d", alert.Timestamp.Unix())),
		FooterIcon: "",
	}

	if alert.Agent != nil {
		attachment.Fields = []slack.AttachmentField{
			{
				Title: "Agent",
				Value: alert.Agent.Name(),
				Short: true,
			},
			{
				Title: "Status",
				Value: alert.Agent.Status().String(),
				Short: true,
			},
		}
	}

	msg := slack.WebhookMessage{
		Channel:     c.channel,
		Attachments: []slack.Attachment{attachment},
	}

	return slack.PostWebhookContext(ctx, c.webhookURL, &msg)
}

// DiscordChannel sends Discord notifications
type DiscordChannel struct {
	webhookURL string
	httpClient *http.Client
}

func (c *DiscordChannel) Name() string {
	return "discord"
}

func (c *DiscordChannel) Send(ctx context.Context, alert *Alert) error {
	color := 2201331 // info blue
	switch alert.Level {
	case LevelWarning:
		color = 16750592 // warning orange
	case LevelError:
		color = 15158332 // error red
	case LevelSuccess:
		color = 3066993 // success green
	}

	embed := map[string]interface{}{
		"title":       alert.Title,
		"description": alert.Message,
		"color":       color,
		"timestamp":   alert.Timestamp.Format(time.RFC3339),
		"footer": map[string]string{
			"text": "AUTO",
		},
	}

	if alert.Agent != nil {
		embed["fields"] = []map[string]interface{}{
			{
				"name":   "Agent",
				"value":  alert.Agent.Name(),
				"inline": true,
			},
			{
				"name":   "Status",
				"value":  alert.Agent.Status().String(),
				"inline": true,
			},
		}
	}

	payload := map[string]interface{}{
		"embeds": []interface{}{embed},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := c.httpClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}
