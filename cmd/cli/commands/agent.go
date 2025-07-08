package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewAgentCommand creates the agent command
func NewAgentCommand(apiClient **client.Client, format *string) *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent management commands",
		Long:  "Commands for managing AI agents in the R&D platform",
	}

	// List command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		Long:  "List all agents with optional filtering",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, _ := cmd.Flags().GetString("status")
			agentType, _ := cmd.Flags().GetString("type")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching agents...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			agents, err := (*apiClient).GetAgents(ctx)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch agents: %w", err)
			}

			// Apply filters
			if status != "" || agentType != "" {
				filtered := agents[:0]
				for _, agent := range agents {
					if status != "" && agent.Status != status {
						continue
					}
					if agentType != "" && agent.Type != agentType {
						continue
					}
					filtered = append(filtered, agent)
				}
				agents = filtered
			}

			if len(agents) == 0 {
				utils.PrintInfo("No agents found.")
				return nil
			}

			// Display agents
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(agents)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			case "table":
				fmt.Println(utils.FormatAgentsTable(agents))
			default:
				fmt.Println(utils.FormatAgentsTable(agents))
			}

			utils.PrintInfo(fmt.Sprintf("Found %d agents", len(agents)))
			return nil
		},
	}

	// Create/Spawn command
	spawnCmd := &cobra.Command{
		Use:   "spawn",
		Short: "Spawn a new agent",
		Long:  "Create and spawn a new AI agent with specified capabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			agentType, _ := cmd.Flags().GetString("type")
			capabilities, _ := cmd.Flags().GetStringSlice("capabilities")
			interactive, _ := cmd.Flags().GetBool("interactive")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Interactive mode
			if interactive || name == "" || agentType == "" {
				var err error
				if name == "" {
					name, err = utils.PromptString("Agent name")
					if err != nil {
						return err
					}
				}
				if agentType == "" {
					types := []string{"researcher", "coder", "analyst", "tester", "designer", "coordinator", "optimizer"}
					choice, err := utils.PromptChoice("Agent type", types)
					if err != nil {
						return err
					}
					agentType = types[choice]
				}
				if len(capabilities) == 0 {
					capStr, err := utils.PromptString("Capabilities (comma-separated)")
					if err != nil {
						return err
					}
					if capStr != "" {
						capabilities = strings.Split(capStr, ",")
						for i, cap := range capabilities {
							capabilities[i] = strings.TrimSpace(cap)
						}
					}
				}
			}

			// Validate required fields
			if name == "" {
				return fmt.Errorf("agent name is required")
			}
			if agentType == "" {
				return fmt.Errorf("agent type is required")
			}

			// Set default capabilities based on type
			if len(capabilities) == 0 {
				capabilities = getDefaultCapabilities(agentType)
			}

			// Create agent request
			agentReq := client.CreateAgentRequest{
				Name:         name,
				Type:         agentType,
				Capabilities: capabilities,
				Config:       getDefaultConfig(agentType),
			}

			spinner := utils.NewSpinner("Spawning agent...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			agent, err := (*apiClient).CreateAgent(ctx, agentReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to spawn agent: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Agent '%s' spawned successfully", agent.Name))
			fmt.Printf("Agent ID: %s\n", agent.ID)
			fmt.Printf("Type: %s\n", agent.Type)
			fmt.Printf("Status: %s\n", agent.Status)

			return nil
		},
	}

	// Show command
	showCmd := &cobra.Command{
		Use:   "show <agent-id>",
		Short: "Show agent details",
		Long:  "Display detailed information about a specific agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching agent details...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			agent, err := (*apiClient).GetAgent(ctx, agentID)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch agent: %w", err)
			}

			// Display agent details
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(agent)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				utils.PrintHeader(fmt.Sprintf("Agent: %s", agent.Name))
				fmt.Print(utils.FormatAgent(agent))
			}

			return nil
		},
	}

	// Update command
	updateCmd := &cobra.Command{
		Use:   "update <agent-id>",
		Short: "Update an agent",
		Long:  "Update agent configuration and settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Build update request
			updateReq := client.UpdateAgentRequest{}

			if cmd.Flags().Changed("name") {
				name, _ := cmd.Flags().GetString("name")
				updateReq.Name = &name
			}
			if cmd.Flags().Changed("status") {
				status, _ := cmd.Flags().GetString("status")
				updateReq.Status = &status
			}
			if cmd.Flags().Changed("capabilities") {
				capabilities, _ := cmd.Flags().GetStringSlice("capabilities")
				updateReq.Capabilities = capabilities
			}

			spinner := utils.NewSpinner("Updating agent...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			updatedAgent, err := (*apiClient).UpdateAgent(ctx, agentID, updateReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to update agent: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Agent '%s' updated successfully", updatedAgent.Name))

			return nil
		},
	}

	// Start command
	startCmd := &cobra.Command{
		Use:   "start <agent-id>",
		Short: "Start an agent",
		Long:  "Activate an agent to begin processing tasks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Update agent status to active
			updateReq := client.UpdateAgentRequest{
				Status: stringPtr("active"),
			}

			spinner := utils.NewSpinner("Starting agent...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			updatedAgent, err := (*apiClient).UpdateAgent(ctx, agentID, updateReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to start agent: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Agent '%s' started successfully", updatedAgent.Name))

			return nil
		},
	}

	// Stop command
	stopCmd := &cobra.Command{
		Use:   "stop <agent-id>",
		Short: "Stop an agent",
		Long:  "Deactivate an agent to stop processing tasks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Update agent status to inactive
			updateReq := client.UpdateAgentRequest{
				Status: stringPtr("inactive"),
			}

			spinner := utils.NewSpinner("Stopping agent...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			updatedAgent, err := (*apiClient).UpdateAgent(ctx, agentID, updateReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to stop agent: %w", err)
			}

			utils.PrintSuccess(fmt.Sprintf("Agent '%s' stopped successfully", updatedAgent.Name))

			return nil
		},
	}

	// Monitor command
	monitorCmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitor agent activity",
		Long:  "Monitor real-time agent activity and performance",
		RunE: func(cmd *cobra.Command, args []string) error {
			interval, _ := cmd.Flags().GetInt("interval")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			utils.PrintHeader("Agent Activity Monitor")
			fmt.Printf("Refreshing every %d seconds (Press Ctrl+C to stop)\n\n", interval)

			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			defer ticker.Stop()

			// Show initial status
			if err := showAgentMonitor(*apiClient); err != nil {
				return err
			}

			for {
				select {
				case <-ticker.C:
					// Clear screen
					fmt.Print("\033[H\033[2J")

					// Show updated status
					if err := showAgentMonitor(*apiClient); err != nil {
						utils.PrintError(fmt.Sprintf("Failed to update monitor: %v", err))
						return err
					}
				}
			}
		},
	}

	// Delete command
	deleteCmd := &cobra.Command{
		Use:   "delete <agent-id>",
		Short: "Delete an agent",
		Long:  "Delete an agent (requires confirmation)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			force, _ := cmd.Flags().GetBool("force")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			if !force {
				confirmed, err := utils.PromptConfirm("Are you sure you want to delete this agent?")
				if err != nil {
					return err
				}
				if !confirmed {
					utils.PrintInfo("Operation cancelled")
					return nil
				}
			}

			spinner := utils.NewSpinner("Deleting agent...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := (*apiClient).DeleteAgent(ctx, agentID)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to delete agent: %w", err)
			}

			utils.PrintSuccess("Agent deleted successfully")

			return nil
		},
	}

	// Add flags
	listCmd.Flags().StringP("status", "s", "", "Filter by status (active, inactive, busy, error)")
	listCmd.Flags().StringP("type", "t", "", "Filter by type (researcher, coder, analyst, tester, designer, coordinator, optimizer)")

	spawnCmd.Flags().StringP("name", "n", "", "Agent name")
	spawnCmd.Flags().StringP("type", "t", "", "Agent type (researcher, coder, analyst, tester, designer, coordinator, optimizer)")
	spawnCmd.Flags().StringSliceP("capabilities", "c", []string{}, "Agent capabilities")
	spawnCmd.Flags().BoolP("interactive", "i", false, "Interactive mode")

	updateCmd.Flags().StringP("name", "n", "", "Agent name")
	updateCmd.Flags().StringP("status", "s", "", "Agent status (active, inactive, busy, error)")
	updateCmd.Flags().StringSliceP("capabilities", "c", []string{}, "Agent capabilities")

	monitorCmd.Flags().IntP("interval", "i", 5, "Refresh interval in seconds")

	deleteCmd.Flags().BoolP("force", "f", false, "Force delete without confirmation")

	// Add subcommands
	agentCmd.AddCommand(listCmd)
	agentCmd.AddCommand(spawnCmd)
	agentCmd.AddCommand(showCmd)
	agentCmd.AddCommand(updateCmd)
	agentCmd.AddCommand(startCmd)
	agentCmd.AddCommand(stopCmd)
	agentCmd.AddCommand(monitorCmd)
	agentCmd.AddCommand(deleteCmd)

	return agentCmd
}

