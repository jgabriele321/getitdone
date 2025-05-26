# Go Telegram TODO Bot

This bot helps you manage tasks by parsing your natural language messages from Telegram, extracting task details using an LLM, and logging them to a Google Sheet.

## Features

- **Natural Language Processing**: Send tasks in plain English.
- **LLM Integration**: Uses an LLM (via OpenRouter) to parse messages, identify people, and summarize tasks.
- **Google Sheets Logging**: Automatically saves tasks to a designated Google Sheet.
- **Telegram Commands**:
    - `/start`: Welcome message.
    - `/help`: Shows help text with usage examples.
    - `/status`: Checks if the bot is running.
- **Configurable**: Uses a `.env` file for easy configuration.
- **Error Handling**: Provides feedback to the user if issues occur.
- **Graceful Shutdown**: Handles `Ctrl+C` to shut down cleanly.
- **Background Cron (Placeholder)**: Includes a cron manager for future scheduled tasks (like daily digests).
- **Health Check (Placeholder)**: Includes a basic HTTP health check endpoint.

## Project Structure

```
.
├── .env.example        # Example environment file
├── .gitignore          # Files to ignore in Git
├── README.md           # This file
├── bot                 # Compiled bot executable
├── cmd/
│   └── bot/
│       └── main.go     # Main application entry point
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── internal/
│   ├── config/
│   │   └── config.go   # Configuration loading
│   ├── cron/
│   │   └── manager.go  # Cron job management (placeholder)
│   ├── llm/
│   │   └── client.go   # LLM API client
│   ├── sheets/
│   │   └── client.go   # Google Sheets API client
│   └── telegram/
│       ├── bot.go      # Telegram bot logic (message handling, commands)
│       └── handler.go  # Alternative Telegram handler (currently used)
└── scripts/
    └── deploy_webhook.gs # Google Apps Script for the Sheets webhook
```

## Setup

1.  **Clone the repository:**
```bash
git clone https://github.com/jgabriele321/getitdone.git
cd getitdone
```

2.  **Set up Google Sheets & Apps Script:**
    *   Create a new Google Sheet.
    *   Rename the first tab to `todo`.
    *   Create a second tab named `team`.
        *   Add headers: `Name` in cell A1 and `Email` in cell B1.
        *   Populate with team members' names (lowercase) and their email addresses. This is used by the `/get_team` action in the Apps Script (though the bot doesn't use this directly yet).
    *   Open **Extensions > Apps Script**.
    *   Copy the content of `scripts/deploy_webhook.gs` into the Apps Script editor.
    *   Click **Deploy > New deployment**.
        *   Select type: **Web app**.
        *   Description: (e.g., "Go TODO Bot Webhook")
        *   Execute as: **Me**
        *   Who has access: **Anyone** (This is important for the webhook to be callable from your bot)
    *   Click **Deploy**.
    *   Authorize the script if prompted.
    *   Copy the **Web app URL**. This will be your `GOOGLE_SCRIPT_URL`.

3.  **Configure Environment Variables:**
    *   Create a `.env` file in the root of the project by copying `.env.example`:
```bash
cp .env.example .env
```
    *   Edit `.env` and fill in the following values:
        *   `TELEGRAM_TOKEN`: Your Telegram Bot Token (get this from BotFather on Telegram).
        *   `OPENROUTER_API_KEY`: Your API key for OpenRouter.ai.
        *   `LLM_MODEL_NAME`: The LLM model you want to use (e.g., `deepseek/deepseek-chat`).
        *   `GOOGLE_SCRIPT_URL`: The Web app URL you copied from Google Apps Script deployment.
        *   `AUTHORIZED_CHAT_IDS` (Optional): A comma-separated list of Telegram Chat IDs that are allowed to use the bot. If empty, all users can interact with it.
        *   `DATABASE_PATH`: Path to the SQLite database file (default: `./cache.db`).
        *   `PORT`: Port for the health check server (default: `8080`).
        *   `ENVIRONMENT`: Set to `development` or `production`.
        *   `TEST_MODE`: Set to `true` or `false`.

4.  **Build the bot:**
```bash
go build -o bot ./cmd/bot
```

5.  **Run the bot:**
```bash
./bot
```

## Usage

Interact with your bot on Telegram:

*   **Send tasks:**
    *   "Lilly to reach out to Johnny about the project"
    *   "Team needs to review the quarterly budget by Friday"
    *   "Sarah to finish the report AND Marcus to schedule the meeting for next week"
    *   "Remind me to call the doctor tomorrow at 10am"
*   **Use commands:**
    *   `/start` - Display a welcome message.
    *   `/help` - Show detailed help and examples.
    *   `/status` - Check if the bot is operational.

The bot will parse your message, log the task(s) to your Google Sheet, and send you a confirmation.

## How it Works

1.  **Telegram Input**: The user sends a message or command to the bot on Telegram.
2.  **Message Handling (`internal/telegram/handler.go` or `bot.go`)**:
    *   The bot receives the update.
    *   If it's a command, it's processed directly (e.g., `/start`, `/help`).
    *   If it's a regular message, it's sent for LLM processing.
3.  **LLM Parsing (`internal/llm/client.go`)**:
    *   The message text is sent to the configured LLM via OpenRouter.
    *   The LLM is prompted to extract tasks, assignees (people), and provide a confidence score.
    *   It returns a JSON response with the parsed task(s).
4.  **Google Sheets Logging (`internal/sheets/client.go`)**:
    *   The parsed tasks are formatted into the structure expected by the Google Apps Script.
    *   An HTTP POST request is made to the `GOOGLE_SCRIPT_URL` (your Apps Script webhook).
5.  **Google Apps Script (`scripts/deploy_webhook.gs`)**:
    *   The `doPost(e)` function in the Apps Script receives the request.
    *   It parses the JSON payload.
    *   For an `add_tasks` action, it iterates through the tasks and appends new rows to the "todo" sheet.
    *   It returns a JSON response indicating success or failure.
6.  **Confirmation (`internal/telegram/handler.go` or `bot.go`)**:
    *   The bot receives the response from the Google Sheets webhook.
    *   It sends a confirmation message back to the user on Telegram, summarizing the tasks added or reporting an error.

## Future Enhancements (Phases 2 & 3 from Scratchpad)

*   **Phase 2: Daily Email Digests**
    *   Cron job to fetch tasks from Google Sheets (e.g., tasks due today or overdue).
    *   Format tasks into an email.
    *   Send digest emails to relevant team members (using email addresses from the "team" sheet).
*   **Phase 3: Deployment to Render**
    *   Containerize the Go application using Docker.
    *   Set up a deployment pipeline on Render.com.
    *   Configure environment variables on Render.
    *   Ensure persistent storage for `cache.db` if needed beyond Render's ephemeral filesystem.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate. 