package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/activity"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/auth"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/monitoring"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/security"
	"github.com/sirupsen/logrus"
)

// Logger returns a gin middleware for logging HTTP requests
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.WithFields(logrus.Fields{
			"status_code": param.StatusCode,
			"latency":     param.Latency,
			"client_ip":   param.ClientIP,
			"method":      param.Method,
			"path":        param.Path,
			"user_agent":  param.Request.UserAgent(),
			"error":       param.ErrorMessage,
		}).Info("HTTP Request")
		return ""
	})
}

// Metrics returns a gin middleware for collecting HTTP metrics
func Metrics(monitor *monitoring.Monitor) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment active requests
		monitor.IncrementActiveRequests()
		defer monitor.DecrementActiveRequests()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)
		success := c.Writer.Status() < 400
		monitor.RecordRequest(duration, success)
	}
}

// AuthRequired returns a gin middleware that requires authentication
func AuthRequired(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from header
		token := c.GetHeader("Authorization")
		if token == "" {
			// Try to get from cookie
			cookie, err := c.Cookie("auth_token")
			if err != nil || cookie == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Authentication required",
					"message": "No authentication token provided",
				})
				c.Abort()
				return
			}
			token = cookie
		}

		// Remove "Bearer " prefix if present
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		}

		// Validate JWT token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid authentication token",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// AdminRequired returns a gin middleware that requires admin role
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Admin access required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SecureCORS returns a gin middleware for secure CORS handling
func SecureCORS(environment string, allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Default to restrictive CORS in production
		allowedOrigin := ""
		if environment == "development" {
			// Allow localhost in development
			if origin == "http://localhost:3000" || origin == "http://localhost:8080" || origin == "http://127.0.0.1:3000" || origin == "http://127.0.0.1:8080" {
				allowedOrigin = origin
			}
		} else {
			// Check against whitelist in production
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					allowedOrigin = origin
					break
				}
			}
		}

		// Set CORS headers
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Access-Control-Allow-Credentials", "true")
		} else {
			c.Header("Access-Control-Allow-Origin", "null")
			c.Header("Access-Control-Allow-Credentials", "false")
		}

		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORS returns a gin middleware for handling CORS (legacy function for backward compatibility)
func CORS() gin.HandlerFunc {
	return SecureCORS("development", []string{})
}

// RateLimiter returns a gin middleware for rate limiting
func RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	// Create default rate limiter config
	config := &security.RateLimitConfig{
		GlobalRPS:           requestsPerMinute,
		GlobalBurst:         requestsPerMinute * 2,
		PerIPRPS:            requestsPerMinute / 2,
		PerIPBurst:          requestsPerMinute,
		PerUserRPS:          requestsPerMinute,
		PerUserBurst:        requestsPerMinute * 2,
		WindowSize:          time.Minute,
		CleanupInterval:     5 * time.Minute,
		BlockDuration:       15 * time.Minute,
		SuspiciousThreshold: 5,
	}

	// Create rate limiter instance
	rateLimiter := security.NewRateLimiter(config, nil)
	return rateLimiter.Middleware()
}

// ActivityAware returns a gin middleware that provides activity logging capability to handlers
func ActivityAware(activityService *activity.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set activity service in context so handlers can use it
		c.Set("activity_service", activityService)
		c.Next()
	}
}

// RequestID returns a gin middleware that adds a unique request ID
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	// Simple timestamp-based ID for demo
	return time.Now().Format("20060102150405.000")
}

// ActivityLogger returns a gin middleware for logging activities
func ActivityLogger(activityService *activity.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain requests
		if !activityService.ShouldLogRequest(c) {
			c.Next()
			return
		}

		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration and response size
		duration := time.Since(start)
		responseSize := int64(c.Writer.Size())

		// Get any error from the context
		var err error
		if len(c.Errors) > 0 {
			err = c.Errors[0]
		}

		// Log the HTTP request activity
		activityService.LogHTTPRequest(
			c.Request.Context(),
			c,
			c.Writer.Status(),
			duration,
			responseSize,
			err,
		)
	}
}

