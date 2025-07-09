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

// SecurityChecker implements security vulnerability checks
type SecurityChecker struct {
	config                 models.VibeConfig
	enableSQLInjection     bool
	enableXSS              bool
	enableHardcodedSecrets bool
	enableInsecureRandom   bool
	enablePathTraversal    bool
	enableCommandInjection bool
	enableCryptoWeakness   bool
}

// NewSecurityChecker creates a new security checker
func NewSecurityChecker() *SecurityChecker {
	return &SecurityChecker{
		enableSQLInjection:     true,
		enableXSS:              true,
		enableHardcodedSecrets: true,
		enableInsecureRandom:   true,
		enablePathTraversal:    true,
		enableCommandInjection: true,
		enableCryptoWeakness:   true,
	}
}

func (sc *SecurityChecker) Name() string          { return "SecurityVibe" }
func (sc *SecurityChecker) Type() models.VibeType { return models.VibeTypeSecurity }

func (sc *SecurityChecker) Configure(config models.VibeConfig) error {
	sc.config = config
	
	if val, exists := config.Settings["enable_sql_injection"]; exists {
		if boolVal, ok := val.(bool); ok {
			sc.enableSQLInjection = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_xss"]; exists {
		if boolVal, ok := val.(bool); ok {
			sc.enableXSS = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_hardcoded_secrets"]; exists {
		if boolVal, ok := val.(bool); ok {
			sc.enableHardcodedSecrets = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_insecure_random"]; exists {
		if boolVal, ok := val.(bool); ok {
			sc.enableInsecureRandom = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_path_traversal"]; exists {
		if boolVal, ok := val.(bool); ok {
			sc.enablePathTraversal = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_command_injection"]; exists {
		if boolVal, ok := val.(bool); ok {
			sc.enableCommandInjection = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_crypto_weakness"]; exists {
		if boolVal, ok := val.(bool); ok {
			sc.enableCryptoWeakness = boolVal
		}
	}
	
	return nil
}

func (sc *SecurityChecker) Supports(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	supportedExts := []string{".js", ".jsx", ".ts", ".tsx", ".py", ".go", ".java", ".php", ".rb", ".cs", ".cpp", ".c", ".h", ".hpp", ".sql", ".json", ".xml", ".yaml", ".yml"}
	
	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return true
		}
	}
	return false
}

func (sc *SecurityChecker) Check(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	for _, file := range files {
		if !sc.Supports(file) {
			continue
		}
		
		fileIssues, err := sc.checkFile(ctx, file)
		if err != nil {
			return nil, fmt.Errorf("failed to check file %s: %w", file, err)
		}
		
		issues = append(issues, fileIssues...)
	}
	
	return issues, nil
}

func (sc *SecurityChecker) checkFile(ctx context.Context, filename string) ([]models.Issue, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var issues []models.Issue
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		lineIssues := sc.checkLine(filename, line, lineNumber)
		issues = append(issues, lineIssues...)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return issues, nil
}

func (sc *SecurityChecker) checkLine(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	ext := strings.ToLower(filepath.Ext(filename))
	
	// Check for SQL injection vulnerabilities
	if sc.enableSQLInjection {
		issues = append(issues, sc.checkSQLInjection(filename, line, lineNumber)...)
	}
	
	// Check for XSS vulnerabilities
	if sc.enableXSS {
		issues = append(issues, sc.checkXSS(filename, line, lineNumber)...)
	}
	
	// Check for hardcoded secrets
	if sc.enableHardcodedSecrets {
		issues = append(issues, sc.checkHardcodedSecrets(filename, line, lineNumber)...)
	}
	
	// Check for insecure random usage
	if sc.enableInsecureRandom {
		issues = append(issues, sc.checkInsecureRandom(filename, line, lineNumber, ext)...)
	}
	
	// Check for path traversal
	if sc.enablePathTraversal {
		issues = append(issues, sc.checkPathTraversal(filename, line, lineNumber)...)
	}
	
	// Check for command injection
	if sc.enableCommandInjection {
		issues = append(issues, sc.checkCommandInjection(filename, line, lineNumber)...)
	}
	
	// Check for crypto weaknesses
	if sc.enableCryptoWeakness {
		issues = append(issues, sc.checkCryptoWeakness(filename, line, lineNumber, ext)...)
	}
	
	return issues
}

func (sc *SecurityChecker) checkSQLInjection(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// SQL injection patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`"SELECT.*\+.*"`, "String concatenation in SQL query", 0.8},
		{`'SELECT.*\+.*'`, "String concatenation in SQL query", 0.8},
		{`query.*=.*".*\+.*"`, "SQL query with string concatenation", 0.9},
		{`query.*=.*'.*\+.*'`, "SQL query with string concatenation", 0.9},
		{`exec\(.*\+.*\)`, "Dynamic SQL execution", 0.7},
		{`executeQuery\(.*\+.*\)`, "Dynamic SQL execution", 0.7},
		{`fmt\.Sprintf.*SELECT`, "SQL query with format string", 0.6},
		{`%s.*SELECT`, "SQL query with string formatting", 0.6},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeSecurity,
				Severity:    models.SeverityError,
				Title:       "Potential SQL injection vulnerability",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "sql-injection",
				Message:     "Use parameterized queries to prevent SQL injection",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (sc *SecurityChecker) checkXSS(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// XSS patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`innerHTML.*=.*\+`, "Dynamic innerHTML assignment", 0.8},
		{`document\.write\(.*\+.*\)`, "Dynamic document.write", 0.9},
		{`dangerouslySetInnerHTML`, "Dangerous innerHTML in React", 0.7},
		{`v-html.*=.*\+`, "Dynamic v-html in Vue", 0.8},
		{`\$\{.*\}.*innerHTML`, "Template literal in innerHTML", 0.7},
		{`outerHTML.*=.*\+`, "Dynamic outerHTML assignment", 0.8},
		{`insertAdjacentHTML\(.*\+.*\)`, "Dynamic HTML insertion", 0.8},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeSecurity,
				Severity:    models.SeverityError,
				Title:       "Potential XSS vulnerability",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "xss-vulnerability",
				Message:     "Sanitize user input before rendering as HTML",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (sc *SecurityChecker) checkHardcodedSecrets(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// Hardcoded secrets patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`(?i)(password|pwd)\s*=\s*["'][^"'\s]{6,}["']`, "Hardcoded password", 0.9},
		{`(?i)(api[_-]?key|apikey)\s*=\s*["'][^"'\s]{10,}["']`, "Hardcoded API key", 0.9},
		{`(?i)(secret|token)\s*=\s*["'][^"'\s]{10,}["']`, "Hardcoded secret/token", 0.8},
		{`(?i)(auth[_-]?token|access[_-]?token)\s*=\s*["'][^"'\s]{10,}["']`, "Hardcoded auth token", 0.9},
		{`(?i)(private[_-]?key|privkey)\s*=\s*["'][^"'\s]{20,}["']`, "Hardcoded private key", 0.9},
		{`(?i)(client[_-]?secret|clientsecret)\s*=\s*["'][^"'\s]{10,}["']`, "Hardcoded client secret", 0.9},
		{`(?i)(database[_-]?url|db[_-]?url)\s*=\s*["'][^"'\s]{10,}["']`, "Hardcoded database URL", 0.8},
		{`(?i)(connection[_-]?string|conn[_-]?str)\s*=\s*["'][^"'\s]{10,}["']`, "Hardcoded connection string", 0.8},
		{`(?i)(jwt[_-]?secret|jwtsecret)\s*=\s*["'][^"'\s]{10,}["']`, "Hardcoded JWT secret", 0.9},
		{`(?i)(encryption[_-]?key|encrypt[_-]?key)\s*=\s*["'][^"'\s]{10,}["']`, "Hardcoded encryption key", 0.9},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeSecurity,
				Severity:    models.SeverityCritical,
				Title:       "Hardcoded secret detected",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "hardcoded-secrets",
				Message:     "Use environment variables or secure configuration for secrets",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (sc *SecurityChecker) checkInsecureRandom(filename, line string, lineNumber int, ext string) []models.Issue {
	var issues []models.Issue
	
	// Language-specific insecure random patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
		extensions  []string
	}{
		{`Math\.random\(\)`, "Insecure random number generation", 0.8, []string{".js", ".jsx", ".ts", ".tsx"}},
		{`random\.Random\(\)`, "Insecure random number generation", 0.8, []string{".py"}},
		{`rand\.Intn\(`, "Insecure random number generation", 0.7, []string{".go"}},
		{`new Random\(\)`, "Insecure random number generation", 0.8, []string{".java", ".cs"}},
		{`rand\(\)`, "Insecure random number generation", 0.7, []string{".c", ".cpp", ".h", ".hpp"}},
		{`Random\.new`, "Insecure random number generation", 0.8, []string{".rb"}},
		{`mt_rand\(`, "Insecure random number generation", 0.7, []string{".php"}},
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
				Type:        models.VibeTypeSecurity,
				Severity:    models.SeverityWarning,
				Title:       "Insecure random number generation",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "insecure-random",
				Message:     "Use cryptographically secure random number generation",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (sc *SecurityChecker) checkPathTraversal(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// Path traversal patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`\.\./`, "Path traversal sequence", 0.9},
		{`\.\.\\`, "Path traversal sequence (Windows)", 0.9},
		{`\.\.%2f`, "URL-encoded path traversal", 0.8},
		{`\.\.%5c`, "URL-encoded path traversal (Windows)", 0.8},
		{`os\.path\.join\(.*\.\./`, "Path traversal in os.path.join", 0.8},
		{`filepath\.Join\(.*\.\./`, "Path traversal in filepath.Join", 0.8},
		{`Path\.Combine\(.*\.\./`, "Path traversal in Path.Combine", 0.8},
		{`File\.ReadAllText\(.*\.\./`, "Path traversal in file read", 0.8},
		{`open\(.*\.\./`, "Path traversal in file open", 0.7},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeSecurity,
				Severity:    models.SeverityError,
				Title:       "Path traversal vulnerability",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "path-traversal",
				Message:     "Validate and sanitize file paths to prevent path traversal",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (sc *SecurityChecker) checkCommandInjection(filename, line string, lineNumber int) []models.Issue {
	var issues []models.Issue
	
	// Command injection patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`exec\(.*\+.*\)`, "Command injection via exec", 0.9},
		{`system\(.*\+.*\)`, "Command injection via system", 0.9},
		{`os\.system\(.*\+.*\)`, "Command injection via os.system", 0.9},
		{`subprocess\.call\(.*\+.*\)`, "Command injection via subprocess", 0.8},
		{`Runtime\.getRuntime\(\)\.exec\(.*\+.*\)`, "Command injection via Runtime.exec", 0.9},
		{`Process\.Start\(.*\+.*\)`, "Command injection via Process.Start", 0.8},
		{`shell_exec\(.*\+.*\)`, "Command injection via shell_exec", 0.9},
		{`passthru\(.*\+.*\)`, "Command injection via passthru", 0.9},
		{"`.*\\+.*`", "Command injection via backticks", 0.8},
		{`eval\(.*\+.*\)`, "Code injection via eval", 0.9},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeSecurity,
				Severity:    models.SeverityError,
				Title:       "Command injection vulnerability",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "command-injection",
				Message:     "Use safe command execution methods and validate inputs",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}

func (sc *SecurityChecker) checkCryptoWeakness(filename, line string, lineNumber int, ext string) []models.Issue {
	var issues []models.Issue
	
	// Weak cryptography patterns
	patterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`(?i)(md5|sha1)`, "Weak cryptographic hash function", 0.8},
		{`(?i)des[^c]`, "Weak encryption algorithm (DES)", 0.9},
		{`(?i)3des`, "Weak encryption algorithm (3DES)", 0.8},
		{`(?i)rc4`, "Weak encryption algorithm (RC4)", 0.9},
		{`(?i)blowfish`, "Weak encryption algorithm (Blowfish)", 0.7},
		{`(?i)ssl[^context]`, "Deprecated SSL protocol", 0.8},
		{`(?i)tls.*1\.[01]`, "Weak TLS version", 0.8},
		{`(?i)rsa.*1024`, "Weak RSA key size", 0.8},
		{`(?i)sha.*1`, "Weak SHA-1 hash", 0.8},
		{`(?i)cbc.*padding`, "Padding oracle vulnerability", 0.7},
	}
	
	for _, p := range patterns {
		if regexp.MustCompile(p.pattern).MatchString(line) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeSecurity,
				Severity:    models.SeverityWarning,
				Title:       "Weak cryptography detected",
				Description: p.description,
				File:        filename,
				Line:        lineNumber,
				Rule:        "weak-crypto",
				Message:     "Use strong cryptographic algorithms and proper key sizes",
				Confidence:  p.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues
}