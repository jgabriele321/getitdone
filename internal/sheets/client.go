package sheets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/giovannigabriele/go-todo-bot/internal/config"
	"github.com/rs/zerolog/log"
)

// Client handles communication with Google Sheets via Apps Script
type Client struct {
	scriptURL string
}

// Task represents a task to be added to the sheet
type Task struct {
	People      []string `json:"people"`
	Client      string   `json:"client"`
	Summary     string   `json:"summary"`
	FullMessage string   `json:"fullMessage"`
	DueDate     string   `json:"dueDate"`
	BotNotes    string   `json:"botNotes,omitempty"`
}

// TeamMember represents a team member from the team sheet
type TeamMember struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AddTasksRequest represents the request to add tasks
type AddTasksRequest struct {
	Action string `json:"action"`
	Tasks  []Task `json:"tasks"`
}

// GetTeamRequest represents the request to get team members
type GetTeamRequest struct {
	Action string `json:"action"`
}

// GetTasksRequest represents the request to get tasks
type GetTasksRequest struct {
	Action string `json:"action"`
	Status string `json:"status,omitempty"`
}

// Response represents the standard response from Apps Script
type Response struct {
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	RowsAdded int    `json:"rowsAdded,omitempty"`
}

// TeamResponse represents the response when getting team members
type TeamResponse struct {
	Status string       `json:"status"`
	Team   []TeamMember `json:"team"`
}

// TasksResponse represents the response when getting tasks
type TasksResponse struct {
	Status string      `json:"status"`
	Tasks  []SheetTask `json:"tasks"`
}

// SheetTask represents a task from the sheet
type SheetTask struct {
	Timestamp   time.Time `json:"timestamp"`
	People      []string  `json:"people"`
	Client      string    `json:"client"`
	Summary     string    `json:"summary"`
	FullMessage string    `json:"fullMessage"`
	Status      string    `json:"status"`
	DueDate     string    `json:"dueDate"`
	BotNotes    string    `json:"botNotes"`
}

// TaskRow represents a task row for the Google Sheet
type TaskRow struct {
	Timestamp   string   `json:"timestamp"`
	People      []string `json:"people"`
	Summary     string   `json:"summary"`
	FullMessage string   `json:"fullMessage"`
	Status      string   `json:"status"`
	BotNotes    string   `json:"botNotes"`
}

// AddTasksResponse represents the response from adding tasks
type AddTasksResponse struct {
	Status    string `json:"status"`
	RowsAdded int    `json:"rowsAdded"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// NewClient creates a new Google Sheets client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		scriptURL: cfg.GoogleScriptURL,
	}
}

// AddTasks adds new tasks to the Google Sheet
func (c *Client) AddTasks(ctx context.Context, tasks []Task) error {
	fmt.Printf("DEBUG: AddTasks called with %d TaskRow tasks\n", len(tasks))
	log.Debug().Int("task_count", len(tasks)).Msg("Adding tasks to Google Sheets")

	request := AddTasksRequest{
		Action: "add_tasks",
		Tasks:  tasks,
	}

	fmt.Printf("DEBUG: Created AddTasksRequest: %+v\n", request)

	var response AddTasksResponse
	if err := c.makeRequest(ctx, request, &response); err != nil {
		fmt.Printf("DEBUG: makeRequest failed: %v\n", err)
		return fmt.Errorf("failed to add tasks: %w", err)
	}

	fmt.Printf("DEBUG: AddTasksResponse: %+v\n", response)

	if response.Status != "success" {
		fmt.Printf("DEBUG: Response status is not success: %s, error: '%s'\n", response.Status, response.Error)
		return fmt.Errorf("sheets API error: %s", response.Error)
	}

	log.Info().
		Int("rows_added", response.RowsAdded).
		Msg("Successfully added tasks to Google Sheets")

	return nil
}

// GetTeam retrieves team members from the team sheet
func (c *Client) GetTeam(ctx context.Context) ([]TeamMember, error) {
	request := GetTeamRequest{
		Action: "get_team",
	}

	var response TeamResponse
	if err := c.makeRequest(ctx, request, &response); err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("sheets API error getting team")
	}

	log.Debug().
		Int("team_members", len(response.Team)).
		Msg("Retrieved team members from Google Sheets")

	return response.Team, nil
}

// GetTasks retrieves tasks from the sheet, optionally filtered by status
func (c *Client) GetTasks(ctx context.Context, statusFilter string) ([]SheetTask, error) {
	request := GetTasksRequest{
		Action: "get_tasks",
		Status: statusFilter,
	}

	var response TasksResponse
	if err := c.makeRequest(ctx, request, &response); err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("sheets API error getting tasks")
	}

	log.Debug().
		Int("tasks_count", len(response.Tasks)).
		Str("status_filter", statusFilter).
		Msg("Retrieved tasks from Google Sheets")

	return response.Tasks, nil
}

// makeRequest makes an HTTP request to the Apps Script webhook
func (c *Client) makeRequest(ctx context.Context, payload interface{}, response interface{}) error {
	// Marshal request payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	fmt.Printf("DEBUG: Making request to URL: %s\n", c.scriptURL)
	fmt.Printf("DEBUG: Payload: %s\n", string(jsonData))

	log.Debug().
		Str("url", c.scriptURL).
		RawJSON("payload", jsonData).
		Msg("Making request to Google Sheets")

	// Create HTTP request using standard client instead of retryable
	req, err := http.NewRequestWithContext(ctx, "POST", c.scriptURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Use standard HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("DEBUG: HTTP request failed: %v\n", err)
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf("DEBUG: Status Code: %d\n", resp.StatusCode)
	fmt.Printf("DEBUG: Response Body: %s\n", responseBody.String())

	log.Debug().
		Int("status_code", resp.StatusCode).
		Str("response_body", responseBody.String()).
		Msg("Received response from Google Sheets")

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d, body: %s", resp.StatusCode, responseBody.String())
	}

	// Decode response
	if err := json.NewDecoder(&responseBody).Decode(response); err != nil {
		fmt.Printf("DEBUG: JSON decode failed: %v\n", err)
		fmt.Printf("DEBUG: Response body was: %s\n", responseBody.String())
		return fmt.Errorf("failed to decode response: %w, body: %s", err, responseBody.String())
	}

	fmt.Printf("DEBUG: Successfully decoded response\n")
	return nil
}
