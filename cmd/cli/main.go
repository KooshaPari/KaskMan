package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/commands"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	baseURL string
	format  string
	verbose bool
	timeout int

	// Client instance
	apiClient *client.Client
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kaskman",
	Short: "KaskManager R&D Platform CLI",
	Long: `KaskManager R&D Platform CLI provides command-line access to the 
persistent, always-on, self-improving utility and R&D platform.

This CLI allows you to manage projects, agents, tasks, and monitor 
the system without using the web interface.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize API client
		apiClient = client.NewClient(baseURL)
		apiClient.HTTPClient.Timeout = time.Duration(timeout) * time.Second

		// Load auth config if available (except for auth commands)
		if cmd.Name() != "auth" && cmd.Parent() != nil && cmd.Parent().Name() != "auth" {
			authConfig, err := utils.LoadAuthConfig()
			if err != nil {
				// Only return error if command requires authentication
				if cmd.Name() != "version" && cmd.Name() != "help" {
					return err
				}
			} else {
				apiClient.SetToken(authConfig.Token)
				if authConfig.BaseURL != "" {
					baseURL = authConfig.BaseURL
					apiClient.BaseURL = baseURL
				}
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		utils.PrintHeader("KaskManager R&D Platform CLI")
		fmt.Println("Use 'kaskman --help' for available commands")
		fmt.Println()

		// Show quick status if authenticated
		if apiClient.Token != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if health, err := apiClient.Health(ctx); err == nil {
				utils.PrintInfo(fmt.Sprintf("Connected to %s - Status: %s", baseURL, health.Status))
			}
		} else {
			utils.PrintWarning("Not authenticated. Use 'kaskman auth login' to authenticate.")
		}
	},
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("KaskManager R&D Platform CLI v1.0.0")
		fmt.Println("Built with Go")

		// Show server version if connected
		if apiClient.Token != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if health, err := apiClient.Health(ctx); err == nil {
				fmt.Printf("Server Version: %s\n", health.Version)
			}
		}
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&baseURL, "url", "u", "http://localhost:8080", "KaskManager server URL")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table", "Output format (table, json, yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().IntVarP(&timeout, "timeout", "t", 30, "Request timeout in seconds")

	// Add subcommands to root
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(commands.NewStartCommand())
	rootCmd.AddCommand(commands.NewStatusCommand(&apiClient))
	rootCmd.AddCommand(commands.NewConfigCommand(&apiClient))
	rootCmd.AddCommand(commands.NewAuthCommand(&apiClient, &baseURL))
	rootCmd.AddCommand(commands.NewProjectCommand(&apiClient, &format))
	rootCmd.AddCommand(commands.NewTaskCommand(&apiClient, &format))
	rootCmd.AddCommand(commands.NewAgentCommand(&apiClient, &format))
	rootCmd.AddCommand(commands.NewRnDCommand(&apiClient, &format))
	rootCmd.AddCommand(commands.NewCompletionCommand())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		utils.PrintError(err.Error())
		os.Exit(1)
	}
}
