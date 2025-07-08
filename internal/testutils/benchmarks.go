package testutils

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/config"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// PerformanceBenchmarks contains performance benchmark tests
type PerformanceBenchmarks struct {
	suite    *TestSuite
	db       *database.Database
	fixtures *TestFixtures
	logger   *logrus.Logger
}

// NewPerformanceBenchmarks creates a new performance benchmark suite
func NewPerformanceBenchmarks(suite *TestSuite) *PerformanceBenchmarks {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce logging noise during benchmarks

	return &PerformanceBenchmarks{
		suite:    suite,
		db:       &database.Database{DB: suite.DB},
		fixtures: NewTestFixtures(suite.DB),
		logger:   logger,
	}
}

// BenchmarkDatabaseOperations benchmarks core database operations
func (pb *PerformanceBenchmarks) BenchmarkDatabaseOperations(b *testing.B) {
	userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
	projectRepo := repositories.NewProjectRepository(pb.suite.DB, pb.logger, nil)
	taskRepo := repositories.NewTaskRepository(pb.suite.DB, pb.logger, nil)

	// Create test user
	testUser := pb.fixtures.CreateUser()
	ctx := context.Background()

	b.Run("UserRepository_Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			user := &models.User{
				BaseModel: models.BaseModel{
					ID: uuid.New(),
				},
				Username:     fmt.Sprintf("bench_user_%d", i),
				Email:        fmt.Sprintf("bench_user_%d@example.com", i),
				PasswordHash: "$2a$10$hashedpassword",
				Role:         "user",
				IsActive:   true,
			}
			err := userRepo.Create(ctx, user)
			require.NoError(b, err)
		}
	})

	b.Run("UserRepository_GetByID", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var user models.User
			err := userRepo.GetByID(ctx, testUser.ID, &user)
			require.NoError(b, err)
		}
	})

	b.Run("ProjectRepository_Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			project := &models.Project{
				BaseModel: models.BaseModel{
					ID: uuid.New(),
				},
				Name:        fmt.Sprintf("Bench Project %d", i),
				Description: fmt.Sprintf("Benchmark project %d", i),
				Type:        "research",
				Status:      "active",
				Priority:    "medium",
				CreatedBy:   testUser.ID,
				StartDate:   func() *time.Time { t := time.Now(); return &t }(),
			}
			err := projectRepo.Create(ctx, project)
			require.NoError(b, err)
		}
	})

	// Create test project for task benchmarks
	testProject := pb.fixtures.CreateProject(testUser.ID)

	b.Run("TaskRepository_Create", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			task := &models.Task{
				BaseModel: models.BaseModel{
					ID: uuid.New(),
				},
				Title:       fmt.Sprintf("Bench Task %d", i),
				Description: fmt.Sprintf("Benchmark task %d", i),
				Type:        "development",
				Status:      "pending",
				Priority:    "medium",
				ProjectID:   &testProject.ID,
			}
			err := taskRepo.Create(ctx, task)
			require.NoError(b, err)
		}
	})
}

// BenchmarkRepositoryQueries benchmarks complex repository queries
func (pb *PerformanceBenchmarks) BenchmarkRepositoryQueries(b *testing.B) {
	userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
	projectRepo := repositories.NewProjectRepository(pb.suite.DB, pb.logger, nil)
	taskRepo := repositories.NewTaskRepository(pb.suite.DB, pb.logger, nil)

	// Create test data
	testUser := pb.fixtures.CreateUser()
	testProject := pb.fixtures.CreateProject(testUser.ID)

	// Create multiple tasks for query benchmarks
	for i := 0; i < 100; i++ {
		pb.fixtures.CreateTask(testProject.ID, map[string]interface{}{
			"title":    fmt.Sprintf("Query Test Task %d", i),
			"status":   []string{"pending", "in_progress", "completed"}[i%3],
			"priority": []string{"low", "medium", "high"}[i%3],
		})
	}

	ctx := context.Background()
	pagination := repositories.Pagination{Page: 1, PageSize: 20}

	b.Run("UserRepository_SearchUsers", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := userRepo.SearchUsers(ctx, "test", pagination)
			require.NoError(b, err)
		}
	})

	b.Run("ProjectRepository_GetByStatus", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := projectRepo.GetByStatus(ctx, "active", pagination)
			require.NoError(b, err)
		}
	})

	b.Run("TaskRepository_GetByProject", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := taskRepo.GetByProject(ctx, testProject.ID, pagination)
			require.NoError(b, err)
		}
	})

	b.Run("TaskRepository_GetByStatus", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := taskRepo.GetByStatus(ctx, "pending", pagination)
			require.NoError(b, err)
		}
	})

	b.Run("TaskRepository_SearchTasks", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := taskRepo.SearchTasks(ctx, "Query Test", pagination)
			require.NoError(b, err)
		}
	})
}

