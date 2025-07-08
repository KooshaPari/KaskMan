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

// UserRepositoryImpl implements the UserRepository interface
type UserRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewUserRepository creates a new user repository instance
func NewUserRepository(db *gorm.DB, logger *logrus.Logger, cache CacheManager) UserRepository {
	return &UserRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db, logger, cache),
	}
}

// GetByUsername retrieves a user by username
func (r *UserRepositoryImpl) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	// Try cache first
	cacheKey := r.getCacheKey("user", "username", username)
	var user models.User
	if err := r.cache.Get(ctx, cacheKey, &user); err == nil {
		return &user, nil
	}

	db := r.getDB(ctx)
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with username '%s' not found", username)
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get user by username")
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	// Cache the result
	r.cache.Set(ctx, cacheKey, user, time.Hour)

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	// Try cache first
	cacheKey := r.getCacheKey("user", "email", email)
	var user models.User
	if err := r.cache.Get(ctx, cacheKey, &user); err == nil {
		return &user, nil
	}

	db := r.getDB(ctx)
	if err := db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with email '%s' not found", email)
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get user by email")
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Cache the result
	r.cache.Set(ctx, cacheKey, user, time.Hour)

	return &user, nil
}

// GetByCredentials retrieves a user by username or email
func (r *UserRepositoryImpl) GetByCredentials(ctx context.Context, identifier string) (*models.User, error) {
	db := r.getDB(ctx)
	var user models.User

	if err := db.Where("username = ? OR email = ?", identifier, identifier).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with identifier '%s' not found", identifier)
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get user by credentials")
		return nil, fmt.Errorf("failed to get user by credentials: %w", err)
	}

	return &user, nil
}

// GetActiveUsers retrieves all active users with pagination
func (r *UserRepositoryImpl) GetActiveUsers(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	var users []models.User
	filters := Filter{"is_active": true}

	return r.ListWithPagination(ctx, &users, pagination, filters)
}

// GetUsersByRole retrieves users by role with pagination
func (r *UserRepositoryImpl) GetUsersByRole(ctx context.Context, role string, pagination Pagination) (*PaginationResult, error) {
	var users []models.User
	filters := Filter{"role": role}

	return r.ListWithPagination(ctx, &users, pagination, filters)
}

// UpdateLastLogin updates the last login time for a user
func (r *UserRepositoryImpl) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	db := r.getDB(ctx)
	now := time.Now()

	if err := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"last_login_at": &now,
	}).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update last login")
		return fmt.Errorf("failed to update last login: %w", err)
	}

	// Invalidate cache
	r.cache.Delete(ctx, r.getCacheKey("user", "id", userID.String()))

	return nil
}

// IncrementLoginAttempts increments the login attempts counter for a user
func (r *UserRepositoryImpl) IncrementLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	db := r.getDB(ctx)

	if err := db.Model(&models.User{}).Where("id = ?", userID).Update("login_attempts", gorm.Expr("login_attempts + 1")).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to increment login attempts")
		return fmt.Errorf("failed to increment login attempts: %w", err)
	}

	// Invalidate cache
	r.cache.Delete(ctx, r.getCacheKey("user", "id", userID.String()))

	return nil
}

// ResetLoginAttempts resets the login attempts counter for a user
func (r *UserRepositoryImpl) ResetLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	db := r.getDB(ctx)

	if err := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"login_attempts": 0,
		"locked_until":   nil,
	}).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to reset login attempts")
		return fmt.Errorf("failed to reset login attempts: %w", err)
	}

	// Invalidate cache
	r.cache.Delete(ctx, r.getCacheKey("user", "id", userID.String()))

	return nil
}

// LockUser locks a user account for a specified duration
func (r *UserRepositoryImpl) LockUser(ctx context.Context, userID uuid.UUID, duration time.Duration) error {
	db := r.getDB(ctx)
	lockedUntil := time.Now().Add(duration)

	if err := db.Model(&models.User{}).Where("id = ?", userID).Update("locked_until", &lockedUntil).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to lock user")
		return fmt.Errorf("failed to lock user: %w", err)
	}

	// Invalidate cache
	r.cache.Delete(ctx, r.getCacheKey("user", "id", userID.String()))

	r.logger.WithContext(ctx).WithFields(logrus.Fields{
		"user_id":      userID,
		"locked_until": lockedUntil,
	}).Info("User account locked")

	return nil
}

