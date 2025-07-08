package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewConfigCommand creates the config command
func NewConfigCommand(apiClient **client.Client) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management commands",
		Long:  "Commands for managing CLI and server configuration",
	}

	// Show command
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display current CLI and server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetBool("server")

			if server {
				return showServerConfig(*apiClient)
			}

			return showCliConfig()
		},
	}

	// Set command
	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  "Set a configuration value in the CLI config file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			return setCliConfig(key, value)
		},
	}

	// Get command
	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  "Get a configuration value from the CLI config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			return getCliConfig(key)
		},
	}

	// Init command
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize CLI configuration",
		Long:  "Initialize CLI configuration with default values",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			return initCliConfig(force)
		},
	}

	// Validate command
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long:  "Validate CLI and server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			server, _ := cmd.Flags().GetBool("server")

			if server {
				return validateServerConfig(*apiClient)
			}

			return validateCliConfig()
		},
	}

	// Edit command
	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit configuration file",
		Long:  "Open configuration file in default editor",
		RunE: func(cmd *cobra.Command, args []string) error {
			return editCliConfig()
		},
	}

	// Add flags
	showCmd.Flags().BoolP("server", "s", false, "Show server configuration")
	validateCmd.Flags().BoolP("server", "s", false, "Validate server configuration")
	initCmd.Flags().BoolP("force", "f", false, "Force initialization, overwrite existing config")

	// Add subcommands
	configCmd.AddCommand(showCmd)
	configCmd.AddCommand(setCmd)
	configCmd.AddCommand(getCmd)
	configCmd.AddCommand(initCmd)
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(editCmd)

	return configCmd
}

func showCliConfig() error {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	utils.PrintHeader("CLI Configuration")
	fmt.Printf("Config Directory: %s\n", configDir)
	fmt.Printf("Config File: %s\n", configFile)

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		utils.PrintWarning("Configuration file does not exist. Run 'kaskman config init' to create it.")
		return nil
	}

	// Load and display config
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	utils.PrintSubHeader("Current Settings")
	settings := viper.AllSettings()
	for key, value := range settings {
		fmt.Printf("%s: %v\n", key, value)
	}

	// Show auth status
	authConfig, err := utils.LoadAuthConfig()
	if err == nil {
		utils.PrintSubHeader("Authentication")
		fmt.Printf("Server URL: %s\n", authConfig.BaseURL)
		fmt.Printf("Username: %s\n", authConfig.User.Username)
		fmt.Printf("Role: %s\n", authConfig.User.Role)
	} else {
		utils.PrintWarning("Not authenticated")
	}

	return nil
}

func showServerConfig(apiClient *client.Client) error {
	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	utils.PrintHeader("Server Configuration")

	// This would require a server endpoint to get configuration
	// For now, we'll show what we can determine from status
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, err := apiClient.GetSystemStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to get server status: %w", err)
	}

	utils.PrintSubHeader("Server Information")
	fmt.Printf("Version: %s\n", status.Version)
	fmt.Printf("Uptime: %s\n", status.Uptime)
	fmt.Printf("Status: %s\n", status.Status)

	utils.PrintSubHeader("Services Configuration")
	fmt.Printf("Database Connected: %t\n", status.Database.Connected)
	fmt.Printf("Redis Connected: %t\n", status.Redis.Connected)
	fmt.Printf("WebSocket Enabled: %t\n", status.WebSocket.Enabled)
	fmt.Printf("R&D Module Enabled: %t\n", status.RnD.Enabled)
	fmt.Printf("Monitoring Enabled: %t\n", status.Monitoring.Enabled)

	return nil
}

func setCliConfig(key, value string) error {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// Create config file if it doesn't exist
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := initCliConfig(false); err != nil {
			return err
		}
	}

	// Load existing config
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Set the value
	viper.Set(key, value)

	// Write back to file
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Set %s = %s", key, value))

	return nil
}

func getCliConfig(key string) error {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		utils.PrintWarning("Configuration file does not exist. Run 'kaskman config init' to create it.")
		return nil
	}

	// Load config
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Get the value
	value := viper.Get(key)
	if value == nil {
		utils.PrintWarning(fmt.Sprintf("Configuration key '%s' not found", key))
		return nil
	}

	fmt.Printf("%s: %v\n", key, value)

	return nil
}

func initCliConfig(force bool) error {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configFile); err == nil && !force {
		return fmt.Errorf("configuration file already exists. Use --force to overwrite")
	}

	// Create default configuration
	defaultConfig := `# KaskManager CLI Configuration
default_url: http://localhost:8080
default_format: table
timeout: 30
retry_attempts: 3
retry_delay: 1s
log_level: info
auto_update_check: true
editor: ""
pager: ""

# Output settings
colors: true
show_headers: true
show_timestamps: true

# Completion settings
completion_cache_ttl: 1h
`

	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	utils.PrintSuccess(fmt.Sprintf("Configuration initialized at %s", configFile))

	return nil
}

