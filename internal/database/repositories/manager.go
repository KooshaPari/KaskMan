package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RepositoryManagerImpl implements the RepositoryManager interface
type RepositoryManagerImpl struct {
	db                 *gorm.DB
	logger             *logrus.Logger
	cache              CacheManager
	transactionManager *TransactionManagerImpl

	// Repository instances
	userRepo        UserRepository
	projectRepo     ProjectRepository
	taskRepo        TaskRepository
	proposalRepo    ProposalRepository
	agentRepo       AgentRepository
	activityLogRepo ActivityLogRepository
	patternRepo     PatternRepository
	insightRepo     InsightRepository
}

// NewRepositoryManager creates a new repository manager instance
func NewRepositoryManager(db *gorm.DB, logger *logrus.Logger, enableCache bool) RepositoryManager {
	var cache CacheManager
	if enableCache {
		cache = NewInMemoryCacheManager(logger)
	} else {
		cache = NewNoCacheManager()
	}

	manager := &RepositoryManagerImpl{
		db:                 db,
		logger:             logger,
		cache:              cache,
		transactionManager: NewTransactionManager(db, logger),
	}

	// Initialize all repositories
	manager.initializeRepositories()

	return manager
}

// initializeRepositories creates instances of all repositories
func (rm *RepositoryManagerImpl) initializeRepositories() {
	rm.userRepo = NewUserRepository(rm.db, rm.logger, rm.cache)
	rm.projectRepo = NewProjectRepository(rm.db, rm.logger, rm.cache)
	rm.taskRepo = NewTaskRepository(rm.db, rm.logger, rm.cache)
	rm.proposalRepo = NewProposalRepository(rm.db, rm.logger, rm.cache)
	rm.agentRepo = NewAgentRepository(rm.db, rm.logger, rm.cache)
	rm.activityLogRepo = NewActivityLogRepository(rm.db)
	rm.patternRepo = NewPatternRepository(rm.db, rm.logger, rm.cache)
	rm.insightRepo = NewInsightRepository(rm.db, rm.logger, rm.cache)
}

// Repository getters
func (rm *RepositoryManagerImpl) User() UserRepository {
	return rm.userRepo
}

func (rm *RepositoryManagerImpl) Project() ProjectRepository {
	return rm.projectRepo
}

func (rm *RepositoryManagerImpl) Task() TaskRepository {
	return rm.taskRepo
}

func (rm *RepositoryManagerImpl) Proposal() ProposalRepository {
	return rm.proposalRepo
}

func (rm *RepositoryManagerImpl) Agent() AgentRepository {
	return rm.agentRepo
}

func (rm *RepositoryManagerImpl) ActivityLog() ActivityLogRepository {
	return rm.activityLogRepo
}

func (rm *RepositoryManagerImpl) Pattern() PatternRepository {
	return rm.patternRepo
}

func (rm *RepositoryManagerImpl) Insight() InsightRepository {
	return rm.insightRepo
}

// Transaction management
func (rm *RepositoryManagerImpl) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return rm.transactionManager.WithTransaction(ctx, fn)
}

func (rm *RepositoryManagerImpl) BeginTransaction(ctx context.Context) (context.Context, error) {
	return rm.transactionManager.BeginTransaction(ctx)
}

func (rm *RepositoryManagerImpl) CommitTransaction(ctx context.Context) error {
	return rm.transactionManager.CommitTransaction(ctx)
}

func (rm *RepositoryManagerImpl) RollbackTransaction(ctx context.Context) error {
	return rm.transactionManager.RollbackTransaction(ctx)
}

// Health and stats
func (rm *RepositoryManagerImpl) Health() error {
	// Test database connection
	sqlDB, err := rm.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

func (rm *RepositoryManagerImpl) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Database connection stats
	sqlDB, err := rm.db.DB()
	if err == nil {
		dbStats := sqlDB.Stats()
		stats["database"] = map[string]interface{}{
			"open_connections":     dbStats.OpenConnections,
			"in_use":               dbStats.InUse,
			"idle":                 dbStats.Idle,
			"wait_count":           dbStats.WaitCount,
			"wait_duration":        dbStats.WaitDuration.String(),
			"max_idle_closed":      dbStats.MaxIdleClosed,
			"max_idle_time_closed": dbStats.MaxIdleTimeClosed,
			"max_lifetime_closed":  dbStats.MaxLifetimeClosed,
		}
	}

	// Cache stats if available
	if inMemCache, ok := rm.cache.(*InMemoryCacheManager); ok {
		stats["cache"] = inMemCache.GetStats()
	}

	// Repository counts (example - you could add more detailed stats)
	ctx := context.Background()
	stats["repository_counts"] = rm.getRepositoryCounts(ctx)

	return stats
}

