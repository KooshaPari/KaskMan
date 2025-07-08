package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecurityConfig holds security middleware configuration
type SecurityConfig struct {
	Enabled           bool            `mapstructure:"enabled" json:"enabled"`
	CORS              CORSConfig      `mapstructure:"cors" json:"cors"`
	Headers           SecurityHeaders `mapstructure:"headers" json:"headers"`
	APIKey            APIKeyConfig    `mapstructure:"api_key" json:"api_key"`
	ContentSecurity   ContentSecurity `mapstructure:"content_security" json:"content_security"`
	TrustedProxies    []string        `mapstructure:"trusted_proxies" json:"trusted_proxies"`
	AllowedHosts      []string        `mapstructure:"allowed_hosts" json:"allowed_hosts"`
	BlockedIPs        []string        `mapstructure:"blocked_ips" json:"blocked_ips"`
	BlockedUserAgents []string        `mapstructure:"blocked_user_agents" json:"blocked_user_agents"`
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled          bool          `mapstructure:"enabled" json:"enabled"`
	AllowOrigins     []string      `mapstructure:"allow_origins" json:"allow_origins"`
	AllowMethods     []string      `mapstructure:"allow_methods" json:"allow_methods"`
	AllowHeaders     []string      `mapstructure:"allow_headers" json:"allow_headers"`
	ExposeHeaders    []string      `mapstructure:"expose_headers" json:"expose_headers"`
	AllowCredentials bool          `mapstructure:"allow_credentials" json:"allow_credentials"`
	MaxAge           time.Duration `mapstructure:"max_age" json:"max_age"`
}

// SecurityHeaders holds security header configuration
type SecurityHeaders struct {
	Enabled                   bool   `mapstructure:"enabled" json:"enabled"`
	ContentTypeNosniff        bool   `mapstructure:"content_type_nosniff" json:"content_type_nosniff"`
	FrameOptions              string `mapstructure:"frame_options" json:"frame_options"`
	ContentSecurityPolicy     string `mapstructure:"content_security_policy" json:"content_security_policy"`
	ReferrerPolicy            string `mapstructure:"referrer_policy" json:"referrer_policy"`
	StrictTransportSecurity   string `mapstructure:"strict_transport_security" json:"strict_transport_security"`
	PermissionsPolicy         string `mapstructure:"permissions_policy" json:"permissions_policy"`
	CrossOriginEmbedderPolicy string `mapstructure:"cross_origin_embedder_policy" json:"cross_origin_embedder_policy"`
	CrossOriginOpenerPolicy   string `mapstructure:"cross_origin_opener_policy" json:"cross_origin_opener_policy"`
	CrossOriginResourcePolicy string `mapstructure:"cross_origin_resource_policy" json:"cross_origin_resource_policy"`
}

// APIKeyConfig holds API key authentication configuration
type APIKeyConfig struct {
	Enabled     bool              `mapstructure:"enabled" json:"enabled"`
	HeaderName  string            `mapstructure:"header_name" json:"header_name"`
	QueryParam  string            `mapstructure:"query_param" json:"query_param"`
	Keys        map[string]string `mapstructure:"keys" json:"keys"`                 // key -> description
	RequiredFor []string          `mapstructure:"required_for" json:"required_for"` // paths that require API key
	Optional    []string          `mapstructure:"optional" json:"optional"`         // paths where API key is optional
}

// ContentSecurity holds content security configuration
type ContentSecurity struct {
	Enabled              bool     `mapstructure:"enabled" json:"enabled"`
	MaxUploadSize        int64    `mapstructure:"max_upload_size" json:"max_upload_size"`
	AllowedFileTypes     []string `mapstructure:"allowed_file_types" json:"allowed_file_types"`
	AllowedImageTypes    []string `mapstructure:"allowed_image_types" json:"allowed_image_types"`
	ScanUploads          bool     `mapstructure:"scan_uploads" json:"scan_uploads"`
	BlockExecutableFiles bool     `mapstructure:"block_executable_files" json:"block_executable_files"`
}

// SecurityMiddleware creates comprehensive security middleware
func SecurityMiddleware(config SecurityConfig, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		// Apply security checks in order
		if !applyHostValidation(c, config, logger) {
			return
		}

		if !applyIPBlocking(c, config, logger) {
			return
		}

		if !applyUserAgentBlocking(c, config, logger) {
			return
		}

		if !applyAPIKeyValidation(c, config, logger) {
			return
		}

		applySecurityHeaders(c, config)

		c.Next()
	}
}

// CORSMiddleware creates CORS middleware
func CORSMiddleware(config CORSConfig, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			handlePreflightRequest(c, config, origin, logger)
			return
		}

		// Apply CORS headers for actual requests
		applyCORSHeaders(c, config, origin)

		c.Next()
	}
}

