0. High‑Level Roadmap
Phase	Goal	Owner	Key Deliverables	Target Duration
 1	Capture every Telegram message → parse with OpenRouter → log clean rows in Google Sheets	Go service	• Running bot
• Apps Script webhook
• Sheet schema & validation	3 days
 2	At 06:00 GMT daily send each teammate an email of their open tasks	Same service	• Cron job
• Task‑query layer
• Templated HTML e‑mail	2 days (after Phase 1 stable)
 3	Harden & ship to Render	Same service	• Health check endpoint
• Dockerfile & render.yaml
• Alerts/logging	1 day

1. Information Architecture
csharp
Copy
Edit
[Telegram user]
   │  (message)
   ▼
[Bot Service ‑ Go] ──▶ /parse                       (OpenRouter LLM)
   │      ▲               │
   │      └───────────────┘
   │   HTTP POST JSON
   ▼
[Apps Script Web‑App] ─▶ Google Sheet (single tab “todo”)
   │
   └─▶ (Phase 2) /daily‑digest (cron 06:00 GMT)
           │
           └─▶ SMTP | SendGrid → Individual e‑mails
Google Sheet columns (A → F):

Timestamp (RFC 3339)

People (comma‑separated, alpha; “team” normalized)

Summary (≤ 80 chars)

FullMessage

Status (Data‑validation: Not Started ▾ / In Progress / Complete)

OwnerEmail (resolved from People→Env map)

2. Tech Stack
Concern	Choice	Note
Telegram API	github.com/go-telegram-bot-api/telegram-bot-api/v5	Well‑maintained
LLM parsing	OpenRouter REST	pick any ≥ 8k‑token model
Outbound HTTP	net/http + retryablehttp	automatic back‑off
Scheduling	github.com/robfig/cron/v3	cron‑style strings
Email	SendGrid (SMTP fallback)	API key in ENV
Health	/healthz JSON	Render health‑check path
Container	scratch→alpine→distroless	final size < 25 MB

3. Repository /go‑todo‑bot
go
Copy
Edit
├── cmd/
│   └── bot/               (main.go)
├── internal/
│   ├── config/            (env, flags)
│   ├── telegram/          (handler.go, router.go)
│   ├── llm/               (client.go, prompt.go, parser.go)
│   ├── sheets/            (client.go, payload.go)
│   ├── email/             (sender.go, templates/)
│   ├── cron/              (jobs.go)
│   └── middleware/        (logging, recover)
├── scripts/
│   └── deploy_webhook.gs  (Apps Script)
├── Dockerfile
└── render.yaml
4. Core Functionality
Telegram Listener

Long‑poll ⟶ upgrade to webhook once Render HTTPS URL known.

For each Message, build IncomingMessage struct {text, user, timestamp}.

LLM Parser

Prompt: “Given the following chat note, extract… People[], Summary, SplitCount (1‑3). …Return JSON.”

Normalization rules (lower‑case, remove articles, singularize “team”).

If SplitCount > 1, clone rows and append “(2/3)” to summar(ies).

Google‑Sheet Writer

Build []interface{} value slice → marshal to JSON → POST to Apps Script.

Retry 3× exponential back‑off; log failures.

Daily Digest Cron (06:00 GMT)

Call Sheet “export?format=csv” to pull current table.

Filter Status != "Complete" AND OwnerEmail == <x>.

Compose HTML e‑mail with list (<ol>).

Send via SendGrid; on error push to dead_letter.log.

Health & Metrics

/healthz returns 200 {"ok":true,"uptime":123}.

/metrics optional Prometheus counter for sent rows / e‑mails.

5. Step‑by‑Step Implementation
Scaffold repo

bash
Copy
Edit
go mod init github.com/yourorg/go‑todo‑bot && mkdir -p cmd/internal
(.env) Minimal Variables

ini
Copy
Edit
TELEGRAM_TOKEN=xxx
OPENROUTER_API_KEY=xxx
GOOGLE_SCRIPT_URL=https://script.google.com/...
TEAM_EMAIL_MAP={"alice":"alice@acme.com","bob":"bob@acme.com","team":"team@acme.com"}
SENDGRID_KEY=xxx
Write Apps Script (deploy_webhook.gs)

Paste template from question; change column order; add data‑validation rule:

js
Copy
Edit
sheet.getRange("E:E").setDataValidation(
  SpreadsheetApp.newDataValidation()
   .requireValueInList(["Not Started","In Progress","Complete"], true)
   .build());
Telegram Handler (telegram/handler.go)

Switch on Message.IsCommand() for /start, else process note.

Ack user with 🔖 emoji and row number.

LLM Client

go
Copy
Edit
func ParseNote(ctx context.Context, raw string) (Rows []SheetRow, err error)
Stream or single call; unmarshal to struct.

Sheets Client

go
Copy
Edit
func (c *Client) AppendRows(ctx context.Context, rows []SheetRow) error
HTTP POST to GOOGLE_SCRIPT_URL.

Cron Job

go
Copy
Edit
c := cron.New(cron.WithLocation(time.UTC))
c.AddFunc("0 6 * * *", email.SendDailyDigest)
c.Start()
Email Sender

Parse CSV → map[email][]Task.

Use text/template + inline CSS.

SendGrid request; log ResponseID.

Dockerfile (distroless)

dockerfile
Copy
Edit
FROM golang:1.22-alpine AS build
RUN apk add --no-cache git
WORKDIR /app
COPY . .
RUN go build -o bot ./cmd/bot

FROM gcr.io/distroless/static
COPY --from=build /app/bot /bot
ENTRYPOINT ["/bot"]
render.yaml

yaml
Copy
Edit
services:
  - type: web
    name: go-todo-bot
    env: docker
    plan: free
    healthCheckPath: /healthz
    envVars:
      - fromGroup: todo-bot
Local Smoke Test

go run ./cmd/bot

Send Telegram message “Call Bob and Alice about Q3 OKRs by Friday.”

Confirm sheet row created, People="alice,bob", status “Not Started”.

Deploy

git push render main

Set webhook:

php-template
Copy
Edit
curl -X POST "https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://<render-url>/telegram"
Validate Cron (first run next 06:00 GMT). For immediate test: change schedule to @every 1m.

6. Next‑Step Enhancements
Inline “/done 42” command → toggle status via Apps Script PATCH.

Slack or email fall‑through if Telegram unreachable.

Per‑chat Sheets; command /switch project X.

Dashboard in Google Data Studio pulling from the Sheet.