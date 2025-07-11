package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewProjectManagerCommand creates the enhanced project manager command
func NewProjectManagerCommand(apiClient **client.Client, format *string) *cobra.Command {
	pmCmd := &cobra.Command{
		Use:   "pm",
		Short: "Enhanced project management commands",
		Long:  "Enhanced project management with git integration, asset management, and workflow automation",
	}

	// Scan command - discover git projects
	scanCmd := &cobra.Command{
		Use:   "scan [directory]",
		Short: "Scan directory for git projects",
		Long:  "Scan directory recursively for git repositories and add them as projects",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			scanPath := "."
			if len(args) > 0 {
				scanPath = args[0]
			}

			autoAdd, _ := cmd.Flags().GetBool("auto-add")
			dryRun, _ := cmd.Flags().GetBool("dry-run")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Scanning for git projects...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Get absolute path
			absPath, err := filepath.Abs(scanPath)
			if err != nil {
				spinner.Stop()
				return fmt.Errorf("failed to get absolute path: %w", err)
			}

			// Scan for projects
			projects, err := (*apiClient).ScanProjects(ctx, absPath)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to scan projects: %w", err)
			}

			if len(projects) == 0 {
				utils.PrintInfo("No git projects found.")
				return nil
			}

			utils.PrintSuccess(fmt.Sprintf("Found %d git projects:", len(projects)))

			// Display found projects
			for i, project := range projects {
				fmt.Printf("\n%d. %s\n", i+1, project.Name)
				fmt.Printf("   Path: %s\n", project.Path)
				fmt.Printf("   Remote: %s\n", project.RemoteURL)
				fmt.Printf("   Language: %s\n", project.Language)
				fmt.Printf("   Framework: %s\n", project.Framework)
				fmt.Printf("   Has README: %v\n", project.HasReadme)
				fmt.Printf("   Has CI/CD: %v\n", project.HasCI)
			}

			if dryRun {
				utils.PrintInfo("Dry run mode - no projects added.")
				return nil
			}

			if autoAdd {
				// Add all projects automatically
				spinner = utils.NewSpinner("Adding projects...")
				spinner.Start()

				added := 0
				for _, project := range projects {
					if err := (*apiClient).AddProjectFromGit(ctx, project); err != nil {
						utils.PrintWarning(fmt.Sprintf("Failed to add project %s: %v", project.Name, err))
					} else {
						added++
					}
				}

				spinner.Stop()
				utils.PrintSuccess(fmt.Sprintf("Added %d projects successfully", added))
			} else {
				// Interactive selection
				for i, project := range projects {
					confirmed, err := utils.PromptConfirm(fmt.Sprintf("Add project '%s'?", project.Name))
					if err != nil {
						return err
					}

					if confirmed {
						spinner = utils.NewSpinner(fmt.Sprintf("Adding project %s...", project.Name))
						spinner.Start()

						if err := (*apiClient).AddProjectFromGit(ctx, project); err != nil {
							spinner.Stop()
							utils.PrintError(fmt.Sprintf("Failed to add project %s: %v", project.Name, err))
						} else {
							spinner.Stop()
							utils.PrintSuccess(fmt.Sprintf("Project '%s' added successfully", project.Name))
						}
					}
				}
			}

			return nil
		},
	}

	// Assets command group
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Manage project assets",
		Long:  "Manage project screenshots, videos, and other assets",
	}

	// Assets list command
	assetsListCmd := &cobra.Command{
		Use:   "list <project-id>",
		Short: "List project assets",
		Long:  "List all assets for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			assetType, _ := cmd.Flags().GetString("type")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching project assets...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			assets, err := (*apiClient).GetProjectAssets(ctx, projectID, assetType)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch project assets: %w", err)
			}

			if len(assets) == 0 {
				utils.PrintInfo("No assets found for this project.")
				return nil
			}

			// Display assets
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(assets)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				fmt.Println(utils.FormatAssetsTable(assets))
			}

			utils.PrintInfo(fmt.Sprintf("Found %d assets", len(assets)))
			return nil
		},
	}

	// Assets generate command
	assetsGenerateCmd := &cobra.Command{
		Use:   "generate <project-id>",
		Short: "Generate project assets",
		Long:  "Generate screenshots and videos for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			url, _ := cmd.Flags().GetString("url")
			generateScreenshots, _ := cmd.Flags().GetBool("screenshots")
			generateVideos, _ := cmd.Flags().GetBool("videos")
			videoDuration, _ := cmd.Flags().GetInt("video-duration")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			if url == "" {
				var err error
				url, err = utils.PromptString("URL to capture")
				if err != nil {
					return err
				}
			}

			if !generateScreenshots && !generateVideos {
				generateScreenshots = true // Default to screenshots
			}

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			var generatedAssets []string

			if generateScreenshots {
				spinner := utils.NewSpinner("Generating screenshot...")
				spinner.Start()

				asset, err := (*apiClient).GenerateScreenshot(ctx, projectID, url)
				spinner.Stop()

				if err != nil {
					utils.PrintError(fmt.Sprintf("Failed to generate screenshot: %v", err))
				} else {
					generatedAssets = append(generatedAssets, fmt.Sprintf("Screenshot: %s", asset.ID))
					utils.PrintSuccess("Screenshot generated successfully")
				}
			}

			if generateVideos {
				spinner := utils.NewSpinner(fmt.Sprintf("Generating %d-second video...", videoDuration))
				spinner.Start()

				asset, err := (*apiClient).GenerateVideo(ctx, projectID, url, videoDuration)
				spinner.Stop()

				if err != nil {
					utils.PrintError(fmt.Sprintf("Failed to generate video: %v", err))
				} else {
					generatedAssets = append(generatedAssets, fmt.Sprintf("Video: %s", asset.ID))
					utils.PrintSuccess("Video generated successfully")
				}
			}

			if len(generatedAssets) > 0 {
				fmt.Println("\nGenerated assets:")
				for _, asset := range generatedAssets {
					fmt.Printf("  - %s\n", asset)
				}
			}

			return nil
		},
	}

	// State command group
	stateCmd := &cobra.Command{
		Use:   "state",
		Short: "Check project state",
		Long:  "Check project build status, tests, linting, and overall health",
	}

	// State check command
	stateCheckCmd := &cobra.Command{
		Use:   "check <project-id>",
		Short: "Check project state",
		Long:  "Perform comprehensive project state check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			detailed, _ := cmd.Flags().GetBool("detailed")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Checking project state...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			healthCheck, err := (*apiClient).CheckProjectState(ctx, projectID)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to check project state: %w", err)
			}

			// Display results
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(healthCheck)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				utils.PrintHeader(fmt.Sprintf("Project Health Check - Score: %d/100", healthCheck.HealthScore))
				
				fmt.Printf("Build Status:      %s\n", utils.FormatStatus(healthCheck.BuildStatus))
				fmt.Printf("Test Status:       %s\n", utils.FormatStatus(healthCheck.TestStatus))
				fmt.Printf("Lint Status:       %s\n", utils.FormatStatus(healthCheck.LintStatus))
				fmt.Printf("Security Status:   %s\n", utils.FormatStatus(healthCheck.SecurityStatus))
				fmt.Printf("Deployment Status: %s\n", utils.FormatStatus(healthCheck.DeploymentStatus))
				fmt.Printf("Test Coverage:     %.1f%%\n", healthCheck.Coverage)

				if len(healthCheck.NextSteps) > 0 {
					fmt.Println("\nNext Steps:")
					for _, step := range healthCheck.NextSteps {
						fmt.Printf("  - %s\n", step)
					}
				}

				if detailed {
					if len(healthCheck.Errors) > 0 {
						fmt.Println("\nErrors:")
						for _, error := range healthCheck.Errors {
							fmt.Printf("  ❌ %s\n", error)
						}
					}

					if len(healthCheck.Warnings) > 0 {
						fmt.Println("\nWarnings:")
						for _, warning := range healthCheck.Warnings {
							fmt.Printf("  ⚠️  %s\n", warning)
						}
					}

					if len(healthCheck.Suggestions) > 0 {
						fmt.Println("\nSuggestions:")
						for _, suggestion := range healthCheck.Suggestions {
							fmt.Printf("  💡 %s\n", suggestion)
						}
					}
				}
			}

			return nil
		},
	}

	// Workflow command group
	workflowCmd := &cobra.Command{
		Use:   "workflow",
		Short: "Manage project workflows",
		Long:  "Execute and manage automated project workflows",
	}

	// Workflow run command
	workflowRunCmd := &cobra.Command{
		Use:   "run <project-id> <workflow-type>",
		Short: "Run a workflow",
		Long:  "Execute a workflow for a project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			workflowType := args[1]
			config, _ := cmd.Flags().GetStringToString("config")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Validate workflow type
			validWorkflows := []string{"asset_generation", "state_check", "full_analysis", "git_sync"}
			if !contains(validWorkflows, workflowType) {
				return fmt.Errorf("invalid workflow type. Valid types: %s", strings.Join(validWorkflows, ", "))
			}

			spinner := utils.NewSpinner(fmt.Sprintf("Executing %s workflow...", workflowType))
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
			defer cancel()

			result, err := (*apiClient).ExecuteWorkflow(ctx, projectID, workflowType, config)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to execute workflow: %w", err)
			}

			// Display results
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(result)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				utils.PrintHeader(fmt.Sprintf("Workflow Execution: %s", result.ExecutionID))
				fmt.Printf("Status:   %s\n", utils.FormatStatus(result.Status))
				fmt.Printf("Duration: %v\n", result.Duration)

				if len(result.Artifacts) > 0 {
					fmt.Println("\nGenerated Artifacts:")
					for _, artifact := range result.Artifacts {
						fmt.Printf("  - %s\n", artifact)
					}
				}

				if len(result.Errors) > 0 {
					fmt.Println("\nErrors:")
					for _, error := range result.Errors {
						fmt.Printf("  ❌ %s\n", error)
					}
				}
			}

			return nil
		},
	}

	// Workflow list command
	workflowListCmd := &cobra.Command{
		Use:   "list <project-id>",
		Short: "List workflow executions",
		Long:  "List all workflow executions for a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			status, _ := cmd.Flags().GetString("status")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching workflow executions...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			executions, err := (*apiClient).GetWorkflowExecutions(ctx, projectID, status)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch workflow executions: %w", err)
			}

			if len(executions) == 0 {
				utils.PrintInfo("No workflow executions found.")
				return nil
			}

			// Display executions
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(executions)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				fmt.Println(utils.FormatWorkflowExecutionsTable(executions))
			}

			utils.PrintInfo(fmt.Sprintf("Found %d workflow executions", len(executions)))
			return nil
		},
	}

	// Overview command - comprehensive project overview
	overviewCmd := &cobra.Command{
		Use:   "overview <project-id>",
		Short: "Project overview",
		Long:  "Get comprehensive project overview including state, assets, and recent activity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching project overview...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			overview, err := (*apiClient).GetProjectOverview(ctx, projectID)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch project overview: %w", err)
			}

			// Display overview
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(overview)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				utils.PrintHeader(fmt.Sprintf("Project Overview: %s", overview.Project.Name))
				
				// Basic info
				fmt.Printf("ID:          %s\n", overview.Project.ID)
				fmt.Printf("Type:        %s\n", overview.Project.Type)
				fmt.Printf("Status:      %s\n", overview.Project.Status)
				fmt.Printf("Priority:    %s\n", overview.Project.Priority)
				fmt.Printf("Progress:    %d%%\n", overview.Project.Progress)
				
				// Health info
				if overview.State != nil {
					fmt.Printf("Health Score: %d/100\n", overview.State.HealthScore)
					fmt.Printf("Build:       %s\n", utils.FormatStatus(overview.State.BuildStatus))
					fmt.Printf("Tests:       %s\n", utils.FormatStatus(overview.State.TestStatus))
					fmt.Printf("Coverage:    %.1f%%\n", overview.State.Coverage)
				}

				// Assets info
				if overview.Assets != nil {
					fmt.Printf("Assets:      %d total\n", len(overview.Assets))
					assetTypes := make(map[string]int)
					for _, asset := range overview.Assets {
						assetTypes[asset.AssetType]++
					}
					for assetType, count := range assetTypes {
						fmt.Printf("  - %s: %d\n", assetType, count)
					}
				}

				// Git info
				if overview.GitRepo != nil {
					fmt.Printf("Repository:  %s\n", overview.GitRepo.RepositoryURL)
					fmt.Printf("Branch:      %s\n", overview.GitRepo.Branch)
					if overview.GitRepo.LastSyncAt != nil {
						fmt.Printf("Last Sync:   %s\n", overview.GitRepo.LastSyncAt.Format("2006-01-02 15:04:05"))
					}
				}

				// Next steps
				if overview.State != nil && overview.State.NextSteps != "" {
					fmt.Printf("\nNext Steps: %s\n", overview.State.NextSteps)
				}

				// Readme and demo links
				if overview.State != nil {
					if overview.State.ReadmePath != "" {
						fmt.Printf("README:     %s\n", overview.State.ReadmePath)
					}
					if overview.State.DemoURL != "" {
						fmt.Printf("Demo:       %s\n", overview.State.DemoURL)
					}
				}
			}

			return nil
		},
	}

	// Add flags
	scanCmd.Flags().BoolP("auto-add", "a", false, "Automatically add all found projects")
	scanCmd.Flags().BoolP("dry-run", "d", false, "Show what would be added without adding")

	assetsListCmd.Flags().StringP("type", "t", "", "Filter by asset type (screenshot, video, document, demo)")

	assetsGenerateCmd.Flags().StringP("url", "u", "", "URL to capture")
	assetsGenerateCmd.Flags().BoolP("screenshots", "s", false, "Generate screenshots")
	assetsGenerateCmd.Flags().BoolP("videos", "v", false, "Generate videos")
	assetsGenerateCmd.Flags().IntP("video-duration", "d", 30, "Video duration in seconds")

	stateCheckCmd.Flags().BoolP("detailed", "d", false, "Show detailed information")

	workflowRunCmd.Flags().StringToStringP("config", "c", map[string]string{}, "Workflow configuration (key=value)")

	workflowListCmd.Flags().StringP("status", "s", "", "Filter by status")

	// Add subcommands
	assetsCmd.AddCommand(assetsListCmd)
	assetsCmd.AddCommand(assetsGenerateCmd)

	stateCmd.AddCommand(stateCheckCmd)

	workflowCmd.AddCommand(workflowRunCmd)
	workflowCmd.AddCommand(workflowListCmd)

	pmCmd.AddCommand(scanCmd)
	pmCmd.AddCommand(assetsCmd)
	pmCmd.AddCommand(stateCmd)
	pmCmd.AddCommand(workflowCmd)
	pmCmd.AddCommand(overviewCmd)

	return pmCmd
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}