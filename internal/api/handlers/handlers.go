package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/activity"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/auth"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/monitoring"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd/coordinator"
	ws "github.com/kooshapari/kaskmanager-rd-platform/internal/websocket"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	db              *database.Database
	wsHub           *ws.Hub
	rndModule       *rnd.Module
	monitor         *monitoring.Monitor
	logger          *logrus.Logger
	authService     *auth.Service
	activityService *activity.Service
	upgrader        websocket.Upgrader
}

// NewHandlers creates a new handlers instance
func NewHandlers(
	db *database.Database,
	wsHub *ws.Hub,
	rndModule *rnd.Module,
	monitor *monitoring.Monitor,
	logger *logrus.Logger,
	authService *auth.Service,
	activityService *activity.Service,
) *Handlers {
	return &Handlers{
		db:              db,
		wsHub:           wsHub,
		rndModule:       rndModule,
		monitor:         monitor,
		logger:          logger,
		authService:     authService,
		activityService: activityService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}
}

// Health and Monitoring Handlers

// GetHealth returns system health status
func (h *Handlers) GetHealth(c *gin.Context) {
	health := h.monitor.GetHealth()
	c.JSON(http.StatusOK, health)
}

// GetMetrics returns system metrics
func (h *Handlers) GetMetrics(c *gin.Context) {
	metrics := h.monitor.GetMetrics()
	c.JSON(http.StatusOK, metrics)
}

// GetSystemStatus returns detailed system status
func (h *Handlers) GetSystemStatus(c *gin.Context) {
	status := h.monitor.GetSystemStatus()
	status["database"] = h.db.GetStats()
	status["rnd_module"] = h.rndModule.Health()
	status["websocket_clients"] = h.wsHub.GetClientCount()

	c.JSON(http.StatusOK, status)
}

// WebSocket Handler

// HandleWebSocket handles WebSocket connections
func (h *Handlers) HandleWebSocket(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	client := ws.NewClient(h.wsHub, conn, h.logger)
	h.wsHub.RegisterClient(client)

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}

// Authentication Handlers

// Project Handlers

// GetProjects returns all projects
func (h *Handlers) GetProjects(c *gin.Context) {
	var projects []models.Project

	query := h.db.DB.Preload("Creator").Preload("Tasks")

	// Handle query parameters
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if projectType := c.Query("type"); projectType != "" {
		query = query.Where("type = ?", projectType)
	}

	if err := query.Find(&projects).Error; err != nil {
		h.logger.WithError(err).Error("Failed to get projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"total":    len(projects),
	})
}

// CreateProject creates a new project
func (h *Handlers) CreateProject(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Type        string `json:"type" binding:"required"`
		Priority    string `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userIDStr, _ := c.Get("user_id")
	creatorID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		creatorID = uuid.New() // Fallback for demo
	}

	project := models.Project{
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Priority:    req.Priority,
		Status:      "active",
		Progress:    0,
		CreatedBy:   creatorID,
	}

	if err := h.db.DB.Create(&project).Error; err != nil {
		h.logger.WithError(err).Error("Failed to create project")

		// Log failed project creation activity
		userID, username := h.activityService.ExtractUserInfoFromContext(c)
		h.activityService.LogCRUDActivity(
			c.Request.Context(),
			userID,
			username,
			activity.ActivityTypeProjectCreate,
			activity.ResourceTypeProject,
			nil,
			map[string]interface{}{
				"name":  req.Name,
				"type":  req.Type,
				"error": err.Error(),
			},
			false,
			c.ClientIP(),
			c.Request.UserAgent(),
			err.Error(),
		)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	// Log successful project creation activity
	userID, username := h.activityService.ExtractUserInfoFromContext(c)
	h.activityService.LogCRUDActivity(
		c.Request.Context(),
		userID,
		username,
		activity.ActivityTypeProjectCreate,
		activity.ResourceTypeProject,
		&project.ID,
		map[string]interface{}{
			"name":        project.Name,
			"type":        project.Type,
			"priority":    project.Priority,
			"description": project.Description,
		},
		true,
		c.ClientIP(),
		c.Request.UserAgent(),
		"",
	)

	// Broadcast project update
	h.wsHub.BroadcastProjectUpdate(project)

	c.JSON(http.StatusCreated, project)
}

// GetProject returns a specific project
func (h *Handlers) GetProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var project models.Project
	if err := h.db.DB.Preload("Creator").Preload("Tasks").First(&project, "id = ?", projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get project")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
		return
	}

	c.JSON(http.StatusOK, project)
}

// UpdateProject updates a project
func (h *Handlers) UpdateProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var project models.Project
	if err := h.db.DB.First(&project, "id = ?", projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project"})
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		Priority    *string `json:"priority"`
		Progress    *int    `json:"progress"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Track changes for activity logging
	changes := make(map[string]interface{})
	oldValues := make(map[string]interface{})
	newValues := make(map[string]interface{})

	// Update fields if provided and track changes
	if req.Name != nil && project.Name != *req.Name {
		oldValues["name"] = project.Name
		newValues["name"] = *req.Name
		changes["name"] = map[string]interface{}{"from": project.Name, "to": *req.Name}
		project.Name = *req.Name
	}
	if req.Description != nil && project.Description != *req.Description {
		oldValues["description"] = project.Description
		newValues["description"] = *req.Description
		changes["description"] = map[string]interface{}{"from": project.Description, "to": *req.Description}
		project.Description = *req.Description
	}
	if req.Status != nil && project.Status != *req.Status {
		oldValues["status"] = project.Status
		newValues["status"] = *req.Status
		changes["status"] = map[string]interface{}{"from": project.Status, "to": *req.Status}
		project.Status = *req.Status
	}
	if req.Priority != nil && project.Priority != *req.Priority {
		oldValues["priority"] = project.Priority
		newValues["priority"] = *req.Priority
		changes["priority"] = map[string]interface{}{"from": project.Priority, "to": *req.Priority}
		project.Priority = *req.Priority
	}
	if req.Progress != nil && project.Progress != *req.Progress {
		oldValues["progress"] = project.Progress
		newValues["progress"] = *req.Progress
		changes["progress"] = map[string]interface{}{"from": project.Progress, "to": *req.Progress}
		project.Progress = *req.Progress
	}

	if err := h.db.DB.Save(&project).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update project")

		// Log failed project update activity
		userID, username := h.activityService.ExtractUserInfoFromContext(c)
		h.activityService.LogCRUDActivity(
			c.Request.Context(),
			userID,
			username,
			activity.ActivityTypeProjectUpdate,
			activity.ResourceTypeProject,
			&project.ID,
			map[string]interface{}{
				"attempted_changes": changes,
				"error":             err.Error(),
			},
			false,
			c.ClientIP(),
			c.Request.UserAgent(),
			err.Error(),
		)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	// Log successful project update activity (only if there were actual changes)
	if len(changes) > 0 {
		userID, username := h.activityService.ExtractUserInfoFromContext(c)
		h.activityService.LogCRUDActivity(
			c.Request.Context(),
			userID,
			username,
			activity.ActivityTypeProjectUpdate,
			activity.ResourceTypeProject,
			&project.ID,
			map[string]interface{}{
				"changes":    changes,
				"old_values": oldValues,
				"new_values": newValues,
			},
			true,
			c.ClientIP(),
			c.Request.UserAgent(),
			"",
		)
	}

	// Broadcast project update
	h.wsHub.BroadcastProjectUpdate(project)

	c.JSON(http.StatusOK, project)
}

