package vibes

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kooshapari/kodevibe-go/internal/models"
)

// GitChecker implements Git repository analysis checks
type GitChecker struct {
	config                  models.VibeConfig
	enableCommitMessages    bool
	enableBranchNaming      bool
	enableLargeFiles        bool
	enableIgnoredFiles      bool
	enableCommitFrequency   bool
	enableFileHistory       bool
	enableConflictMarkers   bool
	maxCommitMessageLength  int
	maxFileSize             int64
	maxCommitsPerDay        int
}

// NewGitChecker creates a new Git checker
func NewGitChecker() *GitChecker {
	return &GitChecker{
		enableCommitMessages:   true,
		enableBranchNaming:     true,
		enableLargeFiles:       true,
		enableIgnoredFiles:     true,
		enableCommitFrequency:  true,
		enableFileHistory:      true,
		enableConflictMarkers:  true,
		maxCommitMessageLength: 72,
		maxFileSize:            100 * 1024 * 1024, // 100MB
		maxCommitsPerDay:       50,
	}
}

func (gc *GitChecker) Name() string          { return "GitVibe" }
func (gc *GitChecker) Type() models.VibeType { return models.VibeTypeGit }

func (gc *GitChecker) Configure(config models.VibeConfig) error {
	gc.config = config
	
	if val, exists := config.Settings["enable_commit_messages"]; exists {
		if boolVal, ok := val.(bool); ok {
			gc.enableCommitMessages = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_branch_naming"]; exists {
		if boolVal, ok := val.(bool); ok {
			gc.enableBranchNaming = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_large_files"]; exists {
		if boolVal, ok := val.(bool); ok {
			gc.enableLargeFiles = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_ignored_files"]; exists {
		if boolVal, ok := val.(bool); ok {
			gc.enableIgnoredFiles = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_commit_frequency"]; exists {
		if boolVal, ok := val.(bool); ok {
			gc.enableCommitFrequency = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_file_history"]; exists {
		if boolVal, ok := val.(bool); ok {
			gc.enableFileHistory = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_conflict_markers"]; exists {
		if boolVal, ok := val.(bool); ok {
			gc.enableConflictMarkers = boolVal
		}
	}
	
	if val, exists := config.Settings["max_commit_message_length"]; exists {
		if intVal, ok := val.(int); ok {
			gc.maxCommitMessageLength = intVal
		}
	}
	
	if val, exists := config.Settings["max_file_size"]; exists {
		if int64Val, ok := val.(int64); ok {
			gc.maxFileSize = int64Val
		}
	}
	
	if val, exists := config.Settings["max_commits_per_day"]; exists {
		if intVal, ok := val.(int); ok {
			gc.maxCommitsPerDay = intVal
		}
	}
	
	return nil
}

func (gc *GitChecker) Supports(filename string) bool {
	// Git checker supports all files in a Git repository
	return gc.isInGitRepo()
}

func (gc *GitChecker) Check(ctx context.Context, files []string) ([]models.Issue, error) {
	if !gc.isInGitRepo() {
		return []models.Issue{}, nil // Not a Git repo, nothing to check
	}
	
	var issues []models.Issue
	
	// Check individual files
	for _, file := range files {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		fileIssues, err := gc.checkFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check file %s: %w", file, err)
		}
		issues = append(issues, fileIssues...)
	}
	
	// Check repository-level issues
	repoIssues, err := gc.checkRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check repository: %w", err)
	}
	issues = append(issues, repoIssues...)
	
	return issues, nil
}

func (gc *GitChecker) checkFile(ctx context.Context, filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Check for conflict markers
	if gc.enableConflictMarkers {
		conflictIssues, err := gc.checkConflictMarkers(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to check conflict markers: %w", err)
		}
		issues = append(issues, conflictIssues...)
	}
	
	// Check if file should be ignored
	if gc.enableIgnoredFiles {
		ignoredIssues := gc.checkIgnoredFiles(filename)
		issues = append(issues, ignoredIssues...)
	}
	
	// Check file size in Git
	if gc.enableLargeFiles {
		largeFileIssues := gc.checkLargeFiles(filename)
		issues = append(issues, largeFileIssues...)
	}
	
	return issues, nil
}

func (gc *GitChecker) checkRepository(ctx context.Context) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Check commit messages
	if gc.enableCommitMessages {
		commitIssues, err := gc.checkCommitMessages(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check commit messages: %w", err)
		}
		issues = append(issues, commitIssues...)
	}
	
	// Check branch naming
	if gc.enableBranchNaming {
		branchIssues, err := gc.checkBranchNaming(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check branch naming: %w", err)
		}
		issues = append(issues, branchIssues...)
	}
	
	// Check commit frequency
	if gc.enableCommitFrequency {
		frequencyIssues, err := gc.checkCommitFrequency(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check commit frequency: %w", err)
		}
		issues = append(issues, frequencyIssues...)
	}
	
	return issues, nil
}

func (gc *GitChecker) checkConflictMarkers(filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	
	// Conflict marker patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`^<{7}\s`, "Git conflict marker (ours)", 1.0},
		{`^>{7}\s`, "Git conflict marker (theirs)", 1.0},
		{`^={7}$`, "Git conflict separator", 1.0},
		{`^<{7}$`, "Git conflict marker start", 1.0},
		{`^>{7}$`, "Git conflict marker end", 1.0},
		{`^\|{7}\s`, "Git conflict marker (common ancestor)", 1.0},
	}
	
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		
		for _, p := range patterns {
			if regexp.MustCompile(p.pattern).MatchString(line) {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeGit,
					Severity:    models.SeverityError,
					Title:       "Git conflict marker found",
					Description: p.description,
					File:        filename,
					Line:        lineNumber,
					Rule:        "conflict-marker",
					Message:     "Resolve merge conflicts before committing",
					Confidence:  p.confidence,
					Timestamp:   time.Now(),
				})
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	return issues, nil
}

func (gc *GitChecker) checkIgnoredFiles(filename string) []models.Issue {
	var issues []models.Issue
	
	// Check if file is tracked but should probably be ignored
	baseName := filepath.Base(filename)
	
	// Common patterns that should be gitignored
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`\.log$`, "Log files should be gitignored", 0.8},
		{`\.tmp$`, "Temporary files should be gitignored", 0.9},
		{`\.temp$`, "Temporary files should be gitignored", 0.9},
		{`\.cache$`, "Cache files should be gitignored", 0.9},
		{`\.DS_Store$`, "macOS metadata files should be gitignored", 1.0},
		{`Thumbs\.db$`, "Windows thumbnail cache should be gitignored", 1.0},
		{`\.swp$`, "Vim swap files should be gitignored", 1.0},
		{`\.swo$`, "Vim swap files should be gitignored", 1.0},
		{`~$`, "Backup files should be gitignored", 0.9},
		{`\.bak$`, "Backup files should be gitignored", 0.9},
		{`\.backup$`, "Backup files should be gitignored", 0.9},
		{`\.pid$`, "Process ID files should be gitignored", 0.8},
		{`\.lock$`, "Lock files should be gitignored", 0.8},
		{`node_modules/`, "Node.js dependencies should be gitignored", 1.0},
		{`\.git/`, "Git metadata should not be tracked", 1.0},
		{`\.env$`, "Environment files should be gitignored", 0.9},
		{`\.env\.local$`, "Local environment files should be gitignored", 1.0},
		{`config\.local$`, "Local config files should be gitignored", 0.8},
		{`\.pyc$`, "Python compiled files should be gitignored", 1.0},
		{`__pycache__/`, "Python cache directory should be gitignored", 1.0},
		{`\.class$`, "Java class files should be gitignored", 0.9},
		{`\.jar$`, "JAR files might need gitignoring", 0.6},
		{`\.exe$`, "Executable files might need gitignoring", 0.7},
		{`\.dll$`, "DLL files might need gitignoring", 0.8},
		{`\.so$`, "Shared object files might need gitignoring", 0.8},
		{`\.dylib$`, "Dynamic library files might need gitignoring", 0.8},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(baseName) || regexp.MustCompile(p.pattern).MatchString(filename) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeGit,
				Severity:    models.SeverityWarning,
				Title:       "File should probably be gitignored",
				Description: p.description,
				File:        filename,
				Line:        1,
				Rule:        "should-be-ignored",
				Message:     "Consider adding this file pattern to .gitignore",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (gc *GitChecker) checkLargeFiles(filename string) []models.Issue {
	var issues []models.Issue
	
	info, err := os.Stat(filename)
	if err != nil {
		return issues // Skip if we can't stat the file
	}
	
	if info.Size() > gc.maxFileSize {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeGit,
			Severity:    models.SeverityWarning,
			Title:       "Large file in Git repository",
			Description: fmt.Sprintf("File size is %d bytes (max: %d)", info.Size(), gc.maxFileSize),
			File:        filename,
			Line:        1,
			Rule:        "large-file",
			Message:     "Large files can slow down Git operations and increase repository size",
			Confidence:  1.0,
			Timestamp:   time.Now(),
		})
	}
	
	return issues
}

