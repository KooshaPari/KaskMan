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

// CodeChecker implements code quality checks
type CodeChecker struct {
	config               models.VibeConfig
	maxFunctionLength    int
	maxNestingDepth      int
	maxLineLength        int
	complexityThreshold  int
}

// NewCodeChecker creates a new code checker
func NewCodeChecker() *CodeChecker {
	return &CodeChecker{
		maxFunctionLength:   50,
		maxNestingDepth:     4,
		maxLineLength:       120,
		complexityThreshold: 10,
	}
}

func (cc *CodeChecker) Name() string          { return "CodeVibe" }
func (cc *CodeChecker) Type() models.VibeType { return models.VibeTypeCode }

func (cc *CodeChecker) Configure(config models.VibeConfig) error {
	cc.config = config
	
	if val, exists := config.Settings["max_function_length"]; exists {
		if intVal, ok := val.(int); ok {
			cc.maxFunctionLength = intVal
		}
	}
	
	if val, exists := config.Settings["max_nesting_depth"]; exists {
		if intVal, ok := val.(int); ok {
			cc.maxNestingDepth = intVal
		}
	}
	
	if val, exists := config.Settings["max_line_length"]; exists {
		if intVal, ok := val.(int); ok {
			cc.maxLineLength = intVal
		}
	}
	
	if val, exists := config.Settings["complexity_threshold"]; exists {
		if intVal, ok := val.(int); ok {
			cc.complexityThreshold = intVal
		}
	}
	
	return nil
}

func (cc *CodeChecker) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedExts := []string{".js", ".jsx", ".ts", ".tsx", ".py", ".go", ".java", ".cpp", ".c", ".h", ".hpp", ".cs", ".php", ".rb", ".rs", ".swift", ".kt"}
	
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

func (cc *CodeChecker) Check(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	for _, file := range files {
		if !cc.Supports(file) {
			continue
		}
		
		fileIssues, err := cc.checkFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check file %s: %w", file, err)
		}
		
		issues = append(issues, fileIssues...)
	}
	
	return issues, nil
}

func (cc *CodeChecker) checkFile(ctx context.Context, filename string) ([]models.Issue, error) {
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
		
		lineIssues := cc.checkLine(filename, line, lineNumber)
		issues = append(issues, lineIssues...)
	}
	
	// Check function-level issues
	functionIssues := cc.checkFunctions(filename, lines)
	issues = append(issues, functionIssues...)
	
	return issues, nil
}

func (cc *CodeChecker) checkLine(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	ext := strings.ToLower(filepath.Ext(filename))
	
	// Check line length
	if len(line) > cc.maxLineLength {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityWarning,
			Title:       "Line too long",
			Description: fmt.Sprintf("Line exceeds maximum length of %d characters", cc.maxLineLength),
			File:        filename,
			Line:        lineNumber,
			Rule:        "line-length",
			Message:     fmt.Sprintf("Line has %d characters, maximum allowed is %d", len(line), cc.maxLineLength),
			Confidence:  0.9,
			Timestamp:   time.Now(),
		})
	}
	
	// Language-specific checks
	switch ext {
	case ".js", ".jsx", ".ts", ".tsx":
		issues = append(issues, cc.checkJavaScript(filename, line, lineNumber)...)
	case ".py":
		issues = append(issues, cc.checkPython(filename, line, lineNumber)...)
	case ".go":
		issues = append(issues, cc.checkGo(filename, line, lineNumber)...)
	case ".java":
		issues = append(issues, cc.checkJava(filename, line, lineNumber)...)
	}
	
	// Check for TODO comments
	if cc.hasTodoComment(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityInfo,
			Title:       "TODO comment found",
			Description: "TODO comment should be addressed",
			File:        filename,
			Line:        lineNumber,
			Rule:        "todo-comments",
			Message:     "TODO comment found",
			Confidence:  1.0,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for commented out code
	if cc.hasCommentedCode(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityInfo,
			Title:       "Commented out code",
			Description: "Commented out code should be removed",
			File:        filename,
			Line:        lineNumber,
			Rule:        "commented-code",
			Message:     "Commented out code found",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for magic numbers
	if cc.hasMagicNumber(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityWarning,
			Title:       "Magic number detected",
			Description: "Magic numbers should be defined as constants",
			File:        filename,
			Line:        lineNumber,
			Rule:        "magic-numbers",
			Message:     "Magic number found",
			Confidence:  0.7,
			Timestamp:   time.Now(),
		})
	}
	
	return issues
}

