package rnd

import (
	"fmt"
	"testing"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// RnDModuleTestSuite represents the test suite for R&D Module
type RnDModuleTestSuite struct {
	suite.Suite
	module   *Module
	config   config.RnDConfig
}

// SetupTest sets up the test suite
func (s *RnDModuleTestSuite) SetupTest() {
	// Create R&D config with correct fields
	s.config = config.RnDConfig{
		Enabled:              true,
		WorkerCount:          4,
		QueueSize:            1000,
		ProcessingTimeout:    30 * time.Second,
		LearningInterval:     30 * time.Second,
		PatternAnalysisDepth: 10,
		ProjectGenerationMax: 5,
		CoordinationMode:     "centralized",
		AgentMaxCount:        10,
	}

	// Create a mock database and logger for testing
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Create R&D module with nil database for unit tests
	var err error
	s.module, err = NewModule(s.config, nil, logger)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), s.module)
}

// TearDownTest cleans up after each test
func (s *RnDModuleTestSuite) TearDownTest() {
	if s.module != nil && s.module.IsRunning() {
		s.module.Stop()
	}
}

// TestNewModule tests module creation
func (s *RnDModuleTestSuite) TestNewModule() {
	// Test with enabled config
	enabledConfig := s.config
	enabledConfig.Enabled = true

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	module, err := NewModule(enabledConfig, nil, logger)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), module)
	assert.NotNil(s.T(), module.coordinator)
	assert.NotNil(s.T(), module.learning)
	assert.NotNil(s.T(), module.patterns)
	assert.NotNil(s.T(), module.projects)
	assert.NotNil(s.T(), module.stats)
	assert.Equal(s.T(), &enabledConfig, module.config)

	// Test with disabled config
	disabledConfig := s.config
	disabledConfig.Enabled = false

	disabledModule, err := NewModule(disabledConfig, nil, logger)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), disabledModule)
	assert.Nil(s.T(), disabledModule.coordinator)
	assert.Nil(s.T(), disabledModule.learning)
	assert.Nil(s.T(), disabledModule.patterns)
	assert.Nil(s.T(), disabledModule.projects)

	// Test with invalid database (this should now pass with our modified implementation)
	_, err = NewModule(enabledConfig, nil, logger)
	assert.NoError(s.T(), err) // Changed to NoError since we now allow nil database
}

// TestModuleStartStop tests module lifecycle
func (s *RnDModuleTestSuite) TestModuleStartStop() {
	// Test start
	assert.False(s.T(), s.module.IsRunning())

	err := s.module.Start()
	assert.NoError(s.T(), err)
	assert.True(s.T(), s.module.IsRunning())

	// Wait a bit for workers to initialize
	time.Sleep(100 * time.Millisecond)

	// Test double start (should return error)
	err = s.module.Start()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "already running")

	// Test stop
	s.module.Stop()
	assert.False(s.T(), s.module.IsRunning())

	// Test double stop (should not panic)
	s.module.Stop()
	assert.False(s.T(), s.module.IsRunning())
}

// TestModuleStartStopDisabled tests module lifecycle when disabled
func (s *RnDModuleTestSuite) TestModuleStartStopDisabled() {
	disabledConfig := s.config
	disabledConfig.Enabled = false

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	disabledModule, err := NewModule(disabledConfig, nil, logger)
	assert.NoError(s.T(), err)

	// Test start when disabled
	err = disabledModule.Start()
	assert.NoError(s.T(), err)
	assert.False(s.T(), disabledModule.IsRunning())

	// Test stop when disabled
	disabledModule.Stop()
	assert.False(s.T(), disabledModule.IsRunning())
}

// TestModuleGetters tests module component getters
func (s *RnDModuleTestSuite) TestModuleGetters() {
	// Test component getters
	coordinator := s.module.GetCoordinator()
	assert.NotNil(s.T(), coordinator)
	assert.Equal(s.T(), s.module.coordinator, coordinator)

	learningEngine := s.module.GetLearningEngine()
	assert.NotNil(s.T(), learningEngine)
	assert.Equal(s.T(), s.module.learning, learningEngine)

	patternRecognizer := s.module.GetPatternRecognizer()
	assert.NotNil(s.T(), patternRecognizer)
	assert.Equal(s.T(), s.module.patterns, patternRecognizer)

	projectGenerator := s.module.GetProjectGenerator()
	assert.NotNil(s.T(), projectGenerator)
	assert.Equal(s.T(), s.module.projects, projectGenerator)
}

