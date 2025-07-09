package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/kooshapari/kodevibe-go/internal/config"
	"github.com/kooshapari/kodevibe-go/internal/models"
	"github.com/kooshapari/kodevibe-go/pkg/vibes"
)

// CheckerInfo represents information about a checker
type CheckerInfo struct {
	Name         string   `json:"name" yaml:"name"`
	Type         string   `json:"type" yaml:"type"`
	Description  string   `json:"description,omitempty" yaml:"description,omitempty"`
	Version      string   `json:"version,omitempty" yaml:"version,omitempty"`
	Enabled      bool     `json:"enabled" yaml:"enabled"`
	Languages    []string `json:"languages,omitempty" yaml:"languages,omitempty"`
	FileTypes    []string `json:"file_types,omitempty" yaml:"file_types,omitempty"`
	ConfigKeys   []string `json:"config_keys,omitempty" yaml:"config_keys,omitempty"`
}

func newCheckersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkers",
		Short: "Manage and inspect code checkers",
		Long: `The checkers command allows you to:
• List all available checkers
• Get detailed information about specific checkers
• View and modify checker configurations
• Enable or disable checkers

Examples:
  kodevibe checkers list                    # List all checkers
  kodevibe checkers info security          # Get info about security checker
  kodevibe checkers config security        # Show security checker config
  kodevibe checkers enable security        # Enable security checker
  kodevibe checkers disable performance    # Disable performance checker`,
	}

	cmd.AddCommand(
		newCheckersListCommand(),
		newCheckersInfoCommand(),
		newCheckersConfigCommand(),
		newCheckersEnableCommand(),
		newCheckersDisableCommand(),
	)

	return cmd
}

func newCheckersListCommand() *cobra.Command {
	var (
		enabledOnly  bool
		disabledOnly bool
		typeFilter   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available checkers",
		Long: `List all available code checkers with their status and basic information.

Examples:
  kodevibe checkers list                    # List all checkers
  kodevibe checkers list --enabled-only    # List only enabled checkers
  kodevibe checkers list --type security   # List only security checkers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckersList(enabledOnly, disabledOnly, typeFilter)
		},
	}

	cmd.Flags().BoolVar(&enabledOnly, "enabled-only", false, "show only enabled checkers")
	cmd.Flags().BoolVar(&disabledOnly, "disabled-only", false, "show only disabled checkers")
	cmd.Flags().StringVar(&typeFilter, "type", "", "filter by checker type (security, performance, etc.)")

	return cmd
}

func newCheckersInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <checker-name>",
		Short: "Get detailed information about a specific checker",
		Long: `Display detailed information about a specific checker including:
• Description and purpose
• Supported file types and languages
• Configuration options
• Version information

Examples:
  kodevibe checkers info security          # Info about security checker
  kodevibe checkers info performance       # Info about performance checker`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckersInfo(args[0])
		},
	}

	return cmd
}

func newCheckersConfigCommand() *cobra.Command {
	var (
		set    []string
		unset  []string
		reset  bool
		edit   bool
	)

	cmd := &cobra.Command{
		Use:   "config <checker-name>",
		Short: "View or modify checker configuration",
		Long: `View or modify the configuration of a specific checker.

Examples:
  kodevibe checkers config security                    # Show security checker config
  kodevibe checkers config security --set key=value   # Set a configuration value
  kodevibe checkers config security --unset key       # Remove a configuration value
  kodevibe checkers config security --reset           # Reset to default config`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckersConfig(args[0], set, unset, reset, edit)
		},
	}

	cmd.Flags().StringSliceVar(&set, "set", []string{}, "set configuration value (key=value)")
	cmd.Flags().StringSliceVar(&unset, "unset", []string{}, "unset configuration key")
	cmd.Flags().BoolVar(&reset, "reset", false, "reset configuration to defaults")
	cmd.Flags().BoolVar(&edit, "edit", false, "open configuration in editor")

	return cmd
}

func newCheckersEnableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "enable <checker-name>",
		Short: "Enable a specific checker",
		Long: `Enable a specific checker so it will be included in scans by default.

Examples:
  kodevibe checkers enable security        # Enable security checker
  kodevibe checkers enable all            # Enable all checkers`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckersEnable(args[0])
		},
	}

	return cmd
}

func newCheckersDisableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disable <checker-name>",
		Short: "Disable a specific checker",
		Long: `Disable a specific checker so it will not be included in scans by default.

Examples:
  kodevibe checkers disable performance    # Disable performance checker
  kodevibe checkers disable all           # Disable all checkers`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCheckersDisable(args[0])
		},
	}

	return cmd
}

func runCheckersList(enabledOnly, disabledOnly bool, typeFilter string) error {
	// Initialize registry
	registry := vibes.NewRegistry()
	cfg := config.Default()

	// Register checkers
	if err := registerAllCheckers(registry); err != nil {
		return fmt.Errorf("failed to register checkers: %w", err)
	}

	// Get checker information
	checkers := registry.GetAllCheckers()
	checkersInfo := make([]CheckerInfo, 0, len(checkers))

	for _, checker := range checkers {
		info := CheckerInfo{
			Name:    checker.Name(),
			Type:    string(checker.Type()),
			Enabled: isCheckerEnabled(checker.Name(), cfg),
		}

		// Get description if available
		if desc, ok := checker.(interface{ Description() string }); ok {
			info.Description = desc.Description()
		}

		// Get version if available
		if ver, ok := checker.(interface{ Version() string }); ok {
			info.Version = ver.Version()
		}

		// Get supported languages if available
		if lang, ok := checker.(interface{ SupportedLanguages() []string }); ok {
			info.Languages = lang.SupportedLanguages()
		}

		// Filter by enabled/disabled status
		if enabledOnly && !info.Enabled {
			continue
		}
		if disabledOnly && info.Enabled {
			continue
		}

		// Filter by type
		if typeFilter != "" && !strings.EqualFold(info.Type, typeFilter) {
			continue
		}

		checkersInfo = append(checkersInfo, info)
	}

	// Sort by name
	sort.Slice(checkersInfo, func(i, j int) bool {
		return checkersInfo[i].Name < checkersInfo[j].Name
	})

	// Output based on format
	switch outputFormat {
	case "json":
		return outputCheckersJSON(checkersInfo)
	case "yaml":
		return outputCheckersYAML(checkersInfo)
	default:
		return outputCheckersTable(checkersInfo)
	}
}

func runCheckersInfo(checkerName string) error {
	// Initialize registry
	registry := vibes.NewRegistry()
	cfg := config.Default()

	// Register checkers
	if err := registerAllCheckers(registry); err != nil {
		return fmt.Errorf("failed to register checkers: %w", err)
	}

	// Find checker
	checker, err := registry.GetChecker(models.VibeType(checkerName))
	if err != nil {
		return fmt.Errorf("checker '%s' not found: %w", checkerName, err)
	}

	// Build detailed info
	info := CheckerInfo{
		Name:    checker.Name(),
		Type:    string(checker.Type()),
		Enabled: isCheckerEnabled(checker.Name(), cfg),
	}

	if desc, ok := checker.(interface{ Description() string }); ok {
		info.Description = desc.Description()
	}

	if ver, ok := checker.(interface{ Version() string }); ok {
		info.Version = ver.Version()
	}

	if lang, ok := checker.(interface{ SupportedLanguages() []string }); ok {
		info.Languages = lang.SupportedLanguages()
	}

	// Get configuration keys if available
	if checkerConfig, exists := cfg.Vibes.CheckerConfigs[checker.Name()]; exists {
		for key := range checkerConfig.Settings {
			info.ConfigKeys = append(info.ConfigKeys, key)
		}
		sort.Strings(info.ConfigKeys)
	}

	// Output detailed information
	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(info)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer encoder.Close()
		return encoder.Encode(info)
	default:
		return outputCheckerInfoTable(info)
	}
}

