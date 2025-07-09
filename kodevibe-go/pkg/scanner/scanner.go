// Package scanner provides file scanning and analysis functionality.
package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kooshapari/kodevibe-go/internal/models"
	"github.com/kooshapari/kodevibe-go/internal/utils"
	"github.com/kooshapari/kodevibe-go/pkg/vibes"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Scanner performs code analysis across multiple vibes
type Scanner struct {
	config         *models.Configuration
	vibeRegistry   *vibes.Registry
	logger         *logrus.Logger
	cache          *utils.Cache
	metrics        *utils.Metrics
	maxConcurrency int
	timeout        time.Duration
	vibes          []string
}

// NewScanner creates a new scanner instance
func NewScanner(config *models.Configuration, logger *logrus.Logger) (*Scanner, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	// Create vibe registry and register all vibes
	registry := vibes.NewRegistry()
	if err := registry.RegisterAllVibes(config); err != nil {
		return nil, fmt.Errorf("failed to register vibes: %w", err)
	}

	// Initialize cache if enabled
	var cache *utils.Cache
	if config.Advanced.CacheEnabled {
		cache = utils.NewCache(config.Advanced.CacheTTL)
	}

	// Initialize metrics
	metrics := utils.NewMetrics()

	return &Scanner{
		config:         config,
		vibeRegistry:   registry,
		logger:         logger,
		cache:          cache,
		metrics:        metrics,
		maxConcurrency: config.Scanner.MaxConcurrency,
		timeout:        time.Duration(config.Scanner.Timeout) * time.Second,
		vibes:          config.Scanner.EnabledVibes,
	}, nil
}

// Scan performs a comprehensive scan of the specified paths
func (s *Scanner) Scan(ctx context.Context, request *models.ScanRequest) (*models.ScanResult, error) {
	if request == nil {
		return nil, fmt.Errorf("scan request is required")
	}

	startTime := time.Now()
	scanID := request.ID
	if scanID == "" {
		scanID = uuid.New().String()
	}

	s.logger.WithFields(logrus.Fields{
		"scan_id": scanID,
		"paths":   request.Paths,
		"vibes":   request.Vibes,
	}).Info("Starting scan")

	// Initialize scan result
	result := &models.ScanResult{
		ScanID:        scanID,
		ID:            scanID,
		StartTime:     startTime,
		ProjectPath:   strings.Join(request.Paths, ","),
		Configuration: s.config,
		Issues:        []models.Issue{},
		Metadata:      make(map[string]interface{}),
	}

	// Discover files to scan
	files, err := s.discoverFiles(request.Paths, request.StagedOnly, request.DiffTarget)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	// Filter files based on exclusion patterns
	filteredFiles := s.filterFiles(files)
	result.FilesScanned = len(filteredFiles)
	result.FilesSkipped = len(files) - len(filteredFiles)

	s.logger.WithFields(logrus.Fields{
		"scan_id":       scanID,
		"files_found":   len(files),
		"files_scanned": len(filteredFiles),
		"files_skipped": result.FilesSkipped,
	}).Info("File discovery completed")

	if len(filteredFiles) == 0 {
		s.logger.WithField("scan_id", scanID).Warn("No files to scan")
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		result.Summary = result.CalculateSummary()
		return result, nil
	}

	// Determine which vibes to run
	vibesToRun := s.getVibesToRun(request.Vibes)
	if len(vibesToRun) == 0 {
		return nil, fmt.Errorf("no valid vibes specified")
	}

	// Set timeout context
	timeoutCtx := ctx
	if request.Timeout > 0 {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, time.Duration(request.Timeout)*time.Second)
		defer cancel()
	}

	// Run vibe checks
	issues, err := s.runVibeChecks(timeoutCtx, filteredFiles, vibesToRun)
	if err != nil {
		return nil, fmt.Errorf("failed to run vibe checks: %w", err)
	}

	// Finalize result
	result.Issues = issues
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Summary = result.CalculateSummary()

	// Record metrics
	s.metrics.RecordScan(result)

	s.logger.WithFields(logrus.Fields{
		"scan_id":     scanID,
		"duration":    result.Duration,
		"issues":      len(result.Issues),
		"files":       result.FilesScanned,
		"score":       result.Summary.Score,
		"grade":       result.Summary.Grade,
	}).Info("Scan completed")

	return result, nil
}

