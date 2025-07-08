package routes

import (
	"net/http"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/activity"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/api/handlers"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/api/middleware"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/auth"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/repositories"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/monitoring"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/rnd"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/websocket"
	"github.com/sirupsen/logrus"
)

// SetupRoutes configures all routes for the application
func SetupRoutes(
	router *gin.Engine,
	db *database.Database,
	wsHub *websocket.Hub,
	rndModule *rnd.Module,
	monitor *monitoring.Monitor,
	logger *logrus.Logger,
	authService *auth.Service,
	environment string,
	allowedOrigins []string,
) {
	// Create activity log repository and service
	activityRepo := repositories.NewActivityLogRepository(db.DB)
	activityService := activity.NewService(activityRepo, logger)

	// Create handlers
	h := handlers.NewHandlers(db, wsHub, rndModule, monitor, logger, authService, activityService)

	// Apply comprehensive security middleware stack
	securityMiddlewares := middleware.SecurityMiddleware(environment, logger)
	for _, mw := range securityMiddlewares {
		router.Use(mw)
	}

	// CORS middleware with security
	router.Use(middleware.SecureCORS(environment, allowedOrigins))

	// Security audit middleware
	router.Use(middleware.SecurityAuditMiddleware(logger))

	// Request logging middleware
	router.Use(middleware.Logger(logger))

	// Activity logging middleware
	router.Use(middleware.ActivityLogger(activityService))

	// Metrics middleware
	router.Use(middleware.Metrics(monitor))

	// Rate limiting middleware
	router.Use(middleware.RateLimiter(100)) // 100 requests per minute

	// Serve static files (existing web UI)
	router.Use(static.Serve("/", static.LocalFile("./", false)))
	router.Use(static.Serve("/static", static.LocalFile("./web/static", false)))

	// Serve the main dashboard HTML at root
	router.GET("/", func(c *gin.Context) {
		c.File("./dashboard-web.html")
	})

	// WebSocket endpoint
	router.GET("/ws", h.HandleWebSocket)

	// Health and monitoring endpoints
	router.GET("/health", h.GetHealth)
	router.GET("/metrics", h.GetMetrics)
	router.GET("/status", h.GetSystemStatus)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", h.Login)
			auth.POST("/register", h.Register)
			auth.POST("/logout", h.Logout)
			auth.POST("/refresh", h.RefreshToken)
		}

		// Protected routes (require authentication)
		protected := v1.Group("/")
		protected.Use(middleware.AuthRequired(authService))
		{
			// Project routes
			projects := protected.Group("/projects")
			{
				projects.GET("", h.GetProjects)
				projects.POST("", h.CreateProject)
				projects.GET("/:id", h.GetProject)
				projects.PUT("/:id", h.UpdateProject)
				projects.DELETE("/:id", h.DeleteProject)
				projects.GET("/:id/tasks", h.GetProjectTasks)
				projects.POST("/:id/tasks", h.CreateProjectTask)
			}

			// Agent routes
			agents := protected.Group("/agents")
			{
				agents.GET("", h.GetAgents)
				agents.POST("", h.CreateAgent)
				agents.GET("/:id", h.GetAgent)
				agents.PUT("/:id", h.UpdateAgent)
				agents.DELETE("/:id", h.DeleteAgent)
				agents.POST("/:id/tasks", h.AssignTaskToAgent)
			}

			// Task routes
			tasks := protected.Group("/tasks")
			{
				tasks.GET("", h.GetTasks)
				tasks.POST("", h.CreateTask)
				tasks.GET("/:id", h.GetTask)
				tasks.PUT("/:id", h.UpdateTask)
				tasks.DELETE("/:id", h.DeleteTask)
			}

			// Proposal routes
			proposals := protected.Group("/proposals")
			{
				proposals.GET("", h.GetProposals)
				proposals.POST("", h.CreateProposal)
				proposals.GET("/:id", h.GetProposal)
				proposals.PUT("/:id", h.UpdateProposal)
				proposals.DELETE("/:id", h.DeleteProposal)
				proposals.POST("/:id/approve", h.ApproveProposal)
				proposals.POST("/:id/reject", h.RejectProposal)
			}

			// R&D operations routes
			rnd := protected.Group("/rnd")
			{
				rnd.POST("/analyze", h.AnalyzePatterns)
				rnd.POST("/generate", h.GenerateProjects)
				rnd.GET("/insights", h.GetInsights)
				rnd.POST("/coordinate", h.CoordinateAgents)
				rnd.GET("/patterns", h.GetPatterns)
				rnd.GET("/stats", h.GetRnDStats)
			}

			// User routes
			users := protected.Group("/users")
			{
				users.GET("/me", h.GetProfile)
				users.PUT("/me/password", h.UpdatePassword)
				users.GET("", h.GetUsers)
				users.POST("", h.CreateUser)
				users.GET("/:id", h.GetUser)
				users.PUT("/:id", h.UpdateUser)
				users.DELETE("/:id", h.DeleteUser)
			}

			// Activity routes
			activities := protected.Group("/activities")
			{
				activities.GET("", h.GetActivities)
				activities.GET("/recent", h.GetRecentActivities)
			}

			// System management routes
			system := protected.Group("/system")
			{
				system.GET("/info", h.GetSystemInfo)
				system.GET("/stats", h.GetSystemStats)
				system.POST("/backup", h.CreateBackup)
				system.POST("/restore", h.RestoreBackup)
			}
		}

		// Admin routes (require admin role)
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired(authService))
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/users", h.GetAllUsers)
			admin.POST("/users/:id/activate", h.ActivateUser)
			admin.POST("/users/:id/deactivate", h.DeactivateUser)
			admin.GET("/logs", h.GetSystemLogs)
			admin.POST("/maintenance", h.StartMaintenance)
			admin.POST("/maintenance/stop", h.StopMaintenance)
		}
	}

	// Dashboard API routes (for the existing web UI)
	dashboard := router.Group("/dashboard")
	{
		dashboard.GET("/data", h.GetDashboardData)
		dashboard.GET("/projects", h.GetDashboardProjects)
		dashboard.GET("/agents", h.GetDashboardAgents)
		dashboard.GET("/metrics", h.GetDashboardMetrics)
		dashboard.GET("/activities", h.GetDashboardActivities)
	}

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		// For API routes, return JSON error
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "endpoint not found",
				"path":  c.Request.URL.Path,
			})
			return
		}

		// For other routes, serve the main dashboard
		c.File("./dashboard-web.html")
	})

	logger.Info("Routes configured successfully")
}
