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

// DocumentationChecker implements documentation quality and completeness checks
type DocumentationChecker struct {
	config                     models.VibeConfig
	enableReadmeChecks         bool
	enableCodeComments         bool
	enableAPIDocumentation     bool
	enableTODOTracking         bool
	enableSpellingChecks       bool
	enableLinkValidation       bool
	enableCodeExamples         bool
	enableChangelogChecks      bool
	minReadmeLength            int
	maxTODOAge                 int // days
	requiredSections           []string
}

// NewDocumentationChecker creates a new documentation checker
func NewDocumentationChecker() *DocumentationChecker {
	return &DocumentationChecker{
		enableReadmeChecks:     true,
		enableCodeComments:     true,
		enableAPIDocumentation: true,
		enableTODOTracking:     true,
		enableSpellingChecks:   true,
		enableLinkValidation:   true,
		enableCodeExamples:     true,
		enableChangelogChecks:  true,
		minReadmeLength:        500,
		maxTODOAge:            30,
		requiredSections: []string{
			"installation", "usage", "examples", "contributing",
			"license", "description", "overview",
		},
	}
}

func (doc *DocumentationChecker) Name() string          { return "DocumentationVibe" }
func (doc *DocumentationChecker) Type() models.VibeType { return models.VibeTypeDocumentation }

