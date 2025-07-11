package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// WorkflowService handles automated workflow execution
type WorkflowService struct {
	logger              *logrus.Logger
	workflowExecRepo    repositories.WorkflowExecutionRepository
	projectRepo         repositories.ProjectRepository
	assetService        *AssetService
	stateCheckerService *StateCheckerService
	gitService          *GitService
}

// NewWorkflowService creates a new workflow service
func NewWorkflowService(
	logger *logrus.Logger,
	workflowExecRepo repositories.WorkflowExecutionRepository,
	projectRepo repositories.ProjectRepository,
	assetService *AssetService,
	stateCheckerService *StateCheckerService,
	gitService *GitService,
) *WorkflowService {
	return &WorkflowService{
		logger:              logger,
		workflowExecRepo:    workflowExecRepo,
		projectRepo:         projectRepo,
		assetService:        assetService,
		stateCheckerService: stateCheckerService,
		gitService:          gitService,
	}
}

// WorkflowTrigger represents a workflow trigger configuration
type WorkflowTrigger struct {
	ProjectID    uuid.UUID
	WorkflowType string // asset_generation, state_check, full_analysis
	TriggerType  string // manual, scheduled, webhook, git_push
	Configuration map[string]interface{}
	TriggeredBy  uuid.UUID
}

// WorkflowResult represents the result of a workflow execution
type WorkflowResult struct {
	ExecutionID uuid.UUID
	Status      string
	Duration    time.Duration
	Result      map[string]interface{}
	Artifacts   []string
	Errors      []string
}

// ExecuteWorkflow executes a workflow based on the trigger
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, trigger WorkflowTrigger) (*WorkflowResult, error) {
	// Create workflow execution record
	execution := &models.WorkflowExecution{
		ProjectID:     trigger.ProjectID,
		WorkflowType:  trigger.WorkflowType,
		TriggerType:   trigger.TriggerType,
		Status:        "running",
		StartedAt:     timePtr(time.Now()),
		Configuration: s.mapToJSON(trigger.Configuration),
		TriggeredBy:   trigger.TriggeredBy,
	}

	if err := s.workflowExecRepo.Create(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to create workflow execution: %w", err)
	}

	// Execute workflow based on type
	result := &WorkflowResult{
		ExecutionID: execution.ID,
		Status:      "running",
		Result:      make(map[string]interface{}),
		Artifacts:   []string{},
		Errors:      []string{},
	}

	startTime := time.Now()
	
	switch trigger.WorkflowType {
	case "asset_generation":
		err := s.executeAssetGeneration(ctx, trigger, result)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Status = "failed"
		} else {
			result.Status = "completed"
		}
	case "state_check":
		err := s.executeStateCheck(ctx, trigger, result)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Status = "failed"
		} else {
			result.Status = "completed"
		}
	case "full_analysis":
		err := s.executeFullAnalysis(ctx, trigger, result)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Status = "failed"
		} else {
			result.Status = "completed"
		}
	case "git_sync":
		err := s.executeGitSync(ctx, trigger, result)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Status = "failed"
		} else {
			result.Status = "completed"
		}
	default:
		result.Errors = append(result.Errors, fmt.Sprintf("unknown workflow type: %s", trigger.WorkflowType))
		result.Status = "failed"
	}

	result.Duration = time.Since(startTime)

	// Update execution record
	execution.Status = result.Status
	execution.CompletedAt = timePtr(time.Now())
	execution.Duration = int(result.Duration.Seconds())
	execution.Result = s.mapToJSON(result.Result)
	execution.Artifacts = s.sliceToJSON(result.Artifacts)
	if len(result.Errors) > 0 {
		execution.ErrorMessage = result.Errors[0] // Store first error
	}

	if err := s.workflowExecRepo.Update(ctx, execution); err != nil {
		s.logger.WithError(err).Error("Failed to update workflow execution")
	}

	s.logger.WithFields(logrus.Fields{
		"execution_id":   execution.ID,
		"project_id":     trigger.ProjectID,
		"workflow_type":  trigger.WorkflowType,
		"status":         result.Status,
		"duration":       result.Duration,
	}).Info("Workflow execution completed")

	return result, nil
}