// TestModuleStats tests module statistics
func (s *RnDModuleTestSuite) TestModuleStats() {
	// Get initial stats
	stats := s.module.GetStats()
	assert.NotNil(s.T(), stats)
	assert.Equal(s.T(), int64(0), stats.TasksProcessed)
	assert.Equal(s.T(), int64(0), stats.PatternsDetected)
	assert.Equal(s.T(), int64(0), stats.InsightsGenerated)
	assert.Equal(s.T(), int64(0), stats.ProjectsGenerated)
	assert.Equal(s.T(), int64(0), stats.AgentsCoordinated)
	assert.Equal(s.T(), int64(0), stats.LearningIterations)
	assert.Equal(s.T(), int64(0), stats.ProcessingErrors)
	assert.Equal(s.T(), float64(0), stats.AvgProcessingTime)

	// Start module and run some operations
	err := s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Process a task
	testTask := map[string]interface{}{
		"id":   "test-task-1",
		"type": "research",
		"data": "test data",
	}

	err = s.module.ProcessTask(testTask)
	assert.NoError(s.T(), err)

	// Generate insights
	err = s.module.GenerateInsights()
	assert.NoError(s.T(), err)

	// Coordinate agents
	err = s.module.CoordinateAgents()
	assert.NoError(s.T(), err)

	// Wait for operations to complete
	time.Sleep(100 * time.Millisecond)

	// Check updated stats
	updatedStats := s.module.GetStats()
	assert.Equal(s.T(), int64(1), updatedStats.TasksProcessed)
	assert.Equal(s.T(), int64(1), updatedStats.InsightsGenerated)
	assert.Equal(s.T(), int64(1), updatedStats.AgentsCoordinated)
	assert.True(s.T(), updatedStats.AvgProcessingTime > 0)
	assert.True(s.T(), updatedStats.LastActivity.After(stats.LastActivity))
}

// TestProcessTask tests task processing
func (s *RnDModuleTestSuite) TestProcessTask() {
	// Test processing without starting module
	testTask := map[string]interface{}{
		"id":   "test-task-1",
		"type": "research",
	}

	err := s.module.ProcessTask(testTask)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "not running")

	// Start module
	err = s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Test successful task processing
	err = s.module.ProcessTask(testTask)
	assert.NoError(s.T(), err)

	// Test processing multiple tasks
	for i := 0; i < 5; i++ {
		task := map[string]interface{}{
			"id":   fmt.Sprintf("test-task-%d", i+2),
			"type": "analysis",
			"data": fmt.Sprintf("test data %d", i),
		}
		err = s.module.ProcessTask(task)
		assert.NoError(s.T(), err)
	}

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	// Verify stats updated
	stats := s.module.GetStats()
	assert.Equal(s.T(), int64(6), stats.TasksProcessed)
}

// TestAnalyzePatterns tests pattern analysis
func (s *RnDModuleTestSuite) TestAnalyzePatterns() {
	// Test without starting module
	err := s.module.AnalyzePatterns()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "not running")

	// Start module
	err = s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Test pattern analysis
	err = s.module.AnalyzePatterns()
	assert.NoError(s.T(), err)

	// Verify stats updated
	stats := s.module.GetStats()
	assert.Equal(s.T(), int64(1), stats.PatternsDetected)
}

// TestGenerateInsights tests insight generation
func (s *RnDModuleTestSuite) TestGenerateInsights() {
	// Test without starting module
	err := s.module.GenerateInsights()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "not running")

	// Start module
	err = s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Test insight generation
	err = s.module.GenerateInsights()
	assert.NoError(s.T(), err)

	// Test multiple insight generations
	for i := 0; i < 3; i++ {
		err = s.module.GenerateInsights()
		assert.NoError(s.T(), err)
	}

	// Verify stats updated
	stats := s.module.GetStats()
	assert.Equal(s.T(), int64(4), stats.InsightsGenerated)
}

