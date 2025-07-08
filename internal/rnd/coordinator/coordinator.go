package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/config"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/sirupsen/logrus"
)

// Coordinator manages and coordinates multiple AI agents
type Coordinator struct {
	config *config.RnDConfig
	db     *database.Database
	logger *logrus.Logger

	// Agent management
	agents    map[string]*AgentInstance
	agentsMux sync.RWMutex

	// Task queue and processing
	taskQueue chan *Task

	// Control
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	runMux  sync.RWMutex

	// Statistics
	stats    *CoordinatorStats
	statsMux sync.RWMutex
}

// AgentInstance represents an active agent instance
type AgentInstance struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Status          string                 `json:"status"`
	Config          map[string]interface{} `json:"config"`
	CreatedAt       time.Time              `json:"created_at"`
	LastActive      time.Time              `json:"last_active"`
	TaskCount       int                    `json:"task_count"`
	SuccessRate     float64                `json:"success_rate"`
	AvgResponseTime float64                `json:"avg_response_time"`
	CurrentTask     *Task                  `json:"current_task,omitempty"`
}

// Task represents a task to be executed by an agent
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	Status      string                 `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// CoordinatorStats holds coordination statistics
type CoordinatorStats struct {
	ActiveAgents   int       `json:"active_agents"`
	TotalTasks     int64     `json:"total_tasks"`
	CompletedTasks int64     `json:"completed_tasks"`
	FailedTasks    int64     `json:"failed_tasks"`
	QueueLength    int       `json:"queue_length"`
	AvgTaskTime    float64   `json:"avg_task_time_ms"`
	LastActivity   time.Time `json:"last_activity"`
}

// NewCoordinator creates a new agent coordinator
func NewCoordinator(cfg config.RnDConfig, db *database.Database, logger *logrus.Logger) (*Coordinator, error) {
	ctx, cancel := context.WithCancel(context.Background())

	coord := &Coordinator{
		config:    &cfg,
		db:        db,
		logger:    logger,
		agents:    make(map[string]*AgentInstance),
		taskQueue: make(chan *Task, cfg.QueueSize),
		ctx:       ctx,
		cancel:    cancel,
		stats: &CoordinatorStats{
			LastActivity: time.Now(),
		},
	}

	return coord, nil
}

// Start starts the coordinator and begins processing tasks
func (c *Coordinator) Start(ctx context.Context) error {
	c.runMux.Lock()
	defer c.runMux.Unlock()

	if c.running {
		return fmt.Errorf("coordinator is already running")
	}

	c.logger.Info("Starting agent coordinator")

	// Load existing agents from database
	if err := c.loadAgents(); err != nil {
		return fmt.Errorf("failed to load agents: %w", err)
	}

	// Start worker goroutines
	for i := 0; i < c.config.WorkerCount; i++ {
		go c.taskWorker(i)
	}

	// Start agent monitor
	go c.agentMonitor()

	// Start task scheduler
	go c.taskScheduler()

	c.running = true
	c.logger.Info("Agent coordinator started successfully")

	return nil
}

// Stop stops the coordinator
func (c *Coordinator) Stop() error {
	c.runMux.Lock()
	defer c.runMux.Unlock()

	if !c.running {
		return nil
	}

	c.logger.Info("Stopping agent coordinator")

	// Cancel context to stop all goroutines
	c.cancel()

	// Close task queue
	close(c.taskQueue)

	// Update all agents as inactive
	c.agentsMux.Lock()
	for _, agent := range c.agents {
		agent.Status = "inactive"
		c.updateAgentInDB(agent)
	}
	c.agentsMux.Unlock()

	c.running = false
	c.logger.Info("Agent coordinator stopped")

	return nil
}

// IsRunning returns whether the coordinator is running
func (c *Coordinator) IsRunning() bool {
	c.runMux.RLock()
	defer c.runMux.RUnlock()
	return c.running
}

// GetStats returns current coordinator statistics
func (c *Coordinator) GetStats() interface{} {
	c.statsMux.RLock()
	defer c.statsMux.RUnlock()

	stats := *c.stats
	stats.QueueLength = len(c.taskQueue)
	stats.ActiveAgents = c.getActiveAgentCount()

	return stats
}

// Health returns coordinator health status
func (c *Coordinator) Health() map[string]interface{} {
	return map[string]interface{}{
		"running":       c.IsRunning(),
		"active_agents": c.getActiveAgentCount(),
		"queue_length":  len(c.taskQueue),
		"stats":         c.GetStats(),
	}
}

// CoordinateAgents performs agent coordination tasks
func (c *Coordinator) CoordinateAgents() error {
	c.logger.Debug("Coordinating agents")

	// Rebalance workload
	if err := c.rebalanceWorkload(); err != nil {
		c.logger.WithError(err).Error("Failed to rebalance workload")
		return err
	}

	// Check agent health
	if err := c.checkAgentHealth(); err != nil {
		c.logger.WithError(err).Error("Failed to check agent health")
		return err
	}

	// Scale agents if needed
	if err := c.scaleAgents(); err != nil {
		c.logger.WithError(err).Error("Failed to scale agents")
		return err
	}

	c.updateLastActivity()
	return nil
}

// CreateAgent creates a new agent instance
func (c *Coordinator) CreateAgent(agentType string, config map[string]interface{}) (*AgentInstance, error) {
	c.agentsMux.Lock()
	defer c.agentsMux.Unlock()

	// Check if we've reached the maximum agent count
	if len(c.agents) >= c.config.AgentMaxCount {
		return nil, fmt.Errorf("maximum agent count reached (%d)", c.config.AgentMaxCount)
	}

	agent := &AgentInstance{
		ID:              fmt.Sprintf("agent_%d_%d", time.Now().Unix(), len(c.agents)),
		Type:            agentType,
		Status:          "active",
		Config:          config,
		CreatedAt:       time.Now(),
		LastActive:      time.Now(),
		TaskCount:       0,
		SuccessRate:     0.5, // Start with reasonable success rate
		AvgResponseTime: 0.0,
	}

	// Save to database
	dbAgent := &models.Agent{
		Name:            agent.ID,
		Type:            agent.Type,
		Status:          agent.Status,
		Capabilities:    "[]", // JSON array of capabilities
		Config:          "{}", // JSON configuration
		LastActive:      &agent.LastActive,
		TaskCount:       agent.TaskCount,
		SuccessRate:     agent.SuccessRate,
		AvgResponseTime: 0.0,
	}

	if err := c.db.DB.Create(dbAgent).Error; err != nil {
		return nil, fmt.Errorf("failed to save agent to database: %w", err)
	}

	c.agents[agent.ID] = agent

	c.logger.WithFields(logrus.Fields{
		"agent_id":   agent.ID,
		"agent_type": agent.Type,
	}).Info("Created new agent")

	return agent, nil
}

// GetAgent returns an agent by ID
func (c *Coordinator) GetAgent(agentID string) (*AgentInstance, error) {
	c.agentsMux.RLock()
	defer c.agentsMux.RUnlock()

	agent, exists := c.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// GetAgents returns all agents
func (c *Coordinator) GetAgents() []*AgentInstance {
	c.agentsMux.RLock()
	defer c.agentsMux.RUnlock()

	agents := make([]*AgentInstance, 0, len(c.agents))
	for _, agent := range c.agents {
		agents = append(agents, agent)
	}

	return agents
}

// SubmitTask submits a task to the task queue
func (c *Coordinator) SubmitTask(task *Task) error {
	select {
	case c.taskQueue <- task:
		c.updateStats(func(s *CoordinatorStats) {
			s.TotalTasks++
		})
		return nil
	default:
		return fmt.Errorf("task queue is full")
	}
}

// taskWorker processes tasks from the queue
func (c *Coordinator) taskWorker(workerID int) {
	c.logger.WithField("worker_id", workerID).Debug("Task worker started")

	for {
		select {
		case task, ok := <-c.taskQueue:
			if !ok {
				c.logger.WithField("worker_id", workerID).Debug("Task worker stopping")
				return
			}

			c.processTask(task, workerID)

		case <-c.ctx.Done():
			c.logger.WithField("worker_id", workerID).Debug("Task worker context cancelled")
			return
		}
	}
}

// processTask processes a single task
func (c *Coordinator) processTask(task *Task, workerID int) {
	startTime := time.Now()

	c.logger.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"task_type": task.Type,
		"worker_id": workerID,
	}).Debug("Processing task")

	// Find best agent for this task
	agent := c.findBestAgent(task)
	if agent == nil {
		c.logger.WithField("task_id", task.ID).Error("No suitable agent found for task")
		c.updateStats(func(s *CoordinatorStats) { s.FailedTasks++ })
		return
	}

	// Assign task to agent
	task.AssignedTo = agent.ID
	task.StartedAt = &startTime
	task.Status = "in_progress"

	// Simulate task execution
	c.executeTask(task, agent)

	// Update statistics
	processingTime := float64(time.Since(startTime).Nanoseconds()) / 1e6
	c.updateStats(func(s *CoordinatorStats) {
		s.CompletedTasks++
		s.AvgTaskTime = (s.AvgTaskTime + processingTime) / 2
		s.LastActivity = time.Now()
	})
}

// findBestAgent finds the best agent for a given task
func (c *Coordinator) findBestAgent(task *Task) *AgentInstance {
	c.agentsMux.RLock()
	defer c.agentsMux.RUnlock()

	var bestAgent *AgentInstance
	bestScore := -1.0

	for _, agent := range c.agents {
		if agent.Status != "active" || agent.CurrentTask != nil {
			continue
		}

		// Calculate suitability score
		score := c.calculateAgentScore(agent, task)
		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	return bestAgent
}

// calculateAgentScore calculates how suitable an agent is for a task
func (c *Coordinator) calculateAgentScore(agent *AgentInstance, task *Task) float64 {
	score := 0.0

	// Type matching - exact match gets high score
	if agent.Type == task.Type {
		score += 2.0
	} else if c.isCompatibleType(agent.Type, task.Type) {
		score += 1.0
	}

	// Success rate (weighted heavily)
	score += agent.SuccessRate * 2.0

	// Inverse of task count (prefer less busy agents)
	if agent.TaskCount > 0 {
		score += 1.0 / float64(agent.TaskCount)
	} else {
		score += 1.0
	}

	// Priority factor - high priority tasks get better agents
	priorityFactor := float64(task.Priority) / 4.0
	score *= (1.0 + priorityFactor)

	// Recency factor - recently active agents score higher
	timeSinceActive := time.Since(agent.LastActive)
	if timeSinceActive < 5*time.Minute {
		score += 0.5
	} else if timeSinceActive > 30*time.Minute {
		score -= 0.5
	}

	return score
}

// isCompatibleType checks if agent type is compatible with task type
func (c *Coordinator) isCompatibleType(agentType, taskType string) bool {
	compatibility := map[string][]string{
		"researcher":  {"analysis", "research", "investigation"},
		"coder":       {"coding", "development", "programming", "implementation"},
		"analyst":     {"analysis", "data_analysis", "investigation", "research"},
		"tester":      {"testing", "validation", "quality_assurance"},
		"designer":    {"design", "ui_design", "architecture"},
		"reviewer":    {"review", "code_review", "documentation_review"},
		"optimizer":   {"optimization", "performance", "refactoring"},
		"coordinator": {"coordination", "management", "orchestration"},
	}

	compatibleTypes, exists := compatibility[agentType]
	if !exists {
		return false
	}

	for _, compatibleType := range compatibleTypes {
		if compatibleType == taskType {
			return true
		}
	}

	return false
}

// executeTask executes a task with an agent
func (c *Coordinator) executeTask(task *Task, agent *AgentInstance) {
	startTime := time.Now()

	// Update agent to show current task
	agent.CurrentTask = task
	agent.LastActive = time.Now()
	agent.TaskCount++

	// Simulate realistic task execution time based on task type and complexity
	executionTime := c.calculateExecutionTime(task, agent)
	time.Sleep(executionTime)

	// Calculate response time
	responseTime := time.Since(startTime)

	// Update agent's average response time
	if agent.AvgResponseTime == 0 {
		agent.AvgResponseTime = float64(responseTime.Milliseconds())
	} else {
		agent.AvgResponseTime = (agent.AvgResponseTime + float64(responseTime.Milliseconds())) / 2
	}

	// Determine success based on agent capability and task complexity
	successProbability := c.calculateSuccessProbability(task, agent)
	success := c.simulateTaskExecution(successProbability)

	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if success {
		task.Status = "completed"
		task.Result = map[string]interface{}{
			"status":        "success",
			"data":          c.generateTaskResult(task, agent),
			"response_time": responseTime.Milliseconds(),
			"agent_type":    agent.Type,
		}
		// Improve success rate gradually
		agent.SuccessRate = math.Min(1.0, agent.SuccessRate+0.05)
	} else {
		task.Status = "failed"
		task.Error = c.generateTaskError(task, agent)
		// Decrease success rate slightly
		agent.SuccessRate = math.Max(0.0, agent.SuccessRate-0.02)
	}

	agent.CurrentTask = nil
	c.updateAgentInDB(agent)

	// Update task in database
	c.updateTaskInDB(task)
}

// calculateExecutionTime calculates realistic execution time for a task
func (c *Coordinator) calculateExecutionTime(task *Task, agent *AgentInstance) time.Duration {
	baseTime := 50 * time.Millisecond

	// Factor in task priority (higher priority might be more complex)
	priorityFactor := float64(task.Priority)

	// Factor in agent experience (higher success rate = faster execution)
	experienceFactor := 2.0 - agent.SuccessRate

	// Factor in task type complexity
	complexityFactor := c.getTaskComplexityFactor(task.Type)

	totalTime := float64(baseTime) * priorityFactor * experienceFactor * complexityFactor
	return time.Duration(totalTime)
}

// calculateSuccessProbability calculates the probability of task success
func (c *Coordinator) calculateSuccessProbability(task *Task, agent *AgentInstance) float64 {
	baseProbability := agent.SuccessRate

	// Exact type match increases success probability
	if agent.Type == task.Type {
		baseProbability += 0.2
	} else if c.isCompatibleType(agent.Type, task.Type) {
		baseProbability += 0.1
	} else {
		baseProbability -= 0.2
	}

	// Higher priority tasks might be more challenging
	priorityPenalty := (float64(task.Priority) - 2.0) * 0.05
	baseProbability -= priorityPenalty

	return math.Max(0.1, math.Min(0.95, baseProbability))
}

// simulateTaskExecution simulates task execution based on success probability
func (c *Coordinator) simulateTaskExecution(successProbability float64) bool {
	// Use a deterministic approach based on current time and success probability
	randomValue := float64((time.Now().UnixNano() % 1000)) / 1000.0
	return randomValue < successProbability
}

// getTaskComplexityFactor returns complexity factor for different task types
func (c *Coordinator) getTaskComplexityFactor(taskType string) float64 {
	complexity := map[string]float64{
		"research":       1.5,
		"analysis":       1.3,
		"coding":         2.0,
		"development":    2.0,
		"testing":        1.2,
		"review":         1.0,
		"optimization":   1.8,
		"coordination":   1.1,
		"design":         1.6,
		"implementation": 1.9,
	}

	if factor, exists := complexity[taskType]; exists {
		return factor
	}
	return 1.0
}

// generateTaskResult generates a realistic task result
func (c *Coordinator) generateTaskResult(task *Task, agent *AgentInstance) string {
	results := map[string]string{
		"research":     "Research findings compiled with key insights and recommendations",
		"analysis":     "Data analysis completed with statistical findings and visualizations",
		"coding":       "Code implementation completed with unit tests and documentation",
		"development":  "Feature development completed with integration tests",
		"testing":      "Testing completed with test report and bug findings",
		"review":       "Code review completed with feedback and improvement suggestions",
		"optimization": "Performance optimization completed with 25% improvement metrics",
		"coordination": "Task coordination completed with updated project timeline",
		"design":       "Design specifications completed with mockups and technical requirements",
	}

	if result, exists := results[task.Type]; exists {
		return fmt.Sprintf("%s (executed by %s agent)", result, agent.Type)
	}
	return fmt.Sprintf("Task of type '%s' completed successfully by %s agent", task.Type, agent.Type)
}

// generateTaskError generates a realistic task error message
func (c *Coordinator) generateTaskError(task *Task, agent *AgentInstance) string {
	errors := map[string]string{
		"research":     "Research task failed due to insufficient data sources",
		"analysis":     "Analysis task failed due to data quality issues",
		"coding":       "Coding task failed due to compilation errors",
		"development":  "Development task failed due to dependency conflicts",
		"testing":      "Testing task failed due to environment setup issues",
		"review":       "Review task failed due to incomplete code submission",
		"optimization": "Optimization task failed due to performance constraints",
		"coordination": "Coordination task failed due to communication issues",
		"design":       "Design task failed due to unclear requirements",
	}

	if errorMsg, exists := errors[task.Type]; exists {
		return fmt.Sprintf("%s (attempted by %s agent)", errorMsg, agent.Type)
	}
	return fmt.Sprintf("Task of type '%s' failed during execution by %s agent", task.Type, agent.Type)
}

// updateTaskInDB updates task status in database
func (c *Coordinator) updateTaskInDB(task *Task) {
	taskID, err := uuid.Parse(task.ID)
	if err != nil {
		c.logger.WithError(err).WithField("task_id", task.ID).Error("Invalid task ID format")
		return
	}

	updates := map[string]interface{}{
		"status": task.Status,
	}

	if task.StartedAt != nil {
		updates["started_at"] = *task.StartedAt
	}
	if task.CompletedAt != nil {
		updates["completed_at"] = *task.CompletedAt
	}
	if task.Result != nil {
		resultJSON, _ := json.Marshal(task.Result)
		updates["result"] = string(resultJSON)
	}
	if task.Error != "" {
		updates["error_message"] = task.Error
	}

	if err := c.db.DB.Model(&models.Task{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		c.logger.WithError(err).WithField("task_id", task.ID).Error("Failed to update task in database")
	}
}

// loadAgents loads existing agents from the database
func (c *Coordinator) loadAgents() error {
	var dbAgents []models.Agent
	if err := c.db.DB.Where("status IN ?", []string{"active", "inactive"}).Find(&dbAgents).Error; err != nil {
		return fmt.Errorf("failed to load agents from database: %w", err)
	}

	c.agentsMux.Lock()
	defer c.agentsMux.Unlock()

	for _, dbAgent := range dbAgents {
		agent := &AgentInstance{
			ID:              dbAgent.Name,
			Type:            dbAgent.Type,
			Status:          dbAgent.Status,
			Config:          make(map[string]interface{}),
			CreatedAt:       dbAgent.CreatedAt,
			TaskCount:       dbAgent.TaskCount,
			SuccessRate:     dbAgent.SuccessRate,
			AvgResponseTime: dbAgent.AvgResponseTime,
		}

		if dbAgent.LastActive != nil {
			agent.LastActive = *dbAgent.LastActive
		}

		c.agents[agent.ID] = agent
	}

	c.logger.WithField("agent_count", len(c.agents)).Info("Loaded agents from database")
	return nil
}

// updateAgentInDB updates an agent in the database
func (c *Coordinator) updateAgentInDB(agent *AgentInstance) {
	updates := map[string]interface{}{
		"status":            agent.Status,
		"last_active":       agent.LastActive,
		"task_count":        agent.TaskCount,
		"success_rate":      agent.SuccessRate,
		"avg_response_time": agent.AvgResponseTime,
	}

	if err := c.db.DB.Model(&models.Agent{}).Where("name = ?", agent.ID).Updates(updates).Error; err != nil {
		c.logger.WithError(err).WithField("agent_id", agent.ID).Error("Failed to update agent in database")
	}
}

// agentMonitor monitors agent health and activity
func (c *Coordinator) agentMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.checkAgentHealth()
		case <-c.ctx.Done():
			return
		}
	}
}

// taskScheduler handles task scheduling and prioritization
func (c *Coordinator) taskScheduler() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.schedulePendingTasks()
			c.optimizeAgentAssignments()
		case <-c.ctx.Done():
			return
		}
	}
}

// schedulePendingTasks schedules pending tasks based on priority and agent availability
func (c *Coordinator) schedulePendingTasks() {
	// Load pending tasks from database
	var pendingTasks []models.Task
	if err := c.db.DB.Where("status = ?", "pending").Order("priority desc, created_at asc").Find(&pendingTasks).Error; err != nil {
		c.logger.WithError(err).Error("Failed to load pending tasks")
		return
	}

	for _, dbTask := range pendingTasks {
		// Convert to coordinator task format
		task := &Task{
			ID:       dbTask.ID.String(),
			Type:     dbTask.Type,
			Priority: c.getPriorityScore(dbTask.Priority),
			Data: map[string]interface{}{
				"title":       dbTask.Title,
				"description": dbTask.Description,
				"project_id":  dbTask.ProjectID,
				"assigned_to": dbTask.AssignedTo,
			},
			CreatedAt: dbTask.CreatedAt,
			Status:    dbTask.Status,
		}

		// Try to submit task to queue
		if err := c.SubmitTask(task); err != nil {
			c.logger.WithError(err).WithField("task_id", task.ID).Debug("Failed to submit task to queue")
			break // Queue is full, stop trying
		}

		// Update task status in database
		if err := c.db.DB.Model(&dbTask).Update("status", "queued").Error; err != nil {
			c.logger.WithError(err).WithField("task_id", task.ID).Error("Failed to update task status")
		}
	}
}

// optimizeAgentAssignments reassigns tasks to optimize performance
func (c *Coordinator) optimizeAgentAssignments() {
	c.agentsMux.RLock()
	defer c.agentsMux.RUnlock()

	// Identify overloaded and underloaded agents
	var overloaded, underloaded []*AgentInstance
	avgTaskCount := c.calculateAverageTaskCount()

	for _, agent := range c.agents {
		if agent.Status != "active" {
			continue
		}

		if float64(agent.TaskCount) > avgTaskCount*1.5 {
			overloaded = append(overloaded, agent)
		} else if float64(agent.TaskCount) < avgTaskCount*0.5 {
			underloaded = append(underloaded, agent)
		}
	}

	// Log optimization opportunity
	if len(overloaded) > 0 && len(underloaded) > 0 {
		c.logger.WithFields(logrus.Fields{
			"overloaded_agents":  len(overloaded),
			"underloaded_agents": len(underloaded),
			"avg_task_count":     avgTaskCount,
		}).Debug("Found optimization opportunity for task rebalancing")
	}
}

// getPriorityScore converts string priority to numeric score
func (c *Coordinator) getPriorityScore(priority string) int {
	switch strings.ToLower(priority) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 2
	}
}

// calculateAverageTaskCount calculates the average task count across active agents
func (c *Coordinator) calculateAverageTaskCount() float64 {
	totalTasks := 0
	activeAgents := 0

	for _, agent := range c.agents {
		if agent.Status == "active" {
			totalTasks += agent.TaskCount
			activeAgents++
		}
	}

	if activeAgents == 0 {
		return 0
	}

	return float64(totalTasks) / float64(activeAgents)
}

// checkAgentHealth checks the health of all agents
func (c *Coordinator) checkAgentHealth() error {
	c.agentsMux.RLock()
	defer c.agentsMux.RUnlock()

	now := time.Now()
	for _, agent := range c.agents {
		// Check if agent has been inactive for too long
		if now.Sub(agent.LastActive) > 5*time.Minute && agent.Status == "active" {
			agent.Status = "inactive"
			c.updateAgentInDB(agent)
			c.logger.WithField("agent_id", agent.ID).Warn("Agent marked as inactive due to inactivity")
		}
	}

	return nil
}

// rebalanceWorkload rebalances tasks among agents
func (c *Coordinator) rebalanceWorkload() error {
	c.agentsMux.Lock()
	defer c.agentsMux.Unlock()

	activeAgents := make([]*AgentInstance, 0)
	for _, agent := range c.agents {
		if agent.Status == "active" {
			activeAgents = append(activeAgents, agent)
		}
	}

	if len(activeAgents) < 2 {
		return nil // Need at least 2 agents to rebalance
	}

	// Sort agents by task count (descending)
	sort.Slice(activeAgents, func(i, j int) bool {
		return activeAgents[i].TaskCount > activeAgents[j].TaskCount
	})

	maxTasks := activeAgents[0].TaskCount
	minTasks := activeAgents[len(activeAgents)-1].TaskCount

	// If difference is significant, perform rebalancing
	if maxTasks-minTasks > 2 {
		c.logger.WithFields(logrus.Fields{
			"max_tasks": maxTasks,
			"min_tasks": minTasks,
			"agents":    len(activeAgents),
		}).Info("Performing workload rebalancing")

		// Move tasks from overloaded to underloaded agents
		for i := 0; i < len(activeAgents)/2; i++ {
			overloaded := activeAgents[i]
			underloaded := activeAgents[len(activeAgents)-1-i]

			if overloaded.TaskCount > underloaded.TaskCount+1 {
				// Simulate task reassignment
				c.reassignTask(overloaded, underloaded)
			}
		}
	}

	return nil
}

// reassignTask reassigns a task from one agent to another
func (c *Coordinator) reassignTask(from, to *AgentInstance) {
	if from.TaskCount > 0 {
		from.TaskCount--
		to.TaskCount++

		c.logger.WithFields(logrus.Fields{
			"from_agent": from.ID,
			"to_agent":   to.ID,
			"from_count": from.TaskCount,
			"to_count":   to.TaskCount,
		}).Debug("Reassigned task between agents")

		// Update both agents in database
		c.updateAgentInDB(from)
		c.updateAgentInDB(to)
	}
}

// scaleAgents scales agents up or down based on demand
func (c *Coordinator) scaleAgents() error {
	queueLength := len(c.taskQueue)
	activeAgents := c.getActiveAgentCount()

	// Scale up if queue is getting full
	if queueLength > c.config.QueueSize/2 && activeAgents < c.config.AgentMaxCount {
		_, err := c.CreateAgent("auto-scaler", map[string]interface{}{
			"auto_created": true,
			"created_at":   time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to scale up agents: %w", err)
		}
		c.logger.Info("Scaled up agents due to high queue length")
	}

	return nil
}

// getActiveAgentCount returns the number of active agents
func (c *Coordinator) getActiveAgentCount() int {
	c.agentsMux.RLock()
	defer c.agentsMux.RUnlock()

	count := 0
	for _, agent := range c.agents {
		if agent.Status == "active" {
			count++
		}
	}
	return count
}

// updateStats safely updates coordinator statistics
func (c *Coordinator) updateStats(updater func(*CoordinatorStats)) {
	c.statsMux.Lock()
	defer c.statsMux.Unlock()
	updater(c.stats)
}

// updateLastActivity updates the last activity timestamp
func (c *Coordinator) updateLastActivity() {
	c.updateStats(func(s *CoordinatorStats) {
		s.LastActivity = time.Now()
	})
}
