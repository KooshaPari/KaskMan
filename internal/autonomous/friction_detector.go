package autonomous

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/sirupsen/logrus"
)

// FrictionDetector identifies friction points from user behavior and system logs
type FrictionDetector struct {
	logger            *logrus.Logger
	commandAnalyzer   *CommandAnalyzer
	behaviorTracker   *BehaviorTracker
	frictionThresholds map[string]int
}

// CommandAnalyzer detects patterns in command execution
type CommandAnalyzer struct {
	logger        *logrus.Logger
	commandCache  map[string]*CommandPattern
	sessionCache  map[string]*SessionData
}

// CommandPattern represents analysis of command usage
type CommandPattern struct {
	Command         string            `json:"command"`
	Frequency       int               `json:"frequency"`
	Context         string            `json:"context"`
	Arguments       []string          `json:"arguments"`
	Timing          []time.Duration   `json:"timing"`
	ErrorPatterns   []string          `json:"error_patterns"`
	SuccessPatterns []string          `json:"success_patterns"`
	UserFrustration float64           `json:"user_frustration"` // 0.0 to 1.0
}

// SessionData tracks user behavior within a session
type SessionData struct {
	SessionID       string                 `json:"session_id"`
	StartTime       time.Time              `json:"start_time"`
	Commands        []string               `json:"commands"`
	Repetitions     map[string]int         `json:"repetitions"`
	ErrorSequences  []string               `json:"error_sequences"`
	WorkflowContext map[string]interface{} `json:"workflow_context"`
}

// BehaviorTracker monitors user behavior patterns
type BehaviorTracker struct {
	logger           *logrus.Logger
	frustrationScore float64
	workflowEfficiency float64
	toolUsagePatterns map[string]int
}

// NewFrictionDetector creates a new friction detection system
func NewFrictionDetector(logger *logrus.Logger) *FrictionDetector {
	return &FrictionDetector{
		logger:          logger,
		commandAnalyzer: NewCommandAnalyzer(logger),
		behaviorTracker: NewBehaviorTracker(logger),
		frictionThresholds: map[string]int{
			"command_repetition":    3,  // 3+ repetitions indicate friction
			"error_frequency":       2,  // 2+ errors in sequence
			"workflow_inefficiency": 5,  // 5+ steps for simple task
			"tool_switching":        4,  // 4+ tool switches
		},
	}
}

// AnalyzeCommandFriction detects friction in command usage patterns
func (fd *FrictionDetector) AnalyzeCommandFriction(ctx context.Context, activity models.ActivityLog) (*FrictionPoint, error) {
	// Parse command from activity details
	var details map[string]interface{}
	if err := json.Unmarshal([]byte(activity.Details), &details); err != nil {
		return nil, err
	}

	command, ok := details["command"].(string)
	if !ok {
		return nil, nil // Not a command activity
	}

	// Analyze command pattern
	pattern := fd.commandAnalyzer.AnalyzeCommand(command, activity)
	
	// Check for friction indicators
	if pattern.Frequency >= fd.frictionThresholds["command_repetition"] {
		return fd.createFrictionFromCommand(activity, pattern), nil
	}

	// Detect specific friction patterns
	if fd.isClipboardFriction(command, pattern) {
		return fd.createClipboardFriction(activity, pattern), nil
	}

	if fd.isCompileLintTestFriction(command, pattern) {
		return fd.createCompileFriction(activity, pattern), nil
	}

	return nil, nil
}

// isClipboardFriction detects clipboard-related friction
func (fd *FrictionDetector) isClipboardFriction(command string, pattern *CommandPattern) bool {
	clipboardIndicators := []string{
		"pbpaste",
		"pbcopy", 
		"xclip",
		"clipboard",
		"paste",
		"copy",
	}

	imageIndicators := []string{
		"image",
		"png",
		"jpg",
		"jpeg",
		"screenshot",
		"capture",
	}

	// Check if command involves clipboard operations
	commandLower := strings.ToLower(command)
	hasClipboard := false
	hasImage := false

	for _, indicator := range clipboardIndicators {
		if strings.Contains(commandLower, indicator) {
			hasClipboard = true
			break
		}
	}

	for _, indicator := range imageIndicators {
		if strings.Contains(commandLower, indicator) {
			hasImage = true
			break
		}
	}

	// Check error patterns for manual file handling
	manualFileHandling := false
	for _, errorPattern := range pattern.ErrorPatterns {
		if strings.Contains(errorPattern, "file not found") ||
		   strings.Contains(errorPattern, "path") ||
		   strings.Contains(errorPattern, "directory") {
			manualFileHandling = true
			break
		}
	}

	return (hasClipboard || hasImage) && (pattern.Frequency > 2 || manualFileHandling)
}

