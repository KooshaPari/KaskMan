// Package models provides data structures for the KodeVibe code analysis tool.
package models

import (
	"time"
)

// SeverityLevel represents the severity of an issue
type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "info"
	SeverityWarning  SeverityLevel = "warning"
	SeverityError    SeverityLevel = "error"
	SeverityCritical SeverityLevel = "critical"
	SeverityHint     SeverityLevel = "hint"
)

func (s SeverityLevel) String() string {
	return string(s)
}

// VibeType represents different types of code analysis
type VibeType string

const (
	VibeTypeSecurity      VibeType = "security"
	VibeTypeCode          VibeType = "code"
	VibeTypePerformance   VibeType = "performance"
	VibeTypeFile          VibeType = "file"
	VibeTypeGit           VibeType = "git"
	VibeTypeDependency    VibeType = "dependency"
	VibeTypeDocumentation VibeType = "documentation"
)

func (v VibeType) String() string {
	return string(v)
}

// Issue represents a single issue found during scanning
type Issue struct {
	ID          string        `json:"id" yaml:"id"`
	Type        VibeType      `json:"type" yaml:"type"`
	Severity    SeverityLevel `json:"severity" yaml:"severity"`
	Title       string        `json:"title" yaml:"title"`
	Description string        `json:"description" yaml:"description"`
	File        string        `json:"file" yaml:"file"`
	Line        int           `json:"line" yaml:"line"`
	Column      int           `json:"column" yaml:"column"`
	Rule        string        `json:"rule" yaml:"rule"`
	Message     string        `json:"message" yaml:"message"`
	Confidence  float64       `json:"confidence" yaml:"confidence"`
	FixSuggestion string      `json:"fix_suggestion,omitempty" yaml:"fix_suggestion,omitempty"`
	Reference   string        `json:"reference,omitempty" yaml:"reference,omitempty"`
	Timestamp   time.Time     `json:"timestamp" yaml:"timestamp"`
}

