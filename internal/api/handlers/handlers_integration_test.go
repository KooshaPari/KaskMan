package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/activity"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/auth"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/config"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/monitoring"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd"
	customTesting "github.com/kooshapari/kaskmanager-rd-platform/internal/testutils"
	ws "github.com/kooshapari/kaskmanager-rd-platform/internal/websocket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// HandlersIntegrationTestSuite represents the integration test suite for API handlers
type HandlersIntegrationTestSuite struct {
	customTesting.TestSuite
	handlers        *Handlers
	router          *gin.Engine
	authService     *auth.Service
	activityService *activity.Service
	rndModule       *rnd.Module
	monitor         *monitoring.Monitor
	wsHub           *ws.Hub
	fixtures        *customTesting.TestFixtures
	helpers         *customTesting.TestHelpers
	testUser        *models.User
	testProject     *models.Project
	testAgent       *models.Agent
	testTask        *models.Task
	authToken       string
}

// SetupTest sets up the test suite
func (s *HandlersIntegrationTestSuite) SetupTest() {
	s.TestSuite.SetupTest()
	gin.SetMode(gin.TestMode)

	// Create fixtures and helpers
	s.fixtures = customTesting.NewTestFixtures(s.DB)
	s.helpers = customTesting.NewTestHelpers(s.T())

	// Create database wrapper
	db := &database.Database{DB: s.DB}

	// Create services
	s.authService = auth.NewService(db, s.Config.Logger)
	s.activityService = activity.NewService(db, s.Config.Logger)

	// Create R&D module
	rndConfig := config.RnDConfig{
		Enabled:          true,
		LearningInterval: 1 * time.Hour, // Long interval for testing
		PatternThreshold: 0.75,
		InsightThreshold: 0.80,
		MaxAgents:        10,
	}
	var err error
	s.rndModule, err = rnd.NewModule(rndConfig, db, s.Config.Logger)
	s.helpers.AssertNoError(err)

	// Create monitor
	s.monitor = monitoring.NewMonitor(s.Config.Logger)

	// Create WebSocket hub
	s.wsHub = ws.NewHub()
	go s.wsHub.Run()

	// Create handlers
	s.handlers = NewHandlers(
		db,
		s.wsHub,
		s.rndModule,
		s.monitor,
		s.Config.Logger,
		s.authService,
		s.activityService,
	)

	// Setup router
	s.setupRouter()

	// Create test data
	s.createTestData()
}

// TearDownTest cleans up after each test
func (s *HandlersIntegrationTestSuite) TearDownTest() {
	if s.rndModule != nil && s.rndModule.IsRunning() {
		s.rndModule.Stop()
	}
	if s.wsHub != nil {
		s.wsHub.Stop()
	}
	s.TestSuite.TearDownTest()
}