// Helper functions
func getDefaultCapabilities(agentType string) []string {
	switch agentType {
	case "researcher":
		return []string{"data_analysis", "pattern_recognition", "report_generation", "web_search"}
	case "coder":
		return []string{"code_generation", "code_review", "debugging", "testing", "documentation"}
	case "analyst":
		return []string{"data_analysis", "statistical_analysis", "visualization", "reporting"}
	case "tester":
		return []string{"test_generation", "test_execution", "bug_detection", "performance_testing"}
	case "designer":
		return []string{"ui_design", "ux_design", "prototype_creation", "design_review"}
	case "coordinator":
		return []string{"task_management", "resource_allocation", "project_coordination", "communication"}
	case "optimizer":
		return []string{"performance_optimization", "resource_optimization", "cost_optimization", "process_improvement"}
	default:
		return []string{"general_purpose", "task_execution", "data_processing"}
	}
}

func getDefaultConfig(agentType string) map[string]interface{} {
	config := map[string]interface{}{
		"max_concurrent_tasks": 3,
		"timeout_seconds":      300,
		"retry_attempts":       3,
		"log_level":            "info",
	}

	switch agentType {
	case "researcher":
		config["search_depth"] = 5
		config["source_limit"] = 10
	case "coder":
		config["code_quality_threshold"] = 0.8
		config["test_coverage_threshold"] = 0.85
	case "analyst":
		config["analysis_depth"] = 3
		config["confidence_threshold"] = 0.7
	case "tester":
		config["test_coverage_target"] = 0.9
		config["performance_threshold"] = 1000
	case "designer":
		config["design_iterations"] = 5
		config["feedback_integration"] = true
	case "coordinator":
		config["max_managed_tasks"] = 10
		config["escalation_threshold"] = 0.8
	case "optimizer":
		config["optimization_target"] = "performance"
		config["improvement_threshold"] = 0.1
	}

	return config
}