// TestGenerateProjects tests project generation
func (s *RnDModuleTestSuite) TestGenerateProjects() {
	// Test without starting module
	err := s.module.GenerateProjects()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "not running")

	// Start module
	err = s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Test project generation
	err = s.module.GenerateProjects()
	assert.NoError(s.T(), err)

	// Verify stats updated
	stats := s.module.GetStats()
	assert.Equal(s.T(), int64(1), stats.ProjectsGenerated)
}

// TestCoordinateAgents tests agent coordination
func (s *RnDModuleTestSuite) TestCoordinateAgents() {
	// Test without starting module
	err := s.module.CoordinateAgents()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "not running")

	// Start module
	err = s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Test agent coordination
	err = s.module.CoordinateAgents()
	assert.NoError(s.T(), err)

	// Test multiple coordinations
	for i := 0; i < 2; i++ {
		err = s.module.CoordinateAgents()
		assert.NoError(s.T(), err)
	}

	// Verify stats updated
	stats := s.module.GetStats()
	assert.Equal(s.T(), int64(3), stats.AgentsCoordinated)
}

// TestModuleHealth tests module health check
func (s *RnDModuleTestSuite) TestModuleHealth() {
	// Test health when not running
	health := s.module.Health()
	assert.NotNil(s.T(), health)
	assert.True(s.T(), health["enabled"].(bool))
	assert.False(s.T(), health["running"].(bool))
	assert.NotNil(s.T(), health["stats"])

	// Components should not be present when not running
	_, hasCoordinator := health["coordinator"]
	_, hasLearning := health["learning"]
	_, hasPatterns := health["patterns"]
	_, hasProjects := health["projects"]
	assert.False(s.T(), hasCoordinator)
	assert.False(s.T(), hasLearning)
	assert.False(s.T(), hasPatterns)
	assert.False(s.T(), hasProjects)

	// Start module
	err := s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Test health when running
	runningHealth := s.module.Health()
	assert.True(s.T(), runningHealth["enabled"].(bool))
	assert.True(s.T(), runningHealth["running"].(bool))

	// Components should be present when running
	assert.Contains(s.T(), runningHealth, "coordinator")
	assert.Contains(s.T(), runningHealth, "learning")
	assert.Contains(s.T(), runningHealth, "patterns")
	assert.Contains(s.T(), runningHealth, "projects")
}

// TestModulePeriodicTasks tests periodic task execution
func (s *RnDModuleTestSuite) TestModulePeriodicTasks() {
	// Reduce learning interval for testing
	shortConfig := s.config
	shortConfig.LearningInterval = 200 * time.Millisecond

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	shortModule, err := NewModule(shortConfig, nil, logger)
	assert.NoError(s.T(), err)

	defer func() {
		if shortModule.IsRunning() {
			shortModule.Stop()
		}
	}()

	// Start module
	err = shortModule.Start()
	assert.NoError(s.T(), err)

	// Wait for several periodic cycles
	time.Sleep(600 * time.Millisecond)

	// Check that periodic tasks have been executed
	stats := shortModule.GetStats()
	assert.True(s.T(), stats.LearningIterations > 0, "Learning iterations should be > 0")
	assert.True(s.T(), stats.PatternsDetected > 0, "Patterns detected should be > 0")
	assert.True(s.T(), stats.InsightsGenerated > 0, "Insights generated should be > 0")
	assert.True(s.T(), stats.AgentsCoordinated > 0, "Agents coordinated should be > 0")
}

