package testutils

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, entity interface{}) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, entity interface{}) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockUserRepository) SoftDelete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, entities interface{}, filters repositories.Filter) error {
	args := m.Called(ctx, entities, filters)
	return args.Error(0)
}

func (m *MockUserRepository) ListWithPagination(ctx context.Context, entities interface{}, pagination repositories.Pagination, filters repositories.Filter) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, entities, pagination, filters)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockUserRepository) Count(ctx context.Context, entity interface{}, filters repositories.Filter) (int64, error) {
	args := m.Called(ctx, entity, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) Exists(ctx context.Context, id uuid.UUID, entity interface{}) (bool, error) {
	args := m.Called(ctx, id, entity)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) BatchCreate(ctx context.Context, entities interface{}) error {
	args := m.Called(ctx, entities)
	return args.Error(0)
}

func (m *MockUserRepository) BatchUpdate(ctx context.Context, entities interface{}) error {
	args := m.Called(ctx, entities)
	return args.Error(0)
}

func (m *MockUserRepository) BatchDelete(ctx context.Context, ids []uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, ids, entity)
	return args.Error(0)
}

func (m *MockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, hashedPassword string) error {
	args := m.Called(ctx, userID, hashedPassword)
	return args.Error(0)
}

func (m *MockUserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockUserRepository) GetActiveUsers(ctx context.Context, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockUserRepository) GetUsersByRole(ctx context.Context, role string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, role, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockUserRepository) SearchUsers(ctx context.Context, query string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, query, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockUserRepository) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockProjectRepository is a mock implementation of ProjectRepository
type MockProjectRepository struct {
	mock.Mock
}

func (m *MockProjectRepository) Create(ctx context.Context, entity interface{}) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockProjectRepository) GetByID(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockProjectRepository) Update(ctx context.Context, entity interface{}) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockProjectRepository) Delete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockProjectRepository) SoftDelete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockProjectRepository) List(ctx context.Context, entities interface{}, filters repositories.Filter) error {
	args := m.Called(ctx, entities, filters)
	return args.Error(0)
}

func (m *MockProjectRepository) ListWithPagination(ctx context.Context, entities interface{}, pagination repositories.Pagination, filters repositories.Filter) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, entities, pagination, filters)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) Count(ctx context.Context, entity interface{}, filters repositories.Filter) (int64, error) {
	args := m.Called(ctx, entity, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProjectRepository) Exists(ctx context.Context, id uuid.UUID, entity interface{}) (bool, error) {
	args := m.Called(ctx, id, entity)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectRepository) BatchCreate(ctx context.Context, entities interface{}) error {
	args := m.Called(ctx, entities)
	return args.Error(0)
}

func (m *MockProjectRepository) BatchUpdate(ctx context.Context, entities interface{}) error {
	args := m.Called(ctx, entities)
	return args.Error(0)
}

func (m *MockProjectRepository) BatchDelete(ctx context.Context, ids []uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, ids, entity)
	return args.Error(0)
}

func (m *MockProjectRepository) GetByUser(ctx context.Context, userID uuid.UUID, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, userID, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) GetByStatus(ctx context.Context, status string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, status, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) GetByType(ctx context.Context, projectType string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, projectType, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) GetByPriority(ctx context.Context, priority string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, priority, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) GetByDateRange(ctx context.Context, startDate, endDate time.Time, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, startDate, endDate, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) GetActiveProjects(ctx context.Context, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) SearchProjects(ctx context.Context, query string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, query, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) GetProjectStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockProjectRepository) GetProjectsByTags(ctx context.Context, tags []string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, tags, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockProjectRepository) UpdateStatus(ctx context.Context, projectID uuid.UUID, status string) error {
	args := m.Called(ctx, projectID, status)
	return args.Error(0)
}

func (m *MockProjectRepository) UpdatePriority(ctx context.Context, projectID uuid.UUID, priority string) error {
	args := m.Called(ctx, projectID, priority)
	return args.Error(0)
}

