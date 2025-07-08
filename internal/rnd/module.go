package rnd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/internal/config"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd/coordinator"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd/learning"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd/patterns"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd/projects"
	"github.com/sirupsen/logrus"
)

// Module represents the main R&D module
type Module struct {
	config      *config.RnDConfig
	db          *database.Database
	logger      *logrus.Logger
	coordinator *coordinator.Coordinator
	learning    *learning.Engine
	patterns    *patterns.Recognizer
	projects    *projects.Generator

	// Control channels
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}
	doneCh chan struct{}

	// State
	running bool
	mutex   sync.RWMutex

	// Workers
	workers  []Worker
	workerWg sync.WaitGroup

	// Statistics
	stats *Statistics
}

// Worker represents a background worker
type Worker interface {
	Start(ctx context.Context) error
	Stop() error
	GetStats() interface{}
}

// Statistics holds R&D module statistics
type Statistics struct {
	StartTime          time.Time `json:"start_time"`
	TasksProcessed     int64     `json:"tasks_processed"`
	PatternsDetected   int64     `json:"patterns_detected"`
	InsightsGenerated  int64     `json:"insights_generated"`
	ProjectsGenerated  int64     `json:"projects_generated"`
	AgentsCoordinated  int64     `json:"agents_coordinated"`
	LearningIterations int64     `json:"learning_iterations"`
	ProcessingErrors   int64     `json:"processing_errors"`
	AvgProcessingTime  float64   `json:"avg_processing_time_ms"`
	LastActivity       time.Time `json:"last_activity"`

	mutex sync.RWMutex
}

// NewModule creates a new R&D module instance
func NewModule(cfg config.RnDConfig, db *database.Database, logger *logrus.Logger) (*Module, error) {
	if !cfg.Enabled {
		logger.Info("R&D module is disabled")
		return &Module{
			config: &cfg,
			db:     db,
			logger: logger,
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize components
	coord, err := coordinator.NewCoordinator(cfg, db, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create coordinator: %w", err)
	}

	learningEngine, err := learning.NewEngine(cfg, db, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create learning engine: %w", err)
	}

	patternRecognizer := patterns.NewRecognizer(db.DB, logger, nil)

	projectGenerator := projects.NewGenerator(db.DB, logger, patternRecognizer, nil)

	module := &Module{
		config:      &cfg,
		db:          db,
		logger:      logger,
		coordinator: coord,
		learning:    learningEngine,
		patterns:    patternRecognizer,
		projects:    projectGenerator,
		ctx:         ctx,
		cancel:      cancel,
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
		stats: &Statistics{
			StartTime:    time.Now(),
			LastActivity: time.Now(),
		},
	}

	// Initialize workers (pattern recognizer and project generator manage their own lifecycle)
	module.workers = []Worker{
		coord,
		learningEngine,
	}

	logger.Info("R&D module initialized successfully")
	return module, nil
}

// Start starts the R&D module and all its components
func (m *Module) Start() error {
	if !m.config.Enabled {
		m.logger.Info("R&D module is disabled, skipping start")
		return nil
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		return fmt.Errorf("R&D module is already running")
	}

	m.logger.Info("Starting R&D module...")

	// Start all workers
	for i, worker := range m.workers {
		m.workerWg.Add(1)
		go func(idx int, w Worker) {
			defer m.workerWg.Done()

			if err := w.Start(m.ctx); err != nil {
				m.logger.WithError(err).WithField("worker_index", idx).Error("Worker failed to start")
			}
		}(i, worker)
	}

	// Start pattern recognizer
	if err := m.patterns.Start(); err != nil {
		m.logger.WithError(err).Error("Failed to start pattern recognizer")
	}

	// Start project generator
	if err := m.projects.Start(); err != nil {
		m.logger.WithError(err).Error("Failed to start project generator")
	}

	// Start main processing loop
	go m.run()

	// Start statistics collector
	go m.collectStatistics()

	m.running = true
	m.stats.StartTime = time.Now()

	m.logger.Info("R&D module started successfully")
	return nil
}

// Stop stops the R&D module and all its components
func (m *Module) Stop() {
	if !m.config.Enabled {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return
	}

	m.logger.Info("Stopping R&D module...")

	// Signal stop
	close(m.stopCh)

	// Cancel context to stop all workers
	m.cancel()

	// Wait for workers to finish
	m.workerWg.Wait()

	// Stop individual components
	for _, worker := range m.workers {
		if err := worker.Stop(); err != nil {
			m.logger.WithError(err).Error("Failed to stop worker")
		}
	}

	// Stop pattern recognizer
	if err := m.patterns.Stop(); err != nil {
		m.logger.WithError(err).Error("Failed to stop pattern recognizer")
	}

	// Stop project generator
	if err := m.projects.Stop(); err != nil {
		m.logger.WithError(err).Error("Failed to stop project generator")
	}

	// Wait for main loop to finish
	<-m.doneCh

	m.running = false
	m.logger.Info("R&D module stopped")
}

// IsRunning returns whether the module is currently running
func (m *Module) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.running
}

// GetStats returns current module statistics
func (m *Module) GetStats() *Statistics {
	m.stats.mutex.RLock()
	defer m.stats.mutex.RUnlock()

	// Create a copy to avoid race conditions
	stats := *m.stats
	return &stats
}

// GetCoordinator returns the agent coordinator
func (m *Module) GetCoordinator() *coordinator.Coordinator {
	return m.coordinator
}

// GetLearningEngine returns the learning engine
func (m *Module) GetLearningEngine() *learning.Engine {
	return m.learning
}

// GetPatternRecognizer returns the pattern recognizer
func (m *Module) GetPatternRecognizer() *patterns.Recognizer {
	return m.patterns
}

// GetProjectGenerator returns the project generator
func (m *Module) GetProjectGenerator() *projects.Generator {
	return m.projects
}

// ProcessTask processes a task through the R&D pipeline
func (m *Module) ProcessTask(task interface{}) error {
	if !m.running {
		return fmt.Errorf("R&D module is not running")
	}

	startTime := time.Now()
	defer func() {
		m.updateStats(func(s *Statistics) {
			s.TasksProcessed++
			processingTime := float64(time.Since(startTime).Nanoseconds()) / 1e6
			s.AvgProcessingTime = (s.AvgProcessingTime + processingTime) / 2
			s.LastActivity = time.Now()
		})
	}()

	// TODO: Implement task processing pipeline
	m.logger.WithField("task", task).Debug("Processing R&D task")

	return nil
}

// AnalyzePatterns triggers pattern analysis
func (m *Module) AnalyzePatterns() error {
	if !m.running {
		return fmt.Errorf("R&D module is not running")
	}

	defer m.updateStats(func(s *Statistics) {
		s.PatternsDetected++
		s.LastActivity = time.Now()
	})

	// The pattern recognizer runs continuously, no need for explicit trigger
	return nil
}

// GenerateInsights triggers insight generation
func (m *Module) GenerateInsights() error {
	if !m.running {
		return fmt.Errorf("R&D module is not running")
	}

	defer m.updateStats(func(s *Statistics) {
		s.InsightsGenerated++
		s.LastActivity = time.Now()
	})

	return m.learning.GenerateInsights()
}

// GenerateProjects triggers project generation
func (m *Module) GenerateProjects() error {
	if !m.running {
		return fmt.Errorf("R&D module is not running")
	}

	defer m.updateStats(func(s *Statistics) {
		s.ProjectsGenerated++
		s.LastActivity = time.Now()
	})

	// Generate projects with default preferences
	_, err := m.projects.GenerateProjectSuggestions("system", nil)
	return err
}

// CoordinateAgents triggers agent coordination
func (m *Module) CoordinateAgents() error {
	if !m.running {
		return fmt.Errorf("R&D module is not running")
	}

	defer m.updateStats(func(s *Statistics) {
		s.AgentsCoordinated++
		s.LastActivity = time.Now()
	})

	return m.coordinator.CoordinateAgents()
}

// run is the main processing loop
func (m *Module) run() {
	defer close(m.doneCh)

	m.logger.Info("R&D module main loop started")

	ticker := time.NewTicker(m.config.LearningInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			m.logger.Info("R&D module main loop stopping")
			return

		case <-ticker.C:
			// Periodic learning and analysis
			m.performPeriodicTasks()

		case <-m.ctx.Done():
			m.logger.Info("R&D module context cancelled")
			return
		}
	}
}

