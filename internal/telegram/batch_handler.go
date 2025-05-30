package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"

	"github.com/giovannigabriele/go-todo-bot/internal/llm"
	"github.com/giovannigabriele/go-todo-bot/internal/queue"
	"github.com/giovannigabriele/go-todo-bot/internal/sheets"
)

// BatchHandler extends the base handler with batch processing capabilities
type BatchHandler struct {
	*Handler
	queueManager *queue.Manager
	worker       *queue.Worker
}

// NewBatchHandler creates a new batch-capable handler
func NewBatchHandler(token string, llmClient *llm.Client, sheetsClient *sheets.Client, queueManager *queue.Manager) (*BatchHandler, error) {
	baseHandler, err := NewHandler(token, llmClient, sheetsClient)
	if err != nil {
		return nil, err
	}

	handler := &BatchHandler{
		Handler:      baseHandler,
		queueManager: queueManager,
	}

	// Create worker with task processor
	handler.worker = queue.NewWorker(queueManager, handler.processQueuedTask, 2, 500*time.Millisecond)

	return handler, nil
}

// Start starts the bot and worker
func (h *BatchHandler) Start(ctx context.Context) error {
	// Start the worker
	h.worker.Start(ctx)

	// Start the bot (this blocks until context is cancelled)
	err := h.Handler.Start(ctx)

	// Stop the worker when the bot stops
	h.worker.Stop()

	return err
}

// processTaskMessage handles incoming messages with batch processing
func (h *BatchHandler) processTaskMessage(ctx context.Context, message *tgbotapi.Message) {
	// Send immediate acknowledgment
	h.sendMessage(message.Chat.ID, "🔖 Processing your message...")

	// Check if this is likely a batch message
	format := queue.DetectMessageFormat(message.Text)
	if format == queue.FormatSingleTask {
		// Fast path for single tasks
		h.Handler.processTaskMessage(ctx, message)
		return
	}

	// Split message into individual tasks
	tasks := queue.SplitMessage(message.Text, format)
	if len(tasks) == 0 {
		h.sendMessage(message.Chat.ID, "❌ No tasks could be identified in your message.")
		return
	}

	// Create batch tasks
	var queueTasks []struct {
		MessageText string
		FormatType  queue.FormatType
	}
	for _, task := range tasks {
		queueTasks = append(queueTasks, struct {
			MessageText string
			FormatType  queue.FormatType
		}{
			MessageText: task,
			FormatType:  queue.FormatSingleTask, // Each split task is treated as single
		})
	}

	// Enqueue batch
	queuedTasks, err := h.queueManager.EnqueueBatchTasks(ctx, queueTasks)
	if err != nil {
		log.Error().Err(err).Msg("Failed to enqueue batch tasks")
		h.sendMessage(message.Chat.ID, "❌ Failed to process your tasks. Please try again.")
		return
	}

	// Send initial batch status
	batchID := queuedTasks[0].BatchID // All tasks in batch have same ID
	h.sendBatchStatus(message.Chat.ID, batchID, len(queuedTasks))

	// Start monitoring batch progress
	go h.monitorBatchProgress(ctx, message.Chat.ID, batchID)
}

// processQueuedTask processes a single queued task
func (h *BatchHandler) processQueuedTask(ctx context.Context, task *queue.QueuedTask) error {
	// Parse with LLM
	parseResp, err := h.llmClient.ParseMessage(ctx, task.MessageText)
	if err != nil {
		return fmt.Errorf("failed to parse message with LLM: %w", err)
	}

	// Convert to sheet rows
	var taskRows []sheets.TaskRow
	for i, parsedTask := range parseResp.Tasks {
		summary := parsedTask.Summary
		if len(parseResp.Tasks) > 1 {
			summary = fmt.Sprintf("%s (%d/%d)", parsedTask.Summary, i+1, len(parseResp.Tasks))
		}

		botNotes := fmt.Sprintf("Batch ID: %s, Confidence: %.2f", task.BatchID, parsedTask.Confidence)
		if parsedTask.Confidence < 0.7 {
			botNotes += " (Low confidence)"
		}

		taskRow := sheets.CreateTaskRow(
			parsedTask.People,
			parsedTask.Client,
			summary,
			task.MessageText,
			parsedTask.DueDate,
			botNotes,
		)
		taskRows = append(taskRows, taskRow)
	}

	// Save to Google Sheets
	if err := h.sheetsClient.AddTasks(ctx, taskRows); err != nil {
		return fmt.Errorf("failed to save tasks to Google Sheets: %w", err)
	}

	return nil
}

// sendBatchStatus sends the initial batch status message
func (h *BatchHandler) sendBatchStatus(chatID int64, batchID string, totalTasks int) {
	message := fmt.Sprintf("🔄 Processing batch of %d tasks...\n\nBatch ID: %s", totalTasks, batchID)
	h.sendMessage(chatID, message)
}

// monitorBatchProgress monitors and reports batch progress
func (h *BatchHandler) monitorBatchProgress(ctx context.Context, chatID int64, batchID string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastProgress := make(map[queue.TaskStatus]int)
	unchanged := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get current progress
			progress, err := h.worker.GetBatchProgress(ctx, batchID)
			if err != nil {
				log.Error().Err(err).Str("batch_id", batchID).Msg("Failed to get batch progress")
				return
			}

			// Check if progress has changed
			changed := false
			for status, count := range progress {
				if lastProgress[status] != count {
					changed = true
					break
				}
			}

			if changed {
				// Send progress update
				h.sendProgressUpdate(chatID, progress)
				lastProgress = progress
				unchanged = 0
			} else {
				unchanged++
				if unchanged >= 5 { // No changes for 10 seconds
					// Check if batch is complete
					complete, err := h.worker.IsBatchComplete(ctx, batchID)
					if err != nil {
						log.Error().Err(err).Str("batch_id", batchID).Msg("Failed to check batch completion")
						return
					}
					if complete {
						h.sendBatchComplete(chatID, progress)
						return
					}
				}
			}
		}
	}
}