// applyHostValidation validates the Host header
func applyHostValidation(c *gin.Context, config SecurityConfig, logger *logrus.Logger) bool {
	if len(config.AllowedHosts) == 0 {
		return true
	}

	host := c.GetHeader("Host")
	if host == "" {
		logger.WithField("ip", c.ClientIP()).Warn("Request with empty Host header blocked")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Host header is required"})
		c.Abort()
		return false
	}

	// Check if host is allowed
	for _, allowedHost := range config.AllowedHosts {
		if host == allowedHost || strings.HasSuffix(host, "."+allowedHost) {
			return true
		}
	}

	logger.WithFields(logrus.Fields{
		"host": host,
		"ip":   c.ClientIP(),
	}).Warn("Request with unauthorized Host header blocked")

	c.JSON(http.StatusForbidden, gin.H{"error": "Host not allowed"})
	c.Abort()
	return false
}

// applyIPBlocking blocks requests from blocked IP addresses
func applyIPBlocking(c *gin.Context, config SecurityConfig, logger *logrus.Logger) bool {
	if len(config.BlockedIPs) == 0 {
		return true
	}

	clientIP := c.ClientIP()

	for _, blockedIP := range config.BlockedIPs {
		if clientIP == blockedIP {
			logger.WithField("ip", clientIP).Warn("Request from blocked IP address")
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return false
		}
	}

	return true
}

// applyUserAgentBlocking blocks requests from blocked user agents
func applyUserAgentBlocking(c *gin.Context, config SecurityConfig, logger *logrus.Logger) bool {
	if len(config.BlockedUserAgents) == 0 {
		return true
	}

	userAgent := c.GetHeader("User-Agent")

	for _, blockedUA := range config.BlockedUserAgents {
		if strings.Contains(strings.ToLower(userAgent), strings.ToLower(blockedUA)) {
			logger.WithFields(logrus.Fields{
				"user_agent": userAgent,
				"ip":         c.ClientIP(),
			}).Warn("Request from blocked user agent")

			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return false
		}
	}

	return true
}

// applyAPIKeyValidation validates API keys for protected endpoints
func applyAPIKeyValidation(c *gin.Context, config SecurityConfig, logger *logrus.Logger) bool {
	if !config.APIKey.Enabled {
		return true
	}

	path := c.Request.URL.Path

	// Check if API key is required for this path
	requiresAPIKey := false
	for _, requiredPath := range config.APIKey.RequiredFor {
		if strings.HasPrefix(path, requiredPath) {
			requiresAPIKey = true
			break
		}
	}

	if !requiresAPIKey {
		return true
	}

	// Extract API key from header or query parameter
	apiKey := c.GetHeader(config.APIKey.HeaderName)
	if apiKey == "" && config.APIKey.QueryParam != "" {
		apiKey = c.Query(config.APIKey.QueryParam)
	}

	if apiKey == "" {
		logger.WithFields(logrus.Fields{
			"path": path,
			"ip":   c.ClientIP(),
		}).Warn("API key required but not provided")

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "API key required",
			"details": map[string]string{
				"header": config.APIKey.HeaderName,
			},
		})
		c.Abort()
		return false
	}

	// Validate API key
	validKey := false
	var keyDescription string
	for validAPIKey, description := range config.APIKey.Keys {
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validAPIKey)) == 1 {
			validKey = true
			keyDescription = description
			break
		}
	}

	if !validKey {
		logger.WithFields(logrus.Fields{
			"path":    path,
			"ip":      c.ClientIP(),
			"api_key": apiKey[:minInt(len(apiKey), 8)] + "...", // Log only first 8 chars
		}).Warn("Invalid API key provided")

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
		c.Abort()
		return false
	}

	// Set API key info in context
	c.Set("api_key_description", keyDescription)
	c.Set("authenticated_via", "api_key")

	logger.WithFields(logrus.Fields{
		"path":     path,
		"ip":       c.ClientIP(),
		"key_desc": keyDescription,
	}).Debug("API key authentication successful")

	return true
}

// applySecurityHeaders applies security headers to the response
func applySecurityHeaders(c *gin.Context, config SecurityConfig) {
	if !config.Headers.Enabled {
		return
	}

	headers := config.Headers

	// X-Content-Type-Options
	if headers.ContentTypeNosniff {
		c.Header("X-Content-Type-Options", "nosniff")
	}

	// X-Frame-Options
	if headers.FrameOptions != "" {
		c.Header("X-Frame-Options", headers.FrameOptions)
	}

	// Content-Security-Policy
	if headers.ContentSecurityPolicy != "" {
		c.Header("Content-Security-Policy", headers.ContentSecurityPolicy)
	}

	// Referrer-Policy
	if headers.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", headers.ReferrerPolicy)
	}

	// Strict-Transport-Security (only for HTTPS)
	if headers.StrictTransportSecurity != "" && c.Request.TLS != nil {
		c.Header("Strict-Transport-Security", headers.StrictTransportSecurity)
	}

	// Permissions-Policy
	if headers.PermissionsPolicy != "" {
		c.Header("Permissions-Policy", headers.PermissionsPolicy)
	}

	// Cross-Origin-Embedder-Policy
	if headers.CrossOriginEmbedderPolicy != "" {
		c.Header("Cross-Origin-Embedder-Policy", headers.CrossOriginEmbedderPolicy)
	}

	// Cross-Origin-Opener-Policy
	if headers.CrossOriginOpenerPolicy != "" {
		c.Header("Cross-Origin-Opener-Policy", headers.CrossOriginOpenerPolicy)
	}

	// Cross-Origin-Resource-Policy
	if headers.CrossOriginResourcePolicy != "" {
		c.Header("Cross-Origin-Resource-Policy", headers.CrossOriginResourcePolicy)
	}

	// Additional security headers
	c.Header("X-XSS-Protection", "1; mode=block")
	c.Header("X-Powered-By", "") // Remove this header
}

