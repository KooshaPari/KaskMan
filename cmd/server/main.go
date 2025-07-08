package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/api/routes"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/auth"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/config"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/monitoring"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/security"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/websocket"
	"github.com/kooshapari/kaskmanager-rd-platform/pkg/logger"
	"github.com/sirupsen/logrus"
)

// Server represents the main application server
type Server struct {
	config          *config.Config
	logger          *logrus.Logger
	router          *gin.Engine
	httpServer      *http.Server
	db              *database.Database
	wsHub           *websocket.Hub
	rndModule       *rnd.Module
	monitor         *monitoring.Monitor
	authService     *auth.Service
	securityManager *security.SecurityManager
}

// NewServer creates a new server instance
func NewServer() (*Server, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)

	// Initialize database
	db, err := database.NewDatabase(cfg.Database, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub(log)

	// Initialize R&D module
	rndModule, err := rnd.NewModule(cfg.RnD, db, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize R&D module: %w", err)
	}

	// Initialize monitoring
	monitor := monitoring.NewMonitor(cfg.Monitoring, log)

	// Initialize auth service
	authService := auth.NewService(db.DB, cfg.Auth.JWTSecret)

	// Create default admin user if it doesn't exist
	if err := authService.CreateDefaultAdmin(); err != nil {
		log.WithError(err).Warn("Failed to create default admin user")
	}

	// Initialize security manager
	securityConfig := convertToSecurityConfig(cfg)
	securityManager := security.NewSecurityManager(db.DB, securityConfig, log)

	// Validate security configuration
	if err := security.ValidateSecurityConfig(securityConfig); err != nil {
		return nil, fmt.Errorf("invalid security configuration: %w", err)
	}

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.New()

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	server := &Server{
		config:          cfg,
		logger:          log,
		router:          router,
		httpServer:      httpServer,
		db:              db,
		wsHub:           wsHub,
		rndModule:       rndModule,
		monitor:         monitor,
		authService:     authService,
		securityManager: securityManager,
	}

	// Setup routes with security
	allowedOrigins := []string{} // Configure based on environment
	if cfg.Environment == "development" {
		allowedOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
	} else {
		// Production origins from config
		allowedOrigins = cfg.Security.CORS.AllowedOrigins
	}

	routes.SetupRoutes(router, server.db, server.wsHub, server.rndModule, server.monitor, log, server.authService, cfg.Environment, allowedOrigins)

	return server, nil
}

// Start starts the server and all background services
func (s *Server) Start() error {
	s.logger.Info("Starting KaskManager R&D Platform Server...")

	// Start WebSocket hub
	go s.wsHub.Start()

	// Start R&D module
	if err := s.rndModule.Start(); err != nil {
		return fmt.Errorf("failed to start R&D module: %w", err)
	}

	// Start monitoring
	s.monitor.Start()

	// Log security status
	s.logger.WithFields(logrus.Fields{
		"security_level": s.securityManager.GetSecurityLevel().String(),
		"policies":       len(s.securityManager.GetSecurityPolicies()),
	}).Info("Security manager initialized")

	// Start HTTP server
	s.logger.WithField("port", s.config.Server.Port).Info("Starting HTTP server")
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	s.logger.Info("KaskManager R&D Platform Server started successfully")
	return nil
}

// Stop gracefully stops the server and all services
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Shutting down KaskManager R&D Platform Server...")

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to shutdown HTTP server gracefully")
	}

	// Stop WebSocket hub
	s.wsHub.Stop()

	// Stop R&D module
	s.rndModule.Stop()

	// Stop monitoring
	s.monitor.Stop()

	// Close database
	if err := s.db.Close(); err != nil {
		s.logger.WithError(err).Error("Failed to close database connection")
	}

	s.logger.Info("KaskManager R&D Platform Server stopped")
	return nil
}

func main() {
	// Create server
	server, err := NewServer()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create server")
	}

	// Start server
	if err := server.Start(); err != nil {
		logrus.WithError(err).Fatal("Failed to start server")
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Stop(ctx); err != nil {
		logrus.WithError(err).Error("Failed to stop server gracefully")
		os.Exit(1)
	}

	logrus.Info("Server exited cleanly")
}