func runCheckersConfig(checkerName string, set, unset []string, reset, edit bool) error {
	// Load current configuration
	cfg := config.Default()

	// Initialize registry to validate checker exists
	registry := vibes.NewRegistry()
	if err := registerAllCheckers(registry); err != nil {
		return fmt.Errorf("failed to register checkers: %w", err)
	}

	_, err := registry.GetChecker(models.VibeType(checkerName))
	if err != nil {
		return fmt.Errorf("checker '%s' not found: %w", checkerName, err)
	}

	// Get current config
	checkerConfig := config.CheckerConfig{
		Enabled:  true,
		Settings: make(map[string]interface{}),
	}

	if existing, exists := cfg.Vibes.CheckerConfigs[checkerName]; exists {
		checkerConfig = existing
	}

	// Handle modifications
	if reset {
		checkerConfig = config.CheckerConfig{
			Enabled:  true,
			Settings: make(map[string]interface{}),
		}
		fmt.Printf("Reset configuration for checker '%s'\n", checkerName)
	}

	// Handle set operations
	for _, setting := range set {
		parts := strings.SplitN(setting, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid setting format: %s (expected key=value)", setting)
		}
		
		key, value := parts[0], parts[1]
		
		// Try to parse as different types
		if value == "true" || value == "false" {
			checkerConfig.Settings[key] = value == "true"
		} else if intVal, err := fmt.Sscanf(value, "%d", new(int)); err == nil && intVal == 1 {
			var i int
			fmt.Sscanf(value, "%d", &i)
			checkerConfig.Settings[key] = i
		} else if floatVal, err := fmt.Sscanf(value, "%f", new(float64)); err == nil && floatVal == 1 {
			var f float64
			fmt.Sscanf(value, "%f", &f)
			checkerConfig.Settings[key] = f
		} else {
			checkerConfig.Settings[key] = value
		}
		
		fmt.Printf("Set %s = %v for checker '%s'\n", key, checkerConfig.Settings[key], checkerName)
	}

	// Handle unset operations
	for _, key := range unset {
		if _, exists := checkerConfig.Settings[key]; exists {
			delete(checkerConfig.Settings, key)
			fmt.Printf("Unset %s for checker '%s'\n", key, checkerName)
		} else {
			fmt.Printf("Key %s not found for checker '%s'\n", key, checkerName)
		}
	}

	// Update configuration
	cfg.Vibes.CheckerConfigs[checkerName] = checkerConfig

	// If no modifications, just show current config
	if len(set) == 0 && len(unset) == 0 && !reset {
		switch outputFormat {
		case "json":
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(checkerConfig)
		case "yaml":
			encoder := yaml.NewEncoder(os.Stdout)
			defer encoder.Close()
			return encoder.Encode(checkerConfig)
		default:
			fmt.Printf("Configuration for checker '%s':\n", checkerName)
			fmt.Printf("  Enabled: %v\n", checkerConfig.Enabled)
			if len(checkerConfig.Settings) > 0 {
				fmt.Printf("  Settings:\n")
				for key, value := range checkerConfig.Settings {
					fmt.Printf("    %s: %v\n", key, value)
				}
			} else {
				fmt.Printf("  Settings: (none)\n")
			}
		}
	}

	return nil
}

func runCheckersEnable(checkerName string) error {
	cfg := config.Default()

	if checkerName == "all" {
		// Enable all checkers
		registry := vibes.NewRegistry()
		if err := registerAllCheckers(registry); err != nil {
			return fmt.Errorf("failed to register checkers: %w", err)
		}

		checkers := registry.GetAllCheckers()
		enabledNames := make([]string, 0, len(checkers))
		for _, checker := range checkers {
			enabledNames = append(enabledNames, checker.Name())
		}
		cfg.Vibes.EnabledCheckers = enabledNames
		fmt.Printf("Enabled all %d checkers\n", len(enabledNames))
	} else {
		// Enable specific checker
		registry := vibes.NewRegistry()
		if err := registerAllCheckers(registry); err != nil {
			return fmt.Errorf("failed to register checkers: %w", err)
		}

		_, err := registry.GetChecker(models.VibeType(checkerName))
		if err != nil {
			return fmt.Errorf("checker '%s' not found: %w", checkerName, err)
		}

		// Add to enabled checkers if not already present
		found := false
		for _, name := range cfg.Vibes.EnabledCheckers {
			if name == checkerName {
				found = true
				break
			}
		}

		if !found {
			cfg.Vibes.EnabledCheckers = append(cfg.Vibes.EnabledCheckers, checkerName)
		}

		// Ensure checker config exists and is enabled
		checkerConfig := cfg.Vibes.CheckerConfigs[checkerName]
		checkerConfig.Enabled = true
		cfg.Vibes.CheckerConfigs[checkerName] = checkerConfig

		fmt.Printf("Enabled checker '%s'\n", checkerName)
	}

	return nil
}

