package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/localrivet/auto/internal/agent"
	"github.com/localrivet/auto/internal/config"
	"github.com/localrivet/auto/internal/session"
)

func setupTestServer() (*Server, *session.Manager) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	manager := session.NewManager(cfg, nil, registry, nil)

	// Add some mock agents
	agent1 := agent.NewMockAgent("agent-1", "Test Agent 1")
	agent1.MockStatus = agent.StatusRunning
	agent2 := agent.NewMockAgent("agent-2", "Test Agent 2")
	agent2.MockStatus = agent.StatusIdle

	manager.AddAgentForTesting(agent1)
	manager.AddAgentForTesting(agent2)

	server := NewServer(manager, ":0")
	return server, manager
}

func TestNewServer(t *testing.T) {
	cfg := &config.Config{}
	registry := agent.NewRegistry()
	manager := session.NewManager(cfg, nil, registry, nil)

	server := NewServer(manager, ":8080")
	if server == nil {
		t.Fatal("NewServer() returned nil")
	}
	if server.Addr() != ":8080" {
		t.Errorf("Addr() = %v, want :8080", server.Addr())
	}
}

func TestHandleHealth(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Error("response should be successful")
	}
}

func TestHandleHealthMethodNotAllowed(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAgents(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	w := httptest.NewRecorder()

	server.handleAgents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Error("response should be successful")
	}

	agents, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("data should be array")
	}
	if len(agents) != 2 {
		t.Errorf("got %d agents, want 2", len(agents))
	}
}

func TestHandleAgent(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/agents/agent-1", nil)
	w := httptest.NewRecorder()

	server.handleAgent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Error("response should be successful")
	}
}

func TestHandleAgentNotFound(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/agents/non-existent", nil)
	w := httptest.NewRecorder()

	server.handleAgent(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleStats(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	w := httptest.NewRecorder()

	server.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Error("response should be successful")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("data should be object")
	}
	if data["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", data["total"])
	}
}

func TestHandleAgentTerminate(t *testing.T) {
	server, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/agents/agent-1/terminate", nil)
	w := httptest.NewRecorder()

	server.handleAgent(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