// BenchmarkConcurrentOperations benchmarks concurrent database operations
func (pb *PerformanceBenchmarks) BenchmarkConcurrentOperations(b *testing.B) {
	userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
	ctx := context.Background()

	b.Run("ConcurrentUserCreation", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				user := &models.User{
					BaseModel: models.BaseModel{
						ID: uuid.New(),
					},
					Username:     fmt.Sprintf("concurrent_user_%d_%d", b.N, i),
					Email:        fmt.Sprintf("concurrent_user_%d_%d@example.com", b.N, i),
					PasswordHash: "$2a$10$hashedpassword",
					Role:         "user",
					IsActive:   true,
				}
				err := userRepo.Create(ctx, user)
				require.NoError(b, err)
				i++
			}
		})
	})

	// Create test user for concurrent reads
	testUser := pb.fixtures.CreateUser()

	b.Run("ConcurrentUserReads", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var user models.User
				err := userRepo.GetByID(ctx, testUser.ID, &user)
				require.NoError(b, err)
			}
		})
	})
}

// BenchmarkRnDModule benchmarks R&D module operations
func (pb *PerformanceBenchmarks) BenchmarkRnDModule(b *testing.B) {
	rndConfig := config.RnDConfig{
		Enabled:              true,
		WorkerCount:          4,
		QueueSize:            1000,
		ProcessingTimeout:    30 * time.Second,
		LearningInterval:     1 * time.Hour, // Long interval for benchmarks
		PatternAnalysisDepth: 10,
		ProjectGenerationMax: 5,
		CoordinationMode:     "centralized",
		AgentMaxCount:        10,
	}

	module, err := rnd.NewModule(rndConfig, pb.db, pb.logger)
	require.NoError(b, err)

	err = module.Start()
	require.NoError(b, err)
	defer module.Stop()

	// Wait for module to initialize
	time.Sleep(100 * time.Millisecond)

	b.Run("ProcessTask", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			task := map[string]interface{}{
				"id":   fmt.Sprintf("bench_task_%d", i),
				"type": "research",
				"data": fmt.Sprintf("benchmark data %d", i),
			}
			err := module.ProcessTask(task)
			require.NoError(b, err)
		}
	})

	b.Run("GenerateInsights", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := module.GenerateInsights()
			require.NoError(b, err)
		}
	})

	b.Run("AnalyzePatterns", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := module.AnalyzePatterns()
			require.NoError(b, err)
		}
	})

	b.Run("CoordinateAgents", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := module.CoordinateAgents()
			require.NoError(b, err)
		}
	})
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func (pb *PerformanceBenchmarks) BenchmarkMemoryUsage(b *testing.B) {
	b.Run("UserCreationMemory", func(b *testing.B) {
		userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
		ctx := context.Background()

		// Measure memory before
		var m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m1)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			user := &models.User{
				BaseModel: models.BaseModel{
					ID: uuid.New(),
				},
				Username:     fmt.Sprintf("memory_user_%d", i),
				Email:        fmt.Sprintf("memory_user_%d@example.com", i),
				PasswordHash: "$2a$10$hashedpassword",
				Role:         "user",
				IsActive:   true,
			}
			err := userRepo.Create(ctx, user)
			require.NoError(b, err)
		}
		b.StopTimer()

		// Measure memory after
		var m2 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m2)

		// Report memory usage
		allocatedBytes := m2.Alloc - m1.Alloc
		b.ReportMetric(float64(allocatedBytes)/float64(b.N), "bytes/op")
	})
}

// BenchmarkBatchOperations benchmarks batch database operations
func (pb *PerformanceBenchmarks) BenchmarkBatchOperations(b *testing.B) {
	userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
	ctx := context.Background()

	b.Run("BatchUserCreation", func(b *testing.B) {
		batchSizes := []int{10, 50, 100, 500}

		for _, batchSize := range batchSizes {
			b.Run(fmt.Sprintf("BatchSize_%d", batchSize), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					users := make([]*models.User, batchSize)
					for j := 0; j < batchSize; j++ {
						users[j] = &models.User{
							BaseModel: models.BaseModel{ID: uuid.New()},
							Username: fmt.Sprintf("batch_user_%d_%d", i, j),
							Email:    fmt.Sprintf("batch_user_%d_%d@example.com", i, j),
							PasswordHash: "$2a$10$hashedpassword",
							Role:     "user",
							IsActive: true,
						}
					}

					// Simulate batch creation
					for _, user := range users {
						err := userRepo.Create(ctx, user)
						require.NoError(b, err)
					}
				}
			})
		}
	})
}

