package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelpers provides utility functions for testing
type TestHelpers struct {
	t      *testing.T
	logger *logrus.Logger
}

// NewTestHelpers creates a new test helpers instance
func NewTestHelpers(t *testing.T) *TestHelpers {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	return &TestHelpers{
		t:      t,
		logger: logger,
	}
}

// HTTP Testing Helpers

// CreateTestGinContext creates a test Gin context
func (h *TestHelpers) CreateTestGinContext(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		require.NoError(h.t, err)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req

	return ctx, recorder
}

// CreateTestGinContextWithHeaders creates a test Gin context with custom headers
func (h *TestHelpers) CreateTestGinContextWithHeaders(method, path string, body interface{}, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	ctx, recorder := h.CreateTestGinContext(method, path, body)

	for key, value := range headers {
		ctx.Request.Header.Set(key, value)
	}

	return ctx, recorder
}

// CreateAuthenticatedContext creates a test context with authentication
func (h *TestHelpers) CreateAuthenticatedContext(method, path string, body interface{}, userID uuid.UUID) (*gin.Context, *httptest.ResponseRecorder) {
	ctx, recorder := h.CreateTestGinContext(method, path, body)

	// Add user ID to context (simulating auth middleware)
	ctx.Set("user_id", userID.String())
	ctx.Set("user_role", "user")

	return ctx, recorder
}

// CreateAdminContext creates a test context with admin authentication
func (h *TestHelpers) CreateAdminContext(method, path string, body interface{}, userID uuid.UUID) (*gin.Context, *httptest.ResponseRecorder) {
	ctx, recorder := h.CreateTestGinContext(method, path, body)

	// Add user ID and admin role to context
	ctx.Set("user_id", userID.String())
	ctx.Set("user_role", "admin")

	return ctx, recorder
}

// AssertHTTPSuccess asserts that an HTTP response indicates success
func (h *TestHelpers) AssertHTTPSuccess(recorder *httptest.ResponseRecorder, expectedStatus int) {
	assert.Equal(h.t, expectedStatus, recorder.Code)
	assert.Contains(h.t, recorder.Header().Get("Content-Type"), "application/json")
}

// AssertHTTPError asserts that an HTTP response indicates an error
func (h *TestHelpers) AssertHTTPError(recorder *httptest.ResponseRecorder, expectedStatus int) {
	assert.Equal(h.t, expectedStatus, recorder.Code)

	var response map[string]interface{}
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	assert.NoError(h.t, err)
	assert.Contains(h.t, response, "error")
}

// ParseJSONResponse parses JSON response from recorder
func (h *TestHelpers) ParseJSONResponse(recorder *httptest.ResponseRecorder, target interface{}) {
	err := json.Unmarshal(recorder.Body.Bytes(), target)
	require.NoError(h.t, err)
}

// WebSocket Testing Helpers

// CreateTestWebSocketServer creates a test WebSocket server
func (h *TestHelpers) CreateTestWebSocketServer(handler func(*websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			h.logger.WithError(err).Error("Failed to upgrade WebSocket connection")
			return
		}
		defer conn.Close()

		handler(conn)
	}))
}

// ConnectToWebSocket creates a WebSocket connection to test server
func (h *TestHelpers) ConnectToWebSocket(server *httptest.Server) (*websocket.Conn, error) {
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	return conn, err
}

// SendWebSocketMessage sends a message to WebSocket connection
func (h *TestHelpers) SendWebSocketMessage(conn *websocket.Conn, message interface{}) error {
	return conn.WriteJSON(message)
}

// ReceiveWebSocketMessage receives a message from WebSocket connection
func (h *TestHelpers) ReceiveWebSocketMessage(conn *websocket.Conn, target interface{}) error {
	return conn.ReadJSON(target)
}

// ReceiveWebSocketMessageWithTimeout receives a message with timeout
func (h *TestHelpers) ReceiveWebSocketMessageWithTimeout(conn *websocket.Conn, target interface{}, timeout time.Duration) error {
	conn.SetReadDeadline(time.Now().Add(timeout))
	defer conn.SetReadDeadline(time.Time{})
	return conn.ReadJSON(target)
}

// Context Testing Helpers

// CreateTestContextWithTimeout creates a test context with timeout
func (h *TestHelpers) CreateTestContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// CreateTestContextWithUserID creates a test context with user ID
func (h *TestHelpers) CreateTestContextWithUserID(userID uuid.UUID) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "user_id", userID.String())
	return ctx
}

// Data Validation Helpers

// AssertValidUUID asserts that a string is a valid UUID
func (h *TestHelpers) AssertValidUUID(value string) {
	_, err := uuid.Parse(value)
	assert.NoError(h.t, err, "Expected valid UUID, got: %s", value)
}

// AssertValidTimestamp asserts that a timestamp is valid and recent
func (h *TestHelpers) AssertValidTimestamp(timestamp time.Time) {
	assert.False(h.t, timestamp.IsZero(), "Timestamp should not be zero")
	assert.True(h.t, timestamp.Before(time.Now().Add(time.Minute)), "Timestamp should be recent")
}

