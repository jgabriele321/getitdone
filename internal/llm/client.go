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
		if parseResp.Tasks[i].Client == "" {
			parseResp.Tasks[i].Client = "Internal"
		}

		// Ensure dueDate field is set
		if parseResp.Tasks[i].DueDate == "" {
			parseResp.Tasks[i].DueDate = "Unsure"
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
	return fmt.Sprintf(`You are a task parser. Parse this message and return ONLY valid JSON.

Current Date: %s
Message: "%s"

Extract:
1. people: array of names (lowercase) or ["team"] if no specific person
2. client: analyze the message to find client name (e.g. "for Microsoft" → "Microsoft", "to Johnny" → "Johnny")
3. summary: brief task description (max 80 chars)
4. dueDate: ONLY if explicitly mentioned in message, convert to YYYY-MM-DD. If no date mentioned, use "Unsure"
5. confidence: 0.0-1.0

Return this exact JSON format:
{
  "tasks": [{
    "people": ["lexi"],
    "client": "Johnny", 
    "summary": "Give Johnny a kiss by EOD today",
    "dueDate": "%s",
    "confidence": 0.95
  }],
  "original_message": "%s"
}`,
		currentTime.Format("2006-01-02"),
		message,
		currentTime.Format("2006-01-02"), // today for example
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
