package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel provides common fields for all models
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// User represents a user in the system
type User struct {
	BaseModel
	Username      string     `gorm:"uniqueIndex;not null" json:"username"`
	Email         string     `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash  string     `gorm:"not null" json:"-"`
	FirstName     string     `json:"first_name"`
	LastName      string     `json:"last_name"`
	Role          string     `gorm:"default:'user'" json:"role"`
	IsActive      bool       `gorm:"default:true" json:"is_active"`
	IsVerified    bool       `gorm:"default:false" json:"is_verified"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	LoginAttempts int        `gorm:"default:0" json:"-"`
	LockedUntil   *time.Time `json:"-"`

	// Relationships
	Projects     []Project     `gorm:"foreignKey:CreatedBy" json:"projects,omitempty"`
	Tasks        []Task        `gorm:"foreignKey:AssignedTo" json:"tasks,omitempty"`
	ActivityLogs []ActivityLog `gorm:"foreignKey:UserID" json:"activity_logs,omitempty"`
}

// Project represents a research/development project
type Project struct {
	BaseModel
	Name           string     `gorm:"not null" json:"name"`
	Description    string     `gorm:"type:text" json:"description"`
	Type           string     `gorm:"not null" json:"type"`             // research, development, analysis, innovation
	Status         string     `gorm:"default:'active'" json:"status"`   // active, completed, paused, cancelled
	Priority       string     `gorm:"default:'medium'" json:"priority"` // low, medium, high, critical
	Progress       int        `gorm:"default:0" json:"progress"`        // 0-100
	StartDate      *time.Time `json:"start_date"`
	EndDate        *time.Time `json:"end_date"`
	EstimatedHours int        `gorm:"default:0" json:"estimated_hours"`
	ActualHours    int        `gorm:"default:0" json:"actual_hours"`
	Budget         float64    `gorm:"default:0" json:"budget"`
	Tags           string     `json:"tags"`                       // JSON array of tags
	Metadata       string     `gorm:"type:jsonb" json:"metadata"` // Additional project data
	CreatedBy      uuid.UUID  `gorm:"type:uuid;not null" json:"created_by"`

	// Relationships
	Creator   User       `gorm:"references:ID" json:"creator,omitempty"`
	Tasks     []Task     `gorm:"foreignKey:ProjectID" json:"tasks,omitempty"`
	Proposals []Proposal `gorm:"foreignKey:ProjectID" json:"proposals,omitempty"`
	Patterns  []Pattern  `gorm:"foreignKey:ProjectID" json:"patterns,omitempty"`
}

// Agent represents an AI agent in the system
type Agent struct {
	BaseModel
	Name            string     `gorm:"not null" json:"name"`
	Type            string     `gorm:"not null" json:"type"`             // researcher, coder, analyst, etc.
	Status          string     `gorm:"default:'inactive'" json:"status"` // active, inactive, busy, error
	Capabilities    string     `gorm:"type:jsonb" json:"capabilities"`   // JSON array of capabilities
	Config          string     `gorm:"type:jsonb" json:"config"`         // Agent configuration
	LastActive      *time.Time `json:"last_active"`
	TaskCount       int        `gorm:"default:0" json:"task_count"`
	SuccessRate     float64    `gorm:"default:0" json:"success_rate"`
	AvgResponseTime float64    `gorm:"default:0" json:"avg_response_time"` // in milliseconds

	// Relationships
	Tasks []Task `gorm:"foreignKey:AgentID" json:"tasks,omitempty"`
}

// Task represents a task within a project
type Task struct {
	BaseModel
	Title         string     `gorm:"not null" json:"title"`
	Description   string     `gorm:"type:text" json:"description"`
	Type          string     `gorm:"not null" json:"type"`            // analysis, coding, research, etc.
	Status        string     `gorm:"default:'pending'" json:"status"` // pending, in_progress, completed, failed
	Priority      string     `gorm:"default:'medium'" json:"priority"`
	Progress      int        `gorm:"default:0" json:"progress"`
	EstimatedTime int        `gorm:"default:0" json:"estimated_time"` // in minutes
	ActualTime    int        `gorm:"default:0" json:"actual_time"`
	StartedAt     *time.Time `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	Result        string     `gorm:"type:text" json:"result"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message"`
	ProjectID     *uuid.UUID `gorm:"type:uuid" json:"project_id"`
	AssignedTo    *uuid.UUID `gorm:"type:uuid" json:"assigned_to"`
	AgentID       *uuid.UUID `gorm:"type:uuid" json:"agent_id"`

	// Relationships
	Project      *Project `gorm:"references:ID" json:"project,omitempty"`
	AssignedUser *User    `gorm:"references:ID" json:"assigned_user,omitempty"`
	Agent        *Agent   `gorm:"references:ID" json:"agent,omitempty"`
}

