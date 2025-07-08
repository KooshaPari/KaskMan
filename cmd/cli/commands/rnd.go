package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewRnDCommand creates the R&D command
func NewRnDCommand(apiClient **client.Client, format *string) *cobra.Command {
	rndCmd := &cobra.Command{
		Use:   "rnd",
		Short: "R&D operations commands",
		Long:  "Commands for R&D operations including pattern analysis, project generation, and insights",
	}

	// Analyze command
	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze patterns in the system",
		Long:  "Trigger pattern analysis to identify trends and insights",
		RunE: func(cmd *cobra.Command, args []string) error {
			patternType, _ := cmd.Flags().GetString("type")
			analysisContext, _ := cmd.Flags().GetString("context")
			depth, _ := cmd.Flags().GetInt("depth")
			timeRange, _ := cmd.Flags().GetString("time-range")
			interactive, _ := cmd.Flags().GetBool("interactive")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Interactive mode
			if interactive {
				var err error
				if patternType == "" {
					types := []string{"user_behavior", "system_usage", "project_trend", "performance", "error_pattern"}
					choice, err := utils.PromptChoice("Pattern type", types)
					if err != nil {
						return err
					}
					patternType = types[choice]
				}
				if analysisContext == "" {
					analysisContext, err = utils.PromptString("Analysis context (optional)")
					if err != nil {
						return err
					}
				}
				if depth == 0 {
					depth = 5 // default depth
				}
				if timeRange == "" {
					timeRanges := []string{"1h", "24h", "7d", "30d", "all"}
					choice, err := utils.PromptChoice("Time range", timeRanges)
					if err != nil {
						return err
					}
					timeRange = timeRanges[choice]
				}
			}

			// Set defaults
			if patternType == "" {
				patternType = "system_usage"
			}
			if depth == 0 {
				depth = 5
			}
			if timeRange == "" {
				timeRange = "24h"
			}

			// Create analysis request
			analysisReq := client.AnalyzePatternsRequest{
				Type:      patternType,
				Context:   analysisContext,
				Depth:     depth,
				TimeRange: timeRange,
				Filters:   make(map[string]interface{}),
			}

			spinner := utils.NewSpinner("Analyzing patterns...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 5 minutes timeout
			defer cancel()

			result, err := (*apiClient).AnalyzePatterns(ctx, analysisReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to analyze patterns: %w", err)
			}

			// Display results
			utils.PrintHeader("Pattern Analysis Results")
			fmt.Printf("Patterns Found: %d\n", result.PatternsFound)
			fmt.Printf("Analysis Type: %s\n", patternType)
			fmt.Printf("Time Range: %s\n", timeRange)
			fmt.Printf("Analysis Depth: %d\n", depth)

			if len(result.Patterns) > 0 {
				utils.PrintSubHeader("Discovered Patterns")
				for _, pattern := range result.Patterns {
					fmt.Printf("• %s (%s) - Confidence: %.2f, Frequency: %d\n",
						pattern.Name, pattern.Type, pattern.Confidence, pattern.Frequency)
				}
			}

			if len(result.Insights) > 0 {
				utils.PrintSubHeader("Generated Insights")
				for _, insight := range result.Insights {
					fmt.Printf("• %s (%s) - Impact: %s, Confidence: %.2f\n",
						insight.Title, insight.Type, insight.Impact, insight.Confidence)
					if len(insight.ActionItems) > 0 {
						fmt.Printf("  Actions: %s\n", strings.Join(insight.ActionItems, ", "))
					}
				}
			}

			return nil
		},
	}

	// Generate command
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate project suggestions",
		Long:  "Generate project suggestions based on patterns and insights",
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")
			focus, _ := cmd.Flags().GetString("focus")
			priority, _ := cmd.Flags().GetString("priority")
			maxProjects, _ := cmd.Flags().GetInt("max-projects")
			interactive, _ := cmd.Flags().GetBool("interactive")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Interactive mode
			if interactive {
				var err error
				if category == "" {
					categories := []string{"research", "development", "analysis", "innovation", "optimization"}
					choice, err := utils.PromptChoice("Project category", categories)
					if err != nil {
						return err
					}
					category = categories[choice]
				}
				if focus == "" {
					focus, err = utils.PromptString("Focus area (optional)")
					if err != nil {
						return err
					}
				}
				if priority == "" {
					priorities := []string{"low", "medium", "high", "critical"}
					choice, err := utils.PromptChoice("Priority", priorities)
					if err != nil {
						return err
					}
					priority = priorities[choice]
				}
				if maxProjects == 0 {
					maxProjects = 5
				}
			}

			// Set defaults
			if category == "" {
				category = "development"
			}
			if priority == "" {
				priority = "medium"
			}
			if maxProjects == 0 {
				maxProjects = 5
			}

			// Create generation request
			genReq := client.GenerateProjectsRequest{
				Category:     category,
				Focus:        focus,
				Priority:     priority,
				MaxProjects:  maxProjects,
				Constraints:  make(map[string]interface{}),
				Requirements: []string{},
			}

			spinner := utils.NewSpinner("Generating project suggestions...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			result, err := (*apiClient).GenerateProjects(ctx, genReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to generate projects: %w", err)
			}

			// Display results
			utils.PrintHeader("Generated Project Suggestions")
			fmt.Printf("Projects Generated: %d\n", result.ProjectsGenerated)
			fmt.Printf("Category: %s\n", category)
			fmt.Printf("Priority: %s\n", priority)

			if len(result.Projects) > 0 {
				utils.PrintSubHeader("Project Suggestions")
				for i, project := range result.Projects {
					fmt.Printf("\n%d. %s\n", i+1, project.Name)
					fmt.Printf("   Type: %s\n", project.Type)
					fmt.Printf("   Priority: %s\n", project.Priority)
					fmt.Printf("   Description: %s\n", project.Description)
					fmt.Printf("   Estimated Hours: %d\n", project.EstimatedHours)
					fmt.Printf("   Budget: $%.2f\n", project.Budget)
					if len(project.Tags) > 0 {
						fmt.Printf("   Tags: %s\n", strings.Join(project.Tags, ", "))
					}
					fmt.Printf("   Justification: %s\n", project.Justification)
					fmt.Printf("   Expected Outcome: %s\n", project.ExpectedOutcome)
				}
			}

			return nil
		},
	}

	// Insights command
	insightsCmd := &cobra.Command{
		Use:   "insights",
		Short: "List generated insights",
		Long:  "List all generated insights from pattern analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			impact, _ := cmd.Flags().GetString("impact")
			insightType, _ := cmd.Flags().GetString("type")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching insights...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			insights, err := (*apiClient).GetInsights(ctx)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch insights: %w", err)
			}

			// Apply filters
			if impact != "" || insightType != "" {
				filtered := insights[:0]
				for _, insight := range insights {
					if impact != "" && insight.Impact != impact {
						continue
					}
					if insightType != "" && insight.Type != insightType {
						continue
					}
					filtered = append(filtered, insight)
				}
				insights = filtered
			}

			if len(insights) == 0 {
				utils.PrintInfo("No insights found.")
				return nil
			}

			// Display insights
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(insights)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				utils.PrintHeader("Generated Insights")
				for _, insight := range insights {
					fmt.Printf("\n• %s (%s)\n", insight.Title, insight.Type)
					fmt.Printf("  Impact: %s\n", insight.Impact)
					fmt.Printf("  Confidence: %.2f\n", insight.Confidence)
					fmt.Printf("  Description: %s\n", insight.Description)
					fmt.Printf("  Actionable: %t\n", insight.IsActionable)
					fmt.Printf("  Implemented: %t\n", insight.IsImplemented)
					fmt.Printf("  Created: %s\n", insight.CreatedAt.Format("2006-01-02 15:04:05"))
				}
			}

			utils.PrintInfo(fmt.Sprintf("Found %d insights", len(insights)))
			return nil
		},
	}

	// Patterns command
	patternsCmd := &cobra.Command{
		Use:   "patterns",
		Short: "List discovered patterns",
		Long:  "List all discovered patterns from analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			patternType, _ := cmd.Flags().GetString("type")
			minConfidence, _ := cmd.Flags().GetFloat64("min-confidence")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching patterns...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			patterns, err := (*apiClient).GetPatterns(ctx)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch patterns: %w", err)
			}

			// Apply filters
			if patternType != "" || minConfidence > 0 {
				filtered := patterns[:0]
				for _, pattern := range patterns {
					if patternType != "" && pattern.Type != patternType {
						continue
					}
					if minConfidence > 0 && pattern.Confidence < minConfidence {
						continue
					}
					filtered = append(filtered, pattern)
				}
				patterns = filtered
			}

			if len(patterns) == 0 {
				utils.PrintInfo("No patterns found.")
				return nil
			}

			// Display patterns
			switch *format {
			case "json":
				jsonOutput, err := utils.FormatJSON(patterns)
				if err != nil {
					return err
				}
				fmt.Println(jsonOutput)
			default:
				utils.PrintHeader("Discovered Patterns")
				for _, pattern := range patterns {
					fmt.Printf("\n• %s (%s)\n", pattern.Name, pattern.Type)
					fmt.Printf("  Confidence: %.2f\n", pattern.Confidence)
					fmt.Printf("  Frequency: %d\n", pattern.Frequency)
					fmt.Printf("  Significance: %.2f\n", pattern.Significance)
					fmt.Printf("  Description: %s\n", pattern.Description)
					fmt.Printf("  Last Seen: %s\n", pattern.LastSeen.Format("2006-01-02 15:04:05"))
				}
			}

			utils.PrintInfo(fmt.Sprintf("Found %d patterns", len(patterns)))
			return nil
		},
	}

	// Coordinate command
	coordinateCmd := &cobra.Command{
		Use:   "coordinate",
		Short: "Coordinate agents for a task",
		Long:  "Coordinate multiple agents to work on a task or project",
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, _ := cmd.Flags().GetString("task")
			projectID, _ := cmd.Flags().GetString("project")
			agentTypes, _ := cmd.Flags().GetStringSlice("agent-types")
			strategy, _ := cmd.Flags().GetString("strategy")
			interactive, _ := cmd.Flags().GetBool("interactive")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			// Interactive mode
			if interactive {
				if taskID == "" && projectID == "" {
					choice, err := utils.PromptChoice("Coordinate for", []string{"Task", "Project"})
					if err != nil {
						return err
					}
					if choice == 0 {
						var err error
						taskID, err = utils.PromptString("Task ID")
						if err != nil {
							return err
						}
					} else {
						var err error
						projectID, err = utils.PromptString("Project ID")
						if err != nil {
							return err
						}
					}
				}
				if len(agentTypes) == 0 {
					typesStr, err := utils.PromptString("Agent types (comma-separated)")
					if err != nil {
						return err
					}
					if typesStr != "" {
						agentTypes = strings.Split(typesStr, ",")
						for i, t := range agentTypes {
							agentTypes[i] = strings.TrimSpace(t)
						}
					}
				}
				if strategy == "" {
					strategies := []string{"sequential", "parallel", "hierarchical", "collaborative"}
					choice, err := utils.PromptChoice("Coordination strategy", strategies)
					if err != nil {
						return err
					}
					strategy = strategies[choice]
				}
			}

			// Validate required fields
			if taskID == "" && projectID == "" {
				return fmt.Errorf("either task ID or project ID is required")
			}
			if strategy == "" {
				strategy = "collaborative"
			}

			// Create coordination request
			coordReq := client.CoordinateAgentsRequest{
				AgentTypes: agentTypes,
				Strategy:   strategy,
				Config:     make(map[string]interface{}),
			}

			if taskID != "" {
				if taskUUID, err := uuid.Parse(taskID); err == nil {
					coordReq.TaskID = taskUUID
				} else {
					return fmt.Errorf("invalid task ID: %w", err)
				}
			}

			if projectID != "" {
				if projectUUID, err := uuid.Parse(projectID); err == nil {
					coordReq.ProjectID = projectUUID
				} else {
					return fmt.Errorf("invalid project ID: %w", err)
				}
			}

			spinner := utils.NewSpinner("Coordinating agents...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := (*apiClient).CoordinateAgents(ctx, coordReq)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to coordinate agents: %w", err)
			}

			// Display results
			utils.PrintHeader("Agent Coordination Results")
			fmt.Printf("Coordination ID: %s\n", result.CoordinationID)
			fmt.Printf("Strategy: %s\n", result.Strategy)
			fmt.Printf("Status: %s\n", result.Status)
			fmt.Printf("Assigned Agents: %d\n", len(result.AssignedAgents))

			if len(result.AssignedAgents) > 0 {
				utils.PrintSubHeader("Agent Assignments")
				for _, assignment := range result.AssignedAgents {
					fmt.Printf("• %s (%s) - Role: %s\n",
						assignment.AgentName, assignment.AgentID, assignment.Role)
					if len(assignment.Tasks) > 0 {
						fmt.Printf("  Tasks: %s\n", strings.Join(assignment.Tasks, ", "))
					}
				}
			}

			return nil
		},
	}

	// Stats command
	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show R&D statistics",
		Long:  "Display comprehensive R&D statistics and performance metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			spinner := utils.NewSpinner("Fetching R&D statistics...")
			spinner.Start()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			stats, err := (*apiClient).GetRnDStats(ctx)
			spinner.Stop()

			if err != nil {
				return fmt.Errorf("failed to fetch R&D statistics: %w", err)
			}

			// Display statistics
			utils.PrintHeader("R&D Statistics")
			fmt.Printf("Total Patterns: %d\n", stats.TotalPatterns)
			fmt.Printf("Total Insights: %d\n", stats.TotalInsights)
			fmt.Printf("Active Agents: %d\n", stats.ActiveAgents)
			fmt.Printf("Processing Jobs: %d\n", stats.ProcessingJobs)
			fmt.Printf("Completed Analyses: %d\n", stats.CompletedAnalyses)
			fmt.Printf("Average Processing Time: %.2f seconds\n", stats.AverageProcessingTime)

			if len(stats.PatternsByType) > 0 {
				utils.PrintSubHeader("Patterns by Type")
				for patternType, count := range stats.PatternsByType {
					fmt.Printf("  %s: %d\n", patternType, count)
				}
			}

			if len(stats.InsightsByImpact) > 0 {
				utils.PrintSubHeader("Insights by Impact")
				for impact, count := range stats.InsightsByImpact {
					fmt.Printf("  %s: %d\n", impact, count)
				}
			}

			if len(stats.RecentActivity) > 0 {
				utils.PrintSubHeader("Recent Activity")
				for _, activity := range stats.RecentActivity {
					status := "✓"
					if !activity.Success {
						status = "✗"
					}
					fmt.Printf("  %s %s %s - %s\n",
						status, activity.Type, activity.Action,
						activity.Timestamp.Format("2006-01-02 15:04:05"))
				}
			}

			return nil
		},
	}

	// Add flags
	analyzeCmd.Flags().StringP("type", "t", "", "Pattern type (user_behavior, system_usage, project_trend, performance, error_pattern)")
	analyzeCmd.Flags().StringP("context", "c", "", "Analysis context")
	analyzeCmd.Flags().IntP("depth", "d", 5, "Analysis depth")
	analyzeCmd.Flags().StringP("time-range", "r", "24h", "Time range (1h, 24h, 7d, 30d, all)")
	analyzeCmd.Flags().BoolP("interactive", "i", false, "Interactive mode")

	generateCmd.Flags().StringP("category", "c", "", "Project category (research, development, analysis, innovation, optimization)")
	generateCmd.Flags().StringP("focus", "f", "", "Focus area")
	generateCmd.Flags().StringP("priority", "p", "", "Priority (low, medium, high, critical)")
	generateCmd.Flags().IntP("max-projects", "m", 5, "Maximum projects to generate")
	generateCmd.Flags().BoolP("interactive", "i", false, "Interactive mode")

	insightsCmd.Flags().StringP("impact", "i", "", "Filter by impact (low, medium, high, critical)")
	insightsCmd.Flags().StringP("type", "t", "", "Filter by type (optimization, recommendation, warning, trend)")

	patternsCmd.Flags().StringP("type", "t", "", "Filter by pattern type")
	patternsCmd.Flags().Float64P("min-confidence", "c", 0, "Minimum confidence threshold")

	coordinateCmd.Flags().StringP("task", "t", "", "Task ID")
	coordinateCmd.Flags().StringP("project", "p", "", "Project ID")
	coordinateCmd.Flags().StringSliceP("agent-types", "a", []string{}, "Agent types to coordinate")
	coordinateCmd.Flags().StringP("strategy", "s", "", "Coordination strategy (sequential, parallel, hierarchical, collaborative)")
	coordinateCmd.Flags().BoolP("interactive", "i", false, "Interactive mode")

	// Add subcommands
	rndCmd.AddCommand(analyzeCmd)
	rndCmd.AddCommand(generateCmd)
	rndCmd.AddCommand(insightsCmd)
	rndCmd.AddCommand(patternsCmd)
	rndCmd.AddCommand(coordinateCmd)
	rndCmd.AddCommand(statsCmd)

	return rndCmd
}
