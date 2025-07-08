package activity

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// LogLevel represents the severity level of an activity log entry
type LogLevel string

const (
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelDebug   LogLevel = "debug"
)

// ActivityType represents the type of activity being logged
type ActivityType string

const (
	// Authentication activities
	ActivityTypeLogin    ActivityType = "login"
	ActivityTypeLogout   ActivityType = "logout"
	ActivityTypeRegister ActivityType = "register"

	// CRUD operations
	ActivityTypeCreate ActivityType = "create"
	ActivityTypeRead   ActivityType = "read"
	ActivityTypeUpdate ActivityType = "update"
	ActivityTypeDelete ActivityType = "delete"

	// Project activities
	ActivityTypeProjectCreate ActivityType = "project_create"
	ActivityTypeProjectUpdate ActivityType = "project_update"
	ActivityTypeProjectDelete ActivityType = "project_delete"
	ActivityTypeProjectView   ActivityType = "project_view"

	// Task activities
	ActivityTypeTaskCreate   ActivityType = "task_create"
	ActivityTypeTaskUpdate   ActivityType = "task_update"
	ActivityTypeTaskDelete   ActivityType = "task_delete"
	ActivityTypeTaskAssign   ActivityType = "task_assign"
	ActivityTypeTaskComplete ActivityType = "task_complete"

	// Proposal activities
	ActivityTypeProposalCreate  ActivityType = "proposal_create"
	ActivityTypeProposalUpdate  ActivityType = "proposal_update"
	ActivityTypeProposalDelete  ActivityType = "proposal_delete"
	ActivityTypeProposalApprove ActivityType = "proposal_approve"
	ActivityTypeProposalReject  ActivityType = "proposal_reject"

	// Agent activities
	ActivityTypeAgentCreate ActivityType = "agent_create"
	ActivityTypeAgentUpdate ActivityType = "agent_update"
	ActivityTypeAgentDelete ActivityType = "agent_delete"

	// System activities
	ActivityTypeSystemStart    ActivityType = "system_start"
	ActivityTypeSystemShutdown ActivityType = "system_shutdown"
	ActivityTypeSystemError    ActivityType = "system_error"
	ActivityTypeSystemBackup   ActivityType = "system_backup"
	ActivityTypeSystemRestore  ActivityType = "system_restore"

	// API activities
	ActivityTypeAPIRequest  ActivityType = "api_request"
	ActivityTypeAPIResponse ActivityType = "api_response"
	ActivityTypeAPIError    ActivityType = "api_error"
)

// ResourceType represents the type of resource being acted upon
type ResourceType string

const (
	ResourceTypeUser     ResourceType = "user"
	ResourceTypeProject  ResourceType = "project"
	ResourceTypeTask     ResourceType = "task"
	ResourceTypeProposal ResourceType = "proposal"
	ResourceTypeAgent    ResourceType = "agent"
	ResourceTypeSystem   ResourceType = "system"
	ResourceTypeAPI      ResourceType = "api"
)

