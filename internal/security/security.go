package security

import (
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SecurityManager coordinates all security components
type SecurityManager struct {
	RateLimiter    *RateLimiter
	InputValidator *InputValidator
	ErrorHandler   *ErrorHandler
	AuthEnhancer   *AuthEnhancer
	APIKeyManager  *APIKeyManager

	config *SecurityManagerConfig
	logger *logrus.Logger
}

// SecurityManagerConfig holds configuration for the security manager
type SecurityManagerConfig struct {
	Environment string

	// Component configurations
	RateLimit  *RateLimitConfig
	Validation *ValidationConfig
	Headers    *SecurityHeadersConfig
	Auth       *AuthConfig
	APIKey     *APIKeyConfig

	// Redis configuration (shared)
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// NewSecurityManager creates a new security manager with all components
func NewSecurityManager(db *gorm.DB, config *SecurityManagerConfig, logger *logrus.Logger) *SecurityManager {
	// Initialize rate limiter
	rateLimiter := NewRateLimiter(config.RateLimit, logger)

	// Initialize input validator
	inputValidator := NewInputValidator(config.Validation)

	// Initialize error handler
	errorHandler := NewErrorHandler(logger, config.Environment)

	// Initialize auth enhancer
	authEnhancer := NewAuthEnhancer(db, config.Auth, logger)

	// Initialize API key manager
	apiKeyManager := NewAPIKeyManager(db, config.APIKey, logger)

	return &SecurityManager{
		RateLimiter:    rateLimiter,
		InputValidator: inputValidator,
		ErrorHandler:   errorHandler,
		AuthEnhancer:   authEnhancer,
		APIKeyManager:  apiKeyManager,
		config:         config,
		logger:         logger,
	}
}

// DefaultSecurityManagerConfig returns default security manager configuration
func DefaultSecurityManagerConfig(environment string, logger *logrus.Logger) *SecurityManagerConfig {
	return &SecurityManagerConfig{
		Environment: environment,
		RateLimit: &RateLimitConfig{
			GlobalRPS:           1000,
			GlobalBurst:         2000,
			PerIPRPS:            100,
			PerIPBurst:          200,
			PerUserRPS:          500,
			PerUserBurst:        1000,
			WindowSize:          time.Minute,
			CleanupInterval:     5 * time.Minute,
			BlockDuration:       15 * time.Minute,
			SuspiciousThreshold: 5,
			EndpointLimits: map[string]EndpointLimit{
				"/api/auth/login":    {RPS: 10, Burst: 20},
				"/api/auth/register": {RPS: 5, Burst: 10},
				"/api/auth/refresh":  {RPS: 20, Burst: 40},
			},
			WhitelistedIPs:   []string{},
			WhitelistedUsers: []string{},
		},
		Validation: DefaultValidationConfig(logger),
		Headers:    DefaultSecurityHeadersConfig(environment),
		Auth: &AuthConfig{
			MFAEnabled:               false,
			MFAIssuer:                "KaskManager",
			MFASecretLength:          20,
			SessionTimeout:           24 * time.Hour,
			RefreshTokenTTL:          7 * 24 * time.Hour,
			MaxActiveSessions:        5,
			MaxLoginAttempts:         5,
			LockoutDuration:          15 * time.Minute,
			LockoutWindow:            15 * time.Minute,
			PasswordMinLength:        8,
			PasswordRequireUpper:     true,
			PasswordRequireLower:     true,
			PasswordRequireDigit:     true,
			PasswordRequireSpecial:   true,
			PasswordMaxAge:           90 * 24 * time.Hour,
			RequireEmailVerification: false,
			AllowRememberMe:          true,
			RequirePasswordChange:    false,
		},
		APIKey: DefaultAPIKeyConfig(),
	}
}

// GetStats returns comprehensive security statistics
func (sm *SecurityManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"rate_limiter":    sm.RateLimiter.GetStats(),
		"auth_enhancer":   sm.AuthEnhancer.GetStats(),
		"api_key_manager": sm.APIKeyManager.GetStats(),
		"environment":     sm.config.Environment,
		"components": map[string]bool{
			"rate_limiting":    true,
			"input_validation": true,
			"security_headers": true,
			"error_handling":   true,
			"auth_enhancement": true,
			"api_key_mgmt":     true,
		},
	}
}

// ValidateSecurityConfig validates the security configuration
func ValidateSecurityConfig(config *SecurityManagerConfig) error {
	// Validate rate limit config
	if config.RateLimit.GlobalRPS <= 0 {
		return NewValidationError("Global RPS must be positive", nil)
	}

	if config.RateLimit.PerIPRPS <= 0 {
		return NewValidationError("Per-IP RPS must be positive", nil)
	}

	// Validate auth config
	if config.Auth.PasswordMinLength < 4 {
		return NewValidationError("Password minimum length must be at least 4", nil)
	}

	if config.Auth.MaxLoginAttempts <= 0 {
		return NewValidationError("Max login attempts must be positive", nil)
	}

	// Validate security headers config
	if err := ValidateSecurityHeaders(config.Headers); err != nil {
		return err
	}

	return nil
}

// SecurityMetrics holds security-related metrics
type SecurityMetrics struct {
	// Rate limiting metrics
	TotalRequests       int64 `json:"total_requests"`
	BlockedRequests     int64 `json:"blocked_requests"`
	RateLimitedRequests int64 `json:"rate_limited_requests"`

	// Authentication metrics
	LoginAttempts  int64 `json:"login_attempts"`
	FailedLogins   int64 `json:"failed_logins"`
	LockedAccounts int64 `json:"locked_accounts"`
	ActiveSessions int64 `json:"active_sessions"`

	// API key metrics
	APIKeyRequests int64 `json:"api_key_requests"`
	InvalidAPIKeys int64 `json:"invalid_api_keys"`

	// Security events
	SQLInjectionAttempts int64 `json:"sql_injection_attempts"`
	XSSAttempts          int64 `json:"xss_attempts"`
	ValidationFailures   int64 `json:"validation_failures"`

	// Error metrics
	InternalErrors int64 `json:"internal_errors"`
	SecurityErrors int64 `json:"security_errors"`

	Timestamp time.Time `json:"timestamp"`
}

// GetSecurityMetrics returns current security metrics
func (sm *SecurityManager) GetSecurityMetrics() *SecurityMetrics {
	// This would be implemented to collect real metrics
	// For now, return a sample structure
	return &SecurityMetrics{
		TotalRequests:        0,
		BlockedRequests:      0,
		RateLimitedRequests:  0,
		LoginAttempts:        0,
		FailedLogins:         0,
		LockedAccounts:       0,
		ActiveSessions:       0,
		APIKeyRequests:       0,
		InvalidAPIKeys:       0,
		SQLInjectionAttempts: 0,
		XSSAttempts:          0,
		ValidationFailures:   0,
		InternalErrors:       0,
		SecurityErrors:       0,
		Timestamp:            time.Now(),
	}
}

// LogSecurityEvent logs a security-related event
func (sm *SecurityManager) LogSecurityEvent(eventType, description string, metadata map[string]interface{}) {
	fields := logrus.Fields{
		"event_type":  eventType,
		"description": description,
		"timestamp":   time.Now(),
		"component":   "security_manager",
	}

	// Add metadata fields
	for k, v := range metadata {
		fields[k] = v
	}

	sm.logger.WithFields(fields).Warn("Security event logged")
}

// SecurityEventTypes defines common security event types
var SecurityEventTypes = struct {
	RateLimitExceeded    string
	AuthenticationFailed string
	AuthorizationFailed  string
	InvalidAPIKey        string
	SQLInjectionAttempt  string
	XSSAttempt           string
	ValidationFailed     string
	SuspiciousActivity   string
	AccountLocked        string
	SessionExpired       string
}{
	RateLimitExceeded:    "rate_limit_exceeded",
	AuthenticationFailed: "authentication_failed",
	AuthorizationFailed:  "authorization_failed",
	InvalidAPIKey:        "invalid_api_key",
	SQLInjectionAttempt:  "sql_injection_attempt",
	XSSAttempt:           "xss_attempt",
	ValidationFailed:     "validation_failed",
	SuspiciousActivity:   "suspicious_activity",
	AccountLocked:        "account_locked",
	SessionExpired:       "session_expired",
}

// SecurityLevel represents different security levels
type SecurityLevel int

const (
	SecurityLevelLow SecurityLevel = iota
	SecurityLevelMedium
	SecurityLevelHigh
	SecurityLevelCritical
)

// String returns the string representation of security level
func (sl SecurityLevel) String() string {
	switch sl {
	case SecurityLevelLow:
		return "low"
	case SecurityLevelMedium:
		return "medium"
	case SecurityLevelHigh:
		return "high"
	case SecurityLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// GetSecurityLevel returns the current security level based on recent events
func (sm *SecurityManager) GetSecurityLevel() SecurityLevel {
	// This would implement logic to determine current security level
	// based on recent security events, failed attempts, etc.
	// For now, return medium as default
	return SecurityLevelMedium
}

// SecurityPolicy defines security policy enforcement
type SecurityPolicy struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Rules       []string  `json:"rules"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DefaultSecurityPolicies returns default security policies
func DefaultSecurityPolicies() []*SecurityPolicy {
	return []*SecurityPolicy{
		{
			Name:        "Rate Limiting",
			Description: "Enforce rate limits on API endpoints",
			Rules: []string{
				"Global rate limit: 1000 requests/minute",
				"Per-IP rate limit: 100 requests/minute",
				"Per-user rate limit: 500 requests/minute",
				"Block IPs after 5 violations for 15 minutes",
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "Input Validation",
			Description: "Validate and sanitize all input data",
			Rules: []string{
				"Check for SQL injection patterns",
				"Check for XSS patterns",
				"Enforce field length limits",
				"Sanitize HTML content",
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "Authentication Security",
			Description: "Enforce strong authentication policies",
			Rules: []string{
				"Minimum password length: 8 characters",
				"Require uppercase, lowercase, digit, and special character",
				"Lock account after 5 failed attempts",
				"Session timeout: 24 hours",
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "API Security",
			Description: "Secure API access and usage",
			Rules: []string{
				"Require API keys for programmatic access",
				"Enforce API key rate limits",
				"Log all API key usage",
				"Validate API key constraints (IP, referer)",
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

// GetSecurityPolicies returns current security policies
func (sm *SecurityManager) GetSecurityPolicies() []*SecurityPolicy {
	return DefaultSecurityPolicies()
}