// discoverFiles discovers all files to be scanned
func (s *Scanner) discoverFiles(paths []string, stagedOnly bool, diffTarget string) ([]string, error) {
	var allFiles []string

	for _, path := range paths {
		files, err := s.discoverFilesInPath(path, stagedOnly, diffTarget)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"path":  path,
				"error": err.Error(),
			}).Warn("Failed to discover files in path, skipping")
			continue
		}
		allFiles = append(allFiles, files...)
	}

	// Remove duplicates
	fileSet := make(map[string]bool)
	var uniqueFiles []string
	for _, file := range allFiles {
		if !fileSet[file] {
			fileSet[file] = true
			uniqueFiles = append(uniqueFiles, file)
		}
	}

	return uniqueFiles, nil
}

// discoverFilesInPath discovers files in a specific path
func (s *Scanner) discoverFilesInPath(path string, stagedOnly bool, diffTarget string) ([]string, error) {
	var files []string

	// Handle git-specific file discovery
	if stagedOnly || diffTarget != "" {
		return s.discoverGitFiles(path, stagedOnly, diffTarget)
	}

	// Walk the directory tree
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Check if file should be ignored based on patterns
		if s.shouldIgnore(filePath) {
			return nil
		}

		// Add file to list
		files = append(files, filePath)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return files, nil
}

// discoverGitFiles discovers git-specific files (staged or diff)
func (s *Scanner) discoverGitFiles(path string, stagedOnly bool, diffTarget string) ([]string, error) {
	gitUtil := utils.NewGitUtil(path)

	if stagedOnly {
		return gitUtil.GetStagedFiles()
	}

	if diffTarget != "" {
		return gitUtil.GetDiffFiles(diffTarget)
	}

	return nil, fmt.Errorf("invalid git file discovery parameters")
}

