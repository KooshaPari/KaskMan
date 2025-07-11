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
	Creator           User                 `gorm:"references:ID" json:"creator,omitempty"`
	Tasks             []Task               `gorm:"foreignKey:ProjectID" json:"tasks,omitempty"`
	Proposals         []Proposal           `gorm:"foreignKey:ProjectID" json:"proposals,omitempty"`
	Patterns          []Pattern            `gorm:"foreignKey:ProjectID" json:"patterns,omitempty"`
	GitRepository     *GitRepository       `gorm:"foreignKey:ProjectID" json:"git_repository,omitempty"`
	Assets            []ProjectAsset       `gorm:"foreignKey:ProjectID" json:"assets,omitempty"`
	State             *ProjectState        `gorm:"foreignKey:ProjectID" json:"state,omitempty"`
	WorkflowExecutions []WorkflowExecution `gorm:"foreignKey:ProjectID" json:"workflow_executions,omitempty"`
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

// GitRepository represents a git repository associated with a project
type GitRepository struct {
	BaseModel
	ProjectID     uuid.UUID `gorm:"type:uuid;not null" json:"project_id"`
	RepositoryURL string    `gorm:"not null" json:"repository_url"`
	Branch        string    `gorm:"default:'main'" json:"branch"`
	LastCommitSHA string    `json:"last_commit_sha"`
	LastSyncAt    *time.Time `json:"last_sync_at"`
	Status        string    `gorm:"default:'active'" json:"status"` // active, inactive, error
	Credentials   string    `gorm:"type:jsonb" json:"-"`         // encrypted credentials
	WebhookSecret string    `json:"-"`                            // webhook secret for auto-sync

	// Relationships
	Project *Project `gorm:"references:ID" json:"project,omitempty"`
}

// ProjectAsset represents assets (screenshots, videos, docs) for a project
type ProjectAsset struct {
	BaseModel
	ProjectID   uuid.UUID `gorm:"type:uuid;not null" json:"project_id"`
	AssetType   string    `gorm:"not null" json:"asset_type"` // screenshot, video, document, demo
	Title       string    `gorm:"not null" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	FilePath    string    `gorm:"not null" json:"file_path"`
	FileSize    int64     `gorm:"default:0" json:"file_size"`
	MimeType    string    `json:"mime_type"`
	Thumbnail   string    `json:"thumbnail"`
	Metadata    string    `gorm:"type:jsonb" json:"metadata"` // resolution, duration, etc.
	Tags        string    `json:"tags"`
	IsPublic    bool      `gorm:"default:false" json:"is_public"`
	ViewCount   int       `gorm:"default:0" json:"view_count"`
	UploadedBy  uuid.UUID `gorm:"type:uuid;not null" json:"uploaded_by"`

	// Relationships
	Project  *Project `gorm:"references:ID" json:"project,omitempty"`
	Uploader User     `gorm:"references:ID" json:"uploader,omitempty"`
}

// ProjectState represents the current state of a project (CI/CD, tests, etc.)
type ProjectState struct {
	BaseModel
	ProjectID        uuid.UUID  `gorm:"type:uuid;not null" json:"project_id"`
	BuildStatus      string     `gorm:"default:'unknown'" json:"build_status"`    // success, failure, pending, unknown
	TestStatus       string     `gorm:"default:'unknown'" json:"test_status"`     // success, failure, pending, unknown
	LintStatus       string     `gorm:"default:'unknown'" json:"lint_status"`     // success, failure, pending, unknown
	SecurityStatus   string     `gorm:"default:'unknown'" json:"security_status"` // success, failure, pending, unknown
	DeploymentStatus string     `gorm:"default:'unknown'" json:"deployment_status"` // success, failure, pending, unknown
	Coverage         float64    `gorm:"default:0" json:"coverage"`
	LastCheckAt      *time.Time `json:"last_check_at"`
	CheckErrors      string     `gorm:"type:text" json:"check_errors"`
	HealthScore      int        `gorm:"default:0" json:"health_score"` // 0-100
	NextSteps        string     `gorm:"type:text" json:"next_steps"`
	ReadmePath       string     `json:"readme_path"`
	DemoURL          string     `json:"demo_url"`
	DocumentationURL string     `json:"documentation_url"`

	// Relationships
	Project *Project `gorm:"references:ID" json:"project,omitempty"`
}

// WorkflowExecution represents automated workflow executions
type WorkflowExecution struct {
	BaseModel
	ProjectID    uuid.UUID  `gorm:"type:uuid;not null" json:"project_id"`
	WorkflowType string     `gorm:"not null" json:"workflow_type"` // asset_generation, state_check, deployment
	TriggerType  string     `gorm:"not null" json:"trigger_type"`  // manual, scheduled, webhook, event
	Status       string     `gorm:"default:'pending'" json:"status"` // pending, running, completed, failed
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	Duration     int        `gorm:"default:0" json:"duration"` // in seconds
	Result       string     `gorm:"type:text" json:"result"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	Artifacts    string     `gorm:"type:jsonb" json:"artifacts"` // generated files, reports
	Configuration string    `gorm:"type:jsonb" json:"configuration"`
	TriggeredBy  uuid.UUID  `gorm:"type:uuid;not null" json:"triggered_by"`

	// Relationships
	Project   *Project `gorm:"references:ID" json:"project,omitempty"`
	Triggerer User     `gorm:"references:ID" json:"triggerer,omitempty"`
}

// ProjectTemplate represents reusable project templates
type ProjectTemplate struct {
	BaseModel
	Name         string    `gorm:"not null" json:"name"`
	Description  string    `gorm:"type:text" json:"description"`
	Category     string    `gorm:"not null" json:"category"`
	ProjectType  string    `gorm:"not null" json:"project_type"`
	Template     string    `gorm:"type:jsonb;not null" json:"template"` // project structure template
	Workflows    string    `gorm:"type:jsonb" json:"workflows"`          // default workflows
	IsPublic     bool      `gorm:"default:false" json:"is_public"`
	UsageCount   int       `gorm:"default:0" json:"usage_count"`
	Rating       float64   `gorm:"default:0" json:"rating"`
	CreatedBy    uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`

	// Relationships
	Creator User `gorm:"references:ID" json:"creator,omitempty"`
}

// TableName methods for custom table names (if needed)
func (User) TableName() string                { return "users" }
func (Project) TableName() string             { return "projects" }
func (Agent) TableName() string               { return "agents" }
func (Task) TableName() string                { return "tasks" }
func (Proposal) TableName() string            { return "proposals" }
func (Pattern) TableName() string             { return "patterns" }
func (Insight) TableName() string             { return "insights" }
func (SystemMetric) TableName() string        { return "system_metrics" }
func (ActivityLog) TableName() string         { return "activity_logs" }
func (GitRepository) TableName() string       { return "git_repositories" }
func (ProjectAsset) TableName() string        { return "project_assets" }
func (ProjectState) TableName() string        { return "project_states" }
func (WorkflowExecution) TableName() string   { return "workflow_executions" }
func (ProjectTemplate) TableName() string     { return "project_templates" }