// BenchmarkComplexQueries benchmarks complex database queries
func (pb *PerformanceBenchmarks) BenchmarkComplexQueries(b *testing.B) {
	// Create substantial test data
	users := make([]*models.User, 100)
	projects := make([]*models.Project, 200)

	for i := 0; i < 100; i++ {
		users[i] = pb.fixtures.CreateUser(map[string]interface{}{
			"username": fmt.Sprintf("complex_user_%d", i),
			"email":    fmt.Sprintf("complex_user_%d@example.com", i),
		})
	}

	for i := 0; i < 200; i++ {
		userIndex := i % 100
		projects[i] = pb.fixtures.CreateProject(users[userIndex].ID, map[string]interface{}{
			"name":        fmt.Sprintf("Complex Project %d", i),
			"description": fmt.Sprintf("Complex project description %d", i),
			"status":      []string{"active", "completed", "on_hold"}[i%3],
			"priority":    []string{"low", "medium", "high"}[i%3],
		})
	}

	// Create tasks for projects
	for i := 0; i < 500; i++ {
		projectIndex := i % 200
		pb.fixtures.CreateTask(projects[projectIndex].ID, map[string]interface{}{
			"title":    fmt.Sprintf("Complex Task %d", i),
			"status":   []string{"pending", "in_progress", "completed"}[i%3],
			"priority": []string{"low", "medium", "high"}[i%3],
		})
	}

	userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
	projectRepo := repositories.NewProjectRepository(pb.suite.DB, pb.logger, nil)
	taskRepo := repositories.NewTaskRepository(pb.suite.DB, pb.logger, nil)

	ctx := context.Background()
	pagination := repositories.Pagination{Page: 1, PageSize: 50}

	b.Run("UserStatistics", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			userIndex := i % 100
			_, err := userRepo.GetUserStatistics(ctx, users[userIndex].ID)
			require.NoError(b, err)
		}
	})

	b.Run("ProjectStatistics", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			projectIndex := i % 200
			_, err := projectRepo.GetProjectStatistics(ctx, projects[projectIndex].ID)
			require.NoError(b, err)
		}
	})

	b.Run("TaskStatistics", func(b *testing.B) {
		filters := repositories.Filter{"status": "completed"}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := taskRepo.GetTaskStatistics(ctx, filters)
			require.NoError(b, err)
		}
	})

	b.Run("ComplexProjectQuery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := projectRepo.GetProjectsWithTaskCounts(ctx, pagination)
			require.NoError(b, err)
		}
	})
}

// BenchmarkHighLoad simulates high load scenarios
func (pb *PerformanceBenchmarks) BenchmarkHighLoad(b *testing.B) {
	userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
	projectRepo := repositories.NewProjectRepository(pb.suite.DB, pb.logger, nil)
	taskRepo := repositories.NewTaskRepository(pb.suite.DB, pb.logger, nil)

	// Create base test data
	testUser := pb.fixtures.CreateUser()
	testProject := pb.fixtures.CreateProject(testUser.ID)

	ctx := context.Background()

	b.Run("HighLoadMixedOperations", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				operation := i % 4

				switch operation {
				case 0: // Create user
					user := &models.User{
						BaseModel: models.BaseModel{ID: uuid.New()},
						Username: fmt.Sprintf("load_user_%d", i),
						Email:    fmt.Sprintf("load_user_%d@example.com", i),
						PasswordHash: "$2a$10$hashedpassword",
						Role:     "user",
						IsActive: true,
					}
					userRepo.Create(ctx, user)

				case 1: // Read user
					var user models.User
					userRepo.GetByID(ctx, testUser.ID, &user)

				case 2: // Create task
					task := &models.Task{
						BaseModel: models.BaseModel{
							ID: uuid.New(),
						},
						Title:       fmt.Sprintf("Load Task %d", i),
						Description: fmt.Sprintf("High load task %d", i),
						Type:        "development",
						Status:      "pending",
						Priority:    "medium",
						ProjectID:   &testProject.ID,
					}
					taskRepo.Create(ctx, task)

				case 3: // Query tasks
					pagination := repositories.Pagination{Page: 1, PageSize: 10}
					projectRepo.GetByID(ctx, testProject.ID, &models.Project{})
					taskRepo.GetByProject(ctx, testProject.ID, pagination)
				}
				i++
			}
		})
	})
}