func (m *MockProjectRepository) ArchiveProject(ctx context.Context, projectID uuid.UUID) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockProjectRepository) RestoreProject(ctx context.Context, projectID uuid.UUID) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

// MockAgentRepository is a mock implementation of AgentRepository
type MockAgentRepository struct {
	mock.Mock
}

func (m *MockAgentRepository) Create(ctx context.Context, entity interface{}) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockAgentRepository) GetByID(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockAgentRepository) Update(ctx context.Context, entity interface{}) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockAgentRepository) Delete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockAgentRepository) SoftDelete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockAgentRepository) List(ctx context.Context, entities interface{}, filters repositories.Filter) error {
	args := m.Called(ctx, entities, filters)
	return args.Error(0)
}

func (m *MockAgentRepository) ListWithPagination(ctx context.Context, entities interface{}, pagination repositories.Pagination, filters repositories.Filter) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, entities, pagination, filters)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockAgentRepository) Count(ctx context.Context, entity interface{}, filters repositories.Filter) (int64, error) {
	args := m.Called(ctx, entity, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockAgentRepository) Exists(ctx context.Context, id uuid.UUID, entity interface{}) (bool, error) {
	args := m.Called(ctx, id, entity)
	return args.Bool(0), args.Error(1)
}

func (m *MockAgentRepository) BatchCreate(ctx context.Context, entities interface{}) error {
	args := m.Called(ctx, entities)
	return args.Error(0)
}

func (m *MockAgentRepository) BatchUpdate(ctx context.Context, entities interface{}) error {
	args := m.Called(ctx, entities)
	return args.Error(0)
}

func (m *MockAgentRepository) BatchDelete(ctx context.Context, ids []uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, ids, entity)
	return args.Error(0)
}

func (m *MockAgentRepository) GetByProject(ctx context.Context, projectID uuid.UUID, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, projectID, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockAgentRepository) GetByType(ctx context.Context, agentType string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, agentType, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockAgentRepository) GetByStatus(ctx context.Context, status string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, status, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockAgentRepository) GetActiveAgents(ctx context.Context, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockAgentRepository) GetAgentsByCapabilities(ctx context.Context, capabilities []string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, capabilities, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockAgentRepository) UpdateStatus(ctx context.Context, agentID uuid.UUID, status string) error {
	args := m.Called(ctx, agentID, status)
	return args.Error(0)
}

func (m *MockAgentRepository) UpdateConfiguration(ctx context.Context, agentID uuid.UUID, config map[string]interface{}) error {
	args := m.Called(ctx, agentID, config)
	return args.Error(0)
}

func (m *MockAgentRepository) GetAgentStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockAgentRepository) GetAgentPerformance(ctx context.Context, agentID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, agentID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockAgentRepository) AssignToProject(ctx context.Context, agentID, projectID uuid.UUID) error {
	args := m.Called(ctx, agentID, projectID)
	return args.Error(0)
}

func (m *MockAgentRepository) UnassignFromProject(ctx context.Context, agentID uuid.UUID) error {
	args := m.Called(ctx, agentID)
	return args.Error(0)
}

// MockTaskRepository is a mock implementation of TaskRepository
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, entity interface{}) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockTaskRepository) Update(ctx context.Context, entity interface{}) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

func (m *MockTaskRepository) Delete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockTaskRepository) SoftDelete(ctx context.Context, id uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, id, entity)
	return args.Error(0)
}

func (m *MockTaskRepository) List(ctx context.Context, entities interface{}, filters repositories.Filter) error {
	args := m.Called(ctx, entities, filters)
	return args.Error(0)
}

