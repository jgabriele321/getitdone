package queue

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

// Schema version for database migrations
const schemaVersion = 1

// TaskStatus represents the status of a queued task
type TaskStatus string

const (
	StatusPending  TaskStatus = "pending"
	StatusRunning  TaskStatus = "running"
	StatusComplete TaskStatus = "complete"
	StatusFailed   TaskStatus = "failed"
)

// FormatType represents the type of message format
type FormatType string

const (
	FormatSingleTask     FormatType = "SINGLE_TASK"
	FormatNarrativeMulti FormatType = "NARRATIVE_MULTI"
	FormatBulletList     FormatType = "BULLET_LIST"
	FormatMixed          FormatType = "MIXED"
)

// QueuedTask represents a task in the processing queue
type QueuedTask struct {
	ID          int64
	BatchID     string
	MessageText string
	FormatType  FormatType
	Status      TaskStatus
	CreatedAt   time.Time
	ProcessedAt *time.Time
	Error       *string
}

// Manager handles queue operations
type Manager struct {
	db *sql.DB
}

// NewManager creates a new queue manager
func NewManager(dbPath string) (*Manager, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	m := &Manager{db: db}
	if err := m.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return m, nil
}

// initSchema creates the necessary database tables
func (m *Manager) initSchema() error {
	log.Info().Msg("Initializing queue database schema...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create schema_version table
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER PRIMARY KEY
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_version table: %w", err)
	}

	// Create tasks_queue table
	_, err = m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tasks_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_id TEXT NOT NULL,
			message_text TEXT NOT NULL,
			format_type TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			processed_at TIMESTAMP,
			error TEXT,
			UNIQUE(batch_id, message_text)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create tasks_queue table: %w", err)
	}

	// Create indexes
	_, err = m.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_tasks_queue_status ON tasks_queue(status);
		CREATE INDEX IF NOT EXISTS idx_tasks_queue_batch_id ON tasks_queue(batch_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Check/update schema version
	var version int
	err = m.db.QueryRowContext(ctx, "SELECT version FROM schema_version").Scan(&version)
	if err == sql.ErrNoRows {
		_, err = m.db.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", schemaVersion)
		if err != nil {
			return fmt.Errorf("failed to set initial schema version: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check schema version: %w", err)
	}

	log.Info().Int("version", schemaVersion).Msg("Queue database schema initialized")
	return nil
}

// Close closes the database connection
func (m *Manager) Close() error {
	return m.db.Close()
}
