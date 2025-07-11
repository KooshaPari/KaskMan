package chat

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Simple project interface for mock data
type SimpleProject struct {
	ID                  uuid.UUID     `json:"id"`
	Name                string        `json:"name"`
	Path                string        `json:"path"`
	Tasks               []*SimpleTask `json:"tasks"`
	Resources           []interface{} `json:"resources"`
	Status              string        `json:"status"`
	Progress            float64       `json:"progress"`
	Health              string        `json:"health"`
	SuccessProbability  float64       `json:"success_probability"`
	PredictedCompletion time.Time     `json:"predicted_completion"`
	ConfidenceLevel     float64       `json:"confidence_level"`
	QualityPrediction   float64       `json:"quality_prediction"`
	StartTime           time.Time     `json:"start_time"`
}

type SimpleTask struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Status string    `json:"status"`
}

// ChatServer provides HTTP and WebSocket endpoints for the project chat interface
type ChatServer struct {
	logger           *logrus.Logger
	chatInterface    *ProjectChatInterface
	tuiParser        *TUIParserImpl
	activeSessions   map[uuid.UUID]*ActiveChatSession
	websocketClients map[uuid.UUID]*websocket.Conn
	upgrader         websocket.Upgrader
	port             int
}

// ActiveChatSession tracks active WebSocket connections
type ActiveChatSession struct {
	SessionID    uuid.UUID              `json:"session_id"`
	UserID       uuid.UUID              `json:"user_id"`
	ProjectID    *uuid.UUID             `json:"project_id,omitempty"`
	Connection   *websocket.Conn        `json:"-"`
	LastActivity time.Time              `json:"last_activity"`
	Context      *ChatContext           `json:"context"`
}

// Chat API request/response types
type ChatMessageRequest struct {
	SessionID uuid.UUID `json:"session_id"`
	Message   string    `json:"message"`
	Type      string    `json:"type,omitempty"` // text, action, navigation
}

type ChatMessageResponse struct {
	MessageID        uuid.UUID        `json:"message_id"`
	SessionID        uuid.UUID        `json:"session_id"`
	Content          string           `json:"content"`
	Type             string           `json:"type"`
	Timestamp        time.Time        `json:"timestamp"`
	TUIComponents    []*TUIComponent  `json:"tui_components,omitempty"`
	QuickActions     []*ChatAction    `json:"quick_actions,omitempty"`
	RelatedQuestions []string         `json:"related_questions,omitempty"`
	ProcessingTime   time.Duration    `json:"processing_time"`
}

type SessionCreateRequest struct {
	UserID    uuid.UUID  `json:"user_id"`
	ProjectID *uuid.UUID `json:"project_id,omitempty"`
}

type SessionCreateResponse struct {
	SessionID    uuid.UUID      `json:"session_id"`
	Session      *ChatSession   `json:"session"`
	WelcomeMessage *ChatMessage `json:"welcome_message"`
}

type ProjectListResponse struct {
	Projects []ProjectSummary `json:"projects"`
}

type ProjectSummary struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Progress    float64   `json:"progress"`
	Health      string    `json:"health"`
	LastUpdated time.Time `json:"last_updated"`
}

type TUIParseRequest struct {
	TUIOutput string           `json:"tui_output"`
	Context   *TUIParseContext `json:"context,omitempty"`
}

type TUIParseResponse struct {
	Components []*TUIComponent `json:"components"`
	ParseTime  time.Duration   `json:"parse_time"`
}