// getRepositoryCounts gets count statistics from all repositories
func (rm *RepositoryManagerImpl) getRepositoryCounts(ctx context.Context) map[string]interface{} {
	counts := make(map[string]interface{})

	// Helper function to safely get counts
	safeCount := func(name string, countFunc func() (int64, error)) {
		if count, err := countFunc(); err == nil {
			counts[name] = count
		} else {
			rm.logger.WithError(err).WithField("entity", name).Warn("Failed to get entity count")
			counts[name] = -1 // Indicate error
		}
	}

	// Get counts for each entity type
	safeCount("users", func() (int64, error) {
		return rm.userRepo.Count(ctx, &struct{}{}, Filter{})
	})

	safeCount("projects", func() (int64, error) {
		return rm.projectRepo.Count(ctx, &struct{}{}, Filter{})
	})

	safeCount("tasks", func() (int64, error) {
		return rm.taskRepo.Count(ctx, &struct{}{}, Filter{})
	})

	safeCount("proposals", func() (int64, error) {
		return rm.proposalRepo.Count(ctx, &struct{}{}, Filter{})
	})

	safeCount("agents", func() (int64, error) {
		return rm.agentRepo.Count(ctx, &struct{}{}, Filter{})
	})

	safeCount("activity_logs", func() (int64, error) {
		return rm.activityLogRepo.Count(ctx, &struct{}{}, Filter{})
	})

	safeCount("patterns", func() (int64, error) {
		return rm.patternRepo.Count(ctx, &struct{}{}, Filter{})
	})

	safeCount("insights", func() (int64, error) {
		return rm.insightRepo.Count(ctx, &struct{}{}, Filter{})
	})

	return counts
}

