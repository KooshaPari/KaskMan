package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/kooshapari/kodevibe-go/internal/config"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage KodeVibe configuration",
		Long: `The config command allows you to:
• View current configuration
• Set and get configuration values
• Initialize default configuration
• Validate configuration files

Examples:
  kodevibe config show                      # Show current configuration
  kodevibe config get server.port          # Get specific value
  kodevibe config set server.port 9090     # Set specific value
  kodevibe config init                      # Create default config file
  kodevibe config validate                  # Validate configuration`,
	}

	cmd.AddCommand(
		newConfigShowCommand(),
		newConfigGetCommand(),
		newConfigSetCommand(),
		newConfigUnsetCommand(),
		newConfigInitCommand(),
		newConfigValidateCommand(),
		newConfigEditCommand(),
	)

	return cmd
}

func newConfigShowCommand() *cobra.Command {
	var (
		section string
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long: `Display the current configuration. You can optionally specify a section to show.

Examples:
  kodevibe config show                      # Show full configuration
  kodevibe config show --section server    # Show only server configuration
  kodevibe config show --section vibes     # Show only vibes configuration`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigShow(section)
		},
	}

	cmd.Flags().StringVar(&section, "section", "", "show only specific section (server, scanner, vibes, etc.)")

	return cmd
}

func newConfigGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long: `Get a specific configuration value using dot notation.

Examples:
  kodevibe config get server.port          # Get server port
  kodevibe config get vibes.max_issues     # Get max issues setting
  kodevibe config get scanner.max_files    # Get max files setting`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigGet(args[0])
		},
	}

	return cmd
}

func newConfigSetCommand() *cobra.Command {
	var (
		persist bool
	)

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long: `Set a specific configuration value using dot notation.

Examples:
  kodevibe config set server.port 9090     # Set server port
  kodevibe config set vibes.max_issues 100 # Set max issues
  kodevibe config set --persist server.host 0.0.0.0  # Set and save to file`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigSet(args[0], args[1], persist)
		},
	}

	cmd.Flags().BoolVar(&persist, "persist", false, "save changes to configuration file")

	return cmd
}

func newConfigUnsetCommand() *cobra.Command {
	var (
		persist bool
	)

	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Unset a configuration value",
		Long: `Remove a configuration value, reverting it to the default.

Examples:
  kodevibe config unset server.port        # Unset server port
  kodevibe config unset --persist vibes.max_issues  # Unset and save`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigUnset(args[0], persist)
		},
	}

	cmd.Flags().BoolVar(&persist, "persist", false, "save changes to configuration file")

	return cmd
}

