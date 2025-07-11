package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// StateCheckerService handles project state checking (CI/CD, tests, linting, etc.)
type StateCheckerService struct {
	logger           *logrus.Logger
	projectStateRepo repositories.ProjectStateRepository
	projectRepo      repositories.ProjectRepository
	gitRepo          repositories.GitRepositoryRepository
}

// NewStateCheckerService creates a new state checker service
func NewStateCheckerService(
	logger *logrus.Logger,
	projectStateRepo repositories.ProjectStateRepository,
	projectRepo repositories.ProjectRepository,
	gitRepo repositories.GitRepositoryRepository,
) *StateCheckerService {
	return &StateCheckerService{
		logger:           logger,
		projectStateRepo: projectStateRepo,
		projectRepo:      projectRepo,
		gitRepo:          gitRepo,
	}
}

// ProjectHealthCheck represents a comprehensive project health check
type ProjectHealthCheck struct {
	ProjectID        uuid.UUID
	BuildStatus      string
	TestStatus       string
	LintStatus       string
	SecurityStatus   string
	DeploymentStatus string
	Coverage         float64
	HealthScore      int
	NextSteps        []string
	Errors           []string
	Warnings         []string
	Suggestions      []string
}

// CheckProjectState performs a comprehensive state check on a project
func (s *StateCheckerService) CheckProjectState(ctx context.Context, projectID uuid.UUID) (*ProjectHealthCheck, error) {
	// Get project and git repository
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	gitRepo, err := s.gitRepo.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("git repository not found: %w", err)
	}

	// Extract project path from metadata
	projectPath := s.extractProjectPath(project.Metadata)
	if projectPath == "" {
		return nil, fmt.Errorf("project path not found in metadata")
	}

	// Initialize health check
	healthCheck := &ProjectHealthCheck{
		ProjectID: projectID,
		NextSteps: []string{},
		Errors:    []string{},
		Warnings:  []string{},
		Suggestions: []string{},
	}

	// Check build status
	healthCheck.BuildStatus = s.checkBuildStatus(projectPath)
	
	// Check test status and coverage
	healthCheck.TestStatus, healthCheck.Coverage = s.checkTestStatus(projectPath)
	
	// Check lint status
	healthCheck.LintStatus = s.checkLintStatus(projectPath)
	
	// Check security status
	healthCheck.SecurityStatus = s.checkSecurityStatus(projectPath)
	
	// Check deployment status
	healthCheck.DeploymentStatus = s.checkDeploymentStatus(projectPath)
	
	// Calculate health score
	healthCheck.HealthScore = s.calculateHealthScore(healthCheck)
	
	// Generate next steps and recommendations
	s.generateRecommendations(healthCheck, projectPath)
	
	// Save state to database
	if err := s.saveProjectState(ctx, healthCheck); err != nil {
		return nil, fmt.Errorf("failed to save project state: %w", err)
	}
	
	s.logger.WithFields(logrus.Fields{
		"project_id":    projectID,
		"health_score":  healthCheck.HealthScore,
		"build_status":  healthCheck.BuildStatus,
		"test_status":   healthCheck.TestStatus,
		"lint_status":   healthCheck.LintStatus,
	}).Info("Project state check completed")
	
	return healthCheck, nil
}

// checkBuildStatus checks if the project builds successfully
func (s *StateCheckerService) checkBuildStatus(projectPath string) string {
	// Check different build systems
	if s.fileExists(projectPath, "package.json") {
		return s.checkNodeBuild(projectPath)
	}
	if s.fileExists(projectPath, "go.mod") {
		return s.checkGoBuild(projectPath)
	}
	if s.fileExists(projectPath, "requirements.txt") || s.fileExists(projectPath, "pyproject.toml") {
		return s.checkPythonBuild(projectPath)
	}
	if s.fileExists(projectPath, "Cargo.toml") {
		return s.checkRustBuild(projectPath)
	}
	if s.fileExists(projectPath, "pom.xml") || s.fileExists(projectPath, "build.gradle") {
		return s.checkJavaBuild(projectPath)
	}
	
	return "unknown"
}