// DeleteProject deletes a project
func (h *Handlers) DeleteProject(c *gin.Context) {
	id := c.Param("id")
	projectID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Get project details before deletion for logging
	var project models.Project
	if err := h.db.DB.First(&project, "id = ?", projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find project"})
		return
	}

	if err := h.db.DB.Delete(&models.Project{}, "id = ?", projectID).Error; err != nil {
		h.logger.WithError(err).Error("Failed to delete project")

		// Log failed project deletion activity
		userID, username := h.activityService.ExtractUserInfoFromContext(c)
		h.activityService.LogCRUDActivity(
			c.Request.Context(),
			userID,
			username,
			activity.ActivityTypeProjectDelete,
			activity.ResourceTypeProject,
			&projectID,
			map[string]interface{}{
				"project_name": project.Name,
				"project_type": project.Type,
				"error":        err.Error(),
			},
			false,
			c.ClientIP(),
			c.Request.UserAgent(),
			err.Error(),
		)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}

	// Log successful project deletion activity
	userID, username := h.activityService.ExtractUserInfoFromContext(c)
	h.activityService.LogCRUDActivity(
		c.Request.Context(),
		userID,
		username,
		activity.ActivityTypeProjectDelete,
		activity.ResourceTypeProject,
		&projectID,
		map[string]interface{}{
			"project_name": project.Name,
			"project_type": project.Type,
		},
		true,
		c.ClientIP(),
		c.Request.UserAgent(),
		"",
	)

	c.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

// Agent Handlers

// GetAgents returns all agents
func (h *Handlers) GetAgents(c *gin.Context) {
	agents := h.rndModule.GetCoordinator().GetAgents()
	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"total":  len(agents),
	})
}

// CreateAgent creates a new agent
func (h *Handlers) CreateAgent(c *gin.Context) {
	var req struct {
		Type   string                 `json:"type" binding:"required"`
		Config map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent, err := h.rndModule.GetCoordinator().CreateAgent(req.Type, req.Config)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent"})
		return
	}

	// Broadcast agent update
	h.wsHub.BroadcastAgentUpdate(agent)

	c.JSON(http.StatusCreated, agent)
}

// GetAgent returns a specific agent
func (h *Handlers) GetAgent(c *gin.Context) {
	id := c.Param("id")

	agent, err := h.rndModule.GetCoordinator().GetAgent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// UpdateAgent updates an agent
func (h *Handlers) UpdateAgent(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string                 `json:"status"`
		Config map[string]interface{} `json:"config"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agent, err := h.rndModule.GetCoordinator().GetAgent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	// Update agent properties
	if req.Status != "" {
		agent.Status = req.Status
	}
	if req.Config != nil {
		agent.Config = req.Config
	}

	// Update in database through coordinator
	// Since we don't have a direct update method, we'll simulate it by updating the internal state
	// In a real implementation, you'd add an UpdateAgent method to the coordinator

	// Log agent update activity
	userID, username := h.activityService.ExtractUserInfoFromContext(c)
	h.activityService.LogCRUDActivity(
		c.Request.Context(),
		userID,
		username,
		activity.ActivityTypeAgentUpdate,
		activity.ResourceTypeAgent,
		nil, // Agent doesn't have UUID ID
		map[string]interface{}{
			"agent_id": agent.ID,
			"status":   agent.Status,
			"type":     agent.Type,
		},
		true,
		c.ClientIP(),
		c.Request.UserAgent(),
		"",
	)

	// Broadcast agent update
	h.wsHub.BroadcastAgentUpdate(agent)

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent updated successfully",
		"agent":   agent,
	})
}

// DeleteAgent deletes an agent
func (h *Handlers) DeleteAgent(c *gin.Context) {
	id := c.Param("id")

	agent, err := h.rndModule.GetCoordinator().GetAgent(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	// Check if agent has active tasks
	if agent.CurrentTask != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":        "Cannot delete agent with active tasks",
			"current_task": agent.CurrentTask.ID,
		})
		return
	}

	// Mark agent as inactive first
	agent.Status = "inactive"

	// Delete agent from database by marking as deleted
	if err := h.db.DB.Where("name = ?", agent.ID).Delete(&models.Agent{}).Error; err != nil {
		h.logger.WithError(err).Error("Failed to delete agent from database")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	// Log agent deletion activity
	userID, username := h.activityService.ExtractUserInfoFromContext(c)
	h.activityService.LogCRUDActivity(
		c.Request.Context(),
		userID,
		username,
		activity.ActivityTypeAgentDelete,
		activity.ResourceTypeAgent,
		nil, // Agent doesn't have UUID ID
		map[string]interface{}{
			"agent_id":     agent.ID,
			"agent_type":   agent.Type,
			"task_count":   agent.TaskCount,
			"success_rate": agent.SuccessRate,
		},
		true,
		c.ClientIP(),
		c.Request.UserAgent(),
		"",
	)

	// Note: In a real implementation, you'd want to add a DeleteAgent method to the coordinator
	// to properly remove the agent from the internal maps and handle cleanup

	c.JSON(http.StatusOK, gin.H{
		"message":  "Agent deleted successfully",
		"agent_id": agent.ID,
	})
}

// Dashboard Handlers for existing web UI

// GetDashboardData returns dashboard overview data
func (h *Handlers) GetDashboardData(c *gin.Context) {
	// Get project count
	var projectCount int64
	h.db.DB.Model(&models.Project{}).Where("status = ?", "active").Count(&projectCount)

	// Get task count
	var taskCount int64
	h.db.DB.Model(&models.Task{}).Where("status IN ?", []string{"pending", "in_progress"}).Count(&taskCount)

	// Get agent count
	agents := h.rndModule.GetCoordinator().GetAgents()
	activeAgents := 0
	for _, agent := range agents {
		if agent.Status == "active" {
			activeAgents++
		}
	}

	// Get completed tasks today
	today := time.Now().Truncate(24 * time.Hour)
	var completedToday int64
	h.db.DB.Model(&models.Task{}).Where("status = ? AND completed_at >= ?", "completed", today).Count(&completedToday)

	data := gin.H{
		"active_projects": projectCount,
		"pending_tasks":   taskCount,
		"active_agents":   activeAgents,
		"completed_today": completedToday,
		"system_status":   "online",
		"last_update":     time.Now(),
	}

	c.JSON(http.StatusOK, data)
}

// GetDashboardProjects returns projects for dashboard
func (h *Handlers) GetDashboardProjects(c *gin.Context) {
	var projects []models.Project

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if err := h.db.DB.Preload("Creator").Limit(limit).Order("updated_at DESC").Find(&projects).Error; err != nil {
		h.logger.WithError(err).Error("Failed to get dashboard projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"total":    len(projects),
	})
}

// GetDashboardAgents returns agents for dashboard
func (h *Handlers) GetDashboardAgents(c *gin.Context) {
	agents := h.rndModule.GetCoordinator().GetAgents()
	c.JSON(http.StatusOK, gin.H{
		"agents": agents,
		"total":  len(agents),
	})
}

// GetDashboardMetrics returns metrics for dashboard
func (h *Handlers) GetDashboardMetrics(c *gin.Context) {
	metrics := h.monitor.GetMetrics()
	c.JSON(http.StatusOK, metrics)
}

// GetDashboardActivities returns recent activities for dashboard
func (h *Handlers) GetDashboardActivities(c *gin.Context) {
	var activities []models.ActivityLog

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if err := h.db.DB.Preload("User").Limit(limit).Order("created_at DESC").Find(&activities).Error; err != nil {
		h.logger.WithError(err).Error("Failed to get dashboard activities")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activities"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"total":      len(activities),
	})
}

// R&D Operation Handlers

// AnalyzePatterns triggers pattern analysis
func (h *Handlers) AnalyzePatterns(c *gin.Context) {
	if err := h.rndModule.AnalyzePatterns(); err != nil {
		h.logger.WithError(err).Error("Failed to analyze patterns")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze patterns"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pattern analysis started"})
}

// GenerateProjects triggers project generation
func (h *Handlers) GenerateProjects(c *gin.Context) {
	if err := h.rndModule.GenerateProjects(); err != nil {
		h.logger.WithError(err).Error("Failed to generate projects")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project generation started"})
}

// CoordinateAgents triggers agent coordination
func (h *Handlers) CoordinateAgents(c *gin.Context) {
	if err := h.rndModule.CoordinateAgents(); err != nil {
		h.logger.WithError(err).Error("Failed to coordinate agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to coordinate agents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent coordination started"})
}

// GetRnDStats returns R&D module statistics
func (h *Handlers) GetRnDStats(c *gin.Context) {
	stats := h.rndModule.GetStats()
	c.JSON(http.StatusOK, stats)
}

// Placeholder handlers for other endpoints

func (h *Handlers) GetProjectTasks(c *gin.Context)   { h.notImplemented(c) }
func (h *Handlers) CreateProjectTask(c *gin.Context) { h.notImplemented(c) }

// GetTasks returns all tasks with optional filtering
func (h *Handlers) GetTasks(c *gin.Context) {
	// Parse query parameters
	status := c.Query("status")
	assignedTo := c.Query("assigned_to")
	projectID := c.Query("project_id")
	limit := 50 // Default limit
	offset := 0 // Default offset

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 { // Max limit
				limit = 100
			}
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build query
	query := h.db.DB.Model(&models.Task{}).
		Preload("Project").
		Preload("AssignedUser").
		Preload("Agent")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if assignedTo != "" {
		query = query.Where("assigned_to = ?", assignedTo)
	}
	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get tasks
	var tasks []models.Task
	err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&tasks).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch tasks")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch tasks",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// CreateTask creates a new task
func (h *Handlers) CreateTask(c *gin.Context) {
	var req struct {
		Title         string  `json:"title" binding:"required"`
		Description   string  `json:"description"`
		Type          string  `json:"type" binding:"required"`
		Priority      string  `json:"priority"`
		EstimatedTime int     `json:"estimated_time"`
		ProjectID     *string `json:"project_id"`
		AssignedTo    *string `json:"assigned_to"`
		AgentID       *string `json:"agent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Set default priority if not provided
	if req.Priority == "" {
		req.Priority = "medium"
	}

	// Create task
	task := models.Task{
		Title:         req.Title,
		Description:   req.Description,
		Type:          req.Type,
		Priority:      req.Priority,
		EstimatedTime: req.EstimatedTime,
		Status:        "pending",
	}

	// Set project ID if provided
	if req.ProjectID != nil {
		projectUUID, err := uuid.Parse(*req.ProjectID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid project ID",
				"message": err.Error(),
			})
			return
		}
		task.ProjectID = &projectUUID
	}

	// Set assigned user if provided
	if req.AssignedTo != nil {
		userUUID, err := uuid.Parse(*req.AssignedTo)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid assigned user ID",
				"message": err.Error(),
			})
			return
		}
		task.AssignedTo = &userUUID
	}

	// Set agent ID if provided
	if req.AgentID != nil {
		agentUUID, err := uuid.Parse(*req.AgentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid agent ID",
				"message": err.Error(),
			})
			return
		}
		task.AgentID = &agentUUID
	}

	// Save to database
	err := h.db.DB.Create(&task).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to create task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create task",
			"message": err.Error(),
		})
		return
	}

	// Load relationships for response
	h.db.DB.Preload("Project").Preload("AssignedUser").Preload("Agent").First(&task, task.ID)

	h.logger.WithField("task_id", task.ID).Info("Task created successfully")

	// Broadcast update via WebSocket
	h.wsHub.BroadcastToSubscribed(ws.Message{
		Type: ws.MessageTypeTaskUpdate,
		Data: map[string]interface{}{
			"action": "created",
			"task":   task,
		},
		Timestamp: time.Now().Unix(),
	}, "tasks")

	c.JSON(http.StatusCreated, gin.H{
		"message": "Task created successfully",
		"task":    task,
	})
}

