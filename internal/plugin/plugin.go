// Package plugin provides an extensible plugin system for AUTO
package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/localrivet/auto/internal/agent"
)

// Info contains metadata about a plugin
type Info struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Type        string `json:"type"` // Provider type identifier
}

// Plugin extends agent.Provider with metadata
type Plugin interface {
	agent.Provider
	Info() Info
}

// Manager handles plugin lifecycle
type Manager struct {
	plugins  map[string]Plugin
	registry *agent.Registry
	mu       sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager(registry *agent.Registry) *Manager {
	return &Manager{
		plugins:  make(map[string]Plugin),
		registry: registry,
	}
}

// Register adds a plugin to the manager
func (m *Manager) Register(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info := plugin.Info()
	if info.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if _, exists := m.plugins[info.Name]; exists {
		return fmt.Errorf("plugin %s already registered", info.Name)
	}

	m.plugins[info.Name] = plugin

	// Also register with the agent registry
	if m.registry != nil {
		m.registry.Register(plugin)
	}

	return nil
}

// Unregister removes a plugin from the manager
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	delete(m.plugins, name)
	return nil
}

// Get returns a plugin by name
func (m *Manager) Get(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, ok := m.plugins[name]
	return plugin, ok
}

// List returns all registered plugins
func (m *Manager) List() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// ListInfo returns info for all registered plugins
func (m *Manager) ListInfo() []Info {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]Info, 0, len(m.plugins))
	for _, p := range m.plugins {
		infos = append(infos, p.Info())
	}
	return infos
}

// DiscoverAll discovers agents from all plugins
func (m *Manager) DiscoverAll(ctx context.Context) ([]agent.Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allAgents []agent.Agent
	for _, p := range m.plugins {
		agents, err := p.Discover(ctx)
		if err != nil {
			continue
		}
		allAgents = append(allAgents, agents...)
	}
	return allAgents, nil
}

// BasePlugin provides a base implementation for plugins
type BasePlugin struct {
	info Info
}

// NewBasePlugin creates a new base plugin with the given info
func NewBasePlugin(info Info) *BasePlugin {
	return &BasePlugin{info: info}
}

// Info returns the plugin info
func (p *BasePlugin) Info() Info {
	return p.info
}

// Name returns the plugin name
func (p *BasePlugin) Name() string {
	return p.info.Name
}

// Type returns the plugin type
func (p *BasePlugin) Type() string {
	return p.info.Type
}