func newConfigInitCommand() *cobra.Command {
	var (
		force bool
		path  string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize default configuration",
		Long: `Create a default configuration file in the current directory or home directory.

Examples:
  kodevibe config init                      # Create .kodevibe.yaml in home dir
  kodevibe config init --path ./config.yaml # Create config at specific path
  kodevibe config init --force             # Overwrite existing config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigInit(path, force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing configuration file")
	cmd.Flags().StringVar(&path, "path", "", "path for configuration file (default: ~/.kodevibe.yaml)")

	return cmd
}

func newConfigValidateCommand() *cobra.Command {
	var (
		configPath string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		Long: `Validate a configuration file for syntax and logical errors.

Examples:
  kodevibe config validate                  # Validate current config
  kodevibe config validate --config ./config.yaml  # Validate specific file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigValidate(configPath)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "path to configuration file to validate")

	return cmd
}

func newConfigEditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration in default editor",
		Long: `Open the configuration file in your default editor (defined by $EDITOR).

Examples:
  kodevibe config edit                      # Edit current config file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigEdit()
		},
	}

	return cmd
}

func runConfigShow(section string) error {
	cfg := config.Default()

	// Load current config if available
	if viper.ConfigFileUsed() != "" {
		if err := viper.Unmarshal(cfg); err != nil {
			return fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	var output interface{} = cfg

	// Filter by section if specified
	if section != "" {
		switch strings.ToLower(section) {
		case "server":
			output = cfg.Server
		case "scanner":
			output = cfg.Scanner
		case "vibes":
			output = cfg.Vibes
		case "database":
			output = cfg.Database
		case "logging":
			output = cfg.Logging
		case "security":
			output = cfg.Security
		default:
			return fmt.Errorf("unknown section: %s", section)
		}
	}

	// Output in requested format
	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer encoder.Close()
		return encoder.Encode(output)
	default:
		// Default to YAML for config display
		encoder := yaml.NewEncoder(os.Stdout)
		defer encoder.Close()
		return encoder.Encode(output)
	}
}

func runConfigGet(key string) error {
	initConfig()

	value := viper.Get(key)
	if value == nil {
		return fmt.Errorf("configuration key '%s' not found", key)
	}

	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{key: value})
	case "yaml":
		encoder := yaml.NewEncoder(os.Stdout)
		defer encoder.Close()
		return encoder.Encode(map[string]interface{}{key: value})
	default:
		fmt.Printf("%s: %v\n", key, value)
		return nil
	}
}

func runConfigSet(key, value string, persist bool) error {
	initConfig()

	// Parse value to appropriate type
	var parsedValue interface{}
	switch strings.ToLower(value) {
	case "true":
		parsedValue = true
	case "false":
		parsedValue = false
	default:
		// Try to parse as number
		var intVal int
		var floatVal float64
		if n, err := fmt.Sscanf(value, "%d", &intVal); err == nil && n == 1 {
			parsedValue = intVal
		} else if n, err := fmt.Sscanf(value, "%f", &floatVal); err == nil && n == 1 {
			parsedValue = floatVal
		} else {
			parsedValue = value
		}
	}

	viper.Set(key, parsedValue)

	if persist {
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			// Create default config file
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			configFile = filepath.Join(home, ".kodevibe.yaml")
		}

		if err := viper.WriteConfigAs(configFile); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		fmt.Printf("Set %s = %v (saved to %s)\n", key, parsedValue, configFile)
	} else {
		fmt.Printf("Set %s = %v (session only)\n", key, parsedValue)
	}

	return nil
}

func runConfigUnset(key string, persist bool) error {
	initConfig()

	// Check if key exists
	if !viper.IsSet(key) {
		return fmt.Errorf("configuration key '%s' not found", key)
	}

	// Remove the key
	viper.Set(key, nil)

	if persist {
		configFile := viper.ConfigFileUsed()
		if configFile == "" {
			return fmt.Errorf("no configuration file loaded")
		}

		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		fmt.Printf("Unset %s (saved to %s)\n", key, configFile)
	} else {
		fmt.Printf("Unset %s (session only)\n", key)
	}

	return nil
}

func runConfigInit(path string, force bool) error {
	// Determine config file path
	configPath := path
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, ".kodevibe.yaml")
	}

	// Check if file exists
	if _, err := os.Stat(configPath); err == nil && !force {
		return fmt.Errorf("configuration file already exists at %s (use --force to overwrite)", configPath)
	}

	// Create default configuration
	cfg := config.Default()

	// Save to file
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("Created default configuration file: %s\n", configPath)
	fmt.Printf("You can now edit it or use 'kodevibe config set' to modify values.\n")

	return nil
}

func runConfigValidate(configPath string) error {
	// Use provided path or find existing config
	if configPath == "" {
		initConfig()
		configPath = viper.ConfigFileUsed()
		if configPath == "" {
			return fmt.Errorf("no configuration file found")
		}
	}

	// Load and validate configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	fmt.Printf("Configuration file %s is valid\n", configPath)

	// Print summary information
	fmt.Printf("\nConfiguration Summary:\n")
	fmt.Printf("  Server port: %d\n", cfg.Server.Port)
	fmt.Printf("  Enabled checkers: %d\n", len(cfg.Vibes.EnabledCheckers))
	fmt.Printf("  Max file size: %d bytes\n", cfg.Scanner.MaxFileSize)
	fmt.Printf("  Max issues: %d\n", cfg.Vibes.MaxIssues)
	fmt.Printf("  Output format: %s\n", cfg.Vibes.OutputFormat)

	return nil
}

func runConfigEdit() error {
	initConfig()
	configFile := viper.ConfigFileUsed()
	
	if configFile == "" {
		return fmt.Errorf("no configuration file found (use 'kodevibe config init' to create one)")
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		// Try common editors
		editors := []string{"nano", "vim", "vi", "emacs", "code"}
		for _, e := range editors {
			if _, err := os.Stat("/usr/bin/" + e); err == nil {
				editor = e
				break
			}
		}
		
		if editor == "" {
			return fmt.Errorf("no editor found (set $EDITOR environment variable)")
		}
	}

	fmt.Printf("Opening %s in %s...\n", configFile, editor)
	
	// This would execute the editor - simplified for example
	fmt.Printf("Editor command: %s %s\n", editor, configFile)
	fmt.Printf("(Editor execution not implemented in this example)\n")

	return nil
}