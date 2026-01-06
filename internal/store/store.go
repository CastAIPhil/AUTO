// Package store handles persistent storage using SQLite
package store

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store handles persistent storage
type Store struct {
	db *sql.DB
}

// SessionRecord represents a stored session
type SessionRecord struct {
	ID            string    `json:"id"`
	AgentID       string    `json:"agent_id"`
	AgentType     string    `json:"agent_type"`
	AgentName     string    `json:"agent_name"`
	Directory     string    `json:"directory"`
	ProjectID     string    `json:"project_id"`
	Status        string    `json:"status"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	LastActivity  time.Time `json:"last_activity"`
	TokensIn      int64     `json:"tokens_in"`
	TokensOut     int64     `json:"tokens_out"`
	EstimatedCost float64   `json:"estimated_cost"`
	ToolCalls     int       `json:"tool_calls"`
	ErrorCount    int       `json:"error_count"`
	Output        string    `json:"output"`
	Metadata      string    `json:"metadata"`
}

// AlertRecord represents a stored alert
type AlertRecord struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
	Metadata  string    `json:"metadata"`
}

// MetricRecord represents a stored metric point
type MetricRecord struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agent_id"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// New creates a new store
func New(dbPath string) (*Store, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// migrate runs database migrations
func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL,
			agent_type TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			directory TEXT,
			project_id TEXT,
			status TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			last_activity DATETIME,
			tokens_in INTEGER DEFAULT 0,
			tokens_out INTEGER DEFAULT 0,
			estimated_cost REAL DEFAULT 0,
			tool_calls INTEGER DEFAULT 0,
			error_count INTEGER DEFAULT 0,
			output TEXT,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_agent_id ON sessions(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_start_time ON sessions(start_time)`,

		`CREATE TABLE IF NOT EXISTS alerts (
			id TEXT PRIMARY KEY,
			agent_id TEXT,
			level TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			read BOOLEAN DEFAULT FALSE,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_agent_id ON alerts(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_timestamp ON alerts(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_read ON alerts(read)`,

		`CREATE TABLE IF NOT EXISTS metrics (
			id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL,
			metric TEXT NOT NULL,
			value REAL NOT NULL,
			timestamp DATETIME NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_agent_id ON metrics(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_metric ON metrics(metric)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp)`,

		`CREATE TABLE IF NOT EXISTS output_chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			chunk TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (session_id) REFERENCES sessions(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_output_chunks_session_id ON output_chunks(session_id)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the database
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveSession saves or updates a session
func (s *Store) SaveSession(rec *SessionRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (
			id, agent_id, agent_type, agent_name, directory, project_id,
			status, start_time, end_time, last_activity,
			tokens_in, tokens_out, estimated_cost, tool_calls, error_count,
			output, metadata, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			end_time = excluded.end_time,
			last_activity = excluded.last_activity,
			tokens_in = excluded.tokens_in,
			tokens_out = excluded.tokens_out,
			estimated_cost = excluded.estimated_cost,
			tool_calls = excluded.tool_calls,
			error_count = excluded.error_count,
			output = excluded.output,
			metadata = excluded.metadata,
			updated_at = CURRENT_TIMESTAMP
	`, rec.ID, rec.AgentID, rec.AgentType, rec.AgentName, rec.Directory, rec.ProjectID,
		rec.Status, rec.StartTime, rec.EndTime, rec.LastActivity,
		rec.TokensIn, rec.TokensOut, rec.EstimatedCost, rec.ToolCalls, rec.ErrorCount,
		rec.Output, rec.Metadata)
	return err
}

