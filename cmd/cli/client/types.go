package client

import (
	"time"

	"github.com/google/uuid"
)

// Authentication types
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      struct {
		ID       uuid.UUID `json:"id"`
		Username string    `json:"username"`
		Email    string    `json:"email"`
		Role     string    `json:"role"`
	} `json:"user"`
}

// Health and Status types
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

type SystemStatusResponse struct {
	Status     string                 `json:"status"`
	Timestamp  time.Time              `json:"timestamp"`
	Version    string                 `json:"version"`
	Uptime     string                 `json:"uptime"`
	Services   map[string]interface{} `json:"services"`
	Metrics    map[string]interface{} `json:"metrics"`
	Database   DatabaseStatus         `json:"database"`
	Redis      RedisStatus            `json:"redis"`
	WebSocket  WebSocketStatus        `json:"websocket"`
	RnD        RnDStatus              `json:"rnd"`
	Monitoring MonitoringStatus       `json:"monitoring"`
}

type DatabaseStatus struct {
	Connected     bool `json:"connected"`
	MaxOpenConns  int  `json:"max_open_conns"`
	OpenConns     int  `json:"open_conns"`
	InUse         int  `json:"in_use"`
	Idle          int  `json:"idle"`
	WaitCount     int  `json:"wait_count"`
	WaitDuration  int  `json:"wait_duration"`
	MaxIdleTime   int  `json:"max_idle_time"`
	MaxLifetime   int  `json:"max_lifetime"`
	MaxIdleClosed int  `json:"max_idle_closed"`
	MaxLifeClosed int  `json:"max_life_closed"`
}

type RedisStatus struct {
	Connected   bool `json:"connected"`
	PoolSize    int  `json:"pool_size"`
	ActiveConns int  `json:"active_conns"`
	IdleConns   int  `json:"idle_conns"`
}

type WebSocketStatus struct {
	Enabled bool `json:"enabled"`
	Clients int  `json:"clients"`
}

type RnDStatus struct {
	Enabled        bool `json:"enabled"`
	Workers        int  `json:"workers"`
	QueueSize      int  `json:"queue_size"`
	ProcessingJobs int  `json:"processing_jobs"`
}

type MonitoringStatus struct {
	Enabled bool `json:"enabled"`
	Metrics int  `json:"metrics"`
}

// Project types
type CreateProjectRequest struct {
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	Type           string     `json:"type"`
	Priority       string     `json:"priority"`
	StartDate      *time.Time `json:"start_date"`
	EndDate        *time.Time `json:"end_date"`
	EstimatedHours int        `json:"estimated_hours"`
	Budget         float64    `json:"budget"`
	Tags           []string   `json:"tags"`
}

type UpdateProjectRequest struct {
	Name           *string    `json:"name,omitempty"`
	Description    *string    `json:"description,omitempty"`
	Type           *string    `json:"type,omitempty"`
	Status         *string    `json:"status,omitempty"`
	Priority       *string    `json:"priority,omitempty"`
	Progress       *int       `json:"progress,omitempty"`
	StartDate      *time.Time `json:"start_date,omitempty"`
	EndDate        *time.Time `json:"end_date,omitempty"`
	EstimatedHours *int       `json:"estimated_hours,omitempty"`
	ActualHours    *int       `json:"actual_hours,omitempty"`
	Budget         *float64   `json:"budget,omitempty"`
	Tags           []string   `json:"tags,omitempty"`
}

// Task types
type CreateTaskRequest struct {
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Type          string     `json:"type"`
	Priority      string     `json:"priority"`
	EstimatedTime int        `json:"estimated_time"`
	ProjectID     *uuid.UUID `json:"project_id,omitempty"`
	AssignedTo    *uuid.UUID `json:"assigned_to,omitempty"`
	AgentID       *uuid.UUID `json:"agent_id,omitempty"`
}

type UpdateTaskRequest struct {
	Title         *string    `json:"title,omitempty"`
	Description   *string    `json:"description,omitempty"`
	Type          *string    `json:"type,omitempty"`
	Status        *string    `json:"status,omitempty"`
	Priority      *string    `json:"priority,omitempty"`
	Progress      *int       `json:"progress,omitempty"`
	EstimatedTime *int       `json:"estimated_time,omitempty"`
	ActualTime    *int       `json:"actual_time,omitempty"`
	Result        *string    `json:"result,omitempty"`
	ProjectID     *uuid.UUID `json:"project_id,omitempty"`
	AssignedTo    *uuid.UUID `json:"assigned_to,omitempty"`
	AgentID       *uuid.UUID `json:"agent_id,omitempty"`
}