func (m *MockTaskRepository) ListWithPagination(ctx context.Context, entities interface{}, pagination repositories.Pagination, filters repositories.Filter) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, entities, pagination, filters)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) Count(ctx context.Context, entity interface{}, filters repositories.Filter) (int64, error) {
	args := m.Called(ctx, entity, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTaskRepository) Exists(ctx context.Context, id uuid.UUID, entity interface{}) (bool, error) {
	args := m.Called(ctx, id, entity)
	return args.Bool(0), args.Error(1)
}

func (m *MockTaskRepository) BatchCreate(ctx context.Context, entities interface{}) error {
	args := m.Called(ctx, entities)
	return args.Error(0)
}

func (m *MockTaskRepository) BatchUpdate(ctx context.Context, entities interface{}) error {
	args := m.Called(ctx, entities)
	return args.Error(0)
}

func (m *MockTaskRepository) BatchDelete(ctx context.Context, ids []uuid.UUID, entity interface{}) error {
	args := m.Called(ctx, ids, entity)
	return args.Error(0)
}

func (m *MockTaskRepository) GetByProject(ctx context.Context, projectID uuid.UUID, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, projectID, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) GetByAgent(ctx context.Context, agentID uuid.UUID, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, agentID, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) GetByStatus(ctx context.Context, status string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, status, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) GetByPriority(ctx context.Context, priority string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, priority, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) GetByDueDate(ctx context.Context, dueDate time.Time, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, dueDate, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) GetOverdueTasks(ctx context.Context, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) GetTasksByDateRange(ctx context.Context, startDate, endDate time.Time, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, startDate, endDate, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) UpdateStatus(ctx context.Context, taskID uuid.UUID, status string) error {
	args := m.Called(ctx, taskID, status)
	return args.Error(0)
}

func (m *MockTaskRepository) UpdatePriority(ctx context.Context, taskID uuid.UUID, priority string) error {
	args := m.Called(ctx, taskID, priority)
	return args.Error(0)
}

func (m *MockTaskRepository) AssignToAgent(ctx context.Context, taskID, agentID uuid.UUID) error {
	args := m.Called(ctx, taskID, agentID)
	return args.Error(0)
}

func (m *MockTaskRepository) UnassignFromAgent(ctx context.Context, taskID uuid.UUID) error {
	args := m.Called(ctx, taskID)
	return args.Error(0)
}

func (m *MockTaskRepository) GetTaskStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTaskRepository) GetTasksByTags(ctx context.Context, tags []string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, tags, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) SearchTasks(ctx context.Context, query string, pagination repositories.Pagination) (*repositories.PaginationResult, error) {
	args := m.Called(ctx, query, pagination)
	return args.Get(0).(*repositories.PaginationResult), args.Error(1)
}

func (m *MockTaskRepository) GetTaskProgress(ctx context.Context, taskID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, taskID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockTaskRepository) UpdateProgress(ctx context.Context, taskID uuid.UUID, progress float64) error {
	args := m.Called(ctx, taskID, progress)
	return args.Error(0)
}

func (m *MockTaskRepository) SetDueDate(ctx context.Context, taskID uuid.UUID, dueDate time.Time) error {
	args := m.Called(ctx, taskID, dueDate)
	return args.Error(0)
}

func (m *MockTaskRepository) RemoveDueDate(ctx context.Context, taskID uuid.UUID) error {
	args := m.Called(ctx, taskID)
	return args.Error(0)
}

// MockWebSocketHub is a mock implementation of WebSocket Hub
type MockWebSocketHub struct {
	mock.Mock
}

func (m *MockWebSocketHub) RegisterClient(client interface{}) {
	m.Called(client)
}

func (m *MockWebSocketHub) UnregisterClient(client interface{}) {
	m.Called(client)
}

func (m *MockWebSocketHub) BroadcastMessage(message interface{}) {
	m.Called(message)
}

func (m *MockWebSocketHub) SendToClient(clientID string, message interface{}) {
	m.Called(clientID, message)
}

func (m *MockWebSocketHub) SendToSubscribers(topic string, message interface{}) {
	m.Called(topic, message)
}

func (m *MockWebSocketHub) GetClientCount() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockWebSocketHub) GetActiveTopics() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockWebSocketHub) Start() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockWebSocketHub) Stop() error {
	args := m.Called()
	return args.Error(0)
}

// MockRnDCoordinator is a mock implementation of R&D Coordinator
type MockRnDCoordinator struct {
	mock.Mock
}