// ActivityDetails contains additional information about the activity
type ActivityDetails struct {
	UserAgent    string                 `json:"user_agent,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	Method       string                 `json:"method,omitempty"`
	Path         string                 `json:"path,omitempty"`
	StatusCode   int                    `json:"status_code,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	RequestSize  int64                  `json:"request_size,omitempty"`
	ResponseSize int64                  `json:"response_size,omitempty"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	OldValues    map[string]interface{} `json:"old_values,omitempty"`
	NewValues    map[string]interface{} `json:"new_values,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ActivityLogEntry represents a structured activity log entry
type ActivityLogEntry struct {
	UserID     *uuid.UUID      `json:"user_id,omitempty"`
	Username   string          `json:"username,omitempty"`
	Action     ActivityType    `json:"action"`
	Resource   ResourceType    `json:"resource"`
	ResourceID *uuid.UUID      `json:"resource_id,omitempty"`
	Details    ActivityDetails `json:"details"`
	Success    bool            `json:"success"`
	Error      string          `json:"error,omitempty"`
	Level      LogLevel        `json:"level"`
	Timestamp  time.Time       `json:"timestamp"`
}

// Service provides activity logging functionality
type Service struct {
	repo   repositories.ActivityLogRepository
	logger *logrus.Logger
}

// NewService creates a new activity logging service
func NewService(repo repositories.ActivityLogRepository, logger *logrus.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// LogActivity logs an activity with detailed information
func (s *Service) LogActivity(ctx context.Context, entry ActivityLogEntry) error {
	// Create details map
	details := map[string]interface{}{
		"user_agent":    entry.Details.UserAgent,
		"request_id":    entry.Details.RequestID,
		"method":        entry.Details.Method,
		"path":          entry.Details.Path,
		"status_code":   entry.Details.StatusCode,
		"duration_ms":   entry.Details.Duration.Milliseconds(),
		"request_size":  entry.Details.RequestSize,
		"response_size": entry.Details.ResponseSize,
		"changes":       entry.Details.Changes,
		"old_values":    entry.Details.OldValues,
		"new_values":    entry.Details.NewValues,
		"metadata":      entry.Details.Metadata,
		"level":         entry.Level,
		"timestamp":     entry.Timestamp,
	}

	// Log the activity
	err := s.repo.LogActivity(
		ctx,
		entry.UserID,
		string(entry.Action),
		string(entry.Resource),
		entry.ResourceID,
		details,
		entry.Success,
		entry.Error,
	)

	if err != nil {
		s.logger.WithError(err).Error("Failed to log activity")
		return err
	}

	// Also log to structured logger for debugging
	logEntry := s.logger.WithFields(logrus.Fields{
		"user_id":     entry.UserID,
		"username":    entry.Username,
		"action":      entry.Action,
		"resource":    entry.Resource,
		"resource_id": entry.ResourceID,
		"success":     entry.Success,
		"ip_address":  entry.Details.IPAddress,
		"user_agent":  entry.Details.UserAgent,
		"method":      entry.Details.Method,
		"path":        entry.Details.Path,
		"status_code": entry.Details.StatusCode,
		"duration":    entry.Details.Duration,
	})

	switch entry.Level {
	case LogLevelError:
		logEntry.Error("Activity logged")
	case LogLevelWarning:
		logEntry.Warn("Activity logged")
	case LogLevelDebug:
		logEntry.Debug("Activity logged")
	default:
		logEntry.Info("Activity logged")
	}

	return nil
}

// LogAuthActivity logs authentication-related activities
func (s *Service) LogAuthActivity(ctx context.Context, userID *uuid.UUID, username string, action ActivityType, success bool, ipAddress, userAgent, errorMsg string) error {
	entry := ActivityLogEntry{
		UserID:   userID,
		Username: username,
		Action:   action,
		Resource: ResourceTypeUser,
		Details: ActivityDetails{
			IPAddress: ipAddress,
			UserAgent: userAgent,
		},
		Success:   success,
		Error:     errorMsg,
		Level:     LogLevelInfo,
		Timestamp: time.Now(),
	}

	if !success {
		entry.Level = LogLevelWarning
	}

	return s.LogActivity(ctx, entry)
}

// LogCRUDActivity logs Create, Read, Update, Delete activities
func (s *Service) LogCRUDActivity(ctx context.Context, userID *uuid.UUID, username string, action ActivityType, resource ResourceType, resourceID *uuid.UUID, changes map[string]interface{}, success bool, ipAddress, userAgent, errorMsg string) error {
	entry := ActivityLogEntry{
		UserID:     userID,
		Username:   username,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details: ActivityDetails{
			IPAddress: ipAddress,
			UserAgent: userAgent,
			Changes:   changes,
		},
		Success:   success,
		Error:     errorMsg,
		Level:     LogLevelInfo,
		Timestamp: time.Now(),
	}

	if !success {
		entry.Level = LogLevelError
	}

	return s.LogActivity(ctx, entry)
}

// LogAPIActivity logs API request activities
func (s *Service) LogAPIActivity(ctx context.Context, userID *uuid.UUID, username string, method, path, ipAddress, userAgent string, statusCode int, duration time.Duration, requestSize, responseSize int64, errorMsg string) error {
	success := statusCode < 400

	entry := ActivityLogEntry{
		UserID:   userID,
		Username: username,
		Action:   ActivityTypeAPIRequest,
		Resource: ResourceTypeAPI,
		Details: ActivityDetails{
			IPAddress:    ipAddress,
			UserAgent:    userAgent,
			Method:       method,
			Path:         path,
			StatusCode:   statusCode,
			Duration:     duration,
			RequestSize:  requestSize,
			ResponseSize: responseSize,
		},
		Success:   success,
		Error:     errorMsg,
		Level:     LogLevelInfo,
		Timestamp: time.Now(),
	}

	if !success {
		entry.Level = LogLevelWarning
	}

	return s.LogActivity(ctx, entry)
}

// LogSystemActivity logs system-level activities
func (s *Service) LogSystemActivity(ctx context.Context, action ActivityType, details map[string]interface{}, success bool, errorMsg string) error {
	entry := ActivityLogEntry{
		Action:   action,
		Resource: ResourceTypeSystem,
		Details: ActivityDetails{
			Metadata: details,
		},
		Success:   success,
		Error:     errorMsg,
		Level:     LogLevelInfo,
		Timestamp: time.Now(),
	}

	if !success {
		entry.Level = LogLevelError
	}

	return s.LogActivity(ctx, entry)
}

// GetActivities retrieves activities with filtering and pagination
func (s *Service) GetActivities(ctx context.Context, filters repositories.Filter, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	return s.repo.ListWithPagination(ctx, &[]models.ActivityLog{}, pagination, filters)
}

// GetRecentActivities retrieves the most recent activities
func (s *Service) GetRecentActivities(ctx context.Context, limit int) ([]models.ActivityLog, error) {
	return s.repo.GetRecentActivities(ctx, limit)
}

// GetUserActivities retrieves activities for a specific user
func (s *Service) GetUserActivities(ctx context.Context, userID uuid.UUID, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	return s.repo.GetByUser(ctx, userID, pagination)
}

// GetActivitiesByResource retrieves activities for a specific resource
func (s *Service) GetActivitiesByResource(ctx context.Context, resource string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	return s.repo.GetByResource(ctx, resource, pagination)
}

// GetActivitiesByAction retrieves activities for a specific action
func (s *Service) GetActivitiesByAction(ctx context.Context, action string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	return s.repo.GetByAction(ctx, action, pagination)
}

// GetActivitiesByDateRange retrieves activities within a date range
func (s *Service) GetActivitiesByDateRange(ctx context.Context, startDate, endDate time.Time, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	return s.repo.GetByDateRange(ctx, startDate, endDate, pagination)
}

// GetUserActivityStats retrieves activity statistics for a user
func (s *Service) GetUserActivityStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	return s.repo.GetUserActivityStats(ctx, userID)
}

// GetSystemActivityStats retrieves system-wide activity statistics
func (s *Service) GetSystemActivityStats(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetSystemActivityStats(ctx)
}

// GetActivityTrends retrieves activity trends over time
func (s *Service) GetActivityTrends(ctx context.Context, period string) (map[string]interface{}, error) {
	return s.repo.GetActivityTrends(ctx, period)
}

// ExtractDetailsFromContext extracts activity details from Gin context
func (s *Service) ExtractDetailsFromContext(c *gin.Context) ActivityDetails {
	details := ActivityDetails{
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
	}

	// Extract request ID if available
	if requestID, exists := c.Get("request_id"); exists {
		details.RequestID = requestID.(string)
	}

	// Extract content length
	if c.Request.ContentLength > 0 {
		details.RequestSize = c.Request.ContentLength
	}

	return details
}

// ExtractUserInfoFromContext extracts user information from Gin context
func (s *Service) ExtractUserInfoFromContext(c *gin.Context) (*uuid.UUID, string) {
	var userID *uuid.UUID
	var username string

	if userIDStr, exists := c.Get("user_id"); exists {
		if parsedID, err := uuid.Parse(userIDStr.(string)); err == nil {
			userID = &parsedID
		}
	}

	if usernameStr, exists := c.Get("username"); exists {
		username = usernameStr.(string)
	}

	return userID, username
}

// LogHTTPRequest logs HTTP request details
func (s *Service) LogHTTPRequest(ctx context.Context, c *gin.Context, statusCode int, duration time.Duration, responseSize int64, err error) {
	userID, username := s.ExtractUserInfoFromContext(c)
	details := s.ExtractDetailsFromContext(c)

	details.StatusCode = statusCode
	details.Duration = duration
	details.ResponseSize = responseSize

	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	logEntry := ActivityLogEntry{
		UserID:    userID,
		Username:  username,
		Action:    ActivityTypeAPIRequest,
		Resource:  ResourceTypeAPI,
		Details:   details,
		Success:   statusCode < 400,
		Error:     errorMsg,
		Level:     LogLevelInfo,
		Timestamp: time.Now(),
	}

	if statusCode >= 400 {
		logEntry.Level = LogLevelWarning
	}
	if statusCode >= 500 {
		logEntry.Level = LogLevelError
	}

	if logErr := s.LogActivity(ctx, logEntry); logErr != nil {
		s.logger.WithError(logErr).Error("Failed to log HTTP request activity")
	}
}

// ShouldLogRequest determines if a request should be logged based on path and method
func (s *Service) ShouldLogRequest(c *gin.Context) bool {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Skip logging for health checks and static assets
	skipPaths := []string{
		"/health",
		"/metrics",
		"/favicon.ico",
		"/static/",
		"/css/",
		"/js/",
		"/img/",
		"/assets/",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return false
		}
	}

	// Skip logging for OPTIONS requests (CORS preflight)
	if method == "OPTIONS" {
		return false
	}

	return true
}

// CleanupOldActivities removes activities older than the specified duration
func (s *Service) CleanupOldActivities(ctx context.Context, olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	return s.repo.CleanupOldActivities(ctx, cutoffTime)
}

// SearchActivities searches activities by query string
func (s *Service) SearchActivities(ctx context.Context, query string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	return s.repo.SearchActivities(ctx, query, pagination)
}