// isCompileLintTestFriction detects compile/lint/test repetition friction
func (fd *FrictionDetector) isCompileLintTestFriction(command string, pattern *CommandPattern) bool {
	buildCommands := []string{
		"tsc",
		"typescript",
		"eslint",
		"lint",
		"test",
		"jest",
		"npm test",
		"yarn test",
		"go test",
		"cargo test",
		"pytest",
		"npm run build",
		"make",
		"build",
	}

	commandLower := strings.ToLower(command)
	for _, buildCmd := range buildCommands {
		if strings.Contains(commandLower, buildCmd) {
			// High frequency of build commands indicates friction
			return pattern.Frequency >= 3
		}
	}

	return false
}

// createClipboardFriction creates friction point for clipboard issues
func (fd *FrictionDetector) createClipboardFriction(activity models.ActivityLog, pattern *CommandPattern) *FrictionPoint {
	return &FrictionPoint{
		ID:          uuid.New(),
		Type:        "command_repetition",
		Context:     "zsh_terminal",
		Description: "Manual clipboard image handling causing workflow friction",
		Frequency:   pattern.Frequency,
		Impact:      fd.calculateImpact(pattern),
		FirstSeen:   activity.CreatedAt,
		LastSeen:    activity.CreatedAt,
		UserID:      *activity.UserID,
		
		CommandPatterns: []string{pattern.Command},
		EnvironmentData: map[string]interface{}{
			"shell": "zsh",
			"os":    fd.detectOS(pattern.Command),
			"clipboard_tool": fd.detectClipboardTool(pattern.Command),
		},
		UserBehavior: map[string]interface{}{
			"frustration_score": pattern.UserFrustration,
			"repetition_rate":   pattern.Frequency,
			"error_rate":        len(pattern.ErrorPatterns),
		},
	}
}

// createCompileFriction creates friction point for compile/lint/test repetition
func (fd *FrictionDetector) createCompileFriction(activity models.ActivityLog, pattern *CommandPattern) *FrictionPoint {
	return &FrictionPoint{
		ID:          uuid.New(),
		Type:        "command_repetition", 
		Context:     "development_workflow",
		Description: fmt.Sprintf("Repetitive execution of %s command indicating need for automation", pattern.Command),
		Frequency:   pattern.Frequency,
		Impact:      fd.calculateImpact(pattern),
		FirstSeen:   activity.CreatedAt,
		LastSeen:    activity.CreatedAt,
		UserID:      *activity.UserID,
		
		CommandPatterns: []string{pattern.Command},
		EnvironmentData: map[string]interface{}{
			"build_tool": fd.detectBuildTool(pattern.Command),
			"language":   fd.detectLanguage(pattern.Command),
			"framework":  fd.detectFramework(pattern.Command),
		},
		UserBehavior: map[string]interface{}{
			"automation_candidate": true,
			"frequency_score":      float64(pattern.Frequency) / 10.0,
			"efficiency_impact":    "high",
		},
	}
}

// createFrictionFromCommand creates a generic friction point from command analysis
func (fd *FrictionDetector) createFrictionFromCommand(activity models.ActivityLog, pattern *CommandPattern) *FrictionPoint {
	return &FrictionPoint{
		ID:          uuid.New(),
		Type:        "command_repetition",
		Context:     fd.determineContext(pattern.Command),
		Description: fmt.Sprintf("High frequency command usage: %s", pattern.Command),
		Frequency:   pattern.Frequency,
		Impact:      fd.calculateImpact(pattern),
		FirstSeen:   activity.CreatedAt,
		LastSeen:    activity.CreatedAt,
		UserID:      *activity.UserID,
		
		CommandPatterns: []string{pattern.Command},
		EnvironmentData: map[string]interface{}{
			"command_type": fd.classifyCommand(pattern.Command),
		},
		UserBehavior: map[string]interface{}{
			"usage_pattern": "repetitive",
			"frequency":     pattern.Frequency,
		},
	}
}

// Helper methods for analysis

func (fd *FrictionDetector) calculateImpact(pattern *CommandPattern) string {
	score := float64(pattern.Frequency) * 0.3 + pattern.UserFrustration * 0.7
	
	if score >= 0.8 {
		return "critical"
	} else if score >= 0.6 {
		return "high"
	} else if score >= 0.4 {
		return "medium"
	}
	return "low"
}

func (fd *FrictionDetector) detectOS(command string) string {
	if strings.Contains(command, "pbpaste") || strings.Contains(command, "pbcopy") {
		return "macos"
	} else if strings.Contains(command, "xclip") {
		return "linux"
	} else if strings.Contains(command, "clip") {
		return "windows"
	}
	return "unknown"
}

