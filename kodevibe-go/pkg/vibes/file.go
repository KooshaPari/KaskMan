package vibes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kooshapari/kodevibe-go/internal/models"
	"github.com/kooshapari/kodevibe-go/internal/utils"
)

// FileChecker implements file structure and organization checks
type FileChecker struct {
	config                  models.VibeConfig
	enableLargeFiles        bool
	enableDuplicateFiles    bool
	enableFileNaming        bool
	enableFileStructure     bool
	enableUnusedFiles       bool
	enableTempFiles         bool
	enableBinaryFiles       bool
	maxFileSize             int64
	maxDirectoryDepth       int
	allowedFileExtensions   []string
	forbiddenFileExtensions []string
}

// NewFileChecker creates a new file checker
func NewFileChecker() *FileChecker {
	return &FileChecker{
		enableLargeFiles:        true,
		enableDuplicateFiles:    true,
		enableFileNaming:        true,
		enableFileStructure:     true,
		enableUnusedFiles:       true,
		enableTempFiles:         true,
		enableBinaryFiles:       true,
		maxFileSize:             10 * 1024 * 1024, // 10MB
		maxDirectoryDepth:       8,
		allowedFileExtensions:   []string{},
		forbiddenFileExtensions: []string{".exe", ".dll", ".so", ".dylib"},
	}
}

func (fc *FileChecker) Name() string          { return "FileVibe" }
func (fc *FileChecker) Type() models.VibeType { return models.VibeTypeFile }

