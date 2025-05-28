package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/giovannigabriele/go-todo-bot/internal/config"
	"github.com/giovannigabriele/go-todo-bot/internal/llm"
	"github.com/giovannigabriele/go-todo-bot/internal/sheets"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
)

// Bot represents the Telegram bot
type Bot struct {
	api          *tgbotapi.BotAPI
	config       *config.Config
	llmClient    *llm.Client
	sheetsClient *sheets.Client
}

// NewBot creates a new Telegram bot instance
func NewBot(cfg *config.Config) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, err
	}

	// Set debug mode in development
	bot.Debug = cfg.Environment == "development"

	log.Info().Str("username", bot.Self.UserName).Msg("Telegram bot authorized")

	return &Bot{
		api:          bot,
		config:       cfg,
		llmClient:    llm.NewClient(cfg),
		sheetsClient: sheets.NewClient(cfg),
	}, nil
}

// Start begins listening for messages
func (b *Bot) Start(ctx context.Context) error {
	log.Info().Msg("Starting Telegram bot...")

	// Create update config
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Get updates channel
	updates := b.api.GetUpdatesChan(u)

	// Process updates
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Stopping Telegram bot...")
			return nil
		case update := <-updates:
			if update.Message != nil {
				go b.handleMessage(ctx, update.Message)
			}
		}
	}
}

// handleMessage processes incoming messages
func (b *Bot) handleMessage(ctx context.Context, message *tgbotapi.Message) {
	log.Debug().
		Str("from", message.From.UserName).
		Str("text", message.Text).
		Msg("Received message")

	// Handle commands
	if message.IsCommand() {
		b.handleCommand(ctx, message)
		return
	}

	// Skip empty messages
	if message.Text == "" {
		return
	}

	// Send initial acknowledgment
	ackMsg := tgbotapi.NewMessage(message.Chat.ID, "🔖 Processing your task...")
	if _, err := b.api.Send(ackMsg); err != nil {
		log.Error().Err(err).Msg("Failed to send acknowledgment")
	}

	// Parse message with LLM
	parseResponse, err := b.llmClient.ParseMessage(ctx, message.Text)
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse message with LLM")
		b.sendErrorMessage(message.Chat.ID, "Sorry, I couldn't process your message. Please try again.")
		return
	}

	// Convert to sheet tasks
	var sheetTasks []sheets.Task
	for _, task := range parseResponse.Tasks {
		sheetTask := sheets.Task{
			People:      task.People,
			Client:      task.Client,
			Summary:     task.Summary,
			FullMessage: message.Text,
			DueDate:     task.DueDate,
			BotNotes:    fmt.Sprintf("Confidence: %.2f, Parsed by: %s", task.Confidence, b.llmClient.GetModel()),
		}
		sheetTasks = append(sheetTasks, sheetTask)
	}

	// Add tasks to Google Sheets
	if len(sheetTasks) > 0 {
		log.Debug().
			Interface("sheet_tasks", sheetTasks).
			Msg("About to call AddTasks with these tasks")

		if err := b.sheetsClient.AddTasks(ctx, sheetTasks); err != nil {
			log.Error().Err(err).Msg("Failed to save tasks to Google Sheets")
			b.sendErrorMessage(message.Chat.ID, "I processed your message but couldn't save it to the sheet. Please try again.")
			return
		}

		log.Info().
			Int("tasks_saved", len(sheetTasks)).
			Msg("Successfully saved tasks to Google Sheets")
	}

	// Send success response
	responseText := b.buildSuccessResponse(parseResponse)
	successMsg := tgbotapi.NewMessage(message.Chat.ID, responseText)
	if _, err := b.api.Send(successMsg); err != nil {
		log.Error().Err(err).Msg("Failed to send success message")
	}
}

