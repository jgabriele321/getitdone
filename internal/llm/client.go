package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client handles LLM interactions
type Client struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// Task represents a parsed task
type Task struct {
	People     []string `json:"people"`
	Client     string   `json:"client"`
	Summary    string   `json:"summary"`
	DueDate    string   `json:"dueDate"`
	Confidence float64  `json:"confidence"`
}

// ParseResponse represents the LLM response
type ParseResponse struct {
	Tasks           []Task `json:"tasks"`
	OriginalMessage string `json:"original_message"`
}

// OpenRouterRequest represents the request to OpenRouter
type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response from OpenRouter
type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
}

// Choice represents a choice in the response
type Choice struct {
	Message Message `json:"message"`
}

// NewClient creates a new LLM client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://openrouter.ai/api/v1/chat/completions",
		model:   "openai/gpt-4o-mini",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetModel returns the current model being used
func (c *Client) GetModel() string {
	return c.model
}

// ParseMessage parses a message using the LLM
func (c *Client) ParseMessage(ctx context.Context, message string) (*ParseResponse, error) {
	log.Debug().Str("message", message).Msg("Parsing message with LLM")

	prompt := c.buildPrompt(message)

	request := OpenRouterRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are a task parser that ONLY returns valid JSON. Never include explanations or additional text.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var openRouterResp OpenRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&openRouterResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openRouterResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := openRouterResp.Choices[0].Message.Content

	// Clean the content - remove any non-JSON text
	content = regexp.MustCompile(`(?s)^.*?(\{.*\}).*?$`).ReplaceAllString(content, "$1")

	// Try to parse the JSON response
	var parseResp ParseResponse
	if err := json.Unmarshal([]byte(content), &parseResp); err != nil {
		log.Warn().Str("content", content).Err(err).Msg("Failed to parse LLM response as JSON, creating fallback")
		// Create fallback response
		parseResp = ParseResponse{
			Tasks: []Task{
				{
					People:     []string{"team"},
					Client:     "Internal",
					Summary:    c.extractSummary(message),
					DueDate:    "Unsure",
					Confidence: 0.5,
				},
			},
			OriginalMessage: message,
		}
	}

	// Normalize people names
	for i := range parseResp.Tasks {
		parseResp.Tasks[i].People = c.normalizeNames(parseResp.Tasks[i].People)
		parseResp.Tasks[i].Summary = c.truncateSummary(parseResp.Tasks[i].Summary)

		// Ensure client field is set
		if parseResp.Tasks[i].Client == "" || parseResp.Tasks[i].Client == "Internal" {
			// For task chains, try to use the client from the first task
			if i > 0 && parseResp.Tasks[0].Client != "Internal" && parseResp.Tasks[0].Client != "Unsure" {
				parseResp.Tasks[i].Client = parseResp.Tasks[0].Client
			} else {
				parseResp.Tasks[i].Client = "Unsure"
			}
		}

		// Ensure dueDate field is set
		if parseResp.Tasks[i].DueDate == "" {
			parseResp.Tasks[i].DueDate = "Unsure"
		}

		// If confidence not set, use a default
		if parseResp.Tasks[i].Confidence == 0 {
			parseResp.Tasks[i].Confidence = 0.8
		}
	}

	log.Info().
		Int("task_count", len(parseResp.Tasks)).
		Interface("tasks", parseResp.Tasks).
		Msg("Successfully parsed message")

	return &parseResp, nil
}

// buildPrompt creates the prompt for the LLM
func (c *Client) buildPrompt(message string) string {
	currentTime := time.Now()
	return fmt.Sprintf(`Parse this message into tasks and return ONLY a JSON object, no other text.

Current Date: %s
Message: "%s"

Rules:
1. Split multi-task messages into separate tasks (look for bullet points, "AND", or clear task boundaries)
2. For each task:
   - people: array of who is DOING the task (lowercase) or ["team"] if unclear
   - client: who the task is FOR. Important rules for client:
     * If task is part of a chain/dependency, use the same client for all related tasks
     * If someone is asking for something, they are the client
     * If unclear, use "Unsure"
   - summary: brief task description (max 80 chars)
   - dueDate: ONLY if explicitly mentioned (YYYY-MM-DD format)
   - confidence: 0.0-1.0

Example Input: "Gemma to ask oxccu for press release, then Lilly to draft it by friday"
Example Output:
{
  "tasks": [
    {
      "people": ["gemma"],
      "client": "oxccu",
      "summary": "Ask for press release",
      "dueDate": "Unsure",
      "confidence": 0.95
    },
    {
      "people": ["lilly"],
      "client": "oxccu",
      "summary": "Draft press release",
      "dueDate": "%s",
      "confidence": 0.95
    }
  ],
  "original_message": "%s"
}

Return ONLY the JSON for the given message, no other text:`,
		currentTime.Format("2006-01-02"),
		message,
		currentTime.AddDate(0, 0, 5).Format("2006-01-02"), // Example future date
		message)
}

// normalizeNames normalizes person names
func (c *Client) normalizeNames(names []string) []string {
	var normalized []string
	for _, name := range names {
		clean := strings.ToLower(strings.TrimSpace(name))

		// Handle common variations
		if clean == "the team" || clean == "everyone" || clean == "all" {
			clean = "team"
		}

		// Remove articles and common words
		clean = regexp.MustCompile(`\b(the|a|an)\b`).ReplaceAllString(clean, "")
		clean = strings.TrimSpace(clean)

		if clean != "" {
			normalized = append(normalized, clean)
		}
	}

	if len(normalized) == 0 {
		normalized = []string{"team"}
	}

	return normalized
}

// truncateSummary ensures summary is within character limit
func (c *Client) truncateSummary(summary string) string {
	const maxLength = 80
	if len(summary) <= maxLength {
		return summary
	}
	return summary[:maxLength-3] + "..."
}

// getNextWeekday returns the next occurrence of the given weekday
func (c *Client) getNextWeekday(weekday time.Weekday) time.Time {
	now := time.Now()
	days := int(weekday - now.Weekday())
	if days <= 0 {
		days += 7
	}
	return now.AddDate(0, 0, days)
}

// extractSummary creates a basic summary from the message
func (c *Client) extractSummary(message string) string {
	// Simple extraction - take first sentence or first 80 chars
	sentences := strings.Split(message, ".")
	if len(sentences) > 0 && len(sentences[0]) > 0 {
		return c.truncateSummary(strings.TrimSpace(sentences[0]))
	}
	return c.truncateSummary(message)
}