func (gc *GitChecker) checkCommitMessages(ctx context.Context) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Get recent commit messages
	cmd := exec.CommandContext(ctx, "git", "log", "--pretty=format:%s", "-10")
	output, err := cmd.Output()
	if err != nil {
		return issues, nil // Not a critical error, just skip
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	for i, message := range lines {
		if message == "" {
			continue
		}
		
		// Check commit message patterns
		messageIssues := gc.analyzeCommitMessage(message, i+1)
		issues = append(issues, messageIssues...)
	}
	
	return issues, nil
}

func (gc *GitChecker) analyzeCommitMessage(message string, commitNumber int) []models.Issue {
	var issues []models.Issue
	
	// Commit message patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
		severity    models.SeverityLevel
	}{
		{`^fix$`, "Generic commit message", 0.9, models.SeverityWarning},
		{`^update$`, "Generic commit message", 0.9, models.SeverityWarning},
		{`^changes$`, "Generic commit message", 0.9, models.SeverityWarning},
		{`^stuff$`, "Vague commit message", 0.9, models.SeverityWarning},
		{`^wip$`, "Work in progress commit", 0.8, models.SeverityInfo},
		{`^WIP`, "Work in progress commit", 0.8, models.SeverityInfo},
		{`^temp`, "Temporary commit", 0.8, models.SeverityInfo},
		{`^test`, "Test commit", 0.7, models.SeverityInfo},
		{`^asdf`, "Random keyboard mashing", 0.9, models.SeverityWarning},
		{`^qwer`, "Random keyboard mashing", 0.9, models.SeverityWarning},
		{`^\.\.\.$`, "Ellipsis as commit message", 0.9, models.SeverityWarning},
		{`^-+$`, "Dashes as commit message", 0.9, models.SeverityWarning},
		{`^[a-z]$`, "Single letter commit message", 0.9, models.SeverityWarning},
		{`^[0-9]+$`, "Numbers only commit message", 0.8, models.SeverityWarning},
		{`fuck|shit|damn|crap`, "Profanity in commit message", 0.9, models.SeverityWarning},
		{`^Merge branch`, "Merge commit (might be automatic)", 0.5, models.SeverityInfo},
		{`^Revert`, "Revert commit", 0.5, models.SeverityInfo},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(`(?i)`+p.pattern).MatchString(message) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeGit,
				Severity:    p.severity,
				Title:       "Commit message issue",
				Description: p.description,
				File:        ".git",
				Line:        commitNumber,
				Rule:        "commit-message",
				Message:     fmt.Sprintf("Commit message: '%s'", message),
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check message length
	if len(message) > gc.maxCommitMessageLength {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeGit,
			Severity:    models.SeverityInfo,
			Title:       "Long commit message",
			Description: fmt.Sprintf("Commit message is %d characters (recommended max: %d)", len(message), gc.maxCommitMessageLength),
			File:        ".git",
			Line:        commitNumber,
			Rule:        "commit-message-length",
			Message:     "Consider keeping commit messages concise",
			Confidence:  0.7,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for very short messages
	if len(message) < 10 {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeGit,
			Severity:    models.SeverityWarning,
			Title:       "Very short commit message",
			Description: fmt.Sprintf("Commit message is only %d characters", len(message)),
			File:        ".git",
			Line:        commitNumber,
			Rule:        "commit-message-length",
			Message:     "Commit messages should be descriptive",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	return issues
}

func (gc *GitChecker) checkBranchNaming(ctx context.Context) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Get current branch name
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return issues, nil // Not critical, skip
	}
	
	branchName := strings.TrimSpace(string(output))
	if branchName == "" {
		return issues, nil
	}
	
	// Branch naming patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
		severity    models.SeverityLevel
	}{
		{`\s`, "Spaces in branch name", 1.0, models.SeverityError},
		{`[A-Z]`, "Uppercase letters in branch name", 0.7, models.SeverityInfo},
		{`^test$`, "Generic 'test' branch name", 0.8, models.SeverityWarning},
		{`^temp$`, "Generic 'temp' branch name", 0.8, models.SeverityWarning},
		{`^tmp$`, "Generic 'tmp' branch name", 0.8, models.SeverityWarning},
		{`^branch$`, "Generic 'branch' branch name", 0.9, models.SeverityWarning},
		{`^new-branch$`, "Generic 'new-branch' name", 0.9, models.SeverityWarning},
		{`^feature$`, "Generic 'feature' branch name", 0.8, models.SeverityWarning},
		{`^fix$`, "Generic 'fix' branch name", 0.8, models.SeverityWarning},
		{`[^a-z0-9\-_\/]`, "Special characters in branch name", 0.8, models.SeverityWarning},
		{`--`, "Double hyphens in branch name", 0.7, models.SeverityInfo},
		{`__`, "Double underscores in branch name", 0.7, models.SeverityInfo},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(branchName) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeGit,
				Severity:    p.severity,
				Title:       "Branch naming issue",
				Description: p.description,
				File:        ".git",
				Line:        1,
				Rule:        "branch-naming",
				Message:     fmt.Sprintf("Branch name: '%s'", branchName),
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check branch name length
	if len(branchName) > 50 {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeGit,
			Severity:    models.SeverityInfo,
			Title:       "Long branch name",
			Description: fmt.Sprintf("Branch name is %d characters", len(branchName)),
			File:        ".git",
			Line:        1,
			Rule:        "branch-name-length",
			Message:     "Consider using shorter, more concise branch names",
			Confidence:  0.7,
			Timestamp:   time.Now(),
		})
	}
	
	return issues, nil
}

