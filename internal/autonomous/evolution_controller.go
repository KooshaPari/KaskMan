package autonomous

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// EvolutionController manages autonomous project evolution and spawning
type EvolutionController struct {
	logger           *logrus.Logger
	learningEngine   *LearningEngine
	hiveCoordinator  *HiveCoordinator
	projectManager   *AutonomousProjectManager
	evolutionTracker *EvolutionTracker
	seedsDirectory   string
}

// AutonomousProjectManager manages the lifecycle of autonomous projects
type AutonomousProjectManager struct {
	logger         *logrus.Logger
	activeProjects map[uuid.UUID]*AutonomousProject
	seedProjects   map[string]*SeedProject
	spawnQueue     chan *SpawnRequest
}

// SeedProject represents a project in its initial seed phase
type SeedProject struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Type              string                 `json:"type"`
	LearningPhase     string                 `json:"learning_phase"`
	ManifestPath      string                 `json:"manifest_path"`
	LearningDataPath  string                 `json:"learning_data_path"`
	EvolutionSignalPath string               `json:"evolution_signal_path"`
	LastChecked       time.Time              `json:"last_checked"`
	Metrics           map[string]interface{} `json:"metrics"`
}

// EvolutionTracker tracks evolution patterns across all projects
type EvolutionTracker struct {
	logger              *logrus.Logger
	evolutionHistory    []EvolutionEvent     `json:"evolution_history"`
	spawningPatterns    []SpawningPattern    `json:"spawning_patterns"`
	successfulEvolutions []SuccessfulEvolution `json:"successful_evolutions"`
	learningInsights    map[string]interface{} `json:"learning_insights"`
}

// EvolutionEvent represents a project evolution occurrence
type EvolutionEvent struct {
	ID               uuid.UUID              `json:"id"`
	ProjectID        string                 `json:"project_id"`
	FromPhase        string                 `json:"from_phase"`
	ToPhase          string                 `json:"to_phase"`
	Timestamp        time.Time              `json:"timestamp"`
	TriggerReason    string                 `json:"trigger_reason"`
	SuccessMetrics   map[string]float64     `json:"success_metrics"`
	NewCapabilities  []string               `json:"new_capabilities"`
	LearningOutcomes []string               `json:"learning_outcomes"`
	HiveDecision     map[string]interface{} `json:"hive_decision"`
}

// SpawningPattern represents learned patterns for project spawning
type SpawningPattern struct {
	FrictionType     string    `json:"friction_type"`
	SolutionPattern  string    `json:"solution_pattern"`
	SuccessRate      float64   `json:"success_rate"`
	EvolutionPath    []string  `json:"evolution_path"`
	OptimalTiming    float64   `json:"optimal_timing"`
	ResourceRequirements string `json:"resource_requirements"`
}

// SuccessfulEvolution tracks completed evolutions for learning
type SuccessfulEvolution struct {
	ProjectID        string                 `json:"project_id"`
	EvolutionPath    []string               `json:"evolution_path"`
	TimeToEvolution  map[string]float64     `json:"time_to_evolution"`
	UserSatisfaction float64                `json:"user_satisfaction"`
	ImpactMetrics    map[string]interface{} `json:"impact_metrics"`
	SpawnedProjects  []string               `json:"spawned_projects"`
}

// SpawnRequest represents a request to spawn a new project
type SpawnRequest struct {
	FrictionPoint    *FrictionPoint         `json:"friction_point"`
	LearningPattern  *LearningPattern       `json:"learning_pattern"`
	Priority         string                 `json:"priority"`
	RequestedBy      string                 `json:"requested_by"`
	Context          map[string]interface{} `json:"context"`
}

// NewEvolutionController creates the autonomous evolution system
func NewEvolutionController(
	logger *logrus.Logger,
	learningEngine *LearningEngine,
	hiveCoordinator *HiveCoordinator,
	seedsDirectory string,
) *EvolutionController {
	return &EvolutionController{
		logger:           logger,
		learningEngine:   learningEngine,
		hiveCoordinator:  hiveCoordinator,
		projectManager:   NewAutonomousProjectManager(logger),
		evolutionTracker: NewEvolutionTracker(logger),
		seedsDirectory:   seedsDirectory,
	}
}