// checkTestStatus checks test status and coverage
func (s *StateCheckerService) checkTestStatus(projectPath string) (string, float64) {
	// Check different test frameworks
	if s.fileExists(projectPath, "package.json") {
		return s.checkNodeTests(projectPath)
	}
	if s.fileExists(projectPath, "go.mod") {
		return s.checkGoTests(projectPath)
	}
	if s.fileExists(projectPath, "requirements.txt") || s.fileExists(projectPath, "pyproject.toml") {
		return s.checkPythonTests(projectPath)
	}
	if s.fileExists(projectPath, "Cargo.toml") {
		return s.checkRustTests(projectPath)
	}
	if s.fileExists(projectPath, "pom.xml") || s.fileExists(projectPath, "build.gradle") {
		return s.checkJavaTests(projectPath)
	}
	
	return "unknown", 0.0
}

// checkLintStatus checks linting status
func (s *StateCheckerService) checkLintStatus(projectPath string) string {
	// Check different linters
	if s.fileExists(projectPath, ".eslintrc.json") || s.fileExists(projectPath, ".eslintrc.js") {
		return s.checkESLint(projectPath)
	}
	if s.fileExists(projectPath, ".golangci.yml") {
		return s.checkGolangCI(projectPath)
	}
	if s.fileExists(projectPath, ".flake8") || s.fileExists(projectPath, ".pylintrc") {
		return s.checkPythonLint(projectPath)
	}
	if s.fileExists(projectPath, "rustfmt.toml") {
		return s.checkRustfmt(projectPath)
	}
	
	return "unknown"
}

// checkSecurityStatus checks for security vulnerabilities
func (s *StateCheckerService) checkSecurityStatus(projectPath string) string {
	// Check different security tools
	if s.fileExists(projectPath, "package.json") {
		return s.checkNodeSecurity(projectPath)
	}
	if s.fileExists(projectPath, "go.mod") {
		return s.checkGoSecurity(projectPath)
	}
	if s.fileExists(projectPath, "requirements.txt") {
		return s.checkPythonSecurity(projectPath)
	}
	if s.fileExists(projectPath, "Cargo.toml") {
		return s.checkRustSecurity(projectPath)
	}
	
	return "unknown"
}

// checkDeploymentStatus checks deployment status
func (s *StateCheckerService) checkDeploymentStatus(projectPath string) string {
	// Check for deployment indicators
	if s.fileExists(projectPath, "Dockerfile") {
		return s.checkDockerDeployment(projectPath)
	}
	if s.fileExists(projectPath, "vercel.json") || s.fileExists(projectPath, "netlify.toml") {
		return s.checkStaticDeployment(projectPath)
	}
	if s.fileExists(projectPath, "kubernetes") || s.fileExists(projectPath, "k8s") {
		return s.checkKubernetesDeployment(projectPath)
	}
	
	return "unknown"
}

// Language-specific build checks
func (s *StateCheckerService) checkNodeBuild(projectPath string) string {
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkGoBuild(projectPath string) string {
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkPythonBuild(projectPath string) string {
	// Check if Python code compiles
	cmd := exec.Command("python", "-m", "py_compile", ".")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkRustBuild(projectPath string) string {
	cmd := exec.Command("cargo", "build")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkJavaBuild(projectPath string) string {
	if s.fileExists(projectPath, "pom.xml") {
		cmd := exec.Command("mvn", "compile")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "failure"
		}
	} else if s.fileExists(projectPath, "build.gradle") {
		cmd := exec.Command("gradle", "build")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "failure"
		}
	}
	return "success"
}

// Language-specific test checks
func (s *StateCheckerService) checkNodeTests(projectPath string) (string, float64) {
	cmd := exec.Command("npm", "test", "--", "--coverage")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "failure", 0.0
	}
	
	coverage := s.parseNodeCoverage(string(output))
	return "success", coverage
}

func (s *StateCheckerService) checkGoTests(projectPath string) (string, float64) {
	cmd := exec.Command("go", "test", "-cover", "./...")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "failure", 0.0
	}
	
	coverage := s.parseGoCoverage(string(output))
	return "success", coverage
}

