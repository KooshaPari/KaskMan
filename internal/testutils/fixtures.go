package testutils

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"gorm.io/gorm"
)

// TestFixtures provides test data creation utilities
type TestFixtures struct {
	DB *gorm.DB
}

// NewTestFixtures creates a new test fixtures instance
func NewTestFixtures(db *gorm.DB) *TestFixtures {
	return &TestFixtures{DB: db}
}

// User creation fixtures
func (f *TestFixtures) CreateUser(overrides ...map[string]interface{}) *models.User {
	user := &models.User{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
		},
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "$2a$10$YourHashedPasswordHere", // bcrypt hash of "password"
		Role:         "user",
		IsActive:     true,
	}

	// Apply overrides
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			switch key {
			case "username":
				user.Username = value.(string)
			case "email":
				user.Email = value.(string)
			case "password":
				user.PasswordHash = value.(string)
			case "role":
				user.Role = value.(string)
			case "active":
				user.IsActive = value.(bool)
			}
		}
	}

	f.DB.Create(user)
	return user
}

func (f *TestFixtures) CreateAdmin(overrides ...map[string]interface{}) *models.User {
	defaults := map[string]interface{}{
		"username": "admin",
		"email":    "admin@example.com",
		"role":     "admin",
	}

	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			defaults[key] = value
		}
	}

	return f.CreateUser(defaults)
}

// Project creation fixtures
func (f *TestFixtures) CreateProject(userID uuid.UUID, overrides ...map[string]interface{}) *models.Project {
	project := &models.Project{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
		},
		Name:        "Test Project",
		Description: "A test project for testing purposes",
		Status:      "active",
		Priority:    "medium",
		Type:        "research",
		CreatedBy:   userID,
		StartDate:   &[]time.Time{time.Now()}[0],
	}

	// Apply overrides
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			switch key {
			case "name":
				project.Name = value.(string)
			case "description":
				project.Description = value.(string)
			case "status":
				project.Status = value.(string)
			case "priority":
				project.Priority = value.(string)
			case "type":
				project.Type = value.(string)
			case "end_date":
				if value != nil {
					endDate := value.(time.Time)
					project.EndDate = &endDate
				}
			}
		}
	}

	f.DB.Create(project)
	return project
}

// Agent creation fixtures
func (f *TestFixtures) CreateAgent(projectID uuid.UUID, overrides ...map[string]interface{}) *models.Agent {
	agent := &models.Agent{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
		},
		Name:        "Test Agent",
		Type:        "researcher",
		Status:      "active",
	}

	// Apply overrides
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			switch key {
			case "name":
				agent.Name = value.(string)
			case "type":
				agent.Type = value.(string)
			case "status":
				agent.Status = value.(string)
			case "description":
				// Description field not available in Agent model
			}
		}
	}

	f.DB.Create(agent)
	return agent
}

// Task creation fixtures
func (f *TestFixtures) CreateTask(projectID uuid.UUID, overrides ...map[string]interface{}) *models.Task {
	task := &models.Task{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Title:       "Test Task",
		Description: "A test task for testing purposes",
		Status:      "pending",
		Priority:    "medium",
		ProjectID:   &projectID,
	}

	// Apply overrides
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			switch key {
			case "title":
				task.Title = value.(string)
			case "description":
				task.Description = value.(string)
			case "status":
				task.Status = value.(string)
			case "priority":
				task.Priority = value.(string)
			case "agent_id":
				if value != nil {
					agentID := value.(uuid.UUID)
					task.AgentID = &agentID
				}
			case "due_date":
				// DueDate field not available in Task model
			}
		}
	}

	f.DB.Create(task)
	return task
}