// ActivityLoggerWithOptions returns a gin middleware for logging activities with custom options
func ActivityLoggerWithOptions(activityService *activity.Service, options ActivityLoggerOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if we should log this request
		if !options.ShouldLog(c) {
			c.Next()
			return
		}

		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration and response size
		duration := time.Since(start)
		responseSize := int64(c.Writer.Size())

		// Get any error from the context
		var err error
		if len(c.Errors) > 0 {
			err = c.Errors[0]
		}

		// Log the HTTP request activity
		activityService.LogHTTPRequest(
			c.Request.Context(),
			c,
			c.Writer.Status(),
			duration,
			responseSize,
			err,
		)
	}
}

// ActivityLoggerOptions provides configuration for the activity logger middleware
type ActivityLoggerOptions struct {
	// ShouldLog determines if a request should be logged
	ShouldLog func(c *gin.Context) bool
	// IncludeBody determines if request/response bodies should be logged
	IncludeBody bool
	// SkipPaths defines paths to skip logging
	SkipPaths []string
	// SkipMethods defines HTTP methods to skip logging
	SkipMethods []string
	// MaxBodySize defines the maximum body size to log
	MaxBodySize int64
}

// DefaultActivityLoggerOptions returns default options for activity logging
func DefaultActivityLoggerOptions() ActivityLoggerOptions {
	return ActivityLoggerOptions{
		ShouldLog: func(c *gin.Context) bool {
			path := c.Request.URL.Path
			method := c.Request.Method

			// Skip logging for health checks and static assets
			skipPaths := []string{
				"/health",
				"/metrics",
				"/favicon.ico",
				"/static/",
				"/css/",
				"/js/",
				"/img/",
				"/assets/",
			}

			for _, skipPath := range skipPaths {
				if strings.HasPrefix(path, skipPath) {
					return false
				}
			}

			// Skip logging for OPTIONS requests (CORS preflight)
			if method == "OPTIONS" {
				return false
			}

			return true
		},
		IncludeBody: false,
		MaxBodySize: 10 * 1024, // 10KB
	}
}

// SecurityMiddleware returns a comprehensive security middleware stack
func SecurityMiddleware(environment string, logger *logrus.Logger) []gin.HandlerFunc {
	// Security headers config
	headersConfig := security.DefaultSecurityHeadersConfig(environment)

	// Input validation config
	validationConfig := security.DefaultValidationConfig(logger)

	// Error handler
	errorHandler := security.NewErrorHandler(logger, environment)

	// Input validator
	inputValidator := security.NewInputValidator(validationConfig)

	return []gin.HandlerFunc{
		// Panic recovery (should be first)
		errorHandler.PanicRecoveryMiddleware(),

		// Request timeout
		security.TimeoutMiddleware(30 * time.Second),

		// Request size limit
		security.SizeLimitMiddleware(10 * 1024 * 1024), // 10MB

		// Security headers
		security.SecurityHeaders(headersConfig),

		// No cache for sensitive endpoints
		security.NoCacheMiddleware(),

		// Input validation
		inputValidator.ValidationMiddleware(),

		// Error handling (should be last)
		errorHandler.ErrorMiddleware(),
	}
}

// SecurityAuditMiddleware returns middleware for security audit logging
func SecurityAuditMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Get request info
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		method := c.Request.Method
		path := c.Request.URL.Path

		// Process request
		c.Next()

		// Log security-relevant requests
		if shouldAuditRequest(path, method, c.Writer.Status()) {
			logger.WithFields(logrus.Fields{
				"security_audit": true,
				"client_ip":      clientIP,
				"user_agent":     userAgent,
				"method":         method,
				"path":           path,
				"status_code":    c.Writer.Status(),
				"duration":       time.Since(start).Milliseconds(),
				"user_id":        c.GetString("user_id"),
			}).Info("Security audit log")
		}
	}
}

// shouldAuditRequest determines if a request should be audited
func shouldAuditRequest(path, method string, statusCode int) bool {
	// Always audit authentication attempts
	if strings.HasPrefix(path, "/api/auth/") {
		return true
	}

	// Audit admin endpoints
	if strings.HasPrefix(path, "/api/admin/") {
		return true
	}

	// Audit failed requests
	if statusCode >= 400 {
		return true
	}

	// Audit sensitive operations
	sensitivePaths := []string{
		"/api/users/",
		"/api/system/",
	}

	for _, sensitivePath := range sensitivePaths {
		if strings.HasPrefix(path, sensitivePath) {
			return true
		}
	}

	return false
}