// StartEvolutionCycle begins the autonomous evolution monitoring
func (ec *EvolutionController) StartEvolutionCycle(ctx context.Context) error {
	ec.logger.Info("Starting autonomous evolution cycle")

	// Discover existing seed projects
	if err := ec.discoverSeedProjects(); err != nil {
		return fmt.Errorf("failed to discover seed projects: %w", err)
	}

	// Start monitoring loops
	go ec.monitorSeedEvolution(ctx)
	go ec.processSpawnQueue(ctx)
	go ec.trackEvolutionPatterns(ctx)

	return nil
}

// discoverSeedProjects scans for existing seed projects
func (ec *EvolutionController) discoverSeedProjects() error {
	seedDirs, err := ioutil.ReadDir(ec.seedsDirectory)
	if err != nil {
		return err
	}

	for _, dir := range seedDirs {
		if !dir.IsDir() {
			continue
		}

		manifestPath := filepath.Join(ec.seedsDirectory, dir.Name(), "project_manifest.json")
		if _, err := ioutil.ReadFile(manifestPath); err == nil {
			seedProject := &SeedProject{
				ID:                  dir.Name(),
				ManifestPath:        manifestPath,
				LearningDataPath:    filepath.Join(ec.seedsDirectory, dir.Name(), "learning_data.json"),
				EvolutionSignalPath: filepath.Join(ec.seedsDirectory, dir.Name(), "evolution_ready.json"),
				LastChecked:         time.Now(),
			}

			// Load manifest data
			if err := ec.loadSeedManifest(seedProject); err != nil {
				ec.logger.WithError(err).Warnf("Failed to load manifest for seed %s", dir.Name())
				continue
			}

			ec.projectManager.seedProjects[dir.Name()] = seedProject
			
			ec.logger.WithFields(logrus.Fields{
				"seed_id":   seedProject.ID,
				"name":      seedProject.Name,
				"phase":     seedProject.LearningPhase,
			}).Info("Discovered seed project")
		}
	}

	return nil
}

// loadSeedManifest loads project manifest and learning data
func (ec *EvolutionController) loadSeedManifest(seed *SeedProject) error {
	// Load manifest
	manifestData, err := ioutil.ReadFile(seed.ManifestPath)
	if err != nil {
		return err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return err
	}

	seed.Name = manifest["name"].(string)
	seed.Type = manifest["type"].(string)
	seed.LearningPhase = manifest["learning_phase"].(string)

	// Load learning data if available
	if learningData, err := ioutil.ReadFile(seed.LearningDataPath); err == nil {
		var metrics map[string]interface{}
		if err := json.Unmarshal(learningData, &metrics); err == nil {
			seed.Metrics = metrics
		}
	}

	return nil
}

// monitorSeedEvolution monitors seed projects for evolution triggers
func (ec *EvolutionController) monitorSeedEvolution(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ec.checkSeedEvolutionTriggers()
		}
	}
}

// checkSeedEvolutionTriggers checks if any seeds are ready to evolve
func (ec *EvolutionController) checkSeedEvolutionTriggers() {
	for seedID, seed := range ec.projectManager.seedProjects {
		// Check for evolution signal file
		if ec.hasEvolutionSignal(seed) {
			ec.logger.WithField("seed_id", seedID).Info("Evolution signal detected")
			
			// Process evolution
			if err := ec.processEvolution(seed); err != nil {
				ec.logger.WithError(err).Errorf("Failed to process evolution for seed %s", seedID)
			}
		}

		// Check learning metrics for automatic triggers
		if ec.shouldAutoEvolve(seed) {
			ec.logger.WithField("seed_id", seedID).Info("Auto-evolution criteria met")
			
			if err := ec.triggerAutoEvolution(seed); err != nil {
				ec.logger.WithError(err).Errorf("Failed to trigger auto-evolution for seed %s", seedID)
			}
		}
	}
}

// hasEvolutionSignal checks if evolution signal file exists
func (ec *EvolutionController) hasEvolutionSignal(seed *SeedProject) bool {
	_, err := ioutil.ReadFile(seed.EvolutionSignalPath)
	return err == nil
}