// Proposal creation fixtures
func (f *TestFixtures) CreateProposal(projectID uuid.UUID, overrides ...map[string]interface{}) *models.Proposal {
	proposal := &models.Proposal{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Title:       "Test Proposal",
		Description: "A test proposal for testing purposes",
		Status:      "pending",
		Priority:    "medium",
		ProjectID:   &projectID,
	}

	// Apply overrides
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			switch key {
			case "title":
				proposal.Title = value.(string)
			case "description":
				proposal.Description = value.(string)
			case "status":
				proposal.Status = value.(string)
			case "priority":
				proposal.Priority = value.(string)
			case "agent_id":
				// AgentID field not available in Proposal model
			}
		}
	}

	f.DB.Create(proposal)
	return proposal
}

// Pattern creation fixtures
func (f *TestFixtures) CreatePattern(overrides ...map[string]interface{}) *models.Pattern {
	pattern := &models.Pattern{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Name:        "Test Pattern",
		Description: "A test pattern for testing purposes",
		Type:        "behavioral",
		Confidence:  0.85,
		LastSeen:    time.Now(),
	}

	// Apply overrides
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			switch key {
			case "name":
				pattern.Name = value.(string)
			case "description":
				pattern.Description = value.(string)
			case "type":
				pattern.Type = value.(string)
			case "confidence":
				pattern.Confidence = value.(float64)
			}
		}
	}

	f.DB.Create(pattern)
	return pattern
}

// Insight creation fixtures
func (f *TestFixtures) CreateInsight(overrides ...map[string]interface{}) *models.Insight {
	insight := &models.Insight{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Title:       "Test Insight",
		Description: "A test insight for testing purposes",
		Type:        "trend",
		Confidence:  0.75,
	}

	// Apply overrides
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			switch key {
			case "title":
				insight.Title = value.(string)
			case "description":
				insight.Description = value.(string)
			case "type":
				insight.Type = value.(string)
			case "confidence":
				insight.Confidence = value.(float64)
			}
		}
	}

	f.DB.Create(insight)
	return insight
}

