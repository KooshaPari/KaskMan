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

// InsightRepositoryImpl implements the InsightRepository interface
type InsightRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewInsightRepository creates a new insight repository instance
func NewInsightRepository(db *gorm.DB, logger *logrus.Logger, cache CacheManager) InsightRepository {
	return &InsightRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db, logger, cache),
	}
}

// GetByType retrieves insights by type with pagination
func (r *InsightRepositoryImpl) GetByType(ctx context.Context, insightType string, pagination Pagination) (*PaginationResult, error) {
	var insights []models.Insight
	filters := Filter{"type": insightType}
	return r.ListWithPagination(ctx, &insights, pagination, filters)
}

// GetByImpact retrieves insights by impact with pagination
func (r *InsightRepositoryImpl) GetByImpact(ctx context.Context, impact string, pagination Pagination) (*PaginationResult, error) {
	var insights []models.Insight
	filters := Filter{"impact": impact}
	return r.ListWithPagination(ctx, &insights, pagination, filters)
}

// GetByPattern retrieves insights by pattern ID with pagination
func (r *InsightRepositoryImpl) GetByPattern(ctx context.Context, patternID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var insights []models.Insight
	filters := Filter{"pattern_id": patternID}
	return r.ListWithPagination(ctx, &insights, pagination, filters)
}

// GetByConfidenceRange retrieves insights by confidence range with pagination
func (r *InsightRepositoryImpl) GetByConfidenceRange(ctx context.Context, minConfidence, maxConfidence float64, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var insights []models.Insight

	query := r.buildNumericRangeQuery(db, "confidence", minConfidence, maxConfidence)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Insight{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count insights: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("confidence DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Pattern").
		Find(&insights).Error; err != nil {
		return nil, fmt.Errorf("failed to get insights by confidence range: %w", err)
	}

	return &PaginationResult{
		Data:       insights,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetActionableInsights retrieves actionable insights with pagination
func (r *InsightRepositoryImpl) GetActionableInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	var insights []models.Insight
	filters := Filter{"is_actionable": true}
	return r.ListWithPagination(ctx, &insights, pagination, filters)
}

// GetNonActionableInsights retrieves non-actionable insights with pagination
func (r *InsightRepositoryImpl) GetNonActionableInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	var insights []models.Insight
	filters := Filter{"is_actionable": false}
	return r.ListWithPagination(ctx, &insights, pagination, filters)
}

// GetImplementedInsights retrieves implemented insights with pagination
func (r *InsightRepositoryImpl) GetImplementedInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	var insights []models.Insight
	filters := Filter{"is_implemented": true}
	return r.ListWithPagination(ctx, &insights, pagination, filters)
}

// GetUnimplementedInsights retrieves unimplemented insights with pagination
func (r *InsightRepositoryImpl) GetUnimplementedInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	var insights []models.Insight
	filters := Filter{"is_implemented": false}
	return r.ListWithPagination(ctx, &insights, pagination, filters)
}

// GetHighImpactInsights retrieves high impact insights with pagination
func (r *InsightRepositoryImpl) GetHighImpactInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByImpact(ctx, "high", pagination)
}

// GetCriticalInsights retrieves critical insights with pagination
func (r *InsightRepositoryImpl) GetCriticalInsights(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByImpact(ctx, "critical", pagination)
}

// GetRecentInsights retrieves the most recent insights
func (r *InsightRepositoryImpl) GetRecentInsights(ctx context.Context, limit int) ([]models.Insight, error) {
	db := r.getDB(ctx)
	var insights []models.Insight

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	err := db.Order("created_at DESC").
		Limit(limit).
		Preload("Pattern").
		Find(&insights).Error

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get recent insights")
		return nil, fmt.Errorf("failed to get recent insights: %w", err)
	}

	return insights, nil
}

// MarkAsImplemented marks an insight as implemented
func (r *InsightRepositoryImpl) MarkAsImplemented(ctx context.Context, insightID uuid.UUID) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Insight{}).Where("id = ?", insightID).Update("is_implemented", true).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to mark insight as implemented")
		return fmt.Errorf("failed to mark insight as implemented: %w", err)
	}

	r.logger.WithContext(ctx).WithField("insight_id", insightID).Info("Insight marked as implemented")
	return nil
}

// MarkAsUnimplemented marks an insight as unimplemented
func (r *InsightRepositoryImpl) MarkAsUnimplemented(ctx context.Context, insightID uuid.UUID) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Insight{}).Where("id = ?", insightID).Update("is_implemented", false).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to mark insight as unimplemented")
		return fmt.Errorf("failed to mark insight as unimplemented: %w", err)
	}

	r.logger.WithContext(ctx).WithField("insight_id", insightID).Info("Insight marked as unimplemented")
	return nil
}

// UpdateConfidence updates an insight's confidence
func (r *InsightRepositoryImpl) UpdateConfidence(ctx context.Context, insightID uuid.UUID, confidence float64) error {
	db := r.getDB(ctx)

	err := db.Model(&models.Insight{}).Where("id = ?", insightID).Update("confidence", confidence).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update insight confidence")
		return fmt.Errorf("failed to update insight confidence: %w", err)
	}

	return nil
}