// Migration and maintenance
func (rm *RepositoryManagerImpl) RunMigrations() error {
	rm.logger.Info("Running database migrations")

	// Auto-migrate all models
	err := rm.db.AutoMigrate(
		&struct{}{}, // Add your models here if needed for specific migrations
	)

	if err != nil {
		rm.logger.WithError(err).Error("Failed to run migrations")
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	rm.logger.Info("Database migrations completed successfully")
	return nil
}

func (rm *RepositoryManagerImpl) CleanupOldData(ctx context.Context, config map[string]interface{}) error {
	rm.logger.Info("Starting data cleanup")

	// Extract cleanup configuration
	activityLogRetention := 30 * 24 * time.Hour // Default 30 days
	if retention, ok := config["activity_log_retention"]; ok {
		if d, ok := retention.(time.Duration); ok {
			activityLogRetention = d
		}
	}

	// Cleanup old activity logs
	cutoffTime := time.Now().Add(-activityLogRetention)
	if err := rm.activityLogRepo.CleanupOldActivities(ctx, cutoffTime); err != nil {
		rm.logger.WithError(err).Error("Failed to cleanup activity logs")
		return fmt.Errorf("failed to cleanup activity logs: %w", err)
	}

	// Add more cleanup operations as needed

	rm.logger.Info("Data cleanup completed")
	return nil
}

// Backup and restore
func (rm *RepositoryManagerImpl) CreateBackup(ctx context.Context, path string) error {
	rm.logger.WithField("path", path).Info("Creating database backup")

	// This is a placeholder implementation
	// In a real application, you'd implement proper database backup
	// using pg_dump for PostgreSQL or similar tools for other databases

	return fmt.Errorf("backup functionality not implemented")
}

func (rm *RepositoryManagerImpl) RestoreBackup(ctx context.Context, path string) error {
	rm.logger.WithField("path", path).Info("Restoring database backup")

	// This is a placeholder implementation
	// In a real application, you'd implement proper database restore
	// using pg_restore for PostgreSQL or similar tools for other databases

	return fmt.Errorf("restore functionality not implemented")
}

// Cache operations
func (rm *RepositoryManagerImpl) InvalidateCache(ctx context.Context, patterns ...string) error {
	for _, pattern := range patterns {
		if err := rm.cache.Clear(ctx, pattern); err != nil {
			rm.logger.WithError(err).WithField("pattern", pattern).Error("Failed to invalidate cache pattern")
			return fmt.Errorf("failed to invalidate cache pattern %s: %w", pattern, err)
		}
	}

	rm.logger.WithField("patterns", patterns).Debug("Cache patterns invalidated")
	return nil
}

func (rm *RepositoryManagerImpl) ClearAllCache(ctx context.Context) error {
	if err := rm.cache.Clear(ctx, "*"); err != nil {
		rm.logger.WithError(err).Error("Failed to clear all cache")
		return fmt.Errorf("failed to clear all cache: %w", err)
	}

	rm.logger.Info("All cache cleared")
	return nil
}

// Batch operations
func (rm *RepositoryManagerImpl) BatchExecute(ctx context.Context, operations []func(ctx context.Context) error) error {
	return rm.WithTransaction(ctx, func(txCtx context.Context) error {
		for i, op := range operations {
			if err := op(txCtx); err != nil {
				rm.logger.WithError(err).WithField("operation_index", i).Error("Batch operation failed")
				return fmt.Errorf("batch operation %d failed: %w", i, err)
			}
		}
		return nil
	})
}

// Database connection
func (rm *RepositoryManagerImpl) GetDB() *gorm.DB {
	return rm.db
}

func (rm *RepositoryManagerImpl) Close() error {
	sqlDB, err := rm.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		rm.logger.WithError(err).Error("Failed to close database connection")
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	rm.logger.Info("Database connection closed")
	return nil
}

// Additional utility functions

// GetRepositoryHealth returns health status of all repositories
func (rm *RepositoryManagerImpl) GetRepositoryHealth(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})

	// Test each repository with a simple operation
	testRepo := func(name string, testFunc func() error) {
		if err := testFunc(); err != nil {
			health[name] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			health[name] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}

	// Test user repository
	testRepo("user", func() error {
		_, err := rm.userRepo.Count(ctx, &struct{}{}, Filter{})
		return err
	})

	// Test project repository
	testRepo("project", func() error {
		_, err := rm.projectRepo.Count(ctx, &struct{}{}, Filter{})
		return err
	})

	// Test task repository
	testRepo("task", func() error {
		_, err := rm.taskRepo.Count(ctx, &struct{}{}, Filter{})
		return err
	})

	// Test proposal repository
	testRepo("proposal", func() error {
		_, err := rm.proposalRepo.Count(ctx, &struct{}{}, Filter{})
		return err
	})

	// Test agent repository
	testRepo("agent", func() error {
		_, err := rm.agentRepo.Count(ctx, &struct{}{}, Filter{})
		return err
	})

	// Test activity log repository
	testRepo("activity_log", func() error {
		_, err := rm.activityLogRepo.Count(ctx, &struct{}{}, Filter{})
		return err
	})

	// Test pattern repository
	testRepo("pattern", func() error {
		_, err := rm.patternRepo.Count(ctx, &struct{}{}, Filter{})
		return err
	})

	// Test insight repository
	testRepo("insight", func() error {
		_, err := rm.insightRepo.Count(ctx, &struct{}{}, Filter{})
		return err
	})

	return health
}

// ExecuteInTransaction is a helper for executing multiple repository operations in a transaction
func (rm *RepositoryManagerImpl) ExecuteInTransaction(ctx context.Context, operations ...func(ctx context.Context, repos RepositoryManager) error) error {
	return rm.WithTransaction(ctx, func(txCtx context.Context) error {
		for i, op := range operations {
			if err := op(txCtx, rm); err != nil {
				rm.logger.WithError(err).WithField("operation", i).Error("Transaction operation failed")
				return fmt.Errorf("transaction operation %d failed: %w", i, err)
			}
		}
		return nil
	})
}