// handleCommand processes bot commands
func (b *Bot) handleCommand(ctx context.Context, message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		welcomeText := `👋 Welcome to the TODO Bot!

I help you manage tasks by parsing your natural language messages and organizing them in Google Sheets.

Just send me a message like:
• "Lilly to reach out to Johnny about the project"
• "Team needs to review the quarterly budget"
• "Sarah to finish the report AND Marcus to schedule the meeting"

I'll automatically:
✅ Extract who's responsible
✅ Identify the tasks
✅ Save everything to your shared Google Sheet

Try sending me a task now!`

		msg := tgbotapi.NewMessage(message.Chat.ID, welcomeText)
		if _, err := b.api.Send(msg); err != nil {
			log.Error().Err(err).Msg("Failed to send welcome message")
		}

	case "help":
		helpText := `🤖 TODO Bot Help

**How to use:**
Send me any message describing tasks and I'll parse them automatically.

**Examples:**
• "Alice to call the client tomorrow"
• "Bob and Sarah to review the proposal"
• "Team meeting at 3pm to discuss budget"
• "Fix the bug AND deploy to staging"

**Features:**
• Automatic person detection
• Task splitting (using "AND")
• Team task assignment
• Google Sheets integration

**Commands:**
/start - Welcome message
/help - This help text
/status - Check bot status`

		msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
		if _, err := b.api.Send(msg); err != nil {
			log.Error().Err(err).Msg("Failed to send help message")
		}

	case "status":
		statusText := "✅ Bot is running and ready to process tasks!"
		msg := tgbotapi.NewMessage(message.Chat.ID, statusText)
		if _, err := b.api.Send(msg); err != nil {
			log.Error().Err(err).Msg("Failed to send status message")
		}

	default:
		unknownText := "❓ Unknown command. Use /help to see available commands."
		msg := tgbotapi.NewMessage(message.Chat.ID, unknownText)
		if _, err := b.api.Send(msg); err != nil {
			log.Error().Err(err).Msg("Failed to send unknown command message")
		}
	}
}

// buildSuccessResponse creates a success message for the user
func (b *Bot) buildSuccessResponse(parseResponse *llm.ParseResponse) string {
	if len(parseResponse.Tasks) == 0 {
		return "✅ Message received, but no tasks were identified."
	}

	response := fmt.Sprintf("✅ Added %d task(s) to the sheet:\n\n", len(parseResponse.Tasks))

	for i, task := range parseResponse.Tasks {
		peopleStr := "👥 " + formatPeople(task.People)
		clientStr := "🏢 " + formatClient(task.Client)
		dueDateStr := "📅 " + formatDueDate(task.DueDate)

		response += fmt.Sprintf("%d. %s\n   %s\n   %s\n   📝 %s\n\n",
			i+1,
			peopleStr,
			clientStr,
			dueDateStr,
			task.Summary)
	}

	response += "📊 Check the Google Sheet for full details!"
	return response
}

// formatPeople formats the people list for display
func formatPeople(people []string) string {
	if len(people) == 0 {
		return "Team"
	}

	if len(people) == 1 {
		if people[0] == "team" {
			return "Team"
		}
		return people[0]
	}

	// Multiple people
	result := ""
	for i, person := range people {
		if person == "team" {
			person = "Team"
		}
		if i == 0 {
			result = person
		} else if i == len(people)-1 {
			result += " & " + person
		} else {
			result += ", " + person
		}
	}
	return result
}

// formatClient formats the client name for display
func formatClient(client string) string {
	if client == "" || client == "unclear" {
		return "Client: unclear"
	}
	return "Client: " + strings.Title(client)
}

// formatDueDate formats the due date for display
func formatDueDate(dueDate string) string {
	if dueDate == "" || dueDate == "unclear" {
		return "Due: unclear"
	}
	return "Due: " + dueDate
}

// sendErrorMessage sends an error message to the user
func (b *Bot) sendErrorMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, "❌ "+text)
	if _, err := b.api.Send(msg); err != nil {
		log.Error().Err(err).Msg("Failed to send error message")
	}
}