func (s *StateCheckerService) checkPythonTests(projectPath string) (string, float64) {
	cmd := exec.Command("python", "-m", "pytest", "--cov", ".")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "failure", 0.0
	}
	
	coverage := s.parsePythonCoverage(string(output))
	return "success", coverage
}

func (s *StateCheckerService) checkRustTests(projectPath string) (string, float64) {
	cmd := exec.Command("cargo", "test")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure", 0.0
	}
	return "success", 0.0 // Rust doesn't have built-in coverage
}

func (s *StateCheckerService) checkJavaTests(projectPath string) (string, float64) {
	if s.fileExists(projectPath, "pom.xml") {
		cmd := exec.Command("mvn", "test")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "failure", 0.0
		}
	} else if s.fileExists(projectPath, "build.gradle") {
		cmd := exec.Command("gradle", "test")
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			return "failure", 0.0
		}
	}
	return "success", 0.0
}

// Language-specific lint checks
func (s *StateCheckerService) checkESLint(projectPath string) string {
	cmd := exec.Command("npx", "eslint", ".")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkGolangCI(projectPath string) string {
	cmd := exec.Command("golangci-lint", "run")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkPythonLint(projectPath string) string {
	cmd := exec.Command("flake8", ".")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkRustfmt(projectPath string) string {
	cmd := exec.Command("cargo", "fmt", "--check")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

// Security checks
func (s *StateCheckerService) checkNodeSecurity(projectPath string) string {
	cmd := exec.Command("npm", "audit")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkGoSecurity(projectPath string) string {
	cmd := exec.Command("gosec", "./...")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkPythonSecurity(projectPath string) string {
	cmd := exec.Command("bandit", "-r", ".")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkRustSecurity(projectPath string) string {
	cmd := exec.Command("cargo", "audit")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

// Deployment checks
func (s *StateCheckerService) checkDockerDeployment(projectPath string) string {
	cmd := exec.Command("docker", "build", "-t", "test", ".")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

func (s *StateCheckerService) checkStaticDeployment(projectPath string) string {
	// Check if build artifacts exist
	if s.fileExists(projectPath, "dist") || s.fileExists(projectPath, "build") {
		return "success"
	}
	return "unknown"
}

func (s *StateCheckerService) checkKubernetesDeployment(projectPath string) string {
	// Check if Kubernetes manifests are valid
	cmd := exec.Command("kubectl", "apply", "--dry-run=client", "-f", "kubernetes/")
	cmd.Dir = projectPath
	if err := cmd.Run(); err != nil {
		return "failure"
	}
	return "success"
}

// Helper methods
func (s *StateCheckerService) fileExists(projectPath, fileName string) bool {
	_, err := os.Stat(filepath.Join(projectPath, fileName))
	return err == nil
}

func (s *StateCheckerService) extractProjectPath(metadata string) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(metadata), &data); err != nil {
		return ""
	}
	
	if path, ok := data["path"].(string); ok {
		return path
	}
	
	return ""
}

func (s *StateCheckerService) parseNodeCoverage(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Statements") && strings.Contains(line, "%") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, "%") {
					if coverage, err := strconv.ParseFloat(strings.TrimSuffix(part, "%"), 64); err == nil {
						return coverage
					}
				}
			}
		}
	}
	return 0.0
}

func (s *StateCheckerService) parseGoCoverage(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "coverage:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "coverage:" && i+1 < len(parts) {
					coverageStr := strings.TrimSuffix(parts[i+1], "%")
					if coverage, err := strconv.ParseFloat(coverageStr, 64); err == nil {
						return coverage
					}
				}
			}
		}
	}
	return 0.0
}

func (s *StateCheckerService) parsePythonCoverage(output string) float64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "TOTAL") && strings.Contains(line, "%") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, "%") {
					coverageStr := strings.TrimSuffix(part, "%")
					if coverage, err := strconv.ParseFloat(coverageStr, 64); err == nil {
						return coverage
					}
				}
			}
		}
	}
	return 0.0
}

