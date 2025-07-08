package security

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeAuthorization  ErrorType = "authorization"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeInternal       ErrorType = "internal"
	ErrorTypeDatabase       ErrorType = "database"
	ErrorTypeExternal       ErrorType = "external"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeCircuitBreaker ErrorType = "circuit_breaker"
)

// ErrorSeverity represents error severity levels
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// AppError represents an application error with context
type AppError struct {
	Type        ErrorType              `json:"type"`
	Code        string                 `json:"code"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Severity    ErrorSeverity          `json:"severity"`
	Timestamp   time.Time              `json:"timestamp"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Stack       string                 `json:"stack,omitempty"`
	Cause       error                  `json:"cause,omitempty"`
	Recoverable bool                   `json:"recoverable"`
	HTTPStatus  int                    `json:"http_status"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// ErrorResponse represents the JSON error response
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	RequestID string                 `json:"request_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// ErrorHandler handles application errors
type ErrorHandler struct {
	logger          *logrus.Logger
	environment     string
	circuitBreakers map[string]*gobreaker.CircuitBreaker

	// Error recovery strategies
	retryConfig     RetryConfig
	fallbackHandler FallbackHandler
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts    int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	BackoffFactor  float64
	RetryableTypes []ErrorType
}

// FallbackHandler handles fallback responses
type FallbackHandler func(c *gin.Context, err *AppError) bool

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *logrus.Logger, environment string) *ErrorHandler {
	return &ErrorHandler{
		logger:          logger,
		environment:     environment,
		circuitBreakers: make(map[string]*gobreaker.CircuitBreaker),
		retryConfig: RetryConfig{
			MaxAttempts:   3,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
			RetryableTypes: []ErrorType{
				ErrorTypeTimeout,
				ErrorTypeExternal,
				ErrorTypeDatabase,
			},
		},
	}
}

// ErrorMiddleware returns a Gin middleware for error handling
func (eh *ErrorHandler) ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process the request
		c.Next()

		// Check for errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			eh.HandleError(c, err.Err)
		}
	}
}

// HandleError handles application errors
func (eh *ErrorHandler) HandleError(c *gin.Context, err error) {
	// Convert to AppError if needed
	var appErr *AppError
	if ae, ok := err.(*AppError); ok {
		appErr = ae
	} else {
		appErr = eh.createAppError(err, c)
	}

	// Set request context
	if requestID, exists := c.Get("request_id"); exists {
		appErr.RequestID = requestID.(string)
	}

	if userID, exists := c.Get("user_id"); exists {
		appErr.UserID = userID.(string)
	}

	// Log the error
	eh.logError(appErr, c)

	// Try fallback handler first
	if eh.fallbackHandler != nil && eh.fallbackHandler(c, appErr) {
		return
	}

	// Send error response
	eh.sendErrorResponse(c, appErr)
}

// createAppError creates an AppError from a regular error
func (eh *ErrorHandler) createAppError(err error, c *gin.Context) *AppError {
	appErr := &AppError{
		Type:        ErrorTypeInternal,
		Code:        "INTERNAL_ERROR",
		Message:     "An internal error occurred",
		Severity:    SeverityHigh,
		Timestamp:   time.Now(),
		Cause:       err,
		Recoverable: false,
		HTTPStatus:  http.StatusInternalServerError,
	}

	// Determine error type from error message or type
	switch {
	case strings.Contains(err.Error(), "validation"):
		appErr.Type = ErrorTypeValidation
		appErr.Code = "VALIDATION_ERROR"
		appErr.Message = "Validation failed"
		appErr.Severity = SeverityMedium
		appErr.HTTPStatus = http.StatusBadRequest
		appErr.Recoverable = true
	case strings.Contains(err.Error(), "authentication"):
		appErr.Type = ErrorTypeAuthentication
		appErr.Code = "AUTHENTICATION_ERROR"
		appErr.Message = "Authentication failed"
		appErr.Severity = SeverityMedium
		appErr.HTTPStatus = http.StatusUnauthorized
		appErr.Recoverable = true
	case strings.Contains(err.Error(), "authorization"):
		appErr.Type = ErrorTypeAuthorization
		appErr.Code = "AUTHORIZATION_ERROR"
		appErr.Message = "Authorization failed"
		appErr.Severity = SeverityMedium
		appErr.HTTPStatus = http.StatusForbidden
		appErr.Recoverable = true
	case strings.Contains(err.Error(), "not found"):
		appErr.Type = ErrorTypeNotFound
		appErr.Code = "NOT_FOUND"
		appErr.Message = "Resource not found"
		appErr.Severity = SeverityLow
		appErr.HTTPStatus = http.StatusNotFound
		appErr.Recoverable = true
	case strings.Contains(err.Error(), "timeout"):
		appErr.Type = ErrorTypeTimeout
		appErr.Code = "TIMEOUT"
		appErr.Message = "Request timeout"
		appErr.Severity = SeverityHigh
		appErr.HTTPStatus = http.StatusRequestTimeout
		appErr.Recoverable = true
	case strings.Contains(err.Error(), "rate limit"):
		appErr.Type = ErrorTypeRateLimit
		appErr.Code = "RATE_LIMIT"
		appErr.Message = "Rate limit exceeded"
		appErr.Severity = SeverityMedium
		appErr.HTTPStatus = http.StatusTooManyRequests
		appErr.Recoverable = true
	}

	// Add stack trace in development
	if eh.environment == "development" {
		appErr.Stack = eh.getStackTrace()
	}

	return appErr
}

