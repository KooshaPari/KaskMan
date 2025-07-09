package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kooshapari/kodevibe-go/internal/api"
	"github.com/kooshapari/kodevibe-go/internal/config"
	"github.com/kooshapari/kodevibe-go/pkg/vibes"
)

var (
	port        = flag.Int("port", 8080, "Server port")
	host        = flag.String("host", "localhost", "Server host")
	configPath  = flag.String("config", "config.yaml", "Configuration file path")
	development = flag.Bool("dev", false, "Development mode")
	version     = "1.0.0"
	buildTime   = "unknown"
	commit      = "unknown"
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Failed to load config from %s: %v", *configPath, err)
		log.Println("Using default configuration...")
		cfg = config.Default()
	}

	// Override config with command line flags
	if *port != 8080 {
		cfg.Server.Port = *port
	}
	if *host != "localhost" {
		cfg.Server.Host = *host
	}
	cfg.Server.Development = *development

	// Initialize vibe registry
	registry := vibes.NewRegistry()
	
	// Register all available vibe checkers
	if err := registerVibeCheckers(registry); err != nil {
		log.Fatalf("Failed to register vibe checkers: %v", err)
	}

	// Create API server
	server := api.NewServer(cfg, registry, version, buildTime, commit)

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
		log.Printf("Starting KodeVibe API server on %s", addr)
		log.Printf("Version: %s, Build: %s, Commit: %s", version, buildTime, commit)
		log.Printf("Development mode: %v", cfg.Server.Development)
		
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func registerVibeCheckers(registry *vibes.Registry) error {
	// Register all vibe checkers
	checkers := []vibes.Checker{
		vibes.NewCodeChecker(),
		vibes.NewSecurityChecker(),
		vibes.NewPerformanceChecker(),
		vibes.NewFileChecker(),
		vibes.NewGitChecker(),
		vibes.NewDependencyChecker(),
		vibes.NewDocumentationChecker(),
	}

	for _, checker := range checkers {
		if err := registry.RegisterChecker(checker); err != nil {
			return fmt.Errorf("failed to register %s: %w", checker.Name(), err)
		}
		log.Printf("Registered vibe checker: %s", checker.Name())
	}

	return nil
}

func init() {
	// Check for environment variable overrides
	if envPort := os.Getenv("KODEVIBE_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			*port = p
		}
	}
	
	if envHost := os.Getenv("KODEVIBE_HOST"); envHost != "" {
		*host = envHost
	}
	
	if envConfig := os.Getenv("KODEVIBE_CONFIG"); envConfig != "" {
		*configPath = envConfig
	}
	
	if envDev := os.Getenv("KODEVIBE_DEV"); envDev == "true" {
		*development = true
	}
}