// Proposal represents a project proposal
type Proposal struct {
	BaseModel
	Title           string     `gorm:"not null" json:"title"`
	Description     string     `gorm:"type:text;not null" json:"description"`
	Category        string     `gorm:"not null" json:"category"`
	Status          string     `gorm:"default:'pending'" json:"status"` // pending, approved, rejected, under_review
	Priority        string     `gorm:"default:'medium'" json:"priority"`
	EstimatedEffort int        `gorm:"default:0" json:"estimated_effort"` // in hours
	ExpectedOutcome string     `gorm:"type:text" json:"expected_outcome"`
	Justification   string     `gorm:"type:text" json:"justification"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	ReviewedBy      *uuid.UUID `gorm:"type:uuid" json:"reviewed_by"`
	ReviewNotes     string     `gorm:"type:text" json:"review_notes"`
	ProjectID       *uuid.UUID `gorm:"type:uuid" json:"project_id"`
	SubmittedBy     uuid.UUID  `gorm:"type:uuid;not null" json:"submitted_by"`

	// Relationships
	Project   *Project `gorm:"references:ID" json:"project,omitempty"`
	Submitter User     `gorm:"references:ID" json:"submitter,omitempty"`
	Reviewer  *User    `gorm:"references:ID" json:"reviewer,omitempty"`
}

// Pattern represents a recognized pattern in the system
type Pattern struct {
	BaseModel
	Name         string     `gorm:"not null" json:"name"`
	Type         string     `gorm:"not null" json:"type"` // user_behavior, system_usage, project_trend
	Description  string     `gorm:"type:text" json:"description"`
	Confidence   float64    `gorm:"default:0" json:"confidence"` // 0.0 - 1.0
	Frequency    int        `gorm:"default:0" json:"frequency"`
	Significance float64    `gorm:"default:0" json:"significance"`
	Data         string     `gorm:"type:jsonb" json:"data"`    // Pattern data
	Context      string     `gorm:"type:jsonb" json:"context"` // Context information
	LastSeen     time.Time  `json:"last_seen"`
	ProjectID    *uuid.UUID `gorm:"type:uuid" json:"project_id"`

	// Relationships
	Project  *Project  `gorm:"references:ID" json:"project,omitempty"`
	Insights []Insight `gorm:"foreignKey:PatternID" json:"insights,omitempty"`
}

// Insight represents an insight generated from patterns
type Insight struct {
	BaseModel
	Title         string     `gorm:"not null" json:"title"`
	Description   string     `gorm:"type:text;not null" json:"description"`
	Type          string     `gorm:"not null" json:"type"`   // optimization, recommendation, warning, trend
	Impact        string     `gorm:"not null" json:"impact"` // low, medium, high, critical
	Confidence    float64    `gorm:"default:0" json:"confidence"`
	ActionItems   string     `gorm:"type:jsonb" json:"action_items"` // JSON array of actions
	Data          string     `gorm:"type:jsonb" json:"data"`         // Supporting data
	IsActionable  bool       `gorm:"default:true" json:"is_actionable"`
	IsImplemented bool       `gorm:"default:false" json:"is_implemented"`
	PatternID     *uuid.UUID `gorm:"type:uuid" json:"pattern_id"`

	// Relationships
	Pattern *Pattern `gorm:"references:ID" json:"pattern,omitempty"`
}

// SystemMetric represents system performance metrics
type SystemMetric struct {
	BaseModel
	MetricType string    `gorm:"not null" json:"metric_type"` // cpu, memory, disk, network, response_time
	Value      float64   `gorm:"not null" json:"value"`
	Unit       string    `gorm:"not null" json:"unit"`
	Source     string    `gorm:"not null" json:"source"`   // server, database, api, etc.
	Labels     string    `gorm:"type:jsonb" json:"labels"` // Additional metric labels
	Timestamp  time.Time `gorm:"not null" json:"timestamp"`
}

// ActivityLog represents user and system activity
type ActivityLog struct {
	BaseModel
	Action       string     `gorm:"not null" json:"action"`
	Resource     string     `gorm:"not null" json:"resource"` // project, task, agent, etc.
	ResourceID   *uuid.UUID `gorm:"type:uuid" json:"resource_id"`
	Details      string     `gorm:"type:jsonb" json:"details"` // Additional activity details
	IPAddress    string     `json:"ip_address"`
	UserAgent    string     `json:"user_agent"`
	Success      bool       `gorm:"default:true" json:"success"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	UserID       *uuid.UUID `gorm:"type:uuid" json:"user_id"`

	// Relationships
	User *User `gorm:"references:ID" json:"user,omitempty"`
}

// TableName methods for custom table names (if needed)
func (User) TableName() string         { return "users" }
func (Project) TableName() string      { return "projects" }
func (Agent) TableName() string        { return "agents" }
func (Task) TableName() string         { return "tasks" }
func (Proposal) TableName() string     { return "proposals" }
func (Pattern) TableName() string      { return "patterns" }
func (Insight) TableName() string      { return "insights" }
func (SystemMetric) TableName() string { return "system_metrics" }
func (ActivityLog) TableName() string  { return "activity_logs" }
