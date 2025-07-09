package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"github.com/sirupsen/logrus"

	"github.com/kooshapari/kodevibe-go/internal/config"
	"github.com/kooshapari/kodevibe-go/internal/models"
	"github.com/kooshapari/kodevibe-go/pkg/scanner"
	"github.com/kooshapari/kodevibe-go/pkg/vibes"
)

// Scan command flags
type scanFlags struct {
	paths             []string
	checkers          []string
	excludeCheckers   []string
	outputFile        string
	maxIssues         int
	minConfidence     float64
	severity          []string
	includePatterns   []string
	excludePatterns   []string
	maxFileSize       string
	maxDepth          int
	followSymlinks    bool
	timeout           time.Duration
	workers           int
	failOnIssues      bool
	statsOnly         bool
	dryRun            bool
}

func newScanCommand() *cobra.Command {
	flags := &scanFlags{}

	cmd := &cobra.Command{
		Use:   "scan [paths...]",
		Short: "Scan code for issues and vulnerabilities",
		Long: `Scan analyzes your code for various issues including:
• Security vulnerabilities (SQL injection, XSS, secrets, etc.)
• Performance issues (inefficient algorithms, memory leaks, etc.)
• Code quality problems (complexity, duplication, etc.)
• File organization issues (naming, structure, etc.)
• Git repository health problems
• Dependency management issues
• Documentation quality problems

Examples:
  kodevibe scan .                          # Scan current directory
  kodevibe scan src/ tests/                # Scan multiple directories
  kodevibe scan . --checkers security     # Run only security checker
  kodevibe scan . --output json           # Output in JSON format
  kodevibe scan . --min-confidence 0.8    # Only high-confidence issues
  kodevibe scan . --max-issues 50         # Limit output to 50 issues`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.paths = args
			return runScan(cmd, flags)
		},
	}

	// Add flags
	cmd.Flags().StringSliceVarP(&flags.checkers, "checkers", "c", []string{}, "checkers to run (comma-separated)")
	cmd.Flags().StringSliceVar(&flags.excludeCheckers, "exclude-checkers", []string{}, "checkers to exclude (comma-separated)")
	cmd.Flags().StringVarP(&flags.outputFile, "output-file", "f", "", "output file (default: stdout)")
	cmd.Flags().IntVar(&flags.maxIssues, "max-issues", 0, "maximum number of issues to report (0 = unlimited)")
	cmd.Flags().Float64Var(&flags.minConfidence, "min-confidence", 0.5, "minimum confidence level (0.0-1.0)")
	cmd.Flags().StringSliceVar(&flags.severity, "severity", []string{}, "filter by severity (critical, error, warning, info, hint)")
	cmd.Flags().StringSliceVar(&flags.includePatterns, "include", []string{}, "file patterns to include (glob patterns)")
	cmd.Flags().StringSliceVar(&flags.excludePatterns, "exclude", []string{}, "file patterns to exclude (glob patterns)")
	cmd.Flags().StringVar(&flags.maxFileSize, "max-file-size", "50MB", "maximum file size to scan")
	cmd.Flags().IntVar(&flags.maxDepth, "max-depth", 10, "maximum directory depth")
	cmd.Flags().BoolVar(&flags.followSymlinks, "follow-symlinks", false, "follow symbolic links")
	cmd.Flags().DurationVar(&flags.timeout, "timeout", 5*time.Minute, "scan timeout")
	cmd.Flags().IntVar(&flags.workers, "workers", 4, "number of concurrent workers")
	cmd.Flags().BoolVar(&flags.failOnIssues, "fail-on-issues", false, "exit with non-zero code if issues found")
	cmd.Flags().BoolVar(&flags.statsOnly, "stats-only", false, "only show statistics, not individual issues")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false, "show what would be scanned without actually scanning")

	return cmd
}