// Agent types
type CreateAgentRequest struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Capabilities []string               `json:"capabilities"`
	Config       map[string]interface{} `json:"config"`
}

type UpdateAgentRequest struct {
	Name         *string                `json:"name,omitempty"`
	Type         *string                `json:"type,omitempty"`
	Status       *string                `json:"status,omitempty"`
	Capabilities []string               `json:"capabilities,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// R&D Operation types
type AnalyzePatternsRequest struct {
	Type      string                 `json:"type"`
	Context   string                 `json:"context"`
	Depth     int                    `json:"depth"`
	TimeRange string                 `json:"time_range"`
	Filters   map[string]interface{} `json:"filters"`
}

type AnalyzePatternsResponse struct {
	PatternsFound int                    `json:"patterns_found"`
	Patterns      []PatternSummary       `json:"patterns"`
	Insights      []InsightSummary       `json:"insights"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type PatternSummary struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	Confidence   float64   `json:"confidence"`
	Frequency    int       `json:"frequency"`
	Significance float64   `json:"significance"`
}

type InsightSummary struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	Impact      string    `json:"impact"`
	Confidence  float64   `json:"confidence"`
	ActionItems []string  `json:"action_items"`
}

type GenerateProjectsRequest struct {
	Category     string                 `json:"category"`
	Focus        string                 `json:"focus"`
	Priority     string                 `json:"priority"`
	MaxProjects  int                    `json:"max_projects"`
	Constraints  map[string]interface{} `json:"constraints"`
	Requirements []string               `json:"requirements"`
}

type GenerateProjectsResponse struct {
	ProjectsGenerated int                    `json:"projects_generated"`
	Projects          []ProjectSuggestion    `json:"projects"`
	Metadata          map[string]interface{} `json:"metadata"`
}

type ProjectSuggestion struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Type            string   `json:"type"`
	Priority        string   `json:"priority"`
	EstimatedHours  int      `json:"estimated_hours"`
	Budget          float64  `json:"budget"`
	Tags            []string `json:"tags"`
	Justification   string   `json:"justification"`
	ExpectedOutcome string   `json:"expected_outcome"`
}

type CoordinateAgentsRequest struct {
	TaskID     uuid.UUID              `json:"task_id"`
	ProjectID  uuid.UUID              `json:"project_id"`
	AgentTypes []string               `json:"agent_types"`
	Strategy   string                 `json:"strategy"`
	Config     map[string]interface{} `json:"config"`
}

type CoordinateAgentsResponse struct {
	CoordinationID uuid.UUID              `json:"coordination_id"`
	AssignedAgents []AgentAssignment      `json:"assigned_agents"`
	Strategy       string                 `json:"strategy"`
	Status         string                 `json:"status"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type AgentAssignment struct {
	AgentID   uuid.UUID `json:"agent_id"`
	AgentName string    `json:"agent_name"`
	Role      string    `json:"role"`
	Tasks     []string  `json:"tasks"`
}

type RnDStatsResponse struct {
	TotalPatterns         int                    `json:"total_patterns"`
	TotalInsights         int                    `json:"total_insights"`
	ActiveAgents          int                    `json:"active_agents"`
	ProcessingJobs        int                    `json:"processing_jobs"`
	CompletedAnalyses     int                    `json:"completed_analyses"`
	AverageProcessingTime float64                `json:"average_processing_time"`
	PatternsByType        map[string]int         `json:"patterns_by_type"`
	InsightsByImpact      map[string]int         `json:"insights_by_impact"`
	RecentActivity        []ActivitySummary      `json:"recent_activity"`
	PerformanceMetrics    map[string]interface{} `json:"performance_metrics"`
}

type ActivitySummary struct {
	Type      string    `json:"type"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// Configuration types
type ConfigResponse struct {
	Environment string                 `json:"environment"`
	Server      map[string]interface{} `json:"server"`
	Database    map[string]interface{} `json:"database"`
	Redis       map[string]interface{} `json:"redis"`
	Auth        map[string]interface{} `json:"auth"`
	RnD         map[string]interface{} `json:"rnd"`
	Monitoring  map[string]interface{} `json:"monitoring"`
	WebSocket   map[string]interface{} `json:"websocket"`
}

// Error types
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// List response wrapper
type ListResponse[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}
