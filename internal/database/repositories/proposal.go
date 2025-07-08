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

// ProposalRepositoryImpl implements the ProposalRepository interface
type ProposalRepositoryImpl struct {
	*BaseRepositoryImpl
}

// NewProposalRepository creates a new proposal repository instance
func NewProposalRepository(db *gorm.DB, logger *logrus.Logger, cache CacheManager) ProposalRepository {
	return &ProposalRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository(db, logger, cache),
	}
}

// GetBySubmitter retrieves proposals by submitter ID with pagination
func (r *ProposalRepositoryImpl) GetBySubmitter(ctx context.Context, submitterID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var proposals []models.Proposal
	filters := Filter{"submitted_by": submitterID}
	return r.ListWithPagination(ctx, &proposals, pagination, filters)
}

// GetByStatus retrieves proposals by status with pagination
func (r *ProposalRepositoryImpl) GetByStatus(ctx context.Context, status string, pagination Pagination) (*PaginationResult, error) {
	var proposals []models.Proposal
	filters := Filter{"status": status}
	return r.ListWithPagination(ctx, &proposals, pagination, filters)
}

// GetByCategory retrieves proposals by category with pagination
func (r *ProposalRepositoryImpl) GetByCategory(ctx context.Context, category string, pagination Pagination) (*PaginationResult, error) {
	var proposals []models.Proposal
	filters := Filter{"category": category}
	return r.ListWithPagination(ctx, &proposals, pagination, filters)
}

// GetByPriority retrieves proposals by priority with pagination
func (r *ProposalRepositoryImpl) GetByPriority(ctx context.Context, priority string, pagination Pagination) (*PaginationResult, error) {
	var proposals []models.Proposal
	filters := Filter{"priority": priority}
	return r.ListWithPagination(ctx, &proposals, pagination, filters)
}

// GetByProject retrieves proposals by project ID with pagination
func (r *ProposalRepositoryImpl) GetByProject(ctx context.Context, projectID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var proposals []models.Proposal
	filters := Filter{"project_id": projectID}
	return r.ListWithPagination(ctx, &proposals, pagination, filters)
}

// GetByReviewer retrieves proposals by reviewer ID with pagination
func (r *ProposalRepositoryImpl) GetByReviewer(ctx context.Context, reviewerID uuid.UUID, pagination Pagination) (*PaginationResult, error) {
	var proposals []models.Proposal
	filters := Filter{"reviewed_by": reviewerID}
	return r.ListWithPagination(ctx, &proposals, pagination, filters)
}

// GetPendingProposals retrieves pending proposals with pagination
func (r *ProposalRepositoryImpl) GetPendingProposals(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "pending", pagination)
}

// GetApprovedProposals retrieves approved proposals with pagination
func (r *ProposalRepositoryImpl) GetApprovedProposals(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "approved", pagination)
}

// GetRejectedProposals retrieves rejected proposals with pagination
func (r *ProposalRepositoryImpl) GetRejectedProposals(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "rejected", pagination)
}

// GetUnderReviewProposals retrieves under review proposals with pagination
func (r *ProposalRepositoryImpl) GetUnderReviewProposals(ctx context.Context, pagination Pagination) (*PaginationResult, error) {
	return r.GetByStatus(ctx, "under_review", pagination)
}