// executeAssetGeneration handles asset generation workflows
func (s *WorkflowService) executeAssetGeneration(ctx context.Context, trigger WorkflowTrigger, result *WorkflowResult) error {
	// Get configuration
	config := trigger.Configuration
	
	// Generate screenshots if requested
	if generateScreenshots, ok := config["generate_screenshots"].(bool); ok && generateScreenshots {
		if url, ok := config["screenshot_url"].(string); ok {
			asset, err := s.assetService.GenerateScreenshot(ctx, trigger.ProjectID, url, trigger.TriggeredBy)
			if err != nil {
				return fmt.Errorf("failed to generate screenshot: %w", err)
			}
			result.Artifacts = append(result.Artifacts, asset.FilePath)
			result.Result["screenshot_id"] = asset.ID
		}
	}
	
	// Generate videos if requested
	if generateVideos, ok := config["generate_videos"].(bool); ok && generateVideos {
		if url, ok := config["video_url"].(string); ok {
			duration := 30 // default duration
			if d, ok := config["video_duration"].(float64); ok {
				duration = int(d)
			}
			
			asset, err := s.assetService.GenerateVideo(ctx, trigger.ProjectID, url, duration, trigger.TriggeredBy)
			if err != nil {
				return fmt.Errorf("failed to generate video: %w", err)
			}
			result.Artifacts = append(result.Artifacts, asset.FilePath)
			result.Result["video_id"] = asset.ID
		}
	}
	
	// Get asset summary
	assets, err := s.assetService.GetProjectAssets(ctx, trigger.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get project assets: %w", err)
	}
	
	result.Result["total_assets"] = len(assets)
	result.Result["asset_types"] = s.getAssetTypeCounts(assets)
	
	return nil
}

// executeStateCheck handles state checking workflows
func (s *WorkflowService) executeStateCheck(ctx context.Context, trigger WorkflowTrigger, result *WorkflowResult) error {
	// Perform comprehensive state check
	healthCheck, err := s.stateCheckerService.CheckProjectState(ctx, trigger.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to check project state: %w", err)
	}
	
	// Store results
	result.Result["health_score"] = healthCheck.HealthScore
	result.Result["build_status"] = healthCheck.BuildStatus
	result.Result["test_status"] = healthCheck.TestStatus
	result.Result["lint_status"] = healthCheck.LintStatus
	result.Result["security_status"] = healthCheck.SecurityStatus
	result.Result["deployment_status"] = healthCheck.DeploymentStatus
	result.Result["coverage"] = healthCheck.Coverage
	result.Result["next_steps"] = healthCheck.NextSteps
	result.Result["errors"] = healthCheck.Errors
	result.Result["warnings"] = healthCheck.Warnings
	result.Result["suggestions"] = healthCheck.Suggestions
	
	return nil
}

// executeFullAnalysis handles comprehensive project analysis
func (s *WorkflowService) executeFullAnalysis(ctx context.Context, trigger WorkflowTrigger, result *WorkflowResult) error {
	// Perform state check
	if err := s.executeStateCheck(ctx, trigger, result); err != nil {
		return fmt.Errorf("state check failed: %w", err)
	}
	
	// Generate assets if URL provided
	if url, ok := trigger.Configuration["analysis_url"].(string); ok {
		// Generate screenshot
		asset, err := s.assetService.GenerateScreenshot(ctx, trigger.ProjectID, url, trigger.TriggeredBy)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to generate screenshot during full analysis")
		} else {
			result.Artifacts = append(result.Artifacts, asset.FilePath)
			result.Result["analysis_screenshot_id"] = asset.ID
		}
	}
	
	// Get project assets summary
	assets, err := s.assetService.GetProjectAssets(ctx, trigger.ProjectID)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get project assets during full analysis")
	} else {
		result.Result["asset_summary"] = map[string]interface{}{
			"total_assets": len(assets),
			"asset_types":  s.getAssetTypeCounts(assets),
		}
	}
	
	return nil
}

// executeGitSync handles git synchronization workflows
func (s *WorkflowService) executeGitSync(ctx context.Context, trigger WorkflowTrigger, result *WorkflowResult) error {
	// This would sync git repository information
	// For now, we'll just update the sync timestamp
	result.Result["sync_completed"] = true
	result.Result["sync_timestamp"] = time.Now().Format(time.RFC3339)
	
	return nil
}

