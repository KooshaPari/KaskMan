package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kooshapari/kodevibe-go/internal/models"
	"github.com/kooshapari/kodevibe-go/pkg/scanner"
	"github.com/kooshapari/kodevibe-go/pkg/vibes"
	"github.com/sirupsen/logrus"
)

// handleScan performs a code analysis scan
func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// Parse request
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_JSON",
			"Invalid JSON in request body", err.Error())
		return
	}
	
	// Validate request
	if len(req.Paths) == 0 {
		s.errorResponse(w, http.StatusBadRequest, "MISSING_PATHS",
			"At least one path must be specified", "")
		return
	}
	
	// Set defaults
	if req.OutputFormat == "" {
		req.OutputFormat = s.config.Vibes.OutputFormat
	}
	if req.MaxIssues == 0 {
		req.MaxIssues = s.config.Vibes.MaxIssues
	}
	if len(req.Checkers) == 0 {
		req.Checkers = s.config.Vibes.EnabledCheckers
	}
	
	// Create scan context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), s.config.Scanner.ScanTimeout)
	defer cancel()
	
	// Perform scan
	response, err := s.performScan(ctx, &req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			s.errorResponse(w, http.StatusRequestTimeout, "SCAN_TIMEOUT",
				"Scan operation timed out", err.Error())
		} else {
			s.errorResponse(w, http.StatusInternalServerError, "SCAN_ERROR",
				"Failed to perform scan", err.Error())
		}
		return
	}
	
	response.Duration = time.Since(start)
	
	// Create metadata
	meta := &APIMeta{
		Version: s.version,
		Total:   len(response.Issues),
	}
	
	// Add request ID if available
	if requestID := r.Context().Value("request_id"); requestID != nil {
		meta.RequestID = requestID.(string)
	}
	
	s.successResponse(w, response, meta)
}

// handleValidateScan validates a scan request without performing the actual scan
func (s *Server) handleValidateScan(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "INVALID_JSON",
			"Invalid JSON in request body", err.Error())
		return
	}
	
	validation := s.validateScanRequest(&req)
	s.successResponse(w, validation, nil)
}

// performScan executes the actual scan operation
func (s *Server) performScan(ctx context.Context, req *ScanRequest) (*ScanResponse, error) {
	// Create models.Configuration for the scanner
	config := &models.Configuration{
		Scanner: models.ScannerConfig{
			MaxConcurrency: s.config.Scanner.ConcurrentWorkers,
			Timeout:        int(s.config.Scanner.ScanTimeout.Seconds()),
			ExcludePatterns: s.config.Scanner.IgnorePatterns,
			IncludePatterns: s.config.Scanner.IncludePatterns,
			EnabledVibes:    s.config.Vibes.EnabledCheckers,
		},
		Vibes: make(map[models.VibeType]models.VibeConfig),
	}
	
	// Convert config vibes to models format
	for name, checkerConfig := range s.config.Vibes.CheckerConfigs {
		vibeType := models.VibeType(name)
		config.Vibes[vibeType] = models.VibeConfig{
			Enabled:  checkerConfig.Enabled,
			Settings: checkerConfig.Settings,
		}
	}
	
	// Create logger
	logger := logrus.New()
	
	// Initialize scanner
	fileScanner, err := scanner.NewScanner(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create scanner: %w", err)
	}
	
	// Create scan request
	scanRequest := &models.ScanRequest{
		Paths:      req.Paths,
		Vibes:      req.Checkers,
		StagedOnly: false,
		DiffTarget: "",
		Timeout:    int(s.config.Scanner.ScanTimeout.Seconds()),
	}
	
	// Perform scan
	result, err := fileScanner.Scan(ctx, scanRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to scan: %w", err)
	}
	
	// Get all issues
	allIssues := result.Issues
	
	// Filter issues by confidence
	allIssues = s.filterIssuesByConfidence(allIssues, s.config.Vibes.MinConfidence)
	
	// Generate checker stats
	checkerStats := make(map[string]int)
	for _, issue := range allIssues {
		checkerStats[string(issue.Type)]++
	}
	
	// Sort issues by severity and confidence
	sort.Slice(allIssues, func(i, j int) bool {
		if allIssues[i].Severity != allIssues[j].Severity {
			return s.severityWeight(allIssues[i].Severity) > s.severityWeight(allIssues[j].Severity)
		}
		return allIssues[i].Confidence > allIssues[j].Confidence
	})
	
	// Limit issues if requested
	if req.MaxIssues > 0 && len(allIssues) > req.MaxIssues {
		allIssues = allIssues[:req.MaxIssues]
	}
	
	// Generate statistics
	stats := s.generateStatistics(result.FilesScanned, allIssues, checkerStats)
	
	// Generate metadata
	metadata := s.generateMetadata(req.Checkers, req.Options)
	
	return &ScanResponse{
		Issues:     allIssues,
		Statistics: stats,
		Metadata:   metadata,
	}, nil
}

// scanFiles scans for files in the given paths (removed - using scanner directly)

// filterFiles filters files based on request options
func (s *Server) filterFiles(files []string, options map[string]interface{}) []string {
	if options == nil {
		return files
	}
	
	filtered := []string{}
	
	// Filter by file extensions
	if exts, ok := options["file_extensions"].([]interface{}); ok {
		allowedExts := make(map[string]bool)
		for _, ext := range exts {
			if extStr, ok := ext.(string); ok {
				allowedExts[strings.ToLower(extStr)] = true
			}
		}
		
		for _, file := range files {
			ext := strings.ToLower(filepath.Ext(file))
			if allowedExts[ext] {
				filtered = append(filtered, file)
			}
		}
		return filtered
	}
	
	// Filter by patterns
	if patterns, ok := options["include_patterns"].([]interface{}); ok {
		for _, file := range files {
			for _, pattern := range patterns {
				if patternStr, ok := pattern.(string); ok {
					if matched, _ := filepath.Match(patternStr, filepath.Base(file)); matched {
						filtered = append(filtered, file)
						break
					}
				}
			}
		}
		return filtered
	}
	
	return files
}