func (doc *DocumentationChecker) Configure(config models.VibeConfig) error {
	doc.config = config
	
	if val, exists := config.Settings["enable_readme_checks"]; exists {
		if boolVal, ok := val.(bool); ok {
			doc.enableReadmeChecks = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_code_comments"]; exists {
		if boolVal, ok := val.(bool); ok {
			doc.enableCodeComments = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_api_documentation"]; exists {
		if boolVal, ok := val.(bool); ok {
			doc.enableAPIDocumentation = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_todo_tracking"]; exists {
		if boolVal, ok := val.(bool); ok {
			doc.enableTODOTracking = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_spelling_checks"]; exists {
		if boolVal, ok := val.(bool); ok {
			doc.enableSpellingChecks = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_link_validation"]; exists {
		if boolVal, ok := val.(bool); ok {
			doc.enableLinkValidation = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_code_examples"]; exists {
		if boolVal, ok := val.(bool); ok {
			doc.enableCodeExamples = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_changelog_checks"]; exists {
		if boolVal, ok := val.(bool); ok {
			doc.enableChangelogChecks = boolVal
		}
	}
	
	if val, exists := config.Settings["min_readme_length"]; exists {
		if intVal, ok := val.(int); ok {
			doc.minReadmeLength = intVal
		}
	}
	
	if val, exists := config.Settings["max_todo_age"]; exists {
		if intVal, ok := val.(int); ok {
			doc.maxTODOAge = intVal
		}
	}
	
	if val, exists := config.Settings["required_sections"]; exists {
		if strSlice, ok := val.([]string); ok {
			doc.requiredSections = strSlice
		}
	}
	
	return nil
}

func (doc *DocumentationChecker) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	baseName := strings.ToLower(filepath.Base(filename))
	
	// Documentation file extensions
	docExts := []string{".md", ".txt", ".rst", ".adoc", ".org", ".wiki"}
	
	// Check extensions
	for _, docExt := range docExts {
		if ext == docExt {
			return true
		}
	}
	
	// Check specific filenames
	docFiles := []string{
		"readme", "readme.md", "readme.txt", "readme.rst",
		"changelog", "changelog.md", "changelog.txt",
		"contributing", "contributing.md",
		"license", "license.md", "license.txt",
		"todo", "todo.md", "todo.txt",
		"docs", "documentation",
		"api", "api.md", "api.txt",
		"guide", "guide.md",
		"tutorial", "tutorial.md",
		"examples", "examples.md",
		"faq", "faq.md",
		"troubleshooting", "troubleshooting.md",
	}
	
	for _, docFile := range docFiles {
		if strings.Contains(baseName, docFile) {
			return true
		}
	}
	
	// Check code files for comments
	codeExts := []string{".js", ".jsx", ".ts", ".tsx", ".py", ".go", ".java", ".php", ".rb", ".cs", ".cpp", ".c", ".h", ".hpp", ".rs", ".swift", ".kt"}
	for _, codeExt := range codeExts {
		if ext == codeExt {
			return true
		}
	}
	
	return false
}

func (doc *DocumentationChecker) Check(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Group files by type
	readmeFiles := []string{}
	changelogFiles := []string{}
	codeFiles := []string{}
	otherDocFiles := []string{}
	
	for _, file := range files {
		if !doc.Supports(file) {
			continue
		}
		
		baseName := strings.ToLower(filepath.Base(file))
		ext := strings.ToLower(filepath.Ext(file))
		
		if strings.Contains(baseName, "readme") {
			readmeFiles = append(readmeFiles, file)
		} else if strings.Contains(baseName, "changelog") || strings.Contains(baseName, "changes") {
			changelogFiles = append(changelogFiles, file)
		} else if doc.isCodeFile(ext) {
			codeFiles = append(codeFiles, file)
		} else {
			otherDocFiles = append(otherDocFiles, file)
		}
	}
	
	// Check README files
	if doc.enableReadmeChecks {
		readmeIssues, err := doc.checkReadmeFiles(ctx, readmeFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to check README files: %w", err)
		}
		issues = append(issues, readmeIssues...)
	}
	
	// Check changelog files
	if doc.enableChangelogChecks {
		changelogIssues, err := doc.checkChangelogFiles(ctx, changelogFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to check changelog files: %w", err)
		}
		issues = append(issues, changelogIssues...)
	}
	
	// Check code files for comments
	if doc.enableCodeComments {
		commentIssues, err := doc.checkCodeComments(ctx, codeFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to check code comments: %w", err)
		}
		issues = append(issues, commentIssues...)
	}
	
	// Check other documentation files
	for _, file := range otherDocFiles {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		fileIssues, err := doc.checkDocumentationFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check documentation file %s: %w", file, err)
		}
		issues = append(issues, fileIssues...)
	}
	
	return issues, nil
}

func (doc *DocumentationChecker) checkReadmeFiles(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	if len(files) == 0 {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDocumentation,
			Severity:    models.SeverityWarning,
			Title:       "Missing README file",
			Description: "No README file found in the project",
			File:        ".",
			Line:        1,
			Rule:        "missing-readme",
			Message:     "Add a README file to document your project",
			Confidence:  1.0,
			Timestamp:   time.Now(),
		})
		return issues, nil
	}
	
	for _, file := range files {
		fileIssues, err := doc.checkReadmeFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check README file: %w", err)
		}
		issues = append(issues, fileIssues...)
	}
	
	return issues, nil
}

func (doc *DocumentationChecker) checkReadmeFile(ctx context.Context, filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read README file: %w", err)
	}
	
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")
	
	// Check README length
	if len(contentStr) < doc.minReadmeLength {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDocumentation,
			Severity:    models.SeverityWarning,
			Title:       "Short README file",
			Description: fmt.Sprintf("README is only %d characters (minimum recommended: %d)", len(contentStr), doc.minReadmeLength),
			File:        filename,
			Line:        1,
			Rule:        "short-readme",
			Message:     "Consider expanding the README with more details",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for required sections
	contentLower := strings.ToLower(contentStr)
	missingSections := []string{}
	
	for _, section := range doc.requiredSections {
		if !strings.Contains(contentLower, section) {
			missingSections = append(missingSections, section)
		}
	}
	
	if len(missingSections) > 0 {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDocumentation,
			Severity:    models.SeverityInfo,
			Title:       "Missing README sections",
			Description: fmt.Sprintf("README is missing sections: %s", strings.Join(missingSections, ", ")),
			File:        filename,
			Line:        1,
			Rule:        "missing-sections",
			Message:     "Consider adding these sections to improve documentation",
			Confidence:  0.7,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for broken links
	if doc.enableLinkValidation {
		linkIssues := doc.checkLinksInContent(filename, contentStr)
		issues = append(issues, linkIssues...)
	}
	
	// Check for code examples
	if doc.enableCodeExamples {
		if !strings.Contains(contentStr, "```") && !strings.Contains(contentStr, "    ") {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeDocumentation,
				Severity:    models.SeverityInfo,
				Title:       "No code examples in README",
				Description: "README doesn't contain any code examples",
				File:        filename,
				Line:        1,
				Rule:        "no-code-examples",
				Message:     "Consider adding code examples to help users",
				Confidence:  0.6,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check for common typos and issues
	if doc.enableSpellingChecks {
		spellingIssues := doc.checkSpelling(filename, lines)
		issues = append(issues, spellingIssues...)
	}
	
	return issues, nil
}

func (doc *DocumentationChecker) checkChangelogFiles(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	if len(files) == 0 {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDocumentation,
			Severity:    models.SeverityInfo,
			Title:       "Missing changelog",
			Description: "No changelog file found in the project",
			File:        ".",
			Line:        1,
			Rule:        "missing-changelog",
			Message:     "Consider adding a changelog to track project changes",
			Confidence:  0.7,
			Timestamp:   time.Now(),
		})
		return issues, nil
	}
	
	for _, file := range files {
		fileIssues, err := doc.checkChangelogFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check changelog file: %w", err)
		}
		issues = append(issues, fileIssues...)
	}
	
	return issues, nil
}

func (doc *DocumentationChecker) checkChangelogFile(ctx context.Context, filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read changelog file: %w", err)
	}
	
	contentStr := string(content)
	
	// Check if changelog follows standard format
	hasVersions := regexp.MustCompile(`(?i)(version|v\d+\.\d+|\d+\.\d+\.\d+|## \[)`).MatchString(contentStr)
	if !hasVersions {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDocumentation,
			Severity:    models.SeverityInfo,
			Title:       "Unstructured changelog",
			Description: "Changelog doesn't follow standard versioning format",
			File:        filename,
			Line:        1,
			Rule:        "unstructured-changelog",
			Message:     "Consider using semantic versioning in changelog",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for recent updates
	hasRecent := regexp.MustCompile(`(?i)(2024|2023)`).MatchString(contentStr)
	if !hasRecent {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDocumentation,
			Severity:    models.SeverityInfo,
			Title:       "Outdated changelog",
			Description: "Changelog doesn't contain recent entries",
			File:        filename,
			Line:        1,
			Rule:        "outdated-changelog",
			Message:     "Update changelog with recent changes",
			Confidence:  0.6,
			Timestamp:   time.Now(),
		})
	}
	
	return issues, nil
}