// ScanResult represents the complete result of a scan
type ScanResult struct {
	ScanID        string                 `json:"scan_id" yaml:"scan_id"`
	ID            string                 `json:"id" yaml:"id"`
	StartTime     time.Time              `json:"start_time" yaml:"start_time"`
	EndTime       time.Time              `json:"end_time" yaml:"end_time"`
	Duration      time.Duration          `json:"duration" yaml:"duration"`
	Timestamp     time.Time              `json:"timestamp" yaml:"timestamp"`
	ProjectPath   string                 `json:"project_path" yaml:"project_path"`
	FilesScanned  int                    `json:"files_scanned" yaml:"files_scanned"`
	FilesSkipped  int                    `json:"files_skipped" yaml:"files_skipped"`
	Issues        []Issue                `json:"issues" yaml:"issues"`
	Summary       ScanSummary            `json:"summary" yaml:"summary"`
	Configuration *Configuration         `json:"configuration,omitempty" yaml:"configuration,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// ScanSummary provides summary statistics
type ScanSummary struct {
	TotalIssues      int                `json:"total_issues" yaml:"total_issues"`
	IssuesByType     map[VibeType]int   `json:"issues_by_type" yaml:"issues_by_type"`
	IssuesBySeverity map[string]int     `json:"issues_by_severity" yaml:"issues_by_severity"`
	FilesScanned     int                `json:"files_scanned" yaml:"files_scanned"`
	CriticalIssues   int                `json:"critical_issues" yaml:"critical_issues"`
	ErrorIssues      int                `json:"error_issues" yaml:"error_issues"`
	WarningIssues    int                `json:"warning_issues" yaml:"warning_issues"`
	InfoIssues       int                `json:"info_issues" yaml:"info_issues"`
	Score            float64            `json:"score" yaml:"score"`
	Grade            string             `json:"grade" yaml:"grade"`
}

// ScanRequest represents a request to scan files
type ScanRequest struct {
	ID         string     `json:"id" yaml:"id"`
	Paths      []string   `json:"paths" yaml:"paths"`
	Vibes      []string   `json:"vibes" yaml:"vibes"`
	StagedOnly bool       `json:"staged_only" yaml:"staged_only"`
	DiffTarget string     `json:"diff_target,omitempty" yaml:"diff_target,omitempty"`
	Timeout    int        `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// Configuration represents the tool configuration
type Configuration struct {
	Scanner  ScannerConfig           `json:"scanner" yaml:"scanner"`
	Vibes    map[VibeType]VibeConfig `json:"vibes" yaml:"vibes"`
	Server   ServerConfig            `json:"server" yaml:"server"`
	Advanced AdvancedConfig          `json:"advanced" yaml:"advanced"`
}

// ScannerConfig represents scanner configuration
type ScannerConfig struct {
	MaxConcurrency  int      `json:"max_concurrency" yaml:"max_concurrency"`
	Timeout         int      `json:"timeout" yaml:"timeout"`
	EnabledVibes    []string `json:"enabled_vibes" yaml:"enabled_vibes"`
	ExcludePatterns []string `json:"exclude_patterns" yaml:"exclude_patterns"`
	IncludePatterns []string `json:"include_patterns" yaml:"include_patterns"`
}

// VibeConfig represents configuration for a specific vibe
type VibeConfig struct {
	Enabled  bool                   `json:"enabled" yaml:"enabled"`
	Settings map[string]interface{} `json:"settings" yaml:"settings"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host string `json:"host" yaml:"host"`
	Port int    `json:"port" yaml:"port"`
}

// AdvancedConfig represents advanced configuration options
type AdvancedConfig struct {
	MaxConcurrency int           `json:"max_concurrency" yaml:"max_concurrency"`
	CacheEnabled   bool          `json:"cache_enabled" yaml:"cache_enabled"`
	CacheTTL       time.Duration `json:"cache_ttl" yaml:"cache_ttl"`
	LogLevel       string        `json:"log_level" yaml:"log_level"`
}

// ReportFormat represents output format types
type ReportFormat string

const (
	ReportFormatJSON ReportFormat = "json"
	ReportFormatText ReportFormat = "text"
	ReportFormatHTML ReportFormat = "html"
	ReportFormatXML  ReportFormat = "xml"
	ReportFormatCSV  ReportFormat = "csv"
)

func (r ReportFormat) String() string {
	return string(r)
}

// IsValid validates an Issue
func (i *Issue) IsValid() bool {
	return i.Title != "" && i.Line > 0 && i.Confidence >= 0 && i.Confidence <= 1
}

// IsValid validates a ScanRequest
func (sr *ScanRequest) IsValid() bool {
	if len(sr.Paths) == 0 || len(sr.Vibes) == 0 {
		return false
	}
	for _, path := range sr.Paths {
		if path == "" {
			return false
		}
	}
	for _, vibe := range sr.Vibes {
		if vibe == "" {
			return false
		}
	}
	return true
}

// CalculateSummary calculates summary statistics for scan results
func (sr *ScanResult) CalculateSummary() ScanSummary {
	summary := ScanSummary{
		TotalIssues:      len(sr.Issues),
		IssuesByType:     make(map[VibeType]int),
		IssuesBySeverity: make(map[string]int),
		FilesScanned:     0,
	}

	filesMap := make(map[string]bool)

	for _, issue := range sr.Issues {
		// Count by severity
		switch issue.Severity {
		case SeverityCritical:
			summary.CriticalIssues++
		case SeverityError:
			summary.ErrorIssues++
		case SeverityWarning:
			summary.WarningIssues++
		case SeverityInfo:
			summary.InfoIssues++
		}

		// Count by type
		summary.IssuesByType[issue.Type]++
		summary.IssuesBySeverity[string(issue.Severity)]++

		// Track unique files
		if issue.File != "" {
			filesMap[issue.File] = true
		}
	}

	summary.FilesScanned = len(filesMap)
	
	// Calculate score and grade
	summary.Score = calculateScore(summary.TotalIssues, summary.CriticalIssues, summary.ErrorIssues, summary.WarningIssues)
	summary.Grade = getGrade(summary.Score)

	return summary
}

// IsValid validates the configuration
func (c *Configuration) IsValid() bool {
	return c.Scanner.MaxConcurrency > 0 && c.Scanner.Timeout > 0 && c.Server.Port > 0 && c.Server.Port < 65536
}

// IsEnabled checks if a vibe is enabled
func (vc *VibeConfig) IsEnabled() bool {
	return vc.Enabled
}

// calculateScore calculates a quality score based on issue counts
func calculateScore(total, critical, errors, warnings int) float64 {
	if total == 0 {
		return 100.0
	}
	
	// Penalty system: critical=30, error=5, warning=1 per issue
	penalty := float64(critical*30 + errors*5 + warnings*1)
	score := 100.0 - penalty
	
	if score < 0 {
		score = 0
	}
	
	return score
}

// getGrade converts a score to a letter grade
func getGrade(score float64) string {
	switch {
	case score >= 97:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}