// GetTask returns a specific task by ID
func (h *Handlers) GetTask(c *gin.Context) {
	taskID := c.Param("id")

	var task models.Task
	err := h.db.DB.Preload("Project").Preload("AssignedUser").Preload("Agent").
		Where("id = ?", taskID).First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch task",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task": task,
	})
}

// UpdateTask updates an existing task
func (h *Handlers) UpdateTask(c *gin.Context) {
	taskID := c.Param("id")

	var req struct {
		Title         *string `json:"title"`
		Description   *string `json:"description"`
		Type          *string `json:"type"`
		Status        *string `json:"status"`
		Priority      *string `json:"priority"`
		Progress      *int    `json:"progress"`
		EstimatedTime *int    `json:"estimated_time"`
		ActualTime    *int    `json:"actual_time"`
		Result        *string `json:"result"`
		ErrorMessage  *string `json:"error_message"`
		AssignedTo    *string `json:"assigned_to"`
		AgentID       *string `json:"agent_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Find existing task
	var task models.Task
	err := h.db.DB.Where("id = ?", taskID).First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find task",
			"message": err.Error(),
		})
		return
	}

	// Update fields if provided
	updateMap := make(map[string]interface{})

	if req.Title != nil {
		updateMap["title"] = *req.Title
	}
	if req.Description != nil {
		updateMap["description"] = *req.Description
	}
	if req.Type != nil {
		updateMap["type"] = *req.Type
	}
	if req.Status != nil {
		updateMap["status"] = *req.Status
		// Set completion time if status is completed
		if *req.Status == "completed" && task.CompletedAt == nil {
			now := time.Now()
			updateMap["completed_at"] = &now
		}
		// Set start time if status is in_progress and not already set
		if *req.Status == "in_progress" && task.StartedAt == nil {
			now := time.Now()
			updateMap["started_at"] = &now
		}
	}
	if req.Priority != nil {
		updateMap["priority"] = *req.Priority
	}
	if req.Progress != nil {
		updateMap["progress"] = *req.Progress
	}
	if req.EstimatedTime != nil {
		updateMap["estimated_time"] = *req.EstimatedTime
	}
	if req.ActualTime != nil {
		updateMap["actual_time"] = *req.ActualTime
	}
	if req.Result != nil {
		updateMap["result"] = *req.Result
	}
	if req.ErrorMessage != nil {
		updateMap["error_message"] = *req.ErrorMessage
	}

	// Handle assigned user update
	if req.AssignedTo != nil {
		if *req.AssignedTo == "" {
			updateMap["assigned_to"] = nil
		} else {
			userUUID, err := uuid.Parse(*req.AssignedTo)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid assigned user ID",
					"message": err.Error(),
				})
				return
			}
			updateMap["assigned_to"] = userUUID
		}
	}

	// Handle agent ID update
	if req.AgentID != nil {
		if *req.AgentID == "" {
			updateMap["agent_id"] = nil
		} else {
			agentUUID, err := uuid.Parse(*req.AgentID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Invalid agent ID",
					"message": err.Error(),
				})
				return
			}
			updateMap["agent_id"] = agentUUID
		}
	}

	// Perform update
	err = h.db.DB.Model(&task).Updates(updateMap).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to update task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update task",
			"message": err.Error(),
		})
		return
	}

	// Reload task with relationships
	h.db.DB.Preload("Project").Preload("AssignedUser").Preload("Agent").First(&task, task.ID)

	h.logger.WithField("task_id", task.ID).Info("Task updated successfully")

	// Broadcast update via WebSocket
	h.wsHub.BroadcastToSubscribed(ws.Message{
		Type: ws.MessageTypeTaskUpdate,
		Data: map[string]interface{}{
			"action": "updated",
			"task":   task,
		},
		Timestamp: time.Now().Unix(),
	}, "tasks")

	c.JSON(http.StatusOK, gin.H{
		"message": "Task updated successfully",
		"task":    task,
	})
}

// DeleteTask deletes a task
func (h *Handlers) DeleteTask(c *gin.Context) {
	taskID := c.Param("id")

	// Find task first to check if it exists
	var task models.Task
	err := h.db.DB.Where("id = ?", taskID).First(&task).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find task",
			"message": err.Error(),
		})
		return
	}

	// Soft delete the task
	err = h.db.DB.Delete(&task).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete task",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithField("task_id", task.ID).Info("Task deleted successfully")

	// Broadcast update via WebSocket
	h.wsHub.BroadcastToSubscribed(ws.Message{
		Type: ws.MessageTypeTaskUpdate,
		Data: map[string]interface{}{
			"action": "deleted",
			"task":   gin.H{"id": task.ID},
		},
		Timestamp: time.Now().Unix(),
	}, "tasks")

	c.JSON(http.StatusOK, gin.H{
		"message": "Task deleted successfully",
	})
}

// AssignTaskToAgent assigns a specific task to an agent
func (h *Handlers) AssignTaskToAgent(c *gin.Context) {
	taskID := c.Param("id")

	var req struct {
		AgentID string `json:"agent_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse task ID
	taskUUID, err := uuid.Parse(taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Get task from database
	var task models.Task
	if err := h.db.DB.First(&task, "id = ?", taskUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
		h.logger.WithError(err).Error("Failed to get task")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		return
	}

	// Check if task is available for assignment
	if task.Status != "pending" && task.Status != "queued" {
		c.JSON(http.StatusConflict, gin.H{
			"error":  "Task cannot be assigned",
			"status": task.Status,
		})
		return
	}

	// Get agent from coordinator
	agent, err := h.rndModule.GetCoordinator().GetAgent(req.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	// Check if agent is available
	if agent.Status != "active" {
		c.JSON(http.StatusConflict, gin.H{
			"error":        "Agent is not active",
			"agent_status": agent.Status,
		})
		return
	}

	if agent.CurrentTask != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":        "Agent is already assigned to a task",
			"current_task": agent.CurrentTask.ID,
		})
		return
	}

	// Create coordinator task and submit to queue
	coordinatorTask := &coordinator.Task{
		ID:       task.ID.String(),
		Type:     task.Type,
		Priority: getPriorityScore(task.Priority),
		Data: map[string]interface{}{
			"title":       task.Title,
			"description": task.Description,
			"project_id":  task.ProjectID,
			"assigned_to": req.AgentID,
		},
		CreatedAt:  task.CreatedAt,
		Status:     "assigned",
		AssignedTo: req.AgentID,
	}

	// Submit task to coordinator
	if err := h.rndModule.GetCoordinator().SubmitTask(coordinatorTask); err != nil {
		h.logger.WithError(err).Error("Failed to submit task to coordinator")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign task"})
		return
	}

	// Update task in database
	updates := map[string]interface{}{
		"status":   "assigned",
		"agent_id": parseAgentUUID(req.AgentID), // Helper function to convert agent ID to UUID if needed
	}

	if err := h.db.DB.Model(&task).Updates(updates).Error; err != nil {
		h.logger.WithError(err).Error("Failed to update task assignment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Log task assignment activity
	userID, username := h.activityService.ExtractUserInfoFromContext(c)
	h.activityService.LogCRUDActivity(
		c.Request.Context(),
		userID,
		username,
		activity.ActivityTypeTaskAssign,
		activity.ResourceTypeTask,
		&task.ID,
		map[string]interface{}{
			"task_id":   task.ID,
			"agent_id":  req.AgentID,
			"task_type": task.Type,
			"priority":  task.Priority,
		},
		true,
		c.ClientIP(),
		c.Request.UserAgent(),
		"",
	)

	// Broadcast task update
	h.wsHub.BroadcastTaskUpdate(&task)

	c.JSON(http.StatusOK, gin.H{
		"message": "Task assigned successfully",
		"task": gin.H{
			"id":       task.ID,
			"title":    task.Title,
			"status":   "assigned",
			"agent_id": req.AgentID,
		},
	})
}

// Helper function to get priority score
func getPriorityScore(priority string) int {
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

// Helper function to parse agent ID to UUID (if your system requires it)
func parseAgentUUID(agentID string) *uuid.UUID {
	// If agent IDs are UUIDs, parse them. Otherwise, return nil
	// For now, since agent IDs are strings, we'll return nil
	return nil
}

// GetProposals returns all proposals with optional filtering
func (h *Handlers) GetProposals(c *gin.Context) {
	// Parse query parameters
	status := c.Query("status")
	category := c.Query("category")
	priority := c.Query("priority")
	limit := 50 // Default limit
	offset := 0 // Default offset

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 {
				limit = 100
			}
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build query
	query := h.db.DB.Model(&models.Proposal{})

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get proposals
	var proposals []models.Proposal
	err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&proposals).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch proposals")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch proposals",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"proposals": proposals,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// CreateProposal creates a new project proposal
func (h *Handlers) CreateProposal(c *gin.Context) {
	var req struct {
		Title           string `json:"title" binding:"required"`
		Description     string `json:"description" binding:"required"`
		Category        string `json:"category" binding:"required"`
		Priority        string `json:"priority"`
		EstimatedEffort int    `json:"estimated_effort"`
		ExpectedOutcome string `json:"expected_outcome"`
		Justification   string `json:"justification"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Set default priority if not provided
	if req.Priority == "" {
		req.Priority = "medium"
	}

	// Create proposal
	proposal := models.Proposal{
		Title:           req.Title,
		Description:     req.Description,
		Category:        req.Category,
		Priority:        req.Priority,
		EstimatedEffort: req.EstimatedEffort,
		ExpectedOutcome: req.ExpectedOutcome,
		Justification:   req.Justification,
		Status:          "pending",
	}

	// Save to database
	err := h.db.DB.Create(&proposal).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to create proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create proposal",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithField("proposal_id", proposal.ID).Info("Proposal created successfully")

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Proposal created successfully",
		"proposal": proposal,
	})
}

// GetProposal returns a specific proposal by ID
func (h *Handlers) GetProposal(c *gin.Context) {
	proposalID := c.Param("id")

	var proposal models.Proposal
	err := h.db.DB.Where("id = ?", proposalID).First(&proposal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Proposal not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch proposal",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"proposal": proposal,
	})
}

// UpdateProposal updates an existing proposal
func (h *Handlers) UpdateProposal(c *gin.Context) {
	proposalID := c.Param("id")

	var req struct {
		Title           *string `json:"title"`
		Description     *string `json:"description"`
		Category        *string `json:"category"`
		Priority        *string `json:"priority"`
		EstimatedEffort *int    `json:"estimated_effort"`
		ExpectedOutcome *string `json:"expected_outcome"`
		Justification   *string `json:"justification"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Find existing proposal
	var proposal models.Proposal
	err := h.db.DB.Where("id = ?", proposalID).First(&proposal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Proposal not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find proposal",
			"message": err.Error(),
		})
		return
	}

	// Check if proposal can be edited (only pending proposals)
	if proposal.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot edit proposal that is not in pending status",
		})
		return
	}

	// Update fields if provided
	updateMap := make(map[string]interface{})

	if req.Title != nil {
		updateMap["title"] = *req.Title
	}
	if req.Description != nil {
		updateMap["description"] = *req.Description
	}
	if req.Category != nil {
		updateMap["category"] = *req.Category
	}
	if req.Priority != nil {
		updateMap["priority"] = *req.Priority
	}
	if req.EstimatedEffort != nil {
		updateMap["estimated_effort"] = *req.EstimatedEffort
	}
	if req.ExpectedOutcome != nil {
		updateMap["expected_outcome"] = *req.ExpectedOutcome
	}
	if req.Justification != nil {
		updateMap["justification"] = *req.Justification
	}

	// Perform update
	err = h.db.DB.Model(&proposal).Updates(updateMap).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to update proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update proposal",
			"message": err.Error(),
		})
		return
	}

	// Reload proposal
	h.db.DB.First(&proposal, proposal.ID)

	h.logger.WithField("proposal_id", proposal.ID).Info("Proposal updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":  "Proposal updated successfully",
		"proposal": proposal,
	})
}

