package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// AgentRepositoryImpl implements the AgentRepository interface
type AgentRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewAgentRepository creates a new agent repository instance
func NewAgentRepository(db *gorm.DB, logger *logrus.Logger, cache CacheManager) AgentRepository {
	return &AgentRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db, logger, cache),
	}
}

// GetByType retrieves agents by type with pagination
func (r *AgentRepositoryImpl) GetByType(ctx context.Context, agentType string, pagination Pagination) (*PaginationResult, error) {
	var agents []models.Agent
	filters := Filter{"type": agentType}
	return r.ListWithPagination(ctx, &agents, pagination, filters)
}

// GetByStatus retrieves agents by status with pagination
func (r *AgentRepositoryImpl) GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error) {
	var agents []models.Agent
	filters := Filter{"status": status}
	return r.ListWithPagination(ctx, &agents, pagination, filters)
}

// GetActiveAgents retrieves active agents with pagination
func (r *AgentRepositoryImpl) GetActiveAgents(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "active", pagination)
}

// GetInactiveAgents retrieves inactive agents with pagination
func (r *AgentRepositoryImpl) GetInactiveAgents(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "inactive", pagination)
}

// GetBusyAgents retrieves busy agents with pagination
func (r *AgentRepositoryImpl) GetBusyAgents(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "busy", pagination)
}

// GetAvailableAgents retrieves available agents of a specific type
func (r *AgentRepositoryImpl) GetAvailableAgents(ctx context.Context, agentType string) ([]models.Agent, error) {
	db := r.getDB(ctx)
	var agents []models.Agent

	query := db.Where("status = ?", "active")
	if agentType != "" {
		query = query.Where("type = ?", agentType)
	}

	err := query.Preload("Tasks").Find(&agents).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get available agents")
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	return agents, nil
}

// GetAgentWithTasks retrieves an agent with its tasks
func (r *AgentRepositoryImpl) GetAgentWithTasks(ctx context.Context, agentID uuid.UUID) (*models.Agent, error) {
	db := r.getDB(ctx)
	var agent models.Agent

	err := db.Preload("Tasks").First(&agent, "id = ?", agentID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("agent with id %s not found", agentID.String())
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get agent with tasks")
		return nil, fmt.Errorf("failed to get agent with tasks: %w", err)
	}

	return &agent, nil
}

// UpdateStatus updates an agent's status
func (r *AgentRepositoryImpl) UpdateStatus(ctx context.Context, agentID uuid.UUID, status string) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Agent{}).Where("id = ?", agentID).Update("status", status).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update agent status")
		return fmt.Errorf("failed to update agent status: %w", err)
	}

	return nil
}

// UpdateLastActive updates an agent's last active timestamp
func (r *AgentRepositoryImpl) UpdateLastActive(ctx context.Context, agentID uuid.UUID) error {
	db := r.getDB(ctx)

	now := time.Now()
	err := db.Model(&models.Agent{}).Where("id = ?", agentID).Update("last_active", &now).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update agent last active")
		return fmt.Errorf("failed to update agent last active: %w", err)
	}

	return nil
}

// UpdateTaskCount updates an agent's task count
func (r *AgentRepositoryImpl) UpdateTaskCount(ctx context.Context, agentID uuid.UUID, count int) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Agent{}).Where("id = ?", agentID).Update("task_count", count).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update agent task count")
		return fmt.Errorf("failed to update agent task count: %w", err)
	}

	return nil
}

// UpdateSuccessRate updates an agent's success rate
func (r *AgentRepositoryImpl) UpdateSuccessRate(ctx context.Context, agentID uuid.UUID, rate float64) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Agent{}).Where("id = ?", agentID).Update("success_rate", rate).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update agent success rate")
		return fmt.Errorf("failed to update agent success rate: %w", err)
	}

	return nil
}

// UpdateResponseTime updates an agent's average response time
func (r *AgentRepositoryImpl) UpdateResponseTime(ctx context.Context, agentID uuid.UUID, responseTime float64) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Agent{}).Where("id = ?", agentID).Update("avg_response_time", responseTime).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update agent response time")
		return fmt.Errorf("failed to update agent response time: %w", err)
	}

	return nil
}

