package security

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig holds configuration for security headers
type SecurityHeadersConfig struct {
	// Content Security Policy
	ContentSecurityPolicy string

	// HSTS (HTTP Strict Transport Security)
	HSTSMaxAge            int  // Max age in seconds
	HSTSIncludeSubdomains bool // Include subdomains
	HSTSPreload           bool // Enable preload

	// X-Frame-Options
	XFrameOptions string // DENY, SAMEORIGIN, or ALLOW-FROM uri

	// X-Content-Type-Options
	XContentTypeOptions string // nosniff

	// X-XSS-Protection
	XXSSProtection string // 1; mode=block

	// Referrer-Policy
	ReferrerPolicy string

	// Permissions-Policy (formerly Feature-Policy)
	PermissionsPolicy string

	// Cross-Origin policies
	CrossOriginEmbedderPolicy string // require-corp, unsafe-none
	CrossOriginOpenerPolicy   string // same-origin, same-origin-allow-popups, unsafe-none
	CrossOriginResourcePolicy string // same-site, same-origin, cross-origin

	// Server header
	ServerHeader string // Custom server header or empty to remove

	// Additional custom headers
	CustomHeaders map[string]string

	// Environment-specific settings
	IsDevelopment bool
	IsProduction  bool
}

// DefaultSecurityHeadersConfig returns default security headers configuration
func DefaultSecurityHeadersConfig(environment string) *SecurityHeadersConfig {
	isDev := environment == "development"
	isProd := environment == "production"

	config := &SecurityHeadersConfig{
		HSTSMaxAge:                31536000, // 1 year
		HSTSIncludeSubdomains:     true,
		HSTSPreload:               isProd,
		XFrameOptions:             "DENY",
		XContentTypeOptions:       "nosniff",
		XXSSProtection:            "1; mode=block",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		ServerHeader:              "KaskManager/1.0",
		IsDevelopment:             isDev,
		IsProduction:              isProd,
	}

	// Development vs Production CSP
	if isDev {
		config.ContentSecurityPolicy = "default-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self' ws: wss:; " +
			"font-src 'self'; " +
			"object-src 'none'; " +
			"base-uri 'self'"
	} else {
		config.ContentSecurityPolicy = "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"connect-src 'self' wss:; " +
			"font-src 'self'; " +
			"object-src 'none'; " +
			"base-uri 'self'; " +
			"frame-ancestors 'none'; " +
			"upgrade-insecure-requests"
	}

	// Production-specific permissions policy
	if isProd {
		config.PermissionsPolicy = "geolocation=(), microphone=(), camera=(), " +
			"payment=(), usb=(), magnetometer=(), gyroscope=(), " +
			"speaker=(), notifications=(), push=(), sync-xhr=()"
	} else {
		config.PermissionsPolicy = "geolocation=(), microphone=(), camera=()"
	}

	return config
}

