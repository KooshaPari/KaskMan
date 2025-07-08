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

// ProjectRepositoryImpl implements the ProjectRepository interface
type ProjectRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewProjectRepository creates a new project repository instance
func NewProjectRepository(db *gorm.DB, logger *logrus.Logger, cache CacheManager) ProjectRepository {
	return &ProjectRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db, logger, cache),
	}
}

// GetByCreator retrieves projects created by a specific user
func (r *ProjectRepositoryImpl) GetByCreator(ctx context.Context, creatorID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var projects []models.Project
	filters := Filter{"created_by": creatorID}

	result, err := r.ListWithPagination(ctx, &projects, pagination, filters)
	if err != nil {
		return nil, err
	}

	// Preload creator and tasks for each project
	db := r.getDB(ctx)
	for i := range projects {
		db.Preload("Creator").Preload("Tasks").First(&projects[i], projects[i].ID)
	}

	result.Data = projects
	return result, nil
}

// GetByStatus retrieves projects by status
func (r *ProjectRepositoryImpl) GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error) {
	var projects []models.Project
	filters := Filter{"status": status}

	return r.ListWithPagination(ctx, &projects, pagination, filters)
}

// GetByType retrieves projects by type
func (r *ProjectRepositoryImpl) GetByType(ctx context.Context, projectType string, pagination Pagination) (*PaginationResult, error) {
	var projects []models.Project
	filters := Filter{"type": projectType}

	return r.ListWithPagination(ctx, &projects, pagination, filters)
}

// GetByPriority retrieves projects by priority
func (r *ProjectRepositoryImpl) GetByPriority(ctx context.Context, priority string, pagination Pagination) (*PaginationResult, error) {
	var projects []models.Project
	filters := Filter{"priority": priority}

	return r.ListWithPagination(ctx, &projects, pagination, filters)
}

// GetWithTasks retrieves a project with its associated tasks
func (r *ProjectRepositoryImpl) GetWithTasks(ctx context.Context, projectID uuid.UUID) (*models.Project, error) {
	db := r.getDB(ctx)
	var project models.Project

	if err := db.Preload("Tasks").Preload("Creator").First(&project, "id = ?", projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("project with id %s not found", projectID.String())
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get project with tasks")
		return nil, fmt.Errorf("failed to get project with tasks: %w", err)
	}

	return &project, nil
}

// GetWithProposals retrieves a project with its associated proposals
func (r *ProjectRepositoryImpl) GetWithProposals(ctx context.Context, projectID uuid.UUID) (*models.Project, error) {
	db := r.getDB(ctx)
	var project models.Project

	if err := db.Preload("Proposals").Preload("Creator").First(&project, "id = ?", projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("project with id %s not found", projectID.String())
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get project with proposals")
		return nil, fmt.Errorf("failed to get project with proposals: %w", err)
	}

	return &project, nil
}

// GetWithPatterns retrieves a project with its associated patterns
func (r *ProjectRepositoryImpl) GetWithPatterns(ctx context.Context, projectID uuid.UUID) (*models.Project, error) {
	db := r.getDB(ctx)
	var project models.Project

	if err := db.Preload("Patterns").Preload("Creator").First(&project, "id = ?", projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("project with id %s not found", projectID.String())
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get project with patterns")
		return nil, fmt.Errorf("failed to get project with patterns: %w", err)
	}

	return &project, nil
}