// shouldAutoEvolve determines if seed should auto-evolve based on metrics
func (ec *EvolutionController) shouldAutoEvolve(seed *SeedProject) bool {
	if seed.Metrics == nil {
		return false
	}

	performanceMetrics, ok := seed.Metrics["performance_metrics"].(map[string]interface{})
	if !ok {
		return false
	}

	// Check evolution triggers from manifest
	totalUses, ok := performanceMetrics["total_uses"].(float64)
	if !ok {
		return false
	}

	successRate, ok := performanceMetrics["success_rate"].(float64)
	if !ok {
		return false
	}

	// Default evolution thresholds
	return totalUses >= 50 && successRate >= 0.85
}

// processEvolution handles the evolution of a seed project
func (ec *EvolutionController) processEvolution(seed *SeedProject) error {
	// Read evolution signal
	signalData, err := ioutil.ReadFile(seed.EvolutionSignalPath)
	if err != nil {
		return err
	}

	var signal map[string]interface{}
	if err := json.Unmarshal(signalData, &signal); err != nil {
		return err
	}

	// Coordinate with hive mind for evolution decision
	evolutionPlan := ec.coordinateEvolutionWithHive(seed, signal)

	// Execute evolution
	evolutionEvent := &EvolutionEvent{
		ID:            uuid.New(),
		ProjectID:     seed.ID,
		FromPhase:     seed.LearningPhase,
		ToPhase:       "growth",
		Timestamp:     time.Now(),
		TriggerReason: signal["trigger"].(string),
		HiveDecision:  evolutionPlan,
	}

	// Implement evolution changes
	if err := ec.implementEvolution(seed, evolutionEvent); err != nil {
		return err
	}

	// Record evolution
	ec.evolutionTracker.evolutionHistory = append(ec.evolutionTracker.evolutionHistory, *evolutionEvent)

	// Clean up evolution signal
	ioutil.WriteFile(seed.EvolutionSignalPath, []byte{}, 0644) // Clear file

	ec.logger.WithFields(logrus.Fields{
		"seed_id":        seed.ID,
		"evolution_id":   evolutionEvent.ID,
		"from_phase":     evolutionEvent.FromPhase,
		"to_phase":       evolutionEvent.ToPhase,
		"trigger_reason": evolutionEvent.TriggerReason,
	}).Info("Evolution completed successfully")

	return nil
}

// coordinateEvolutionWithHive gets hive mind input on evolution
func (ec *EvolutionController) coordinateEvolutionWithHive(seed *SeedProject, signal map[string]interface{}) map[string]interface{} {
	// Simulate hive coordination - in real implementation, use HiveCoordinator
	return map[string]interface{}{
		"consensus": "approved",
		"strategy": "incremental_enhancement",
		"new_capabilities": []string{
			"enhanced_format_support",
			"performance_optimization",
			"user_preference_adaptation",
		},
		"resource_allocation": map[string]interface{}{
			"development_time": "2_weeks",
			"testing_phase": "1_week",
			"agents_assigned": []string{"researcher", "coder", "tester"},
		},
		"success_criteria": map[string]interface{}{
			"performance_improvement": 0.2,
			"user_satisfaction": 0.9,
			"capability_expansion": 3,
		},
	}
}

// implementEvolution executes the evolution plan
func (ec *EvolutionController) implementEvolution(seed *SeedProject, event *EvolutionEvent) error {
	// Update seed project phase
	seed.LearningPhase = event.ToPhase

	// Generate new capabilities based on hive decision
	newCapabilities := event.HiveDecision["new_capabilities"].([]string)
	event.NewCapabilities = newCapabilities

	// Simulate capability implementation
	event.LearningOutcomes = []string{
		"Enhanced image format detection",
		"Improved processing performance",
		"Adaptive user preference learning",
		"Advanced storage optimization",
	}

	// Update success metrics
	event.SuccessMetrics = map[string]float64{
		"capability_count": float64(len(newCapabilities)),
		"implementation_confidence": 0.92,
		"expected_performance_gain": 0.25,
	}

	return nil
}