// UnlockUser unlocks a user account
func (r *UserRepositoryImpl) UnlockUser(ctx context.Context, userID uuid.UUID) error {
	db := r.getDB(ctx)

	if err := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"locked_until":   nil,
		"login_attempts": 0,
	}).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to unlock user")
		return fmt.Errorf("failed to unlock user: %w", err)
	}

	// Invalidate cache
	r.cache.Delete(ctx, r.getCacheKey("user", "id", userID.String()))

	r.logger.WithContext(ctx).WithField("user_id", userID).Info("User account unlocked")

	return nil
}

// GetLockedUsers retrieves all locked users
func (r *UserRepositoryImpl) GetLockedUsers(ctx context.Context) ([]models.User, error) {
	db := r.getDB(ctx)
	var users []models.User

	if err := db.Where("locked_until IS NOT NULL AND locked_until > ?", time.Now()).Find(&users).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get locked users")
		return nil, fmt.Errorf("failed to get locked users: %w", err)
	}

	return users, nil
}

// UpdatePassword updates the password hash for a user
func (r *UserRepositoryImpl) UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	db := r.getDB(ctx)

	if err := db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", hashedPassword).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update password")
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Invalidate cache
	r.cache.Delete(ctx, r.getCacheKey("user", "id", userID.String()))

	r.logger.WithContext(ctx).WithField("user_id", userID).Info("User password updated")

	return nil
}

// GetUserStatistics retrieves statistics for a specific user
func (r *UserRepositoryImpl) GetUserStatistics(ctx context.Context, userID uuid.UUID) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	// Get project count
	var projectCount int64
	db.Model(&models.Project{}).Where("created_by = ?", userID).Count(&projectCount)

	// Get task count
	var taskCount int64
	db.Model(&models.Task{}).Where("assigned_to = ?", userID).Count(&taskCount)

	// Get completed task count
	var completedTaskCount int64
	db.Model(&models.Task{}).Where("assigned_to = ? AND status = ?", userID, "completed").Count(&completedTaskCount)

	// Get proposal count
	var proposalCount int64
	db.Model(&models.Proposal{}).Where("submitted_by = ?", userID).Count(&proposalCount)

	// Get recent activity count (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var recentActivityCount int64
	db.Model(&models.ActivityLog{}).Where("user_id = ? AND created_at >= ?", userID, thirtyDaysAgo).Count(&recentActivityCount)

	stats := map[string]interface{}{
		"projects_created":    projectCount,
		"tasks_assigned":      taskCount,
		"tasks_completed":     completedTaskCount,
		"proposals_submitted": proposalCount,
		"recent_activities":   recentActivityCount,
		"completion_rate":     0.0,
	}

	// Calculate completion rate
	if taskCount > 0 {
		stats["completion_rate"] = float64(completedTaskCount) / float64(taskCount) * 100
	}

	return stats, nil
}

// SearchUsers searches for users by username, email, first name, or last name
func (r *UserRepositoryImpl) SearchUsers(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)

	// Build search query
	searchFields := []string{"username", "email", "first_name", "last_name"}
	db = r.buildSearchQuery(db, query, searchFields)

	// Get total count
	var total int64
	if err := db.Model(&models.User{}).Count(&total).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to count users in search")
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Set pagination defaults
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}
	if pagination.Sort == "" {
		pagination.Sort = "username"
	}
	if pagination.Order == "" {
		pagination.Order = "asc"
	}

	// Get users
	var users []models.User
	offset := (pagination.Page - 1) * pagination.PageSize
	orderClause := fmt.Sprintf("%s %s", pagination.Sort, pagination.Order)

	if err := db.Order(orderClause).Limit(pagination.PageSize).Offset(offset).Find(&users).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to search users")
		return nil, fmt.Errorf("failed to search users: %w", err)
	}

	// Calculate total pages
	totalPages := int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize))

	return &PaginationResult{
		Data:       users,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}