// GetInsightStatistics retrieves insight statistics with optional filters
func (r *InsightRepositoryImpl) GetInsightStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	// Build base query with filters
	query := db.Model(&models.Insight{})
	for key, value := range filters {
		if value != nil {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	stats := make(map[string]interface{})

	// Total insights
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total insights: %w", err)
	}
	stats["total"] = total

	// Insights by type
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

	// Insights by impact
	var impactCounts []struct {
		Impact string
		Count  int64
	}
	if err := query.Select("impact, COUNT(*) as count").Group("impact").Scan(&impactCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get impact counts: %w", err)
	}

	impactStats := make(map[string]int64)
	for _, ic := range impactCounts {
		impactStats[ic.Impact] = ic.Count
	}
	stats["by_impact"] = impactStats

	// Implementation statistics
	var implementationStats struct {
		Actionable     int64
		NonActionable  int64
		Implemented    int64
		NotImplemented int64
	}

	query.Where("is_actionable = ?", true).Count(&implementationStats.Actionable)
	query.Where("is_actionable = ?", false).Count(&implementationStats.NonActionable)
	query.Where("is_implemented = ?", true).Count(&implementationStats.Implemented)
	query.Where("is_implemented = ?", false).Count(&implementationStats.NotImplemented)

	stats["implementation"] = implementationStats

	// Average confidence
	var avgConfidence struct {
		AvgConfidence float64
	}
	if err := query.Select("AVG(confidence) as avg_confidence").Scan(&avgConfidence).Error; err != nil {
		return nil, fmt.Errorf("failed to get average confidence: %w", err)
	}
	stats["avg_confidence"] = avgConfidence.AvgConfidence

	return stats, nil
}

// GetInsightTrends retrieves insight trends over time
func (r *InsightRepositoryImpl) GetInsightTrends(ctx context.Context, period string) (map[string]interface{}, error) {
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

	// Get insight creation trends
	var creationTrends []struct {
		Date  time.Time
		Count int64
	}

	err := db.Model(&models.Insight{}).
		Select(fmt.Sprintf("%s as date, COUNT(*) as count", dateFormat)).
		Group("date").
		Order("date").
		Scan(&creationTrends).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get creation trends: %w", err)
	}

	trends["creation_trends"] = creationTrends

	// Get implementation trends
	var implementationTrends []struct {
		Date  time.Time
		Count int64
	}

	err = db.Model(&models.Insight{}).
		Where("is_implemented = ?", true).
		Select(fmt.Sprintf("%s as date, COUNT(*) as count", dateFormat)).
		Group("date").
		Order("date").
		Scan(&implementationTrends).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get implementation trends: %w", err)
	}

	trends["implementation_trends"] = implementationTrends

	return trends, nil
}

// SearchInsights searches insights by query string with pagination
func (r *InsightRepositoryImpl) SearchInsights(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var insights []models.Insight

	searchQuery := r.buildSearchQuery(db, query, []string{"title", "description", "type", "action_items", "data"})

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := searchQuery.Model(&models.Insight{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count insights: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := searchQuery.Order("confidence DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Pattern").
		Find(&insights).Error; err != nil {
		return nil, fmt.Errorf("failed to search insights: %w", err)
	}

	return &PaginationResult{
		Data:       insights,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetInsightsByActionItems retrieves insights by action items with pagination
func (r *InsightRepositoryImpl) GetInsightsByActionItems(ctx context.Context, actionItems []string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var insights []models.Insight

	// For this implementation, we'll do a simple JSON field search
	// In a real implementation, you'd want to use proper JSON queries
	query := db
	for _, item := range actionItems {
		query = query.Where("action_items::text LIKE ?", "%"+item+"%")
	}

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Insight{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count insights: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Pattern").
		Find(&insights).Error; err != nil {
		return nil, fmt.Errorf("failed to get insights by action items: %w", err)
	}

	return &PaginationResult{
		Data:       insights,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetInsightEffectiveness retrieves effectiveness metrics for an insight
func (r *InsightRepositoryImpl) GetInsightEffectiveness(ctx context.Context, insightID uuid.UUID) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	var insight models.Insight
	err := db.First(&insight, "id = ?", insightID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get insight: %w", err)
	}

	effectiveness := map[string]interface{}{
		"insight_id":     insight.ID,
		"title":          insight.Title,
		"type":           insight.Type,
		"impact":         insight.Impact,
		"confidence":     insight.Confidence,
		"is_actionable":  insight.IsActionable,
		"is_implemented": insight.IsImplemented,
		"created_at":     insight.CreatedAt,
	}

	// Calculate time to implementation if implemented
	if insight.IsImplemented {
		// In a real implementation, you'd track implementation date
		// For now, we'll estimate based on updated_at
		timeSinceCreation := insight.UpdatedAt.Sub(insight.CreatedAt)
		effectiveness["time_to_implementation"] = timeSinceCreation.String()
	}

	// Calculate related metrics if pattern is available
	if insight.PatternID != nil {
		var pattern models.Pattern
		if err := db.First(&pattern, "id = ?", *insight.PatternID).Error; err == nil {
			effectiveness["pattern_confidence"] = pattern.Confidence
			effectiveness["pattern_frequency"] = pattern.Frequency
			effectiveness["pattern_significance"] = pattern.Significance

			// Calculate insight-to-pattern correlation
			correlation := insight.Confidence * pattern.Confidence
			effectiveness["pattern_correlation"] = correlation
		}
	}

	return effectiveness, nil
}
