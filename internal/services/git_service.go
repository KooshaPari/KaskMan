package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// GitService handles git repository operations and project detection
type GitService struct {
	logger           *logrus.Logger
	projectRepo      repositories.ProjectRepository
	gitRepo          repositories.GitRepositoryRepository
	projectStateRepo repositories.ProjectStateRepository
}

// NewGitService creates a new git service instance
func NewGitService(
	logger *logrus.Logger,
	projectRepo repositories.ProjectRepository,
	gitRepo repositories.GitRepositoryRepository,
	projectStateRepo repositories.ProjectStateRepository,
) *GitService {
	return &GitService{
		logger:           logger,
		projectRepo:      projectRepo,
		gitRepo:          gitRepo,
		projectStateRepo: projectStateRepo,
	}
}

// GitProjectInfo represents discovered git project information
type GitProjectInfo struct {
	Path           string
	Name           string
	RemoteURL      string
	Branch         string
	LastCommitSHA  string
	LastCommitDate time.Time
	Language       string
	Framework      string
	HasReadme      bool
	HasCI          bool
	HasTests       bool
	HasLinter      bool
}

// ScanDirectory scans a directory for git repositories and returns project information
func (s *GitService) ScanDirectory(ctx context.Context, scanPath string) ([]GitProjectInfo, error) {
	var projects []GitProjectInfo
	
	err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if info.IsDir() && info.Name() == ".git" {
			projectPath := filepath.Dir(path)
			projectInfo, err := s.analyzeGitProject(projectPath)
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to analyze git project at %s", projectPath)
				return nil // Continue scanning other projects
			}
			projects = append(projects, *projectInfo)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}
	
	return projects, nil
}

