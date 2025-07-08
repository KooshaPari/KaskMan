package repositories

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"gorm.io/gorm"
)

// ActivityLogRepositoryImpl implements the ActivityLogRepository interface
type ActivityLogRepositoryImpl struct {
	db *gorm.DB
}

// NewActivityLogRepository creates a new ActivityLogRepository implementation
func NewActivityLogRepository(db *gorm.DB) ActivityLogRepository {
	return &ActivityLogRepositoryImpl{db: db}
}

// Create creates a new activity log entry
func (r *ActivityLogRepositoryImpl) Create(ctx context.Context, entity interface{}) error {
	return r.db.WithContext(ctx).Create(entity).Error
}

// GetByID retrieves an activity log entry by ID
func (r *ActivityLogRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID, entity interface{}) error {
	return r.db.WithContext(ctx).Where("id = ?", id).First(entity).Error
}

// Update updates an activity log entry
func (r *ActivityLogRepositoryImpl) Update(ctx context.Context, entity interface{}) error {
	return r.db.WithContext(ctx).Save(entity).Error
}

// Delete hard deletes an activity log entry
func (r *ActivityLogRepositoryImpl) Delete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	return r.db.WithContext(ctx).Unscoped().Delete(entity, "id = ?", id).Error
}

// SoftDelete soft deletes an activity log entry
func (r *ActivityLogRepositoryImpl) SoftDelete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	return r.db.WithContext(ctx).Delete(entity, "id = ?", id).Error
}

// List retrieves activity log entries with filters
func (r *ActivityLogRepositoryImpl) List(ctx context.Context, entities interface{}, filters Filter) error {
	query := r.db.WithContext(ctx).Model(&models.ActivityLog{})

	// Apply filters
	for key, value := range filters {
		query = query.Where(key+" = ?", value)
	}

	return query.Find(entities).Error
}

