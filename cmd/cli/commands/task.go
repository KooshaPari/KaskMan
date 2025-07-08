package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewTaskCommand creates the task command
func NewTaskCommand(apiClient **client.Client, format *string) *cobra.Command {
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Task management commands",
		Long:  "Commands for managing tasks in the R&D platform",
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all tasks",
		Long:  "List all tasks with optional filtering",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, _ := cmd.Flags().GetString("status")
			taskType, _ := cmd.Flags().GetString("type")
			priority, _ := cmd.Flags().GetString("priority")
			projectID, _ := cmd.Flags().GetString("project")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching tasks...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			tasks, err := (*apiClient).GetTasks(ctx)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch tasks: %w", err)
			}

			// Apply filters
			if status != "" || taskType != "" || priority != "" || projectID != "" {
				filtered := tasks[:0]
				for _, task := range tasks {
					if status != "" && task.Status != status {
						continue
					}
					if taskType != "" && task.Type != taskType {
						continue
					}
					if priority != "" && task.Priority != priority {
						continue
					}
					if projectID != "" && (task.ProjectID == nil || task.ProjectID.String() != projectID) {
						continue
					}
					filtered = append(filtered, task)
				}
				tasks = filtered
			}

			if len(tasks) == 0 {
				utils.PrintInfo("No tasks found.")
				return nil
			}

			// Display tasks
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(tasks)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			case "table":
				fmt.Println(utils.FormatTasksTable(tasks))
			default:
				fmt.Println(utils.FormatTasksTable(tasks))
			}

			utils.PrintInfo(fmt.Sprintf("Found %d tasks", len(tasks)))
			return nil
		},
	}

	// Create command
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new task",
		Long:  "Create a new task with interactive prompts or flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			title, _ := cmd.Flags().GetString("title")
			description, _ := cmd.Flags().GetString("description")
			taskType, _ := cmd.Flags().GetString("type")
			priority, _ := cmd.Flags().GetString("priority")
			estimatedTime, _ := cmd.Flags().GetInt("estimated-time")
			projectID, _ := cmd.Flags().GetString("project")
			assignedTo, _ := cmd.Flags().GetString("assigned-to")
			agentID, _ := cmd.Flags().GetString("agent")
			interactive, _ := cmd.Flags().GetBool("interactive")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Interactive mode
			if interactive || title == "" {
				var err error
				if title == "" {
					title, err = utils.PromptString("Task title")
					if err != nil {
						return err
					}
				}
				if description == "" {
					description, err = utils.PromptString("Task description")
					if err != nil {
						return err
					}
				}
				if taskType == "" {
					types := []string{"analysis", "coding", "research", "testing", "documentation", "design"}
					choice, err := utils.PromptChoice("Task type", types)
					if err != nil {
						return err
					}
					taskType = types[choice]
				}
				if priority == "" {
					priorities := []string{"low", "medium", "high", "critical"}
					choice, err := utils.PromptChoice("Priority", priorities)
					if err != nil {
						return err
					}
					priority = priorities[choice]
				}
				if estimatedTime == 0 {
					timeStr, err := utils.PromptString("Estimated time in minutes (optional)")
					if err != nil {
						return err
					}
					if timeStr != "" {
						estimatedTime, _ = strconv.Atoi(timeStr)
					}
				}
				if projectID == "" {
					projectID, err = utils.PromptString("Project ID (optional)")
					if err != nil {
						return err
					}
				}
			}

			// Validate required fields
			if title == "" {
				return fmt.Errorf("task title is required")
			}
			if taskType == "" {
				taskType = "analysis"
			}
			if priority == "" {
				priority = "medium"
			}

			// Create task request
			taskReq := client.CreateTaskRequest{
				Title:         title,
				Description:   description,
				Type:          taskType,
				Priority:      priority,
				EstimatedTime: estimatedTime,
			}

			if projectID != "" {
				if projectUUID, err := uuid.Parse(projectID); err == nil {
					taskReq.ProjectID = &projectUUID
				}
			}

			if assignedTo != "" {
				if userUUID, err := uuid.Parse(assignedTo); err == nil {
					taskReq.AssignedTo = &userUUID
				}
			}

			if agentID != "" {
				if agentUUID, err := uuid.Parse(agentID); err == nil {
					taskReq.AgentID = &agentUUID
				}
			}

			spinner := utils.NewSpinner("Creating task...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			task, err := (*apiClient).CreateTask(ctx, taskReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Task '%s' created successfully", task.Title))
			fmt.Printf("Task ID: %s\n", task.ID)

			return nil
		},
	}

	// Show command
	showCmd := &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show task details",
		Long:  "Display detailed information about a specific task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching task details...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			task, err := (*apiClient).GetTask(ctx, taskID)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch task: %w", err)
			}

			// Display task details
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(task)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				utils.PrintHeader(fmt.Sprintf("Task: %s", task.Title))
				fmt.Print(utils.FormatTask(task))
			}

			return nil
		},
	}

	// Update command
	updateCmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Update a task",
		Long:  "Update task details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Build update request
			updateReq := client.UpdateTaskRequest{}

			if cmd.Flags().Changed("title") {
				title, _ := cmd.Flags().GetString("title")
				updateReq.Title = &title
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
			if cmd.Flags().Changed("estimated-time") {
				estimatedTime, _ := cmd.Flags().GetInt("estimated-time")
				updateReq.EstimatedTime = &estimatedTime
			}
			if cmd.Flags().Changed("result") {
				result, _ := cmd.Flags().GetString("result")
				updateReq.Result = &result
			}
			if cmd.Flags().Changed("project") {
				projectID, _ := cmd.Flags().GetString("project")
				if projectUUID, err := uuid.Parse(projectID); err == nil {
					updateReq.ProjectID = &projectUUID
				}
			}
			if cmd.Flags().Changed("assigned-to") {
				assignedTo, _ := cmd.Flags().GetString("assigned-to")
				if userUUID, err := uuid.Parse(assignedTo); err == nil {
					updateReq.AssignedTo = &userUUID
				}
			}
			if cmd.Flags().Changed("agent") {
				agentID, _ := cmd.Flags().GetString("agent")
				if agentUUID, err := uuid.Parse(agentID); err == nil {
					updateReq.AgentID = &agentUUID
				}
			}

			spinner := utils.NewSpinner("Updating task...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			updatedTask, err := (*apiClient).UpdateTask(ctx, taskID, updateReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to update task: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Task '%s' updated successfully", updatedTask.Title))

			return nil
		},
	}

	// Complete command
	completeCmd := &cobra.Command{
		Use:   "complete <task-id>",
		Short: "Mark a task as completed",
		Long:  "Mark a task as completed with optional result",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			result, _ := cmd.Flags().GetString("result")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			if result == "" {
				var err error
				result, err = utils.PromptString("Task result (optional)")
				if err != nil {
					return err
				}
			}

			// Update task to completed
			updateReq := client.UpdateTaskRequest{
				Status:   stringPtr("completed"),
				Progress: intPtr(100),
			}

			if result != "" {
				updateReq.Result = &result
			}

			spinner := utils.NewSpinner("Completing task...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			updatedTask, err := (*apiClient).UpdateTask(ctx, taskID, updateReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to complete task: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Task '%s' marked as completed", updatedTask.Title))

			return nil
		},
	}

	// Assign command
	assignCmd := &cobra.Command{
		Use:   "assign <task-id> <user-id-or-agent-id>",
		Short: "Assign a task to a user or agent",
		Long:  "Assign a task to a user or agent",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			assigneeID := args[1]
			toAgent, _ := cmd.Flags().GetBool("agent")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			assigneeUUID, err := uuid.Parse(assigneeID)
			if err != nil {
				return fmt.Errorf("invalid assignee ID: %w", err)
			}

			// Update task assignment
			updateReq := client.UpdateTaskRequest{}

			if toAgent {
				updateReq.AgentID = &assigneeUUID
			} else {
				updateReq.AssignedTo = &assigneeUUID
			}

			spinner := utils.NewSpinner("Assigning task...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			updatedTask, err := (*apiClient).UpdateTask(ctx, taskID, updateReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to assign task: %w", err)
			}

			assigneeType := "user"
			if toAgent {
				assigneeType = "agent"
			}

			utils.PrintSuccess(fmt.Sprintf("Task '%s' assigned to %s %s", updatedTask.Title, assigneeType, assigneeID))

			return nil
		},
	}

	// Delete command
	deleteCmd := &cobra.Command{
		Use:   "delete <task-id>",
		Short: "Delete a task",
		Long:  "Delete a task (requires confirmation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			force, _ := cmd.Flags().GetBool("force")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			if !force {
				confirmed, err := utils.PromptConfirm("Are you sure you want to delete this task?")
				if err != nil {
					return err
				}
				if !confirmed {
					utils.PrintInfo("Operation cancelled")
					return nil
				}
			}

			spinner := utils.NewSpinner("Deleting task...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := (*apiClient).DeleteTask(ctx, taskID)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to delete task: %w", err)
			}

			utils.PrintSuccess("Task deleted successfully")

			return nil
		},
	}

	// Add flags
	listCmd.Flags().StringP("status", "s", "", "Filter by status (pending, in_progress, completed, failed)")
	listCmd.Flags().StringP("type", "t", "", "Filter by type (analysis, coding, research, testing, documentation, design)")
	listCmd.Flags().StringP("priority", "p", "", "Filter by priority (low, medium, high, critical)")
	listCmd.Flags().StringP("project", "P", "", "Filter by project ID")

	createCmd.Flags().StringP("title", "t", "", "Task title")
	createCmd.Flags().StringP("description", "d", "", "Task description")
	createCmd.Flags().StringP("type", "T", "", "Task type (analysis, coding, research, testing, documentation, design)")
	createCmd.Flags().StringP("priority", "p", "", "Priority (low, medium, high, critical)")
	createCmd.Flags().IntP("estimated-time", "e", 0, "Estimated time in minutes")
	createCmd.Flags().StringP("project", "P", "", "Project ID")
	createCmd.Flags().StringP("assigned-to", "u", "", "Assigned user ID")
	createCmd.Flags().StringP("agent", "a", "", "Agent ID")
	createCmd.Flags().BoolP("interactive", "i", false, "Interactive mode")

	updateCmd.Flags().StringP("title", "t", "", "Task title")
	updateCmd.Flags().StringP("description", "d", "", "Task description")
	updateCmd.Flags().StringP("status", "s", "", "Status (pending, in_progress, completed, failed)")
	updateCmd.Flags().StringP("priority", "p", "", "Priority (low, medium, high, critical)")
	updateCmd.Flags().IntP("progress", "P", 0, "Progress (0-100)")
	updateCmd.Flags().IntP("estimated-time", "e", 0, "Estimated time in minutes")
	updateCmd.Flags().StringP("result", "r", "", "Task result")
	updateCmd.Flags().StringP("project", "j", "", "Project ID")
	updateCmd.Flags().StringP("assigned-to", "u", "", "Assigned user ID")
	updateCmd.Flags().StringP("agent", "a", "", "Agent ID")

	completeCmd.Flags().StringP("result", "r", "", "Task result")

	assignCmd.Flags().BoolP("agent", "a", false, "Assign to agent instead of user")

	deleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	// Add subcommands
	taskCmd.AddCommand(listCmd)
	taskCmd.AddCommand(createCmd)
	taskCmd.AddCommand(showCmd)
	taskCmd.AddCommand(updateCmd)
	taskCmd.AddCommand(completeCmd)
	taskCmd.AddCommand(assignCmd)
	taskCmd.AddCommand(deleteCmd)

	return taskCmd
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
