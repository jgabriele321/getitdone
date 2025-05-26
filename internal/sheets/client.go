package sheets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client handles Google Sheets operations
type Client struct {
	webhookURL string
	client     *http.Client
}

// TaskRow represents a task row for the Google Sheet
type TaskRow struct {
	Timestamp   string `json:"timestamp"`
	People      []string `json:"people"`
	Summary     string `json:"summary"`
	FullMessage string `json:"fullMessage"`
	Status      string `json:"status"`
	BotNotes    string `json:"botNotes"`
}

// AddTasksRequest represents the request to add tasks
type AddTasksRequest struct {
	Action string    `json:"action"`
	Tasks  []TaskRow `json:"tasks"`
}

// GetTeamRequest represents the request to get team data
type GetTeamRequest struct {
	Action string `json:"action"`
}

// TeamMember represents a team member
type TeamMember struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetTeamResponse represents the response from getting team data
type GetTeamResponse struct {
	Status  string       `json:"status"`
	Team    []TeamMember `json:"team"`
	Error   string       `json:"error,omitempty"`
}

// AddTasksResponse represents the response from adding tasks
type AddTasksResponse struct {
	Status    string `json:"status"`
	RowsAdded int    `json:"rowsAdded"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
}

// NewClient creates a new Google Sheets client
func NewClient(webhookURL string) *Client {
	return &Client{
		webhookURL: webhookURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddTasks adds tasks to the Google Sheet
func (c *Client) AddTasks(ctx context.Context, tasks []TaskRow) error {
	log.Debug().Int("task_count", len(tasks)).Msg("Adding tasks to Google Sheets")

	request := AddTasksRequest{
		Action: "add_tasks",
		Tasks:  tasks,
	}

	var response AddTasksResponse
	if err := c.makeRequest(ctx, request, &response); err != nil {
		return fmt.Errorf("failed to add tasks: %w", err)
	}

	if response.Status != "success" {
		return fmt.Errorf("sheets API error: %s", response.Error)
	}

	log.Info().
		Int("rows_added", response.RowsAdded).
		Msg("Successfully added tasks to Google Sheets")

	return nil
}

// GetTeam retrieves team members from the Google Sheet
func (c *Client) GetTeam(ctx context.Context) ([]TeamMember, error) {
	log.Debug().Msg("Getting team data from Google Sheets")

	request := GetTeamRequest{
		Action: "get_team",
	}

	var response GetTeamResponse
	if err := c.makeRequest(ctx, request, &response); err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("sheets API error: %s", response.Error)
	}

	log.Info().
		Int("team_size", len(response.Team)).
		Msg("Successfully retrieved team data")

	return response.Team, nil
}

// CreateTaskRow creates a TaskRow from parsed task data
func CreateTaskRow(people []string, summary, fullMessage, botNotes string) TaskRow {
	return TaskRow{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		People:      people,
		Summary:     summary,
		FullMessage: fullMessage,
		Status:      "Not Started",
		BotNotes:    botNotes,
	}
}

// makeRequest makes an HTTP request to the Google Apps Script webhook
func (c *Client) makeRequest(ctx context.Context, request interface{}, response interface{}) error {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// BuildTeamEmailMap builds a map from team member names to emails
func BuildTeamEmailMap(team []TeamMember) map[string]string {
	emailMap := make(map[string]string)
	for _, member := range team {
		// Normalize name to lowercase for consistent lookup
		normalizedName := strings.ToLower(strings.TrimSpace(member.Name))
		emailMap[normalizedName] = member.Email
	}
	return emailMap
}