// setupRouter sets up the Gin router with all routes
func (s *HandlersIntegrationTestSuite) setupRouter() {
	s.router = gin.New()
	s.router.Use(gin.Recovery())

	// Add test middleware for authentication
	s.router.Use(s.testAuthMiddleware())

	// Setup routes (simplified version)
	api := s.router.Group("/api/v1")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/login", s.handlers.Login)
			auth.POST("/register", s.handlers.Register)
			auth.POST("/logout", s.handlers.Logout)
			auth.GET("/me", s.handlers.GetCurrentUser)
		}

		// User routes
		users := api.Group("/users")
		{
			users.GET("", s.handlers.GetUsers)
			users.GET("/:id", s.handlers.GetUser)
			users.PUT("/:id", s.handlers.UpdateUser)
			users.DELETE("/:id", s.handlers.DeleteUser)
		}

		// Project routes
		projects := api.Group("/projects")
		{
			projects.GET("", s.handlers.GetProjects)
			projects.POST("", s.handlers.CreateProject)
			projects.GET("/:id", s.handlers.GetProject)
			projects.PUT("/:id", s.handlers.UpdateProject)
			projects.DELETE("/:id", s.handlers.DeleteProject)
			projects.GET("/:id/tasks", s.handlers.GetProjectTasks)
			projects.GET("/:id/agents", s.handlers.GetProjectAgents)
		}

		// Task routes
		tasks := api.Group("/tasks")
		{
			tasks.GET("", s.handlers.GetTasks)
			tasks.POST("", s.handlers.CreateTask)
			tasks.GET("/:id", s.handlers.GetTask)
			tasks.PUT("/:id", s.handlers.UpdateTask)
			tasks.DELETE("/:id", s.handlers.DeleteTask)
			tasks.POST("/:id/assign", s.handlers.AssignTask)
		}

		// Agent routes
		agents := api.Group("/agents")
		{
			agents.GET("", s.handlers.GetAgents)
			agents.POST("", s.handlers.CreateAgent)
			agents.GET("/:id", s.handlers.GetAgent)
			agents.PUT("/:id", s.handlers.UpdateAgent)
			agents.DELETE("/:id", s.handlers.DeleteAgent)
		}

		// R&D routes
		rnd := api.Group("/rnd")
		{
			rnd.GET("/status", s.handlers.GetRnDStatus)
			rnd.POST("/analyze", s.handlers.TriggerAnalysis)
			rnd.GET("/insights", s.handlers.GetInsights)
			rnd.GET("/patterns", s.handlers.GetPatterns)
			rnd.POST("/generate-project", s.handlers.GenerateProject)
		}

		// System routes
		system := api.Group("/system")
		{
			system.GET("/health", s.handlers.HealthCheck)
			system.GET("/metrics", s.handlers.GetMetrics)
			system.GET("/logs", s.handlers.GetLogs)
		}
	}

	// WebSocket endpoint
	s.router.GET("/ws", s.handlers.HandleWebSocket)
}

// testAuthMiddleware is a simple auth middleware for testing
func (s *HandlersIntegrationTestSuite) testAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == s.authToken && s.testUser != nil {
				c.Set("user_id", s.testUser.ID.String())
				c.Set("user_role", s.testUser.Role)
				c.Set("user", s.testUser)
			}
		}
		c.Next()
	}
}

// createTestData creates test data for integration tests
func (s *HandlersIntegrationTestSuite) createTestData() {
	// Create test user
	s.testUser = s.fixtures.CreateUser(map[string]interface{}{
		"username": "testuser",
		"email":    "test@example.com",
		"role":     "admin",
	})

	// Create auth token
	s.authToken = "test-token-" + s.testUser.ID.String()

	// Create test project
	s.testProject = s.fixtures.CreateProject(s.testUser.ID, map[string]interface{}{
		"name":        "Test Project",
		"description": "Integration test project",
		"status":      "active",
	})

	// Create test agent
	s.testAgent = s.fixtures.CreateAgent(s.testProject.ID, map[string]interface{}{
		"name":   "Test Agent",
		"type":   "researcher",
		"status": "active",
	})

	// Create test task
	s.testTask = s.fixtures.CreateTask(s.testProject.ID, map[string]interface{}{
		"title":       "Test Task",
		"description": "Integration test task",
		"status":      "pending",
		"agent_id":    s.testAgent.ID,
	})
}

// makeRequest is a helper to make HTTP requests
func (s *HandlersIntegrationTestSuite) makeRequest(method, path string, body interface{}, authenticated bool) *httptest.ResponseRecorder {
	var bodyReader *strings.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(bodyBytes))
	} else {
		bodyReader = strings.NewReader("")
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	if authenticated {
		req.Header.Set("Authorization", "Bearer "+s.authToken)
	}

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)

	return recorder
}

// Test Authentication Endpoints