func showAgentMonitor(apiClient *client.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agents, err := apiClient.GetAgents(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch agents: %w", err)
	}

	utils.PrintHeader("Agent Activity Monitor")
	fmt.Printf("Last updated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	// Summary
	activeCount := 0
	busyCount := 0
	totalTasks := 0

	for _, agent := range agents {
		if agent.Status == "active" {
			activeCount++
		}
		if agent.Status == "busy" {
			busyCount++
		}
		totalTasks += agent.TaskCount
	}

	fmt.Printf("Total Agents: %d\n", len(agents))
	fmt.Printf("Active: %d\n", activeCount)
	fmt.Printf("Busy: %d\n", busyCount)
	fmt.Printf("Total Tasks Processed: %d\n\n", totalTasks)

	// Agent details
	headers := []string{"Name", "Type", "Status", "Tasks", "Success Rate", "Avg Response Time"}
	var data [][]string

	for _, agent := range agents {
		status := agent.Status
		if agent.Status == "active" {
			status = utils.ColorGreen.Sprint(status)
		} else if agent.Status == "busy" {
			status = utils.ColorYellow.Sprint(status)
		} else if agent.Status == "error" {
			status = utils.ColorRed.Sprint(status)
		}

		data = append(data, []string{
			agent.Name,
			agent.Type,
			status,
			fmt.Sprintf("%d", agent.TaskCount),
			fmt.Sprintf("%.1f%%", agent.SuccessRate),
			fmt.Sprintf("%.1fms", agent.AvgResponseTime),
		})
	}

	if len(data) > 0 {
		fmt.Println(utils.FormatTable(headers, data))
	}

	return nil
}
