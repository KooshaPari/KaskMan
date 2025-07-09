package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/kooshapari/kodevibe-go/internal/api"
	"github.com/kooshapari/kodevibe-go/internal/config"
	"github.com/kooshapari/kodevibe-go/pkg/vibes"
)

func newServerCommand() *cobra.Command {
	var (
		port        int
		host        string
		development bool
		openBrowser bool
	)

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the KodeVibe API server",
		Long: `Start the KodeVibe HTTP API server for remote code analysis.

The server provides a REST API that can be used by IDE integrations,
CI/CD pipelines, and other tools to perform code analysis remotely.

Examples:
  kodevibe server                           # Start server on default port
  kodevibe server --port 9090              # Start server on port 9090
  kodevibe server --host 0.0.0.0           # Listen on all interfaces
  kodevibe server --dev                    # Start in development mode
  kodevibe server --open                   # Start server and open browser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(port, host, development, openBrowser)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "server port")
	cmd.Flags().StringVarP(&host, "host", "H", "localhost", "server host")
	cmd.Flags().BoolVarP(&development, "dev", "d", false, "development mode (enables debug features)")
	cmd.Flags().BoolVar(&openBrowser, "open", false, "open browser after starting server")

	return cmd
}

func runServer(port int, host string, development, openBrowser bool) error {
	// Load configuration
	cfg := config.Default()
	
	// Override with command line arguments
	cfg.Server.Port = port
	cfg.Server.Host = host
	cfg.Server.Development = development

	if verbose {
		fmt.Printf("Starting KodeVibe API server...\n")
		fmt.Printf("Host: %s\n", cfg.Server.Host)
		fmt.Printf("Port: %d\n", cfg.Server.Port)
		fmt.Printf("Development mode: %v\n", cfg.Server.Development)
	}

	// Initialize vibe registry
	registry := vibes.NewRegistry()
	
	// Register all available vibe checkers
	if err := registerAllCheckers(registry); err != nil {
		return fmt.Errorf("failed to register vibe checkers: %w", err)
	}

	if verbose {
		checkers := registry.GetCheckers()
		fmt.Printf("Registered %d vibe checkers\n", len(checkers))
	}

	// Create API server
	server := api.NewServer(cfg, registry, "1.0.0", "unknown", "unknown")

	// Setup HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Router(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in background
	go func() {
		fmt.Printf("🚀 KodeVibe API server starting on http://%s\n", addr)
		
		if cfg.Server.Development {
			fmt.Printf("📋 API Documentation: http://%s/\n", addr)
			fmt.Printf("🏥 Health Check: http://%s/api/v1/health\n", addr)
			fmt.Printf("ℹ️  Server Info: http://%s/api/v1/info\n", addr)
		}
		
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)

	// Open browser if requested
	if openBrowser {
		url := fmt.Sprintf("http://%s", addr)
		if err := openBrowserURL(url); err != nil {
			fmt.Printf("Failed to open browser: %v\n", err)
			fmt.Printf("Please open %s manually\n", url)
		} else if verbose {
			fmt.Printf("Opened browser at %s\n", url)
		}
	}

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	if !quiet {
		fmt.Println("\n🛑 Shutting down server...")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		fmt.Printf("Server forced to shutdown: %v\n", err)
		return err
	}

	if !quiet {
		fmt.Println("✅ Server shutdown complete")
	}

	return nil
}

// openBrowserURL opens the given URL in the default browser
func openBrowserURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	
	return exec.Command(cmd, args...).Start()
}