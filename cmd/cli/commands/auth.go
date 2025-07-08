package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewAuthCommand creates the auth command
func NewAuthCommand(apiClient **client.Client, baseURL *string) *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
		Long:  "Commands for managing authentication with the KaskManager server",
	}

	// Login command
	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the KaskManager server",
		Long:  "Authenticate with the KaskManager server using username and password",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, _ := cmd.Flags().GetString("username")
			password, _ := cmd.Flags().GetString("password")

			// Prompt for credentials if not provided
			if username == "" {
				var err error
				username, err = utils.PromptString("Username")
				if err != nil {
					return fmt.Errorf("failed to read username: %w", err)
				}
			}

			if password == "" {
				var err error
				password, err = utils.PromptPassword("Password")
				if err != nil {
					return fmt.Errorf("failed to read password: %w", err)
				}
			}

			if username == "" || password == "" {
				return fmt.Errorf("username and password are required")
			}

			// Create temporary client for login
			tmpClient := client.NewClient(*baseURL)

			spinner := utils.NewSpinner("Authenticating...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			loginResp, err := tmpClient.Login(ctx, username, password)
			spinner.Stop()

			if err != nil {
				utils.PrintError(fmt.Sprintf("Login failed: %v", err))
				return err
			}

			// Save auth config
			authConfig := &utils.AuthConfig{
				BaseURL: *baseURL,
				Token:   loginResp.Token,
			}
			authConfig.User.ID = loginResp.User.ID.String()
			authConfig.User.Username = loginResp.User.Username
			authConfig.User.Email = loginResp.User.Email
			authConfig.User.Role = loginResp.User.Role

			if err := utils.SaveAuthConfig(authConfig); err != nil {
				utils.PrintError(fmt.Sprintf("Failed to save authentication: %v", err))
				return err
			}

			utils.PrintSuccess(fmt.Sprintf("Successfully authenticated as %s", loginResp.User.Username))
			utils.PrintInfo(fmt.Sprintf("Token expires at: %s", loginResp.ExpiresAt.Format("2006-01-02 15:04:05")))

			return nil
		},
	}

	// Logout command
	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from the KaskManager server",
		Long:  "Clear stored authentication credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := utils.ClearAuthConfig(); err != nil {
				utils.PrintError(fmt.Sprintf("Failed to clear authentication: %v", err))
				return err
			}

			utils.PrintSuccess("Successfully logged out")
			return nil
		},
	}

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status",
		Long:  "Display current authentication status and user information",
		RunE: func(cmd *cobra.Command, args []string) error {
			authConfig, err := utils.LoadAuthConfig()
			if err != nil {
				utils.PrintWarning("Not authenticated")
				return nil
			}

			utils.PrintHeader("Authentication Status")
			fmt.Printf("Server URL: %s\n", authConfig.BaseURL)
			fmt.Printf("Username: %s\n", authConfig.User.Username)
			fmt.Printf("Email: %s\n", authConfig.User.Email)
			fmt.Printf("Role: %s\n", authConfig.User.Role)
			fmt.Printf("User ID: %s\n", authConfig.User.ID)

			// Test connection
			if *apiClient != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if health, err := (*apiClient).Health(ctx); err == nil {
					utils.PrintSuccess("Connection to server: OK")
					fmt.Printf("Server Status: %s\n", health.Status)
					fmt.Printf("Server Version: %s\n", health.Version)
				} else {
					utils.PrintError(fmt.Sprintf("Connection to server failed: %v", err))
				}
			}

			return nil
		},
	}

	// Add flags
	loginCmd.Flags().StringP("username", "u", "", "Username for authentication")
	loginCmd.Flags().StringP("password", "p", "", "Password for authentication")

	// Add subcommands
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	return authCmd
}