// SecurityHeaders returns a Gin middleware that adds security headers
func SecurityHeaders(config *SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Content Security Policy
		if config.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", config.ContentSecurityPolicy)
		}

		// HSTS (only for HTTPS)
		if c.Request.TLS != nil || isHTTPS(c.Request) {
			hstsValue := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
			if config.HSTSIncludeSubdomains {
				hstsValue += "; includeSubDomains"
			}
			if config.HSTSPreload {
				hstsValue += "; preload"
			}
			c.Header("Strict-Transport-Security", hstsValue)
		}

		// X-Frame-Options
		if config.XFrameOptions != "" {
			c.Header("X-Frame-Options", config.XFrameOptions)
		}

		// X-Content-Type-Options
		if config.XContentTypeOptions != "" {
			c.Header("X-Content-Type-Options", config.XContentTypeOptions)
		}

		// X-XSS-Protection
		if config.XXSSProtection != "" {
			c.Header("X-XSS-Protection", config.XXSSProtection)
		}

		// Referrer-Policy
		if config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", config.ReferrerPolicy)
		}

		// Permissions-Policy
		if config.PermissionsPolicy != "" {
			c.Header("Permissions-Policy", config.PermissionsPolicy)
		}

		// Cross-Origin policies
		if config.CrossOriginEmbedderPolicy != "" {
			c.Header("Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)
		}

		if config.CrossOriginOpenerPolicy != "" {
			c.Header("Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
		}

		if config.CrossOriginResourcePolicy != "" {
			c.Header("Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
		}

		// Server header
		if config.ServerHeader != "" {
			c.Header("Server", config.ServerHeader)
		} else {
			// Remove server header
			c.Header("Server", "")
		}

		// Remove potentially sensitive headers
		c.Header("X-Powered-By", "")

		// Custom headers
		for key, value := range config.CustomHeaders {
			c.Header(key, value)
		}

		// Add cache control for API responses
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// isHTTPS checks if the request is over HTTPS by examining various headers
func isHTTPS(r *http.Request) bool {
	// Check X-Forwarded-Proto header (common with load balancers)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		return true
	}

	// Check X-Forwarded-SSL header
	if ssl := r.Header.Get("X-Forwarded-SSL"); ssl == "on" {
		return true
	}

	// Check X-Forwarded-Scheme header
	if scheme := r.Header.Get("X-Forwarded-Scheme"); scheme == "https" {
		return true
	}

	return false
}

// SecureCSPMiddleware returns a more restrictive CSP middleware for API endpoints
func SecureCSPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For API endpoints, use a very restrictive CSP
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		}

		c.Next()
	}
}

// NoCacheMiddleware adds no-cache headers for sensitive endpoints
func NoCacheMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add no-cache headers for sensitive endpoints
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/auth/") ||
			strings.HasPrefix(path, "/api/users/") ||
			strings.HasPrefix(path, "/api/admin/") ||
			strings.Contains(path, "/password") ||
			strings.Contains(path, "/token") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate, private")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		}

		c.Next()
	}
}

// SecurityEventMiddleware logs security-related events
func SecurityEventMiddleware(logger interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Track security-sensitive endpoints
		path := c.Request.URL.Path
		method := c.Request.Method

		// Log authentication attempts
		if strings.HasPrefix(path, "/api/auth/") {
			// This will be handled by the authentication middleware
		}

		// Log admin access attempts
		if strings.HasPrefix(path, "/api/admin/") {
			// Log admin access
		}

		// Log file upload attempts
		if method == "POST" && strings.Contains(path, "/upload") {
			// Log file uploads
		}

		c.Next()
	}
}

// TimeoutMiddleware adds request timeout handling
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set timeout using context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		// Channel to receive completion signal
		done := make(chan bool, 1)

		go func() {
			c.Next()
			done <- true
		}()

		select {
		case <-done:
			// Request completed normally
			return
		case <-ctx.Done():
			// Request timed out
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error":   "Request timeout",
				"message": "The request took too long to process",
			})
			c.Abort()
			return
		}
	}
}

// SizeLimitMiddleware limits request body size
func SizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Limit request body size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		c.Next()
	}
}

// ValidateSecurityHeaders validates security headers configuration
func ValidateSecurityHeaders(config *SecurityHeadersConfig) error {
	// Validate X-Frame-Options
	if config.XFrameOptions != "" {
		validOptions := []string{"DENY", "SAMEORIGIN"}
		if !strings.HasPrefix(config.XFrameOptions, "ALLOW-FROM") {
			found := false
			for _, option := range validOptions {
				if config.XFrameOptions == option {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("invalid X-Frame-Options value: %s", config.XFrameOptions)
			}
		}
	}

	// Validate HSTS settings
	if config.HSTSMaxAge < 0 {
		return fmt.Errorf("HSTS max-age must be non-negative")
	}

	// Validate CSP
	if config.ContentSecurityPolicy != "" {
		if !strings.Contains(config.ContentSecurityPolicy, "default-src") {
			return fmt.Errorf("CSP must contain default-src directive")
		}
	}

	return nil
}