// logError logs the error with appropriate level
func (eh *ErrorHandler) logError(err *AppError, c *gin.Context) {
	fields := logrus.Fields{
		"error_type":    err.Type,
		"error_code":    err.Code,
		"error_message": err.Message,
		"severity":      err.Severity,
		"timestamp":     err.Timestamp,
		"request_id":    err.RequestID,
		"user_id":       err.UserID,
		"endpoint":      c.Request.URL.Path,
		"method":        c.Request.Method,
		"client_ip":     c.ClientIP(),
		"user_agent":    c.Request.UserAgent(),
		"recoverable":   err.Recoverable,
		"http_status":   err.HTTPStatus,
	}

	if err.Details != nil {
		for k, v := range err.Details {
			fields[fmt.Sprintf("detail_%s", k)] = v
		}
	}

	logger := eh.logger.WithFields(fields)

	switch err.Severity {
	case SeverityCritical:
		logger.Error("Critical error occurred")
	case SeverityHigh:
		logger.Error("High severity error occurred")
	case SeverityMedium:
		logger.Warn("Medium severity error occurred")
	case SeverityLow:
		logger.Info("Low severity error occurred")
	}

	// Log stack trace for internal errors
	if err.Type == ErrorTypeInternal && err.Stack != "" {
		eh.logger.WithField("stack", err.Stack).Error("Stack trace")
	}
}

// sendErrorResponse sends the error response to the client
func (eh *ErrorHandler) sendErrorResponse(c *gin.Context, err *AppError) {
	response := ErrorResponse{
		Error:     string(err.Type),
		Message:   err.Message,
		Code:      err.Code,
		Timestamp: err.Timestamp,
		RequestID: err.RequestID,
	}

	// Include details in development or for certain error types
	if eh.environment == "development" || err.Type == ErrorTypeValidation {
		response.Details = err.Details
	}

	// Don't expose internal error details in production
	if eh.environment == "production" && err.Type == ErrorTypeInternal {
		response.Message = "An internal error occurred"
		response.Details = nil
	}

	c.JSON(err.HTTPStatus, response)
}

// getStackTrace returns the stack trace
func (eh *ErrorHandler) getStackTrace() string {
	var buf [4096]byte
	n := runtime.Stack(buf[:], false)
	return string(buf[:n])
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details map[string]interface{}) *AppError {
	return &AppError{
		Type:        ErrorTypeValidation,
		Code:        "VALIDATION_ERROR",
		Message:     message,
		Details:     details,
		Severity:    SeverityMedium,
		Timestamp:   time.Now(),
		Recoverable: true,
		HTTPStatus:  http.StatusBadRequest,
	}
}

// NewAuthenticationError creates a new authentication error
func NewAuthenticationError(message string) *AppError {
	return &AppError{
		Type:        ErrorTypeAuthentication,
		Code:        "AUTHENTICATION_ERROR",
		Message:     message,
		Severity:    SeverityMedium,
		Timestamp:   time.Now(),
		Recoverable: true,
		HTTPStatus:  http.StatusUnauthorized,
	}
}