// DeleteProposal deletes a proposal
func (h *Handlers) DeleteProposal(c *gin.Context) {
	proposalID := c.Param("id")

	// Find proposal first to check if it exists
	var proposal models.Proposal
	err := h.db.DB.Where("id = ?", proposalID).First(&proposal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Proposal not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find proposal",
			"message": err.Error(),
		})
		return
	}

	// Check if proposal can be deleted (only pending proposals)
	if proposal.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot delete proposal that is not in pending status",
		})
		return
	}

	// Soft delete the proposal
	err = h.db.DB.Delete(&proposal).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete proposal",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithField("proposal_id", proposal.ID).Info("Proposal deleted successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Proposal deleted successfully",
	})
}

// ApproveProposal approves a proposal
func (h *Handlers) ApproveProposal(c *gin.Context) {
	proposalID := c.Param("id")
	userID, _ := c.Get("user_id")

	var req struct {
		ReviewNotes string `json:"review_notes"`
	}
	c.ShouldBindJSON(&req)

	// Find proposal
	var proposal models.Proposal
	err := h.db.DB.Where("id = ?", proposalID).First(&proposal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Proposal not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find proposal",
			"message": err.Error(),
		})
		return
	}

	// Check if proposal is in pending status
	if proposal.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Can only approve proposals in pending status",
		})
		return
	}

	// Parse user ID
	reviewerUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Update proposal
	now := time.Now()
	updateMap := map[string]interface{}{
		"status":       "approved",
		"reviewed_at":  &now,
		"reviewed_by":  reviewerUUID,
		"review_notes": req.ReviewNotes,
	}

	err = h.db.DB.Model(&proposal).Updates(updateMap).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to approve proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to approve proposal",
			"message": err.Error(),
		})
		return
	}

	// Reload proposal
	h.db.DB.First(&proposal, proposal.ID)

	h.logger.WithFields(logrus.Fields{
		"proposal_id": proposal.ID,
		"reviewer_id": userID,
	}).Info("Proposal approved successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":  "Proposal approved successfully",
		"proposal": proposal,
	})
}

