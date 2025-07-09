package vibes

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kooshapari/kodevibe-go/internal/models"
)

// DependencyChecker implements dependency analysis checks
type DependencyChecker struct {
	config                    models.VibeConfig
	enableOutdatedDeps        bool
	enableVulnerableDeps      bool
	enableUnusedDeps          bool
	enableDuplicateDeps       bool
	enableDevDependencies     bool
	enableLicenseChecks       bool
	enableCircularDeps        bool
	maxDependencyCount        int
	allowedLicenses           []string
	forbiddenDependencies     []string
}

// NewDependencyChecker creates a new dependency checker
func NewDependencyChecker() *DependencyChecker {
	return &DependencyChecker{
		enableOutdatedDeps:     true,
		enableVulnerableDeps:   true,
		enableUnusedDeps:       true,
		enableDuplicateDeps:    true,
		enableDevDependencies: true,
		enableLicenseChecks:    true,
		enableCircularDeps:     true,
		maxDependencyCount:     100,
		allowedLicenses:        []string{"MIT", "Apache-2.0", "BSD-3-Clause", "BSD-2-Clause", "ISC"},
		forbiddenDependencies:  []string{},
	}
}

func (dc *DependencyChecker) Name() string          { return "DependencyVibe" }
func (dc *DependencyChecker) Type() models.VibeType { return models.VibeTypeDependency }

