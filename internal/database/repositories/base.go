package repositories

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BaseRepositoryImpl provides a base implementation for common repository operations
type BaseRepositoryImpl struct {
	db     *gorm.DB
	logger *logrus.Logger
	cache  CacheManager
}

// NewBaseRepository creates a new base repository instance
func NewBaseRepository(db *gorm.DB, logger *logrus.Logger, cache CacheManager) *BaseRepositoryImpl {
	return &BaseRepositoryImpl{
		db:     db,
		logger: logger,
		cache:  cache,
	}
}

// getDB returns the database instance from context or falls back to default
func (r *BaseRepositoryImpl) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value("tx").(*gorm.DB); ok {
		return tx
	}
	return r.db
}

// Create creates a new entity in the database
func (r *BaseRepositoryImpl) Create(ctx context.Context, entity interface{}) error {
	db := r.getDB(ctx)

	if err := db.Create(entity).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to create entity")
		return fmt.Errorf("failed to create entity: %w", err)
	}

	r.logger.WithContext(ctx).WithField("entity_type", reflect.TypeOf(entity).String()).Debug("Entity created successfully")
	return nil
}

// GetByID retrieves an entity by its ID
func (r *BaseRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID, entity interface{}) error {
	db := r.getDB(ctx)

	if err := db.First(entity, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("entity with id %s not found", id.String())
		}
		r.logger.WithContext(ctx).WithError(err).Error("Failed to get entity by ID")
		return fmt.Errorf("failed to get entity by ID: %w", err)
	}

	return nil
}

// Update updates an existing entity in the database
func (r *BaseRepositoryImpl) Update(ctx context.Context, entity interface{}) error {
	db := r.getDB(ctx)

	if err := db.Save(entity).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to update entity")
		return fmt.Errorf("failed to update entity: %w", err)
	}

	r.logger.WithContext(ctx).WithField("entity_type", reflect.TypeOf(entity).String()).Debug("Entity updated successfully")
	return nil
}

// Delete permanently deletes an entity from the database
func (r *BaseRepositoryImpl) Delete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	db := r.getDB(ctx)

	if err := db.Unscoped().Delete(entity, "id = ?", id).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to delete entity")
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	r.logger.WithContext(ctx).WithField("entity_id", id.String()).Debug("Entity deleted successfully")
	return nil
}

// SoftDelete soft deletes an entity (marks as deleted but keeps in database)
func (r *BaseRepositoryImpl) SoftDelete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	db := r.getDB(ctx)

	if err := db.Delete(entity, "id = ?", id).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to soft delete entity")
		return fmt.Errorf("failed to soft delete entity: %w", err)
	}

	r.logger.WithContext(ctx).WithField("entity_id", id.String()).Debug("Entity soft deleted successfully")
	return nil
}

// List retrieves all entities with optional filtering
func (r *BaseRepositoryImpl) List(ctx context.Context, entities interface{}, filters Filter) error {
	db := r.getDB(ctx)
	query := db

	// Apply filters
	for key, value := range filters {
		if value != nil {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	if err := query.Find(entities).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to list entities")
		return fmt.Errorf("failed to list entities: %w", err)
	}

	return nil
}

// ListWithPagination retrieves entities with pagination and filtering
func (r *BaseRepositoryImpl) ListWithPagination(ctx context.Context, entities interface{}, pagination Pagination, filters Filter) (*PaginationResult, error) {
	db := r.getDB(ctx)

	// Set default values
	if pagination.Page <= 0 {
		pagination.Page = 1
	}
	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}
	if pagination.PageSize > 100 {
		pagination.PageSize = 100
	}
	if pagination.Sort == "" {
		pagination.Sort = "created_at"
	}
	if pagination.Order == "" {
		pagination.Order = "desc"
	}

	// Build query with filters
	query := db
	for key, value := range filters {
		if value != nil {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	// Get total count
	var total int64
	if err := query.Model(entities).Count(&total).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to count entities")
		return nil, fmt.Errorf("failed to count entities: %w", err)
	}

	// Calculate offset
	offset := (pagination.Page - 1) * pagination.PageSize

	// Apply pagination and sorting
	orderClause := fmt.Sprintf("%s %s", pagination.Sort, pagination.Order)
	if err := query.Order(orderClause).Limit(pagination.PageSize).Offset(offset).Find(entities).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to list entities with pagination")
		return nil, fmt.Errorf("failed to list entities with pagination: %w", err)
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(total) / float64(pagination.PageSize)))

	return &PaginationResult{
		Data:       entities,
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: totalPages,
	}, nil
}

// Count returns the count of entities matching the filters
func (r *BaseRepositoryImpl) Count(ctx context.Context, entity interface{}, filters Filter) (int64, error) {
	db := r.getDB(ctx)
	query := db.Model(entity)

	// Apply filters
	for key, value := range filters {
		if value != nil {
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to count entities")
		return 0, fmt.Errorf("failed to count entities: %w", err)
	}

	return count, nil
}

// Exists checks if an entity exists by ID
func (r *BaseRepositoryImpl) Exists(ctx context.Context, id uuid.UUID, entity interface{}) (bool, error) {
	db := r.getDB(ctx)

	var count int64
	if err := db.Model(entity).Where("id = ?", id).Count(&count).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to check entity existence")
		return false, fmt.Errorf("failed to check entity existence: %w", err)
	}

	return count > 0, nil
}