// ActivityLog creation fixtures
func (f *TestFixtures) CreateActivityLog(overrides ...map[string]interface{}) *models.ActivityLog {
	log := &models.ActivityLog{
		BaseModel: models.BaseModel{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Action:    "test_action",
		Resource:  "test_resource",
		Details:   "{\"test\": \"data\"}",
		Success:   true,
	}

	// Apply overrides
	if len(overrides) > 0 {
		for key, value := range overrides[0] {
			switch key {
			case "user_id":
				if value != nil {
					userID := value.(uuid.UUID)
					log.UserID = &userID
				}
			case "action":
				log.Action = value.(string)
			case "resource":
				log.Resource = value.(string)
			case "resource_id":
				if value != nil {
					resourceID := value.(uuid.UUID)
					log.ResourceID = &resourceID
				}
			case "details":
				if details, ok := value.(string); ok {
					log.Details = details
				} else if detailsMap, ok := value.(map[string]interface{}); ok {
					if detailsJSON, err := json.Marshal(detailsMap); err == nil {
						log.Details = string(detailsJSON)
					}
				}
			case "success":
				log.Success = value.(bool)
			case "error_message":
				log.ErrorMessage = value.(string)
			case "ip_address":
				log.IPAddress = value.(string)
			case "user_agent":
				log.UserAgent = value.(string)
			}
		}
	}

	f.DB.Create(log)
	return log
}

// Complex scenario fixtures
func (f *TestFixtures) CreateCompleteProject(userID uuid.UUID) (*models.Project, *models.Agent, *models.Task, *models.Proposal) {
	project := f.CreateProject(userID, map[string]interface{}{
		"name":        "Complete Test Project",
		"description": "A project with all related entities",
		"status":      "active",
	})

	agent := f.CreateAgent(project.ID, map[string]interface{}{
		"name":        "Project Agent",
		"type":        "coordinator",
		"status":      "active",
		"description": "Agent managing the complete project",
	})

	task := f.CreateTask(project.ID, map[string]interface{}{
		"title":       "Project Task",
		"description": "A task for the complete project",
		"status":      "in_progress",
		"agent_id":    agent.ID,
	})

	proposal := f.CreateProposal(project.ID, map[string]interface{}{
		"title":       "Project Proposal",
		"description": "A proposal for the complete project",
		"status":      "approved",
		"agent_id":    agent.ID,
	})

	return project, agent, task, proposal
}

// Create multiple entities
func (f *TestFixtures) CreateMultipleUsers(count int) []*models.User {
	users := make([]*models.User, count)
	for i := 0; i < count; i++ {
		users[i] = f.CreateUser(map[string]interface{}{
			"username": fmt.Sprintf("user%d", i+1),
			"email":    fmt.Sprintf("user%d@example.com", i+1),
		})
	}
	return users
}

func (f *TestFixtures) CreateMultipleProjects(userID uuid.UUID, count int) []*models.Project {
	projects := make([]*models.Project, count)
	for i := 0; i < count; i++ {
		projects[i] = f.CreateProject(userID, map[string]interface{}{
			"name":        fmt.Sprintf("Project %d", i+1),
			"description": fmt.Sprintf("Description for project %d", i+1),
		})
	}
	return projects
}

func (f *TestFixtures) CreateMultipleAgents(projectID uuid.UUID, count int) []*models.Agent {
	agents := make([]*models.Agent, count)
	agentTypes := []string{"researcher", "analyst", "coordinator", "developer"}

	for i := 0; i < count; i++ {
		agents[i] = f.CreateAgent(projectID, map[string]interface{}{
			"name":        fmt.Sprintf("Agent %d", i+1),
			"type":        agentTypes[i%len(agentTypes)],
			"description": fmt.Sprintf("Description for agent %d", i+1),
		})
	}
	return agents
}

// Clean up fixtures
func (f *TestFixtures) CleanupAll() error {
	tables := []string{
		"activity_logs",
		"insights",
		"patterns",
		"proposals",
		"tasks",
		"agents",
		"projects",
		"users",
	}

	for _, table := range tables {
		if err := f.DB.Exec(fmt.Sprintf("DELETE FROM %s", table)).Error; err != nil {
			return err
		}
	}

	return nil
}

// Batch operations
func (f *TestFixtures) BatchCreateUsers(count int, batchSize int) []*models.User {
	users := make([]*models.User, count)

	for i := 0; i < count; i += batchSize {
		end := i + batchSize
		if end > count {
			end = count
		}

		batch := make([]*models.User, end-i)
		for j := i; j < end; j++ {
			batch[j-i] = &models.User{
				BaseModel: models.BaseModel{
					ID:        uuid.New(),
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Username:     fmt.Sprintf("batchuser%d", j+1),
				Email:        fmt.Sprintf("batchuser%d@example.com", j+1),
				PasswordHash: "$2a$10$YourHashedPasswordHere",
				Role:         "user",
				IsActive:     true,
			}
			users[j] = batch[j-i]
		}

		f.DB.Create(batch)
	}

	return users
}

// Performance testing fixtures
func (f *TestFixtures) CreatePerformanceTestData(userCount, projectCount, agentCount, taskCount int) {
	// Create users
	users := f.BatchCreateUsers(userCount, 100)

	// Create projects
	for _, user := range users {
		projects := f.CreateMultipleProjects(user.ID, projectCount)

		// Create agents and tasks for each project
		for _, project := range projects {
			agents := f.CreateMultipleAgents(project.ID, agentCount)

			// Create tasks
			for i := 0; i < taskCount; i++ {
				var agentID *uuid.UUID
				if len(agents) > 0 {
					agentID = &agents[i%len(agents)].ID
				}

				f.CreateTask(project.ID, map[string]interface{}{
					"title":       fmt.Sprintf("Task %d", i+1),
					"description": fmt.Sprintf("Performance test task %d", i+1),
					"agent_id":    agentID,
				})
			}
		}
	}
}