// triggerAutoEvolution automatically triggers evolution based on metrics
func (ec *EvolutionController) triggerAutoEvolution(seed *SeedProject) error {
	// Create auto-evolution signal
	signal := map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"trigger":   "automatic_metrics_threshold",
		"phase_transition": fmt.Sprintf("%s_to_growth", seed.LearningPhase),
		"auto_triggered": true,
	}

	return ec.processEvolution(seed)
}

// SpawnNewProject creates a new autonomous project from friction
func (ec *EvolutionController) SpawnNewProject(ctx context.Context, friction *FrictionPoint, pattern *LearningPattern) (*AutonomousProject, error) {
	// Create spawn request
	spawnRequest := &SpawnRequest{
		FrictionPoint:   friction,
		LearningPattern: pattern,
		Priority:        friction.Impact,
		RequestedBy:     "autonomous_learning_engine",
		Context: map[string]interface{}{
			"spawn_timestamp": time.Now().Unix(),
			"friction_context": friction.Context,
		},
	}

	// Queue for processing
	select {
	case ec.projectManager.spawnQueue <- spawnRequest:
		ec.logger.WithFields(logrus.Fields{
			"friction_id": friction.ID,
			"pattern_id":  pattern.ID,
			"priority":    spawnRequest.Priority,
		}).Info("Project spawn request queued")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Coordinate with hive mind
	swarm, err := ec.hiveCoordinator.CoordinateFrictionResolution(ctx, friction)
	if err != nil {
		return nil, fmt.Errorf("hive coordination failed: %w", err)
	}

	// Create autonomous project
	project := &AutonomousProject{
		ID:               uuid.New(),
		OriginFriction:   friction.ID,
		Status:           "ideation",
		LearningPhase:    "seed",
		AssignedAgents:   []string{"queen", "researcher", "architect", "coder"},
		CollaborationStatus: "coordinating",
	}

	// Generate project based on swarm results
	if err := ec.generateProjectFromSwarm(project, swarm); err != nil {
		return nil, fmt.Errorf("project generation failed: %w", err)
	}

	ec.projectManager.activeProjects[project.ID] = project

	ec.logger.WithFields(logrus.Fields{
		"project_id":     project.ID,
		"friction_id":    friction.ID,
		"learning_phase": project.LearningPhase,
		"status":         project.Status,
	}).Info("New autonomous project spawned")

	return project, nil
}

// generateProjectFromSwarm creates project structure from swarm results
func (ec *EvolutionController) generateProjectFromSwarm(project *AutonomousProject, swarm *SwarmExecution) error {
	// Extract project details from swarm results
	if solutionPlan, ok := swarm.Results["solution_plan"].(map[string]interface{}); ok {
		project.Name = "autonomous_" + swarm.SwarmType + "_solution"
		project.Purpose = "Solve friction through autonomous R&D"
		project.ProjectType = solutionPlan["solution_type"].(string)
		
		// Set research goals from swarm analysis
		if goals, ok := solutionPlan["learning_objectives"].([]string); ok {
			project.ResearchGoals = goals
		}
	}

	project.Status = "planning"
	return nil
}

// processSpawnQueue processes project spawn requests
func (ec *EvolutionController) processSpawnQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-ec.projectManager.spawnQueue:
			ec.processSpawnRequest(ctx, request)
		}
	}
}

// processSpawnRequest handles individual spawn requests
func (ec *EvolutionController) processSpawnRequest(ctx context.Context, request *SpawnRequest) {
	ec.logger.WithFields(logrus.Fields{
		"friction_type": request.FrictionPoint.Type,
		"priority":      request.Priority,
	}).Info("Processing spawn request")

	// Implement spawn logic here
	// This would create actual project files, directories, and initial code
}

// trackEvolutionPatterns learns from evolution patterns
func (ec *EvolutionController) trackEvolutionPatterns(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Analyze patterns hourly
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ec.analyzeEvolutionPatterns()
		}
	}
}