// BatchCreate creates multiple entities in a single transaction
func (r *BaseRepositoryImpl) BatchCreate(ctx context.Context, entities interface{}) error {
	db := r.getDB(ctx)

	// Use CreateInBatches for better performance with large datasets
	if err := db.CreateInBatches(entities, 100).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to batch create entities")
		return fmt.Errorf("failed to batch create entities: %w", err)
	}

	r.logger.WithContext(ctx).WithField("entity_type", reflect.TypeOf(entities).String()).Debug("Entities batch created successfully")
	return nil
}

// BatchUpdate updates multiple entities in a single transaction
func (r *BaseRepositoryImpl) BatchUpdate(ctx context.Context, entities interface{}) error {
	db := r.getDB(ctx)

	// Use Clauses with OnConflict for upsert behavior
	if err := db.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(entities).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to batch update entities")
		return fmt.Errorf("failed to batch update entities: %w", err)
	}

	r.logger.WithContext(ctx).WithField("entity_type", reflect.TypeOf(entities).String()).Debug("Entities batch updated successfully")
	return nil
}

// BatchDelete deletes multiple entities by IDs
func (r *BaseRepositoryImpl) BatchDelete(ctx context.Context, ids []uuid.UUID, entity interface{}) error {
	db := r.getDB(ctx)

	if len(ids) == 0 {
		return nil
	}

	if err := db.Delete(entity, "id IN ?", ids).Error; err != nil {
		r.logger.WithContext(ctx).WithError(err).Error("Failed to batch delete entities")
		return fmt.Errorf("failed to batch delete entities: %w", err)
	}

	r.logger.WithContext(ctx).WithField("count", len(ids)).Debug("Entities batch deleted successfully")
	return nil
}

// buildSearchQuery builds a query for text search across specified fields
func (r *BaseRepositoryImpl) buildSearchQuery(db *gorm.DB, query string, fields []string) *gorm.DB {
	if query == "" || len(fields) == 0 {
		return db
	}

	// Build OR conditions for each field
	conditions := make([]string, len(fields))
	args := make([]interface{}, len(fields))

	for i, field := range fields {
		conditions[i] = fmt.Sprintf("%s ILIKE ?", field)
		args[i] = "%" + strings.ToLower(query) + "%"
	}

	whereClause := strings.Join(conditions, " OR ")
	return db.Where(whereClause, args...)
}

// buildDateRangeQuery builds a query for date range filtering
func (r *BaseRepositoryImpl) buildDateRangeQuery(db *gorm.DB, field string, startDate, endDate *time.Time) *gorm.DB {
	if startDate != nil {
		db = db.Where(fmt.Sprintf("%s >= ?", field), startDate)
	}
	if endDate != nil {
		db = db.Where(fmt.Sprintf("%s <= ?", field), endDate)
	}
	return db
}

// buildNumericRangeQuery builds a query for numeric range filtering
func (r *BaseRepositoryImpl) buildNumericRangeQuery(db *gorm.DB, field string, min, max interface{}) *gorm.DB {
	if min != nil {
		db = db.Where(fmt.Sprintf("%s >= ?", field), min)
	}
	if max != nil {
		db = db.Where(fmt.Sprintf("%s <= ?", field), max)
	}
	return db
}

// preloadRelations preloads specified relations for better performance
func (r *BaseRepositoryImpl) preloadRelations(db *gorm.DB, relations []string) *gorm.DB {
	for _, relation := range relations {
		db = db.Preload(relation)
	}
	return db
}

// getCacheKey generates a cache key for repository operations
func (r *BaseRepositoryImpl) getCacheKey(prefix string, params ...interface{}) string {
	parts := []string{prefix}
	for _, param := range params {
		parts = append(parts, fmt.Sprintf("%v", param))
	}
	return strings.Join(parts, ":")
}

// TransactionManagerImpl provides transaction management
type TransactionManagerImpl struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(db *gorm.DB, logger *logrus.Logger) *TransactionManagerImpl {
	return &TransactionManagerImpl{
		db:     db,
		logger: logger,
	}
}

// WithTransaction executes a function within a database transaction
func (tm *TransactionManagerImpl) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx := tm.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			tm.logger.WithContext(ctx).WithField("panic", r).Error("Transaction rolled back due to panic")
		}
	}()

	// Add transaction to context
	txCtx := context.WithValue(ctx, "tx", tx)

	if err := fn(txCtx); err != nil {
		tx.Rollback()
		tm.logger.WithContext(ctx).WithError(err).Error("Transaction rolled back due to error")
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		tm.logger.WithContext(ctx).WithError(err).Error("Failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tm.logger.WithContext(ctx).Debug("Transaction committed successfully")
	return nil
}

// BeginTransaction begins a new transaction and returns a context with the transaction
func (tm *TransactionManagerImpl) BeginTransaction(ctx context.Context) (context.Context, error) {
	tx := tm.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}

	return context.WithValue(ctx, "tx", tx), nil
}

// CommitTransaction commits the transaction in the context
func (tm *TransactionManagerImpl) CommitTransaction(ctx context.Context) error {
	tx, ok := ctx.Value("tx").(*gorm.DB)
	if !ok {
		return fmt.Errorf("no transaction found in context")
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tm.logger.WithContext(ctx).Debug("Transaction committed successfully")
	return nil
}

// RollbackTransaction rolls back the transaction in the context
func (tm *TransactionManagerImpl) RollbackTransaction(ctx context.Context) error {
	tx, ok := ctx.Value("tx").(*gorm.DB)
	if !ok {
		return fmt.Errorf("no transaction found in context")
	}

	if err := tx.Rollback().Error; err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	tm.logger.WithContext(ctx).Debug("Transaction rolled back successfully")
	return nil
}
