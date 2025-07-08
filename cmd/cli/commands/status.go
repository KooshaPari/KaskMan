package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/client"
	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewStatusCommand creates the status command
func NewStatusCommand(apiClient **client.Client) *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show system status",
		Long:  "Display comprehensive system status including health, metrics, and service information",
		RunE: func(cmd *cobra.Command, args []string) error {
			detailed, _ := cmd.Flags().GetBool("detailed")
			watch, _ := cmd.Flags().GetBool("watch")
			interval, _ := cmd.Flags().GetInt("interval")

			if *apiClient == nil {
				return fmt.Errorf("API client not initialized")
			}

			if watch {
				return watchStatus(*apiClient, time.Duration(interval)*time.Second, detailed)
			}

			return showStatus(*apiClient, detailed)
		},
	}

	// Add flags
	statusCmd.Flags().BoolP("detailed", "d", false, "Show detailed status information")
	statusCmd.Flags().BoolP("watch", "w", false, "Watch status in real-time")
	statusCmd.Flags().IntP("interval", "i", 5, "Refresh interval in seconds for watch mode")

	return statusCmd
}

func showStatus(apiClient *client.Client, detailed bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	spinner := utils.NewSpinner("Fetching system status...")
	spinner.Start()

	// Get health status
	health, err := apiClient.Health(ctx)
	if err != nil {
		spinner.Stop()
		return fmt.Errorf("failed to get health status: %w", err)
	}

	var systemStatus *client.SystemStatusResponse
	if detailed {
		systemStatus, err = apiClient.GetSystemStatus(ctx)
		if err != nil {
			spinner.Stop()
			utils.PrintWarning(fmt.Sprintf("Failed to get detailed status: %v", err))
		}
	}

	spinner.Stop()

	// Display basic status
	utils.PrintHeader("System Status")
	fmt.Printf("Status: %s\n", getStatusColor(health.Status))
	fmt.Printf("Version: %s\n", health.Version)
	fmt.Printf("Uptime: %s\n", health.Uptime)
	fmt.Printf("Timestamp: %s\n", health.Timestamp.Format("2006-01-02 15:04:05"))

	if !detailed || systemStatus == nil {
		return nil
	}

	// Display detailed status
	utils.PrintSubHeader("Services")
	displayServiceStatus("Database", systemStatus.Database.Connected, fmt.Sprintf("%d/%d connections", systemStatus.Database.OpenConns, systemStatus.Database.MaxOpenConns))
	displayServiceStatus("Redis", systemStatus.Redis.Connected, fmt.Sprintf("%d/%d connections", systemStatus.Redis.ActiveConns, systemStatus.Redis.PoolSize))
	displayServiceStatus("WebSocket", systemStatus.WebSocket.Enabled, fmt.Sprintf("%d clients", systemStatus.WebSocket.Clients))
	displayServiceStatus("R&D Module", systemStatus.RnD.Enabled, fmt.Sprintf("%d workers, %d jobs", systemStatus.RnD.Workers, systemStatus.RnD.ProcessingJobs))
	displayServiceStatus("Monitoring", systemStatus.Monitoring.Enabled, fmt.Sprintf("%d metrics", systemStatus.Monitoring.Metrics))

	// Display database details
	if systemStatus.Database.Connected {
		utils.PrintSubHeader("Database Details")
		fmt.Printf("Max Open Connections: %d\n", systemStatus.Database.MaxOpenConns)
		fmt.Printf("Open Connections: %d\n", systemStatus.Database.OpenConns)
		fmt.Printf("In Use: %d\n", systemStatus.Database.InUse)
		fmt.Printf("Idle: %d\n", systemStatus.Database.Idle)
		fmt.Printf("Wait Count: %d\n", systemStatus.Database.WaitCount)
		fmt.Printf("Wait Duration: %s\n", utils.FormatDuration(time.Duration(systemStatus.Database.WaitDuration)*time.Millisecond))
	}

	// Display metrics if available
	if len(systemStatus.Metrics) > 0 {
		utils.PrintSubHeader("System Metrics")
		for key, value := range systemStatus.Metrics {
			fmt.Printf("%s: %v\n", key, value)
		}
	}

	return nil
}

func watchStatus(apiClient *client.Client, interval time.Duration, detailed bool) error {
	utils.PrintHeader("Watching System Status")
	fmt.Printf("Refreshing every %s (Press Ctrl+C to stop)\n\n", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Show initial status
	if err := showStatus(apiClient, detailed); err != nil {
		return err
	}

	for {
		select {
		case <-ticker.C:
			// Clear screen
			fmt.Print("\033[H\033[2J")

			// Show updated status
			if err := showStatus(apiClient, detailed); err != nil {
				utils.PrintError(fmt.Sprintf("Failed to update status: %v", err))
				return err
			}
		}
	}
}

func displayServiceStatus(name string, enabled bool, details string) {
	status := "✗ Disabled"
	if enabled {
		status = "✓ Enabled"
	}

	if details != "" {
		fmt.Printf("%-15s: %s (%s)\n", name, getStatusColor(status), details)
	} else {
		fmt.Printf("%-15s: %s\n", name, getStatusColor(status))
	}
}

func getStatusColor(status string) string {
	switch status {
	case "healthy", "✓ Enabled":
		return utils.ColorGreen.Sprint(status)
	case "unhealthy", "✗ Disabled":
		return utils.ColorRed.Sprint(status)
	case "degraded":
		return utils.ColorYellow.Sprint(status)
	default:
		return status
	}
}