// RejectProposal rejects a proposal
func (h *Handlers) RejectProposal(c *gin.Context) {
	proposalID := c.Param("id")
	userID, _ := c.Get("user_id")

	var req struct {
		ReviewNotes string `json:"review_notes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Review notes are required for rejection",
			"message": err.Error(),
		})
		return
	}

	// Find proposal
	var proposal models.Proposal
	err := h.db.DB.Where("id = ?", proposalID).First(&proposal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Proposal not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find proposal",
			"message": err.Error(),
		})
		return
	}

	// Check if proposal is in pending status
	if proposal.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Can only reject proposals in pending status",
		})
		return
	}

	// Parse user ID
	reviewerUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Update proposal
	now := time.Now()
	updateMap := map[string]interface{}{
		"status":       "rejected",
		"reviewed_at":  &now,
		"reviewed_by":  reviewerUUID,
		"review_notes": req.ReviewNotes,
	}

	err = h.db.DB.Model(&proposal).Updates(updateMap).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to reject proposal")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to reject proposal",
			"message": err.Error(),
		})
		return
	}

	// Reload proposal
	h.db.DB.First(&proposal, proposal.ID)

	h.logger.WithFields(logrus.Fields{
		"proposal_id": proposal.ID,
		"reviewer_id": userID,
	}).Info("Proposal rejected successfully")

	c.JSON(http.StatusOK, gin.H{
		"message":  "Proposal rejected successfully",
		"proposal": proposal,
	})
}
func (h *Handlers) GetInsights(c *gin.Context) { h.notImplemented(c) }
func (h *Handlers) GetPatterns(c *gin.Context) { h.notImplemented(c) }

// GetCurrentUser returns the current user's profile (alias for GetProfile)
func (h *Handlers) GetCurrentUser(c *gin.Context) {
	h.GetProfile(c)
}

// UpdateCurrentUser updates the current user's profile
func (h *Handlers) UpdateCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
		Email     *string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Parse user ID
	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Find existing user
	var user models.User
	err = h.db.DB.Where("id = ?", userUUID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find user",
			"message": err.Error(),
		})
		return
	}

	// Prepare updates
	updateMap := make(map[string]interface{})

	if req.FirstName != nil {
		updateMap["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updateMap["last_name"] = *req.LastName
	}
	if req.Email != nil {
		// Check if email already exists
		var existingUser models.User
		err := h.db.DB.Where("email = ? AND id != ?", *req.Email, userUUID).First(&existingUser).Error
		if err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already exists",
			})
			return
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.WithError(err).Error("Database error checking email")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to check email",
			})
			return
		}
		updateMap["email"] = *req.Email
	}

	// If no fields to update
	if len(updateMap) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No fields to update",
		})
		return
	}

	// Perform update
	err = h.db.DB.Model(&user).Updates(updateMap).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to update user profile")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update profile",
			"message": err.Error(),
		})
		return
	}

	// Reload user
	h.db.DB.Select("id, username, email, first_name, last_name, role, is_active, is_verified, created_at, updated_at, last_login_at").
		First(&user, user.ID)

	h.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User profile updated successfully")

	// Log activity
	h.logActivity(c, "update_profile", "user", &user.ID, gin.H{
		"fields": updateMap,
	})

	// Broadcast user update
	h.wsHub.BroadcastUserUpdate(gin.H{
		"action": "profile_updated",
		"user":   sanitizeUser(user),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user":    sanitizeUser(user),
	})
}

// GetUsers returns all users with optional filtering and pagination
func (h *Handlers) GetUsers(c *gin.Context) {
	// Parse query parameters
	role := c.Query("role")
	isActiveStr := c.Query("is_active")
	search := c.Query("search")
	limit := 50 // Default limit
	offset := 0 // Default offset

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 { // Max limit
				limit = 100
			}
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build query
	query := h.db.DB.Model(&models.User{}).
		Select("id, username, email, first_name, last_name, role, is_active, is_verified, created_at, updated_at, last_login_at")

	// Apply filters
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			query = query.Where("is_active = ?", isActive)
		}
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"username ILIKE ? OR email ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get users
	var users []models.User
	err := query.Limit(limit).Offset(offset).Order("created_at DESC").Find(&users).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch users")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch users",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// CreateUser creates a new user (admin only)
func (h *Handlers) CreateUser(c *gin.Context) {
	// Check if user has admin role
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Admin access required",
		})
		return
	}

	var req struct {
		Username   string `json:"username" binding:"required,min=3,max=50"`
		Email      string `json:"email" binding:"required,email"`
		Password   string `json:"password" binding:"required,min=6"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		Role       string `json:"role"`
		IsActive   *bool  `json:"is_active"`
		IsVerified *bool  `json:"is_verified"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Check if user already exists
	var existingUser models.User
	err := h.db.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingUser).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error": "User already exists with this username or email",
		})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		h.logger.WithError(err).Error("Database error checking existing user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check existing user",
		})
		return
	}

	// Hash password
	hashedPassword, err := h.authService.HashPassword(req.Password)
	if err != nil {
		h.logger.WithError(err).Error("Failed to hash password")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to process password",
		})
		return
	}

	// Set default values
	role := req.Role
	if role == "" {
		role = "user"
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isVerified := false
	if req.IsVerified != nil {
		isVerified = *req.IsVerified
	}

	// Create user
	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         role,
		IsActive:     isActive,
		IsVerified:   isVerified,
	}

	err = h.db.DB.Create(&user).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to create user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create user",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"created_by": c.GetString("user_id"),
	}).Info("User created successfully")

	// Log activity
	h.logActivity(c, "create_user", "user", &user.ID, gin.H{
		"username": user.Username,
		"role":     user.Role,
	})

	// Broadcast user update
	h.wsHub.BroadcastUserUpdate(gin.H{
		"action": "created",
		"user":   sanitizeUser(user),
	})

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user":    sanitizeUser(user),
	})
}

// GetUser returns a specific user by ID
func (h *Handlers) GetUser(c *gin.Context) {
	userID := c.Param("id")

	// Parse UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	// Get current user ID for access control
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Check if user is trying to access their own profile or is admin
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == "admin"
	isSelf := currentUserID.(string) == userID

	if !isAdmin && !isSelf {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// Get user from database
	var user models.User
	err = h.db.DB.Select("id, username, email, first_name, last_name, role, is_active, is_verified, created_at, updated_at, last_login_at").
		Where("id = ?", userUUID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to fetch user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch user",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": sanitizeUser(user),
	})
}

// UpdateUser updates a user (admin only for most fields, users can update their own profile)
func (h *Handlers) UpdateUser(c *gin.Context) {
	userID := c.Param("id")

	// Parse UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	// Get current user ID for access control
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Check if user is trying to update their own profile or is admin
	userRole, _ := c.Get("user_role")
	isAdmin := userRole == "admin"
	isSelf := currentUserID.(string) == userID

	if !isAdmin && !isSelf {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	var req struct {
		Username   *string `json:"username"`
		Email      *string `json:"email"`
		FirstName  *string `json:"first_name"`
		LastName   *string `json:"last_name"`
		Role       *string `json:"role"`
		IsActive   *bool   `json:"is_active"`
		IsVerified *bool   `json:"is_verified"`
		Password   *string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Find existing user
	var user models.User
	err = h.db.DB.Where("id = ?", userUUID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find user",
			"message": err.Error(),
		})
		return
	}

	// Prepare updates
	updateMap := make(map[string]interface{})

	// Fields that both admin and self can update
	if req.FirstName != nil {
		updateMap["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updateMap["last_name"] = *req.LastName
	}

	// Fields that only admin can update
	if isAdmin {
		if req.Username != nil {
			// Check if username already exists
			var existingUser models.User
			err := h.db.DB.Where("username = ? AND id != ?", *req.Username, userUUID).First(&existingUser).Error
			if err == nil {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Username already exists",
				})
				return
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				h.logger.WithError(err).Error("Database error checking username")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to check username",
				})
				return
			}
			updateMap["username"] = *req.Username
		}

		if req.Email != nil {
			// Check if email already exists
			var existingUser models.User
			err := h.db.DB.Where("email = ? AND id != ?", *req.Email, userUUID).First(&existingUser).Error
			if err == nil {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Email already exists",
				})
				return
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				h.logger.WithError(err).Error("Database error checking email")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to check email",
				})
				return
			}
			updateMap["email"] = *req.Email
		}

		if req.Role != nil {
			updateMap["role"] = *req.Role
		}
		if req.IsActive != nil {
			updateMap["is_active"] = *req.IsActive
		}
		if req.IsVerified != nil {
			updateMap["is_verified"] = *req.IsVerified
		}
	}

	// Password update (admin or self)
	if req.Password != nil {
		if len(*req.Password) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Password must be at least 6 characters long",
			})
			return
		}

		hashedPassword, err := h.authService.HashPassword(*req.Password)
		if err != nil {
			h.logger.WithError(err).Error("Failed to hash password")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to process password",
			})
			return
		}
		updateMap["password_hash"] = hashedPassword
	}

	// If no fields to update
	if len(updateMap) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No fields to update",
		})
		return
	}

	// Perform update
	err = h.db.DB.Model(&user).Updates(updateMap).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to update user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update user",
			"message": err.Error(),
		})
		return
	}

	// Reload user
	h.db.DB.Select("id, username, email, first_name, last_name, role, is_active, is_verified, created_at, updated_at, last_login_at").
		First(&user, user.ID)

	h.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"updated_by": currentUserID,
	}).Info("User updated successfully")

	// Log activity
	h.logActivity(c, "update_user", "user", &user.ID, gin.H{
		"username": user.Username,
		"fields":   updateMap,
	})

	// Broadcast user update
	h.wsHub.BroadcastUserUpdate(gin.H{
		"action": "updated",
		"user":   sanitizeUser(user),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "User updated successfully",
		"user":    sanitizeUser(user),
	})
}

// DeleteUser deletes a user (admin only)
func (h *Handlers) DeleteUser(c *gin.Context) {
	// Check if user has admin role
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Admin access required",
		})
		return
	}

	userID := c.Param("id")

	// Parse UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	// Prevent admin from deleting themselves
	currentUserID, _ := c.Get("user_id")
	if currentUserID.(string) == userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot delete your own account",
		})
		return
	}

	// Find user first to check if it exists
	var user models.User
	err = h.db.DB.Where("id = ?", userUUID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find user",
			"message": err.Error(),
		})
		return
	}

	// Soft delete the user
	err = h.db.DB.Delete(&user).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to delete user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete user",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"deleted_by": currentUserID,
	}).Info("User deleted successfully")

	// Log activity
	h.logActivity(c, "delete_user", "user", &user.ID, gin.H{
		"username": user.Username,
	})

	// Broadcast user update
	h.wsHub.BroadcastUserUpdate(gin.H{
		"action": "deleted",
		"user":   gin.H{"id": user.ID, "username": user.Username},
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "User deleted successfully",
	})
}

// GetActivities returns activities with filtering and pagination
func (h *Handlers) GetActivities(c *gin.Context) {
	// Parse query parameters
	userID := c.Query("user_id")
	action := c.Query("action")
	resource := c.Query("resource")
	resourceID := c.Query("resource_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	search := c.Query("search")

	// Parse pagination parameters
	page := 1
	pageSize := 50
	sort := "created_at"
	order := "desc"

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 {
			pageSize = parsed
			if pageSize > 100 { // Max page size
				pageSize = 100
			}
		}
	}

	if s := c.Query("sort"); s != "" {
		sort = s
	}

	if o := c.Query("order"); o != "" {
		order = o
	}

	pagination := repositories.Pagination{
		Page:     page,
		PageSize: pageSize,
		Sort:     sort,
		Order:    order,
	}

	// Build filters
	filters := repositories.Filter{}

	if userID != "" {
		if parsedID, err := uuid.Parse(userID); err == nil {
			filters["user_id"] = parsedID
		}
	}

	if action != "" {
		filters["action"] = action
	}

	if resource != "" {
		filters["resource"] = resource
	}

	if resourceID != "" {
		if parsedID, err := uuid.Parse(resourceID); err == nil {
			filters["resource_id"] = parsedID
		}
	}

	// Handle date range filtering
	var result *repositories.PaginationResult
	var err error

	if startDate != "" && endDate != "" {
		start, err1 := time.Parse(time.RFC3339, startDate)
		end, err2 := time.Parse(time.RFC3339, endDate)

		if err1 != nil || err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid date format. Use RFC3339 format (e.g., 2023-01-01T00:00:00Z)",
			})
			return
		}

		result, err = h.activityService.GetActivitiesByDateRange(c.Request.Context(), start, end, pagination)
	} else if search != "" {
		result, err = h.activityService.SearchActivities(c.Request.Context(), search, pagination)
	} else {
		result, err = h.activityService.GetActivities(c.Request.Context(), filters, pagination)
	}

	if err != nil {
		h.logger.WithError(err).Error("Failed to get activities")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get activities",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": result.Data,
		"pagination": gin.H{
			"page":        result.Page,
			"page_size":   result.PageSize,
			"total":       result.Total,
			"total_pages": result.TotalPages,
		},
	})
}

// GetRecentActivities returns the most recent activities
func (h *Handlers) GetRecentActivities(c *gin.Context) {
	// Parse limit parameter
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 100 { // Max limit
				limit = 100
			}
		}
	}

	activities, err := h.activityService.GetRecentActivities(c.Request.Context(), limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get recent activities")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get recent activities",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"total":      len(activities),
		"limit":      limit,
	})
}
func (h *Handlers) GetSystemInfo(c *gin.Context)  { h.notImplemented(c) }
func (h *Handlers) GetSystemStats(c *gin.Context) { h.notImplemented(c) }
func (h *Handlers) CreateBackup(c *gin.Context)   { h.notImplemented(c) }
func (h *Handlers) RestoreBackup(c *gin.Context)  { h.notImplemented(c) }

// GetAllUsers returns all users for admin purposes
func (h *Handlers) GetAllUsers(c *gin.Context) {
	// Check if user has admin role
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Admin access required",
		})
		return
	}

	// Parse query parameters
	includeDeleted := c.Query("include_deleted") == "true"
	role := c.Query("role")
	isActiveStr := c.Query("is_active")
	search := c.Query("search")
	sortBy := c.Query("sort_by")
	sortOrder := c.Query("sort_order")
	limit := 100 // Default limit for admin
	offset := 0  // Default offset

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
			if limit > 500 { // Max limit for admin
				limit = 500
			}
		}
	}

	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build query
	query := h.db.DB.Model(&models.User{})

	// Include deleted records if requested
	if includeDeleted {
		query = query.Unscoped()
	}

	// Apply filters
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if isActiveStr != "" {
		if isActive, err := strconv.ParseBool(isActiveStr); err == nil {
			query = query.Where("is_active = ?", isActive)
		}
	}
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"username ILIKE ? OR email ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Apply sorting
	orderClause := "created_at DESC" // Default sorting
	if sortBy != "" {
		validSortFields := []string{"username", "email", "role", "created_at", "updated_at", "last_login_at"}
		for _, field := range validSortFields {
			if sortBy == field {
				if sortOrder == "asc" {
					orderClause = sortBy + " ASC"
				} else {
					orderClause = sortBy + " DESC"
				}
				break
			}
		}
	}

	// Get users with all fields for admin
	var users []models.User
	err := query.Select("id, username, email, first_name, last_name, role, is_active, is_verified, created_at, updated_at, deleted_at, last_login_at, login_attempts, locked_until").
		Limit(limit).Offset(offset).Order(orderClause).Find(&users).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to fetch all users")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch users",
			"message": err.Error(),
		})
		return
	}

	// Get statistics
	var stats struct {
		TotalUsers      int64 `json:"total_users"`
		ActiveUsers     int64 `json:"active_users"`
		InactiveUsers   int64 `json:"inactive_users"`
		VerifiedUsers   int64 `json:"verified_users"`
		UnverifiedUsers int64 `json:"unverified_users"`
		AdminUsers      int64 `json:"admin_users"`
		RegularUsers    int64 `json:"regular_users"`
		DeletedUsers    int64 `json:"deleted_users"`
	}

	statsQuery := h.db.DB.Model(&models.User{})
	if includeDeleted {
		statsQuery = statsQuery.Unscoped()
	}

	statsQuery.Count(&stats.TotalUsers)
	h.db.DB.Model(&models.User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers)
	h.db.DB.Model(&models.User{}).Where("is_active = ?", false).Count(&stats.InactiveUsers)
	h.db.DB.Model(&models.User{}).Where("is_verified = ?", true).Count(&stats.VerifiedUsers)
	h.db.DB.Model(&models.User{}).Where("is_verified = ?", false).Count(&stats.UnverifiedUsers)
	h.db.DB.Model(&models.User{}).Where("role = ?", "admin").Count(&stats.AdminUsers)
	h.db.DB.Model(&models.User{}).Where("role = ?", "user").Count(&stats.RegularUsers)
	h.db.DB.Unscoped().Model(&models.User{}).Where("deleted_at IS NOT NULL").Count(&stats.DeletedUsers)

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"pagination": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
		"statistics": stats,
		"filters": gin.H{
			"role":            role,
			"is_active":       isActiveStr,
			"search":          search,
			"include_deleted": includeDeleted,
			"sort_by":         sortBy,
			"sort_order":      sortOrder,
		},
	})
}

// ActivateUser activates a user account (admin only)
func (h *Handlers) ActivateUser(c *gin.Context) {
	// Check if user has admin role
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Admin access required",
		})
		return
	}

	userID := c.Param("id")

	// Parse UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	// Find user
	var user models.User
	err = h.db.DB.Where("id = ?", userUUID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find user",
			"message": err.Error(),
		})
		return
	}

	// Check if user is already active
	if user.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User is already active",
		})
		return
	}

	// Activate user
	updateMap := map[string]interface{}{
		"is_active":      true,
		"login_attempts": 0,
		"locked_until":   nil,
	}

	err = h.db.DB.Model(&user).Updates(updateMap).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to activate user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to activate user",
			"message": err.Error(),
		})
		return
	}

	// Reload user
	h.db.DB.Select("id, username, email, first_name, last_name, role, is_active, is_verified, created_at, updated_at, last_login_at").
		First(&user, user.ID)

	h.logger.WithFields(logrus.Fields{
		"user_id":      user.ID,
		"username":     user.Username,
		"activated_by": c.GetString("user_id"),
	}).Info("User activated successfully")

	// Log activity
	h.logActivity(c, "activate_user", "user", &user.ID, gin.H{
		"username": user.Username,
	})

	// Broadcast user update
	h.wsHub.BroadcastUserUpdate(gin.H{
		"action": "activated",
		"user":   sanitizeUser(user),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "User activated successfully",
		"user":    sanitizeUser(user),
	})
}

// DeactivateUser deactivates a user account (admin only)
func (h *Handlers) DeactivateUser(c *gin.Context) {
	// Check if user has admin role
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Admin access required",
		})
		return
	}

	userID := c.Param("id")

	// Parse UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	// Prevent admin from deactivating themselves
	currentUserID, _ := c.Get("user_id")
	if currentUserID.(string) == userID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot deactivate your own account",
		})
		return
	}

	// Find user
	var user models.User
	err = h.db.DB.Where("id = ?", userUUID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}
		h.logger.WithError(err).Error("Failed to find user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to find user",
			"message": err.Error(),
		})
		return
	}

	// Check if user is already inactive
	if !user.IsActive {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User is already inactive",
		})
		return
	}

	// Deactivate user
	err = h.db.DB.Model(&user).Update("is_active", false).Error
	if err != nil {
		h.logger.WithError(err).Error("Failed to deactivate user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to deactivate user",
			"message": err.Error(),
		})
		return
	}

	// Reload user
	h.db.DB.Select("id, username, email, first_name, last_name, role, is_active, is_verified, created_at, updated_at, last_login_at").
		First(&user, user.ID)

	h.logger.WithFields(logrus.Fields{
		"user_id":        user.ID,
		"username":       user.Username,
		"deactivated_by": currentUserID,
	}).Info("User deactivated successfully")

	// Log activity
	h.logActivity(c, "deactivate_user", "user", &user.ID, gin.H{
		"username": user.Username,
	})

	// Broadcast user update
	h.wsHub.BroadcastUserUpdate(gin.H{
		"action": "deactivated",
		"user":   sanitizeUser(user),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "User deactivated successfully",
		"user":    sanitizeUser(user),
	})
}
func (h *Handlers) GetSystemLogs(c *gin.Context)    { h.notImplemented(c) }
func (h *Handlers) StartMaintenance(c *gin.Context) { h.notImplemented(c) }
func (h *Handlers) StopMaintenance(c *gin.Context)  { h.notImplemented(c) }

// Authentication Handlers

// Login authenticates a user and returns JWT tokens
func (h *Handlers) Login(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	tokens, user, err := h.authService.Login(&req)
	if err != nil {
		h.logger.WithError(err).Warn("Login attempt failed")

		// Log failed login activity
		h.activityService.LogAuthActivity(
			c.Request.Context(),
			nil,
			req.Username,
			activity.ActivityTypeLogin,
			false,
			c.ClientIP(),
			c.Request.UserAgent(),
			err.Error(),
		)

		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Login failed",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User logged in successfully")

	// Log successful login activity
	h.activityService.LogAuthActivity(
		c.Request.Context(),
		&user.ID,
		user.Username,
		activity.ActivityTypeLogin,
		true,
		c.ClientIP(),
		c.Request.UserAgent(),
		"",
	)

	// Set cookie for browser clients
	c.SetCookie("auth_token", tokens.AccessToken, int(tokens.ExpiresIn), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"tokens":  tokens,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// Register creates a new user account
func (h *Handlers) Register(c *gin.Context) {
	var req auth.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	user, err := h.authService.Register(&req)
	if err != nil {
		h.logger.WithError(err).Warn("User registration failed")

		// Log failed registration activity
		h.activityService.LogAuthActivity(
			c.Request.Context(),
			nil,
			req.Username,
			activity.ActivityTypeRegister,
			false,
			c.ClientIP(),
			c.Request.UserAgent(),
			err.Error(),
		)

		statusCode := http.StatusInternalServerError
		if err == auth.ErrUserExists {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, gin.H{
			"error":   "Registration failed",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User registered successfully")

	// Log successful registration activity
	h.activityService.LogAuthActivity(
		c.Request.Context(),
		&user.ID,
		user.Username,
		activity.ActivityTypeRegister,
		true,
		c.ClientIP(),
		c.Request.UserAgent(),
		"",
	)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registration successful",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// Logout invalidates the user's session (for now just clears cookie)
func (h *Handlers) Logout(c *gin.Context) {
	// Clear auth cookie
	c.SetCookie("auth_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// RefreshToken generates new tokens using a refresh token
func (h *Handlers) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	tokens, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		h.logger.WithError(err).Warn("Token refresh failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Token refresh failed",
			"message": err.Error(),
		})
		return
	}

	// Update cookie
	c.SetCookie("auth_token", tokens.AccessToken, int(tokens.ExpiresIn), "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"message": "Token refreshed successfully",
		"tokens":  tokens,
	})
}

// GetProfile returns the current user's profile
func (h *Handlers) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	user, err := h.authService.GetUserByID(userID.(string))
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user profile")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get profile",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
			"last_login": user.LastLoginAt,
		},
	})
}

// UpdatePassword updates the current user's password
func (h *Handlers) UpdatePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	err := h.authService.UpdatePassword(userID.(string), req.OldPassword, req.NewPassword)
	if err != nil {
		h.logger.WithError(err).Warn("Password update failed")

		statusCode := http.StatusInternalServerError
		if err == auth.ErrInvalidCredentials {
			statusCode = http.StatusUnauthorized
		}

		c.JSON(statusCode, gin.H{
			"error":   "Password update failed",
			"message": err.Error(),
		})
		return
	}

	h.logger.WithField("user_id", userID).Info("Password updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Password updated successfully",
	})
}

// notImplemented returns a "not implemented" response
func (h *Handlers) notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":   "Not implemented yet",
		"message": fmt.Sprintf("The endpoint %s %s is not yet implemented", c.Request.Method, c.Request.URL.Path),
	})
}

// sanitizeUser removes sensitive fields from user data
func sanitizeUser(user models.User) gin.H {
	return gin.H{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"first_name":    user.FirstName,
		"last_name":     user.LastName,
		"role":          user.Role,
		"is_active":     user.IsActive,
		"is_verified":   user.IsVerified,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
		"last_login_at": user.LastLoginAt,
	}
}

// logActivity logs user activity
func (h *Handlers) logActivity(c *gin.Context, action, resource string, resourceID *uuid.UUID, details gin.H) {
	userID, exists := c.Get("user_id")
	if !exists {
		return
	}

	userUUID, err := uuid.Parse(userID.(string))
	if err != nil {
		h.logger.WithError(err).Warn("Failed to parse user ID for activity logging")
		return
	}

	activity := models.ActivityLog{
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		UserID:     &userUUID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		Success:    true,
	}

	if details != nil {
		if detailsJSON, err := json.Marshal(details); err == nil {
			activity.Details = string(detailsJSON)
		}
	}

	if err := h.db.DB.Create(&activity).Error; err != nil {
		h.logger.WithError(err).Error("Failed to log activity")
	}
}