func (fc *FileChecker) Configure(config models.VibeConfig) error {
	fc.config = config
	
	if val, exists := config.Settings["enable_large_files"]; exists {
		if boolVal, ok := val.(bool); ok {
			fc.enableLargeFiles = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_duplicate_files"]; exists {
		if boolVal, ok := val.(bool); ok {
			fc.enableDuplicateFiles = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_file_naming"]; exists {
		if boolVal, ok := val.(bool); ok {
			fc.enableFileNaming = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_file_structure"]; exists {
		if boolVal, ok := val.(bool); ok {
			fc.enableFileStructure = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_unused_files"]; exists {
		if boolVal, ok := val.(bool); ok {
			fc.enableUnusedFiles = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_temp_files"]; exists {
		if boolVal, ok := val.(bool); ok {
			fc.enableTempFiles = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_binary_files"]; exists {
		if boolVal, ok := val.(bool); ok {
			fc.enableBinaryFiles = boolVal
		}
	}
	
	if val, exists := config.Settings["max_file_size"]; exists {
		if sizeStr, ok := val.(string); ok {
			if size, err := utils.ParseSize(sizeStr); err == nil {
				fc.maxFileSize = size
			}
		} else if intVal, ok := val.(int); ok {
			fc.maxFileSize = int64(intVal)
		} else if int64Val, ok := val.(int64); ok {
			fc.maxFileSize = int64Val
		}
	}
	
	if val, exists := config.Settings["max_directory_depth"]; exists {
		if intVal, ok := val.(int); ok {
			fc.maxDirectoryDepth = intVal
		}
	}
	
	if val, exists := config.Settings["allowed_file_extensions"]; exists {
		if strSlice, ok := val.([]string); ok {
			fc.allowedFileExtensions = strSlice
		}
	}
	
	if val, exists := config.Settings["forbidden_file_extensions"]; exists {
		if strSlice, ok := val.([]string); ok {
			fc.forbiddenFileExtensions = strSlice
		}
	}
	
	return nil
}

func (fc *FileChecker) Supports(filename string) bool {
	// File checker supports all files
	return true
}

func (fc *FileChecker) Check(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Check individual files
	for _, file := range files {
		fileIssues, err := fc.checkFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check file %s: %w", file, err)
		}
		issues = append(issues, fileIssues...)
	}
	
	// Check for cross-file issues
	if fc.enableDuplicateFiles {
		duplicateIssues, err := fc.checkDuplicateFiles(ctx, files)
		if err != nil {
			return nil, fmt.Errorf("failed to check for duplicate files: %w", err)
		}
		issues = append(issues, duplicateIssues...)
	}
	
	// Check file structure
	if fc.enableFileStructure {
		structureIssues := fc.checkFileStructure(files)
		issues = append(issues, structureIssues...)
	}
	
	return issues, nil
}

func (fc *FileChecker) checkFile(ctx context.Context, filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Get file info
	info, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	
	// Check file size
	if fc.enableLargeFiles {
		if info.Size() > fc.maxFileSize {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeFile,
				Severity:    models.SeverityWarning,
				Title:       "Large file detected",
				Description: fmt.Sprintf("File size is %s (max: %s)", utils.FormatSize(info.Size()), utils.FormatSize(fc.maxFileSize)),
				File:        filename,
				Line:        1,
				Rule:        "large-file",
				Message:     "Large files can impact performance and repository size",
				Confidence:  1.0,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check file naming
	if fc.enableFileNaming {
		issues = append(issues, fc.checkFileNaming(filename)...)
	}
	
	// Check for temporary files
	if fc.enableTempFiles {
		issues = append(issues, fc.checkTempFiles(filename)...)
	}
	
	// Check for binary files
	if fc.enableBinaryFiles {
		issues = append(issues, fc.checkBinaryFiles(filename)...)
	}
	
	// Check directory depth
	depth := fc.getDirectoryDepth(filename)
	if depth > fc.maxDirectoryDepth {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeFile,
			Severity:    models.SeverityWarning,
			Title:       "Deep directory nesting",
			Description: fmt.Sprintf("Directory depth is %d (max: %d)", depth, fc.maxDirectoryDepth),
			File:        filename,
			Line:        1,
			Rule:        "directory-depth",
			Message:     "Deep directory nesting can make navigation difficult",
			Confidence:  1.0,
			Timestamp:   time.Now(),
		})
	}
	
	// Check allowed/forbidden extensions
	if len(fc.allowedFileExtensions) > 0 || len(fc.forbiddenFileExtensions) > 0 {
		issues = append(issues, fc.checkFileExtensions(filename)...)
	}
	
	return issues, nil
}

func (fc *FileChecker) checkFileNaming(filename string) []models.Issue {
	var issues []models.Issue
	
	baseName := filepath.Base(filename)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	
	// File naming patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
		severity    models.SeverityLevel
	}{
		{`\s+`, "Spaces in filename", 0.9, models.SeverityWarning},
		{`[A-Z]{2,}`, "Multiple consecutive uppercase letters", 0.7, models.SeverityInfo},
		{`[^\w\-\.]`, "Special characters in filename", 0.8, models.SeverityWarning},
		{`^[0-9]`, "Filename starts with number", 0.6, models.SeverityInfo},
		{`^\.`, "Hidden file", 0.5, models.SeverityInfo},
		{`_{2,}`, "Multiple consecutive underscores", 0.7, models.SeverityInfo},
		{`-{2,}`, "Multiple consecutive hyphens", 0.7, models.SeverityInfo},
		{`\.(tmp|temp|bak|backup|old|orig)$`, "Temporary or backup file extension", 0.9, models.SeverityWarning},
		{`\.(log|logs)$`, "Log file", 0.6, models.SeverityInfo},
		{`\.(cache|tmp)$`, "Cache or temporary file", 0.8, models.SeverityWarning},
		{`copy`, "File name contains 'copy'", 0.7, models.SeverityInfo},
		{`untitled`, "Untitled file", 0.8, models.SeverityWarning},
		{`test.*test`, "Multiple 'test' in filename", 0.6, models.SeverityInfo},
		{`new.*file`, "Generic 'new file' name", 0.8, models.SeverityWarning},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(nameWithoutExt) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeFile,
				Severity:    p.severity,
				Title:       "File naming issue",
				Description: p.description,
				File:        filename,
				Line:        1,
				Rule:        "file-naming",
				Message:     "Consider using consistent file naming conventions",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check for very long filenames
	if len(baseName) > 100 {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeFile,
			Severity:    models.SeverityWarning,
			Title:       "Very long filename",
			Description: fmt.Sprintf("Filename is %d characters long", len(baseName)),
			File:        filename,
			Line:        1,
			Rule:        "long-filename",
			Message:     "Long filenames can cause issues on some systems",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for very short filenames
	if len(nameWithoutExt) < 2 {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeFile,
			Severity:    models.SeverityInfo,
			Title:       "Very short filename",
			Description: fmt.Sprintf("Filename is only %d characters", len(nameWithoutExt)),
			File:        filename,
			Line:        1,
			Rule:        "short-filename",
			Message:     "Short filenames may not be descriptive enough",
			Confidence:  0.6,
			Timestamp:   time.Now(),
		})
	}
	
	return issues
}

func (fc *FileChecker) checkTempFiles(filename string) []models.Issue {
	var issues []models.Issue
	
	baseName := filepath.Base(filename)
	
	// Temporary file patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`~$`, "Temporary file ending with ~", 0.9},
		{`\.tmp$`, "Temporary file with .tmp extension", 0.9},
		{`\.temp$`, "Temporary file with .temp extension", 0.9},
		{`\.swp$`, "Vim swap file", 0.9},
		{`\.swo$`, "Vim swap file", 0.9},
		{`\.bak$`, "Backup file", 0.8},
		{`\.backup$`, "Backup file", 0.8},
		{`\.old$`, "Old file", 0.8},
		{`\.orig$`, "Original file", 0.8},
		{`^\.#`, "Emacs temporary file", 0.9},
		{`^#.*#$`, "Emacs auto-save file", 0.9},
		{`\.DS_Store$`, "macOS metadata file", 0.9},
		{`Thumbs\.db$`, "Windows thumbnail cache", 0.9},
		{`\.lockfile$`, "Lock file", 0.8},
		{`\.pid$`, "Process ID file", 0.8},
		{`\.cache$`, "Cache file", 0.7},
		{`\.log$`, "Log file", 0.6},
		{`core\.\d+$`, "Core dump file", 0.9},
		{`\.dump$`, "Dump file", 0.8},
		{`\.stackdump$`, "Stack dump file", 0.9},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(baseName) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeFile,
				Severity:    models.SeverityWarning,
				Title:       "Temporary file detected",
				Description: p.description,
				File:        filename,
				Line:        1,
				Rule:        "temp-file",
				Message:     "Temporary files should not be committed to version control",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (fc *FileChecker) checkBinaryFiles(filename string) []models.Issue {
	var issues []models.Issue
	
	baseName := filepath.Base(filename)
	
	// Binary file patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`\.exe$`, "Windows executable", 0.9},
		{`\.dll$`, "Windows dynamic library", 0.9},
		{`\.so$`, "Unix shared object", 0.9},
		{`\.dylib$`, "macOS dynamic library", 0.9},
		{`\.bin$`, "Binary file", 0.8},
		{`\.obj$`, "Object file", 0.8},
		{`\.lib$`, "Library file", 0.8},
		{`\.class$`, "Java class file", 0.8},
		{`\.jar$`, "Java archive", 0.7},
		{`\.war$`, "Web application archive", 0.7},
		{`\.ear$`, "Enterprise archive", 0.7},
		{`\.pyc$`, "Python compiled file", 0.8},
		{`\.pyo$`, "Python optimized file", 0.8},
		{`\.pyd$`, "Python extension", 0.8},
		{`\.o$`, "Object file", 0.8},
		{`\.a$`, "Archive file", 0.7},
		{`\.out$`, "Output file", 0.7},
		{`\.dSYM$`, "Debug symbols", 0.8},
		{`\.pdb$`, "Program database", 0.8},
		{`\.idb$`, "Intermediate database", 0.8},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(baseName) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeFile,
				Severity:    models.SeverityWarning,
				Title:       "Binary file detected",
				Description: p.description,
				File:        filename,
				Line:        1,
				Rule:        "binary-file",
				Message:     "Binary files should generally not be committed to version control",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (fc *FileChecker) checkFileExtensions(filename string) []models.Issue {
	var issues []models.Issue
	
	ext := strings.ToLower(filepath.Ext(filename))
	
	// Check forbidden extensions
	for _, forbiddenExt := range fc.forbiddenFileExtensions {
		if ext == strings.ToLower(forbiddenExt) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeFile,
				Severity:    models.SeverityError,
				Title:       "Forbidden file extension",
				Description: fmt.Sprintf("File extension %s is not allowed", ext),
				File:        filename,
				Line:        1,
				Rule:        "forbidden-extension",
				Message:     "This file type is not allowed in this project",
				Confidence:  1.0,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check allowed extensions (if specified)
	if len(fc.allowedFileExtensions) > 0 {
		allowed := false
		for _, allowedExt := range fc.allowedFileExtensions {
			if ext == strings.ToLower(allowedExt) {
				allowed = true
				break
			}
		}
		
		if !allowed {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeFile,
				Severity:    models.SeverityWarning,
				Title:       "Unallowed file extension",
				Description: fmt.Sprintf("File extension %s is not in the allowed list", ext),
				File:        filename,
				Line:        1,
				Rule:        "unallowed-extension",
				Message:     "This file type is not in the allowed extensions list",
				Confidence:  1.0,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (fc *FileChecker) checkDuplicateFiles(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Map to track file hashes
	hashToFiles := make(map[string][]string)
	
	// Calculate hash for each file
	for _, file := range files {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		// Skip directories
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			continue
		}
		
		// Read file content and calculate hash
		content, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}
		
		hash := utils.Hash(string(content))
		hashToFiles[hash] = append(hashToFiles[hash], file)
	}
	
	// Find duplicates
	for hash, duplicateFiles := range hashToFiles {
		if len(duplicateFiles) > 1 {
			for _, file := range duplicateFiles {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeFile,
					Severity:    models.SeverityWarning,
					Title:       "Duplicate file content",
					Description: fmt.Sprintf("File has identical content to %d other files", len(duplicateFiles)-1),
					File:        file,
					Line:        1,
					Rule:        "duplicate-file",
					Message:     fmt.Sprintf("Duplicate files: %s", strings.Join(duplicateFiles, ", ")),
					Confidence:  1.0,
					Timestamp:   time.Now(),
					Reference:   hash,
				})
			}
		}
	}
	
	return issues, nil
}

func (fc *FileChecker) checkFileStructure(files []string) []models.Issue {
	var issues []models.Issue
	
	// Track directory structure
	directories := make(map[string]bool)
	filesByDir := make(map[string][]string)
	
	for _, file := range files {
		dir := filepath.Dir(file)
		directories[dir] = true
		filesByDir[dir] = append(filesByDir[dir], file)
	}
	
	// Check for common structure issues
	for dir, dirFiles := range filesByDir {
		// Check for too many files in a single directory
		if len(dirFiles) > 50 {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeFile,
				Severity:    models.SeverityWarning,
				Title:       "Too many files in directory",
				Description: fmt.Sprintf("Directory contains %d files", len(dirFiles)),
				File:        dir,
				Line:        1,
				Rule:        "directory-file-count",
				Message:     "Consider organizing files into subdirectories",
				Confidence:  0.8,
				Timestamp:   time.Now(),
			})
		}
		
		// Check for mixed file types in directory
		extensions := make(map[string]int)
		for _, file := range dirFiles {
			ext := strings.ToLower(filepath.Ext(file))
			if ext != "" {
				extensions[ext]++
			}
		}
		
		if len(extensions) > 5 {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeFile,
				Severity:    models.SeverityInfo,
				Title:       "Mixed file types in directory",
				Description: fmt.Sprintf("Directory contains %d different file types", len(extensions)),
				File:        dir,
				Line:        1,
				Rule:        "mixed-file-types",
				Message:     "Consider organizing files by type",
				Confidence:  0.6,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (fc *FileChecker) getDirectoryDepth(filename string) int {
	depth := 0
	dir := filepath.Dir(filename)
	
	for dir != "." && dir != "/" && dir != filepath.Dir(dir) {
		depth++
		dir = filepath.Dir(dir)
	}
	
	return depth
}