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

// PatternRepositoryImpl implements the PatternRepository interface
type PatternRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewPatternRepository creates a new pattern repository instance
func NewPatternRepository(db *gorm.DB, logger *logrus.Logger, cache CacheManager) PatternRepository {
	return &PatternRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db, logger, cache),
	}
}

// GetByType retrieves patterns by type with pagination
func (r *PatternRepositoryImpl) GetByType(ctx context.Context, patternType string, pagination Pagination) (*PaginationResult, error) {
	var patterns []models.Pattern
	filters := Filter{"type": patternType}
	return r.ListWithPagination(ctx, &patterns, pagination, filters)
}

// GetByProject retrieves patterns by project ID with pagination
func (r *PatternRepositoryImpl) GetByProject(ctx context.Context, projectID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var patterns []models.Pattern
	filters := Filter{"project_id": projectID}
	return r.ListWithPagination(ctx, &patterns, pagination, filters)
}

// GetByConfidenceRange retrieves patterns by confidence range with pagination
func (r *PatternRepositoryImpl) GetByConfidenceRange(ctx context.Context, minConfidence, maxConfidence float64, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	query := r.buildNumericRangeQuery(db, "confidence", minConfidence, maxConfidence)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Pattern{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count patterns: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("confidence DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get patterns by confidence range: %w", err)
	}

	return &PaginationResult{
		Data:       patterns,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetBySignificanceRange retrieves patterns by significance range with pagination
func (r *PatternRepositoryImpl) GetBySignificanceRange(ctx context.Context, minSignificance, maxSignificance float64, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	query := r.buildNumericRangeQuery(db, "significance", minSignificance, maxSignificance)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Pattern{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count patterns: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("significance DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get patterns by significance range: %w", err)
	}

	return &PaginationResult{
		Data:       patterns,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetByFrequencyRange retrieves patterns by frequency range with pagination
func (r *PatternRepositoryImpl) GetByFrequencyRange(ctx context.Context, minFrequency, maxFrequency int, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	query := r.buildNumericRangeQuery(db, "frequency", minFrequency, maxFrequency)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Pattern{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count patterns: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("frequency DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get patterns by frequency range: %w", err)
	}

	return &PaginationResult{
		Data:       patterns,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetRecentPatterns retrieves the most recent patterns
func (r *PatternRepositoryImpl) GetRecentPatterns(ctx context.Context, limit int) ([]models.Pattern, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	err := db.Order("last_seen DESC").
		Limit(limit).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get recent patterns")
		return nil, fmt.Errorf("failed to get recent patterns: %w", err)
	}

	return patterns, nil
}

// GetHighConfidencePatterns retrieves patterns with confidence above threshold
func (r *PatternRepositoryImpl) GetHighConfidencePatterns(ctx context.Context, threshold float64, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	query := db.Where("confidence >= ?", threshold)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Pattern{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count patterns: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("confidence DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get high confidence patterns: %w", err)
	}

	return &PaginationResult{
		Data:       patterns,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetFrequentPatterns retrieves patterns with frequency above threshold
func (r *PatternRepositoryImpl) GetFrequentPatterns(ctx context.Context, threshold int, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	query := db.Where("frequency >= ?", threshold)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Pattern{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count patterns: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("frequency DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get frequent patterns: %w", err)
	}

	return &PaginationResult{
		Data:       patterns,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetPatternWithInsights retrieves a pattern with its insights
func (r *PatternRepositoryImpl) GetPatternWithInsights(ctx context.Context, patternID uuid.UUID) (*models.Pattern, error) {
	db := r.getDB(ctx)
	var pattern models.Pattern

	err := db.Preload("Insights").Preload("Project").First(&pattern, "id = ?", patternID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("pattern with id %s not found", patternID.String())
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get pattern with insights")
		return nil, fmt.Errorf("failed to get pattern with insights: %w", err)
	}

	return &pattern, nil
}

// UpdateConfidence updates a pattern's confidence
func (r *PatternRepositoryImpl) UpdateConfidence(ctx context.Context, patternID uuid.UUID, confidence float64) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Pattern{}).Where("id = ?", patternID).Update("confidence", confidence).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update pattern confidence")
		return fmt.Errorf("failed to update pattern confidence: %w", err)
	}

	return nil
}

// UpdateFrequency updates a pattern's frequency
func (r *PatternRepositoryImpl) UpdateFrequency(ctx context.Context, patternID uuid.UUID, frequency int) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Pattern{}).Where("id = ?", patternID).Update("frequency", frequency).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update pattern frequency")
		return fmt.Errorf("failed to update pattern frequency: %w", err)
	}

	return nil
}

// UpdateSignificance updates a pattern's significance
func (r *PatternRepositoryImpl) UpdateSignificance(ctx context.Context, patternID uuid.UUID, significance float64) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Pattern{}).Where("id = ?", patternID).Update("significance", significance).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update pattern significance")
		return fmt.Errorf("failed to update pattern significance: %w", err)
	}

	return nil
}

// UpdateLastSeen updates a pattern's last seen timestamp
func (r *PatternRepositoryImpl) UpdateLastSeen(ctx context.Context, patternID uuid.UUID) error {
	db := r.getDB(ctx)

	now := time.Now()
	err := db.Model(&models.Pattern{}).Where("id = ?", patternID).Update("last_seen", now).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update pattern last seen")
		return fmt.Errorf("failed to update pattern last seen: %w", err)
	}

	return nil
}

