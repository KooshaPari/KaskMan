package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewProjectCommand creates the project command
func NewProjectCommand(apiClient **client.Client, format *string) *cobra.Command {
	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "Project management commands",
		Long:  "Commands for managing projects in the R&D platform",
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Long:  "List all projects with optional filtering",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, _ := cmd.Flags().GetString("status")
			projectType, _ := cmd.Flags().GetString("type")
			priority, _ := cmd.Flags().GetString("priority")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching projects...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			projects, err := (*apiClient).GetProjects(ctx)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch projects: %w", err)
			}

			// Apply filters
			if status != "" || projectType != "" || priority != "" {
				filtered := projects[:0]
				for _, project := range projects {
					if status != "" && project.Status != status {
						continue
					}
					if projectType != "" && project.Type != projectType {
						continue
					}
					if priority != "" && project.Priority != priority {
						continue
					}
					filtered = append(filtered, project)
				}
				projects = filtered
			}

			if len(projects) == 0 {
				utils.PrintInfo("No projects found.")
				return nil
			}

			// Display projects
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(projects)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			case "table":
				fmt.Println(utils.FormatProjectsTable(projects))
			default:
				fmt.Println(utils.FormatProjectsTable(projects))
			}

			utils.PrintInfo(fmt.Sprintf("Found %d projects", len(projects)))
			return nil
		},
	}

	// Create command
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new project",
		Long:  "Create a new project with interactive prompts or flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			projectType, _ := cmd.Flags().GetString("type")
			priority, _ := cmd.Flags().GetString("priority")
			estimatedHours, _ := cmd.Flags().GetInt("estimated-hours")
			budget, _ := cmd.Flags().GetFloat64("budget")
			tags, _ := cmd.Flags().GetStringSlice("tags")
			interactive, _ := cmd.Flags().GetBool("interactive")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Interactive mode
			if interactive || name == "" {
				var err error
				if name == "" {
					name, err = utils.PromptString("Project name")
					if err != nil {
						return err
					}
				}
				if description == "" {
					description, err = utils.PromptString("Project description")
					if err != nil {
						return err
					}
				}
				if projectType == "" {
					types := []string{"research", "development", "analysis", "innovation"}
					choice, err := utils.PromptChoice("Project type", types)
					if err != nil {
						return err
					}
					projectType = types[choice]
				}
				if priority == "" {
					priorities := []string{"low", "medium", "high", "critical"}
					choice, err := utils.PromptChoice("Priority", priorities)
					if err != nil {
						return err
					}
					priority = priorities[choice]
				}
				if estimatedHours == 0 {
					hoursStr, err := utils.PromptString("Estimated hours (optional)")
					if err != nil {
						return err
					}
					if hoursStr != "" {
						estimatedHours, _ = strconv.Atoi(hoursStr)
					}
				}
				if budget == 0 {
					budgetStr, err := utils.PromptString("Budget (optional)")
					if err != nil {
						return err
					}
					if budgetStr != "" {
						budget, _ = strconv.ParseFloat(budgetStr, 64)
					}
				}
				if len(tags) == 0 {
					tagsStr, err := utils.PromptString("Tags (comma-separated, optional)")
					if err != nil {
						return err
					}
					if tagsStr != "" {
						tags = strings.Split(tagsStr, ",")
						for i, tag := range tags {
							tags[i] = strings.TrimSpace(tag)
						}
					}
				}
			}

			// Validate required fields
			if name == "" {
				return fmt.Errorf("project name is required")
			}
			if projectType == "" {
				projectType = "development"
			}
			if priority == "" {
				priority = "medium"
			}

			// Create project
			projectReq := client.CreateProjectRequest{
				Name:           name,
				Description:    description,
				Type:           projectType,
				Priority:       priority,
				EstimatedHours: estimatedHours,
				Budget:         budget,
				Tags:           tags,
			}

			spinner := utils.NewSpinner("Creating project...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			project, err := (*apiClient).CreateProject(ctx, projectReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to create project: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Project '%s' created successfully", project.Name))
			fmt.Printf("Project ID: %s\n", project.ID)

			return nil
		},
	}

	// Show command
	showCmd := &cobra.Command{
		Use:   "show <project-id>",
		Short: "Show project details",
		Long:  "Display detailed information about a specific project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching project details...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			project, err := (*apiClient).GetProject(ctx, projectID)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch project: %w", err)
			}

			// Display project details
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(project)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				utils.PrintHeader(fmt.Sprintf("Project: %s", project.Name))
				fmt.Print(utils.FormatProject(project))
			}

			return nil
		},
	}

	// Update command
	updateCmd := &cobra.Command{
		Use:   "update <project-id>",
		Short: "Update a project",
		Long:  "Update project details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Build update request
			updateReq := client.UpdateProjectRequest{}

			if cmd.Flags().Changed("name") {
				name, _ := cmd.Flags().GetString("name")
				updateReq.Name = &name
			}
			if cmd.Flags().Changed("description") {
				description, _ := cmd.Flags().GetString("description")
				updateReq.Description = &description
			}
			if cmd.Flags().Changed("status") {
				status, _ := cmd.Flags().GetString("status")
				updateReq.Status = &status
			}
			if cmd.Flags().Changed("priority") {
				priority, _ := cmd.Flags().GetString("priority")
				updateReq.Priority = &priority
			}
			if cmd.Flags().Changed("progress") {
				progress, _ := cmd.Flags().GetInt("progress")
				updateReq.Progress = &progress
			}
			if cmd.Flags().Changed("estimated-hours") {
				estimatedHours, _ := cmd.Flags().GetInt("estimated-hours")
				updateReq.EstimatedHours = &estimatedHours
			}
			if cmd.Flags().Changed("budget") {
				budget, _ := cmd.Flags().GetFloat64("budget")
				updateReq.Budget = &budget
			}
			if cmd.Flags().Changed("tags") {
				tags, _ := cmd.Flags().GetStringSlice("tags")
				updateReq.Tags = tags
			}

			spinner := utils.NewSpinner("Updating project...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			updatedProject, err := (*apiClient).UpdateProject(ctx, projectID, updateReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to update project: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Project '%s' updated successfully", updatedProject.Name))

			return nil
		},
	}

	// Delete command
	deleteCmd := &cobra.Command{
		Use:   "delete <project-id>",
		Short: "Delete a project",
		Long:  "Delete a project (requires confirmation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			force, _ := cmd.Flags().GetBool("force")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			if !force {
				confirmed, err := utils.PromptConfirm("Are you sure you want to delete this project?")
				if err != nil {
					return err
				}
				if !confirmed {
					utils.PrintInfo("Operation cancelled")
					return nil
				}
			}

			spinner := utils.NewSpinner("Deleting project...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := (*apiClient).DeleteProject(ctx, projectID)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to delete project: %w", err)
			}

			utils.PrintSuccess("Project deleted successfully")

			return nil
		},
	}

	// Add flags
	listCmd.Flags().StringP("status", "s", "", "Filter by status (active, completed, paused, cancelled)")
	listCmd.Flags().StringP("type", "t", "", "Filter by type (research, development, analysis, innovation)")
	listCmd.Flags().StringP("priority", "p", "", "Filter by priority (low, medium, high, critical)")

	createCmd.Flags().StringP("name", "n", "", "Project name")
	createCmd.Flags().StringP("description", "d", "", "Project description")
	createCmd.Flags().StringP("type", "t", "", "Project type (research, development, analysis, innovation)")
	createCmd.Flags().StringP("priority", "p", "", "Priority (low, medium, high, critical)")
	createCmd.Flags().IntP("estimated-hours", "e", 0, "Estimated hours")
	createCmd.Flags().Float64P("budget", "b", 0, "Budget")
	createCmd.Flags().StringSliceP("tags", "T", []string{}, "Tags (comma-separated)")
	createCmd.Flags().BoolP("interactive", "i", false, "Interactive mode")

	updateCmd.Flags().StringP("name", "n", "", "Project name")
	updateCmd.Flags().StringP("description", "d", "", "Project description")
	updateCmd.Flags().StringP("status", "s", "", "Status (active, completed, paused, cancelled)")
	updateCmd.Flags().StringP("priority", "p", "", "Priority (low, medium, high, critical)")
	updateCmd.Flags().IntP("progress", "P", 0, "Progress (0-100)")
	updateCmd.Flags().IntP("estimated-hours", "e", 0, "Estimated hours")
	updateCmd.Flags().Float64P("budget", "b", 0, "Budget")
	updateCmd.Flags().StringSliceP("tags", "T", []string{}, "Tags (comma-separated)")

	deleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	// Add subcommands
	projectCmd.AddCommand(listCmd)
	projectCmd.AddCommand(createCmd)
	projectCmd.AddCommand(showCmd)
	projectCmd.AddCommand(updateCmd)
	projectCmd.AddCommand(deleteCmd)

	return projectCmd
}