func (cc *CodeChecker) checkJavaScript(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// Check for var usage
	if regexp.MustCompile(`\bvar\s+\w+`).MatchString(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityWarning,
			Title:       "Use of var keyword",
			Description: "Use let or const instead of var",
			File:        filename,
			Line:        lineNumber,
			Rule:        "no-var",
			Message:     "Avoid using var, use let or const instead",
			Confidence:  0.9,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for console.log
	if regexp.MustCompile(`console\.log\(`).MatchString(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityWarning,
			Title:       "Console.log usage",
			Description: "Console.log should not be used in production code",
			File:        filename,
			Line:        lineNumber,
			Rule:        "no-console-log",
			Message:     "Remove console.log statements",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for non-strict equality (avoid false positives with !== and === and strings)
	if regexp.MustCompile(`[^!=]==\s*[^=]`).MatchString(line) && 
		!regexp.MustCompile(`===|!==`).MatchString(line) &&
		!regexp.MustCompile(`['"][^'"]*==.*['"]`).MatchString(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityWarning,
			Title:       "Non-strict equality",
			Description: "Use strict equality (===) instead of loose equality (==)",
			File:        filename,
			Line:        lineNumber,
			Rule:        "strict-equality",
			Message:     "Use === instead of ==",
			Confidence:  0.9,
			Timestamp:   time.Now(),
		})
	}
	
	return issues
}

func (cc *CodeChecker) checkPython(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// Check for print statements
	if regexp.MustCompile(`\bprint\s*\(`).MatchString(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityInfo,
			Title:       "Print statement usage",
			Description: "Print statements should be removed or replaced with proper logging",
			File:        filename,
			Line:        lineNumber,
			Rule:        "no-print",
			Message:     "Use logging instead of print statements",
			Confidence:  0.7,
			Timestamp:   time.Now(),
		})
	}
	
	return issues
}

func (cc *CodeChecker) checkGo(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// Check for context.TODO()
	if regexp.MustCompile(`context\.TODO\(\)`).MatchString(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityWarning,
			Title:       "Context.TODO() usage",
			Description: "Replace context.TODO() with proper context",
			File:        filename,
			Line:        lineNumber,
			Rule:        "no-context-todo",
			Message:     "Use proper context instead of context.TODO()",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for panic
	if regexp.MustCompile(`\bpanic\(`).MatchString(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityError,
			Title:       "Panic usage",
			Description: "Avoid using panic, return errors instead",
			File:        filename,
			Line:        lineNumber,
			Rule:        "no-panic",
			Message:     "Use error handling instead of panic",
			Confidence:  0.9,
			Timestamp:   time.Now(),
		})
	}
	
	return issues
}

func (cc *CodeChecker) checkJava(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// Check for System.out.println
	if regexp.MustCompile(`System\.out\.println\(`).MatchString(line) {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeCode,
			Severity:    models.SeverityWarning,
			Title:       "System.out.println usage",
			Description: "Use proper logging instead of System.out.println",
			File:        filename,
			Line:        lineNumber,
			Rule:        "no-system-out",
			Message:     "Use logging framework instead of System.out.println",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	return issues
}

func (cc *CodeChecker) hasTodoComment(line string) bool {
	todoPattern := regexp.MustCompile(`(?i)(//|#|/\*|\*)\s*(TODO|FIXME|HACK|XXX|BUG)`)
	return todoPattern.MatchString(line)
}

func (cc *CodeChecker) hasCommentedCode(line string) bool {
	line = strings.TrimSpace(line)
	
	// Check for commented out code patterns
	patterns := []string{
		`^//\s*(var|let|const|function|if|for|while|return|import|export)`,
		`^#\s*(def|class|import|from|if|for|while|return|print)`,
		`^/\*\s*(var|let|const|function|if|for|while|return|import|export)`,
	}
	
	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(line) {
			return true
		}
	}
	
	return false
}

