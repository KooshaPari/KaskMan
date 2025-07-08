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

// TaskRepositoryImpl implements the TaskRepository interface
type TaskRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewTaskRepository creates a new task repository instance
func NewTaskRepository(db *gorm.DB, logger *logrus.Logger, cache CacheManager) TaskRepository {
	return &TaskRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db, logger, cache),
	}
}

// GetByProject retrieves tasks by project ID with pagination
func (r *TaskRepositoryImpl) GetByProject(ctx context.Context, projectID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var tasks []models.Task
	filters := Filter{"project_id": projectID}
	return r.ListWithPagination(ctx, &tasks, pagination, filters)
}

// GetByAssignee retrieves tasks by assignee ID with pagination
func (r *TaskRepositoryImpl) GetByAssignee(ctx context.Context, assigneeID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var tasks []models.Task
	filters := Filter{"assigned_to": assigneeID}
	return r.ListWithPagination(ctx, &tasks, pagination, filters)
}

// GetByAgent retrieves tasks by agent ID with pagination
func (r *TaskRepositoryImpl) GetByAgent(ctx context.Context, agentID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var tasks []models.Task
	filters := Filter{"agent_id": agentID}
	return r.ListWithPagination(ctx, &tasks, pagination, filters)
}

// GetByStatus retrieves tasks by status with pagination
func (r *TaskRepositoryImpl) GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error) {
	var tasks []models.Task
	filters := Filter{"status": status}
	return r.ListWithPagination(ctx, &tasks, pagination, filters)
}

// GetByPriority retrieves tasks by priority with pagination
func (r *TaskRepositoryImpl) GetByPriority(ctx context.Context, priority string, pagination Pagination) (*PaginationResult, error) {
	var tasks []models.Task
	filters := Filter{"priority": priority}
	return r.ListWithPagination(ctx, &tasks, pagination, filters)
}

// GetByType retrieves tasks by type with pagination
func (r *TaskRepositoryImpl) GetByType(ctx context.Context, taskType string, pagination Pagination) (*PaginationResult, error) {
	var tasks []models.Task
	filters := Filter{"type": taskType}
	return r.ListWithPagination(ctx, &tasks, pagination, filters)
}

// GetPendingTasks retrieves pending tasks with pagination
func (r *TaskRepositoryImpl) GetPendingTasks(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "pending", pagination)
}

// GetInProgressTasks retrieves in-progress tasks with pagination
func (r *TaskRepositoryImpl) GetInProgressTasks(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "in_progress", pagination)
}

// GetCompletedTasks retrieves completed tasks with pagination
func (r *TaskRepositoryImpl) GetCompletedTasks(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "completed", pagination)
}

// GetOverdueTasks retrieves overdue tasks
func (r *TaskRepositoryImpl) GetOverdueTasks(ctx context.Context) ([]models.Task, error) {
	db := r.getDB(ctx)
	var tasks []models.Task

	now := time.Now()
	err := db.Where("status NOT IN ? AND created_at < ?", []string{"completed", "cancelled"}, now.AddDate(0, 0, -7)).
		Preload("Project").
		Preload("AssignedUser").
		Find(&tasks).Error

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get overdue tasks")
		return nil, fmt.Errorf("failed to get overdue tasks: %w", err)
	}

	return tasks, nil
}

// GetTasksByDateRange retrieves tasks within a date range with pagination
func (r *TaskRepositoryImpl) GetTasksByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var tasks []models.Task

	query := r.buildDateRangeQuery(db, "created_at", &startDate, &endDate)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Task{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("AssignedUser").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks by date range: %w", err)
	}

	return &PaginationResult{
		Data:       tasks,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetTasksCompletedToday retrieves tasks completed today
func (r *TaskRepositoryImpl) GetTasksCompletedToday(ctx context.Context) ([]models.Task, error) {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)
	return r.GetTasksCompletedInPeriod(ctx, today, tomorrow)
}

// GetTasksCompletedInPeriod retrieves tasks completed within a period
func (r *TaskRepositoryImpl) GetTasksCompletedInPeriod(ctx context.Context, startDate, endDate time.Time) ([]models.Task, error) {
	db := r.getDB(ctx)
	var tasks []models.Task

	err := db.Where("status = ? AND completed_at BETWEEN ? AND ?", "completed", startDate, endDate).
		Preload("Project").
		Preload("AssignedUser").
		Find(&tasks).Error

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get completed tasks in period")
		return nil, fmt.Errorf("failed to get completed tasks in period: %w", err)
	}

	return tasks, nil
}

