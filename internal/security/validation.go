package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/microcosm-cc/bluemonday"
	"github.com/sirupsen/logrus"
)

// ValidationConfig holds configuration for input validation
type ValidationConfig struct {
	// SQL injection patterns
	SQLInjectionPatterns []string

	// XSS patterns
	XSSPatterns []string

	// Maximum field lengths
	MaxFieldLengths map[string]int

	// File upload settings
	AllowedFileTypes []string
	MaxFileSize      int64

	// Custom validation rules
	CustomValidators map[string]func(interface{}) bool

	// Sanitization settings
	EnableHTMLSanitization bool
	StrictMode             bool

	// Logging
	Logger *logrus.Logger
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value,omitempty"`
	Tag     string      `json:"tag"`
	Message string      `json:"message"`
}

// ValidationResponse represents a validation error response
type ValidationResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors"`
}

// InputValidator handles input validation and sanitization
type InputValidator struct {
	config      *ValidationConfig
	validator   *validator.Validate
	sanitizer   *bluemonday.Policy
	sqlPatterns []*regexp.Regexp
	xssPatterns []*regexp.Regexp
	logger      *logrus.Logger
}

// NewInputValidator creates a new input validator
func NewInputValidator(config *ValidationConfig) *InputValidator {
	iv := &InputValidator{
		config:    config,
		validator: validator.New(),
		logger:    config.Logger,
	}

	// Initialize HTML sanitizer
	iv.sanitizer = bluemonday.StrictPolicy()
	if !config.StrictMode {
		iv.sanitizer = bluemonday.UGCPolicy()
	}

	// Compile SQL injection patterns
	iv.sqlPatterns = make([]*regexp.Regexp, len(config.SQLInjectionPatterns))
	for i, pattern := range config.SQLInjectionPatterns {
		iv.sqlPatterns[i] = regexp.MustCompile(`(?i)` + pattern)
	}

	// Compile XSS patterns
	iv.xssPatterns = make([]*regexp.Regexp, len(config.XSSPatterns))
	for i, pattern := range config.XSSPatterns {
		iv.xssPatterns[i] = regexp.MustCompile(`(?i)` + pattern)
	}

	// Register custom validators
	for name, validatorFunc := range config.CustomValidators {
		iv.validator.RegisterValidation(name, func(fl validator.FieldLevel) bool {
			return validatorFunc(fl.Field().Interface())
		})
	}

	return iv
}

// DefaultValidationConfig returns default validation configuration
func DefaultValidationConfig(logger *logrus.Logger) *ValidationConfig {
	return &ValidationConfig{
		SQLInjectionPatterns: []string{
			`(\b(SELECT|INSERT|UPDATE|DELETE|DROP|CREATE|ALTER|EXEC|UNION|SCRIPT)\b)`,
			`(\b(OR|AND)\s+\d+\s*=\s*\d+)`,
			`(\b(OR|AND)\s+['"]?\w+['"]?\s*=\s*['"]?\w+['"]?)`,
			`(--|#|\/\*|\*\/)`,
			`(\b(INFORMATION_SCHEMA|SYSOBJECTS|SYSCOLUMNS)\b)`,
			`(\bxp_cmdshell\b)`,
			`(\bsp_executesql\b)`,
		},
		XSSPatterns: []string{
			`<\s*script[^>]*>.*?<\s*/\s*script\s*>`,
			`<\s*iframe[^>]*>.*?<\s*/\s*iframe\s*>`,
			`<\s*object[^>]*>.*?<\s*/\s*object\s*>`,
			`<\s*embed[^>]*>.*?<\s*/\s*embed\s*>`,
			`<\s*link[^>]*>`,
			`<\s*meta[^>]*>`,
			`javascript:`,
			`vbscript:`,
			`on\w+\s*=`,
			`expression\s*\(`,
		},
		MaxFieldLengths: map[string]int{
			"email":       255,
			"username":    50,
			"password":    255,
			"name":        100,
			"title":       255,
			"description": 5000,
			"comment":     2000,
			"url":         2048,
		},
		AllowedFileTypes: []string{
			".jpg", ".jpeg", ".png", ".gif", ".pdf", ".doc", ".docx",
			".txt", ".csv", ".zip", ".tar", ".gz",
		},
		MaxFileSize:            10 * 1024 * 1024, // 10MB
		EnableHTMLSanitization: true,
		StrictMode:             false,
		Logger:                 logger,
	}
}

