package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/kooshapari/kaskmanager-rd-platform/cmd/cli/utils"
	"github.com/spf13/cobra"
)

// NewStartCommand creates the start command
func NewStartCommand() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the KaskManager server",
		Long:  "Start the KaskManager server with various options",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, _ := cmd.Flags().GetInt("port")
			host, _ := cmd.Flags().GetString("host")
			daemon, _ := cmd.Flags().GetBool("daemon")
			config, _ := cmd.Flags().GetString("config")
			logLevel, _ := cmd.Flags().GetString("log-level")

			utils.PrintHeader("Starting KaskManager Server")

			// Build command arguments
			var cmdArgs []string

			// Use the server binary if it exists, otherwise use go run
			serverBinary := "./server"
			if _, err := os.Stat(serverBinary); os.IsNotExist(err) {
				// Try to find the server binary
				if _, err := os.Stat("./cmd/server/main.go"); err == nil {
					cmdArgs = []string{"run", "cmd/server/main.go"}
				} else {
					return fmt.Errorf("server binary not found and source code not available")
				}
			}

			// Set environment variables
			env := os.Environ()
			if port != 8080 {
				env = append(env, fmt.Sprintf("SERVER_PORT=%d", port))
			}
			if host != "0.0.0.0" {
				env = append(env, fmt.Sprintf("SERVER_HOST=%s", host))
			}
			if logLevel != "" {
				env = append(env, fmt.Sprintf("LOG_LEVEL=%s", logLevel))
			}
			if config != "" {
				env = append(env, fmt.Sprintf("CONFIG_PATH=%s", config))
			}

			// Create command
			var command *exec.Cmd
			if len(cmdArgs) > 0 {
				command = exec.Command("go", cmdArgs...)
			} else {
				command = exec.Command(serverBinary)
			}

			command.Env = env

			if daemon {
				// Start as daemon
				utils.PrintInfo("Starting server in daemon mode...")

				// Redirect output to log files
				logFile, err := os.OpenFile("kaskman.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					return fmt.Errorf("failed to create log file: %w", err)
				}
				defer logFile.Close()

				command.Stdout = logFile
				command.Stderr = logFile

				if err := command.Start(); err != nil {
					return fmt.Errorf("failed to start server: %w", err)
				}

				utils.PrintSuccess(fmt.Sprintf("Server started in daemon mode (PID: %d)", command.Process.Pid))
				utils.PrintInfo(fmt.Sprintf("Server running on %s:%d", host, port))
				utils.PrintInfo("Logs are written to kaskman.log")

				// Save PID file
				pidFile := "kaskman.pid"
				if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", command.Process.Pid)), 0644); err != nil {
					utils.PrintWarning(fmt.Sprintf("Failed to write PID file: %v", err))
				}

				return nil
			} else {
				// Start in foreground
				utils.PrintInfo(fmt.Sprintf("Starting server on %s:%d", host, port))
				utils.PrintInfo("Press Ctrl+C to stop")

				command.Stdout = os.Stdout
				command.Stderr = os.Stderr

				if err := command.Start(); err != nil {
					return fmt.Errorf("failed to start server: %w", err)
				}

				utils.PrintSuccess("Server started successfully")

				// Wait for interrupt signal
				quit := make(chan os.Signal, 1)
				signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

				go func() {
					<-quit
					utils.PrintInfo("Shutting down server...")

					// Try graceful shutdown first
					if err := command.Process.Signal(syscall.SIGTERM); err != nil {
						utils.PrintWarning("Failed to send SIGTERM, forcing shutdown...")
						command.Process.Kill()
					}
				}()

				// Wait for command to finish
				if err := command.Wait(); err != nil {
					utils.PrintError(fmt.Sprintf("Server exited with error: %v", err))
					return err
				}

				utils.PrintSuccess("Server stopped")
				return nil
			}
		},
	}

	// Stop command
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the KaskManager server",
		Long:  "Stop a running KaskManager server daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			pidFile := "kaskman.pid"

			// Read PID file
			pidData, err := os.ReadFile(pidFile)
			if err != nil {
				if os.IsNotExist(err) {
					utils.PrintWarning("PID file not found. Server may not be running.")
					return nil
				}
				return fmt.Errorf("failed to read PID file: %w", err)
			}

			var pid int
			if _, err := fmt.Sscanf(string(pidData), "%d", &pid); err != nil {
				return fmt.Errorf("invalid PID in file: %w", err)
			}

			// Find process
			process, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("failed to find process: %w", err)
			}

			utils.PrintInfo(fmt.Sprintf("Stopping server (PID: %d)...", pid))

			// Send SIGTERM
			if err := process.Signal(syscall.SIGTERM); err != nil {
				utils.PrintWarning("Failed to send SIGTERM, trying SIGKILL...")
				if err := process.Signal(syscall.SIGKILL); err != nil {
					return fmt.Errorf("failed to kill process: %w", err)
				}
			}

			// Wait for process to exit
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				_, err := process.Wait()
				done <- err
			}()

			select {
			case err := <-done:
				if err != nil {
					utils.PrintWarning(fmt.Sprintf("Process wait error: %v", err))
				}
			case <-ctx.Done():
				utils.PrintWarning("Timeout waiting for process to stop")
			}

			// Remove PID file
			if err := os.Remove(pidFile); err != nil {
				utils.PrintWarning(fmt.Sprintf("Failed to remove PID file: %v", err))
			}

			utils.PrintSuccess("Server stopped")
			return nil
		},
	}

	// Restart command
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the KaskManager server",
		Long:  "Stop and start the KaskManager server",
		RunE: func(cmd *cobra.Command, args []string) error {
			utils.PrintHeader("Restarting KaskManager Server")

			// Stop first
			if err := stopCmd.RunE(cmd, args); err != nil {
				utils.PrintWarning(fmt.Sprintf("Stop failed: %v", err))
			}

			// Wait a moment
			time.Sleep(2 * time.Second)

			// Start again
			return startCmd.RunE(cmd, args)
		},
	}

	// Add flags
	startCmd.Flags().IntP("port", "p", 8080, "Port to run the server on")
	startCmd.Flags().StringP("host", "H", "0.0.0.0", "Host to bind the server to")
	startCmd.Flags().BoolP("daemon", "d", false, "Run server in daemon mode")
	startCmd.Flags().StringP("config", "c", "", "Path to configuration file")
	startCmd.Flags().StringP("log-level", "l", "", "Log level (debug, info, warn, error)")

	// Copy flags to restart command
	restartCmd.Flags().IntP("port", "p", 8080, "Port to run the server on")
	restartCmd.Flags().StringP("host", "H", "0.0.0.0", "Host to bind the server to")
	restartCmd.Flags().BoolP("daemon", "d", false, "Run server in daemon mode")
	restartCmd.Flags().StringP("config", "c", "", "Path to configuration file")
	restartCmd.Flags().StringP("log-level", "l", "", "Log level (debug, info, warn, error)")

	// Create parent command
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Server management commands",
		Long:  "Commands for managing the KaskManager server",
	}

	// Add subcommands
	serverCmd.AddCommand(startCmd)
	serverCmd.AddCommand(stopCmd)
	serverCmd.AddCommand(restartCmd)

	return serverCmd
}
