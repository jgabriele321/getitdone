package queue

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// EnqueueTask adds a new task to the queue
func (m *Manager) EnqueueTask(ctx context.Context, messageText string, formatType FormatType) (*QueuedTask, error) {
	batchID := uuid.New().String()

	result, err := m.db.ExecContext(ctx, `
		INSERT INTO tasks_queue (batch_id, message_text, format_type, status)
		VALUES (?, ?, ?, ?)
	`, batchID, messageText, formatType, StatusPending)
	if err != nil {
		return nil, fmt.Errorf("failed to enqueue task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get task ID: %w", err)
	}

	return &QueuedTask{
		ID:          id,
		BatchID:     batchID,
		MessageText: messageText,
		FormatType:  formatType,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
	}, nil
}

// EnqueueBatchTasks adds multiple tasks to the queue with the same batch ID
func (m *Manager) EnqueueBatchTasks(ctx context.Context, tasks []struct {
	MessageText string
	FormatType  FormatType
}) ([]QueuedTask, error) {
	if len(tasks) == 0 {
		return nil, nil
	}

	batchID := uuid.New().String()
	var queuedTasks []QueuedTask

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tasks_queue (batch_id, message_text, format_type, status)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, task := range tasks {
		result, err := stmt.ExecContext(ctx, batchID, task.MessageText, task.FormatType, StatusPending)
		if err != nil {
			return nil, fmt.Errorf("failed to enqueue task: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("failed to get task ID: %w", err)
		}

		queuedTasks = append(queuedTasks, QueuedTask{
			ID:          id,
			BatchID:     batchID,
			MessageText: task.MessageText,
			FormatType:  task.FormatType,
			Status:      StatusPending,
			CreatedAt:   now,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return queuedTasks, nil
}

// GetNextPendingTask retrieves the next pending task
func (m *Manager) GetNextPendingTask(ctx context.Context) (*QueuedTask, error) {
	var task QueuedTask
	var processedAt sql.NullTime
	var errorMsg sql.NullString

	err := m.db.QueryRowContext(ctx, `
		SELECT id, batch_id, message_text, format_type, status, created_at, processed_at, error
		FROM tasks_queue
		WHERE status = ?
		ORDER BY created_at ASC
		LIMIT 1
	`, StatusPending).Scan(
		&task.ID,
		&task.BatchID,
		&task.MessageText,
		&task.FormatType,
		&task.Status,
		&task.CreatedAt,
		&processedAt,
		&errorMsg,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next pending task: %w", err)
	}

	if processedAt.Valid {
		task.ProcessedAt = &processedAt.Time
	}
	if errorMsg.Valid {
		task.Error = &errorMsg.String
	}

	return &task, nil
}

// UpdateTaskStatus updates the status of a task
func (m *Manager) UpdateTaskStatus(ctx context.Context, taskID int64, status TaskStatus, errorMsg *string) error {
	var err error
	if errorMsg != nil {
		_, err = m.db.ExecContext(ctx, `
			UPDATE tasks_queue
			SET status = ?, processed_at = CURRENT_TIMESTAMP, error = ?
			WHERE id = ?
		`, status, errorMsg, taskID)
	} else {
		_, err = m.db.ExecContext(ctx, `
			UPDATE tasks_queue
			SET status = ?, processed_at = CURRENT_TIMESTAMP, error = NULL
			WHERE id = ?
		`, status, taskID)
	}

	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// GetBatchTasks retrieves all tasks in a batch
func (m *Manager) GetBatchTasks(ctx context.Context, batchID string) ([]QueuedTask, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, batch_id, message_text, format_type, status, created_at, processed_at, error
		FROM tasks_queue
		WHERE batch_id = ?
		ORDER BY created_at ASC
	`, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch tasks: %w", err)
	}
	defer rows.Close()

	var tasks []QueuedTask
	for rows.Next() {
		var task QueuedTask
		var processedAt sql.NullTime
		var errorMsg sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.BatchID,
			&task.MessageText,
			&task.FormatType,
			&task.Status,
			&task.CreatedAt,
			&processedAt,
			&errorMsg,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task row: %w", err)
		}

		if processedAt.Valid {
			task.ProcessedAt = &processedAt.Time
		}
		if errorMsg.Valid {
			task.Error = &errorMsg.String
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating task rows: %w", err)
	}

	return tasks, nil
}

// CleanupCompletedTasks removes completed tasks older than the specified duration
func (m *Manager) CleanupCompletedTasks(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	_, err := m.db.ExecContext(ctx, `
		DELETE FROM tasks_queue
		WHERE status IN (?, ?)
		AND processed_at < ?
	`, StatusComplete, StatusFailed, cutoff)

	if err != nil {
		return fmt.Errorf("failed to cleanup completed tasks: %w", err)
	}

	return nil
}

// GetTaskStats returns statistics about the queue
func (m *Manager) GetTaskStats(ctx context.Context) (map[TaskStatus]int, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT status, COUNT(*) as count
		FROM tasks_queue
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query task stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[TaskStatus]int)
	for rows.Next() {
		var status TaskStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan stats row: %w", err)
		}
		stats[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating stats rows: %w", err)
	}

	return stats, nil
}