// GetAgentStatistics retrieves agent statistics
func (r *AgentRepositoryImpl) GetAgentStatistics(ctx context.Context, agentID uuid.UUID) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	var agent models.Agent
	err := db.First(&agent, "id = ?", agentID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	stats := map[string]interface{}{
		"id":                agent.ID,
		"name":              agent.Name,
		"type":              agent.Type,
		"status":            agent.Status,
		"task_count":        agent.TaskCount,
		"success_rate":      agent.SuccessRate,
		"avg_response_time": agent.AvgResponseTime,
		"created_at":        agent.CreatedAt,
	}

	if agent.LastActive != nil {
		stats["last_active"] = *agent.LastActive
		stats["idle_time"] = time.Since(*agent.LastActive).String()
	}

	// Get task statistics for this agent
	var taskStats struct {
		TotalTasks      int64 `json:"total_tasks"`
		PendingTasks    int64 `json:"pending_tasks"`
		InProgressTasks int64 `json:"in_progress_tasks"`
		CompletedTasks  int64 `json:"completed_tasks"`
		FailedTasks     int64 `json:"failed_tasks"`
	}

	db.Model(&models.Task{}).Where("agent_id = ?", agentID).Count(&taskStats.TotalTasks)
	db.Model(&models.Task{}).Where("agent_id = ? AND status = ?", agentID, "pending").Count(&taskStats.PendingTasks)
	db.Model(&models.Task{}).Where("agent_id = ? AND status = ?", agentID, "in_progress").Count(&taskStats.InProgressTasks)
	db.Model(&models.Task{}).Where("agent_id = ? AND status = ?", agentID, "completed").Count(&taskStats.CompletedTasks)
	db.Model(&models.Task{}).Where("agent_id = ? AND status = ?", agentID, "failed").Count(&taskStats.FailedTasks)

	stats["task_statistics"] = taskStats

	return stats, nil
}

// GetAgentsByCapabilities retrieves agents by capabilities with pagination
func (r *AgentRepositoryImpl) GetAgentsByCapabilities(ctx context.Context, capabilities []string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var agents []models.Agent

	// For this implementation, we'll do a simple text search in the capabilities JSON field
	// In a real implementation, you'd want to use proper JSON queries
	query := db
	for _, capability := range capabilities {
		query = query.Where("capabilities LIKE ?", "%"+capability+"%")
	}

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Agent{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count agents: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to get agents by capabilities: %w", err)
	}

	return &PaginationResult{
		Data:       agents,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetAgentsByLastActive retrieves agents by last active time with pagination
func (r *AgentRepositoryImpl) GetAgentsByLastActive(ctx context.Context, since time.Time, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var agents []models.Agent

	query := db.Where("last_active >= ?", since)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Agent{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count agents: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("last_active DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to get agents by last active: %w", err)
	}

	return &PaginationResult{
		Data:       agents,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetTopPerformingAgents retrieves top performing agents by success rate
func (r *AgentRepositoryImpl) GetTopPerformingAgents(ctx context.Context, limit int) ([]models.Agent, error) {
	db := r.getDB(ctx)
	var agents []models.Agent

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	err := db.Where("status = ?", "active").
		Order("success_rate DESC, task_count DESC").
		Limit(limit).
		Find(&agents).Error

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get top performing agents")
		return nil, fmt.Errorf("failed to get top performing agents: %w", err)
	}

	return agents, nil
}

// SearchAgents searches agents by query string with pagination
func (r *AgentRepositoryImpl) SearchAgents(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var agents []models.Agent

	searchQuery := r.buildSearchQuery(db, query, []string{"name", "type", "capabilities"})

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := searchQuery.Model(&models.Agent{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count agents: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := searchQuery.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return &PaginationResult{
		Data:       agents,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetAgentWorkload retrieves workload information for an agent
func (r *AgentRepositoryImpl) GetAgentWorkload(ctx context.Context, agentID uuid.UUID) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	workload := make(map[string]interface{})

	// Get current task counts by status
	var taskCounts []struct {
		Status string
		Count  int64
	}

	err := db.Model(&models.Task{}).
		Where("agent_id = ?", agentID).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&taskCounts).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get agent task counts: %w", err)
	}

	statusCounts := make(map[string]int64)
	var totalTasks int64
	for _, tc := range taskCounts {
		statusCounts[tc.Status] = tc.Count
		totalTasks += tc.Count
	}

	workload["task_counts"] = statusCounts
	workload["total_tasks"] = totalTasks

	// Calculate workload metrics
	workload["current_load"] = statusCounts["in_progress"] + statusCounts["assigned"]
	workload["pending_load"] = statusCounts["pending"] + statusCounts["queued"]

	// Get average task completion time for this agent
	var avgCompletionTime struct {
		AvgMinutes float64
	}

	err = db.Model(&models.Task{}).
		Where("agent_id = ? AND status = ? AND completed_at IS NOT NULL AND started_at IS NOT NULL", agentID, "completed").
		Select("AVG(EXTRACT(EPOCH FROM (completed_at - started_at))/60) as avg_minutes").
		Scan(&avgCompletionTime).Error

	if err == nil {
		workload["avg_completion_time_minutes"] = avgCompletionTime.AvgMinutes
	}

	// Get estimated remaining work time
	var estimatedWork struct {
		TotalMinutes int64
	}

	err = db.Model(&models.Task{}).
		Where("agent_id = ? AND status IN ?", agentID, []string{"pending", "in_progress", "assigned"}).
		Select("SUM(estimated_time) as total_minutes").
		Scan(&estimatedWork).Error

	if err == nil {
		workload["estimated_remaining_minutes"] = estimatedWork.TotalMinutes
	}

	return workload, nil
}