// handlePreflightRequest handles CORS preflight requests
func handlePreflightRequest(c *gin.Context, config CORSConfig, origin string, logger *logrus.Logger) {
	// Check if origin is allowed
	if !isOriginAllowed(origin, config.AllowOrigins) {
		logger.WithFields(logrus.Fields{
			"origin": origin,
			"ip":     c.ClientIP(),
		}).Warn("CORS preflight request from unauthorized origin")

		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Apply CORS headers
	applyCORSHeaders(c, config, origin)

	// Handle preflight-specific headers
	requestMethod := c.GetHeader("Access-Control-Request-Method")
	if requestMethod != "" && isMethodAllowed(requestMethod, config.AllowMethods) {
		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
	}

	requestHeaders := c.GetHeader("Access-Control-Request-Headers")
	if requestHeaders != "" {
		c.Header("Access-Control-Allow-Headers", requestHeaders)
	}

	if config.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", fmt.Sprintf("%.0f", config.MaxAge.Seconds()))
	}

	c.AbortWithStatus(http.StatusNoContent)
}

// applyCORSHeaders applies CORS headers to the response
func applyCORSHeaders(c *gin.Context, config CORSConfig, origin string) {
	// Access-Control-Allow-Origin
	if isOriginAllowed(origin, config.AllowOrigins) {
		c.Header("Access-Control-Allow-Origin", origin)
	}

	// Access-Control-Allow-Credentials
	if config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	// Access-Control-Expose-Headers
	if len(config.ExposeHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
	}

	// Vary header for origin
	c.Header("Vary", "Origin")
}

// isOriginAllowed checks if an origin is allowed
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}

		// Support wildcard subdomains (e.g., *.example.com)
		if strings.HasPrefix(allowedOrigin, "*.") {
			domain := allowedOrigin[2:]
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	return false
}

// isMethodAllowed checks if an HTTP method is allowed
func isMethodAllowed(method string, allowedMethods []string) bool {
	for _, allowedMethod := range allowedMethods {
		if allowedMethod == method {
			return true
		}
	}
	return false
}

// ContentSecurityMiddleware validates file uploads and content
func ContentSecurityMiddleware(config ContentSecurity, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		// Validate content length
		if config.MaxUploadSize > 0 && c.Request.ContentLength > config.MaxUploadSize {
			logger.WithFields(logrus.Fields{
				"content_length": c.Request.ContentLength,
				"max_size":       config.MaxUploadSize,
				"ip":             c.ClientIP(),
			}).Warn("Request exceeds maximum upload size")

			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":    "Request too large",
				"max_size": config.MaxUploadSize,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// Helper function
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetDefaultSecurityConfig returns default security configuration
func GetDefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		Enabled: true,
		CORS: CORSConfig{
			Enabled:      true,
			AllowOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
			AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
			AllowHeaders: []string{
				"Origin", "Content-Type", "Accept", "Authorization",
				"X-Requested-With", "X-Request-ID", "X-API-Key",
			},
			ExposeHeaders: []string{
				"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining",
				"X-RateLimit-Reset",
			},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		},
		Headers: SecurityHeaders{
			Enabled:                   true,
			ContentTypeNosniff:        true,
			FrameOptions:              "DENY",
			ContentSecurityPolicy:     "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';",
			ReferrerPolicy:            "strict-origin-when-cross-origin",
			StrictTransportSecurity:   "max-age=31536000; includeSubDomains",
			PermissionsPolicy:         "geolocation=(), microphone=(), camera=()",
			CrossOriginEmbedderPolicy: "require-corp",
			CrossOriginOpenerPolicy:   "same-origin",
			CrossOriginResourcePolicy: "same-origin",
		},
		APIKey: APIKeyConfig{
			Enabled:    false,
			HeaderName: "X-API-Key",
			QueryParam: "api_key",
			Keys:       make(map[string]string),
			RequiredFor: []string{
				"/api/admin/",
				"/api/system/",
			},
		},
		ContentSecurity: ContentSecurity{
			Enabled:              true,
			MaxUploadSize:        10 * 1024 * 1024, // 10MB
			AllowedFileTypes:     []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".txt", ".csv"},
			AllowedImageTypes:    []string{".jpg", ".jpeg", ".png", ".gif"},
			ScanUploads:          false,
			BlockExecutableFiles: true,
		},
		TrustedProxies: []string{"127.0.0.1", "::1"},
		AllowedHosts:   []string{},
		BlockedIPs:     []string{},
		BlockedUserAgents: []string{
			"bot", "crawler", "spider", "scraper",
		},
	}
}