// GetProposalsByDateRange retrieves proposals within a date range with pagination
func (r *ProposalRepositoryImpl) GetProposalsByDateRange(ctx context.Context, startDate, endDate time.Time, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var proposals []models.Proposal

	query := r.buildDateRangeQuery(db, "created_at", &startDate, &endDate)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Proposal{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count proposals: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Submitter").
		Preload("Reviewer").
		Find(&proposals).Error; err != nil {
		return nil, fmt.Errorf("failed to get proposals by date range: %w", err)
	}

	return &PaginationResult{
		Data:       proposals,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// ApproveProposal approves a proposal and sets reviewer information
func (r *ProposalRepositoryImpl) ApproveProposal(ctx context.Context, proposalID uuid.UUID, reviewerID uuid.UUID, reviewNotes string) error {
	db := r.getDB(ctx)

	now := time.Now()
	updates := map[string]interface{}{
		"status":       "approved",
		"reviewed_by":  reviewerID,
		"reviewed_at":  &now,
		"review_notes": reviewNotes,
	}

	err := db.Model(&models.Proposal{}).Where("id = ?", proposalID).Updates(updates).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to approve proposal")
		return fmt.Errorf("failed to approve proposal: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logrus.Fields{
		"proposal_id": proposalID,
		"reviewer_id": reviewerID,
	}).Info("Proposal approved")

	return nil
}

// RejectProposal rejects a proposal and sets reviewer information
func (r *ProposalRepositoryImpl) RejectProposal(ctx context.Context, proposalID uuid.UUID, reviewerID uuid.UUID, reviewNotes string) error {
	db := r.getDB(ctx)

	now := time.Now()
	updates := map[string]interface{}{
		"status":       "rejected",
		"reviewed_by":  reviewerID,
		"reviewed_at":  &now,
		"review_notes": reviewNotes,
	}

	err := db.Model(&models.Proposal{}).Where("id = ?", proposalID).Updates(updates).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to reject proposal")
		return fmt.Errorf("failed to reject proposal: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logrus.Fields{
		"proposal_id": proposalID,
		"reviewer_id": reviewerID,
	}).Info("Proposal rejected")

	return nil
}

// SetUnderReview sets a proposal as under review
func (r *ProposalRepositoryImpl) SetUnderReview(ctx context.Context, proposalID uuid.UUID, reviewerID uuid.UUID) error {
	db := r.getDB(ctx)

	updates := map[string]interface{}{
		"status":      "under_review",
		"reviewed_by": reviewerID,
	}

	err := db.Model(&models.Proposal{}).Where("id = ?", proposalID).Updates(updates).Error
	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to set proposal under review")
		return fmt.Errorf("failed to set proposal under review: %w", err)
	}

	r.logger.WithContext(ctx).WithFields(logrus.Fields{
		"proposal_id": proposalID,
		"reviewer_id": reviewerID,
	}).Info("Proposal set under review")

	return nil
}

// GetProposalStatistics retrieves proposal statistics with optional filters
func (r *ProposalRepositoryImpl) GetProposalStatistics(ctx context.Context, filters Filter) (map[string]interface{}, error) {
	db := r.getDB(ctx)

	// Build base query with filters
	query := db.Model(&models.Proposal{})
	for key, value := range filters {
		if value != nil {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	stats := make(map[string]interface{})

	// Total proposals
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total proposals: %w", err)
	}
	stats["total"] = total

	// Proposals by status
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

	// Proposals by category
	var categoryCounts []struct {
		Category string
		Count    int64
	}
	if err := query.Select("category, COUNT(*) as count").Group("category").Scan(&categoryCounts).Error; err != nil {
		return nil, fmt.Errorf("failed to get category counts: %w", err)
	}

	categoryStats := make(map[string]int64)
	for _, cc := range categoryCounts {
		categoryStats[cc.Category] = cc.Count
	}
	stats["by_category"] = categoryStats

	// Proposals by priority
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

// GetRecentProposals retrieves the most recent proposals
func (r *ProposalRepositoryImpl) GetRecentProposals(ctx context.Context, limit int) ([]models.Proposal, error) {
	db := r.getDB(ctx)
	var proposals []models.Proposal

	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	err := db.Order("created_at DESC").
		Limit(limit).
		Preload("Project").
		Preload("Submitter").
		Preload("Reviewer").
		Find(&proposals).Error

	if err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get recent proposals")
		return nil, fmt.Errorf("failed to get recent proposals: %w", err)
	}

	return proposals, nil
}

// SearchProposals searches proposals by query string with pagination
func (r *ProposalRepositoryImpl) SearchProposals(ctx context.Context, query string, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var proposals []models.Proposal

	searchQuery := r.buildSearchQuery(db, query, []string{"title", "description", "category", "expected_outcome", "justification"})

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := searchQuery.Model(&models.Proposal{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count proposals: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := searchQuery.Order("created_at DESC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Submitter").
		Preload("Reviewer").
		Find(&proposals).Error; err != nil {
		return nil, fmt.Errorf("failed to search proposals: %w", err)
	}

	return &PaginationResult{
		Data:       proposals,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}

// GetProposalsByEffortRange retrieves proposals by estimated effort range with pagination
func (r *ProposalRepositoryImpl) GetProposalsByEffortRange(ctx context.Context, minEffort, maxEffort int, pagination Pagination) (*PaginationResult, error) {
	db := r.getDB(ctx)
	var proposals []models.Proposal

	query := r.buildNumericRangeQuery(db, "estimated_effort", minEffort, maxEffort)

	// Apply pagination
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	var total int64
	if err := query.Model(&models.Proposal{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count proposals: %w", err)
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	if err := query.Order("estimated_effort ASC").
		Limit(pagination.PageSize).
		Offset(offset).
		Preload("Project").
		Preload("Submitter").
		Preload("Reviewer").
		Find(&proposals).Error; err != nil {
		return nil, fmt.Errorf("failed to get proposals by effort range: %w", err)
	}

	return &PaginationResult{
		Data:       proposals,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: int((total + int64(pagination.PageSize) - 1) / int64(pagination.PageSize)),
	}, nil
}
