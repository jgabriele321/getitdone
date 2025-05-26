package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/giovannigabriele/go-todo-bot/internal/config"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
)

// Client handles communication with OpenRouter LLM API
type Client struct {
	httpClient *retryablehttp.Client
	apiKey     string
	baseURL    string
	model      string
}

// ParsedTask represents a single parsed task
type ParsedTask struct {
	People     []string `json:"people"`
	Summary    string   `json:"summary"`
	Confidence float64  `json:"confidence"`
}

// ParseResponse represents the LLM's parsing response
type ParseResponse struct {
	Tasks           []ParsedTask `json:"tasks"`
	OriginalMessage string       `json:"original_message"`
}

// OpenRouterRequest represents the request to OpenRouter API
type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response from OpenRouter API
type OpenRouterResponse struct {
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a response choice
type Choice struct {
	Message      ResponseMessage `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// ResponseMessage represents the response message
type ResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewClient creates a new LLM client
func NewClient(cfg *config.Config) *Client {
	// Create retryable HTTP client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 10 * time.Second
	retryClient.Logger = nil // Disable retryable client logging

	return &Client{
		httpClient: retryClient,
		apiKey:     cfg.OpenRouterAPIKey,
		baseURL:    "https://openrouter.ai/api/v1",
		model:      "openai/gpt-4o-mini", // Cost-effective model as planned
	}
}

// ParseMessage parses a natural language message into structured tasks
func (c *Client) ParseMessage(ctx context.Context, message string) (*ParseResponse, error) {
	if strings.TrimSpace(message) == "" {
		return &ParseResponse{
			Tasks:           []ParsedTask{},
			OriginalMessage: message,
		}, nil
	}

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

	var response OpenRouterResponse
	if err := c.makeRequest(ctx, request, &response); err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from LLM")
	}

	content := response.Choices[0].Message.Content

	// Parse the JSON response
	var parseResponse ParseResponse
	if err := json.Unmarshal([]byte(content), &parseResponse); err != nil {
		log.Warn().
			Str("raw_response", content).
			Err(err).
			Msg("Failed to parse LLM JSON response, using fallback")

		// Fallback: create a single task assigned to "team"
		return &ParseResponse{
			Tasks: []ParsedTask{
				{
					People:     []string{"team"},
					Summary:    truncateString(message, 80),
					Confidence: 0.5,
				},
			},
			OriginalMessage: message,
		}, nil
	}

	// Validate and normalize the response
	c.normalizeResponse(&parseResponse, message)

	log.Debug().
		Int("tasks_parsed", len(parseResponse.Tasks)).
		Int("total_tokens", response.Usage.TotalTokens).
		Str("model", c.model).
		Msg("Successfully parsed message with LLM")

	return &parseResponse, nil
}

// buildPrompt creates the prompt for the LLM
func (c *Client) buildPrompt(message string) string {
	return fmt.Sprintf(`You are a task parser for a team management system. Parse the following message and extract task information.

Message: "%s"

Rules:
1. Identify WHO is responsible (person names or "team" if no specific person)
2. Extract WHAT needs to be done (the actual task/action)
3. Normalize names to lowercase (e.g., "Lilly" → "lilly", "The Team" → "team")
4. For multiple tasks, look for "AND" or clear task boundaries
5. Keep summaries concise but descriptive (max 80 chars)
6. If no specific person is mentioned, assign to "team"

Examples:
- "Lilly to reach out to Johnny to get a quote" → Person: lilly, Task: "Reach out to Johnny to get a quote"
- "Jemma, Lexi, and Johnny to set up meeting" → People: [jemma, lexi, johnny], Task: "Set up meeting"
- "We need to contact all vendors" → People: [team], Task: "Contact all vendors"

Return ONLY valid JSON in this exact format:
{
  "tasks": [
    {
      "people": ["lilly"],
      "summary": "Reach out to Johnny to get a quote",
      "confidence": 0.95
    }
  ],
  "original_message": "%s"
}`, message, message)
}

// normalizeResponse validates and normalizes the LLM response
func (c *Client) normalizeResponse(response *ParseResponse, originalMessage string) {
	response.OriginalMessage = originalMessage

	// Ensure we have at least one task
	if len(response.Tasks) == 0 {
		response.Tasks = []ParsedTask{
			{
				People:     []string{"team"},
				Summary:    truncateString(originalMessage, 80),
				Confidence: 0.5,
			},
		}
		return
	}

	// Normalize each task
	for i := range response.Tasks {
		task := &response.Tasks[i]

		// Normalize people names
		for j := range task.People {
			task.People[j] = strings.ToLower(strings.TrimSpace(task.People[j]))
			// Normalize "the team" variations
			if strings.Contains(task.People[j], "team") {
				task.People[j] = "team"
			}
		}

		// Remove duplicates and sort people
		task.People = removeDuplicatesAndSort(task.People)

		// Ensure we have at least one person
		if len(task.People) == 0 {
			task.People = []string{"team"}
		}

		// Truncate summary if too long
		task.Summary = truncateString(task.Summary, 80)

		// Ensure confidence is reasonable
		if task.Confidence < 0 || task.Confidence > 1 {
			task.Confidence = 0.8
		}
	}

	// Limit to max 3 tasks as per requirements
	if len(response.Tasks) > 3 {
		response.Tasks = response.Tasks[:3]
	}
}

// makeRequest makes an HTTP request to the OpenRouter API
func (c *Client) makeRequest(ctx context.Context, payload interface{}, response interface{}) error {
	// Marshal request payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := retryablehttp.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Decode response
	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// GetModel returns the current model being used
func (c *Client) GetModel() string {
	return c.model
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func removeDuplicatesAndSort(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	// Simple sort (alphabetical)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}