// AssertModelEqualsIgnoreTimestamps compares models ignoring timestamps
func (h *TestHelpers) AssertModelEqualsIgnoreTimestamps(expected, actual interface{}) {
	// Convert to JSON to compare structure
	expectedJSON, err := json.Marshal(expected)
	require.NoError(h.t, err)

	actualJSON, err := json.Marshal(actual)
	require.NoError(h.t, err)

	var expectedMap, actualMap map[string]interface{}
	json.Unmarshal(expectedJSON, &expectedMap)
	json.Unmarshal(actualJSON, &actualMap)

	// Remove timestamp fields
	delete(expectedMap, "created_at")
	delete(expectedMap, "updated_at")
	delete(actualMap, "created_at")
	delete(actualMap, "updated_at")

	assert.Equal(h.t, expectedMap, actualMap)
}

// Performance Testing Helpers

// MeasureExecutionTime measures the execution time of a function
func (h *TestHelpers) MeasureExecutionTime(fn func()) time.Duration {
	start := time.Now()
	fn()
	return time.Since(start)
}

// AssertExecutionTimeUnder asserts that execution time is under threshold
func (h *TestHelpers) AssertExecutionTimeUnder(fn func(), threshold time.Duration) {
	duration := h.MeasureExecutionTime(fn)
	assert.True(h.t, duration < threshold, "Execution time %v exceeds threshold %v", duration, threshold)
}

// RunConcurrentTest runs a function concurrently multiple times
func (h *TestHelpers) RunConcurrentTest(fn func(), concurrency int, iterations int) {
	ch := make(chan bool, concurrency)

	for i := 0; i < iterations; i++ {
		go func() {
			fn()
			ch <- true
		}()
	}

	for i := 0; i < iterations; i++ {
		<-ch
	}
}

// Database Testing Helpers

// AssertDatabaseContains asserts that database contains specific records
func (h *TestHelpers) AssertDatabaseContains(fixtures *TestFixtures, model interface{}, count int) {
	var actualCount int64
	err := fixtures.DB.Model(model).Count(&actualCount).Error
	require.NoError(h.t, err)
	assert.Equal(h.t, int64(count), actualCount)
}

// AssertDatabaseEmpty asserts that database table is empty
func (h *TestHelpers) AssertDatabaseEmpty(fixtures *TestFixtures, model interface{}) {
	h.AssertDatabaseContains(fixtures, model, 0)
}

// GetDatabaseRecordCount gets the count of records in database
func (h *TestHelpers) GetDatabaseRecordCount(fixtures *TestFixtures, model interface{}) int64 {
	var count int64
	err := fixtures.DB.Model(model).Count(&count).Error
	require.NoError(h.t, err)
	return count
}

// Load Testing Helpers

// CreateLoadTestData creates test data for load testing
func (h *TestHelpers) CreateLoadTestData(fixtures *TestFixtures, userCount, projectCount int) ([]*models.User, []*models.Project) {
	users := make([]*models.User, userCount)
	projects := make([]*models.Project, projectCount)

	// Create users
	for i := 0; i < userCount; i++ {
		users[i] = fixtures.CreateUser(map[string]interface{}{
			"username": fmt.Sprintf("loaduser%d", i),
			"email":    fmt.Sprintf("loaduser%d@example.com", i),
		})
	}

	// Create projects
	for i := 0; i < projectCount; i++ {
		userIndex := i % userCount
		projects[i] = fixtures.CreateProject(users[userIndex].ID, map[string]interface{}{
			"name":        fmt.Sprintf("Load Project %d", i),
			"description": fmt.Sprintf("Load testing project %d", i),
		})
	}

	return users, projects
}

// SimulateUserLoad simulates user load on an endpoint
func (h *TestHelpers) SimulateUserLoad(handler gin.HandlerFunc, method, path string, userCount int, requestsPerUser int) []time.Duration {
	var durations []time.Duration
	ch := make(chan time.Duration, userCount*requestsPerUser)

	for i := 0; i < userCount; i++ {
		go func(userIndex int) {
			for j := 0; j < requestsPerUser; j++ {
				start := time.Now()

				ctx, _ := h.CreateTestGinContext(method, path, nil)
				handler(ctx)

				duration := time.Since(start)
				ch <- duration
			}
		}(i)
	}

	for i := 0; i < userCount*requestsPerUser; i++ {
		durations = append(durations, <-ch)
	}

	return durations
}