// getSelectedCheckers returns the checkers requested for the scan
func (s *Server) getSelectedCheckers(checkerNames []string) ([]vibes.Checker, error) {
	checkers := []vibes.Checker{}
	
	for _, name := range checkerNames {
		checker, err := s.registry.GetChecker(models.VibeType(name))
		if err != nil {
			return nil, fmt.Errorf("checker '%s' not found: %w", name, err)
		}
		checkers = append(checkers, checker)
	}
	
	return checkers, nil
}

// configureChecker configures a checker with options
func (s *Server) configureChecker(checker vibes.Checker, options map[string]interface{}) error {
	// Get global configuration for the checker
	var config models.VibeConfig
	if checkerConfig, exists := s.config.Vibes.CheckerConfigs[checker.Name()]; exists {
		config = models.VibeConfig{
			Enabled:  checkerConfig.Enabled,
			Settings: checkerConfig.Settings,
		}
	} else {
		config = models.VibeConfig{
			Enabled:  true,
			Settings: make(map[string]interface{}),
		}
	}
	
	// Override with request-specific options
	if options != nil {
		for key, value := range options {
			config.Settings[key] = value
		}
	}
	
	// Configure the checker
	if configurable, ok := checker.(interface{ Configure(models.VibeConfig) error }); ok {
		return configurable.Configure(config)
	}
	
	return nil
}

// filterIssuesByConfidence filters issues by minimum confidence level
func (s *Server) filterIssuesByConfidence(issues []models.Issue, minConfidence float64) []models.Issue {
	filtered := []models.Issue{}
	for _, issue := range issues {
		if issue.Confidence >= minConfidence {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// severityWeight returns a numeric weight for severity levels for sorting
func (s *Server) severityWeight(severity models.SeverityLevel) int {
	switch severity {
	case models.SeverityCritical:
		return 4
	case models.SeverityError:
		return 3
	case models.SeverityWarning:
		return 2
	case models.SeverityInfo:
		return 1
	default:
		return 0
	}
}

// generateStatistics generates scan statistics
func (s *Server) generateStatistics(filesScanned int, issues []models.Issue, checkerStats map[string]int) ScanStatistics {
	stats := ScanStatistics{
		TotalFiles:       filesScanned,
		ScannedFiles:     filesScanned,
		TotalIssues:      len(issues),
		IssuesBySeverity: make(map[string]int),
		IssuesByType:     make(map[string]int),
		IssuesByChecker:  checkerStats,
	}
	
	// Count by severity
	for _, issue := range issues {
		stats.IssuesBySeverity[string(issue.Severity)]++
		stats.IssuesByType[string(issue.Type)]++
	}
	
	return stats
}

// generateMetadata generates scan metadata
func (s *Server) generateMetadata(checkerNames []string, options map[string]interface{}) ScanMetadata {
	checkerInfos := make([]CheckerInfo, 0, len(checkerNames))
	
	for _, name := range checkerNames {
		checker, err := s.registry.GetChecker(models.VibeType(name))
		if err != nil {
			continue // Skip invalid checkers
		}
		
		info := CheckerInfo{
			Name:    checker.Name(),
			Type:    string(checker.Type()),
			Enabled: true,
		}
		
		if desc, ok := checker.(interface{ Description() string }); ok {
			info.Description = desc.Description()
		}
		
		if ver, ok := checker.(interface{ Version() string }); ok {
			info.Version = ver.Version()
		}
		
		checkerInfos = append(checkerInfos, info)
	}
	
	return ScanMetadata{
		Timestamp:     time.Now(),
		Version:       s.version,
		Checkers:      checkerInfos,
		Configuration: options,
	}
}

// validateScanRequest validates a scan request
func (s *Server) validateScanRequest(req *ScanRequest) map[string]interface{} {
	validation := map[string]interface{}{
		"valid":  true,
		"errors": []string{},
		"warnings": []string{},
		"info": map[string]interface{}{
			"estimated_files": 0,
			"estimated_duration": "unknown",
			"available_checkers": len(s.registry.GetAllCheckers()),
		},
	}
	
	errors := []string{}
	warnings := []string{}
	
	// Validate paths
	if len(req.Paths) == 0 {
		errors = append(errors, "At least one path must be specified")
	}
	
	// Validate checkers
	if len(req.Checkers) > 0 {
		for _, name := range req.Checkers {
			if _, err := s.registry.GetChecker(models.VibeType(name)); err != nil {
				errors = append(errors, fmt.Sprintf("Checker '%s' not found", name))
			}
		}
	}
	
	// Validate max issues
	if req.MaxIssues < 0 {
		errors = append(errors, "Max issues cannot be negative")
	}
	
	// Validate output format
	validFormats := map[string]bool{
		"json": true, "yaml": true, "xml": true, "csv": true, "table": true,
	}
	if req.OutputFormat != "" && !validFormats[req.OutputFormat] {
		errors = append(errors, fmt.Sprintf("Invalid output format: %s", req.OutputFormat))
	}
	
	// Check for potential performance issues
	if req.MaxIssues > 10000 {
		warnings = append(warnings, "Large number of max issues may impact performance")
	}
	
	if len(req.Checkers) == 0 {
		warnings = append(warnings, "No checkers specified, will use default checkers")
	}
	
	validation["valid"] = len(errors) == 0
	validation["errors"] = errors
	validation["warnings"] = warnings
	
	return validation
}