// sendProgressUpdate sends a progress update message
func (h *BatchHandler) sendProgressUpdate(chatID int64, progress map[queue.TaskStatus]int) {
	total := 0
	for _, count := range progress {
		total += count
	}

	var status strings.Builder
	status.WriteString(fmt.Sprintf("🔄 Processing tasks (%d total):\n\n", total))

	if count := progress[queue.StatusComplete]; count > 0 {
		status.WriteString(fmt.Sprintf("✅ Completed: %d\n", count))
	}
	if count := progress[queue.StatusRunning]; count > 0 {
		status.WriteString(fmt.Sprintf("⚙️ Processing: %d\n", count))
	}
	if count := progress[queue.StatusPending]; count > 0 {
		status.WriteString(fmt.Sprintf("⏳ Pending: %d\n", count))
	}
	if count := progress[queue.StatusFailed]; count > 0 {
		status.WriteString(fmt.Sprintf("❌ Failed: %d\n", count))
	}

	h.sendMessage(chatID, status.String())
}

// sendBatchComplete sends the batch completion message
func (h *BatchHandler) sendBatchComplete(chatID int64, progress map[queue.TaskStatus]int) {
	completed := progress[queue.StatusComplete]
	failed := progress[queue.StatusFailed]
	total := completed + failed

	var message strings.Builder
	message.WriteString(fmt.Sprintf("✅ Batch processing complete!\n\n"))
	message.WriteString(fmt.Sprintf("Total tasks: %d\n", total))
	message.WriteString(fmt.Sprintf("✅ Successfully processed: %d\n", completed))

	if failed > 0 {
		message.WriteString(fmt.Sprintf("❌ Failed: %d\n", failed))
		message.WriteString("\nPlease check the sheet for details on failed tasks.")
	}

	h.sendMessage(chatID, message.String())
}

// getHelpMessage returns the help message with batch processing information
func (h *BatchHandler) getHelpMessage() string {
	return `🤖 TODO Bot Commands:

/start - Show welcome message
/help - Show this help
/status - Check bot status

📝 How to use:
Just send me any message describing tasks and I'll automatically parse and save them.

Single Task Examples:
• "Lilly to reach out to Johnny for a quote"
• "Team needs to review the quarterly budget"

Batch Task Examples:
• Multiple tasks with AND:
  "Call the vendor AND review the contract"

• Narrative style:
  "Gemma to ask oxccu for press release bullet points. Lilly will draft the press release, due friday."

• Bullet points:
  - Gemma to find out how much a clown costs
  - Client to get back to us by tuesday on Budget
  - Birthday girl to be asked her favorite ice cream flavor
  - Everyone to brainstorm themes and come up with 3
  - Lilly to write proposal by end of week

The bot will:
✅ Identify single vs batch tasks automatically
✅ Split batch tasks into individual items
✅ Process each task with proper assignment
✅ Show progress for batch processing
✅ Save everything to your shared Google Sheet`
}

// getStartMessage returns the start message with batch processing information
func (h *BatchHandler) getStartMessage() string {
	return `👋 Welcome to the TODO Bot!

I help you capture and organize tasks in Google Sheets. I can handle both single tasks and batches of multiple tasks!

Single task example:
"Johnny to ask Lexi what she wants for dinner"

Multiple tasks example:
"Review the proposal AND send feedback to client"

Or use bullet points:
- First task here
- Second task there
- Another task for someone

I'll automatically:
✅ Detect if it's a single task or batch
✅ Split multiple tasks when needed
✅ Extract who's responsible
✅ Create task summaries
✅ Save everything to your Google Sheet
✅ Show progress for batch processing

Use /help for more examples and commands.`
}

// getStatusMessage returns the status message with queue information
func (h *BatchHandler) getStatusMessage(ctx context.Context) string {
	// Try to get team data to test connectivity
	_, err := h.sheetsClient.GetTeam(ctx)
	if err != nil {
		return fmt.Sprintf("⚠️ Bot is running but Google Sheets connection failed: %s", err.Error())
	}

	// Get queue stats
	stats, err := h.queueManager.GetTaskStats(ctx)
	if err != nil {
		return "✅ Bot is running and connected to Google Sheets!"
	}

	var status strings.Builder
	status.WriteString("✅ Bot is running and connected to Google Sheets!\n\n")
	status.WriteString("📊 Current Queue Status:\n")

	if count := stats[queue.StatusPending]; count > 0 {
		status.WriteString(fmt.Sprintf("⏳ Pending: %d tasks\n", count))
	}
	if count := stats[queue.StatusRunning]; count > 0 {
		status.WriteString(fmt.Sprintf("⚙️ Processing: %d tasks\n", count))
	}
	if count := stats[queue.StatusComplete]; count > 0 {
		status.WriteString(fmt.Sprintf("✅ Completed: %d tasks\n", count))
	}
	if count := stats[queue.StatusFailed]; count > 0 {
		status.WriteString(fmt.Sprintf("❌ Failed: %d tasks\n", count))
	}

	return status.String()
}