func (dc *DependencyChecker) Configure(config models.VibeConfig) error {
	dc.config = config
	
	if val, exists := config.Settings["enable_outdated_deps"]; exists {
		if boolVal, ok := val.(bool); ok {
			dc.enableOutdatedDeps = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_vulnerable_deps"]; exists {
		if boolVal, ok := val.(bool); ok {
			dc.enableVulnerableDeps = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_unused_deps"]; exists {
		if boolVal, ok := val.(bool); ok {
			dc.enableUnusedDeps = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_duplicate_deps"]; exists {
		if boolVal, ok := val.(bool); ok {
			dc.enableDuplicateDeps = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_dev_dependencies"]; exists {
		if boolVal, ok := val.(bool); ok {
			dc.enableDevDependencies = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_license_checks"]; exists {
		if boolVal, ok := val.(bool); ok {
			dc.enableLicenseChecks = boolVal
		}
	}
	
	if val, exists := config.Settings["enable_circular_deps"]; exists {
		if boolVal, ok := val.(bool); ok {
			dc.enableCircularDeps = boolVal
		}
	}
	
	if val, exists := config.Settings["max_dependency_count"]; exists {
		if intVal, ok := val.(int); ok {
			dc.maxDependencyCount = intVal
		}
	}
	
	if val, exists := config.Settings["allowed_licenses"]; exists {
		if strSlice, ok := val.([]string); ok {
			dc.allowedLicenses = strSlice
		}
	}
	
	if val, exists := config.Settings["forbidden_dependencies"]; exists {
		if strSlice, ok := val.([]string); ok {
			dc.forbiddenDependencies = strSlice
		}
	}
	
	return nil
}

func (dc *DependencyChecker) Supports(filename string) bool {
	baseName := filepath.Base(filename)
	supportedFiles := []string{
		"package.json",       // Node.js
		"yarn.lock",          // Yarn
		"package-lock.json",  // npm
		"requirements.txt",   // Python pip
		"Pipfile",           // Python pipenv
		"Pipfile.lock",      // Python pipenv
		"poetry.lock",       // Python poetry
		"pyproject.toml",    // Python
		"go.mod",            // Go
		"go.sum",            // Go
		"Cargo.toml",        // Rust
		"Cargo.lock",        // Rust
		"Gemfile",           // Ruby
		"Gemfile.lock",      // Ruby
		"composer.json",     // PHP
		"composer.lock",     // PHP
		"pom.xml",           // Maven
		"build.gradle",      // Gradle
		"build.gradle.kts",  // Gradle Kotlin
		"setup.py",          // Python setup
		"environment.yml",   // Conda
		"bower.json",        // Bower
		"project.clj",       // Leiningen (Clojure)
		"deps.edn",          // Clojure
		"mix.exs",           // Elixir
		"rebar.config",      // Erlang
		"*.nuspec",          // NuGet
		"packages.config",   // NuGet
		"*.csproj",          // .NET
		"*.fsproj",          // F#
		"*.vbproj",          // VB.NET
	}
	
	for _, supported := range supportedFiles {
		if strings.Contains(supported, "*") {
			// Handle wildcard patterns
			pattern := strings.ReplaceAll(supported, "*", ".*")
			if matched, _ := regexp.MatchString(pattern+"$", baseName); matched {
				return true
			}
		} else if baseName == supported {
			return true
		}
	}
	return false
}

func (dc *DependencyChecker) Check(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Group files by project type
	projects := dc.groupFilesByProject(files)
	
	for projectType, projectFiles := range projects {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		
		projectIssues, err := dc.checkProject(ctx, projectType, projectFiles)
		if err != nil {
			return nil, fmt.Errorf("failed to check %s project: %w", projectType, err)
		}
		issues = append(issues, projectIssues...)
	}
	
	return issues, nil
}

func (dc *DependencyChecker) groupFilesByProject(files []string) map[string][]string {
	projects := make(map[string][]string)
	
	for _, file := range files {
		baseName := filepath.Base(file)
		
		switch baseName {
		case "package.json", "yarn.lock", "package-lock.json", "bower.json":
			projects["nodejs"] = append(projects["nodejs"], file)
		case "requirements.txt", "Pipfile", "Pipfile.lock", "poetry.lock", "pyproject.toml", "setup.py", "environment.yml":
			projects["python"] = append(projects["python"], file)
		case "go.mod", "go.sum":
			projects["go"] = append(projects["go"], file)
		case "Cargo.toml", "Cargo.lock":
			projects["rust"] = append(projects["rust"], file)
		case "Gemfile", "Gemfile.lock":
			projects["ruby"] = append(projects["ruby"], file)
		case "composer.json", "composer.lock":
			projects["php"] = append(projects["php"], file)
		case "pom.xml":
			projects["maven"] = append(projects["maven"], file)
		case "build.gradle", "build.gradle.kts":
			projects["gradle"] = append(projects["gradle"], file)
		default:
			// Check for pattern-based matches
			if strings.HasSuffix(baseName, ".csproj") || strings.HasSuffix(baseName, ".fsproj") || strings.HasSuffix(baseName, ".vbproj") {
				projects["dotnet"] = append(projects["dotnet"], file)
			} else if strings.HasSuffix(baseName, ".nuspec") || baseName == "packages.config" {
				projects["nuget"] = append(projects["nuget"], file)
			}
		}
	}
	
	return projects
}

func (dc *DependencyChecker) checkProject(ctx context.Context, projectType string, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	switch projectType {
	case "nodejs":
		nodeIssues, err := dc.checkNodeJSProject(ctx, files)
		if err != nil {
			return nil, err
		}
		issues = append(issues, nodeIssues...)
	case "python":
		pythonIssues, err := dc.checkPythonProject(ctx, files)
		if err != nil {
			return nil, err
		}
		issues = append(issues, pythonIssues...)
	case "go":
		goIssues, err := dc.checkGoProject(ctx, files)
		if err != nil {
			return nil, err
		}
		issues = append(issues, goIssues...)
	case "rust":
		rustIssues, err := dc.checkRustProject(ctx, files)
		if err != nil {
			return nil, err
		}
		issues = append(issues, rustIssues...)
	default:
		// Generic checks for unknown project types
		for _, file := range files {
			genericIssues, err := dc.checkGenericDependencyFile(ctx, file)
			if err != nil {
				return nil, err
			}
			issues = append(issues, genericIssues...)
		}
	}
	
	return issues, nil
}

func (dc *DependencyChecker) checkNodeJSProject(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	for _, file := range files {
		baseName := filepath.Base(file)
		
		if baseName == "package.json" {
			packageIssues, err := dc.checkPackageJSON(file)
			if err != nil {
				return nil, fmt.Errorf("failed to check package.json: %w", err)
			}
			issues = append(issues, packageIssues...)
		} else if baseName == "yarn.lock" || baseName == "package-lock.json" {
			lockIssues, err := dc.checkLockFile(file, "nodejs")
			if err != nil {
				return nil, fmt.Errorf("failed to check lock file: %w", err)
			}
			issues = append(issues, lockIssues...)
		}
	}
	
	return issues, nil
}

func (dc *DependencyChecker) checkPackageJSON(filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read package.json: %w", err)
	}
	
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Scripts         map[string]string `json:"scripts"`
		License         string            `json:"license"`
		Name            string            `json:"name"`
		Version         string            `json:"version"`
	}
	
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}
	
	// Check dependency count
	totalDeps := len(pkg.Dependencies) + len(pkg.DevDependencies)
	if totalDeps > dc.maxDependencyCount {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDependency,
			Severity:    models.SeverityWarning,
			Title:       "Too many dependencies",
			Description: fmt.Sprintf("Project has %d dependencies (max recommended: %d)", totalDeps, dc.maxDependencyCount),
			File:        filename,
			Line:        1,
			Rule:        "dependency-count",
			Message:     "Consider reducing the number of dependencies",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	// Check for forbidden dependencies
	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}
	
	for dep := range allDeps {
		for _, forbidden := range dc.forbiddenDependencies {
			if dep == forbidden {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeDependency,
					Severity:    models.SeverityError,
					Title:       "Forbidden dependency",
					Description: fmt.Sprintf("Dependency '%s' is not allowed", dep),
					File:        filename,
					Line:        1,
					Rule:        "forbidden-dependency",
					Message:     "Remove or replace this dependency",
					Confidence:  1.0,
					Timestamp:   time.Now(),
				})
			}
		}
	}
	
	// Check for vulnerable packages (simplified patterns)
	vulnerablePatterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`"lodash":\s*"[<^~]*[0-3]\."`, "Vulnerable lodash version", 0.8},
		{`"express":\s*"[<^~]*[0-3]\."`, "Potentially vulnerable express version", 0.7},
		{`"moment":\s*"`, "Moment.js is deprecated", 0.6},
		{`"jquery":\s*"[<^~]*[0-2]\."`, "Potentially vulnerable jQuery version", 0.7},
		{`"axios":\s*"[<^~]*0\.[0-1][0-9]\."`, "Potentially vulnerable axios version", 0.7},
		{`"ws":\s*"[<^~]*[0-6]\."`, "Potentially vulnerable ws version", 0.7},
		{`"node-fetch":\s*"[<^~]*[0-1]\."`, "Potentially vulnerable node-fetch version", 0.8},
		{`"yargs-parser":\s*"[<^~]*[0-1][0-7]\."`, "Vulnerable yargs-parser version", 0.8},
	}
	
	contentStr := string(content)
	for _, pattern := range vulnerablePatterns {
		if regexp.MustCompile(pattern.pattern).MatchString(contentStr) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeDependency,
				Severity:    models.SeverityWarning,
				Title:       "Potentially vulnerable dependency",
				Description: pattern.description,
				File:        filename,
				Line:        1,
				Rule:        "vulnerable-dependency",
				Message:     "Update to a secure version or find an alternative",
				Confidence:  pattern.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check for outdated version patterns
	outdatedPatterns := []struct {
		pattern     string
		description string
		confidence  float64
	}{
		{`"\^?[0-2]\."`, "Major version 0-2 (might be outdated)", 0.5},
		{`"~[0-9]+\.[0-9]+\.[0-9]+"`, "Tilde range (restrictive)", 0.6},
		{`"[0-9]+\.[0-9]+\.[0-9]+"`, "Exact version pinning", 0.7},
		{`"[*x]"`, "Wildcard version (dangerous)", 0.9},
		{`"latest"`, "Latest tag (unpredictable)", 0.8},
	}
	
	for _, pattern := range outdatedPatterns {
		if regexp.MustCompile(pattern.pattern).MatchString(contentStr) {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeDependency,
				Severity:    models.SeverityInfo,
				Title:       "Dependency versioning issue",
				Description: pattern.description,
				File:        filename,
				Line:        1,
				Rule:        "dependency-versioning",
				Message:     "Review dependency versioning strategy",
				Confidence:  pattern.confidence,
				Timestamp:   time.Now(),
			})
		}
	}
	
	return issues, nil
}