// UpdateStatus updates a task's status
func (r *TaskRepositoryImpl) UpdateStatus(ctx context.Context, taskID uuid.UUID, status string) error {
	db := r.getDB(ctx)

	updates := map[string]interface{}{
		"status": status,
	}

	// Set completion time if marking as completed
	if status == "completed" {
		now := time.Now()
		updates["completed_at"] = &now
	}

	err := db.Model(&models.Task{}).Where("id = ?", taskID).Updates(updates).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update task status")
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// UpdateProgress updates a task's progress
func (r *TaskRepositoryImpl) UpdateProgress(ctx context.Context, taskID uuid.UUID, progress int) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Task{}).Where("id = ?", taskID).Update("progress", progress).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update task progress")
		return fmt.Errorf("failed to update task progress: %w", err)
	}

	return nil
}

// AssignTask assigns a task to a user and/or agent
func (r *TaskRepositoryImpl) AssignTask(ctx context.Context, taskID uuid.UUID, assigneeID *uuid.UUID, agentID *uuid.UUID) error {
	db := r.getDB(ctx)

	updates := map[string]interface{}{}
	if assigneeID != nil {
		updates["assigned_to"] = *assigneeID
	}
	if agentID != nil {
		updates["agent_id"] = *agentID
	}

	if len(updates) == 0 {
		return fmt.Errorf("no assignment provided")
	}

	err := db.Model(&models.Task{}).Where("id = ?", taskID).Updates(updates).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to assign task")
		return fmt.Errorf("failed to assign task: %w", err)
	}

	return nil
}

// UnassignTask removes assignments from a task
func (r *TaskRepositoryImpl) UnassignTask(ctx context.Context, taskID uuid.UUID) error {
	db := r.getDB(ctx)

	updates := map[string]interface{}{
		"assigned_to": nil,
		"agent_id":    nil,
	}

	err := db.Model(&models.Task{}).Where("id = ?", taskID).Updates(updates).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to unassign task")
		return fmt.Errorf("failed to unassign task: %w", err)
	}

	return nil
}

// GetTaskStatistics retrieves task statistics with optional filters
func (r *TaskRepositoryImpl) GetTaskStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	// Build base query with filters
	query := db.Model(&models.Task{})
	for key, value := range filters {
		if value != nil {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	stats := make(map[string]interface{})

	// Total tasks
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total tasks: %w", err)
	}
	stats["total"] = total

	// Tasks by status
	var statusCounts []struct {
		Status string
		Count  int64
	}
	if err := query.Select("status, COUNT(*) as count").Group("status").Scan(&statusCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get status counts: %w", err)
	}

	statusStats := make(map[string]int64)
	for _, sc := range statusCounts {
		statusStats[sc.Status] = sc.Count
	}
	stats["by_status"] = statusStats

	// Tasks by priority
	var priorityCounts []struct {
		Priority string
		Count    int64
	}
	if err := query.Select("priority, COUNT(*) as count").Group("priority").Scan(&priorityCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get priority counts: %w", err)
	}

	priorityStats := make(map[string]int64)
	for _, pc := range priorityCounts {
		priorityStats[pc.Priority] = pc.Count
	}
	stats["by_priority"] = priorityStats

	return stats, nil
}

// GetTasksByEstimatedTime retrieves tasks by estimated time range with pagination
func (r *TaskRepositoryImpl) GetTasksByEstimatedTime(ctx context.Context, minTime, maxTime int, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var tasks []models.Task

	query := r.buildNumericRangeQuery(db, "estimated_time", minTime, maxTime)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Task{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("estimated_time ASC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("AssignedUser").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks by estimated time: %w", err)
	}

	return &PaginationResult{
		Data:       tasks,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// SearchTasks searches tasks by query string with pagination
func (r *TaskRepositoryImpl) SearchTasks(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var tasks []models.Task

	searchQuery := r.buildSearchQuery(db, query, []string{"title", "description", "type"})

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := searchQuery.Model(&models.Task{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := searchQuery.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("AssignedUser").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to search tasks: %w", err)
	}

	return &PaginationResult{
		Data:       tasks,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetTasksWithRelations retrieves tasks with all relations preloaded
func (r *TaskRepositoryImpl) GetTasksWithRelations(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var tasks []models.Task

	relations := []string{"Project", "AssignedUser", "Agent"}
	query := r.preloadRelations(db, relations)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Task{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("failed to get tasks with relations: %w", err)
	}

	return &PaginationResult{
		Data:       tasks,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}