// filterFiles applies include/exclude patterns to filter files
func (s *Scanner) filterFiles(files []string) []string {
	var filteredFiles []string

	for _, file := range files {
		// Apply include patterns if specified
		if len(s.config.Scanner.IncludePatterns) > 0 {
			included := false
			for _, pattern := range s.config.Scanner.IncludePatterns {
				if matched, _ := filepath.Match(pattern, filepath.Base(file)); matched {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		// Skip files that match exclude patterns
		if s.shouldIgnore(file) {
			continue
		}

		filteredFiles = append(filteredFiles, file)
	}

	return filteredFiles
}

// getVibesToRun determines which vibes should be executed
func (s *Scanner) getVibesToRun(requestedVibes []string) []models.VibeType {
	var vibesToRun []models.VibeType

	// If no specific vibes requested, use all enabled vibes
	if len(requestedVibes) == 0 {
		requestedVibes = s.config.Scanner.EnabledVibes
	}

	// Convert strings to VibeType and validate
	for _, vibeStr := range requestedVibes {
		vibeType := models.VibeType(vibeStr)
		
		// Check if vibe is registered
		if _, err := s.vibeRegistry.GetChecker(vibeType); err != nil {
			s.logger.WithFields(logrus.Fields{
				"vibe":  vibeStr,
				"error": err.Error(),
			}).Warn("Vibe not available, skipping")
			continue
		}

		// Check if vibe is enabled in configuration
		if vibeConfig, exists := s.config.Vibes[vibeType]; exists && vibeConfig.Enabled {
			vibesToRun = append(vibesToRun, vibeType)
		}
	}

	return vibesToRun
}

// runVibeChecks executes all vibe checks concurrently
func (s *Scanner) runVibeChecks(ctx context.Context, files []string, vibesToRun []models.VibeType) ([]models.Issue, error) {
	var allIssues []models.Issue
	var mu sync.Mutex

	// Create semaphore for concurrency control
	sem := semaphore.NewWeighted(int64(s.maxConcurrency))

	// Create error group for concurrent execution
	var wg sync.WaitGroup
	errChan := make(chan error, len(vibesToRun))

	// Run each vibe check
	for _, vibeType := range vibesToRun {
		wg.Add(1)
		go func(vType models.VibeType) {
			defer wg.Done()

			// Acquire semaphore
			if err := sem.Acquire(ctx, 1); err != nil {
				errChan <- fmt.Errorf("failed to acquire semaphore: %w", err)
				return
			}
			defer sem.Release(1)

			// Get vibe checker
			checker, err := s.vibeRegistry.GetChecker(vType)
			if err != nil {
				errChan <- fmt.Errorf("failed to get checker for vibe %s: %w", vType, err)
				return
			}

			// Run vibe check
			issues, err := s.runSingleVibeCheck(ctx, checker, files, vType)
			if err != nil {
				errChan <- fmt.Errorf("failed to run vibe check %s: %w", vType, err)
				return
			}

			// Add issues to result
			mu.Lock()
			allIssues = append(allIssues, issues...)
			mu.Unlock()

			s.logger.WithFields(logrus.Fields{
				"vibe":   vType,
				"issues": len(issues),
				"files":  len(files),
			}).Debug("Vibe check completed")
		}(vibeType)
	}
	// Wait for all checks to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return allIssues, nil
}

// runSingleVibeCheck executes a single vibe check
func (s *Scanner) runSingleVibeCheck(ctx context.Context, checker vibes.Checker, files []string, vibeType models.VibeType) ([]models.Issue, error) {
	startTime := time.Now()
	
	// Filter files that this checker supports
	supportedFiles := make([]string, 0, len(files))
	for _, file := range files {
		if checker.Supports(file) {
			supportedFiles = append(supportedFiles, file)
		}
	}

	if len(supportedFiles) == 0 {
		s.logger.WithField("vibe", vibeType).Debug("No supported files for vibe")
		return []models.Issue{}, nil
	}

	// Check cache if enabled
	var cacheKey string
	if s.cache != nil {
		cacheKey = s.generateCacheKey(vibeType, supportedFiles)
		if cached, found := s.cache.Get(cacheKey); found {
			if issues, ok := cached.([]models.Issue); ok {
				s.logger.WithField("vibe", vibeType).Debug("Using cached result")
				return issues, nil
			}
		}
	}

	// Run the actual check
	issues, err := checker.Check(ctx, supportedFiles)
	if err != nil {
		return nil, fmt.Errorf("vibe check failed: %w", err)
	}

	// Cache the result
	if s.cache != nil && cacheKey != "" {
		s.cache.Set(cacheKey, issues)
	}

	// Record metrics
	duration := time.Since(startTime)
	s.metrics.RecordVibeCheck(vibeType, duration, len(issues))

	return issues, nil
}

// generateCacheKey generates a cache key for the given vibe and files
func (s *Scanner) generateCacheKey(vibeType models.VibeType, files []string) string {
	return fmt.Sprintf("%s:%s", vibeType, utils.HashStrings(files))
}

// shouldIgnore checks if a file should be ignored based on patterns
func (s *Scanner) shouldIgnore(path string) bool {
	for _, pattern := range s.config.Scanner.ExcludePatterns {
		// Check filename pattern (e.g., "*.txt")
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}

		// Handle directory patterns like "node_modules/*", ".git/*"
		if strings.HasSuffix(pattern, "/*") {
			dirPattern := strings.TrimSuffix(pattern, "/*")
			if strings.Contains(path, dirPattern+"/") || strings.HasPrefix(path, dirPattern+"/") {
				return true
			}
		}

		// Check full path pattern
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}
	return false
}