// GetProjectStatistics retrieves statistics for a specific project
func (r *ProjectRepositoryImpl) GetProjectStatistics(ctx context.Context, projectID uuid.UUID) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	// Get task counts by status
	var totalTasks, pendingTasks, inProgressTasks, completedTasks int64
	db.Model(&models.Task{}).Where("project_id = ?", projectID).Count(&totalTasks)
	db.Model(&models.Task{}).Where("project_id = ? AND status = ?", projectID, "pending").Count(&pendingTasks)
	db.Model(&models.Task{}).Where("project_id = ? AND status = ?", projectID, "in_progress").Count(&inProgressTasks)
	db.Model(&models.Task{}).Where("project_id = ? AND status = ?", projectID, "completed").Count(&completedTasks)

	// Get proposal counts by status
	var totalProposals, pendingProposals, approvedProposals, rejectedProposals int64
	db.Model(&models.Proposal{}).Where("project_id = ?", projectID).Count(&totalProposals)
	db.Model(&models.Proposal{}).Where("project_id = ? AND status = ?", projectID, "pending").Count(&pendingProposals)
	db.Model(&models.Proposal{}).Where("project_id = ? AND status = ?", projectID, "approved").Count(&approvedProposals)
	db.Model(&models.Proposal{}).Where("project_id = ? AND status = ?", projectID, "rejected").Count(&rejectedProposals)

	// Get pattern count
	var patternCount int64
	db.Model(&models.Pattern{}).Where("project_id = ?", projectID).Count(&patternCount)

	// Calculate completion rate
	completionRate := 0.0
	if totalTasks > 0 {
		completionRate = float64(completedTasks) / float64(totalTasks) * 100
	}

	stats := map[string]interface{}{
		"total_tasks":        totalTasks,
		"pending_tasks":      pendingTasks,
		"in_progress_tasks":  inProgressTasks,
		"completed_tasks":    completedTasks,
		"total_proposals":    totalProposals,
		"pending_proposals":  pendingProposals,
		"approved_proposals": approvedProposals,
		"rejected_proposals": rejectedProposals,
		"pattern_count":      patternCount,
		"completion_rate":    completionRate,
	}

	return stats, nil
}

// GetActiveProjects retrieves all active projects
func (r *ProjectRepositoryImpl) GetActiveProjects(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	var projects []models.Project
	filters := Filter{"status": "active"}

	return r.ListWithPagination(ctx, &projects, pagination, filters)
}

// GetRecentProjects retrieves recently created projects
func (r *ProjectRepositoryImpl) GetRecentProjects(ctx context.Context, limit int) ([]models.Project, error) {
	db := r.getDB(ctx)
	var projects []models.Project

	if err := db.Preload("Creator").Order("created_at DESC").Limit(limit).Find(&projects).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get recent projects")
		return nil, fmt.Errorf("failed to get recent projects: %w", err)
	}

	return projects, nil
}

// UpdateProgress updates the progress of a project
func (r *ProjectRepositoryImpl) UpdateProgress(ctx context.Context, projectID uuid.UUID, progress int) error {
	db := r.getDB(ctx)

	// Validate progress range
	if progress < 0 || progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100")
	}

	if err := db.Model(&models.Project{}).Where("id = ?", projectID).Update("progress", progress).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update project progress")
		return fmt.Errorf("failed to update project progress: %w", err)
	}

	// Invalidate cache
	r.cache.Delete(ctx, r.getCacheKey("project", "id", projectID.String()))

	r.logger.WithContext(ctx).WithFields(logrus.Fields{
		"project_id": projectID,
		"progress":   progress,
	}).Info("Project progress updated")

	return nil
}

// GetOverdueProjects retrieves projects that are overdue
func (r *ProjectRepositoryImpl) GetOverdueProjects(ctx context.Context) ([]models.Project, error) {
	db := r.getDB(ctx)
	var projects []models.Project

	now := time.Now()
	if err := db.Where("end_date < ? AND status NOT IN ?", now, []string{"completed", "cancelled"}).Find(&projects).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get overdue projects")
		return nil, fmt.Errorf("failed to get overdue projects: %w", err)
	}

	return projects, nil
}

