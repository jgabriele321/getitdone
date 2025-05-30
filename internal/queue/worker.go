package queue

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// TaskProcessor is a function that processes a single task
type TaskProcessor func(ctx context.Context, task *QueuedTask) error

// Worker handles task processing
type Worker struct {
	manager    *Manager
	processor  TaskProcessor
	numWorkers int
	interval   time.Duration
	wg         sync.WaitGroup
	stopCh     chan struct{}
}

// NewWorker creates a new task worker
func NewWorker(manager *Manager, processor TaskProcessor, numWorkers int, interval time.Duration) *Worker {
	return &Worker{
		manager:    manager,
		processor:  processor,
		numWorkers: numWorkers,
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
}

// Start begins task processing
func (w *Worker) Start(ctx context.Context) {
	log.Info().Int("workers", w.numWorkers).Msg("Starting task workers...")

	for i := 0; i < w.numWorkers; i++ {
		w.wg.Add(1)
		go w.processLoop(ctx, i)
	}
}

// Stop gracefully stops task processing
func (w *Worker) Stop() {
	log.Info().Msg("Stopping task workers...")
	close(w.stopCh)
	w.wg.Wait()
	log.Info().Msg("Task workers stopped")
}

// processLoop runs the main processing loop for a worker
func (w *Worker) processLoop(ctx context.Context, workerID int) {
	defer w.wg.Done()

	log.Info().Int("worker_id", workerID).Msg("Task worker started")

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Int("worker_id", workerID).Msg("Task worker stopping due to context cancellation")
			return
		case <-w.stopCh:
			log.Info().Int("worker_id", workerID).Msg("Task worker stopping due to stop signal")
			return
		case <-ticker.C:
			if err := w.processPendingTask(ctx); err != nil {
				log.Error().Err(err).Int("worker_id", workerID).Msg("Error processing pending task")
			}
		}
	}
}

// processPendingTask processes a single pending task
func (w *Worker) processPendingTask(ctx context.Context) error {
	// Get next pending task
	task, err := w.manager.GetNextPendingTask(ctx)
	if err != nil {
		return fmt.Errorf("failed to get next pending task: %w", err)
	}
	if task == nil {
		return nil // No pending tasks
	}

	// Update status to running
	if err := w.manager.UpdateTaskStatus(ctx, task.ID, StatusRunning, nil); err != nil {
		return fmt.Errorf("failed to update task status to running: %w", err)
	}

	// Process the task
	err = w.processor(ctx, task)
	if err != nil {
		errMsg := err.Error()
		if err := w.manager.UpdateTaskStatus(ctx, task.ID, StatusFailed, &errMsg); err != nil {
			log.Error().Err(err).Int64("task_id", task.ID).Msg("Failed to update task status to failed")
		}
		return fmt.Errorf("failed to process task: %w", err)
	}

	// Update status to complete
	if err := w.manager.UpdateTaskStatus(ctx, task.ID, StatusComplete, nil); err != nil {
		return fmt.Errorf("failed to update task status to complete: %w", err)
	}

	log.Info().
		Int64("task_id", task.ID).
		Str("batch_id", task.BatchID).
		Str("format_type", string(task.FormatType)).
		Msg("Task processed successfully")

	return nil
}

// GetBatchProgress returns the progress of a batch of tasks
func (w *Worker) GetBatchProgress(ctx context.Context, batchID string) (map[TaskStatus]int, error) {
	tasks, err := w.manager.GetBatchTasks(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch tasks: %w", err)
	}

	progress := make(map[TaskStatus]int)
	for _, task := range tasks {
		progress[task.Status]++
	}

	return progress, nil
}

// IsBatchComplete checks if all tasks in a batch are complete
func (w *Worker) IsBatchComplete(ctx context.Context, batchID string) (bool, error) {
	progress, err := w.GetBatchProgress(ctx, batchID)
	if err != nil {
		return false, err
	}

	total := 0
	for status, count := range progress {
		if status != StatusComplete && status != StatusFailed {
			return false, nil
		}
		total += count
	}

	return total > 0, nil
}
