package middleware

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ErrorConfig holds error handling configuration
type ErrorConfig struct {
	Enabled           bool `mapstructure:"enabled" json:"enabled"`
	IncludeStackTrace bool `mapstructure:"include_stack_trace" json:"include_stack_trace"`
	LogStackTrace     bool `mapstructure:"log_stack_trace" json:"log_stack_trace"`
	MaxStackDepth     int  `mapstructure:"max_stack_depth" json:"max_stack_depth"`
	EnableRecovery    bool `mapstructure:"enable_recovery" json:"enable_recovery"`
	EnableMetrics     bool `mapstructure:"enable_metrics" json:"enable_metrics"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error       string                 `json:"error"`
	Message     string                 `json:"message"`
	Code        string                 `json:"code,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Path        string                 `json:"path"`
	Method      string                 `json:"method"`
	Details     map[string]interface{} `json:"details,omitempty"`
	StackTrace  []string               `json:"stack_trace,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// ErrorMetrics holds error metrics
type ErrorMetrics struct {
	TotalErrors    int64            `json:"total_errors"`
	ErrorsByStatus map[int]int64    `json:"errors_by_status"`
	ErrorsByPath   map[string]int64 `json:"errors_by_path"`
	ErrorsByType   map[string]int64 `json:"errors_by_type"`
	RecentErrors   []RecentError    `json:"recent_errors"`
	LastUpdate     time.Time        `json:"last_update"`
}

// RecentError holds information about recent errors
type RecentError struct {
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	Method    string    `json:"method"`
	Status    int       `json:"status"`
	Error     string    `json:"error"`
	UserID    string    `json:"user_id,omitempty"`
	IP        string    `json:"ip"`
}

// AppError represents an application error with additional context
type AppError struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	StatusCode int                    `json:"status_code"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	Context    context.Context        `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// ErrorHandler provides comprehensive error handling
type ErrorHandler struct {
	config  ErrorConfig
	logger  *logrus.Logger
	metrics *ErrorMetrics
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(config ErrorConfig, logger *logrus.Logger) *ErrorHandler {
	return &ErrorHandler{
		config: config,
		logger: logger,
		metrics: &ErrorMetrics{
			ErrorsByStatus: make(map[int]int64),
			ErrorsByPath:   make(map[string]int64),
			ErrorsByType:   make(map[string]int64),
			RecentErrors:   make([]RecentError, 0),
			LastUpdate:     time.Now(),
		},
	}
}

// ErrorHandlingMiddleware creates error handling middleware
func ErrorHandlingMiddleware(config ErrorConfig, logger *logrus.Logger) gin.HandlerFunc {
	handler := NewErrorHandler(config, logger)

	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		// Add request ID for tracking
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Process request and handle any errors
		c.Next()

		// Check if there were any errors during request processing
		if len(c.Errors) > 0 {
			handler.handleErrors(c)
		}
	}
}

// RecoveryMiddleware creates panic recovery middleware
func RecoveryMiddleware(config ErrorConfig, logger *logrus.Logger) gin.HandlerFunc {
	handler := NewErrorHandler(config, logger)

	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if !config.EnableRecovery {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		handler.handlePanic(c, recovered)
	})
}

// handleErrors processes errors from the Gin context
func (eh *ErrorHandler) handleErrors(c *gin.Context) {
	var mainError error
	var appError *AppError

	// Get the first error
	if len(c.Errors) > 0 {
		mainError = c.Errors[0].Err

		// Check if it's an AppError
		if ae, ok := mainError.(*AppError); ok {
			appError = ae
		} else {
			// Convert to AppError
			appError = &AppError{
				Code:       "INTERNAL_ERROR",
				Message:    mainError.Error(),
				StatusCode: http.StatusInternalServerError,
				Cause:      mainError,
			}
		}
	} else {
		// Generic error
		appError = &AppError{
			Code:       "UNKNOWN_ERROR",
			Message:    "An unknown error occurred",
			StatusCode: http.StatusInternalServerError,
		}
	}

	// Determine status code if not already set
	if c.Writer.Status() == http.StatusOK {
		c.Status(appError.StatusCode)
	}

	// Create error response
	response := eh.createErrorResponse(c, appError)

	// Log the error
	eh.logError(c, appError, response)

	// Update metrics
	if eh.config.EnableMetrics {
		eh.updateMetrics(c, appError)
	}

	// Send response
	c.JSON(c.Writer.Status(), response)
}

// handlePanic processes panic recovery
func (eh *ErrorHandler) handlePanic(c *gin.Context, recovered interface{}) {
	// Create stack trace
	stack := make([]byte, 4096)
	length := runtime.Stack(stack, false)
	stackTrace := string(stack[:length])

	// Create error
	appError := &AppError{
		Code:       "PANIC_RECOVERED",
		Message:    fmt.Sprintf("Internal server error: %v", recovered),
		StatusCode: http.StatusInternalServerError,
		Details: map[string]interface{}{
			"panic_value": recovered,
		},
	}

	// Create error response
	response := eh.createErrorResponse(c, appError)

	// Add stack trace if enabled
	if eh.config.IncludeStackTrace {
		response.StackTrace = eh.parseStackTrace(stackTrace)
	}

	// Log the panic
	eh.logger.WithFields(logrus.Fields{
		"request_id": response.RequestID,
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"ip":         c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"panic":      recovered,
		"stack":      stackTrace,
	}).Error("Panic recovered")

	// Update metrics
	if eh.config.EnableMetrics {
		eh.updateMetrics(c, appError)
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, response)
}

// createErrorResponse creates a standardized error response
func (eh *ErrorHandler) createErrorResponse(c *gin.Context, appError *AppError) *ErrorResponse {
	requestID, _ := c.Get("request_id")
	requestIDStr, _ := requestID.(string)

	response := &ErrorResponse{
		Error:     appError.Code,
		Message:   appError.Message,
		Code:      appError.Code,
		RequestID: requestIDStr,
		Timestamp: time.Now(),
		Path:      c.Request.URL.Path,
		Method:    c.Request.Method,
		Details:   appError.Details,
	}

	// Add suggestions based on error type
	response.Suggestions = eh.getSuggestions(appError)

	return response
}

// getSuggestions returns helpful suggestions based on the error
func (eh *ErrorHandler) getSuggestions(appError *AppError) []string {
	switch appError.Code {
	case "VALIDATION_ERROR":
		return []string{
			"Check the request payload format",
			"Ensure all required fields are provided",
			"Verify data types match the expected format",
		}
	case "AUTHENTICATION_ERROR":
		return []string{
			"Verify your authentication credentials",
			"Check if your session has expired",
			"Ensure you have the required permissions",
		}
	case "RATE_LIMIT_EXCEEDED":
		return []string{
			"Wait before making additional requests",
			"Consider implementing request throttling",
			"Contact support for higher rate limits",
		}
	case "RESOURCE_NOT_FOUND":
		return []string{
			"Check if the resource ID is correct",
			"Verify the resource exists",
			"Ensure you have permission to access this resource",
		}
	case "DATABASE_ERROR":
		return []string{
			"Try the request again in a few moments",
			"Contact support if the issue persists",
		}
	default:
		return []string{
			"Try the request again",
			"Contact support if the issue persists",
		}
	}
}

// parseStackTrace parses a stack trace string into lines
func (eh *ErrorHandler) parseStackTrace(stackTrace string) []string {
	lines := make([]string, 0)

	// Split stack trace into lines and limit depth
	for i, line := range []string{stackTrace} {
		if eh.config.MaxStackDepth > 0 && i >= eh.config.MaxStackDepth {
			break
		}
		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

// logError logs the error with appropriate level and context
func (eh *ErrorHandler) logError(c *gin.Context, appError *AppError, response *ErrorResponse) {
	// Determine log level based on status code
	var logLevel logrus.Level
	switch {
	case appError.StatusCode >= 500:
		logLevel = logrus.ErrorLevel
	case appError.StatusCode >= 400:
		logLevel = logrus.WarnLevel
	default:
		logLevel = logrus.InfoLevel
	}

	// Get user context if available
	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")

	// Create log entry
	entry := eh.logger.WithFields(logrus.Fields{
		"request_id":    response.RequestID,
		"path":          response.Path,
		"method":        response.Method,
		"status_code":   appError.StatusCode,
		"error_code":    appError.Code,
		"ip":            c.ClientIP(),
		"user_agent":    c.GetHeader("User-Agent"),
		"user_id":       userID,
		"user_role":     userRole,
		"error_details": appError.Details,
	})

	// Add stack trace to log if enabled
	if eh.config.LogStackTrace && appError.Cause != nil {
		stack := make([]byte, 4096)
		length := runtime.Stack(stack, false)
		entry = entry.WithField("stack_trace", string(stack[:length]))
	}

	// Log the error
	entry.Log(logLevel, appError.Message)
}

// updateMetrics updates error metrics
func (eh *ErrorHandler) updateMetrics(c *gin.Context, appError *AppError) {
	eh.metrics.TotalErrors++
	eh.metrics.ErrorsByStatus[appError.StatusCode]++
	eh.metrics.ErrorsByPath[c.Request.URL.Path]++
	eh.metrics.ErrorsByType[appError.Code]++

	// Add to recent errors (keep last 100)
	userID, _ := c.Get("user_id")
	userIDStr := ""
	if userID != nil {
		userIDStr = fmt.Sprintf("%v", userID)
	}

	recentError := RecentError{
		Timestamp: time.Now(),
		Path:      c.Request.URL.Path,
		Method:    c.Request.Method,
		Status:    appError.StatusCode,
		Error:     appError.Code,
		UserID:    userIDStr,
		IP:        c.ClientIP(),
	}

	eh.metrics.RecentErrors = append(eh.metrics.RecentErrors, recentError)
	if len(eh.metrics.RecentErrors) > 100 {
		eh.metrics.RecentErrors = eh.metrics.RecentErrors[1:]
	}

	eh.metrics.LastUpdate = time.Now()
}

// GetMetrics returns current error metrics
func (eh *ErrorHandler) GetMetrics() *ErrorMetrics {
	return eh.metrics
}

// Helper functions for creating specific error types

// NewValidationError creates a validation error
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		StatusCode: http.StatusBadRequest,
		Details:    details,
	}
}

// NewAuthenticationError creates an authentication error
func NewAuthenticationError(message string) *AppError {
	return &AppError{
		Code:       "AUTHENTICATION_ERROR",
		Message:    message,
		StatusCode: http.StatusUnauthorized,
	}
}

// NewAuthorizationError creates an authorization error
func NewAuthorizationError(message string) *AppError {
	return &AppError{
		Code:       "AUTHORIZATION_ERROR",
		Message:    message,
		StatusCode: http.StatusForbidden,
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Code:       "RESOURCE_NOT_FOUND",
		Message:    fmt.Sprintf("%s not found", resource),
		StatusCode: http.StatusNotFound,
		Details: map[string]interface{}{
			"resource": resource,
		},
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(limit int64, window time.Duration) *AppError {
	return &AppError{
		Code:       "RATE_LIMIT_EXCEEDED",
		Message:    "Rate limit exceeded",
		StatusCode: http.StatusTooManyRequests,
		Details: map[string]interface{}{
			"limit":  limit,
			"window": window.String(),
		},
	}
}

// NewDatabaseError creates a database error
func NewDatabaseError(operation string, cause error) *AppError {
	return &AppError{
		Code:       "DATABASE_ERROR",
		Message:    fmt.Sprintf("Database operation failed: %s", operation),
		StatusCode: http.StatusInternalServerError,
		Cause:      cause,
		Details: map[string]interface{}{
			"operation": operation,
		},
	}
}

// NewInternalError creates a generic internal error
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    message,
		StatusCode: http.StatusInternalServerError,
		Cause:      cause,
	}
}

// GetDefaultErrorConfig returns default error handling configuration
func GetDefaultErrorConfig() ErrorConfig {
	return ErrorConfig{
		Enabled:           true,
		IncludeStackTrace: false, // Don't expose stack traces in production
		LogStackTrace:     true,
		MaxStackDepth:     10,
		EnableRecovery:    true,
		EnableMetrics:     true,
	}
}