func (gc *GitChecker) checkCommitFrequency(ctx context.Context) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Get commits from last 24 hours
	cmd := exec.CommandContext(ctx, "git", "log", "--since=24 hours ago", "--oneline")
	output, err := cmd.Output()
	if err != nil {
		return issues, nil // Not critical, skip
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commitCount := 0
	if len(lines) > 0 && lines[0] != "" {
		commitCount = len(lines)
	}
	
	if commitCount > gc.maxCommitsPerDay {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeGit,
			Severity:    models.SeverityInfo,
			Title:       "High commit frequency",
			Description: fmt.Sprintf("%d commits in the last 24 hours", commitCount),
			File:        ".git",
			Line:        1,
			Rule:        "commit-frequency",
			Message:     "Consider squashing related commits before pushing",
			Confidence:  0.6,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for very few commits (might indicate infrequent development)
	if commitCount == 0 {
		// Check last commit date
		cmd = exec.CommandContext(ctx, "git", "log", "-1", "--pretty=format:%cr")
		output, err = cmd.Output()
		if err == nil {
			lastCommit := strings.TrimSpace(string(output))
			if strings.Contains(lastCommit, "week") || strings.Contains(lastCommit, "month") {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeGit,
					Severity:    models.SeverityInfo,
					Title:       "Infrequent commits",
					Description: fmt.Sprintf("Last commit was %s", lastCommit),
					File:        ".git",
					Line:        1,
					Rule:        "commit-frequency",
					Message:     "Consider making more frequent commits for better history",
					Confidence:  0.5,
					Timestamp:   time.Now(),
				})
			}
		}
	}
	
	return issues, nil
}

func (gc *GitChecker) isInGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}