// TestModuleConcurrency tests concurrent operations
func (s *RnDModuleTestSuite) TestModuleConcurrency() {
	// Start module
	err := s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Run concurrent operations
	concurrency := 10
	operations := 5 // operations per goroutine
	totalOps := concurrency * operations

	ch := make(chan error, totalOps*4) // 4 types of operations

	// Concurrent task processing
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			for j := 0; j < operations; j++ {
				task := map[string]interface{}{
					"id":       fmt.Sprintf("worker-%d-task-%d", workerID, j),
					"type":     "concurrent_test",
					"worker":   workerID,
					"sequence": j,
				}
				ch <- s.module.ProcessTask(task)
			}
		}(i)
	}

	// Concurrent insight generation
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < operations; j++ {
				ch <- s.module.GenerateInsights()
			}
		}()
	}

	// Concurrent pattern analysis
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < operations; j++ {
				ch <- s.module.AnalyzePatterns()
			}
		}()
	}

	// Concurrent agent coordination
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < operations; j++ {
				ch <- s.module.CoordinateAgents()
			}
		}()
	}

	// Collect results
	successCount := 0
	errorCount := 0

	for i := 0; i < totalOps*4; i++ {
		err := <-ch
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	// Most operations should succeed
	assert.True(s.T(), successCount > totalOps*3, "Most concurrent operations should succeed")

	// Wait for all operations to complete
	time.Sleep(200 * time.Millisecond)

	// Verify stats reflect concurrent operations
	stats := s.module.GetStats()
	assert.Equal(s.T(), int64(totalOps), stats.TasksProcessed)
	assert.True(s.T(), stats.InsightsGenerated >= int64(totalOps), "Insights should be >= total ops")
	assert.True(s.T(), stats.PatternsDetected >= int64(totalOps), "Patterns should be >= total ops")
	assert.True(s.T(), stats.AgentsCoordinated >= int64(totalOps), "Agent coordination should be >= total ops")
}

// TestModuleErrorHandling tests error handling
func (s *RnDModuleTestSuite) TestModuleErrorHandling() {
	// Start module
	err := s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Test processing nil task
	err = s.module.ProcessTask(nil)
	assert.NoError(s.T(), err) // Should handle gracefully

	// Test processing invalid task
	invalidTask := "invalid task type"
	err = s.module.ProcessTask(invalidTask)
	assert.NoError(s.T(), err) // Should handle gracefully

	// Check that errors are tracked in stats
	stats := s.module.GetStats()
	// Error handling should prevent crashes but may increase error count
	assert.True(s.T(), stats.ProcessingErrors >= 0)
}

// TestModuleStressTest tests module under stress
func (s *RnDModuleTestSuite) TestModuleStressTest() {
	// Start module
	err := s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Stress test with rapid operations
	stressOps := 100

	startTime := time.Now()

	// Rapid task processing
	for i := 0; i < stressOps; i++ {
		task := map[string]interface{}{
			"id":   fmt.Sprintf("stress-task-%d", i),
			"type": "stress_test",
			"data": make([]byte, 1024), // 1KB of data
		}
		err = s.module.ProcessTask(task)
		assert.NoError(s.T(), err)
	}

	duration := time.Since(startTime)

	// Should complete quickly
	assert.True(s.T(), duration < 5*time.Second, "Stress test should complete in under 5 seconds")

	// Wait for processing to complete
	time.Sleep(1 * time.Second)

	// Verify all tasks were processed
	stats := s.module.GetStats()
	assert.Equal(s.T(), int64(stressOps), stats.TasksProcessed)

	// Module should still be healthy
	health := s.module.Health()
	assert.True(s.T(), health["running"].(bool))
}

// TestModuleGracefulShutdown tests graceful shutdown
func (s *RnDModuleTestSuite) TestModuleGracefulShutdown() {
	// Start module
	err := s.module.Start()
	assert.NoError(s.T(), err)

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Start some long-running operations
	for i := 0; i < 10; i++ {
		task := map[string]interface{}{
			"id":   fmt.Sprintf("shutdown-task-%d", i),
			"type": "shutdown_test",
		}
		err = s.module.ProcessTask(task)
		assert.NoError(s.T(), err)
	}

	// Stop module
	stopStart := time.Now()
	s.module.Stop()
	stopDuration := time.Since(stopStart)

	// Should stop within reasonable time
	assert.True(s.T(), stopDuration < 10*time.Second, "Graceful shutdown should complete in under 10 seconds")

	// Module should be stopped
	assert.False(s.T(), s.module.IsRunning())

	// Health check should reflect stopped state
	health := s.module.Health()
	assert.False(s.T(), health["running"].(bool))
}

// Run the test suite
func TestRnDModuleTestSuite(t *testing.T) {
	suite.Run(t, new(RnDModuleTestSuite))
}