// WebSocket message types
type WSMessage struct {
	Type      string      `json:"type"`
	SessionID uuid.UUID   `json:"session_id,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewChatServer creates a new chat server instance
func NewChatServer(logger *logrus.Logger, projectManager interface{}, port int) *ChatServer {
	chatInterface := NewProjectChatInterface(logger, nil)
	tuiParser := NewTUIParserImpl(logger)

	return &ChatServer{
		logger:           logger,
		chatInterface:    chatInterface,
		tuiParser:        tuiParser,
		activeSessions:   make(map[uuid.UUID]*ActiveChatSession),
		websocketClients: make(map[uuid.UUID]*websocket.Conn),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		port: port,
	}
}

// StartServer starts the HTTP server with chat endpoints
func (cs *ChatServer) StartServer(ctx context.Context) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Configure CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "X-Requested-With"}
	router.Use(cors.New(config))

	// Serve static files
	router.Static("/static", "./web/chat")
	router.StaticFile("/", "./web/chat/index.html")

	// API routes
	api := router.Group("/api/v1")
	{
		// Session management
		api.POST("/sessions", cs.createSession)
		api.GET("/sessions/:session_id", cs.getSession)
		api.DELETE("/sessions/:session_id", cs.deleteSession)

		// Chat endpoints
		api.POST("/chat/message", cs.sendMessage)
		api.GET("/chat/history/:session_id", cs.getChatHistory)

		// Project endpoints
		api.GET("/projects", cs.listProjects)
		api.GET("/projects/:project_id", cs.getProject)
		api.GET("/projects/:project_id/overview", cs.getProjectOverview)
		api.GET("/projects/:project_id/timeline", cs.getProjectTimeline)
		api.GET("/projects/:project_id/tasks", cs.getProjectTasks)
		api.GET("/projects/:project_id/risks", cs.getProjectRisks)

		// TUI parsing endpoints
		api.POST("/tui/parse", cs.parseTUIOutput)
		api.POST("/tui/interact", cs.handleTUIInteraction)

		// Utility endpoints
		api.GET("/health", cs.healthCheck)
		api.GET("/metrics", cs.getMetrics)
	}

	// WebSocket endpoint
	router.GET("/ws", cs.handleWebSocket)

	cs.logger.WithField("port", cs.port).Info("Starting chat server")

	// Start cleanup routine
	go cs.sessionCleanupRoutine(ctx)

	// Start server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cs.port),
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		cs.logger.Info("Shutting down chat server")
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// HTTP Handlers

func (cs *ChatServer) createSession(c *gin.Context) {
	var req SessionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Create chat session
	session, err := cs.chatInterface.StartChatSession(c.Request.Context(), req.UserID, req.ProjectID)
	if err != nil {
		cs.logger.WithError(err).Error("Failed to create chat session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Store active session
	activeSession := &ActiveChatSession{
		SessionID:    session.ID,
		UserID:       req.UserID,
		ProjectID:    req.ProjectID,
		LastActivity: time.Now(),
		Context:      session.Context,
	}
	cs.activeSessions[session.ID] = activeSession

	response := SessionCreateResponse{
		SessionID:      session.ID,
		Session:        session,
		WelcomeMessage: session.Messages[0], // First message is welcome
	}

	cs.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"user_id":    req.UserID,
		"project_id": req.ProjectID,
	}).Info("Chat session created")

	c.JSON(http.StatusOK, response)
}

func (cs *ChatServer) getSession(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	activeSession, exists := cs.activeSessions[sessionID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, activeSession)
}

func (cs *ChatServer) deleteSession(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	// Close WebSocket connection if exists
	if conn, exists := cs.websocketClients[sessionID]; exists {
		conn.Close()
		delete(cs.websocketClients, sessionID)
	}

	// Remove active session
	delete(cs.activeSessions, sessionID)

	cs.logger.WithField("session_id", sessionID).Info("Chat session deleted")
	c.JSON(http.StatusOK, gin.H{"message": "Session deleted"})
}

func (cs *ChatServer) sendMessage(c *gin.Context) {
	var req ChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	startTime := time.Now()

	// Process message
	chatResponse, err := cs.chatInterface.ProcessMessage(c.Request.Context(), req.SessionID, req.Message)
	if err != nil {
		cs.logger.WithError(err).Error("Failed to process chat message")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process message"})
		return
	}

	processingTime := time.Since(startTime)

	// Update session activity
	if activeSession, exists := cs.activeSessions[req.SessionID]; exists {
		activeSession.LastActivity = time.Now()
	}

	response := ChatMessageResponse{
		MessageID:        chatResponse.Message.ID,
		SessionID:        req.SessionID,
		Content:          chatResponse.Message.Content,
		Type:             chatResponse.Message.Type,
		Timestamp:        chatResponse.Message.Timestamp,
		TUIComponents:    chatResponse.Message.TUIComponents,
		QuickActions:     chatResponse.Message.Actions,
		RelatedQuestions: chatResponse.RelatedQuestions,
		ProcessingTime:   processingTime,
	}

	// Send to WebSocket clients if connected
	cs.broadcastToSession(req.SessionID, WSMessage{
		Type:      "message_response",
		SessionID: req.SessionID,
		Data:      response,
		Timestamp: time.Now(),
	})

	c.JSON(http.StatusOK, response)
}

func (cs *ChatServer) getChatHistory(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	// Get session from conversation manager
	session, err := cs.chatInterface.conversationManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"messages":   session.Messages,
		"total":      len(session.Messages),
	})
}

func (cs *ChatServer) listProjects(c *gin.Context) {
	// Mock project data - in real implementation, fetch from project manager
	projects := []ProjectSummary{
		{
			ID:          uuid.New(),
			Name:        "FinTech Mobile App",
			Status:      "executing",
			Progress:    0.67,
			Health:      "good",
			LastUpdated: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:          uuid.New(),
			Name:        "E-commerce Platform",
			Status:      "planning",
			Progress:    0.23,
			Health:      "excellent",
			LastUpdated: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:          uuid.New(),
			Name:        "AI Assistant",
			Status:      "monitoring",
			Progress:    0.89,
			Health:      "at_risk",
			LastUpdated: time.Now().Add(-30 * time.Minute),
		},
	}

	c.JSON(http.StatusOK, ProjectListResponse{Projects: projects})
}

func (cs *ChatServer) getProject(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Mock project - in real implementation, fetch from project manager
	project := &SimpleProject{
		ID:                  projectID,
		Name:                "FinTech Mobile App",
		Status:              "executing",
		Progress:            0.67,
		Health:              "good",
		SuccessProbability:  0.89,
		ConfidenceLevel:     0.92,
		QualityPrediction:   0.88,
		PredictedCompletion: time.Now().Add(30 * 24 * time.Hour),
		StartTime:           time.Now().Add(-60 * 24 * time.Hour),
		Path:                "/path/to/fintech-app",
		Resources:           []interface{}{},
		Tasks: []*SimpleTask{
			{ID: uuid.New(), Name: "User Authentication", Status: "completed"},
			{ID: uuid.New(), Name: "Payment Integration", Status: "in_progress"},
			{ID: uuid.New(), Name: "Dashboard Design", Status: "pending"},
		},
	}

	c.JSON(http.StatusOK, project)
}

func (cs *ChatServer) getProjectOverview(c *gin.Context) {
	projectIDStr := c.Param("project_id")
	_, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	// Generate project overview data
	overview := map[string]interface{}{
		"metrics": map[string]interface{}{
			"progress":            67.5,
			"success_probability": 89.2,
			"confidence_level":    92.1,
			"quality_prediction":  88.7,
		},
		"tasks": map[string]interface{}{
			"total":       15,
			"completed":   10,
			"in_progress": 3,
			"pending":     2,
		},
		"risks": map[string]interface{}{
			"total":    8,
			"high":     1,
			"medium":   3,
			"low":      4,
		},
		"team": map[string]interface{}{
			"members":           5,
			"avg_utilization":   82.3,
			"performance_trend": "improving",
		},
	}

	c.JSON(http.StatusOK, overview)
}

func (cs *ChatServer) getProjectTimeline(c *gin.Context) {
	timeline := map[string]interface{}{
		"milestones": []map[string]interface{}{
			{
				"id":          uuid.New(),
				"name":        "Authentication System",
				"date":        time.Now().Add(-15 * 24 * time.Hour),
				"status":      "completed",
				"progress":    100,
			},
			{
				"id":          uuid.New(),
				"name":        "Payment Integration",
				"date":        time.Now().Add(5 * 24 * time.Hour),
				"status":      "in_progress",
				"progress":    75,
			},
			{
				"id":          uuid.New(),
				"name":        "Beta Testing",
				"date":        time.Now().Add(15 * 24 * time.Hour),
				"status":      "pending",
				"progress":    0,
			},
		},
		"critical_path": []string{"payment_integration", "testing", "deployment"},
		"estimated_completion": time.Now().Add(30 * 24 * time.Hour),
	}

	c.JSON(http.StatusOK, timeline)
}

func (cs *ChatServer) getProjectTasks(c *gin.Context) {
	tasks := []map[string]interface{}{
		{
			"id":          uuid.New(),
			"name":        "User Authentication System",
			"status":      "completed",
			"priority":    "high",
			"assignee":    "Sarah Chen",
			"due_date":    time.Now().Add(-5 * 24 * time.Hour),
			"progress":    100,
		},
		{
			"id":          uuid.New(),
			"name":        "Payment Gateway Integration",
			"status":      "in_progress",
			"priority":    "critical",
			"assignee":    "Mike Rodriguez",
			"due_date":    time.Now().Add(5 * 24 * time.Hour),
			"progress":    75,
		},
		{
			"id":          uuid.New(),
			"name":        "Mobile UI Components",
			"status":      "in_progress",
			"priority":    "medium",
			"assignee":    "Anna Kim",
			"due_date":    time.Now().Add(10 * 24 * time.Hour),
			"progress":    60,
		},
	}

	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (cs *ChatServer) getProjectRisks(c *gin.Context) {
	risks := []map[string]interface{}{
		{
			"id":          uuid.New(),
			"type":        "technical",
			"description": "API rate limiting may affect payment processing",
			"level":       "medium",
			"probability": 0.4,
			"impact":      "medium",
			"mitigation":  "Implement request queuing and retry logic",
		},
		{
			"id":          uuid.New(),
			"type":        "security",
			"description": "Payment data compliance requirements",
			"level":       "high",
			"probability": 0.7,
			"impact":      "high",
			"mitigation":  "Schedule compliance audit and certification",
		},
	}

	c.JSON(http.StatusOK, gin.H{"risks": risks})
}

func (cs *ChatServer) parseTUIOutput(c *gin.Context) {
	var req TUIParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	startTime := time.Now()

	// Create parse context if not provided
	if req.Context == nil {
		req.Context = &TUIParseContext{
			TerminalSize: TerminalSize{Width: 80, Height: 24},
			ColorSupport: true,
			ViewportSize: ViewportSize{Width: 800, Height: 600},
		}
	}

	// Parse TUI output
	components, err := cs.tuiParser.ParseTUIOutput(c.Request.Context(), req.TUIOutput, req.Context)
	if err != nil {
		cs.logger.WithError(err).Error("TUI parsing failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "TUI parsing failed"})
		return
	}

	parseTime := time.Since(startTime)

	response := TUIParseResponse{
		Components: components,
		ParseTime:  parseTime,
	}

	c.JSON(http.StatusOK, response)
}

func (cs *ChatServer) handleTUIInteraction(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Handle TUI interaction
	result := map[string]interface{}{
		"status":  "processed",
		"actions": []string{"update_view"},
		"changes": req,
	}

	c.JSON(http.StatusOK, result)
}

func (cs *ChatServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":           "healthy",
		"timestamp":        time.Now(),
		"active_sessions":  len(cs.activeSessions),
		"websocket_clients": len(cs.websocketClients),
		"version":          "1.0.0",
	})
}

func (cs *ChatServer) getMetrics(c *gin.Context) {
	metrics := map[string]interface{}{
		"active_sessions":    len(cs.activeSessions),
		"websocket_clients":  len(cs.websocketClients),
		"uptime":            time.Since(time.Now().Add(-1 * time.Hour)), // Mock uptime
		"messages_processed": 1234, // Mock metric
		"avg_response_time":  "150ms",
	}

	c.JSON(http.StatusOK, metrics)
}

// WebSocket Handler

func (cs *ChatServer) handleWebSocket(c *gin.Context) {
	conn, err := cs.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		cs.logger.WithError(err).Error("WebSocket upgrade failed")
		return
	}
	defer conn.Close()

	sessionIDStr := c.Query("session_id")
	if sessionIDStr == "" {
		cs.logger.Error("WebSocket connection missing session_id")
		return
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		cs.logger.WithError(err).Error("Invalid session ID in WebSocket connection")
		return
	}

	// Store WebSocket connection
	cs.websocketClients[sessionID] = conn

	cs.logger.WithField("session_id", sessionID).Info("WebSocket client connected")

	// Handle messages
	for {
		var message WSMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			cs.logger.WithError(err).Debug("WebSocket read error")
			break
		}

		// Process WebSocket message
		cs.handleWebSocketMessage(sessionID, &message)
	}

	// Cleanup
	delete(cs.websocketClients, sessionID)
	cs.logger.WithField("session_id", sessionID).Info("WebSocket client disconnected")
}

func (cs *ChatServer) handleWebSocketMessage(sessionID uuid.UUID, message *WSMessage) {
	switch message.Type {
	case "ping":
		cs.sendToSession(sessionID, WSMessage{
			Type:      "pong",
			SessionID: sessionID,
			Timestamp: time.Now(),
		})
	case "typing":
		// Broadcast typing indicator to other clients in same session
		cs.broadcastToSession(sessionID, *message)
	case "view_change":
		// Handle view changes
		cs.logger.WithFields(logrus.Fields{
			"session_id": sessionID,
			"view":       message.Data,
		}).Debug("View change received")
	}
}

func (cs *ChatServer) broadcastToSession(sessionID uuid.UUID, message WSMessage) {
	if conn, exists := cs.websocketClients[sessionID]; exists {
		cs.sendToConnection(conn, message)
	}
}

func (cs *ChatServer) sendToSession(sessionID uuid.UUID, message WSMessage) {
	if conn, exists := cs.websocketClients[sessionID]; exists {
		cs.sendToConnection(conn, message)
	}
}

func (cs *ChatServer) sendToConnection(conn *websocket.Conn, message WSMessage) {
	if err := conn.WriteJSON(message); err != nil {
		cs.logger.WithError(err).Error("Failed to send WebSocket message")
	}
}

// Background routines

func (cs *ChatServer) sessionCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cs.cleanupInactiveSessions()
		}
	}
}

func (cs *ChatServer) cleanupInactiveSessions() {
	cutoff := time.Now().Add(-30 * time.Minute) // 30 minutes timeout
	var toDelete []uuid.UUID

	for sessionID, session := range cs.activeSessions {
		if session.LastActivity.Before(cutoff) {
			toDelete = append(toDelete, sessionID)
		}
	}

	for _, sessionID := range toDelete {
		// Close WebSocket if exists
		if conn, exists := cs.websocketClients[sessionID]; exists {
			conn.Close()
			delete(cs.websocketClients, sessionID)
		}
		
		delete(cs.activeSessions, sessionID)
		cs.logger.WithField("session_id", sessionID).Info("Cleaned up inactive session")
	}

	if len(toDelete) > 0 {
		cs.logger.WithField("cleaned_sessions", len(toDelete)).Info("Session cleanup completed")
	}
}