func runScan(cmd *cobra.Command, flags *scanFlags) error {
	// Load configuration
	cfg := loadScanConfig(flags)

	// Initialize components
	registry := vibes.NewRegistry()
	if err := registerCheckers(registry, cfg); err != nil {
		return fmt.Errorf("failed to register checkers: %w", err)
	}

	// Create scanner configuration
	scannerConfig := &models.Configuration{
		Scanner: models.ScannerConfig{
			MaxConcurrency:  flags.workers,
			Timeout:        int(flags.timeout.Seconds()),
			ExcludePatterns: append(cfg.Scanner.IgnorePatterns, flags.excludePatterns...),
			IncludePatterns: append(cfg.Scanner.IncludePatterns, flags.includePatterns...),
			EnabledVibes:    flags.checkers,
		},
		Vibes: make(map[models.VibeType]models.VibeConfig),
	}

	// Create logger
	logger := logrus.New()
	
	// Create scanner
	fileScanner, err := scanner.NewScanner(scannerConfig, logger)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), flags.timeout)
	defer cancel()

	// Scan for files
	if verbose {
		fmt.Fprintf(os.Stderr, "Scanning paths: %s\n", strings.Join(flags.paths, ", "))
	}

	// Create scan request
	scanRequest := &models.ScanRequest{
		Paths:      flags.paths,
		Vibes:      flags.checkers,
		StagedOnly: false,
		DiffTarget: "",
		Timeout:    int(flags.timeout.Seconds()),
	}

	// Perform scan
	result, err := fileScanner.Scan(ctx, scanRequest)
	if err != nil {
		return fmt.Errorf("failed to scan: %w", err)
	}

	// Get all issues
	allIssues := result.Issues
	uniqueFiles := []string{} // This would be extracted from result if needed

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d files to analyze\n", len(uniqueFiles))
	}

	// Dry run mode
	if flags.dryRun {
		return printDryRunResults(uniqueFiles, registry, flags)
	}

	// Get selected checkers
	selectedCheckers, err := getSelectedCheckers(registry, flags.checkers, flags.excludeCheckers, cfg)
	if err != nil {
		return fmt.Errorf("failed to get checkers: %w", err)
	}

	// Run analysis
	start := time.Now()
	allIssues := []models.Issue{}
	checkerStats := make(map[string]int)

	for _, checker := range selectedCheckers {
		if verbose {
			fmt.Fprintf(os.Stderr, "Running checker: %s\n", checker.Name())
		}

		// Configure checker
		if err := configureChecker(checker, cfg); err != nil {
			return fmt.Errorf("failed to configure checker %s: %w", checker.Name(), err)
		}

		// Run checker
		issues, err := checker.Check(ctx, uniqueFiles)
		if err != nil {
			return fmt.Errorf("checker %s failed: %w", checker.Name(), err)
		}

		// Filter issues
		filteredIssues := filterIssues(issues, flags)
		allIssues = append(allIssues, filteredIssues...)
		checkerStats[checker.Name()] = len(filteredIssues)
	}

	duration := time.Since(start)

	// Sort and limit issues
	sortedIssues := sortIssues(allIssues)
	if flags.maxIssues > 0 && len(sortedIssues) > flags.maxIssues {
		sortedIssues = sortedIssues[:flags.maxIssues]
	}

	// Generate output
	result := ScanResult{
		Issues:     sortedIssues,
		Statistics: generateStatistics(uniqueFiles, allIssues, checkerStats),
		Metadata: ScanMetadata{
			Timestamp: time.Now(),
			Duration:  duration,
			Version:   cmd.Root().Version,
			Paths:     flags.paths,
			Checkers:  getCheckerNames(selectedCheckers),
		},
	}

	// Output results
	if err := outputResults(result, flags); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	// Exit with appropriate code
	if flags.failOnIssues && len(sortedIssues) > 0 {
		os.Exit(1)
	}

	return nil
}

// ScanResult represents the complete scan results
type ScanResult struct {
	Issues     []models.Issue    `json:"issues" yaml:"issues"`
	Statistics ScanStatistics   `json:"statistics" yaml:"statistics"`
	Metadata   ScanMetadata     `json:"metadata" yaml:"metadata"`
}