// ValidationMiddleware returns a Gin middleware for input validation
func (iv *InputValidator) ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip validation for certain endpoints
		if iv.shouldSkipValidation(c) {
			c.Next()
			return
		}

		// Validate query parameters
		if err := iv.validateQueryParams(c); err != nil {
			iv.handleValidationError(c, err)
			return
		}

		// Validate request body for POST/PUT/PATCH requests
		if c.Request.Method != "GET" && c.Request.Method != "DELETE" {
			if err := iv.validateRequestBody(c); err != nil {
				iv.handleValidationError(c, err)
				return
			}
		}

		c.Next()
	}
}

// shouldSkipValidation checks if validation should be skipped for this request
func (iv *InputValidator) shouldSkipValidation(c *gin.Context) bool {
	// Skip validation for health checks, metrics, etc.
	skipPaths := []string{
		"/health", "/metrics", "/status", "/favicon.ico",
		"/static/", "/css/", "/js/", "/img/", "/assets/",
	}

	path := c.Request.URL.Path
	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}

	return false
}

// validateQueryParams validates query parameters
func (iv *InputValidator) validateQueryParams(c *gin.Context) error {
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			// Check for SQL injection
			if iv.detectSQLInjection(value) {
				iv.logSecurityEvent(c, "sql_injection_attempt", key, value)
				return fmt.Errorf("potential SQL injection detected in parameter: %s", key)
			}

			// Check for XSS
			if iv.detectXSS(value) {
				iv.logSecurityEvent(c, "xss_attempt", key, value)
				return fmt.Errorf("potential XSS detected in parameter: %s", key)
			}

			// Check field length
			if maxLen, exists := iv.config.MaxFieldLengths[key]; exists {
				if len(value) > maxLen {
					return fmt.Errorf("parameter %s exceeds maximum length of %d", key, maxLen)
				}
			}
		}
	}

	return nil
}

// validateRequestBody validates request body
func (iv *InputValidator) validateRequestBody(c *gin.Context) error {
	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %v", err)
	}

	// Restore the request body for further processing
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Skip validation for empty bodies
	if len(body) == 0 {
		return nil
	}

	// Parse JSON body
	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		// If it's not JSON, treat as raw data and validate
		bodyStr := string(body)
		if iv.detectSQLInjection(bodyStr) {
			iv.logSecurityEvent(c, "sql_injection_attempt", "body", bodyStr)
			return fmt.Errorf("potential SQL injection detected in request body")
		}
		if iv.detectXSS(bodyStr) {
			iv.logSecurityEvent(c, "xss_attempt", "body", bodyStr)
			return fmt.Errorf("potential XSS detected in request body")
		}
		return nil
	}

	// Validate JSON fields
	return iv.validateJSONFields(c, jsonData)
}

// validateJSONFields validates JSON fields recursively
func (iv *InputValidator) validateJSONFields(c *gin.Context, data map[string]interface{}) error {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			// Check for SQL injection
			if iv.detectSQLInjection(v) {
				iv.logSecurityEvent(c, "sql_injection_attempt", key, v)
				return fmt.Errorf("potential SQL injection detected in field: %s", key)
			}

			// Check for XSS
			if iv.detectXSS(v) {
				iv.logSecurityEvent(c, "xss_attempt", key, v)
				return fmt.Errorf("potential XSS detected in field: %s", key)
			}

			// Check field length
			if maxLen, exists := iv.config.MaxFieldLengths[key]; exists {
				if len(v) > maxLen {
					return fmt.Errorf("field %s exceeds maximum length of %d", key, maxLen)
				}
			}

			// Sanitize HTML if enabled
			if iv.config.EnableHTMLSanitization {
				sanitized := iv.sanitizer.Sanitize(v)
				if sanitized != v {
					// Update the value in the data map
					data[key] = sanitized
					iv.logSecurityEvent(c, "html_sanitized", key, v)
				}
			}

		case map[string]interface{}:
			// Recursively validate nested objects
			if err := iv.validateJSONFields(c, v); err != nil {
				return err
			}
		}
	}

	return nil
}