func (m *MockRnDCoordinator) CoordinateAgents(ctx context.Context, projectID uuid.UUID, agentIDs []uuid.UUID) error {
	args := m.Called(ctx, projectID, agentIDs)
	return args.Error(0)
}

func (m *MockRnDCoordinator) DistributeTask(ctx context.Context, taskID uuid.UUID, agentIDs []uuid.UUID) error {
	args := m.Called(ctx, taskID, agentIDs)
	return args.Error(0)
}

func (m *MockRnDCoordinator) GetAgentLoad(ctx context.Context, agentID uuid.UUID) (map[string]interface{}, error) {
	args := m.Called(ctx, agentID)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRnDCoordinator) OptimizeAgentAllocation(ctx context.Context, projectID uuid.UUID) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockRnDCoordinator) GetCoordinationStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockPatternRecognizer is a mock implementation of Pattern Recognizer
type MockPatternRecognizer struct {
	mock.Mock
}

func (m *MockPatternRecognizer) AnalyzePatterns(ctx context.Context, data interface{}) ([]models.Pattern, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]models.Pattern), args.Error(1)
}

func (m *MockPatternRecognizer) GetPatternTrends(ctx context.Context, patternType string) (map[string]interface{}, error) {
	args := m.Called(ctx, patternType)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockPatternRecognizer) ValidatePattern(ctx context.Context, patternID uuid.UUID) (bool, error) {
	args := m.Called(ctx, patternID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPatternRecognizer) GetPatternConfidence(ctx context.Context, patternID uuid.UUID) (float64, error) {
	args := m.Called(ctx, patternID)
	return args.Get(0).(float64), args.Error(1)
}

// MockProjectGenerator is a mock implementation of Project Generator
type MockProjectGenerator struct {
	mock.Mock
}

func (m *MockProjectGenerator) GenerateProject(ctx context.Context, requirements map[string]interface{}) (*models.Project, error) {
	args := m.Called(ctx, requirements)
	return args.Get(0).(*models.Project), args.Error(1)
}

func (m *MockProjectGenerator) GenerateProposal(ctx context.Context, projectID uuid.UUID, requirements map[string]interface{}) (*models.Proposal, error) {
	args := m.Called(ctx, projectID, requirements)
	return args.Get(0).(*models.Proposal), args.Error(1)
}

func (m *MockProjectGenerator) ValidateProjectStructure(ctx context.Context, projectID uuid.UUID) (bool, error) {
	args := m.Called(ctx, projectID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectGenerator) GetGeneratorStats(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// MockLearningEngine is a mock implementation of Learning Engine
type MockLearningEngine struct {
	mock.Mock
}

func (m *MockLearningEngine) LearnFromData(ctx context.Context, data interface{}) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockLearningEngine) GetInsights(ctx context.Context, domain string) ([]models.Insight, error) {
	args := m.Called(ctx, domain)
	return args.Get(0).([]models.Insight), args.Error(1)
}

func (m *MockLearningEngine) GetLearningProgress(ctx context.Context) (map[string]interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLearningEngine) TrainModel(ctx context.Context, modelType string, data interface{}) error {
	args := m.Called(ctx, modelType, data)
	return args.Error(0)
}

func (m *MockLearningEngine) EvaluateModel(ctx context.Context, modelType string, testData interface{}) (map[string]interface{}, error) {
	args := m.Called(ctx, modelType, testData)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLearningEngine) GetModelPerformance(ctx context.Context, modelType string) (map[string]interface{}, error) {
	args := m.Called(ctx, modelType)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// Helper function to create a mock that returns specific values
func CreateMockWithReturn(mockFunc func() interface{}, err error) *mock.Mock {
	m := &mock.Mock{}
	m.On("Execute").Return(mockFunc(), err)
	return m
}

// Helper function to create a mock that always succeeds
func CreateSuccessfulMock() *mock.Mock {
	m := &mock.Mock{}
	m.On("Execute").Return(nil)
	return m
}

// Helper function to create a mock that always fails
func CreateFailingMock(err error) *mock.Mock {
	m := &mock.Mock{}
	m.On("Execute").Return(err)
	return m
}
