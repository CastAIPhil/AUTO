// Package api provides HTTP/WebSocket API for AUTO
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/CastAIPhil/AUTO/internal/session"
)

// Server provides the HTTP API
type Server struct {
	manager    *session.Manager
	addr       string
	httpServer *http.Server
	mu         sync.RWMutex
	clients    map[*wsClient]bool
}

// wsClient represents a WebSocket client
type wsClient struct {
	send chan []byte
}

// NewServer creates a new API server
func NewServer(manager *session.Manager, addr string) *Server {
	s := &Server{
		manager: manager,
		addr:    addr,
		clients: make(map[*wsClient]bool),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgent)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.addr
}

// Response is the standard API response format
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// AgentResponse represents an agent in API responses
type AgentResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Status       string    `json:"status"`
	Directory    string    `json:"directory"`
	ProjectID    string    `json:"project_id"`
	CurrentTask  string    `json:"current_task"`
	StartTime    time.Time `json:"start_time"`
	LastActivity time.Time `json:"last_activity"`
	TokensIn     int64     `json:"tokens_in"`
	TokensOut    int64     `json:"tokens_out"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, Response{
		Success: false,
		Error:   message,
	})
}

func (s *Server) writeSuccess(w http.ResponseWriter, data interface{}) {
	s.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.writeSuccess(w, map[string]string{"status": "healthy"})
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	agents := s.manager.List()
	response := make([]AgentResponse, 0, len(agents))

	for _, a := range agents {
		metrics := a.Metrics()
		response = append(response, AgentResponse{
			ID:           a.ID(),
			Name:         a.Name(),
			Type:         a.Type(),
			Status:       a.Status().String(),
			Directory:    a.Directory(),
			ProjectID:    a.ProjectID(),
			CurrentTask:  a.CurrentTask(),
			StartTime:    a.StartTime(),
			LastActivity: a.LastActivity(),
			TokensIn:     metrics.TokensIn,
			TokensOut:    metrics.TokensOut,
		})
	}

	s.writeSuccess(w, response)
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path: /api/agents/{id}
	id := r.URL.Path[len("/api/agents/"):]
	if id == "" {
		s.writeError(w, http.StatusBadRequest, "agent ID required")
		return
	}

	// Handle terminate action
	if len(id) > 10 && id[len(id)-10:] == "/terminate" {
		id = id[:len(id)-10]
		if r.Method != http.MethodPost {
			s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if err := s.manager.Terminate(id); err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to terminate: %v", err))
			return
		}
		s.writeSuccess(w, map[string]string{"status": "terminated"})
		return
	}

	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	agent, found := s.manager.Get(id)
	if !found {
		s.writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	metrics := agent.Metrics()
	s.writeSuccess(w, AgentResponse{
		ID:           agent.ID(),
		Name:         agent.Name(),
		Type:         agent.Type(),
		Status:       agent.Status().String(),
		Directory:    agent.Directory(),
		ProjectID:    agent.ProjectID(),
		CurrentTask:  agent.CurrentTask(),
		StartTime:    agent.StartTime(),
		LastActivity: agent.LastActivity(),
		TokensIn:     metrics.TokensIn,
		TokensOut:    metrics.TokensOut,
	})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	stats := s.manager.Stats()

	// Convert status map to string keys for JSON
	byStatus := make(map[string]int)
	for status, count := range stats.ByStatus {
		byStatus[status.String()] = count
	}

	s.writeSuccess(w, map[string]interface{}{
		"total":           stats.Total,
		"by_status":       byStatus,
		"by_type":         stats.ByType,
		"by_project":      stats.ByProject,
		"total_tokens_in": stats.TotalTokensIn,
		"total_errors":    stats.TotalErrors,
	})
}
