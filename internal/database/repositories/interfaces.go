package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"gorm.io/gorm"
)

// Pagination represents pagination parameters
type Pagination struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Sort     string `json:"sort" form:"sort"`
	Order    string `json:"order" form:"order"`
}

// PaginationResult represents paginated results
type PaginationResult struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// Filter represents generic filtering options
type Filter map[string]interface{}

// TransactionManager manages database transactions
type TransactionManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	BeginTransaction(ctx context.Context) (context.Context, error)
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error
}

// CacheManager manages caching operations
type CacheManager interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context, pattern string) error
	SetMany(ctx context.Context, items map[string]interface{}, expiration time.Duration) error
	GetMany(ctx context.Context, keys []string) (map[string]interface{}, error)
}

// BaseRepository defines common repository operations
type BaseRepository interface {
	Create(ctx context.Context, entity interface{}) error
	GetByID(ctx context.Context, id uuid.UUID, entity interface{}) error
	Update(ctx context.Context, entity interface{}) error
	Delete(ctx context.Context, id uuid.UUID, entity interface{}) error
	SoftDelete(ctx context.Context, id uuid.UUID, entity interface{}) error
	List(ctx context.Context, entities interface{}, filters Filter) error
	ListWithPagination(ctx context.Context, entities interface{}, pagination Pagination, filters Filter) (*PaginationResult, error)
	Count(ctx context.Context, entity interface{}, filters Filter) (int64, error)
	Exists(ctx context.Context, id uuid.UUID, entity interface{}) (bool, error)
	BatchCreate(ctx context.Context, entities interface{}) error
	BatchUpdate(ctx context.Context, entities interface{}) error
	BatchDelete(ctx context.Context, ids []uuid.UUID, entity interface{}) error
}