func (cc *CodeChecker) hasMagicNumber(line string) bool {
	// Skip lines with common patterns that shouldn't trigger magic number warnings
	skipPatterns := []string{
		`array\[.*\]`,                    // Array indexing
		`list\[.*\]`,                     // List indexing
		`\d+\.\d+\.\d+`,                 // Version numbers
		`port\s*[:=]\s*\d+`,             // Port assignments
		`timeout\s*[:=]\s*(1000|2000|3000|5000|10000|30000|60000)`, // Common timeout values only
		`buffer\s*[:=]\s*\d+`,           // Buffer sizes
		`size\s*[:=]\s*\d+`,             // Size specifications
		`length\s*[:=]\s*\d+`,           // Length specifications
		`count\s*[:=]\s*\d+`,            // Count specifications
		`\d+\s*(KB|MB|GB|TB|ms|s|min|h)`, // Units
		`0x[0-9a-fA-F]+`,                // Hex numbers
		`\b\d+\.\d+\b`,                  // Decimal numbers
		`for.*\d+.*\d+`,                 // For loop ranges
		`sleep\(\d+\)`,                  // Sleep calls
		`range\(\d+\)`,                  // Range calls
	}
	
	for _, pattern := range skipPatterns {
		if regexp.MustCompile(pattern).MatchString(line) {
			return false
		}
	}
	
	// Look for numeric literals that are not common numbers
	numberPattern := regexp.MustCompile(`\b(\d+)\b`)
	matches := numberPattern.FindAllStringSubmatch(line, -1)
	
	for _, match := range matches {
		if len(match) > 1 {
			number := match[1]
			// Skip common numbers and numbers in comments
			if !cc.isCommonNumber(number) && !cc.isInComment(line) {
				return true
			}
		}
	}
	
	return false
}

func (cc *CodeChecker) isCommonNumber(number string) bool {
	// Common numbers that are usually not magic numbers
	commonNumbers := []string{
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", // Basic numbers
		"12", "16", "24", "32", "60", "64", "128", "256", "512", "1024", // Powers of 2 and time units
		"100", "200", "300", "400", "500", "1000", "2000", "3000", "5000", // Common multiples
		"80", "443", "8000", "8080", "3000", "3001", "5000", "9000", // Common ports
		"7", "11", "13", "17", "19", "23", "29", "31", // Small primes
		"15", "20", "25", "30", "50", "75", // Common percentages/intervals
		"255", "65535", // Network/binary limits
		"86400", "3600", "604800", // Time constants (seconds in day/hour/week)
	}
	
	for _, common := range commonNumbers {
		if number == common {
			return true
		}
	}
	return false
}

func (cc *CodeChecker) isInComment(line string) bool {
	// Check if the line is a comment
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "/*")
}

func (cc *CodeChecker) checkFunctions(filename string, lines []string) []models.Issue {
	var issues []models.Issue
	
	// Simple function detection for different languages
	ext := strings.ToLower(filepath.Ext(filename))
	
	for i, line := range lines {
		lineNumber := i + 1
		
		// Detect function start
		if cc.isFunctionStart(line, ext) {
			functionLines := cc.countFunctionLines(lines, i)
			if functionLines > cc.maxFunctionLength {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeCode,
					Severity:    models.SeverityWarning,
					Title:       "Function too long",
					Description: fmt.Sprintf("Function has %d lines, maximum allowed is %d", functionLines, cc.maxFunctionLength),
					File:        filename,
					Line:        lineNumber,
					Rule:        "function-length",
					Message:     fmt.Sprintf("Function is %d lines long, consider breaking it down", functionLines),
					Confidence:  0.8,
					Timestamp:   time.Now(),
				})
			}
			
			// Check complexity
			complexity := cc.calculateComplexity(lines[i:i+functionLines])
			if complexity > cc.complexityThreshold {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeCode,
					Severity:    models.SeverityWarning,
					Title:       "High complexity",
					Description: fmt.Sprintf("Function has complexity %d, maximum allowed is %d", complexity, cc.complexityThreshold),
					File:        filename,
					Line:        lineNumber,
					Rule:        "complexity",
					Message:     fmt.Sprintf("Function complexity is %d, consider refactoring", complexity),
					Confidence:  0.7,
					Timestamp:   time.Now(),
				})
			}
		}
	}
	
	return issues
}