// GetSession gets a session by ID
func (s *Store) GetSession(id string) (*SessionRecord, error) {
	rec := &SessionRecord{}
	var endTime, lastActivity sql.NullTime
	var output, metadata sql.NullString

	err := s.db.QueryRow(`
		SELECT id, agent_id, agent_type, agent_name, directory, project_id,
			status, start_time, end_time, last_activity,
			tokens_in, tokens_out, estimated_cost, tool_calls, error_count,
			output, metadata
		FROM sessions WHERE id = ?
	`, id).Scan(
		&rec.ID, &rec.AgentID, &rec.AgentType, &rec.AgentName, &rec.Directory, &rec.ProjectID,
		&rec.Status, &rec.StartTime, &endTime, &lastActivity,
		&rec.TokensIn, &rec.TokensOut, &rec.EstimatedCost, &rec.ToolCalls, &rec.ErrorCount,
		&output, &metadata,
	)
	if err != nil {
		return nil, err
	}

	if endTime.Valid {
		rec.EndTime = endTime.Time
	}
	if lastActivity.Valid {
		rec.LastActivity = lastActivity.Time
	}
	if output.Valid {
		rec.Output = output.String
	}
	if metadata.Valid {
		rec.Metadata = metadata.String
	}

	return rec, nil
}

// ListSessions lists sessions with optional filters
func (s *Store) ListSessions(limit int, status string) ([]*SessionRecord, error) {
	query := `
		SELECT id, agent_id, agent_type, agent_name, directory, project_id,
			status, start_time, end_time, last_activity,
			tokens_in, tokens_out, estimated_cost, tool_calls, error_count,
			output, metadata
		FROM sessions
	`
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}

	query += " ORDER BY start_time DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*SessionRecord
	for rows.Next() {
		rec := &SessionRecord{}
		var endTime, lastActivity sql.NullTime
		var output, metadata sql.NullString

		if err := rows.Scan(
			&rec.ID, &rec.AgentID, &rec.AgentType, &rec.AgentName, &rec.Directory, &rec.ProjectID,
			&rec.Status, &rec.StartTime, &endTime, &lastActivity,
			&rec.TokensIn, &rec.TokensOut, &rec.EstimatedCost, &rec.ToolCalls, &rec.ErrorCount,
			&output, &metadata,
		); err != nil {
			return nil, err
		}

		if endTime.Valid {
			rec.EndTime = endTime.Time
		}
		if lastActivity.Valid {
			rec.LastActivity = lastActivity.Time
		}
		if output.Valid {
			rec.Output = output.String
		}
		if metadata.Valid {
			rec.Metadata = metadata.String
		}

		records = append(records, rec)
	}

	return records, nil
}

// SaveAlert saves an alert
func (s *Store) SaveAlert(rec *AlertRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO alerts (id, agent_id, level, message, timestamp, read, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, rec.ID, rec.AgentID, rec.Level, rec.Message, rec.Timestamp, rec.Read, rec.Metadata)
	return err
}

// ListAlerts lists alerts
func (s *Store) ListAlerts(limit int, unreadOnly bool) ([]*AlertRecord, error) {
	query := `SELECT id, agent_id, level, message, timestamp, read, metadata FROM alerts`
	args := []interface{}{}

	if unreadOnly {
		query += " WHERE read = FALSE"
	}

	query += " ORDER BY timestamp DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*AlertRecord
	for rows.Next() {
		rec := &AlertRecord{}
		var agentID, metadata sql.NullString

		if err := rows.Scan(&rec.ID, &agentID, &rec.Level, &rec.Message, &rec.Timestamp, &rec.Read, &metadata); err != nil {
			return nil, err
		}

		if agentID.Valid {
			rec.AgentID = agentID.String
		}
		if metadata.Valid {
			rec.Metadata = metadata.String
		}

		records = append(records, rec)
	}

	return records, nil
}

// MarkAlertRead marks an alert as read
func (s *Store) MarkAlertRead(id string) error {
	_, err := s.db.Exec(`UPDATE alerts SET read = TRUE WHERE id = ?`, id)
	return err
}

// MarkAllAlertsRead marks all alerts as read
func (s *Store) MarkAllAlertsRead() error {
	_, err := s.db.Exec(`UPDATE alerts SET read = TRUE`)
	return err
}