// UserRepository defines user-specific repository operations
type UserRepository interface {
	BaseRepository
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByCredentials(ctx context.Context, identifier string) (*models.User, error)
	GetActiveUsers(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetUsersByRole(ctx context.Context, role string, pagination Pagination) (*PaginationResult, error)
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	IncrementLoginAttempts(ctx context.Context, userID uuid.UUID) error
	ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error
	LockUser(ctx context.Context, userID uuid.UUID, duration time.Duration) error
	UnlockUser(ctx context.Context, userID uuid.UUID) error
	GetLockedUsers(ctx context.Context) ([]models.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error
	GetUserStatistics(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error)
	SearchUsers(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
}

// ProjectRepository defines project-specific repository operations
type ProjectRepository interface {
	BaseRepository
	GetByCreator(ctx context.Context, creatorID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error)
	GetByType(ctx context.Context, projectType string, pagination Pagination) (*PaginationResult, error)
	GetByPriority(ctx context.Context, priority string, pagination Pagination) (*PaginationResult, error)
	GetWithTasks(ctx context.Context, projectID uuid.UUID) (*models.Project, error)
	GetWithProposals(ctx context.Context, projectID uuid.UUID) (*models.Project, error)
	GetWithPatterns(ctx context.Context, projectID uuid.UUID) (*models.Project, error)
	GetProjectStatistics(ctx context.Context, projectID uuid.UUID) (map[string]interface{}, error)
	GetActiveProjects(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetRecentProjects(ctx context.Context, limit int) ([]models.Project, error)
	UpdateProgress(ctx context.Context, projectID uuid.UUID, progress int) error
	GetOverdueProjects(ctx context.Context) ([]models.Project, error)
	GetProjectsByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error)
	GetProjectsByTags(ctx context.Context, tags []string, pagination Pagination) (*PaginationResult, error)
	SearchProjects(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetProjectsWithTaskCounts(ctx context.Context, pagination Pagination) (*PaginationResult, error)
}

// TaskRepository defines task-specific repository operations
type TaskRepository interface {
	BaseRepository
	GetByProject(ctx context.Context, projectID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByAssignee(ctx context.Context, assigneeID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByAgent(ctx context.Context, agentID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error)
	GetByPriority(ctx context.Context, priority string, pagination Pagination) (*PaginationResult, error)
	GetByType(ctx context.Context, taskType string, pagination Pagination) (*PaginationResult, error)
	GetPendingTasks(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetInProgressTasks(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetCompletedTasks(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetOverdueTasks(ctx context.Context) ([]models.Task, error)
	GetTasksByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error)
	GetTasksCompletedToday(ctx context.Context) ([]models.Task, error)
	GetTasksCompletedInPeriod(ctx context.Context, startDate, endDate time.Time) ([]models.Task, error)
	UpdateStatus(ctx context.Context, taskID uuid.UUID, status string) error
	UpdateProgress(ctx context.Context, taskID uuid.UUID, progress int) error
	AssignTask(ctx context.Context, taskID uuid.UUID, assigneeID *uuid.UUID, agentID *uuid.UUID) error
	UnassignTask(ctx context.Context, taskID uuid.UUID) error
	GetTaskStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error)
	GetTasksByEstimatedTime(ctx context.Context, minTime, maxTime int, pagination Pagination) (*PaginationResult, error)
	SearchTasks(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetTasksWithRelations(ctx context.Context, pagination Pagination) (*PaginationResult, error)
}

// ProposalRepository defines proposal-specific repository operations
type ProposalRepository interface {
	BaseRepository
	GetBySubmitter(ctx context.Context, submitterID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error)
	GetByCategory(ctx context.Context, category string, pagination Pagination) (*PaginationResult, error)
	GetByPriority(ctx context.Context, priority string, pagination Pagination) (*PaginationResult, error)
	GetByProject(ctx context.Context, projectID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByReviewer(ctx context.Context, reviewerID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetPendingProposals(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetApprovedProposals(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetRejectedProposals(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetUnderReviewProposals(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetProposalsByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error)
	ApproveProposal(ctx context.Context, proposalID uuid.UUID, reviewerID uuid.UUID, reviewNotes string) error
	RejectProposal(ctx context.Context, proposalID uuid.UUID, reviewerID uuid.UUID, reviewNotes string) error
	SetUnderReview(ctx context.Context, proposalID uuid.UUID, reviewerID uuid.UUID) error
	GetProposalStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error)
	GetRecentProposals(ctx context.Context, limit int) ([]models.Proposal, error)
	SearchProposals(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetProposalsByEffortRange(ctx context.Context, minEffort, maxEffort int, pagination Pagination) (*PaginationResult, error)
}

// AgentRepository defines agent-specific repository operations
type AgentRepository interface {
	BaseRepository
	GetByType(ctx context.Context, agentType string, pagination Pagination) (*PaginationResult, error)
	GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error)
	GetActiveAgents(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetInactiveAgents(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetBusyAgents(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetAvailableAgents(ctx context.Context, agentType string) ([]models.Agent, error)
	GetAgentWithTasks(ctx context.Context, agentID uuid.UUID) (*models.Agent, error)
	UpdateStatus(ctx context.Context, agentID uuid.UUID, status string) error
	UpdateLastActive(ctx context.Context, agentID uuid.UUID) error
	UpdateTaskCount(ctx context.Context, agentID uuid.UUID, count int) error
	UpdateSuccessRate(ctx context.Context, agentID uuid.UUID, rate float64) error
	UpdateResponseTime(ctx context.Context, agentID uuid.UUID, responseTime float64) error
	GetAgentStatistics(ctx context.Context, agentID uuid.UUID) (map[string]interface{}, error)
	GetAgentsByCapabilities(ctx context.Context, capabilities []string, pagination Pagination) (*PaginationResult, error)
	GetAgentsByLastActive(ctx context.Context, since time.Time, pagination Pagination) (*PaginationResult, error)
	GetTopPerformingAgents(ctx context.Context, limit int) ([]models.Agent, error)
	SearchAgents(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetAgentWorkload(ctx context.Context, agentID uuid.UUID) (map[string]interface{}, error)
}

// ActivityLogRepository defines activity log-specific repository operations
type ActivityLogRepository interface {
	BaseRepository
	GetByUser(ctx context.Context, userID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByAction(ctx context.Context, action string, pagination Pagination) (*PaginationResult, error)
	GetByResource(ctx context.Context, resource string, pagination Pagination) (*PaginationResult, error)
	GetByResourceID(ctx context.Context, resourceID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error)
	GetRecentActivities(ctx context.Context, limit int) ([]models.ActivityLog, error)
	GetSuccessfulActivities(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetFailedActivities(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetActivitiesWithErrors(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetUserActivityStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error)
	GetSystemActivityStats(ctx context.Context) (map[string]interface{}, error)
	GetActivityTrends(ctx context.Context, period string) (map[string]interface{}, error)
	LogActivity(ctx context.Context, userID *uuid.UUID, action, resource string, resourceID *uuid.UUID, details map[string]interface{}, success bool, errorMessage string) error
	SearchActivities(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetActivitiesByIPAddress(ctx context.Context, ipAddress string, pagination Pagination) (*PaginationResult, error)
	CleanupOldActivities(ctx context.Context, olderThan time.Time) error
}

// PatternRepository defines pattern-specific repository operations
type PatternRepository interface {
	BaseRepository
	GetByType(ctx context.Context, patternType string, pagination Pagination) (*PaginationResult, error)
	GetByProject(ctx context.Context, projectID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByConfidenceRange(ctx context.Context, minConfidence, maxConfidence float64, pagination Pagination) (*PaginationResult, error)
	GetBySignificanceRange(ctx context.Context, minSignificance, maxSignificance float64, pagination Pagination) (*PaginationResult, error)
	GetByFrequencyRange(ctx context.Context, minFrequency, maxFrequency int, pagination Pagination) (*PaginationResult, error)
	GetRecentPatterns(ctx context.Context, limit int) ([]models.Pattern, error)
	GetHighConfidencePatterns(ctx context.Context, threshold float64, pagination Pagination) (*PaginationResult, error)
	GetFrequentPatterns(ctx context.Context, threshold int, pagination Pagination) (*PaginationResult, error)
	GetPatternWithInsights(ctx context.Context, patternID uuid.UUID) (*models.Pattern, error)
	UpdateConfidence(ctx context.Context, patternID uuid.UUID, confidence float64) error
	UpdateFrequency(ctx context.Context, patternID uuid.UUID, frequency int) error
	UpdateSignificance(ctx context.Context, patternID uuid.UUID, significance float64) error
	UpdateLastSeen(ctx context.Context, patternID uuid.UUID) error
	GetPatternStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error)
	GetPatternTrends(ctx context.Context, period string) (map[string]interface{}, error)
	SearchPatterns(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetSimilarPatterns(ctx context.Context, patternID uuid.UUID, threshold float64) ([]models.Pattern, error)
	GetPatternsByContext(ctx context.Context, context map[string]interface{}, pagination Pagination) (*PaginationResult, error)
}

// InsightRepository defines insight-specific repository operations
type InsightRepository interface {
	BaseRepository
	GetByType(ctx context.Context, insightType string, pagination Pagination) (*PaginationResult, error)
	GetByImpact(ctx context.Context, impact string, pagination Pagination) (*PaginationResult, error)
	GetByPattern(ctx context.Context, patternID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetByConfidenceRange(ctx context.Context, minConfidence, maxConfidence float64, pagination Pagination) (*PaginationResult, error)
	GetActionableInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetNonActionableInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetImplementedInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetUnimplementedInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetHighImpactInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetCriticalInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetRecentInsights(ctx context.Context, limit int) ([]models.Insight, error)
	MarkAsImplemented(ctx context.Context, insightID uuid.UUID) error
	MarkAsUnimplemented(ctx context.Context, insightID uuid.UUID) error
	UpdateConfidence(ctx context.Context, insightID uuid.UUID, confidence float64) error
	GetInsightStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error)
	GetInsightTrends(ctx context.Context, period string) (map[string]interface{}, error)
	SearchInsights(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetInsightsByActionItems(ctx context.Context, actionItems []string, pagination Pagination) (*PaginationResult, error)
	GetInsightEffectiveness(ctx context.Context, insightID uuid.UUID) (map[string]interface{}, error)
}

// GitRepositoryRepository defines git repository-specific operations
type GitRepositoryRepository interface {
	BaseRepository
	GetByProjectID(ctx context.Context, projectID uuid.UUID) (*models.GitRepository, error)
	GetByRepositoryURL(ctx context.Context, repositoryURL string) (*models.GitRepository, error)
	GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error)
	UpdateLastSync(ctx context.Context, gitRepoID uuid.UUID, commitSHA string) error
	GetActiveRepositories(ctx context.Context) ([]models.GitRepository, error)
	GetRepositoriesNeedingSync(ctx context.Context, lastSyncBefore time.Time) ([]models.GitRepository, error)
	UpdateStatus(ctx context.Context, gitRepoID uuid.UUID, status string) error
	GetRepositoryStatistics(ctx context.Context, gitRepoID uuid.UUID) (map[string]interface{}, error)
}

// ProjectAssetRepository defines project asset-specific operations
type ProjectAssetRepository interface {
	BaseRepository
	GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]models.ProjectAsset, error)
	GetByProjectIDAndType(ctx context.Context, projectID uuid.UUID, assetType string) ([]models.ProjectAsset, error)
	GetByType(ctx context.Context, assetType string, pagination Pagination) (*PaginationResult, error)
	GetByUploader(ctx context.Context, uploaderID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetPublicAssets(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetPrivateAssets(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetByTags(ctx context.Context, tags []string, pagination Pagination) (*PaginationResult, error)
	GetByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error)
	GetRecentAssets(ctx context.Context, limit int) ([]models.ProjectAsset, error)
	GetLargeAssets(ctx context.Context, minSize int64, pagination Pagination) (*PaginationResult, error)
	UpdateViewCount(ctx context.Context, assetID uuid.UUID) error
	GetAssetStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error)
	SearchAssets(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetAssetsByMimeType(ctx context.Context, mimeType string, pagination Pagination) (*PaginationResult, error)
	GetOrphanedAssets(ctx context.Context) ([]models.ProjectAsset, error)
	CleanupOrphanedAssets(ctx context.Context) error
}

// ProjectStateRepository defines project state-specific operations
type ProjectStateRepository interface {
	BaseRepository
	GetByProjectID(ctx context.Context, projectID uuid.UUID) (*models.ProjectState, error)
	GetByBuildStatus(ctx context.Context, buildStatus string, pagination Pagination) (*PaginationResult, error)
	GetByTestStatus(ctx context.Context, testStatus string, pagination Pagination) (*PaginationResult, error)
	GetByHealthScoreRange(ctx context.Context, minScore, maxScore int, pagination Pagination) (*PaginationResult, error)
	GetHealthyProjects(ctx context.Context, minScore int, pagination Pagination) (*PaginationResult, error)
	GetUnhealthyProjects(ctx context.Context, maxScore int, pagination Pagination) (*PaginationResult, error)
	GetProjectsNeedingCheck(ctx context.Context, lastCheckBefore time.Time) ([]models.ProjectState, error)
	GetProjectsWithFailingBuilds(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetProjectsWithFailingTests(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetProjectsWithSecurityIssues(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetByCoverageRange(ctx context.Context, minCoverage, maxCoverage float64, pagination Pagination) (*PaginationResult, error)
	UpdateHealthScore(ctx context.Context, projectID uuid.UUID, score int) error
	UpdateLastCheck(ctx context.Context, projectID uuid.UUID) error
	GetStateStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error)
	GetHealthTrends(ctx context.Context, period string) (map[string]interface{}, error)
	GetProjectsWithReadme(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetProjectsWithDemo(ctx context.Context, pagination Pagination) (*PaginationResult, error)
}

// WorkflowExecutionRepository defines workflow execution-specific operations
type WorkflowExecutionRepository interface {
	BaseRepository
	GetByProjectID(ctx context.Context, projectID uuid.UUID) ([]models.WorkflowExecution, error)
	GetByWorkflowType(ctx context.Context, workflowType string, pagination Pagination) (*PaginationResult, error)
	GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error)
	GetByTriggerType(ctx context.Context, triggerType string, pagination Pagination) (*PaginationResult, error)
	GetByTriggerer(ctx context.Context, triggererID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetRunningExecutions(ctx context.Context) ([]models.WorkflowExecution, error)
	GetPendingExecutions(ctx context.Context) ([]models.WorkflowExecution, error)
	GetCompletedExecutions(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetFailedExecutions(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error)
	GetRecentExecutions(ctx context.Context, limit int) ([]models.WorkflowExecution, error)
	GetExecutionStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error)
	GetExecutionTrends(ctx context.Context, period string) (map[string]interface{}, error)
	GetLongRunningExecutions(ctx context.Context, duration time.Duration) ([]models.WorkflowExecution, error)
	GetExecutionsByDuration(ctx context.Context, minDuration, maxDuration int, pagination Pagination) (*PaginationResult, error)
	CleanupCompletedExecutions(ctx context.Context, olderThan time.Time) error
	CancelRunningExecutions(ctx context.Context, projectID uuid.UUID) error
}

// ProjectTemplateRepository defines project template-specific operations
type ProjectTemplateRepository interface {
	BaseRepository
	GetByCategory(ctx context.Context, category string, pagination Pagination) (*PaginationResult, error)
	GetByCreator(ctx context.Context, creatorID uuid.UUID, pagination Pagination) (*PaginationResult, error)
	GetPublicTemplates(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetPrivateTemplates(ctx context.Context, pagination Pagination) (*PaginationResult, error)
	GetByProjectType(ctx context.Context, projectType string, pagination Pagination) (*PaginationResult, error)
	GetPopularTemplates(ctx context.Context, limit int) ([]models.ProjectTemplate, error)
	GetRecentTemplates(ctx context.Context, limit int) ([]models.ProjectTemplate, error)
	GetByRatingRange(ctx context.Context, minRating, maxRating float64, pagination Pagination) (*PaginationResult, error)
	IncrementUsageCount(ctx context.Context, templateID uuid.UUID) error
	UpdateRating(ctx context.Context, templateID uuid.UUID, rating float64) error
	SearchTemplates(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error)
	GetTemplateStatistics(ctx context.Context, templateID uuid.UUID) (map[string]interface{}, error)
	GetTemplateUsageTrends(ctx context.Context, period string) (map[string]interface{}, error)
}

// RepositoryManager manages all repositories and provides transaction support
type RepositoryManager interface {
	TransactionManager

	// Repository getters
	User() UserRepository
	Project() ProjectRepository
	Task() TaskRepository
	Proposal() ProposalRepository
	Agent() AgentRepository
	ActivityLog() ActivityLogRepository
	Pattern() PatternRepository
	Insight() InsightRepository
	GitRepository() GitRepositoryRepository
	ProjectAsset() ProjectAssetRepository
	ProjectState() ProjectStateRepository
	WorkflowExecution() WorkflowExecutionRepository
	ProjectTemplate() ProjectTemplateRepository

	// Health and stats
	Health() error
	GetStats() map[string]interface{}

	// Migration and maintenance
	RunMigrations() error
	CleanupOldData(ctx context.Context, config map[string]interface{}) error

	// Backup and restore
	CreateBackup(ctx context.Context, path string) error
	RestoreBackup(ctx context.Context, path string) error

	// Cache operations
	InvalidateCache(ctx context.Context, patterns ...string) error
	ClearAllCache(ctx context.Context) error

	// Batch operations
	BatchExecute(ctx context.Context, operations []func(ctx context.Context) error) error

	// Database connection
	GetDB() *gorm.DB
	Close() error
}