// GetProjectsByDateRange retrieves projects within a date range
func (r *ProjectRepositoryImpl) GetProjectsByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)

	// Apply date range filters
	db = r.buildDateRangeQuery(db, "created_at", &startDate, &endDate)

	// Get total count
	var total int64
	if err := db.Model(&models.Project{}).Count(&total).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to count projects in date range")
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	// Set pagination defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}
	if pagination.Sort == "" {
		pagination.Sort = "created_at"
	}
	if pagination.Order == "" {
		pagination.Order = "desc"
	}

	// Get projects
	var projects []models.Project
	offset := (pagination.Page - 1) * pagination.PageSize
	orderClause := fmt.Sprintf("%s %s", pagination.Sort, pagination.Order)

	if err := db.Preload("Creator").Order(orderClause).Limit(pagination.PageSize).Offset(offset).Find(&projects).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get projects by date range")
		return nil, fmt.Errorf("failed to get projects by date range: %w", err)
	}

	// Calculate total pages
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))

	return &PaginationResult{
		Data:       projects,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetProjectsByTags retrieves projects by tags
func (r *ProjectRepositoryImpl) GetProjectsByTags(ctx context.Context, tags []string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)

	// Build tag search query
	for _, tag := range tags {
		db = db.Where("tags LIKE ?", "%"+tag+"%")
	}

	// Get total count
	var total int64
	if err := db.Model(&models.Project{}).Count(&total).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to count projects by tags")
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	// Set pagination defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}
	if pagination.Sort == "" {
		pagination.Sort = "created_at"
	}
	if pagination.Order == "" {
		pagination.Order = "desc"
	}

	// Get projects
	var projects []models.Project
	offset := (pagination.Page - 1) * pagination.PageSize
	orderClause := fmt.Sprintf("%s %s", pagination.Sort, pagination.Order)

	if err := db.Preload("Creator").Order(orderClause).Limit(pagination.PageSize).Offset(offset).Find(&projects).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get projects by tags")
		return nil, fmt.Errorf("failed to get projects by tags: %w", err)
	}

	// Calculate total pages
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))

	return &PaginationResult{
		Data:       projects,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// SearchProjects searches for projects by name, description, or tags
func (r *ProjectRepositoryImpl) SearchProjects(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)

	// Build search query
	searchFields := []string{"name", "description", "tags"}
	db = r.buildSearchQuery(db, query, searchFields)

	// Get total count
	var total int64
	if err := db.Model(&models.Project{}).Count(&total).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to count projects in search")
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	// Set pagination defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}
	if pagination.Sort == "" {
		pagination.Sort = "name"
	}
	if pagination.Order == "" {
		pagination.Order = "asc"
	}

	// Get projects
	var projects []models.Project
	offset := (pagination.Page - 1) * pagination.PageSize
	orderClause := fmt.Sprintf("%s %s", pagination.Sort, pagination.Order)

	if err := db.Preload("Creator").Order(orderClause).Limit(pagination.PageSize).Offset(offset).Find(&projects).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to search projects")
		return nil, fmt.Errorf("failed to search projects: %w", err)
	}

	// Calculate total pages
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))

	return &PaginationResult{
		Data:       projects,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetProjectsWithTaskCounts retrieves projects with their task counts
func (r *ProjectRepositoryImpl) GetProjectsWithTaskCounts(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)

	// Get projects with task counts using a raw query for better performance
	type ProjectWithTaskCount struct {
		models.Project
		TaskCount int64 `json:"task_count"`
	}

	var projects []ProjectWithTaskCount

	// Set pagination defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}
	if pagination.Sort == "" {
		pagination.Sort = "created_at"
	}
	if pagination.Order == "" {
		pagination.Order = "desc"
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	orderClause := fmt.Sprintf("projects.%s %s", pagination.Sort, pagination.Order)

	query := `
		SELECT projects.*, 
			   COALESCE(task_counts.task_count, 0) as task_count
		FROM projects
		LEFT JOIN (
			SELECT project_id, COUNT(*) as task_count
			FROM tasks
			WHERE deleted_at IS NULL
			GROUP BY project_id
		) task_counts ON projects.id = task_counts.project_id
		WHERE projects.deleted_at IS NULL
		ORDER BY ` + orderClause + `
		LIMIT ? OFFSET ?
	`

	if err := db.Raw(query, pagination.PageSize, offset).Scan(&projects).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get projects with task counts")
		return nil, fmt.Errorf("failed to get projects with task counts: %w", err)
	}

	// Get total count
	var total int64
	if err := db.Model(&models.Project{}).Count(&total).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to count projects")
		return nil, fmt.Errorf("failed to count projects: %w", err)
	}

	// Calculate total pages
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))

	return &PaginationResult{
		Data:       projects,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}