// detectSQLInjection checks for SQL injection patterns
func (iv *InputValidator) detectSQLInjection(input string) bool {
	for _, pattern := range iv.sqlPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// detectXSS checks for XSS patterns
func (iv *InputValidator) detectXSS(input string) bool {
	for _, pattern := range iv.xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// logSecurityEvent logs security-related events
func (iv *InputValidator) logSecurityEvent(c *gin.Context, eventType, field, value string) {
	if iv.logger != nil {
		iv.logger.WithFields(logrus.Fields{
			"event_type": eventType,
			"field":      field,
			"value":      value,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"endpoint":   c.Request.URL.Path,
			"method":     c.Request.Method,
		}).Warn("Security event detected")
	}
}

// handleValidationError handles validation errors
func (iv *InputValidator) handleValidationError(c *gin.Context, err error) {
	response := ValidationResponse{
		Error:   "Validation failed",
		Message: err.Error(),
		Errors:  []ValidationError{},
	}

	// If it's a validator error, extract field details
	if validatorErr, ok := err.(validator.ValidationErrors); ok {
		for _, fieldErr := range validatorErr {
			response.Errors = append(response.Errors, ValidationError{
				Field:   fieldErr.Field(),
				Value:   fieldErr.Value(),
				Tag:     fieldErr.Tag(),
				Message: getValidationMessage(fieldErr),
			})
		}
	}

	c.JSON(http.StatusBadRequest, response)
	c.Abort()
}

// getValidationMessage returns a user-friendly validation error message
func getValidationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fieldErr.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fieldErr.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fieldErr.Field(), fieldErr.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fieldErr.Field(), fieldErr.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", fieldErr.Field(), fieldErr.Param())
	case "alphanum":
		return fmt.Sprintf("%s must contain only alphanumeric characters", fieldErr.Field())
	case "numeric":
		return fmt.Sprintf("%s must be numeric", fieldErr.Field())
	case "url":
		return fmt.Sprintf("%s must be a valid URL", fieldErr.Field())
	default:
		return fmt.Sprintf("%s is invalid", fieldErr.Field())
	}
}

// SanitizeInput sanitizes input string
func (iv *InputValidator) SanitizeInput(input string) string {
	if iv.config.EnableHTMLSanitization {
		return iv.sanitizer.Sanitize(input)
	}
	return input
}

// ValidateStruct validates a struct using validator tags
func (iv *InputValidator) ValidateStruct(s interface{}) error {
	return iv.validator.Struct(s)
}

// IsValidEmail checks if email is valid
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// IsValidUsername checks if username is valid
func IsValidUsername(username string) bool {
	// Username should be 3-50 characters, alphanumeric with underscores and hyphens
	if len(username) < 3 || len(username) > 50 {
		return false
	}

	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return usernameRegex.MatchString(username)
}

// IsStrongPassword checks if password meets strength requirements
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasDigit && hasSpecial
}

// ValidateFileUpload validates file upload
func (iv *InputValidator) ValidateFileUpload(c *gin.Context) error {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return fmt.Errorf("failed to get file: %v", err)
	}
	defer file.Close()

	// Check file size
	if header.Size > iv.config.MaxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", iv.config.MaxFileSize)
	}

	// Check file type
	filename := header.Filename
	allowed := false
	for _, allowedType := range iv.config.AllowedFileTypes {
		if strings.HasSuffix(strings.ToLower(filename), allowedType) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file type not allowed: %s", filename)
	}

	// Read file content to validate
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file content: %v", err)
	}

	// Check for malicious content
	contentStr := string(content)
	if iv.detectSQLInjection(contentStr) {
		return fmt.Errorf("malicious content detected in file")
	}

	if iv.detectXSS(contentStr) {
		return fmt.Errorf("malicious content detected in file")
	}

	return nil
}

// FileUploadMiddleware returns a middleware for file upload validation
func (iv *InputValidator) FileUploadMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only validate file uploads
		if c.Request.Method != "POST" || c.Request.Header.Get("Content-Type") == "" {
			c.Next()
			return
		}

		contentType := c.Request.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			c.Next()
			return
		}

		// Validate file upload
		if err := iv.ValidateFileUpload(c); err != nil {
			iv.handleValidationError(c, err)
			return
		}

		c.Next()
	}
}

// CleanInput removes potentially harmful characters from input
func CleanInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters except newlines and tabs
	var cleaned strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) && r != '\n' && r != '\t' && r != '\r' {
			continue
		}
		cleaned.WriteRune(r)
	}

	return cleaned.String()
}

// NormalizeInput normalizes input for consistent processing
func NormalizeInput(input string) string {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Normalize line endings
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")

	// Remove excessive whitespace
	input = regexp.MustCompile(`\s+`).ReplaceAllString(input, " ")

	return input
}