// performPeriodicTasks performs periodic R&D tasks
func (m *Module) performPeriodicTasks() {
	m.logger.Debug("Performing periodic R&D tasks")

	// Run pattern analysis
	if err := m.AnalyzePatterns(); err != nil {
		m.logger.WithError(err).Error("Failed to analyze patterns")
		m.updateStats(func(s *Statistics) { s.ProcessingErrors++ })
	}

	// Generate insights
	if err := m.GenerateInsights(); err != nil {
		m.logger.WithError(err).Error("Failed to generate insights")
		m.updateStats(func(s *Statistics) { s.ProcessingErrors++ })
	}

	// Coordinate agents
	if err := m.CoordinateAgents(); err != nil {
		m.logger.WithError(err).Error("Failed to coordinate agents")
		m.updateStats(func(s *Statistics) { s.ProcessingErrors++ })
	}

	m.updateStats(func(s *Statistics) {
		s.LearningIterations++
		s.LastActivity = time.Now()
	})
}

// collectStatistics periodically collects and updates statistics
func (m *Module) collectStatistics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			// Collect statistics from workers
			for _, worker := range m.workers {
				workerStats := worker.GetStats()
				m.logger.WithFields(logrus.Fields{
					"worker_type": fmt.Sprintf("%T", worker),
					"stats":       workerStats,
				}).Debug("Worker statistics")
			}
		}
	}
}

// updateStats safely updates statistics
func (m *Module) updateStats(updater func(*Statistics)) {
	m.stats.mutex.Lock()
	defer m.stats.mutex.Unlock()
	updater(m.stats)
}

// Health returns the health status of the R&D module
func (m *Module) Health() map[string]interface{} {
	m.mutex.RLock()
	running := m.running
	m.mutex.RUnlock()

	health := map[string]interface{}{
		"enabled": m.config.Enabled,
		"running": running,
		"stats":   m.GetStats(),
	}

	if running {
		health["coordinator"] = m.coordinator.Health()
		health["learning"] = m.learning.Health()
		health["patterns"] = m.patterns.Health()
		health["projects"] = m.projects.Health()
	}

	return health
}