// SaveMetric saves a metric point
func (s *Store) SaveMetric(rec *MetricRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO metrics (id, agent_id, metric, value, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`, rec.ID, rec.AgentID, rec.Metric, rec.Value, rec.Timestamp)
	return err
}

// GetMetrics gets metrics for an agent
func (s *Store) GetMetrics(agentID string, metric string, since time.Time) ([]*MetricRecord, error) {
	rows, err := s.db.Query(`
		SELECT id, agent_id, metric, value, timestamp
		FROM metrics
		WHERE agent_id = ? AND metric = ? AND timestamp >= ?
		ORDER BY timestamp ASC
	`, agentID, metric, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*MetricRecord
	for rows.Next() {
		rec := &MetricRecord{}
		if err := rows.Scan(&rec.ID, &rec.AgentID, &rec.Metric, &rec.Value, &rec.Timestamp); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}

	return records, nil
}

// AppendOutput appends output to a session
func (s *Store) AppendOutput(sessionID string, chunk string) error {
	_, err := s.db.Exec(`
		INSERT INTO output_chunks (session_id, chunk, timestamp)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`, sessionID, chunk)
	return err
}

// GetOutput gets all output for a session
func (s *Store) GetOutput(sessionID string) (string, error) {
	rows, err := s.db.Query(`
		SELECT chunk FROM output_chunks
		WHERE session_id = ?
		ORDER BY id ASC
	`, sessionID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var output string
	for rows.Next() {
		var chunk string
		if err := rows.Scan(&chunk); err != nil {
			return "", err
		}
		output += chunk
	}

	return output, nil
}

// GetStats gets aggregate statistics
func (s *Store) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total sessions
	var totalSessions int
	s.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&totalSessions)
	stats["total_sessions"] = totalSessions

	// Sessions by status
	rows, _ := s.db.Query(`SELECT status, COUNT(*) FROM sessions GROUP BY status`)
	statusCounts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		rows.Scan(&status, &count)
		statusCounts[status] = count
	}
	rows.Close()
	stats["sessions_by_status"] = statusCounts

	// Total tokens
	var tokensIn, tokensOut int64
	s.db.QueryRow(`SELECT COALESCE(SUM(tokens_in), 0), COALESCE(SUM(tokens_out), 0) FROM sessions`).Scan(&tokensIn, &tokensOut)
	stats["total_tokens_in"] = tokensIn
	stats["total_tokens_out"] = tokensOut

	// Total cost
	var totalCost float64
	s.db.QueryRow(`SELECT COALESCE(SUM(estimated_cost), 0) FROM sessions`).Scan(&totalCost)
	stats["total_cost"] = totalCost

	// Total errors
	var totalErrors int
	s.db.QueryRow(`SELECT COALESCE(SUM(error_count), 0) FROM sessions`).Scan(&totalErrors)
	stats["total_errors"] = totalErrors

	// Unread alerts
	var unreadAlerts int
	s.db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE read = FALSE`).Scan(&unreadAlerts)
	stats["unread_alerts"] = unreadAlerts

	return stats, nil
}

// Cleanup removes old data
func (s *Store) Cleanup(maxAgeDays int) error {
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)

	// Delete old output chunks
	_, err := s.db.Exec(`
		DELETE FROM output_chunks WHERE session_id IN (
			SELECT id FROM sessions WHERE end_time < ? AND end_time IS NOT NULL
		)
	`, cutoff)
	if err != nil {
		return err
	}

	// Delete old sessions
	_, err = s.db.Exec(`DELETE FROM sessions WHERE end_time < ? AND end_time IS NOT NULL`, cutoff)
	if err != nil {
		return err
	}

	// Delete old alerts
	_, err = s.db.Exec(`DELETE FROM alerts WHERE timestamp < ?`, cutoff)
	if err != nil {
		return err
	}

	// Delete old metrics
	_, err = s.db.Exec(`DELETE FROM metrics WHERE timestamp < ?`, cutoff)
	return err
}

// ExportJSON exports all data as JSON
func (s *Store) ExportJSON() ([]byte, error) {
	sessions, _ := s.ListSessions(0, "")
	alerts, _ := s.ListAlerts(0, false)
	stats, _ := s.GetStats()

	data := map[string]interface{}{
		"sessions":   sessions,
		"alerts":     alerts,
		"statistics": stats,
		"exported":   time.Now(),
	}

	return json.MarshalIndent(data, "", "  ")
}