func (dc *DependencyChecker) checkPythonProject(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	for _, file := range files {
		baseName := filepath.Base(file)
		
		if baseName == "requirements.txt" {
			reqIssues, err := dc.checkRequirementsTxt(file)
			if err != nil {
				return nil, fmt.Errorf("failed to check requirements.txt: %w", err)
			}
			issues = append(issues, reqIssues...)
		} else if baseName == "Pipfile" || baseName == "pyproject.toml" {
			pipeIssues, err := dc.checkPythonDependencyFile(file)
			if err != nil {
				return nil, fmt.Errorf("failed to check Python dependency file: %w", err)
			}
			issues = append(issues, pipeIssues...)
		}
	}
	
	return issues, nil
}

func (dc *DependencyChecker) checkRequirementsTxt(filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open requirements.txt: %w", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	dependencies := make(map[string]bool)
	
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Extract package name
		packageName := strings.Split(line, "==")[0]
		packageName = strings.Split(packageName, ">=")[0]
		packageName = strings.Split(packageName, "<=")[0]
		packageName = strings.Split(packageName, ">")[0]
		packageName = strings.Split(packageName, "<")[0]
		packageName = strings.Split(packageName, "~=")[0]
		packageName = strings.TrimSpace(packageName)
		
		// Check for duplicates
		if dependencies[packageName] {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeDependency,
				Severity:    models.SeverityWarning,
				Title:       "Duplicate dependency",
				Description: fmt.Sprintf("Package '%s' is listed multiple times", packageName),
				File:        filename,
				Line:        lineNumber,
				Rule:        "duplicate-dependency",
				Message:     "Remove duplicate dependency entries",
				Confidence:  1.0,
				Timestamp:   time.Now(),
			})
		}
		dependencies[packageName] = true
		
		// Check for potentially vulnerable packages
		vulnerablePackages := []string{
			"requests", "urllib3", "pyyaml", "jinja2", "django",
			"flask", "pillow", "numpy", "pandas", "scipy",
		}
		
		for _, vulnerable := range vulnerablePackages {
			if packageName == vulnerable && !strings.Contains(line, ">=") {
				issues = append(issues, models.Issue{
					Type:        models.VibeTypeDependency,
					Severity:    models.SeverityInfo,
					Title:       "Potentially outdated dependency",
					Description: fmt.Sprintf("Package '%s' should specify minimum version", packageName),
					File:        filename,
					Line:        lineNumber,
					Rule:        "outdated-dependency",
					Message:     "Specify minimum version for security updates",
					Confidence:  0.6,
					Timestamp:   time.Now(),
				})
			}
		}
		
		// Check for exact version pinning
		if strings.Contains(line, "==") {
			issues = append(issues, models.Issue{
				Type:        models.VibeTypeDependency,
				Severity:    models.SeverityInfo,
				Title:       "Exact version pinning",
				Description: fmt.Sprintf("Package '%s' is pinned to exact version", packageName),
				File:        filename,
				Line:        lineNumber,
				Rule:        "version-pinning",
				Message:     "Consider using version ranges for flexibility",
				Confidence:  0.5,
				Timestamp:   time.Now(),
			})
		}
	}
	
	// Check total dependency count
	if len(dependencies) > dc.maxDependencyCount {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDependency,
			Severity:    models.SeverityWarning,
			Title:       "Too many dependencies",
			Description: fmt.Sprintf("Project has %d dependencies (max recommended: %d)", len(dependencies), dc.maxDependencyCount),
			File:        filename,
			Line:        1,
			Rule:        "dependency-count",
			Message:     "Consider reducing the number of dependencies",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read requirements.txt: %w", err)
	}
	
	return issues, nil
}