// analyzeGitProject analyzes a git project and extracts information
func (s *GitService) analyzeGitProject(projectPath string) (*GitProjectInfo, error) {
	// Get project name from directory
	projectName := filepath.Base(projectPath)
	
	// Get git information
	remoteURL, err := s.getGitRemoteURL(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote URL: %w", err)
	}
	
	branch, err := s.getGitBranch(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	
	lastCommitSHA, err := s.getLastCommitSHA(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get last commit SHA: %w", err)
	}
	
	lastCommitDate, err := s.getLastCommitDate(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get last commit date: %w", err)
	}
	
	// Detect language and framework
	language, framework := s.detectLanguageAndFramework(projectPath)
	
	// Check for common files
	hasReadme := s.fileExists(projectPath, "README.md") || s.fileExists(projectPath, "readme.md")
	hasCI := s.hasCI(projectPath)
	hasTests := s.hasTests(projectPath)
	hasLinter := s.hasLinter(projectPath)
	
	return &GitProjectInfo{
		Path:           projectPath,
		Name:           projectName,
		RemoteURL:      remoteURL,
		Branch:         branch,
		LastCommitSHA:  lastCommitSHA,
		LastCommitDate: lastCommitDate,
		Language:       language,
		Framework:      framework,
		HasReadme:      hasReadme,
		HasCI:          hasCI,
		HasTests:       hasTests,
		HasLinter:      hasLinter,
	}, nil
}

// AddProjectFromGit creates a new project from git repository information
func (s *GitService) AddProjectFromGit(ctx context.Context, projectInfo GitProjectInfo, userID uuid.UUID) (*models.Project, error) {
	// Create project
	project := &models.Project{
		Name:        projectInfo.Name,
		Description: fmt.Sprintf("Auto-detected project from git repository: %s", projectInfo.RemoteURL),
		Type:        "development",
		Status:      "active",
		Priority:    "medium",
		CreatedBy:   userID,
		Tags:        fmt.Sprintf("[\"%s\", \"%s\", \"auto-detected\"]", projectInfo.Language, projectInfo.Framework),
		Metadata:    fmt.Sprintf("{\"language\": \"%s\", \"framework\": \"%s\", \"path\": \"%s\"}", projectInfo.Language, projectInfo.Framework, projectInfo.Path),
	}
	
	// Save project
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	
	// Create git repository record
	gitRepo := &models.GitRepository{
		ProjectID:     project.ID,
		RepositoryURL: projectInfo.RemoteURL,
		Branch:        projectInfo.Branch,
		LastCommitSHA: projectInfo.LastCommitSHA,
		LastSyncAt:    &projectInfo.LastCommitDate,
		Status:        "active",
	}
	
	if err := s.gitRepo.Create(ctx, gitRepo); err != nil {
		return nil, fmt.Errorf("failed to create git repository record: %w", err)
	}
	
	// Create initial project state
	projectState := &models.ProjectState{
		ProjectID:        project.ID,
		BuildStatus:      "unknown",
		TestStatus:       "unknown",
		LintStatus:       "unknown",
		SecurityStatus:   "unknown",
		DeploymentStatus: "unknown",
		LastCheckAt:      &projectInfo.LastCommitDate,
		HealthScore:      s.calculateInitialHealthScore(projectInfo),
		NextSteps:        s.generateInitialNextSteps(projectInfo),
		ReadmePath:       s.findReadmePath(projectInfo.Path),
	}
	
	if err := s.projectStateRepo.Create(ctx, projectState); err != nil {
		return nil, fmt.Errorf("failed to create project state: %w", err)
	}
	
	s.logger.WithFields(logrus.Fields{
		"project_id": project.ID,
		"name":       project.Name,
		"path":       projectInfo.Path,
	}).Info("Project created from git repository")
	
	return project, nil
}

// Helper methods for git operations
func (s *GitService) getGitRemoteURL(projectPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *GitService) getGitBranch(projectPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *GitService) getLastCommitSHA(projectPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *GitService) getLastCommitDate(projectPath string) (time.Time, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%ct")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	
	timestamp := strings.TrimSpace(string(output))
	return time.Unix(0, 0).Add(time.Duration(len(timestamp)) * time.Second), nil
}

// Helper methods for project analysis
func (s *GitService) detectLanguageAndFramework(projectPath string) (string, string) {
	// Check for common files and determine language/framework
	if s.fileExists(projectPath, "package.json") {
		return "javascript", s.detectJSFramework(projectPath)
	}
	if s.fileExists(projectPath, "go.mod") {
		return "go", s.detectGoFramework(projectPath)
	}
	if s.fileExists(projectPath, "requirements.txt") || s.fileExists(projectPath, "pyproject.toml") {
		return "python", s.detectPythonFramework(projectPath)
	}
	if s.fileExists(projectPath, "Cargo.toml") {
		return "rust", "cargo"
	}
	if s.fileExists(projectPath, "pom.xml") || s.fileExists(projectPath, "build.gradle") {
		return "java", "maven"
	}
	
	return "unknown", "unknown"
}

func (s *GitService) detectJSFramework(projectPath string) string {
	packageJSON := filepath.Join(projectPath, "package.json")
	if s.fileExists(projectPath, "next.config.js") {
		return "nextjs"
	}
	if s.fileExists(projectPath, "nuxt.config.js") {
		return "nuxt"
	}
	if s.fileExists(projectPath, "vue.config.js") {
		return "vue"
	}
	if s.fileExists(projectPath, "angular.json") {
		return "angular"
	}
	// Could parse package.json for more detailed detection
	return "node"
}

func (s *GitService) detectGoFramework(projectPath string) string {
	// Check for common Go frameworks
	if s.fileExists(projectPath, "main.go") {
		return "go"
	}
	return "go"
}

func (s *GitService) detectPythonFramework(projectPath string) string {
	if s.fileExists(projectPath, "manage.py") {
		return "django"
	}
	if s.fileExists(projectPath, "app.py") || s.fileExists(projectPath, "wsgi.py") {
		return "flask"
	}
	if s.fileExists(projectPath, "fastapi") {
		return "fastapi"
	}
	return "python"
}

func (s *GitService) fileExists(projectPath, fileName string) bool {
	_, err := os.Stat(filepath.Join(projectPath, fileName))
	return err == nil
}

func (s *GitService) hasCI(projectPath string) bool {
	ciPaths := []string{
		".github/workflows",
		".gitlab-ci.yml",
		".travis.yml",
		"circle.yml",
		".circleci/config.yml",
		"azure-pipelines.yml",
	}
	
	for _, ciPath := range ciPaths {
		if s.fileExists(projectPath, ciPath) {
			return true
		}
	}
	return false
}

func (s *GitService) hasTests(projectPath string) bool {
	testPaths := []string{
		"test",
		"tests",
		"spec",
		"__tests__",
		"test.go",
		"*_test.go",
	}
	
	for _, testPath := range testPaths {
		if s.fileExists(projectPath, testPath) {
			return true
		}
	}
	return false
}

func (s *GitService) hasLinter(projectPath string) bool {
	linterFiles := []string{
		".eslintrc.json",
		".eslintrc.js",
		".golangci.yml",
		"pyproject.toml",
		".flake8",
		".pylintrc",
		"tslint.json",
	}
	
	for _, linterFile := range linterFiles {
		if s.fileExists(projectPath, linterFile) {
			return true
		}
	}
	return false
}

func (s *GitService) calculateInitialHealthScore(projectInfo GitProjectInfo) int {
	score := 50 // Base score
	
	if projectInfo.HasReadme {
		score += 10
	}
	if projectInfo.HasCI {
		score += 20
	}
	if projectInfo.HasTests {
		score += 15
	}
	if projectInfo.HasLinter {
		score += 5
	}
	
	return score
}

func (s *GitService) generateInitialNextSteps(projectInfo GitProjectInfo) string {
	steps := []string{}
	
	if !projectInfo.HasReadme {
		steps = append(steps, "Add README.md documentation")
	}
	if !projectInfo.HasCI {
		steps = append(steps, "Set up CI/CD pipeline")
	}
	if !projectInfo.HasTests {
		steps = append(steps, "Add test coverage")
	}
	if !projectInfo.HasLinter {
		steps = append(steps, "Configure code linting")
	}
	
	if len(steps) == 0 {
		return "Project appears well-configured. Consider adding more comprehensive documentation or advanced CI/CD features."
	}
	
	return "Next steps: " + strings.Join(steps, ", ")
}

func (s *GitService) findReadmePath(projectPath string) string {
	readmeFiles := []string{"README.md", "readme.md", "README.rst", "readme.rst", "README.txt"}
	
	for _, readmeFile := range readmeFiles {
		if s.fileExists(projectPath, readmeFile) {
			return filepath.Join(projectPath, readmeFile)
		}
	}
	
	return ""
}