func (s *StateCheckerService) calculateHealthScore(healthCheck *ProjectHealthCheck) int {
	score := 0
	
	// Build status (25 points)
	if healthCheck.BuildStatus == "success" {
		score += 25
	} else if healthCheck.BuildStatus == "failure" {
		score -= 10
	}
	
	// Test status (20 points)
	if healthCheck.TestStatus == "success" {
		score += 20
	} else if healthCheck.TestStatus == "failure" {
		score -= 5
	}
	
	// Test coverage (15 points)
	if healthCheck.Coverage >= 90 {
		score += 15
	} else if healthCheck.Coverage >= 70 {
		score += 10
	} else if healthCheck.Coverage >= 50 {
		score += 5
	}
	
	// Lint status (15 points)
	if healthCheck.LintStatus == "success" {
		score += 15
	} else if healthCheck.LintStatus == "failure" {
		score -= 5
	}
	
	// Security status (15 points)
	if healthCheck.SecurityStatus == "success" {
		score += 15
	} else if healthCheck.SecurityStatus == "failure" {
		score -= 10
	}
	
	// Deployment status (10 points)
	if healthCheck.DeploymentStatus == "success" {
		score += 10
	}
	
	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}
	
	return score
}

func (s *StateCheckerService) generateRecommendations(healthCheck *ProjectHealthCheck, projectPath string) {
	// Generate next steps based on status
	if healthCheck.BuildStatus == "failure" {
		healthCheck.NextSteps = append(healthCheck.NextSteps, "Fix build errors")
		healthCheck.Errors = append(healthCheck.Errors, "Build is failing")
	}
	
	if healthCheck.TestStatus == "failure" {
		healthCheck.NextSteps = append(healthCheck.NextSteps, "Fix failing tests")
		healthCheck.Errors = append(healthCheck.Errors, "Tests are failing")
	}
	
	if healthCheck.Coverage < 50 {
		healthCheck.NextSteps = append(healthCheck.NextSteps, "Increase test coverage")
		healthCheck.Warnings = append(healthCheck.Warnings, "Low test coverage")
	}
	
	if healthCheck.LintStatus == "failure" {
		healthCheck.NextSteps = append(healthCheck.NextSteps, "Fix linting errors")
		healthCheck.Warnings = append(healthCheck.Warnings, "Linting issues detected")
	}
	
	if healthCheck.SecurityStatus == "failure" {
		healthCheck.NextSteps = append(healthCheck.NextSteps, "Address security vulnerabilities")
		healthCheck.Errors = append(healthCheck.Errors, "Security vulnerabilities found")
	}
	
	if healthCheck.DeploymentStatus == "unknown" {
		healthCheck.Suggestions = append(healthCheck.Suggestions, "Consider adding deployment configuration")
	}
}

func (s *StateCheckerService) saveProjectState(ctx context.Context, healthCheck *ProjectHealthCheck) error {
	// Get existing state or create new one
	state, err := s.projectStateRepo.GetByProjectID(ctx, healthCheck.ProjectID)
	if err != nil {
		// Create new state
		state = &models.ProjectState{
			ProjectID: healthCheck.ProjectID,
		}
	}
	
	// Update state
	now := time.Now()
	state.BuildStatus = healthCheck.BuildStatus
	state.TestStatus = healthCheck.TestStatus
	state.LintStatus = healthCheck.LintStatus
	state.SecurityStatus = healthCheck.SecurityStatus
	state.DeploymentStatus = healthCheck.DeploymentStatus
	state.Coverage = healthCheck.Coverage
	state.LastCheckAt = &now
	state.HealthScore = healthCheck.HealthScore
	state.NextSteps = strings.Join(healthCheck.NextSteps, ", ")
	
	if len(healthCheck.Errors) > 0 {
		state.CheckErrors = strings.Join(healthCheck.Errors, "; ")
	}
	
	if state.ID == uuid.Nil {
		return s.projectStateRepo.Create(ctx, state)
	} else {
		return s.projectStateRepo.Update(ctx, state)
	}
}