func runCheckersDisable(checkerName string) error {
	cfg := config.Default()

	if checkerName == "all" {
		// Disable all checkers
		cfg.Vibes.EnabledCheckers = []string{}
		for name, checkerConfig := range cfg.Vibes.CheckerConfigs {
			checkerConfig.Enabled = false
			cfg.Vibes.CheckerConfigs[name] = checkerConfig
		}
		fmt.Printf("Disabled all checkers\n")
	} else {
		// Disable specific checker
		registry := vibes.NewRegistry()
		if err := registerAllCheckers(registry); err != nil {
			return fmt.Errorf("failed to register checkers: %w", err)
		}

		_, err := registry.GetChecker(models.VibeType(checkerName))
		if err != nil {
			return fmt.Errorf("checker '%s' not found: %w", checkerName, err)
		}

		// Remove from enabled checkers
		newEnabled := []string{}
		for _, name := range cfg.Vibes.EnabledCheckers {
			if name != checkerName {
				newEnabled = append(newEnabled, name)
			}
		}
		cfg.Vibes.EnabledCheckers = newEnabled

		// Set checker config to disabled
		checkerConfig := cfg.Vibes.CheckerConfigs[checkerName]
		checkerConfig.Enabled = false
		cfg.Vibes.CheckerConfigs[checkerName] = checkerConfig

		fmt.Printf("Disabled checker '%s'\n", checkerName)
	}

	return nil
}

// Helper functions
func registerAllCheckers(registry *vibes.Registry) error {
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

func isCheckerEnabled(name string, cfg *config.Config) bool {
	for _, enabledName := range cfg.Vibes.EnabledCheckers {
		if enabledName == name {
			return true
		}
	}
	return false
}

func outputCheckersJSON(checkers []CheckerInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(checkers)
}

func outputCheckersYAML(checkers []CheckerInfo) error {
	encoder := yaml.NewEncoder(os.Stdout)
	defer encoder.Close()
	return encoder.Encode(checkers)
}

func outputCheckersTable(checkers []CheckerInfo) error {
	if len(checkers) == 0 {
		fmt.Println("No checkers found")
		return nil
	}

	fmt.Printf("Available Checkers (%d total):\n", len(checkers))
	fmt.Printf("%-20s %-12s %-8s %s\n", "NAME", "TYPE", "ENABLED", "DESCRIPTION")
	fmt.Printf("%-20s %-12s %-8s %s\n", strings.Repeat("-", 20), strings.Repeat("-", 12), strings.Repeat("-", 8), strings.Repeat("-", 40))

	for _, checker := range checkers {
		enabled := "No"
		if checker.Enabled {
			enabled = "Yes"
		}

		description := checker.Description
		if len(description) > 40 {
			description = description[:37] + "..."
		}

		fmt.Printf("%-20s %-12s %-8s %s\n", checker.Name, checker.Type, enabled, description)
	}

	return nil
}

func outputCheckerInfoTable(info CheckerInfo) error {
	fmt.Printf("Checker Information: %s\n", info.Name)
	fmt.Printf("====================%s\n", strings.Repeat("=", len(info.Name)))
	fmt.Printf("\n")
	fmt.Printf("Name:        %s\n", info.Name)
	fmt.Printf("Type:        %s\n", info.Type)
	fmt.Printf("Enabled:     %v\n", info.Enabled)
	
	if info.Version != "" {
		fmt.Printf("Version:     %s\n", info.Version)
	}
	
	if info.Description != "" {
		fmt.Printf("Description: %s\n", info.Description)
	}
	
	if len(info.Languages) > 0 {
		fmt.Printf("Languages:   %s\n", strings.Join(info.Languages, ", "))
	}
	
	if len(info.ConfigKeys) > 0 {
		fmt.Printf("Config Keys: %s\n", strings.Join(info.ConfigKeys, ", "))
	}

	return nil
}