// analyzeEvolutionPatterns extracts learning from evolution history
func (ec *EvolutionController) analyzeEvolutionPatterns() {
	// Analyze successful evolutions
	for _, event := range ec.evolutionTracker.evolutionHistory {
		// Extract patterns for future use
		pattern := SpawningPattern{
			FrictionType:     "evolution_success",
			SolutionPattern:  event.ToPhase,
			SuccessRate:      ec.calculateEvolutionSuccessRate(event),
			EvolutionPath:    []string{event.FromPhase, event.ToPhase},
			OptimalTiming:    ec.calculateOptimalTiming(event),
		}

		ec.evolutionTracker.spawningPatterns = append(ec.evolutionTracker.spawningPatterns, pattern)
	}

	ec.logger.WithFields(logrus.Fields{
		"evolution_events": len(ec.evolutionTracker.evolutionHistory),
		"patterns_learned": len(ec.evolutionTracker.spawningPatterns),
	}).Info("Evolution patterns analyzed")
}

// Helper methods

func (ec *EvolutionController) calculateEvolutionSuccessRate(event EvolutionEvent) float64 {
	// Calculate based on success metrics
	if confidence, ok := event.SuccessMetrics["implementation_confidence"]; ok {
		return confidence
	}
	return 0.8 // Default confidence
}

func (ec *EvolutionController) calculateOptimalTiming(event EvolutionEvent) float64 {
	// Calculate optimal timing based on event data
	return 24.0 // Default: 24 hours
}

// GetEvolutionInsights provides system-wide evolution analytics
func (ec *EvolutionController) GetEvolutionInsights(ctx context.Context) (map[string]interface{}, error) {
	insights := map[string]interface{}{
		"active_seeds":        len(ec.projectManager.seedProjects),
		"evolution_events":    len(ec.evolutionTracker.evolutionHistory),
		"spawning_patterns":   len(ec.evolutionTracker.spawningPatterns),
		"autonomous_projects": len(ec.projectManager.activeProjects),
		"evolution_success_rate": ec.calculateOverallSuccessRate(),
		"learning_velocity":   ec.calculateLearningVelocity(),
		"next_evolutions":     ec.predictNextEvolutions(),
	}

	return insights, nil
}

func (ec *EvolutionController) calculateOverallSuccessRate() float64 {
	if len(ec.evolutionTracker.evolutionHistory) == 0 {
		return 0.0
	}
	
	successful := 0
	for _, event := range ec.evolutionTracker.evolutionHistory {
		if event.SuccessMetrics["implementation_confidence"] > 0.8 {
			successful++
		}
	}
	
	return float64(successful) / float64(len(ec.evolutionTracker.evolutionHistory))
}

func (ec *EvolutionController) calculateLearningVelocity() float64 {
	// Calculate how fast the system is learning and evolving
	if len(ec.evolutionTracker.evolutionHistory) < 2 {
		return 0.0
	}
	
	// Simple velocity calculation based on evolution frequency
	firstEvent := ec.evolutionTracker.evolutionHistory[0]
	lastEvent := ec.evolutionTracker.evolutionHistory[len(ec.evolutionTracker.evolutionHistory)-1]
	
	timeDiff := lastEvent.Timestamp.Sub(firstEvent.Timestamp).Hours()
	eventCount := float64(len(ec.evolutionTracker.evolutionHistory))
	
	return eventCount / timeDiff // Evolutions per hour
}

func (ec *EvolutionController) predictNextEvolutions() []string {
	// Predict which seeds are likely to evolve next
	predictions := []string{}
	
	for _, seed := range ec.projectManager.seedProjects {
		if ec.shouldAutoEvolve(seed) {
			predictions = append(predictions, seed.ID)
		}
	}
	
	return predictions
}

// Initialize supporting components

func NewAutonomousProjectManager(logger *logrus.Logger) *AutonomousProjectManager {
	return &AutonomousProjectManager{
		logger:         logger,
		activeProjects: make(map[uuid.UUID]*AutonomousProject),
		seedProjects:   make(map[string]*SeedProject),
		spawnQueue:     make(chan *SpawnRequest, 100),
	}
}

func NewEvolutionTracker(logger *logrus.Logger) *EvolutionTracker {
	return &EvolutionTracker{
		logger:              logger,
		evolutionHistory:    []EvolutionEvent{},
		spawningPatterns:    []SpawningPattern{},
		successfulEvolutions: []SuccessfulEvolution{},
		learningInsights:    make(map[string]interface{}),
	}
}