func validateCliConfig() error {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	utils.PrintHeader("Validating CLI Configuration")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		utils.PrintError("Configuration file does not exist")
		return fmt.Errorf("configuration file not found")
	}

	utils.PrintSuccess("Configuration file exists")

	// Try to load config
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		utils.PrintError(fmt.Sprintf("Failed to read configuration: %v", err))
		return err
	}

	utils.PrintSuccess("Configuration file is valid YAML")

	// Validate specific settings
	defaultURL := viper.GetString("default_url")
	if defaultURL == "" {
		utils.PrintWarning("default_url is not set")
	} else {
		utils.PrintSuccess(fmt.Sprintf("default_url: %s", defaultURL))
	}

	timeout := viper.GetInt("timeout")
	if timeout <= 0 {
		utils.PrintWarning("timeout should be positive")
	} else {
		utils.PrintSuccess(fmt.Sprintf("timeout: %d seconds", timeout))
	}

	format := viper.GetString("default_format")
	validFormats := []string{"table", "json", "yaml"}
	isValidFormat := false
	for _, validFormat := range validFormats {
		if format == validFormat {
			isValidFormat = true
			break
		}
	}
	if !isValidFormat {
		utils.PrintWarning(fmt.Sprintf("default_format '%s' is not valid (should be one of: %v)", format, validFormats))
	} else {
		utils.PrintSuccess(fmt.Sprintf("default_format: %s", format))
	}

	// Check auth config
	if _, err := utils.LoadAuthConfig(); err == nil {
		utils.PrintSuccess("Authentication configuration is valid")
	} else {
		utils.PrintWarning("No valid authentication found")
	}

	utils.PrintInfo("Configuration validation complete")

	return nil
}

func validateServerConfig(apiClient *client.Client) error {
	if apiClient == nil {
		return fmt.Errorf("API client not initialized")
	}

	utils.PrintHeader("Validating Server Configuration")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test basic connectivity
	health, err := apiClient.Health(ctx)
	if err != nil {
		utils.PrintError(fmt.Sprintf("Server health check failed: %v", err))
		return err
	}

	utils.PrintSuccess("Server is reachable")
	utils.PrintSuccess(fmt.Sprintf("Server status: %s", health.Status))
	utils.PrintSuccess(fmt.Sprintf("Server version: %s", health.Version))

	// Test detailed status
	status, err := apiClient.GetSystemStatus(ctx)
	if err != nil {
		utils.PrintWarning(fmt.Sprintf("Failed to get detailed status: %v", err))
	} else {
		utils.PrintSuccess("Detailed status endpoint accessible")

		// Validate services
		if status.Database.Connected {
			utils.PrintSuccess("Database connection: OK")
		} else {
			utils.PrintError("Database connection: FAILED")
		}

		if status.Redis.Connected {
			utils.PrintSuccess("Redis connection: OK")
		} else {
			utils.PrintWarning("Redis connection: FAILED")
		}

		if status.WebSocket.Enabled {
			utils.PrintSuccess("WebSocket service: Enabled")
		} else {
			utils.PrintWarning("WebSocket service: Disabled")
		}

		if status.RnD.Enabled {
			utils.PrintSuccess("R&D module: Enabled")
		} else {
			utils.PrintWarning("R&D module: Disabled")
		}

		if status.Monitoring.Enabled {
			utils.PrintSuccess("Monitoring service: Enabled")
		} else {
			utils.PrintWarning("Monitoring service: Disabled")
		}
	}

	utils.PrintInfo("Server configuration validation complete")

	return nil
}

func editCliConfig() error {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create it first
		if err := initCliConfig(false); err != nil {
			return err
		}
	}

	// Determine editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try to load from config
		viper.SetConfigFile(configFile)
		viper.ReadInConfig()
		editor = viper.GetString("editor")
	}
	if editor == "" {
		// Default editors
		for _, defaultEditor := range []string{"nano", "vi", "vim"} {
			if _, err := exec.LookPath(defaultEditor); err == nil {
				editor = defaultEditor
				break
			}
		}
	}
	if editor == "" {
		return fmt.Errorf("no editor found. Set EDITOR environment variable or editor in config")
	}

	utils.PrintInfo(fmt.Sprintf("Opening %s with %s", configFile, editor))

	// Execute editor
	cmd := exec.Command(editor, configFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