// convertToSecurityConfig converts application config to security manager config
func convertToSecurityConfig(cfg *config.Config) *security.SecurityManagerConfig {
	return &security.SecurityManagerConfig{
		Environment: cfg.Environment,
		RateLimit: &security.RateLimitConfig{
			GlobalRPS:           cfg.Security.RateLimit.GlobalRPS,
			GlobalBurst:         cfg.Security.RateLimit.GlobalBurst,
			PerIPRPS:            cfg.Security.RateLimit.PerIPRPS,
			PerIPBurst:          cfg.Security.RateLimit.PerIPBurst,
			PerUserRPS:          cfg.Security.RateLimit.PerUserRPS,
			PerUserBurst:        cfg.Security.RateLimit.PerUserBurst,
			WindowSize:          cfg.Security.RateLimit.WindowSize,
			CleanupInterval:     cfg.Security.RateLimit.CleanupInterval,
			BlockDuration:       cfg.Security.RateLimit.BlockDuration,
			SuspiciousThreshold: 5,
			EndpointLimits: map[string]security.EndpointLimit{
				"/api/auth/login":    {RPS: 10, Burst: 20},
				"/api/auth/register": {RPS: 5, Burst: 10},
				"/api/auth/refresh":  {RPS: 20, Burst: 40},
			},
			WhitelistedIPs:   cfg.Security.RateLimit.WhitelistedIPs,
			WhitelistedUsers: cfg.Security.RateLimit.WhitelistedUsers,
			RedisAddr:        cfg.Redis.Addr,
			RedisPassword:    cfg.Redis.Password,
			RedisDB:          cfg.Redis.DB,
		},
		Validation: &security.ValidationConfig{
			SQLInjectionPatterns: []string{
				`(\b(SELECT|INSERT|UPDATE|DELETE|DROP|CREATE|ALTER|EXEC|UNION|SCRIPT)\b)`,
				`(\b(OR|AND)\s+\d+\s*=\s*\d+)`,
				`(\b(OR|AND)\s+['"]?\w+['"]?\s*=\s*['"]?\w+['"]?)`,
				`(--|#|\/\*|\*\/)`,
			},
			XSSPatterns: []string{
				`<\s*script[^>]*>.*?<\s*/\s*script\s*>`,
				`<\s*iframe[^>]*>.*?<\s*/\s*iframe\s*>`,
				`javascript:`,
				`on\w+\s*=`,
			},
			MaxFieldLengths: map[string]int{
				"email":       255,
				"username":    50,
				"password":    255,
				"name":        100,
				"title":       255,
				"description": 5000,
			},
			AllowedFileTypes:       cfg.Security.Validation.AllowedFileTypes,
			MaxFileSize:            10 * 1024 * 1024, // 10MB
			EnableHTMLSanitization: cfg.Security.Validation.EnableHTMLSanitization,
			StrictMode:             cfg.Security.Validation.StrictMode,
		},
		Headers: &security.SecurityHeadersConfig{
			ContentSecurityPolicy: cfg.Security.Headers.ContentSecurityPolicy,
			HSTSMaxAge:            cfg.Security.Headers.HSTSMaxAge,
			HSTSIncludeSubdomains: cfg.Security.Headers.HSTSIncludeSubdomains,
			HSTSPreload:           cfg.Security.Headers.HSTSPreload,
			XFrameOptions:         cfg.Security.Headers.XFrameOptions,
			XContentTypeOptions:   cfg.Security.Headers.XContentTypeOptions,
			XXSSProtection:        cfg.Security.Headers.XXSSProtection,
			ReferrerPolicy:        cfg.Security.Headers.ReferrerPolicy,
			PermissionsPolicy:     cfg.Security.Headers.PermissionsPolicy,
			CustomHeaders:         cfg.Security.Headers.CustomHeaders,
			IsDevelopment:         cfg.Environment == "development",
			IsProduction:          cfg.Environment == "production",
		},
		Auth: &security.AuthConfig{
			MFAEnabled:               false, // Can be enabled later
			MFAIssuer:                "KaskManager",
			MFASecretLength:          20,
			SessionTimeout:           24 * time.Hour,
			RefreshTokenTTL:          cfg.Auth.RefreshExpiration,
			MaxActiveSessions:        5,
			MaxLoginAttempts:         cfg.Auth.MaxLoginAttempts,
			LockoutDuration:          cfg.Auth.LoginAttemptWindow,
			LockoutWindow:            cfg.Auth.LoginAttemptWindow,
			PasswordMinLength:        cfg.Auth.PasswordMinLength,
			PasswordRequireUpper:     true,
			PasswordRequireLower:     true,
			PasswordRequireDigit:     true,
			PasswordRequireSpecial:   true,
			PasswordMaxAge:           90 * 24 * time.Hour,
			RequireEmailVerification: cfg.Auth.RequireEmailVerify,
			AllowRememberMe:          true,
			RequirePasswordChange:    false,
			RedisAddr:                cfg.Redis.Addr,
			RedisPassword:            cfg.Redis.Password,
			RedisDB:                  cfg.Redis.DB,
		},
		APIKey: &security.APIKeyConfig{
			KeyLength:        32,
			PrefixLength:     8,
			DefaultTTL:       365 * 24 * time.Hour,
			MaxTTL:           5 * 365 * 24 * time.Hour,
			DefaultRateLimit: 1000,
			MaxRateLimit:     10000,
			HashKeys:         true,
			RequireUserAgent: false,
			RequireReferer:   false,
			AllowedIPs:       []string{},
			RedisAddr:        cfg.Redis.Addr,
			RedisPassword:    cfg.Redis.Password,
			RedisDB:          cfg.Redis.DB,
		},
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
	}
}