// ScanStatistics represents scan statistics
type ScanStatistics struct {
	TotalFiles       int            `json:"total_files" yaml:"total_files"`
	ScannedFiles     int            `json:"scanned_files" yaml:"scanned_files"`
	TotalIssues      int            `json:"total_issues" yaml:"total_issues"`
	IssuesBySeverity map[string]int `json:"issues_by_severity" yaml:"issues_by_severity"`
	IssuesByType     map[string]int `json:"issues_by_type" yaml:"issues_by_type"`
	IssuesByChecker  map[string]int `json:"issues_by_checker" yaml:"issues_by_checker"`
}

// ScanMetadata represents scan metadata
type ScanMetadata struct {
	Timestamp time.Time     `json:"timestamp" yaml:"timestamp"`
	Duration  time.Duration `json:"duration" yaml:"duration"`
	Version   string        `json:"version" yaml:"version"`
	Paths     []string      `json:"paths" yaml:"paths"`
	Checkers  []string      `json:"checkers" yaml:"checkers"`
}

func loadScanConfig(flags *scanFlags) *config.Config {
	cfg := config.Default()

	// Override with viper settings
	if viper.IsSet("checkers") {
		cfg.Vibes.EnabledCheckers = viper.GetStringSlice("checkers")
	}
	if viper.IsSet("max_issues") {
		cfg.Vibes.MaxIssues = viper.GetInt("max_issues")
	}
	if viper.IsSet("min_confidence") {
		cfg.Vibes.MinConfidence = viper.GetFloat64("min_confidence")
	}

	// Override with flags
	if flags.maxIssues > 0 {
		cfg.Vibes.MaxIssues = flags.maxIssues
	}
	if flags.minConfidence != 0.5 {
		cfg.Vibes.MinConfidence = flags.minConfidence
	}

	return cfg
}

func registerCheckers(registry *vibes.Registry, cfg *config.Config) error {
	checkers := []vibes.Checker{
		vibes.NewCodeChecker(),
		vibes.NewSecurityChecker(),
		vibes.NewPerformanceChecker(),
		vibes.NewFileChecker(),
		vibes.NewGitChecker(),
		vibes.NewDependencyChecker(),
		vibes.NewDocumentationChecker(),
	}

	for _, checker := range checkers {
		if err := registry.RegisterChecker(checker); err != nil {
			return fmt.Errorf("failed to register %s: %w", checker.Name(), err)
		}
	}

	return nil
}

func getSelectedCheckers(registry *vibes.Registry, checkers, excludeCheckers []string, cfg *config.Config) ([]vibes.Checker, error) {
	selectedNames := checkers
	if len(selectedNames) == 0 {
		selectedNames = cfg.Vibes.EnabledCheckers
	}

	// Remove excluded checkers
	excludeMap := make(map[string]bool)
	for _, name := range excludeCheckers {
		excludeMap[name] = true
	}

	result := []vibes.Checker{}
	for _, name := range selectedNames {
		if excludeMap[name] {
			continue
		}

		checker := registry.GetChecker(name)
		if checker == nil {
			return nil, fmt.Errorf("checker '%s' not found", name)
		}
		result = append(result, checker)
	}

	return result, nil
}

func configureChecker(checker vibes.Checker, cfg *config.Config) error {
	checkerConfig := models.VibeConfig{
		Enabled:  true,
		Settings: make(map[string]interface{}),
	}

	if config, exists := cfg.Vibes.CheckerConfigs[checker.Name()]; exists {
		checkerConfig.Enabled = config.Enabled
		checkerConfig.Settings = config.Settings
	}

	if configurable, ok := checker.(interface{ Configure(models.VibeConfig) error }); ok {
		return configurable.Configure(checkerConfig)
	}

	return nil
}

func filterIssues(issues []models.Issue, flags *scanFlags) []models.Issue {
	filtered := []models.Issue{}

	severityMap := make(map[string]bool)
	for _, sev := range flags.severity {
		severityMap[strings.ToLower(sev)] = true
	}

	for _, issue := range issues {
		// Filter by confidence
		if issue.Confidence < flags.minConfidence {
			continue
		}

		// Filter by severity
		if len(severityMap) > 0 && !severityMap[strings.ToLower(string(issue.Severity))] {
			continue
		}

		filtered = append(filtered, issue)
	}

	return filtered
}