// ScheduleWorkflow schedules a workflow for future execution
func (s *WorkflowService) ScheduleWorkflow(ctx context.Context, trigger WorkflowTrigger, scheduledTime time.Time) error {
	// Create scheduled workflow execution
	execution := &models.WorkflowExecution{
		ProjectID:     trigger.ProjectID,
		WorkflowType:  trigger.WorkflowType,
		TriggerType:   "scheduled",
		Status:        "pending",
		Configuration: s.mapToJSON(trigger.Configuration),
		TriggeredBy:   trigger.TriggeredBy,
	}

	if err := s.workflowExecRepo.Create(ctx, execution); err != nil {
		return fmt.Errorf("failed to create scheduled workflow execution: %w", err)
	}

	// TODO: Implement actual scheduling mechanism (cron, job queue, etc.)
	s.logger.WithFields(logrus.Fields{
		"execution_id":    execution.ID,
		"project_id":      trigger.ProjectID,
		"workflow_type":   trigger.WorkflowType,
		"scheduled_time":  scheduledTime,
	}).Info("Workflow scheduled")

	return nil
}

// GetWorkflowExecutions retrieves workflow executions for a project
func (s *WorkflowService) GetWorkflowExecutions(ctx context.Context, projectID uuid.UUID) ([]models.WorkflowExecution, error) {
	return s.workflowExecRepo.GetByProjectID(ctx, projectID)
}

// GetWorkflowExecution retrieves a specific workflow execution
func (s *WorkflowService) GetWorkflowExecution(ctx context.Context, executionID uuid.UUID) (*models.WorkflowExecution, error) {
	return s.workflowExecRepo.GetByID(ctx, executionID)
}

// CancelWorkflowExecution cancels a running workflow execution
func (s *WorkflowService) CancelWorkflowExecution(ctx context.Context, executionID uuid.UUID) error {
	execution, err := s.workflowExecRepo.GetByID(ctx, executionID)
	if err != nil {
		return fmt.Errorf("workflow execution not found: %w", err)
	}

	if execution.Status != "running" && execution.Status != "pending" {
		return fmt.Errorf("cannot cancel workflow execution with status: %s", execution.Status)
	}

	execution.Status = "cancelled"
	execution.CompletedAt = timePtr(time.Now())
	execution.ErrorMessage = "Cancelled by user"

	if err := s.workflowExecRepo.Update(ctx, execution); err != nil {
		return fmt.Errorf("failed to update workflow execution: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"execution_id": executionID,
		"project_id":   execution.ProjectID,
	}).Info("Workflow execution cancelled")

	return nil
}

// Helper methods

func (s *WorkflowService) getAssetTypeCounts(assets []models.ProjectAsset) map[string]int {
	counts := make(map[string]int)
	for _, asset := range assets {
		counts[asset.AssetType]++
	}
	return counts
}

func (s *WorkflowService) mapToJSON(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}
	
	// Simple JSON serialization for demonstration
	// In production, use json.Marshal
	parts := []string{}
	for key, value := range m {
		parts = append(parts, fmt.Sprintf("\"%s\": %v", key, value))
	}
	return "{" + fmt.Sprintf("%v", parts) + "}"
}

func (s *WorkflowService) sliceToJSON(s []string) string {
	if len(s) == 0 {
		return "[]"
	}
	
	// Simple JSON serialization for demonstration
	quoted := make([]string, len(s))
	for i, str := range s {
		quoted[i] = fmt.Sprintf("\"%s\"", str)
	}
	return "[" + fmt.Sprintf("%v", quoted) + "]"
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// Predefined workflow templates
var WorkflowTemplates = map[string]WorkflowTrigger{
	"quick_health_check": {
		WorkflowType: "state_check",
		TriggerType:  "manual",
		Configuration: map[string]interface{}{
			"check_build":      true,
			"check_tests":      true,
			"check_lint":       true,
			"check_security":   true,
			"check_deployment": true,
		},
	},
	"generate_demo_assets": {
		WorkflowType: "asset_generation",
		TriggerType:  "manual",
		Configuration: map[string]interface{}{
			"generate_screenshots": true,
			"generate_videos":      true,
			"video_duration":       30,
		},
	},
	"comprehensive_analysis": {
		WorkflowType: "full_analysis",
		TriggerType:  "manual",
		Configuration: map[string]interface{}{
			"generate_screenshots": true,
			"check_all_systems":    true,
			"create_report":        true,
		},
	},
}