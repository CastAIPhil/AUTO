package plugin

import (
	"context"
	"testing"

	"github.com/CastAIPhil/AUTO/internal/agent"
)

// MockPlugin implements Plugin interface for testing
type MockPlugin struct {
	*BasePlugin
	agents []agent.Agent
}

func NewMockPlugin(name, version string) *MockPlugin {
	return &MockPlugin{
		BasePlugin: NewBasePlugin(Info{
			Name:        name,
			Version:     version,
			Description: "Mock plugin for testing",
			Author:      "test",
			Type:        "mock",
		}),
		agents: make([]agent.Agent, 0),
	}
}

func (p *MockPlugin) Discover(ctx context.Context) ([]agent.Agent, error) {
	return p.agents, nil
}

func (p *MockPlugin) Watch(ctx context.Context) (<-chan agent.Event, error) {
	ch := make(chan agent.Event)
	close(ch)
	return ch, nil
}

func (p *MockPlugin) Spawn(ctx context.Context, config agent.SpawnConfig) (agent.Agent, error) {
	return nil, nil
}

func (p *MockPlugin) Get(id string) (agent.Agent, error) {
	return nil, nil
}

func (p *MockPlugin) List() []agent.Agent {
	return p.agents
}

func (p *MockPlugin) Terminate(id string) error {
	return nil
}

func (p *MockPlugin) SendInput(id string, input string) error {
	return nil
}

func TestNewManager(t *testing.T) {
	registry := agent.NewRegistry()
	m := NewManager(registry)

	if m == nil {
		t.Fatal("NewManager() returned nil")
	}

	if len(m.List()) != 0 {
		t.Error("New manager should have no plugins")
	}
}

func TestManagerRegister(t *testing.T) {
	m := NewManager(nil)
	plugin := NewMockPlugin("test-plugin", "1.0.0")

	err := m.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	if len(m.List()) != 1 {
		t.Error("Manager should have 1 plugin after registration")
	}

	// Test duplicate registration
	err = m.Register(plugin)
	if err == nil {
		t.Error("Register() should error on duplicate")
	}
}

func TestManagerRegisterEmptyName(t *testing.T) {
	m := NewManager(nil)
	plugin := NewMockPlugin("", "1.0.0")

	err := m.Register(plugin)
	if err == nil {
		t.Error("Register() should error on empty name")
	}
}

func TestManagerUnregister(t *testing.T) {
	m := NewManager(nil)
	plugin := NewMockPlugin("test-plugin", "1.0.0")

	m.Register(plugin)
	err := m.Unregister("test-plugin")
	if err != nil {
		t.Errorf("Unregister() error = %v", err)
	}

	if len(m.List()) != 0 {
		t.Error("Manager should have 0 plugins after unregister")
	}

	// Test unregister non-existent
	err = m.Unregister("non-existent")
	if err == nil {
		t.Error("Unregister() should error on non-existent plugin")
	}
}

func TestManagerGet(t *testing.T) {
	m := NewManager(nil)
	plugin := NewMockPlugin("test-plugin", "1.0.0")
	m.Register(plugin)

	got, ok := m.Get("test-plugin")
	if !ok {
		t.Error("Get() should find registered plugin")
	}
	if got.Info().Name != "test-plugin" {
		t.Error("Get() returned wrong plugin")
	}

	_, ok = m.Get("non-existent")
	if ok {
		t.Error("Get() should not find non-existent plugin")
	}
}

func TestManagerListInfo(t *testing.T) {
	m := NewManager(nil)
	m.Register(NewMockPlugin("plugin-1", "1.0.0"))
	m.Register(NewMockPlugin("plugin-2", "2.0.0"))

	infos := m.ListInfo()
	if len(infos) != 2 {
		t.Errorf("ListInfo() returned %d, want 2", len(infos))
	}
}

func TestManagerDiscoverAll(t *testing.T) {
	m := NewManager(nil)
	plugin := NewMockPlugin("test-plugin", "1.0.0")
	plugin.agents = []agent.Agent{agent.NewMockAgent("a1", "Agent 1")}
	m.Register(plugin)

	agents, err := m.DiscoverAll(context.Background())
	if err != nil {
		t.Errorf("DiscoverAll() error = %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("DiscoverAll() returned %d agents, want 1", len(agents))
	}
}

func TestBasePlugin(t *testing.T) {
	info := Info{
		Name:        "test",
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      "test",
		Type:        "test-type",
	}
	bp := NewBasePlugin(info)

	if bp.Name() != "test" {
		t.Errorf("Name() = %v, want test", bp.Name())
	}
	if bp.Type() != "test-type" {
		t.Errorf("Type() = %v, want test-type", bp.Type())
	}
	if bp.Info().Version != "1.0.0" {
		t.Errorf("Info().Version = %v, want 1.0.0", bp.Info().Version)
	}
}

func TestManagerWithRegistry(t *testing.T) {
	registry := agent.NewRegistry()
	m := NewManager(registry)

	plugin := NewMockPlugin("test-plugin", "1.0.0")
	m.Register(plugin)

	// Plugin should also be registered in agent registry
	_, found := registry.Get("mock")
	if !found {
		t.Error("Plugin should be registered in agent registry")
	}
}