func (doc *DocumentationChecker) checkCodeComments(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	for _, file := range files {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		fileIssues, err := doc.checkCodeFileComments(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check code comments in %s: %w", file, err)
		}
		issues = append(issues, fileIssues...)
	}
	
	return issues, nil
}

func (doc *DocumentationChecker) checkCodeFileComments(ctx context.Context, filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	totalLines := 0
	commentLines := 0
	functionLines := 0
	documentedFunctions := 0
	
	for scanner.Scan() {
		lineNumber++
		totalLines++
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" {
			continue
		}
		
		// Count comment lines
		if doc.isCommentLine(line, filename) {
			commentLines++
		}
		
		// Count functions and their documentation
		if doc.isFunctionLine(line, filename) {
			functionLines++
			// Check if previous lines contain documentation
			if doc.hasPrecedingDocumentation(scanner, lineNumber) {
				documentedFunctions++
			}
		}
		
		// Check for TODO/FIXME items
		if doc.enableTODOTracking {
			todoIssues := doc.checkTODOItems(filename, line, lineNumber)
			issues = append(issues, todoIssues...)
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	// Calculate comment ratio
	if totalLines > 10 { // Only check files with substantial content
		commentRatio := float64(commentLines) / float64(totalLines)
		
		if commentRatio < 0.1 { // Less than 10% comments
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeDocumentation,
				Severity:    models.SeverityInfo,
				Title:       "Low comment ratio",
				Description: fmt.Sprintf("Only %.1f%% of lines are comments", commentRatio*100),
				File:        filename,
				Line:        1,
				Rule:        "low-comment-ratio",
				Message:     "Consider adding more comments to explain complex logic",
				Confidence:  0.6,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check function documentation ratio
	if functionLines > 0 {
		docRatio := float64(documentedFunctions) / float64(functionLines)
		if docRatio < 0.5 { // Less than 50% of functions documented
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeDocumentation,
				Severity:    models.SeverityInfo,
				Title:       "Undocumented functions",
				Description: fmt.Sprintf("Only %.1f%% of functions have documentation", docRatio*100),
				File:        filename,
				Line:        1,
				Rule:        "undocumented-functions",
				Message:     "Add documentation comments for public functions",
				Confidence:  0.7,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues, nil
}

func (doc *DocumentationChecker) checkDocumentationFile(ctx context.Context, filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read documentation file: %w", err)
	}
	
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")
	
	// Check for spelling issues
	if doc.enableSpellingChecks {
		spellingIssues := doc.checkSpelling(filename, lines)
		issues = append(issues, spellingIssues...)
	}
	
	// Check for broken links
	if doc.enableLinkValidation {
		linkIssues := doc.checkLinksInContent(filename, contentStr)
		issues = append(issues, linkIssues...)
	}
	
	return issues, nil
}

func (doc *DocumentationChecker) isCodeFile(ext string) bool {
	codeExts := []string{".js", ".jsx", ".ts", ".tsx", ".py", ".go", ".java", ".php", ".rb", ".cs", ".cpp", ".c", ".h", ".hpp", ".rs", ".swift", ".kt"}
	for _, codeExt := range codeExts {
		if ext == codeExt {
			return true
		}
	}
	return false
}

func (doc *DocumentationChecker) isCommentLine(line, filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	
	switch ext {
	case ".js", ".jsx", ".ts", ".tsx", ".java", ".go", ".cs", ".cpp", ".c", ".h", ".hpp", ".rs", ".swift", ".kt":
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*")
	case ".py", ".rb":
		return strings.HasPrefix(line, "#")
	case ".php":
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "/*")
	default:
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "/*")
	}
}

func (doc *DocumentationChecker) isFunctionLine(line, filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	
	patterns := map[string][]string{
		".js":   {`function\s+\w+`, `\w+\s*:\s*function`, `\w+\s*=>\s*{`, `\w+\s*=\s*function`},
		".jsx":  {`function\s+\w+`, `\w+\s*:\s*function`, `\w+\s*=>\s*{`, `\w+\s*=\s*function`},
		".ts":   {`function\s+\w+`, `\w+\s*:\s*function`, `\w+\s*=>\s*{`, `\w+\s*=\s*function`},
		".tsx":  {`function\s+\w+`, `\w+\s*:\s*function`, `\w+\s*=>\s*{`, `\w+\s*=\s*function`},
		".py":   {`def\s+\w+`, `class\s+\w+`, `async\s+def\s+\w+`},
		".go":   {`func\s+\w+`, `func\s+\(\w+\s+\w+\)\s+\w+`},
		".java": {`public\s+.*\s+\w+\s*\(`, `private\s+.*\s+\w+\s*\(`, `protected\s+.*\s+\w+\s*\(`},
		".php":  {`function\s+\w+`, `public\s+function\s+\w+`, `private\s+function\s+\w+`},
		".rb":   {`def\s+\w+`, `class\s+\w+`},
		".cs":   {`public\s+.*\s+\w+\s*\(`, `private\s+.*\s+\w+\s*\(`, `protected\s+.*\s+\w+\s*\(`},
		".cpp":  {`\w+\s+\w+\s*\(.*\)\s*{`, `\w+::\w+\s*\(`},
		".c":    {`\w+\s+\w+\s*\(.*\)\s*{`},
		".rs":   {`fn\s+\w+`, `pub\s+fn\s+\w+`},
	}
	
	if funcPatterns, exists := patterns[ext]; exists {
		for _, pattern := range funcPatterns {
			if matched, _ := regexp.MatchString(pattern, line); matched {
				return true
			}
		}
	}
	
	return false
}

func (doc *DocumentationChecker) hasPrecedingDocumentation(scanner *bufio.Scanner, lineNumber int) bool {
	// This is a simplified check - in a real implementation, we'd need to look back
	// at previous lines to check for documentation comments
	return false
}

func (doc *DocumentationChecker) checkTODOItems(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// TODO/FIXME patterns
	patterns := []struct {
		pattern     string
		description string
		severity    models.SeverityLevel
	}{
		{`(?i)todo:?(.*)`, "TODO item found", models.SeverityInfo},
		{`(?i)fixme:?(.*)`, "FIXME item found", models.SeverityWarning},
		{`(?i)hack:?(.*)`, "HACK item found", models.SeverityWarning},
		{`(?i)bug:?(.*)`, "BUG item found", models.SeverityError},
		{`(?i)xxx:?(.*)`, "XXX item found", models.SeverityWarning},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeDocumentation,
				Severity:    p.severity,
				Title:       p.description,
				Description: strings.TrimSpace(line),
				File:        filename,
				Line:        lineNumber,
				Rule:        "todo-item",
				Message:     "Consider addressing this item",
				Confidence:  0.9,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (doc *DocumentationChecker) checkSpelling(filename string, lines []string) []models.Issue {
	var issues []models.Issue
	
	// Common typos and misspellings
	typos := map[string]string{
		"recieve":     "receive",
		"occured":     "occurred",
		"seperate":    "separate",
		"definately":  "definitely",
		"accomodate":  "accommodate",
		"beleive":     "believe",
		"concensus":   "consensus",
		"embarass":    "embarrass",
		"existance":   "existence",
		"foriegn":     "foreign",
		"independant": "independent",
		"neccessary":  "necessary",
		"occassion":   "occasion",
		"priviledge":  "privilege",
		"publically":  "publicly",
		"reccommend":  "recommend",
		"rythm":       "rhythm",
		"succesful":   "successful",
		"tommorow":    "tomorrow",
		"untill":      "until",
	}
	
	for lineNumber, line := range lines {
		lineLower := strings.ToLower(line)
		for typo, correct := range typos {
			if strings.Contains(lineLower, typo) {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeDocumentation,
					Severity:    models.SeverityInfo,
					Title:       "Possible spelling error",
					Description: fmt.Sprintf("'%s' should be '%s'", typo, correct),
					File:        filename,
					Line:        lineNumber + 1,
					Rule:        "spelling-error",
					Message:     "Check spelling and grammar",
					Confidence:  0.8,
					Timestamp:   time.Now(),
				})
			}
		}
	}
	
	return issues
}

func (doc *DocumentationChecker) checkLinksInContent(filename, content string) []models.Issue {
	var issues []models.Issue
	
	// Find markdown links and URLs
	linkPattern := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)|https?://[^\s)]+`)
	matches := linkPattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) > 2 {
			link := match[2]
			// Check for obviously broken links
			if strings.Contains(link, "localhost") || strings.Contains(link, "127.0.0.1") {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeDocumentation,
					Severity:    models.SeverityWarning,
					Title:       "Local link in documentation",
					Description: fmt.Sprintf("Link points to localhost: %s", link),
					File:        filename,
					Line:        1,
					Rule:        "local-link",
					Message:     "Replace localhost links with public URLs",
					Confidence:  0.9,
					Timestamp:   time.Now(),
				})
			}
			
			if strings.Contains(link, "example.com") {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeDocumentation,
					Severity:    models.SeverityInfo,
					Title:       "Example link in documentation",
					Description: fmt.Sprintf("Link uses example.com: %s", link),
					File:        filename,
					Line:        1,
					Rule:        "example-link",
					Message:     "Replace example links with real URLs",
					Confidence:  0.7,
					Timestamp:   time.Now(),
				})
			}
		}
	}
	
	return issues
}