func (dc *DependencyChecker) checkGoProject(ctx context.Context, files []string) ([]models.Issue, error) {
	var issues []models.Issue
	
	for _, file := range files {
		baseName := filepath.Base(file)
		
		if baseName == "go.mod" {
			goModIssues, err := dc.checkGoMod(file)
			if err != nil {
				return nil, fmt.Errorf("failed to check go.mod: %w", err)
			}
			issues = append(issues, goModIssues...)
		}
	}
	
	return issues, nil
}

func (dc *DependencyChecker) checkGoMod(filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	dependencies := make(map[string]bool)
	
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		
		// Check for require statements
		if strings.HasPrefix(line, "require") && strings.Contains(line, "v") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				dep := parts[1]
				version := parts[2]
				
				// Check for duplicates
				if dependencies[dep] {
					issues = append(issues, models.Issue{
						Type:        models.VibeTypeDependency,
						Severity:    models.SeverityWarning,
						Title:       "Duplicate dependency",
						Description: fmt.Sprintf("Dependency '%s' is listed multiple times", dep),
						File:        filename,
						Line:        lineNumber,
						Rule:        "duplicate-dependency",
						Message:     "Remove duplicate dependency entries",
						Confidence:  1.0,
						Timestamp:   time.Now(),
					})
				}
				dependencies[dep] = true
				
				// Check for pre-release versions
				if strings.Contains(version, "-") && (strings.Contains(version, "alpha") || strings.Contains(version, "beta") || strings.Contains(version, "rc")) {
					issues = append(issues, models.Issue{
						Type:        models.VibeTypeDependency,
						Severity:    models.SeverityInfo,
						Title:       "Pre-release dependency",
						Description: fmt.Sprintf("Dependency '%s' uses pre-release version '%s'", dep, version),
						File:        filename,
						Line:        lineNumber,
						Rule:        "prerelease-dependency",
						Message:     "Consider using stable versions in production",
						Confidence:  0.8,
						Timestamp:   time.Now(),
					})
				}
				
				// Check for very old versions (simplified)
				if strings.HasPrefix(version, "v0.") {
					issues = append(issues, models.Issue{
						Type:        models.VibeTypeDependency,
						Severity:    models.SeverityInfo,
						Title:       "Pre-1.0 dependency",
						Description: fmt.Sprintf("Dependency '%s' is at version '%s' (pre-1.0)", dep, version),
						File:        filename,
						Line:        lineNumber,
						Rule:        "immature-dependency",
						Message:     "Pre-1.0 dependencies may have breaking changes",
						Confidence:  0.6,
						Timestamp:   time.Now(),
					})
				}
			}
		}
	}
	
	// Check total dependency count
	if len(dependencies) > dc.maxDependencyCount {
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDependency,
			Severity:    models.SeverityWarning,
			Title:       "Too many dependencies",
			Description: fmt.Sprintf("Project has %d dependencies (max recommended: %d)", len(dependencies), dc.maxDependencyCount),
			File:        filename,
			Line:        1,
			Rule:        "dependency-count",
			Message:     "Consider reducing the number of dependencies",
			Confidence:  0.8,
			Timestamp:   time.Now(),
		})
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read go.mod: %w", err)
	}
	
	return issues, nil
}

