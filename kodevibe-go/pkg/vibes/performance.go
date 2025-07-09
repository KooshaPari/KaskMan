package vibes

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kooshapari/kodevibe-go/internal/models"
)

// PerformanceChecker implements performance analysis checks
type PerformanceChecker struct {
	config                 models.VibeConfig
	enableMemoryLeaks      bool
	enableSlowOperations   bool
	enableLoopOptimization bool
	enableDOMManipulation  bool
	enableDatabaseQueries  bool
	enableFileOperations   bool
	enableNetworkCalls     bool
}

// NewPerformanceChecker creates a new performance checker
func NewPerformanceChecker() *PerformanceChecker {
	return &PerformanceChecker{
		enableMemoryLeaks:      true,
		enableSlowOperations:   true,
		enableLoopOptimization: true,
		enableDOMManipulation:  true,
		enableDatabaseQueries:  true,
		enableFileOperations:   true,
		enableNetworkCalls:     true,
	}
}

func (pc *PerformanceChecker) Name() string          { return "PerformanceVibe" }
func (pc *PerformanceChecker) Type() models.VibeType { return models.VibeTypePerformance }

func (pc *PerformanceChecker) Configure(config models.VibeConfig) error {
	pc.config = config
	
	if val, exists := config.Settings["enable_memory_leaks"]; exists {
		if boolVal, ok := val.(bool); ok {
			pc.enableMemoryLeaks = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_slow_operations"]; exists {
		if boolVal, ok := val.(bool); ok {
			pc.enableSlowOperations = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_loop_optimization"]; exists {
		if boolVal, ok := val.(bool); ok {
			pc.enableLoopOptimization = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_dom_manipulation"]; exists {
		if boolVal, ok := val.(bool); ok {
			pc.enableDOMManipulation = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_database_queries"]; exists {
		if boolVal, ok := val.(bool); ok {
			pc.enableDatabaseQueries = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_file_operations"]; exists {
		if boolVal, ok := val.(bool); ok {
			pc.enableFileOperations = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_network_calls"]; exists {
		if boolVal, ok := val.(bool); ok {
			pc.enableNetworkCalls = boolVal
		}
	}
	
	return nil
}

func (pc *PerformanceChecker) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedExts := []string{".js", ".jsx", ".ts", ".tsx", ".py", ".go", ".java", ".cpp", ".c", ".h", ".hpp", ".cs", ".php", ".rb", ".rs", ".swift", ".kt", ".sql"}
	
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

func (pc *PerformanceChecker) Check(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	for _, file := range files {
		if !pc.Supports(file) {
			continue
		}
		
		fileIssues, err := pc.checkFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check file %s: %w", file, err)
		}
		
		issues = append(issues, fileIssues...)
	}
	
	return issues, nil
}

func (pc *PerformanceChecker) checkFile(ctx context.Context, filename string) ([]models.Issue, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var issues []models.Issue
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	var lines []string

	// Read all lines first
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Check each line
	for i, line := range lines {
		lineNumber = i + 1
		
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		lineIssues := pc.checkLine(filename, line, lineNumber)
		issues = append(issues, lineIssues...)
	}

	// Check for cross-line performance issues
	crossLineIssues := pc.checkCrossLineIssues(filename, lines)
	issues = append(issues, crossLineIssues...)

	return issues, nil
}

func (pc *PerformanceChecker) checkLine(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	ext := strings.ToLower(filepath.Ext(filename))
	
	// Check for memory leaks
	if pc.enableMemoryLeaks {
		issues = append(issues, pc.checkMemoryLeaks(filename, line, lineNumber, ext)...)
	}
	
	// Check for slow operations
	if pc.enableSlowOperations {
		issues = append(issues, pc.checkSlowOperations(filename, line, lineNumber, ext)...)
	}
	
	// Check for loop optimization issues
	if pc.enableLoopOptimization {
		issues = append(issues, pc.checkLoopOptimization(filename, line, lineNumber, ext)...)
	}
	
	// Check for DOM manipulation issues
	if pc.enableDOMManipulation {
		issues = append(issues, pc.checkDOMManipulation(filename, line, lineNumber, ext)...)
	}
	
	// Check for database query issues
	if pc.enableDatabaseQueries {
		issues = append(issues, pc.checkDatabaseQueries(filename, line, lineNumber)...)
	}
	
	// Check for file operation issues
	if pc.enableFileOperations {
		issues = append(issues, pc.checkFileOperations(filename, line, lineNumber, ext)...)
	}
	
	// Check for network call issues
	if pc.enableNetworkCalls {
		issues = append(issues, pc.checkNetworkCalls(filename, line, lineNumber, ext)...)
	}
	
	return issues
}

func (pc *PerformanceChecker) checkMemoryLeaks(filename, line string, lineNumber int, ext string) []models.Issue {
	var issues []models.Issue
	
	// Language-specific memory leak patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
		extensions  []string
	}{
		{`setInterval\(.*\)(?!.*clearInterval)`, "setInterval without clearInterval", 0.8, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`setTimeout\(.*\)(?!.*clearTimeout)`, "setTimeout without clearTimeout", 0.6, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`addEventListener\(.*\)(?!.*removeEventListener)`, "Event listener without removal", 0.7, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`new\s+\w+\(.*\)(?!.*delete)`, "Memory allocation without deallocation", 0.5, []string{".cpp", ".c", ".h", ".hpp"}},
		{`malloc\(.*\)(?!.*free)`, "malloc without free", 0.9, []string{".c", ".cpp", ".h", ".hpp"}},
		{`new\s+\w+\[\](?!.*delete\[\])`, "Array allocation without deallocation", 0.8, []string{".cpp", ".h", ".hpp"}},
		{`fopen\(.*\)(?!.*fclose)`, "File handle without close", 0.7, []string{".c", ".cpp", ".h", ".hpp"}},
		{`open\(.*\)(?!.*close)`, "File descriptor without close", 0.7, []string{".py", ".go"}},
	}
	
	for _, p := range patterns {
		// Check if current file extension is supported for this pattern
		supported := false
		for _, supportedExt := range p.extensions {
			if ext == supportedExt {
				supported = true
				break
			}
		}
		
		if supported && regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypePerformance,
				Severity:    models.SeverityWarning,
				Title:       "Potential memory leak",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "memory-leak",
				Message:     "Ensure proper cleanup of resources to prevent memory leaks",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (pc *PerformanceChecker) checkSlowOperations(filename, line string, lineNumber int, ext string) []models.Issue {
	var issues []models.Issue
	
	// Slow operations patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
		extensions  []string
	}{
		{`document\.getElementsByTagName\(`, "Slow DOM query method", 0.7, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`document\.getElementsByClassName\(`, "Slow DOM query method", 0.7, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`\$\(".*"\)\.each\(`, "jQuery each loop", 0.6, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`for.*in.*Object\.keys\(`, "Inefficient object iteration", 0.7, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`JSON\.parse\(JSON\.stringify\(`, "Deep clone via JSON", 0.8, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`eval\(`, "Slow eval operation", 0.9, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`\.sort\(\)\.reverse\(\)`, "Inefficient sort and reverse", 0.7, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`time\.sleep\(`, "Blocking sleep operation", 0.8, []string{".py"}},
		{`Thread\.sleep\(`, "Blocking sleep operation", 0.8, []string{".java"}},
		{`time\.Sleep\(`, "Blocking sleep operation", 0.8, []string{".go"}},
	}
	
	for _, p := range patterns {
		// Check if current file extension is supported for this pattern
		supported := false
		for _, supportedExt := range p.extensions {
			if ext == supportedExt {
				supported = true
				break
			}
		}
		
		if supported && regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypePerformance,
				Severity:    models.SeverityWarning,
				Title:       "Slow operation detected",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "slow-operation",
				Message:     "Consider using more efficient alternatives",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (pc *PerformanceChecker) checkLoopOptimization(filename, line string, lineNumber int, ext string) []models.Issue {
	var issues []models.Issue
	
	// Loop optimization patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`for.*\.length`, "Length calculation in loop condition", 0.8},
		{`for.*\.size\(\)`, "Size calculation in loop condition", 0.8},
		{`for.*\.count\(\)`, "Count calculation in loop condition", 0.8},
		{`while.*\.length`, "Length calculation in while condition", 0.8},
		{`while.*\.size\(\)`, "Size calculation in while condition", 0.8},
		{`for.*getElementById\(`, "DOM query in loop", 0.9},
		{`for.*querySelector\(`, "DOM query in loop", 0.9},
		{`for.*find\(`, "Search operation in loop", 0.7},
		{`for.*indexOf\(`, "Index search in loop", 0.7},
		{`for.*includes\(`, "Includes check in loop", 0.7},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypePerformance,
				Severity:    models.SeverityWarning,
				Title:       "Loop optimization opportunity",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "loop-optimization",
				Message:     "Cache expensive operations outside the loop",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (pc *PerformanceChecker) checkDOMManipulation(filename, line string, lineNumber int, ext string) []models.Issue {
	var issues []models.Issue
	
	// Only check JavaScript/TypeScript files
	if ext != ".js" && ext != ".jsx" && ext != ".ts" && ext != ".tsx" {
		return issues
	}
	
	// DOM manipulation patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`appendChild\(.*createElement\(`, "Direct DOM manipulation", 0.7},
		{`innerHTML\s*\+=`, "Incremental innerHTML updates", 0.8},
		{`document\.write\(`, "Synchronous document.write", 0.9},
		{`style\.\w+\s*=`, "Direct style manipulation", 0.6},
		{`for.*appendChild\(`, "DOM manipulation in loop", 0.9},
		{`for.*removeChild\(`, "DOM manipulation in loop", 0.9},
		{`for.*insertBefore\(`, "DOM manipulation in loop", 0.9},
		{`\$\(.*\)\.append\(`, "jQuery DOM manipulation", 0.5},
		{`\$\(.*\)\.html\(`, "jQuery HTML manipulation", 0.5},
		{`getAttribute\(.*\)`, "Expensive attribute access", 0.4},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypePerformance,
				Severity:    models.SeverityInfo,
				Title:       "DOM manipulation performance",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "dom-manipulation",
				Message:     "Consider using virtual DOM or batching DOM updates",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (pc *PerformanceChecker) checkDatabaseQueries(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// Database query patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`SELECT \*`, "SELECT * query", 0.8},
		{`(?i)select.*from.*where.*like.*%.*%`, "LIKE query with leading wildcard", 0.9},
		{`(?i)select.*from.*where.*in\s*\(.*select`, "Subquery in WHERE IN", 0.8},
		{`(?i)select.*from.*order by.*without.*limit`, "ORDER BY without LIMIT", 0.7},
		{`(?i)select.*from.*group by.*having`, "HAVING without proper indexing", 0.6},
		{`(?i)select.*from.*where.*or.*where`, "Multiple OR conditions", 0.7},
		{`for.*query\(`, "Database query in loop", 0.9},
		{`for.*execute\(`, "Database execution in loop", 0.9},
		{`while.*query\(`, "Database query in while loop", 0.9},
		{`(?i)select.*from.*join.*join.*join`, "Multiple JOINs", 0.6},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypePerformance,
				Severity:    models.SeverityWarning,
				Title:       "Database query performance",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "database-query",
				Message:     "Optimize database queries for better performance",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (pc *PerformanceChecker) checkFileOperations(filename, line string, lineNumber int, ext string) []models.Issue {
	var issues []models.Issue
	
	// File operation patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
		extensions  []string
	}{
		{`for.*open\(`, "File operation in loop", 0.9, []string{".py", ".go", ".java", ".cpp", ".c"}},
		{`for.*readFile\(`, "File read in loop", 0.9, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`for.*writeFile\(`, "File write in loop", 0.9, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`readFileSync\(`, "Synchronous file read", 0.8, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`writeFileSync\(`, "Synchronous file write", 0.8, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`File\.ReadAllText\(`, "Reading entire file into memory", 0.7, []string{".cs"}},
		{`File\.WriteAllText\(`, "Writing entire file at once", 0.6, []string{".cs"}},
		{`BufferedReader\(.*FileReader\(`, "Unbuffered file reading", 0.6, []string{".java"}},
		{`os\.listdir\(`, "Listing directory contents", 0.5, []string{".py"}},
		{`glob\.glob\(`, "Pattern matching on filesystem", 0.5, []string{".py"}},
	}
	
	for _, p := range patterns {
		// Check if current file extension is supported for this pattern
		supported := false
		for _, supportedExt := range p.extensions {
			if ext == supportedExt {
				supported = true
				break
			}
		}
		
		if supported && regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypePerformance,
				Severity:    models.SeverityWarning,
				Title:       "File operation performance",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "file-operation",
				Message:     "Consider optimizing file operations for better performance",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (pc *PerformanceChecker) checkNetworkCalls(filename, line string, lineNumber int, ext string) []models.Issue {
	var issues []models.Issue
	
	// Network call patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
		extensions  []string
	}{
		{`for.*fetch\(`, "Network request in loop", 0.9, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`for.*axios\(`, "HTTP request in loop", 0.9, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`for.*\$\.ajax\(`, "AJAX request in loop", 0.9, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`for.*requests\.get\(`, "HTTP request in loop", 0.9, []string{".py"}},
		{`for.*http\.Get\(`, "HTTP request in loop", 0.9, []string{".go"}},
		{`for.*HttpClient\(`, "HTTP client in loop", 0.9, []string{".cs", ".java"}},
		{`await.*fetch\(.*await.*fetch\(`, "Sequential network calls", 0.8, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`XMLHttpRequest\(\)`, "Legacy XMLHttpRequest", 0.6, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`fetch\(.*\)\.then\(.*fetch\(`, "Chained network requests", 0.7, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`requests\.get\(.*timeout=None`, "Network request without timeout", 0.7, []string{".py"}},
	}
	
	for _, p := range patterns {
		// Check if current file extension is supported for this pattern
		supported := false
		for _, supportedExt := range p.extensions {
			if ext == supportedExt {
				supported = true
				break
			}
		}
		
		if supported && regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypePerformance,
				Severity:    models.SeverityWarning,
				Title:       "Network call performance",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "network-call",
				Message:     "Optimize network calls for better performance",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (pc *PerformanceChecker) checkCrossLineIssues(filename string, lines []string) []models.Issue {
	var issues []models.Issue
	
	// Check for functions that are too long or complex
	for i, line := range lines {
		if pc.isFunctionStart(line) {
			functionLines := pc.countFunctionLines(lines, i)
			if functionLines > 100 {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypePerformance,
					Severity:    models.SeverityWarning,
					Title:       "Large function detected",
					Description: fmt.Sprintf("Function has %d lines", functionLines),
					File:        filename,
					Line:        i + 1,
					Rule:        "large-function",
					Message:     "Large functions can impact performance and maintainability",
					Confidence:  0.8,
					Timestamp:   time.Now(),
				})
			}
		}
	}
	
	return issues
}

func (pc *PerformanceChecker) isFunctionStart(line string) bool {
	line = strings.TrimSpace(line)
	
	// Common function patterns across languages
	patterns := []string{
		`^function\s+\w+`,        // JavaScript
		`^def\s+\w+`,            // Python
		`^func\s+\w+`,           // Go
		`^public\s+.*\s+\w+\s*\(`, // Java/C#
		`^private\s+.*\s+\w+\s*\(`, // Java/C#
		`^\w+\s+\w+\s*\(`,       // C/C++
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, line); matched {
			return true
		}
	}
	
	return false
}

func (pc *PerformanceChecker) countFunctionLines(lines []string, startIndex int) int {
	if startIndex >= len(lines) {
		return 0
	}
	
	count := 1
	braceCount := 0
	
	// Count opening braces in the first line
	for _, char := range lines[startIndex] {
		if char == '{' {
			braceCount++
		} else if char == '}' {
			braceCount--
		}
	}
	
	// If no braces, this might be a Python function
	if braceCount == 0 && strings.Contains(lines[startIndex], "def ") {
		baseIndent := len(lines[startIndex]) - len(strings.TrimLeft(lines[startIndex], " \t"))
		for i := startIndex + 1; i < len(lines); i++ {
			line := lines[i]
			if strings.TrimSpace(line) == "" {
				count++
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " \t"))
			if indent <= baseIndent {
				break
			}
			count++
		}
		return count
	}
	
	// For brace-based languages
	for i := startIndex + 1; i < len(lines) && braceCount > 0; i++ {
		count++
		for _, char := range lines[i] {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
			}
		}
	}
	
	return count
}