func (cc *CodeChecker) isFunctionStart(line string, ext string) bool {
	line = strings.TrimSpace(line)
	
	switch ext {
	case ".js", ".jsx", ".ts", ".tsx":
		patterns := []string{
			`^function\s+\w+\s*\(`,              // function name()
			`^const\s+\w+\s*=\s*\(`,            // const name = ()
			`^let\s+\w+\s*=\s*\(`,              // let name = ()
			`^var\s+\w+\s*=\s*function`,        // var name = function
			`^\w+\s*:\s*function`,              // name: function
			`^\w+\s*\(.*\)\s*\{`,               // name() {
			`^async\s+function\s+\w+`,          // async function name
			`^export\s+function\s+\w+`,         // export function name
			`^export\s+const\s+\w+\s*=\s*\(`,   // export const name = ()
			`^\w+\s*=\s*async\s*\(`,            // name = async ()
			`^\w+\s*=\s*\(.*\)\s*=>`,          // name = () => (arrow function)
		}
		for _, pattern := range patterns {
			if regexp.MustCompile(pattern).MatchString(line) {
				return true
			}
		}
		return false
	case ".py":
		patterns := []string{
			`^def\s+\w+\s*\(`,                  // def name()
			`^async\s+def\s+\w+\s*\(`,          // async def name()
			`^@\w+\s*$`,                        // Decorator (check next line)
		}
		for _, pattern := range patterns {
			if regexp.MustCompile(pattern).MatchString(line) {
				return true
			}
		}
		return false
	case ".go":
		patterns := []string{
			`^func\s+\w+\s*\(`,                 // func name()
			`^func\s+\(\w+\s+\*?\w+\)\s+\w+\s*\(`, // func (receiver) name()
		}
		for _, pattern := range patterns {
			if regexp.MustCompile(pattern).MatchString(line) {
				return true
			}
		}
		return false
	case ".java":
		patterns := []string{
			`(public|private|protected|static).*\w+\s*\(.*\)\s*\{`,  // method declaration
			`^\s*(public|private|protected)\s+.*\w+\s*\(`,          // method with modifiers
			`^\s*@\w+\s*$`,                                         // Annotation (check next line)
		}
		for _, pattern := range patterns {
			if regexp.MustCompile(pattern).MatchString(line) {
				return true
			}
		}
		return false
	case ".cpp", ".c", ".h", ".hpp":
		patterns := []string{
			`^\w+\s+\w+\s*\(.*\)\s*\{`,         // return_type name()
			`^static\s+\w+\s+\w+\s*\(`,         // static return_type name()
			`^inline\s+\w+\s+\w+\s*\(`,         // inline return_type name()
			`^virtual\s+\w+\s+\w+\s*\(`,        // virtual return_type name()
		}
		for _, pattern := range patterns {
			if regexp.MustCompile(pattern).MatchString(line) {
				return true
			}
		}
		return false
	case ".cs":
		patterns := []string{
			`(public|private|protected|internal).*\w+\s*\(.*\)\s*\{`, // method declaration
			`^\s*(public|private|protected|internal)\s+.*\w+\s*\(`,   // method with modifiers
			`^\s*\[.*\]\s*$`,                                         // Attribute (check next line)
		}
		for _, pattern := range patterns {
			if regexp.MustCompile(pattern).MatchString(line) {
				return true
			}
		}
		return false
	}
	
	return false
}

func (cc *CodeChecker) countFunctionLines(lines []string, startIndex int) int {
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
	
	// If no braces, this might be a Python function or similar
	if braceCount == 0 {
		// For Python, count until we find a line with same or less indentation
		if strings.Contains(lines[startIndex], "def ") {
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

func (cc *CodeChecker) calculateComplexity(lines []string) int {
	complexity := 1 // Base complexity
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Count complexity-increasing constructs
		complexityPatterns := []string{
			`\bif\b`, `\belse\b`, `\bfor\b`, `\bwhile\b`, `\bswitch\b`, `\bcase\b`,
			`\btry\b`, `\bcatch\b`, `\bfinally\b`, `\bthrow\b`,
			`\b&&\b`, `\b\|\|\b`, `\?.*:`,
		}
		
		for _, pattern := range complexityPatterns {
			if regexp.MustCompile(pattern).MatchString(line) {
				complexity++
			}
		}
	}
	
	return complexity
}