func sortIssues(issues []models.Issue) []models.Issue {
	// Sort by severity (critical first), then by confidence (high first)
	sorted := make([]models.Issue, len(issues))
	copy(sorted, issues)

	// Simple bubble sort for demonstration - would use sort.Slice in production
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if severityWeight(sorted[i].Severity) < severityWeight(sorted[j].Severity) ||
				(sorted[i].Severity == sorted[j].Severity && sorted[i].Confidence < sorted[j].Confidence) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

func severityWeight(severity models.SeverityLevel) int {
	switch severity {
	case models.SeverityCritical:
		return 5
	case models.SeverityError:
		return 4
	case models.SeverityWarning:
		return 3
	case models.SeverityInfo:
		return 2
	case models.SeverityHint:
		return 1
	default:
		return 0
	}
}

func generateStatistics(files []string, issues []models.Issue, checkerStats map[string]int) ScanStatistics {
	stats := ScanStatistics{
		TotalFiles:       len(files),
		ScannedFiles:     len(files),
		TotalIssues:      len(issues),
		IssuesBySeverity: make(map[string]int),
		IssuesByType:     make(map[string]int),
		IssuesByChecker:  checkerStats,
	}

	for _, issue := range issues {
		stats.IssuesBySeverity[string(issue.Severity)]++
		stats.IssuesByType[string(issue.Type)]++
	}

	return stats
}

func outputResults(result ScanResult, flags *scanFlags) error {
	var writer *os.File = os.Stdout
	var err error

	if flags.outputFile != "" {
		writer, err = os.Create(flags.outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer writer.Close()
	}

	if flags.statsOnly {
		return outputStatistics(writer, result.Statistics, result.Metadata)
	}

	switch outputFormat {
	case "json":
		return outputJSON(writer, result)
	case "yaml":
		return outputYAML(writer, result)
	case "csv":
		return outputCSV(writer, result.Issues)
	case "table":
		return outputTable(writer, result)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func outputJSON(writer *os.File, result ScanResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func outputYAML(writer *os.File, result ScanResult) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()
	return encoder.Encode(result)
}

func outputCSV(writer *os.File, issues []models.Issue) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	header := []string{"File", "Line", "Column", "Severity", "Type", "Rule", "Message", "Confidence"}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	// Write issues
	for _, issue := range issues {
		record := []string{
			issue.File,
			strconv.Itoa(issue.Line),
			strconv.Itoa(issue.Column),
			string(issue.Severity),
			string(issue.Type),
			issue.Rule,
			issue.Message,
			fmt.Sprintf("%.2f", issue.Confidence),
		}
		if err := csvWriter.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func outputTable(writer *os.File, result ScanResult) error {
	if !quiet {
		// Print summary first
		fmt.Fprintf(writer, "KodeVibe Scan Results\n")
		fmt.Fprintf(writer, "====================\n\n")
		fmt.Fprintf(writer, "Scanned %d files in %v\n", result.Statistics.TotalFiles, result.Metadata.Duration)
		fmt.Fprintf(writer, "Found %d issues\n\n", result.Statistics.TotalIssues)

		// Print statistics
		if len(result.Statistics.IssuesBySeverity) > 0 {
			fmt.Fprintf(writer, "Issues by Severity:\n")
			for severity, count := range result.Statistics.IssuesBySeverity {
				fmt.Fprintf(writer, "  %s: %d\n", severity, count)
			}
			fmt.Fprintf(writer, "\n")
		}

		// Print issues
		if len(result.Issues) > 0 {
			fmt.Fprintf(writer, "Issues:\n")
			fmt.Fprintf(writer, "-------\n")
			for i, issue := range result.Issues {
				fmt.Fprintf(writer, "%d. [%s] %s:%d:%d - %s\n",
					i+1, issue.Severity, issue.File, issue.Line, issue.Column, issue.Message)
				if verbose {
					fmt.Fprintf(writer, "   Rule: %s | Confidence: %.2f | Type: %s\n",
						issue.Rule, issue.Confidence, issue.Type)
				}
				fmt.Fprintf(writer, "\n")
			}
		}
	}

	return nil
}

func outputStatistics(writer *os.File, stats ScanStatistics, metadata ScanMetadata) error {
	fmt.Fprintf(writer, "Scan Statistics\n")
	fmt.Fprintf(writer, "===============\n\n")
	fmt.Fprintf(writer, "Files: %d scanned\n", stats.TotalFiles)
	fmt.Fprintf(writer, "Issues: %d found\n", stats.TotalIssues)
	fmt.Fprintf(writer, "Duration: %v\n", metadata.Duration)
	fmt.Fprintf(writer, "Checkers: %s\n", strings.Join(metadata.Checkers, ", "))
	fmt.Fprintf(writer, "\n")

	if len(stats.IssuesBySeverity) > 0 {
		fmt.Fprintf(writer, "By Severity:\n")
		for severity, count := range stats.IssuesBySeverity {
			fmt.Fprintf(writer, "  %s: %d\n", severity, count)
		}
		fmt.Fprintf(writer, "\n")
	}

	if len(stats.IssuesByChecker) > 0 {
		fmt.Fprintf(writer, "By Checker:\n")
		for checker, count := range stats.IssuesByChecker {
			fmt.Fprintf(writer, "  %s: %d\n", checker, count)
		}
	}

	return nil
}

func printDryRunResults(files []string, registry *vibes.Registry, flags *scanFlags) error {
	fmt.Printf("Dry run results:\n")
	fmt.Printf("================\n\n")
	fmt.Printf("Would scan %d files:\n", len(files))

	if verbose {
		for i, file := range files {
			if i < 10 { // Show first 10 files
				fmt.Printf("  %s\n", file)
			} else if i == 10 {
				fmt.Printf("  ... and %d more files\n", len(files)-10)
				break
			}
		}
		fmt.Printf("\n")
	}

	fmt.Printf("Would use checkers:\n")
	checkers := registry.GetCheckers()
	for _, checker := range checkers {
		fmt.Printf("  %s - %s\n", checker.Name(), string(checker.Type()))
	}

	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Max issues: %d\n", flags.maxIssues)
	fmt.Printf("  Min confidence: %.2f\n", flags.minConfidence)
	fmt.Printf("  Output format: %s\n", outputFormat)
	fmt.Printf("  Workers: %d\n", flags.workers)
	fmt.Printf("  Timeout: %v\n", flags.timeout)

	return nil
}

// Helper functions
func removeDuplicateFiles(files []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, file := range files {
		if !seen[file] {
			seen[file] = true
			result = append(result, file)
		}
	}
	return result
}

func getCheckerNames(checkers []vibes.Checker) []string {
	names := make([]string, len(checkers))
	for i, checker := range checkers {
		names[i] = checker.Name()
	}
	return names
}

func parseFileSize(sizeStr string) int64 {
	// Simple file size parser - would be more robust in production
	sizeStr = strings.ToUpper(strings.TrimSpace(sizeStr))
	
	if strings.HasSuffix(sizeStr, "KB") {
		size, _ := strconv.ParseInt(strings.TrimSuffix(sizeStr, "KB"), 10, 64)
		return size * 1024
	}
	if strings.HasSuffix(sizeStr, "MB") {
		size, _ := strconv.ParseInt(strings.TrimSuffix(sizeStr, "MB"), 10, 64)
		return size * 1024 * 1024
	}
	if strings.HasSuffix(sizeStr, "GB") {
		size, _ := strconv.ParseInt(strings.TrimSuffix(sizeStr, "GB"), 10, 64)
		return size * 1024 * 1024 * 1024
	}
	
	// Default to bytes
	size, _ := strconv.ParseInt(sizeStr, 10, 64)
	return size
}