func (s *HandlersIntegrationTestSuite) TestAuth_Login() {
	// Test valid login
	loginData := map[string]interface{}{
		"username": s.testUser.Username,
		"password": "password",
	}

	recorder := s.makeRequest("POST", "/api/v1/auth/login", loginData, false)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "token")
	assert.Contains(s.T(), response, "user")

	// Test invalid credentials
	invalidLogin := map[string]interface{}{
		"username": "invalid",
		"password": "wrong",
	}

	recorder = s.makeRequest("POST", "/api/v1/auth/login", invalidLogin, false)
	s.helpers.AssertHTTPError(recorder, http.StatusUnauthorized)
}

func (s *HandlersIntegrationTestSuite) TestAuth_Register() {
	registerData := map[string]interface{}{
		"username":  "newuser",
		"email":     "newuser@example.com",
		"password":  "password123",
		"firstName": "New",
		"lastName":  "User",
	}

	recorder := s.makeRequest("POST", "/api/v1/auth/register", registerData, false)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusCreated)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "user")

	// Test duplicate registration
	recorder = s.makeRequest("POST", "/api/v1/auth/register", registerData, false)
	s.helpers.AssertHTTPError(recorder, http.StatusConflict)
}

func (s *HandlersIntegrationTestSuite) TestAuth_GetCurrentUser() {
	// Test authenticated request
	recorder := s.makeRequest("GET", "/api/v1/auth/me", nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Equal(s.T(), s.testUser.Username, response["username"])
	assert.Equal(s.T(), s.testUser.Email, response["email"])

	// Test unauthenticated request
	recorder = s.makeRequest("GET", "/api/v1/auth/me", nil, false)
	s.helpers.AssertHTTPError(recorder, http.StatusUnauthorized)
}

// Test User Endpoints

func (s *HandlersIntegrationTestSuite) TestUsers_GetUsers() {
	recorder := s.makeRequest("GET", "/api/v1/users", nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "data")
	assert.Contains(s.T(), response, "total")

	data := response["data"].([]interface{})
	assert.True(s.T(), len(data) >= 1)
}

func (s *HandlersIntegrationTestSuite) TestUsers_GetUser() {
	path := fmt.Sprintf("/api/v1/users/%s", s.testUser.ID.String())
	recorder := s.makeRequest("GET", path, nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var user models.User
	s.helpers.ParseJSONResponse(recorder, &user)
	assert.Equal(s.T(), s.testUser.ID, user.ID)
	assert.Equal(s.T(), s.testUser.Username, user.Username)

	// Test non-existent user
	path = fmt.Sprintf("/api/v1/users/%s", uuid.New().String())
	recorder = s.makeRequest("GET", path, nil, true)
	s.helpers.AssertHTTPError(recorder, http.StatusNotFound)
}

func (s *HandlersIntegrationTestSuite) TestUsers_UpdateUser() {
	updateData := map[string]interface{}{
		"firstName": "Updated",
		"lastName":  "Name",
	}

	path := fmt.Sprintf("/api/v1/users/%s", s.testUser.ID.String())
	recorder := s.makeRequest("PUT", path, updateData, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var updatedUser models.User
	s.helpers.ParseJSONResponse(recorder, &updatedUser)
	assert.Equal(s.T(), "Updated", updatedUser.FirstName)
	assert.Equal(s.T(), "Name", updatedUser.LastName)
}

// Test Project Endpoints

func (s *HandlersIntegrationTestSuite) TestProjects_GetProjects() {
	recorder := s.makeRequest("GET", "/api/v1/projects", nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "data")
	assert.Contains(s.T(), response, "total")

	data := response["data"].([]interface{})
	assert.True(s.T(), len(data) >= 1)
}

func (s *HandlersIntegrationTestSuite) TestProjects_CreateProject() {
	projectData := map[string]interface{}{
		"name":        "New Integration Project",
		"description": "Created via integration test",
		"type":        "research",
		"priority":    "medium",
		"status":      "active",
	}

	recorder := s.makeRequest("POST", "/api/v1/projects", projectData, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusCreated)

	var project models.Project
	s.helpers.ParseJSONResponse(recorder, &project)
	assert.Equal(s.T(), projectData["name"], project.Name)
	assert.Equal(s.T(), projectData["description"], project.Description)
	assert.Equal(s.T(), s.testUser.ID, project.UserID)
}

func (s *HandlersIntegrationTestSuite) TestProjects_GetProject() {
	path := fmt.Sprintf("/api/v1/projects/%s", s.testProject.ID.String())
	recorder := s.makeRequest("GET", path, nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var project models.Project
	s.helpers.ParseJSONResponse(recorder, &project)
	assert.Equal(s.T(), s.testProject.ID, project.ID)
	assert.Equal(s.T(), s.testProject.Name, project.Name)
}

func (s *HandlersIntegrationTestSuite) TestProjects_UpdateProject() {
	updateData := map[string]interface{}{
		"name":        "Updated Project Name",
		"description": "Updated description",
		"status":      "on_hold",
	}

	path := fmt.Sprintf("/api/v1/projects/%s", s.testProject.ID.String())
	recorder := s.makeRequest("PUT", path, updateData, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var updatedProject models.Project
	s.helpers.ParseJSONResponse(recorder, &updatedProject)
	assert.Equal(s.T(), updateData["name"], updatedProject.Name)
	assert.Equal(s.T(), updateData["description"], updatedProject.Description)
	assert.Equal(s.T(), updateData["status"], updatedProject.Status)
}

func (s *HandlersIntegrationTestSuite) TestProjects_GetProjectTasks() {
	path := fmt.Sprintf("/api/v1/projects/%s/tasks", s.testProject.ID.String())
	recorder := s.makeRequest("GET", path, nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "data")

	data := response["data"].([]interface{})
	assert.True(s.T(), len(data) >= 1)
}

// Test Task Endpoints

func (s *HandlersIntegrationTestSuite) TestTasks_CreateTask() {
	taskData := map[string]interface{}{
		"title":       "New Integration Task",
		"description": "Created via integration test",
		"projectId":   s.testProject.ID.String(),
		"priority":    "high",
		"status":      "pending",
		"type":        "development",
	}

	recorder := s.makeRequest("POST", "/api/v1/tasks", taskData, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusCreated)

	var task models.Task
	s.helpers.ParseJSONResponse(recorder, &task)
	assert.Equal(s.T(), taskData["title"], task.Title)
	assert.Equal(s.T(), taskData["description"], task.Description)
	assert.Equal(s.T(), s.testProject.ID, task.ProjectID)
}

func (s *HandlersIntegrationTestSuite) TestTasks_GetTask() {
	path := fmt.Sprintf("/api/v1/tasks/%s", s.testTask.ID.String())
	recorder := s.makeRequest("GET", path, nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var task models.Task
	s.helpers.ParseJSONResponse(recorder, &task)
	assert.Equal(s.T(), s.testTask.ID, task.ID)
	assert.Equal(s.T(), s.testTask.Title, task.Title)
}

func (s *HandlersIntegrationTestSuite) TestTasks_AssignTask() {
	assignData := map[string]interface{}{
		"agentId": s.testAgent.ID.String(),
	}

	path := fmt.Sprintf("/api/v1/tasks/%s/assign", s.testTask.ID.String())
	recorder := s.makeRequest("POST", path, assignData, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	// Verify assignment
	taskPath := fmt.Sprintf("/api/v1/tasks/%s", s.testTask.ID.String())
	recorder = s.makeRequest("GET", taskPath, nil, true)

	var assignedTask models.Task
	s.helpers.ParseJSONResponse(recorder, &assignedTask)
	assert.Equal(s.T(), s.testAgent.ID, *assignedTask.AgentID)
}

// Test Agent Endpoints

func (s *HandlersIntegrationTestSuite) TestAgents_CreateAgent() {
	agentData := map[string]interface{}{
		"name":        "New Integration Agent",
		"type":        "analyst",
		"description": "Created via integration test",
		"projectId":   s.testProject.ID.String(),
		"status":      "active",
	}

	recorder := s.makeRequest("POST", "/api/v1/agents", agentData, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusCreated)

	var agent models.Agent
	s.helpers.ParseJSONResponse(recorder, &agent)
	assert.Equal(s.T(), agentData["name"], agent.Name)
	assert.Equal(s.T(), agentData["type"], agent.Type)
	assert.Equal(s.T(), s.testProject.ID, agent.ProjectID)
}

func (s *HandlersIntegrationTestSuite) TestAgents_GetAgent() {
	path := fmt.Sprintf("/api/v1/agents/%s", s.testAgent.ID.String())
	recorder := s.makeRequest("GET", path, nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var agent models.Agent
	s.helpers.ParseJSONResponse(recorder, &agent)
	assert.Equal(s.T(), s.testAgent.ID, agent.ID)
	assert.Equal(s.T(), s.testAgent.Name, agent.Name)
}

// Test R&D Endpoints

func (s *HandlersIntegrationTestSuite) TestRnD_GetStatus() {
	recorder := s.makeRequest("GET", "/api/v1/rnd/status", nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "enabled")
	assert.Contains(s.T(), response, "running")
	assert.Contains(s.T(), response, "stats")
}

func (s *HandlersIntegrationTestSuite) TestRnD_TriggerAnalysis() {
	// Start R&D module first
	err := s.rndModule.Start()
	s.helpers.AssertNoError(err)

	analysisData := map[string]interface{}{
		"type": "pattern_analysis",
		"data": map[string]interface{}{
			"scope": "project",
			"id":    s.testProject.ID.String(),
		},
	}

	recorder := s.makeRequest("POST", "/api/v1/rnd/analyze", analysisData, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "success")
	assert.True(s.T(), response["success"].(bool))
}

func (s *HandlersIntegrationTestSuite) TestRnD_GenerateProject() {
	// Start R&D module first
	err := s.rndModule.Start()
	s.helpers.AssertNoError(err)

	generationData := map[string]interface{}{
		"type": "research",
		"preferences": map[string]interface{}{
			"domain":     "machine_learning",
			"complexity": "medium",
			"duration":   "3_months",
		},
	}

	recorder := s.makeRequest("POST", "/api/v1/rnd/generate-project", generationData, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "suggestions")
}

// Test System Endpoints

func (s *HandlersIntegrationTestSuite) TestSystem_HealthCheck() {
	recorder := s.makeRequest("GET", "/api/v1/system/health", nil, false)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "status")
	assert.Equal(s.T(), "healthy", response["status"])
	assert.Contains(s.T(), response, "components")
}

func (s *HandlersIntegrationTestSuite) TestSystem_GetMetrics() {
	recorder := s.makeRequest("GET", "/api/v1/system/metrics", nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Contains(s.T(), response, "system")
	assert.Contains(s.T(), response, "application")
}

// Test Error Handling

func (s *HandlersIntegrationTestSuite) TestErrorHandling_NotFound() {
	recorder := s.makeRequest("GET", "/api/v1/nonexistent", nil, true)
	assert.Equal(s.T(), http.StatusNotFound, recorder.Code)
}

func (s *HandlersIntegrationTestSuite) TestErrorHandling_MethodNotAllowed() {
	recorder := s.makeRequest("PATCH", "/api/v1/auth/login", nil, false)
	assert.Equal(s.T(), http.StatusMethodNotAllowed, recorder.Code)
}

func (s *HandlersIntegrationTestSuite) TestErrorHandling_ValidationErrors() {
	// Test creating project with invalid data
	invalidProjectData := map[string]interface{}{
		"name": "", // Empty name should fail validation
	}

	recorder := s.makeRequest("POST", "/api/v1/projects", invalidProjectData, true)
	s.helpers.AssertHTTPError(recorder, http.StatusBadRequest)
}

// Test Pagination

func (s *HandlersIntegrationTestSuite) TestPagination() {
	// Create additional test data
	for i := 0; i < 15; i++ {
		s.fixtures.CreateProject(s.testUser.ID, map[string]interface{}{
			"name": fmt.Sprintf("Pagination Test Project %d", i),
		})
	}

	// Test first page
	recorder := s.makeRequest("GET", "/api/v1/projects?page=1&page_size=10", nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	var response map[string]interface{}
	s.helpers.ParseJSONResponse(recorder, &response)
	assert.Equal(s.T(), float64(1), response["page"])
	assert.Equal(s.T(), float64(10), response["page_size"])
	assert.True(s.T(), response["total"].(float64) >= 15)

	data := response["data"].([]interface{})
	assert.Len(s.T(), data, 10)

	// Test second page
	recorder = s.makeRequest("GET", "/api/v1/projects?page=2&page_size=10", nil, true)
	s.helpers.AssertHTTPSuccess(recorder, http.StatusOK)

	s.helpers.ParseJSONResponse(recorder, &response)
	data = response["data"].([]interface{})
	assert.True(s.T(), len(data) >= 5) // At least 5 more items
}

// Test Concurrent Requests

func (s *HandlersIntegrationTestSuite) TestConcurrentRequests() {
	concurrency := 10
	iterations := 5

	ch := make(chan int, concurrency*iterations)

	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				recorder := s.makeRequest("GET", "/api/v1/system/health", nil, false)
				ch <- recorder.Code
			}
		}()
	}

	// Collect results
	successCount := 0
	for i := 0; i < concurrency*iterations; i++ {
		statusCode := <-ch
		if statusCode == http.StatusOK {
			successCount++
		}
	}

	// All requests should succeed
	assert.Equal(s.T(), concurrency*iterations, successCount)
}

// Test Rate Limiting (if implemented)

func (s *HandlersIntegrationTestSuite) TestRateLimiting() {
	// Make rapid requests to test rate limiting
	rapidRequests := 100
	statusCodes := make([]int, rapidRequests)

	for i := 0; i < rapidRequests; i++ {
		recorder := s.makeRequest("GET", "/api/v1/system/health", nil, false)
		statusCodes[i] = recorder.Code
	}

	// Count successful vs rate-limited requests
	successCount := 0
	rateLimitedCount := 0

	for _, code := range statusCodes {
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Some requests should succeed, some might be rate limited
	assert.True(s.T(), successCount > 0, "Some requests should succeed")
	// Rate limiting behavior depends on implementation
}

// Test WebSocket Connection (basic)

func (s *HandlersIntegrationTestSuite) TestWebSocketConnection() {
	// Create a test WebSocket server
	server := s.helpers.CreateTestWebSocketServer(func(conn *websocket.Conn) {
		// Echo messages back
		for {
			var message map[string]interface{}
			if err := conn.ReadJSON(&message); err != nil {
				break
			}
			conn.WriteJSON(map[string]interface{}{
				"type": "echo",
				"data": message,
			})
		}
	})
	defer server.Close()

	// Connect to WebSocket
	conn, err := s.helpers.ConnectToWebSocket(server)
	s.helpers.AssertNoError(err)
	defer conn.Close()

	// Send and receive message
	testMessage := map[string]interface{}{
		"type": "test",
		"data": "hello websocket",
	}

	err = s.helpers.SendWebSocketMessage(conn, testMessage)
	s.helpers.AssertNoError(err)

	var response map[string]interface{}
	err = s.helpers.ReceiveWebSocketMessageWithTimeout(conn, &response, 5*time.Second)
	s.helpers.AssertNoError(err)

	assert.Equal(s.T(), "echo", response["type"])
	assert.Equal(s.T(), testMessage, response["data"])
}

// Run the test suite
func TestHandlersIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersIntegrationTestSuite))
}