// ListWithPagination retrieves activity log entries with pagination and filters
func (r *ActivityLogRepositoryImpl) ListWithPagination(ctx context.Context, entities interface{}, pagination Pagination, filters Filter) (*PaginationResult, error) {
	query := r.db.WithContext(ctx).Model(&models.ActivityLog{}).Preload("User")

	// Apply filters
	for key, value := range filters {
		switch key {
		case "user_id", "resource_id":
			if value != nil {
				query = query.Where(key+" = ?", value)
			}
		case "action", "resource":
			query = query.Where(key+" = ?", value)
		case "success":
			query = query.Where(key+" = ?", value)
		case "ip_address":
			query = query.Where(key+" = ?", value)
		default:
			query = query.Where(key+" = ?", value)
		}
	}

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	query = query.Offset(offset).Limit(pagination.PageSize)

	// Apply sorting
	if pagination.Sort != "" {
		orderBy := pagination.Sort
		if pagination.Order != "" {
			orderBy += " " + pagination.Order
		}
		query = query.Order(orderBy)
	} else {
		query = query.Order("created_at DESC")
	}

	// Execute query
	if err := query.Find(entities).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize != 0 {
		totalPages++
	}

	return &PaginationResult{
		Data:       entities,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Count counts activity log entries with filters
func (r *ActivityLogRepositoryImpl) Count(ctx context.Context, entity interface{}, filters Filter) (int64, error) {
	query := r.db.WithContext(ctx).Model(entity)

	// Apply filters
	for key, value := range filters {
		query = query.Where(key+" = ?", value)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}

// Exists checks if an activity log entry exists
func (r *ActivityLogRepositoryImpl) Exists(ctx context.Context, id uuid.UUID, entity interface{}) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(entity).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

// BatchCreate creates multiple activity log entries
func (r *ActivityLogRepositoryImpl) BatchCreate(ctx context.Context, entities interface{}) error {
	return r.db.WithContext(ctx).Create(entities).Error
}

// BatchUpdate updates multiple activity log entries
func (r *ActivityLogRepositoryImpl) BatchUpdate(ctx context.Context, entities interface{}) error {
	return r.db.WithContext(ctx).Save(entities).Error
}

// BatchDelete deletes multiple activity log entries
func (r *ActivityLogRepositoryImpl) BatchDelete(ctx context.Context, ids []uuid.UUID, entity interface{}) error {
	return r.db.WithContext(ctx).Delete(entity, "id IN ?", ids).Error
}

// GetByUser retrieves activity log entries for a specific user
func (r *ActivityLogRepositoryImpl) GetByUser(ctx context.Context, userID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var activities []models.ActivityLog
	return r.ListWithPagination(ctx, &activities, pagination, Filter{"user_id": userID})
}

// GetByAction retrieves activity log entries for a specific action
func (r *ActivityLogRepositoryImpl) GetByAction(ctx context.Context, action string, pagination Pagination) (*PaginationResult, error) {
	var activities []models.ActivityLog
	return r.ListWithPagination(ctx, &activities, pagination, Filter{"action": action})
}

// GetByResource retrieves activity log entries for a specific resource
func (r *ActivityLogRepositoryImpl) GetByResource(ctx context.Context, resource string, pagination Pagination) (*PaginationResult, error) {
	var activities []models.ActivityLog
	return r.ListWithPagination(ctx, &activities, pagination, Filter{"resource": resource})
}

// GetByResourceID retrieves activity log entries for a specific resource ID
func (r *ActivityLogRepositoryImpl) GetByResourceID(ctx context.Context, resourceID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var activities []models.ActivityLog
	return r.ListWithPagination(ctx, &activities, pagination, Filter{"resource_id": resourceID})
}

// GetByDateRange retrieves activity log entries within a date range
func (r *ActivityLogRepositoryImpl) GetByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error) {
	query := r.db.WithContext(ctx).Model(&models.ActivityLog{}).Preload("User")
	query = query.Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	query = query.Offset(offset).Limit(pagination.PageSize)

	// Apply sorting
	if pagination.Sort != "" {
		orderBy := pagination.Sort
		if pagination.Order != "" {
			orderBy += " " + pagination.Order
		}
		query = query.Order(orderBy)
	} else {
		query = query.Order("created_at DESC")
	}

	var activities []models.ActivityLog
	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize != 0 {
		totalPages++
	}

	return &PaginationResult{
		Data:       activities,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetRecentActivities retrieves the most recent activity log entries
func (r *ActivityLogRepositoryImpl) GetRecentActivities(ctx context.Context, limit int) ([]models.ActivityLog, error) {
	var activities []models.ActivityLog
	err := r.db.WithContext(ctx).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&activities).Error
	return activities, err
}

// GetSuccessfulActivities retrieves successful activity log entries
func (r *ActivityLogRepositoryImpl) GetSuccessfulActivities(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	var activities []models.ActivityLog
	return r.ListWithPagination(ctx, &activities, pagination, Filter{"success": true})
}

// GetFailedActivities retrieves failed activity log entries
func (r *ActivityLogRepositoryImpl) GetFailedActivities(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	var activities []models.ActivityLog
	return r.ListWithPagination(ctx, &activities, pagination, Filter{"success": false})
}

// GetActivitiesWithErrors retrieves activity log entries with error messages
func (r *ActivityLogRepositoryImpl) GetActivitiesWithErrors(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	query := r.db.WithContext(ctx).Model(&models.ActivityLog{}).Preload("User")
	query = query.Where("error_message IS NOT NULL AND error_message != ''")

	// Count total records
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	query = query.Offset(offset).Limit(pagination.PageSize)

	// Apply sorting
	if pagination.Sort != "" {
		orderBy := pagination.Sort
		if pagination.Order != "" {
			orderBy += " " + pagination.Order
		}
		query = query.Order(orderBy)
	} else {
		query = query.Order("created_at DESC")
	}

	var activities []models.ActivityLog
	if err := query.Find(&activities).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize != 0 {
		totalPages++
	}

	return &PaginationResult{
		Data:       activities,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetUserActivityStats retrieves activity statistics for a user
func (r *ActivityLogRepositoryImpl) GetUserActivityStats(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	var stats struct {
		TotalActivities   int64     `json:"total_activities"`
		SuccessfulActions int64     `json:"successful_actions"`
		FailedActions     int64     `json:"failed_actions"`
		LastActivity      time.Time `json:"last_activity"`
	}

	// Get total activities
	r.db.WithContext(ctx).Model(&models.ActivityLog{}).
		Where("user_id = ?", userID).
		Count(&stats.TotalActivities)

	// Get successful actions
	r.db.WithContext(ctx).Model(&models.ActivityLog{}).
		Where("user_id = ? AND success = ?", userID, true).
		Count(&stats.SuccessfulActions)

	// Get failed actions
	r.db.WithContext(ctx).Model(&models.ActivityLog{}).
		Where("user_id = ? AND success = ?", userID, false).
		Count(&stats.FailedActions)

	// Get last activity
	var lastActivity models.ActivityLog
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&lastActivity).Error; err == nil {
		stats.LastActivity = lastActivity.CreatedAt
	}

	// Convert to map
	result := map[string]interface{}{
		"total_activities":   stats.TotalActivities,
		"successful_actions": stats.SuccessfulActions,
		"failed_actions":     stats.FailedActions,
		"success_rate":       float64(stats.SuccessfulActions) / float64(stats.TotalActivities) * 100,
		"last_activity":      stats.LastActivity,
	}

	return result, nil
}

// GetSystemActivityStats retrieves system-wide activity statistics
func (r *ActivityLogRepositoryImpl) GetSystemActivityStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalActivities   int64 `json:"total_activities"`
		SuccessfulActions int64 `json:"successful_actions"`
		FailedActions     int64 `json:"failed_actions"`
		UniqueUsers       int64 `json:"unique_users"`
		TodayActivities   int64 `json:"today_activities"`
	}

	// Get total activities
	r.db.WithContext(ctx).Model(&models.ActivityLog{}).Count(&stats.TotalActivities)

	// Get successful actions
	r.db.WithContext(ctx).Model(&models.ActivityLog{}).Where("success = ?", true).Count(&stats.SuccessfulActions)

	// Get failed actions
	r.db.WithContext(ctx).Model(&models.ActivityLog{}).Where("success = ?", false).Count(&stats.FailedActions)

	// Get unique users
	r.db.WithContext(ctx).Model(&models.ActivityLog{}).
		Where("user_id IS NOT NULL").
		Distinct("user_id").
		Count(&stats.UniqueUsers)

	// Get today's activities
	today := time.Now().Truncate(24 * time.Hour)
	r.db.WithContext(ctx).Model(&models.ActivityLog{}).
		Where("created_at >= ?", today).
		Count(&stats.TodayActivities)

	// Convert to map
	result := map[string]interface{}{
		"total_activities":   stats.TotalActivities,
		"successful_actions": stats.SuccessfulActions,
		"failed_actions":     stats.FailedActions,
		"unique_users":       stats.UniqueUsers,
		"today_activities":   stats.TodayActivities,
		"success_rate":       float64(stats.SuccessfulActions) / float64(stats.TotalActivities) * 100,
	}

	return result, nil
}

// GetActivityTrends retrieves activity trends over time
func (r *ActivityLogRepositoryImpl) GetActivityTrends(ctx context.Context, period string) (map[string]interface{}, error) {
	var trends []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	var groupBy string
	switch period {
	case "day":
		groupBy = "DATE(created_at)"
	case "week":
		groupBy = "YEARWEEK(created_at)"
	case "month":
		groupBy = "DATE_FORMAT(created_at, '%Y-%m')"
	default:
		groupBy = "DATE(created_at)"
	}

	err := r.db.WithContext(ctx).
		Model(&models.ActivityLog{}).
		Select(groupBy + " as date, COUNT(*) as count").
		Group(groupBy).
		Order("date DESC").
		Limit(30).
		Find(&trends).Error

	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"period": period,
		"trends": trends,
	}

	return result, nil
}

// LogActivity logs a new activity
func (r *ActivityLogRepositoryImpl) LogActivity(ctx context.Context, userID *uuid.UUID, action, resource string, resourceID *uuid.UUID, details map[string]interface{}, success bool, errorMessage string) error {
	// Convert details to JSON
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	// Extract specific fields from details
	var ipAddress, userAgent string
	if details != nil {
		if ip, ok := details["ip_address"].(string); ok {
			ipAddress = ip
		}
		if ua, ok := details["user_agent"].(string); ok {
			userAgent = ua
		}
	}

	activity := models.ActivityLog{
		Action:       action,
		Resource:     resource,
		ResourceID:   resourceID,
		Details:      string(detailsJSON),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Success:      success,
		ErrorMessage: errorMessage,
		UserID:       userID,
	}

	return r.db.WithContext(ctx).Create(&activity).Error
}

// SearchActivities searches activities by query string
func (r *ActivityLogRepositoryImpl) SearchActivities(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error) {
	dbQuery := r.db.WithContext(ctx).Model(&models.ActivityLog{}).Preload("User")

	// Search in action, resource, and details
	searchPattern := "%" + query + "%"
	dbQuery = dbQuery.Where("action LIKE ? OR resource LIKE ? OR details LIKE ? OR error_message LIKE ?",
		searchPattern, searchPattern, searchPattern, searchPattern)

	// Count total records
	var total int64
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// Apply pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	dbQuery = dbQuery.Offset(offset).Limit(pagination.PageSize)

	// Apply sorting
	if pagination.Sort != "" {
		orderBy := pagination.Sort
		if pagination.Order != "" {
			orderBy += " " + pagination.Order
		}
		dbQuery = dbQuery.Order(orderBy)
	} else {
		dbQuery = dbQuery.Order("created_at DESC")
	}

	var activities []models.ActivityLog
	if err := dbQuery.Find(&activities).Error; err != nil {
		return nil, err
	}

	// Calculate total pages
	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize != 0 {
		totalPages++
	}

	return &PaginationResult{
		Data:       activities,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetActivitiesByIPAddress retrieves activities by IP address
func (r *ActivityLogRepositoryImpl) GetActivitiesByIPAddress(ctx context.Context, ipAddress string, pagination Pagination) (*PaginationResult, error) {
	var activities []models.ActivityLog
	return r.ListWithPagination(ctx, &activities, pagination, Filter{"ip_address": ipAddress})
}

// CleanupOldActivities removes activities older than the specified time
func (r *ActivityLogRepositoryImpl) CleanupOldActivities(ctx context.Context, olderThan time.Time) error {
	return r.db.WithContext(ctx).
		Where("created_at < ?", olderThan).
		Delete(&models.ActivityLog{}).Error
}