func (dc *DependencyChecker) checkRustProject(ctx context.Context, files []string) ([]models.Issue, error) {
	// Implementation for Rust projects would go here
	// For now, return empty slice
	return []models.Issue{}, nil
}

func (dc *DependencyChecker) checkPythonDependencyFile(filename string) ([]models.Issue, error) {
	// Implementation for Pipfile and pyproject.toml would go here
	// For now, return empty slice
	return []models.Issue{}, nil
}

func (dc *DependencyChecker) checkLockFile(filename, projectType string) ([]models.Issue, error) {
	// Implementation for lock file analysis would go here
	// For now, return empty slice
	return []models.Issue{}, nil
}

func (dc *DependencyChecker) checkGenericDependencyFile(ctx context.Context, filename string) ([]models.Issue, error) {
	var issues []models.Issue
	
	// Basic checks that apply to any dependency file
	info, err := os.Stat(filename)
	if err != nil {
		return issues, nil // Skip if we can't stat the file
	}
	
	// Check for very large dependency files
	if info.Size() > 1024*1024 { // 1MB
		issues = append(issues, models.Issue{
			Type:        models.VibeTypeDependency,
			Severity:    models.SeverityWarning,
			Title:       "Large dependency file",
			Description: fmt.Sprintf("Dependency file is %d bytes", info.Size()),
			File:        filename,
			Line:        1,
			Rule:        "large-dependency-file",
			Message:     "Large dependency files may indicate too many dependencies",
			Confidence:  0.6,
			Timestamp:   time.Now(),
		})
	}
	
	return issues, nil
}