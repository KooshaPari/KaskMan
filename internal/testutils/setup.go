package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	_ "github.com/lib/pq" // postgres driver
)

// TestConfig holds test configuration
type TestConfig struct {
	DatabaseURL      string
	TestDatabaseName string
	Logger           *logrus.Logger
	CleanupTimeout   time.Duration
}

// TestSuite is the base test suite with database setup/teardown
type TestSuite struct {
	suite.Suite
	DB     *gorm.DB
	Config *TestConfig
	logger *logrus.Logger
}

// TestDatabaseManager manages test database lifecycle
type TestDatabaseManager struct {
	mainDB         *sql.DB
	testDB         *gorm.DB
	testDBName     string
	originalDBName string
	config         *TestConfig
}

// NewTestConfig creates a new test configuration
func NewTestConfig() *TestConfig {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise during tests

	return &TestConfig{
		DatabaseURL:      getEnv("TEST_DATABASE_URL", "postgres://kaskmanager:password@localhost:5432/kaskmanager_test?sslmode=disable"),
		TestDatabaseName: fmt.Sprintf("test_%s_%d", uuid.New().String()[:8], time.Now().Unix()),
		Logger:           logger,
		CleanupTimeout:   30 * time.Second,
	}
}

// SetupSuite initializes the test suite
func (s *TestSuite) SetupSuite() {
	s.Config = NewTestConfig()
	s.logger = s.Config.Logger

	// Create test database manager
	dbManager, err := NewTestDatabaseManager(s.Config)
	if err != nil {
		s.T().Fatalf("Failed to create test database manager: %v", err)
	}

	// Create test database
	err = dbManager.CreateTestDatabase()
	if err != nil {
		s.T().Fatalf("Failed to create test database: %v", err)
	}

	// Connect to test database
	s.DB, err = dbManager.ConnectToTestDatabase()
	if err != nil {
		s.T().Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	err = s.runMigrations()
	if err != nil {
		s.T().Fatalf("Failed to run migrations: %v", err)
	}

	s.logger.Infof("Test database created: %s", s.Config.TestDatabaseName)
}

// TearDownSuite cleans up the test suite
func (s *TestSuite) TearDownSuite() {
	if s.DB != nil {
		sqlDB, err := s.DB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	// Drop test database
	dbManager, err := NewTestDatabaseManager(s.Config)
	if err != nil {
		s.logger.Errorf("Failed to create database manager for cleanup: %v", err)
		return
	}

	err = dbManager.DropTestDatabase()
	if err != nil {
		s.logger.Errorf("Failed to drop test database: %v", err)
	}

	s.logger.Infof("Test database cleaned up: %s", s.Config.TestDatabaseName)
}

// SetupTest runs before each test
func (s *TestSuite) SetupTest() {
	// Clean all tables
	err := s.cleanAllTables()
	if err != nil {
		s.T().Fatalf("Failed to clean tables: %v", err)
	}
}

// TearDownTest runs after each test
func (s *TestSuite) TearDownTest() {
	// Additional cleanup if needed
}

// NewTestDatabaseManager creates a new database manager for tests
func NewTestDatabaseManager(config *TestConfig) (*TestDatabaseManager, error) {
	// Connect to the main database to create test database
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to main database: %w", err)
	}

	return &TestDatabaseManager{
		mainDB:     db,
		testDBName: config.TestDatabaseName,
		config:     config,
	}, nil
}

// CreateTestDatabase creates a new test database
func (m *TestDatabaseManager) CreateTestDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := fmt.Sprintf("CREATE DATABASE %s", m.testDBName)
	_, err := m.mainDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create test database: %w", err)
	}

	return nil
}

// DropTestDatabase drops the test database
func (m *TestDatabaseManager) DropTestDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Terminate all connections to the test database
	terminateQuery := fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()
	`, m.testDBName)

	_, _ = m.mainDB.ExecContext(ctx, terminateQuery)

	// Drop the test database
	query := fmt.Sprintf("DROP DATABASE IF EXISTS %s", m.testDBName)
	_, err := m.mainDB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop test database: %w", err)
	}

	return nil
}

// ConnectToTestDatabase connects to the test database
func (m *TestDatabaseManager) ConnectToTestDatabase() (*gorm.DB, error) {
	// Build test database URL
	testDatabaseURL := fmt.Sprintf("postgres://kaskmanager:password@localhost:5432/%s?sslmode=disable", m.testDBName)

	// Configure GORM for tests
	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Silent, // Reduce noise
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(testDatabaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Configure connection pool for tests
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	m.testDB = db
	return db, nil
}

// runMigrations runs database migrations for tests
func (s *TestSuite) runMigrations() error {
	err := s.DB.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.Agent{},
		&models.Task{},
		&models.Proposal{},
		&models.Pattern{},
		&models.Insight{},
		&models.ActivityLog{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// cleanAllTables cleans all tables in the test database
func (s *TestSuite) cleanAllTables() error {
	tables := []string{
		"activity_logs",
		"insights",
		"patterns",
		"proposals",
		"tasks",
		"agents",
		"projects",
		"users",
	}

	for _, table := range tables {
		if err := s.DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)).Error; err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	return nil
}

// CreateTestContext creates a test context with timeout
func CreateTestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

// WaitForCondition waits for a condition to be true or timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timer.C:
			t.Fatalf("Timeout waiting for condition: %s", message)
		}
	}
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// SetupTestLogger creates a test logger
func SetupTestLogger(level logrus.Level) *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(level)
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	return logger
}