// GetPatternStatistics retrieves pattern statistics with optional filters
func (r *PatternRepositoryImpl) GetPatternStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	// Build base query with filters
	query := db.Model(&models.Pattern{})
	for key, value := range filters {
		if value != nil {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	stats := make(map[string]interface{})

	// Total patterns
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total patterns: %w", err)
	}
	stats["total"] = total

	// Patterns by type
	var typeCounts []struct {
		Type  string
		Count int64
	}
	if err := query.Select("type, COUNT(*) as count").Group("type").Scan(&typeCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get type counts: %w", err)
	}

	typeStats := make(map[string]int64)
	for _, tc := range typeCounts {
		typeStats[tc.Type] = tc.Count
	}
	stats["by_type"] = typeStats

	// Average confidence and significance
	var averages struct {
		AvgConfidence   float64
		AvgSignificance float64
		AvgFrequency    float64
	}
	if err := query.Select("AVG(confidence) as avg_confidence, AVG(significance) as avg_significance, AVG(frequency) as avg_frequency").Scan(&averages).Error; err != nil {
		return nil, fmt.Errorf("failed to get averages: %w", err)
	}

	stats["avg_confidence"] = averages.AvgConfidence
	stats["avg_significance"] = averages.AvgSignificance
	stats["avg_frequency"] = averages.AvgFrequency

	return stats, nil
}

// GetPatternTrends retrieves pattern trends over time
func (r *PatternRepositoryImpl) GetPatternTrends(ctx context.Context, period string) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	trends := make(map[string]interface{})

	// Determine date format based on period
	var dateFormat string
	switch period {
	case "daily":
		dateFormat = "DATE(created_at)"
	case "weekly":
		dateFormat = "DATE_TRUNC('week', created_at)"
	case "monthly":
		dateFormat = "DATE_TRUNC('month', created_at)"
	default:
		dateFormat = "DATE(created_at)"
	}

	// Get pattern creation trends
	var creationTrends []struct {
		Date  time.Time
		Count int64
	}

	err := db.Model(&models.Pattern{}).
		Select(fmt.Sprintf("%s as date, COUNT(*) as count", dateFormat)).
		Group("date").
		Order("date").
		Scan(&creationTrends).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get creation trends: %w", err)
	}

	trends["creation_trends"] = creationTrends

	// Get confidence trends
	var confidenceTrends []struct {
		Date          time.Time
		AvgConfidence float64
	}

	err = db.Model(&models.Pattern{}).
		Select(fmt.Sprintf("%s as date, AVG(confidence) as avg_confidence", dateFormat)).
		Group("date").
		Order("date").
		Scan(&confidenceTrends).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get confidence trends: %w", err)
	}

	trends["confidence_trends"] = confidenceTrends

	return trends, nil
}

// SearchPatterns searches patterns by query string with pagination
func (r *PatternRepositoryImpl) SearchPatterns(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	searchQuery := r.buildSearchQuery(db, query, []string{"name", "description", "type", "data", "context"})

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := searchQuery.Model(&models.Pattern{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count patterns: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := searchQuery.Order("confidence DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to search patterns: %w", err)
	}

	return &PaginationResult{
		Data:       patterns,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetSimilarPatterns retrieves patterns similar to a given pattern
func (r *PatternRepositoryImpl) GetSimilarPatterns(ctx context.Context, patternID uuid.UUID, threshold float64) ([]models.Pattern, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	// Get the reference pattern
	var refPattern models.Pattern
	if err := db.First(&refPattern, "id = ?", patternID).Error; err != nil {
		return nil, fmt.Errorf("failed to get reference pattern: %w", err)
	}

	// For this implementation, we'll find patterns of the same type with similar confidence
	// In a real implementation, you'd want to use vector similarity or other advanced algorithms
	confidenceRange := 0.1 // 10% range
	minConfidence := refPattern.Confidence - confidenceRange
	maxConfidence := refPattern.Confidence + confidenceRange

	err := db.Where("id != ? AND type = ? AND confidence BETWEEN ? AND ?",
		patternID, refPattern.Type, minConfidence, maxConfidence).
		Order("confidence DESC").
		Limit(10).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get similar patterns")
		return nil, fmt.Errorf("failed to get similar patterns: %w", err)
	}

	return patterns, nil
}

// GetPatternsByContext retrieves patterns by context data with pagination
func (r *PatternRepositoryImpl) GetPatternsByContext(ctx context.Context, context map[string]interface{}, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var patterns []models.Pattern

	// For this implementation, we'll do a simple JSON field search
	// In a real implementation, you'd want to use proper JSON queries
	query := db
	for key, value := range context {
		query = query.Where("context::text LIKE ?", fmt.Sprintf("%%\"%s\":\"%v\"%%", key, value))
	}

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Pattern{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count patterns: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Insights").
		Find(&patterns).Error; err != nil {
		return nil, fmt.Errorf("failed to get patterns by context: %w", err)
	}

	return &PaginationResult{
		Data:       patterns,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}
