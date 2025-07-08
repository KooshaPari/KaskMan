package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
)

// Client represents the API client for KaskManager
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

// NewClient creates a new API client
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) {
	c.Token = token
}

// makeRequest makes an HTTP request with authentication
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

// decodeResponse decodes the response body into the target struct
func decodeResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errorResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}
		return fmt.Errorf("API error: %s", errorResp.Error)
	}

	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}

	return nil
}

// Health checks the API health
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return nil, err
	}

	var health HealthResponse
	if err := decodeResponse(resp, &health); err != nil {
		return nil, err
	}

	return &health, nil
}

// GetSystemStatus gets the system status
func (c *Client) GetSystemStatus(ctx context.Context) (*SystemStatusResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/status", nil)
	if err != nil {
		return nil, err
	}

	var status SystemStatusResponse
	if err := decodeResponse(resp, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// Login authenticates with the API
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.makeRequest(ctx, "POST", "/api/v1/auth/login", loginReq)
	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	if err := decodeResponse(resp, &loginResp); err != nil {
		return nil, err
	}

	c.SetToken(loginResp.Token)
	return &loginResp, nil
}

// Projects
func (c *Client) GetProjects(ctx context.Context) ([]models.Project, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/projects", nil)
	if err != nil {
		return nil, err
	}

	var projects []models.Project
	if err := decodeResponse(resp, &projects); err != nil {
		return nil, err
	}

	return projects, nil
}

func (c *Client) CreateProject(ctx context.Context, project CreateProjectRequest) (*models.Project, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/projects", project)
	if err != nil {
		return nil, err
	}

	var createdProject models.Project
	if err := decodeResponse(resp, &createdProject); err != nil {
		return nil, err
	}

	return &createdProject, nil
}

func (c *Client) GetProject(ctx context.Context, id string) (*models.Project, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/projects/"+id, nil)
	if err != nil {
		return nil, err
	}

	var project models.Project
	if err := decodeResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

func (c *Client) UpdateProject(ctx context.Context, id string, project UpdateProjectRequest) (*models.Project, error) {
	resp, err := c.makeRequest(ctx, "PUT", "/api/v1/projects/"+id, project)
	if err != nil {
		return nil, err
	}

	var updatedProject models.Project
	if err := decodeResponse(resp, &updatedProject); err != nil {
		return nil, err
	}

	return &updatedProject, nil
}

func (c *Client) DeleteProject(ctx context.Context, id string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/api/v1/projects/"+id, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// Tasks
func (c *Client) GetTasks(ctx context.Context) ([]models.Task, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/tasks", nil)
	if err != nil {
		return nil, err
	}

	var tasks []models.Task
	if err := decodeResponse(resp, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (c *Client) CreateTask(ctx context.Context, task CreateTaskRequest) (*models.Task, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/tasks", task)
	if err != nil {
		return nil, err
	}

	var createdTask models.Task
	if err := decodeResponse(resp, &createdTask); err != nil {
		return nil, err
	}

	return &createdTask, nil
}

func (c *Client) GetTask(ctx context.Context, id string) (*models.Task, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/tasks/"+id, nil)
	if err != nil {
		return nil, err
	}

	var task models.Task
	if err := decodeResponse(resp, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

func (c *Client) UpdateTask(ctx context.Context, id string, task UpdateTaskRequest) (*models.Task, error) {
	resp, err := c.makeRequest(ctx, "PUT", "/api/v1/tasks/"+id, task)
	if err != nil {
		return nil, err
	}

	var updatedTask models.Task
	if err := decodeResponse(resp, &updatedTask); err != nil {
		return nil, err
	}

	return &updatedTask, nil
}

func (c *Client) DeleteTask(ctx context.Context, id string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/api/v1/tasks/"+id, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// Agents
func (c *Client) GetAgents(ctx context.Context) ([]models.Agent, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/agents", nil)
	if err != nil {
		return nil, err
	}

	var agents []models.Agent
	if err := decodeResponse(resp, &agents); err != nil {
		return nil, err
	}

	return agents, nil
}

func (c *Client) CreateAgent(ctx context.Context, agent CreateAgentRequest) (*models.Agent, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/agents", agent)
	if err != nil {
		return nil, err
	}

	var createdAgent models.Agent
	if err := decodeResponse(resp, &createdAgent); err != nil {
		return nil, err
	}

	return &createdAgent, nil
}

func (c *Client) GetAgent(ctx context.Context, id string) (*models.Agent, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/agents/"+id, nil)
	if err != nil {
		return nil, err
	}

	var agent models.Agent
	if err := decodeResponse(resp, &agent); err != nil {
		return nil, err
	}

	return &agent, nil
}

func (c *Client) UpdateAgent(ctx context.Context, id string, agent UpdateAgentRequest) (*models.Agent, error) {
	resp, err := c.makeRequest(ctx, "PUT", "/api/v1/agents/"+id, agent)
	if err != nil {
		return nil, err
	}

	var updatedAgent models.Agent
	if err := decodeResponse(resp, &updatedAgent); err != nil {
		return nil, err
	}

	return &updatedAgent, nil
}

func (c *Client) DeleteAgent(ctx context.Context, id string) error {
	resp, err := c.makeRequest(ctx, "DELETE", "/api/v1/agents/"+id, nil)
	if err != nil {
		return err
	}

	return decodeResponse(resp, nil)
}

// R&D Operations
func (c *Client) AnalyzePatterns(ctx context.Context, req AnalyzePatternsRequest) (*AnalyzePatternsResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/rnd/analyze", req)
	if err != nil {
		return nil, err
	}

	var result AnalyzePatternsResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GenerateProjects(ctx context.Context, req GenerateProjectsRequest) (*GenerateProjectsResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/rnd/generate", req)
	if err != nil {
		return nil, err
	}

	var result GenerateProjectsResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetInsights(ctx context.Context) ([]models.Insight, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/rnd/insights", nil)
	if err != nil {
		return nil, err
	}

	var insights []models.Insight
	if err := decodeResponse(resp, &insights); err != nil {
		return nil, err
	}

	return insights, nil
}

func (c *Client) CoordinateAgents(ctx context.Context, req CoordinateAgentsRequest) (*CoordinateAgentsResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/rnd/coordinate", req)
	if err != nil {
		return nil, err
	}

	var result CoordinateAgentsResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetPatterns(ctx context.Context) ([]models.Pattern, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/rnd/patterns", nil)
	if err != nil {
		return nil, err
	}

	var patterns []models.Pattern
	if err := decodeResponse(resp, &patterns); err != nil {
		return nil, err
	}

	return patterns, nil
}

func (c *Client) GetRnDStats(ctx context.Context) (*RnDStatsResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/rnd/stats", nil)
	if err != nil {
		return nil, err
	}

	var stats RnDStatsResponse
	if err := decodeResponse(resp, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}