func (fd *FrictionDetector) detectClipboardTool(command string) string {
	if strings.Contains(command, "pbpaste") {
		return "pbpaste"
	} else if strings.Contains(command, "xclip") {
		return "xclip"
	}
	return "unknown"
}

func (fd *FrictionDetector) detectBuildTool(command string) string {
	if strings.Contains(command, "tsc") {
		return "typescript"
	} else if strings.Contains(command, "eslint") {
		return "eslint"
	} else if strings.Contains(command, "npm") {
		return "npm"
	} else if strings.Contains(command, "yarn") {
		return "yarn"
	} else if strings.Contains(command, "go") {
		return "go"
	} else if strings.Contains(command, "cargo") {
		return "cargo"
	}
	return "unknown"
}

func (fd *FrictionDetector) detectLanguage(command string) string {
	if strings.Contains(command, "tsc") || strings.Contains(command, "typescript") {
		return "typescript"
	} else if strings.Contains(command, "eslint") || strings.Contains(command, "npm") {
		return "javascript"
	} else if strings.Contains(command, "go") {
		return "go"
	} else if strings.Contains(command, "cargo") || strings.Contains(command, "rust") {
		return "rust"
	} else if strings.Contains(command, "pytest") || strings.Contains(command, "python") {
		return "python"
	}
	return "unknown"
}

func (fd *FrictionDetector) detectFramework(command string) string {
	if strings.Contains(command, "jest") {
		return "jest"
	} else if strings.Contains(command, "react") {
		return "react"
	} else if strings.Contains(command, "vue") {
		return "vue"
	} else if strings.Contains(command, "angular") {
		return "angular"
	}
	return "unknown"
}

func (fd *FrictionDetector) determineContext(command string) string {
	if strings.Contains(command, "git") {
		return "version_control"
	} else if strings.Contains(command, "npm") || strings.Contains(command, "yarn") {
		return "package_management"
	} else if strings.Contains(command, "test") {
		return "testing"
	} else if strings.Contains(command, "build") || strings.Contains(command, "compile") {
		return "build_process"
	}
	return "general"
}

func (fd *FrictionDetector) classifyCommand(command string) string {
	if regexp.MustCompile(`^(ls|cd|pwd|find)`).MatchString(command) {
		return "file_system"
	} else if regexp.MustCompile(`^(git|svn|hg)`).MatchString(command) {
		return "version_control"
	} else if regexp.MustCompile(`^(npm|yarn|pip|cargo)`).MatchString(command) {
		return "package_manager"
	} else if regexp.MustCompile(`^(make|gcc|go|rustc|tsc)`).MatchString(command) {
		return "build_tool"
	}
	return "other"
}

// NewCommandAnalyzer creates a command pattern analyzer
func NewCommandAnalyzer(logger *logrus.Logger) *CommandAnalyzer {
	return &CommandAnalyzer{
		logger:       logger,
		commandCache: make(map[string]*CommandPattern),
		sessionCache: make(map[string]*SessionData),
	}
}

// AnalyzeCommand analyzes a command for patterns and friction
func (ca *CommandAnalyzer) AnalyzeCommand(command string, activity models.ActivityLog) *CommandPattern {
	pattern, exists := ca.commandCache[command]
	if !exists {
		pattern = &CommandPattern{
			Command:         command,
			Frequency:       0,
			Context:         "",
			Arguments:       []string{},
			Timing:          []time.Duration{},
			ErrorPatterns:   []string{},
			SuccessPatterns: []string{},
			UserFrustration: 0.0,
		}
		ca.commandCache[command] = pattern
	}

	// Update pattern data
	pattern.Frequency++
	
	// Analyze for frustration indicators
	if !activity.Success {
		pattern.ErrorPatterns = append(pattern.ErrorPatterns, activity.ErrorMessage)
		pattern.UserFrustration = ca.calculateFrustration(pattern)
	} else {
		pattern.SuccessPatterns = append(pattern.SuccessPatterns, "success")
	}

	return pattern
}

// calculateFrustration estimates user frustration based on error patterns
func (ca *CommandAnalyzer) calculateFrustration(pattern *CommandPattern) float64 {
	errorRate := float64(len(pattern.ErrorPatterns)) / float64(pattern.Frequency)
	repetitionFactor := float64(pattern.Frequency) / 10.0
	
	frustration := errorRate * 0.7 + repetitionFactor * 0.3
	if frustration > 1.0 {
		frustration = 1.0
	}
	
	return frustration
}

// NewBehaviorTracker creates a user behavior tracker
func NewBehaviorTracker(logger *logrus.Logger) *BehaviorTracker {
	return &BehaviorTracker{
		logger:            logger,
		frustrationScore:  0.0,
		workflowEfficiency: 1.0,
		toolUsagePatterns: make(map[string]int),
	}
}