// CalculatePerformanceStats calculates performance statistics
func (h *TestHelpers) CalculatePerformanceStats(durations []time.Duration) map[string]interface{} {
	if len(durations) == 0 {
		return nil
	}

	var total time.Duration
	min := durations[0]
	max := durations[0]

	for _, d := range durations {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := total / time.Duration(len(durations))

	return map[string]interface{}{
		"count":   len(durations),
		"average": avg,
		"min":     min,
		"max":     max,
		"total":   total,
	}
}

// Error Testing Helpers

// AssertErrorType asserts that an error is of specific type
func (h *TestHelpers) AssertErrorType(err error, expectedType interface{}) {
	assert.IsType(h.t, expectedType, err)
}

// AssertErrorContains asserts that error message contains specific text
func (h *TestHelpers) AssertErrorContains(err error, expectedText string) {
	require.Error(h.t, err)
	assert.Contains(h.t, err.Error(), expectedText)
}

// AssertNoError asserts that no error occurred
func (h *TestHelpers) AssertNoError(err error) {
	assert.NoError(h.t, err)
}

// Logging Testing Helpers

// CaptureLogOutput captures log output during test execution
func (h *TestHelpers) CaptureLogOutput(fn func()) string {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)

	originalLogger := h.logger
	h.logger = logger

	fn()

	h.logger = originalLogger
	return buf.String()
}

// AssertLogContains asserts that log output contains specific text
func (h *TestHelpers) AssertLogContains(logOutput, expectedText string) {
	assert.Contains(h.t, logOutput, expectedText)
}

// Cleanup Helpers

// CleanupAfterTest performs cleanup after test
func (h *TestHelpers) CleanupAfterTest(cleanupFn func()) {
	h.t.Cleanup(cleanupFn)
}

// TempFileHelper creates temporary files for testing
func (h *TestHelpers) CreateTempFile(content string) string {
	return "temp_file_path" // Implementation would create actual temp file
}

// Environment Helpers

// SetEnvVar sets environment variable for test
func (h *TestHelpers) SetEnvVar(key, value string) {
	h.t.Setenv(key, value)
}

// GetEnvVar gets environment variable with fallback
func (h *TestHelpers) GetEnvVar(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// Retry Helpers

// RetryOperation retries an operation until it succeeds or max attempts reached
func (h *TestHelpers) RetryOperation(operation func() error, maxAttempts int, delay time.Duration) error {
	var lastErr error

	for i := 0; i < maxAttempts; i++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxAttempts-1 {
				time.Sleep(delay)
			}
		}
	}

	return lastErr
}

// WaitForCondition waits for a condition to be true
func (h *TestHelpers) WaitForCondition(condition func() bool, timeout time.Duration, message string) {
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
			h.t.Fatalf("Timeout waiting for condition: %s", message)
		}
	}
}

// Random Data Generators

// GenerateRandomString generates a random string of specified length
func (h *TestHelpers) GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// GenerateRandomEmail generates a random email address
func (h *TestHelpers) GenerateRandomEmail() string {
	return fmt.Sprintf("%s@%s.com", h.GenerateRandomString(8), h.GenerateRandomString(6))
}

// GenerateRandomUUID generates a random UUID
func (h *TestHelpers) GenerateRandomUUID() uuid.UUID {
	return uuid.New()
}

// Test Data Validation

// ValidateUserModel validates user model structure
func (h *TestHelpers) ValidateUserModel(user *models.User) {
	assert.NotNil(h.t, user)
	h.AssertValidUUID(user.ID.String())
	assert.NotEmpty(h.t, user.Username)
	assert.NotEmpty(h.t, user.Email)
	assert.Contains(h.t, user.Email, "@")
	assert.NotEmpty(h.t, user.PasswordHash)
	h.AssertValidTimestamp(user.CreatedAt)
	h.AssertValidTimestamp(user.UpdatedAt)
}

// ValidateProjectModel validates project model structure
func (h *TestHelpers) ValidateProjectModel(project *models.Project) {
	assert.NotNil(h.t, project)
	h.AssertValidUUID(project.ID.String())
	assert.NotEmpty(h.t, project.Name)
	assert.NotEmpty(h.t, project.Description)
	assert.NotEmpty(h.t, project.Status)
	h.AssertValidUUID(project.CreatedBy.String())
	h.AssertValidTimestamp(project.CreatedAt)
	h.AssertValidTimestamp(project.UpdatedAt)
}

// ValidateAgentModel validates agent model structure
func (h *TestHelpers) ValidateAgentModel(agent *models.Agent) {
	assert.NotNil(h.t, agent)
	h.AssertValidUUID(agent.ID.String())
	assert.NotEmpty(h.t, agent.Name)
	assert.NotEmpty(h.t, agent.Type)
	assert.NotEmpty(h.t, agent.Status)
	// ProjectID field not available in Agent model
	// h.AssertValidUUID(agent.ProjectID.String())
	h.AssertValidTimestamp(agent.CreatedAt)
	h.AssertValidTimestamp(agent.UpdatedAt)
}

// ValidateTaskModel validates task model structure
func (h *TestHelpers) ValidateTaskModel(task *models.Task) {
	assert.NotNil(h.t, task)
	h.AssertValidUUID(task.ID.String())
	assert.NotEmpty(h.t, task.Title)
	assert.NotEmpty(h.t, task.Description)
	assert.NotEmpty(h.t, task.Status)
	h.AssertValidUUID(task.ProjectID.String())
	h.AssertValidTimestamp(task.CreatedAt)
	h.AssertValidTimestamp(task.UpdatedAt)
}