// NewAuthorizationError creates a new authorization error
func NewAuthorizationError(message string) *AppError {
	return &AppError{
		Type:        ErrorTypeAuthorization,
		Code:        "AUTHORIZATION_ERROR",
		Message:     message,
		Severity:    SeverityMedium,
		Timestamp:   time.Now(),
		Recoverable: true,
		HTTPStatus:  http.StatusForbidden,
	}
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource string) *AppError {
	return &AppError{
		Type:        ErrorTypeNotFound,
		Code:        "NOT_FOUND",
		Message:     fmt.Sprintf("%s not found", resource),
		Severity:    SeverityLow,
		Timestamp:   time.Now(),
		Recoverable: true,
		HTTPStatus:  http.StatusNotFound,
	}
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(retryAfter int) *AppError {
	return &AppError{
		Type:     ErrorTypeRateLimit,
		Code:     "RATE_LIMIT_EXCEEDED",
		Message:  "Rate limit exceeded",
		Severity: SeverityMedium,
		Details: map[string]interface{}{
			"retry_after": retryAfter,
		},
		Timestamp:   time.Now(),
		Recoverable: true,
		HTTPStatus:  http.StatusTooManyRequests,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message string, cause error) *AppError {
	return &AppError{
		Type:        ErrorTypeInternal,
		Code:        "INTERNAL_ERROR",
		Message:     message,
		Severity:    SeverityHigh,
		Timestamp:   time.Now(),
		Cause:       cause,
		Recoverable: false,
		HTTPStatus:  http.StatusInternalServerError,
	}
}

// NewDatabaseError creates a new database error
func NewDatabaseError(message string, cause error) *AppError {
	return &AppError{
		Type:        ErrorTypeDatabase,
		Code:        "DATABASE_ERROR",
		Message:     message,
		Severity:    SeverityHigh,
		Timestamp:   time.Now(),
		Cause:       cause,
		Recoverable: true,
		HTTPStatus:  http.StatusInternalServerError,
	}
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(operation string) *AppError {
	return &AppError{
		Type:     ErrorTypeTimeout,
		Code:     "TIMEOUT",
		Message:  fmt.Sprintf("Operation %s timed out", operation),
		Severity: SeverityHigh,
		Details: map[string]interface{}{
			"operation": operation,
		},
		Timestamp:   time.Now(),
		Recoverable: true,
		HTTPStatus:  http.StatusRequestTimeout,
	}
}

// NewCircuitBreakerError creates a new circuit breaker error
func NewCircuitBreakerError(service string) *AppError {
	return &AppError{
		Type:     ErrorTypeCircuitBreaker,
		Code:     "CIRCUIT_BREAKER_OPEN",
		Message:  fmt.Sprintf("Service %s is temporarily unavailable", service),
		Severity: SeverityHigh,
		Details: map[string]interface{}{
			"service": service,
		},
		Timestamp:   time.Now(),
		Recoverable: true,
		HTTPStatus:  http.StatusServiceUnavailable,
	}
}

// GetCircuitBreaker returns or creates a circuit breaker for a service
func (eh *ErrorHandler) GetCircuitBreaker(name string) *gobreaker.CircuitBreaker {
	if cb, exists := eh.circuitBreakers[name]; exists {
		return cb
	}

	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			eh.logger.WithFields(logrus.Fields{
				"service": name,
				"from":    from,
				"to":      to,
			}).Info("Circuit breaker state changed")
		},
	}

	cb := gobreaker.NewCircuitBreaker(settings)
	eh.circuitBreakers[name] = cb
	return cb
}

// ExecuteWithCircuitBreaker executes a function with circuit breaker protection
func (eh *ErrorHandler) ExecuteWithCircuitBreaker(serviceName string, fn func() (interface{}, error)) (interface{}, error) {
	cb := eh.GetCircuitBreaker(serviceName)

	result, err := cb.Execute(fn)
	if err != nil {
		if err == gobreaker.ErrOpenState {
			return nil, NewCircuitBreakerError(serviceName)
		}
		return nil, err
	}

	return result, nil
}

// RetryWithBackoff retries a function with exponential backoff
func (eh *ErrorHandler) RetryWithBackoff(ctx context.Context, fn func() error) error {
	var lastErr error
	delay := eh.retryConfig.InitialDelay

	for attempt := 0; attempt < eh.retryConfig.MaxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := fn(); err != nil {
			lastErr = err

			// Check if error is retryable
			if !eh.isRetryableError(err) {
				return err
			}

			// Don't delay on last attempt
			if attempt < eh.retryConfig.MaxAttempts-1 {
				time.Sleep(delay)
				delay = time.Duration(float64(delay) * eh.retryConfig.BackoffFactor)
				if delay > eh.retryConfig.MaxDelay {
					delay = eh.retryConfig.MaxDelay
				}
			}
		} else {
			return nil
		}
	}

	return lastErr
}

// isRetryableError checks if an error is retryable
func (eh *ErrorHandler) isRetryableError(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		for _, retryableType := range eh.retryConfig.RetryableTypes {
			if appErr.Type == retryableType {
				return true
			}
		}
	}
	return false
}

// PanicRecoveryMiddleware recovers from panics and converts them to errors
func (eh *ErrorHandler) PanicRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Log the panic
				eh.logger.WithFields(logrus.Fields{
					"panic":      err,
					"stack":      eh.getStackTrace(),
					"endpoint":   c.Request.URL.Path,
					"method":     c.Request.Method,
					"client_ip":  c.ClientIP(),
					"user_agent": c.Request.UserAgent(),
				}).Error("Panic recovered")

				// Convert panic to error
				appErr := &AppError{
					Type:        ErrorTypeInternal,
					Code:        "PANIC_RECOVERED",
					Message:     "An unexpected error occurred",
					Severity:    SeverityCritical,
					Timestamp:   time.Now(),
					Recoverable: false,
					HTTPStatus:  http.StatusInternalServerError,
				}

				eh.HandleError(c, appErr)
			}
		}()

		c.Next()
	}
}

// SetFallbackHandler sets a fallback handler for errors
func (eh *ErrorHandler) SetFallbackHandler(handler FallbackHandler) {
	eh.fallbackHandler = handler
}