// ComparativeMetrics holds metrics for comparing with Node.js implementation
type ComparativeMetrics struct {
	OperationsPerSecond float64
	AverageResponseTime time.Duration
	MemoryUsageBytes    uint64
	CPUUsagePercent     float64
	ConcurrentUsers     int
	ErrorRate           float64
}

// BenchmarkComparativePerformance creates benchmark metrics for comparison with Node.js
func (pb *PerformanceBenchmarks) BenchmarkComparativePerformance(b *testing.B) {
	userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
	ctx := context.Background()

	// Simulate typical API operations
	b.Run("TypicalWorkload", func(b *testing.B) {
		startTime := time.Now()
		var totalMemory uint64

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate typical user operation (70% reads, 30% writes)
			if i%10 < 7 {
				// Read operation
				testUser := pb.fixtures.CreateUser()
				var user models.User
				err := userRepo.GetByID(ctx, testUser.ID, &user)
				require.NoError(b, err)
			} else {
				// Write operation
				user := &models.User{
					BaseModel: models.BaseModel{ID: uuid.New()},
					Username: fmt.Sprintf("typical_user_%d", i),
					Email:    fmt.Sprintf("typical_user_%d@example.com", i),
					PasswordHash: "$2a$10$hashedpassword",
					Role:     "user",
					IsActive: true,
				}
				err := userRepo.Create(ctx, user)
				require.NoError(b, err)
			}

			// Sample memory usage periodically
			if i%100 == 0 {
				var m runtime.MemStats
				runtime.ReadMemStats(&m)
				totalMemory += m.Alloc
			}
		}
		b.StopTimer()

		duration := time.Since(startTime)
		opsPerSecond := float64(b.N) / duration.Seconds()
		avgResponseTime := duration / time.Duration(b.N)
		avgMemory := totalMemory / uint64(b.N/100+1)

		// Report comparative metrics
		b.ReportMetric(opsPerSecond, "ops/sec")
		b.ReportMetric(float64(avgResponseTime.Nanoseconds()), "ns/op")
		b.ReportMetric(float64(avgMemory), "bytes/op")

		// Log detailed metrics for comparison
		pb.logger.Infof("Comparative Metrics - Ops/sec: %.2f, Avg Response: %v, Avg Memory: %d bytes",
			opsPerSecond, avgResponseTime, avgMemory)
	})
}

// BenchmarkScalability tests scalability characteristics
func (pb *PerformanceBenchmarks) BenchmarkScalability(b *testing.B) {
	userRepo := repositories.NewUserRepository(pb.suite.DB, pb.logger, nil)
	ctx := context.Background()

	// Test different levels of concurrency
	concurrencyLevels := []int{1, 2, 4, 8, 16, 32, 64}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			b.SetParallelism(concurrency)
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					user := &models.User{
						BaseModel: models.BaseModel{ID: uuid.New()},
						Username: fmt.Sprintf("scale_user_%d_%d", concurrency, i),
						Email:    fmt.Sprintf("scale_user_%d_%d@example.com", concurrency, i),
						PasswordHash: "$2a$10$hashedpassword",
						Role:     "user",
						IsActive: true,
					}
					err := userRepo.Create(ctx, user)
					require.NoError(b, err)
					i++
				}
			})
		})
	}
}

// Helper function to run all benchmarks
func RunAllBenchmarks(b *testing.B, suite *TestSuite) {
	benchmarks := NewPerformanceBenchmarks(suite)

	b.Run("DatabaseOperations", benchmarks.BenchmarkDatabaseOperations)
	b.Run("RepositoryQueries", benchmarks.BenchmarkRepositoryQueries)
	b.Run("ConcurrentOperations", benchmarks.BenchmarkConcurrentOperations)
	b.Run("RnDModule", benchmarks.BenchmarkRnDModule)
	b.Run("MemoryUsage", benchmarks.BenchmarkMemoryUsage)
	b.Run("BatchOperations", benchmarks.BenchmarkBatchOperations)
	b.Run("ComplexQueries", benchmarks.BenchmarkComplexQueries)
	b.Run("HighLoad", benchmarks.BenchmarkHighLoad)
	b.Run("ComparativePerformance", benchmarks.BenchmarkComparativePerformance)
	b.Run("Scalability", benchmarks.BenchmarkScalability)
}
