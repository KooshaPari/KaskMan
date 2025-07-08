package middleware

import (
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// ValidationConfig holds validation configuration
type ValidationConfig struct {
	Enabled             bool              `mapstructure:"enabled" json:"enabled"`
	MaxRequestSize      int64             `mapstructure:"max_request_size" json:"max_request_size"`
	AllowedContentTypes []string          `mapstructure:"allowed_content_types" json:"allowed_content_types"`
	SanitizeHTML        bool              `mapstructure:"sanitize_html" json:"sanitize_html"`
	ValidateSQL         bool              `mapstructure:"validate_sql" json:"validate_sql"`
	ValidateXSS         bool              `mapstructure:"validate_xss" json:"validate_xss"`
	RequireContentType  bool              `mapstructure:"require_content_type" json:"require_content_type"`
	CustomValidators    map[string]string `mapstructure:"custom_validators" json:"custom_validators"`
}

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
}

// ValidationResponse represents the validation error response
type ValidationResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// InputValidator provides comprehensive input validation
type InputValidator struct {
	validator *validator.Validate
	config    ValidationConfig
	logger    *logrus.Logger

	// SQL injection patterns
	sqlPatterns []*regexp.Regexp

	// XSS patterns
	xssPatterns []*regexp.Regexp
}

// NewInputValidator creates a new input validator
func NewInputValidator(config ValidationConfig, logger *logrus.Logger) *InputValidator {
	v := validator.New()

	iv := &InputValidator{
		validator: v,
		config:    config,
		logger:    logger,
	}

	// Initialize security patterns
	iv.initializeSecurityPatterns()

	// Register custom validators
	iv.registerCustomValidators()

	return iv
}

// initializeSecurityPatterns compiles security validation patterns
func (iv *InputValidator) initializeSecurityPatterns() {
	// SQL injection patterns
	sqlPatternStrings := []string{
		`(?i)(union\s+select)`,
		`(?i)(drop\s+table)`,
		`(?i)(delete\s+from)`,
		`(?i)(insert\s+into)`,
		`(?i)(update\s+set)`,
		`(?i)(exec\s*\()`,
		`(?i)(execute\s*\()`,
		`(?i)(sp_executesql)`,
		`(?i)(xp_cmdshell)`,
		`(?i)(script\s*>)`,
		`(?i)(javascript:)`,
		`(?i)(vbscript:)`,
		`(?i)(onload\s*=)`,
		`(?i)(onerror\s*=)`,
		`(?i)(onclick\s*=)`,
		`--`,
		`;--`,
		`/*`,
		`*/`,
		`@@`,
		`char\(`,
		`nchar\(`,
		`varchar\(`,
		`nvarchar\(`,
		`alter\s+`,
		`create\s+`,
		`drop\s+`,
		`truncate\s+`,
	}

	for _, pattern := range sqlPatternStrings {
		if compiled, err := regexp.Compile(pattern); err == nil {
			iv.sqlPatterns = append(iv.sqlPatterns, compiled)
		} else {
			iv.logger.WithError(err).WithField("pattern", pattern).Warn("Failed to compile SQL pattern")
		}
	}

	// XSS patterns
	xssPatternStrings := []string{
		`(?i)<script[^>]*>.*?</script>`,
		`(?i)<iframe[^>]*>.*?</iframe>`,
		`(?i)<object[^>]*>.*?</object>`,
		`(?i)<embed[^>]*>.*?</embed>`,
		`(?i)<link[^>]*>`,
		`(?i)<meta[^>]*>`,
		`(?i)javascript:`,
		`(?i)vbscript:`,
		`(?i)onload\s*=`,
		`(?i)onerror\s*=`,
		`(?i)onclick\s*=`,
		`(?i)onmouseover\s*=`,
		`(?i)onfocus\s*=`,
		`(?i)onblur\s*=`,
		`(?i)onchange\s*=`,
		`(?i)onsubmit\s*=`,
		`(?i)expression\s*\(`,
		`(?i)url\s*\(`,
		`(?i)@import`,
		`(?i)<!\[CDATA\[`,
		`(?i)]]>`,
	}

	for _, pattern := range xssPatternStrings {
		if compiled, err := regexp.Compile(pattern); err == nil {
			iv.xssPatterns = append(iv.xssPatterns, compiled)
		} else {
			iv.logger.WithError(err).WithField("pattern", pattern).Warn("Failed to compile XSS pattern")
		}
	}
}

// registerCustomValidators registers custom validation rules
func (iv *InputValidator) registerCustomValidators() {
	// Register password strength validator
	iv.validator.RegisterValidation("password", iv.validatePassword)

	// Register username validator
	iv.validator.RegisterValidation("username", iv.validateUsername)

	// Register safe string validator (no SQL/XSS)
	iv.validator.RegisterValidation("safestring", iv.validateSafeString)

	// Register project type validator
	iv.validator.RegisterValidation("projecttype", iv.validateProjectType)

	// Register priority validator
	iv.validator.RegisterValidation("priority", iv.validatePriority)

	// Register status validator
	iv.validator.RegisterValidation("status", iv.validateStatus)

	// Register UUID validator
	iv.validator.RegisterValidation("uuid", iv.validateUUID)

	// Register email domain validator
	iv.validator.RegisterValidation("allowed_domain", iv.validateAllowedDomain)
}

// ValidationMiddleware creates input validation middleware
func ValidationMiddleware(config ValidationConfig, logger *logrus.Logger) gin.HandlerFunc {
	validator := NewInputValidator(config, logger)

	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		// Validate request size
		if err := validator.validateRequestSize(c); err != nil {
			validator.respondWithError(c, http.StatusRequestEntityTooLarge, "Request too large", err)
			return
		}

		// Validate content type for POST/PUT/PATCH requests
		if err := validator.validateContentType(c); err != nil {
			validator.respondWithError(c, http.StatusUnsupportedMediaType, "Invalid content type", err)
			return
		}

		// Validate query parameters
		if err := validator.validateQueryParams(c); err != nil {
			validator.respondWithError(c, http.StatusBadRequest, "Invalid query parameters", err)
			return
		}

		// Validate headers
		if err := validator.validateHeaders(c); err != nil {
			validator.respondWithError(c, http.StatusBadRequest, "Invalid headers", err)
			return
		}

		c.Next()
	}
}

// validateRequestSize validates the request size
func (iv *InputValidator) validateRequestSize(c *gin.Context) error {
	if iv.config.MaxRequestSize > 0 && c.Request.ContentLength > iv.config.MaxRequestSize {
		return fmt.Errorf("request size %d exceeds maximum allowed size %d",
			c.Request.ContentLength, iv.config.MaxRequestSize)
	}
	return nil
}

// validateContentType validates the content type for requests with body
func (iv *InputValidator) validateContentType(c *gin.Context) error {
	method := c.Request.Method
	if method != "POST" && method != "PUT" && method != "PATCH" {
		return nil
	}

	contentType := c.GetHeader("Content-Type")
	if iv.config.RequireContentType && contentType == "" {
		return fmt.Errorf("content-Type header is required for %s requests", method)
	}

	if contentType != "" && len(iv.config.AllowedContentTypes) > 0 {
		allowed := false
		for _, allowedType := range iv.config.AllowedContentTypes {
			if strings.Contains(contentType, allowedType) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("content-Type %s is not allowed", contentType)
		}
	}

	return nil
}

// validateQueryParams validates query parameters for security issues
func (iv *InputValidator) validateQueryParams(c *gin.Context) error {
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			if err := iv.validateStringForSecurity(key, value); err != nil {
				return fmt.Errorf("query parameter %s: %w", key, err)
			}
		}
	}
	return nil
}

// validateHeaders validates request headers
func (iv *InputValidator) validateHeaders(c *gin.Context) error {
	// Validate User-Agent
	userAgent := c.GetHeader("User-Agent")
	if userAgent != "" {
		if err := iv.validateStringForSecurity("User-Agent", userAgent); err != nil {
			return fmt.Errorf("user-Agent header: %w", err)
		}
	}

	// Validate custom headers
	for key, values := range c.Request.Header {
		if strings.HasPrefix(key, "X-") {
			for _, value := range values {
				if err := iv.validateStringForSecurity(key, value); err != nil {
					return fmt.Errorf("header %s: %w", key, err)
				}
			}
		}
	}

	return nil
}

// validateStringForSecurity validates a string for SQL injection and XSS
func (iv *InputValidator) validateStringForSecurity(field, value string) error {
	if iv.config.ValidateSQL {
		if err := iv.validateSQL(value); err != nil {
			iv.logger.WithFields(logrus.Fields{
				"field": field,
				"value": value,
				"error": err,
			}).Warn("SQL injection attempt detected")
			return fmt.Errorf("potential SQL injection detected")
		}
	}

	if iv.config.ValidateXSS {
		if err := iv.validateXSS(value); err != nil {
			iv.logger.WithFields(logrus.Fields{
				"field": field,
				"value": value,
				"error": err,
			}).Warn("XSS attempt detected")
			return fmt.Errorf("potential XSS detected")
		}
	}

	return nil
}

// validateSQL checks for SQL injection patterns
func (iv *InputValidator) validateSQL(value string) error {
	for _, pattern := range iv.sqlPatterns {
		if pattern.MatchString(value) {
			return fmt.Errorf("SQL injection pattern detected: %s", pattern.String())
		}
	}
	return nil
}

// validateXSS checks for XSS patterns
func (iv *InputValidator) validateXSS(value string) error {
	for _, pattern := range iv.xssPatterns {
		if pattern.MatchString(value) {
			return fmt.Errorf("XSS pattern detected: %s", pattern.String())
		}
	}
	return nil
}

// SanitizeString sanitizes a string for safe storage and display
func (iv *InputValidator) SanitizeString(value string) string {
	if iv.config.SanitizeHTML {
		value = html.EscapeString(value)
	}

	// Trim whitespace
	value = strings.TrimSpace(value)

	// Remove control characters except newlines and tabs
	result := make([]rune, 0, len(value))
	for _, r := range value {
		if unicode.IsPrint(r) || r == '\n' || r == '\t' {
			result = append(result, r)
		}
	}

	return string(result)
}

// ValidateStruct validates a struct using validator tags
func (iv *InputValidator) ValidateStruct(s interface{}) []ValidationError {
	var errors []ValidationError

	err := iv.validator.Struct(s)
	if err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, fieldError := range validationErrors {
				errors = append(errors, ValidationError{
					Field:   fieldError.Field(),
					Tag:     fieldError.Tag(),
					Value:   fieldError.Value(),
					Message: iv.getValidationMessage(fieldError),
				})
			}
		}
	}

	return errors
}

// getValidationMessage returns a user-friendly validation message
func (iv *InputValidator) getValidationMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fieldError.Field())
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fieldError.Field())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", fieldError.Field(), fieldError.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", fieldError.Field(), fieldError.Param())
	case "password":
		return fmt.Sprintf("%s must contain at least 8 characters with uppercase, lowercase, number and special character", fieldError.Field())
	case "username":
		return fmt.Sprintf("%s must contain only letters, numbers, and underscores", fieldError.Field())
	case "safestring":
		return fmt.Sprintf("%s contains potentially unsafe content", fieldError.Field())
	case "uuid":
		return fmt.Sprintf("%s must be a valid UUID", fieldError.Field())
	default:
		return fmt.Sprintf("%s is invalid", fieldError.Field())
	}
}

// respondWithError sends a validation error response
func (iv *InputValidator) respondWithError(c *gin.Context, statusCode int, message string, err error) {
	iv.logger.WithFields(logrus.Fields{
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"ip":         c.ClientIP(),
		"user_agent": c.GetHeader("User-Agent"),
		"error":      err.Error(),
	}).Warn("Validation failed")

	response := ValidationResponse{
		Error:   "validation_failed",
		Message: message,
		Details: map[string]string{
			"error": err.Error(),
		},
	}

	c.JSON(statusCode, response)
	c.Abort()
}

// Custom validation functions

func (iv *InputValidator) validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

func (iv *InputValidator) validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()

	if len(username) < 3 || len(username) > 30 {
		return false
	}

	for _, char := range username {
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && char != '_' {
			return false
		}
	}

	return true
}

func (iv *InputValidator) validateSafeString(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	if iv.validateSQL(value) != nil {
		return false
	}

	if iv.validateXSS(value) != nil {
		return false
	}

	return true
}

func (iv *InputValidator) validateProjectType(fl validator.FieldLevel) bool {
	projectType := fl.Field().String()
	validTypes := []string{"research", "development", "analysis", "innovation", "maintenance"}

	for _, validType := range validTypes {
		if projectType == validType {
			return true
		}
	}

	return false
}

func (iv *InputValidator) validatePriority(fl validator.FieldLevel) bool {
	priority := fl.Field().String()
	validPriorities := []string{"low", "medium", "high", "critical"}

	for _, validPriority := range validPriorities {
		if priority == validPriority {
			return true
		}
	}

	return false
}

func (iv *InputValidator) validateStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"pending", "in_progress", "completed", "cancelled", "failed", "active", "inactive"}

	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}

	return false
}

func (iv *InputValidator) validateUUID(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	// Simple UUID validation pattern
	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return uuidPattern.MatchString(value)
}

func (iv *InputValidator) validateAllowedDomain(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	allowedDomains := []string{"company.com", "contractor.com"}

	for _, domain := range allowedDomains {
		if strings.HasSuffix(email, "@"+domain) {
			return true
		}
	}

	return false
}

// GetDefaultValidationConfig returns default validation configuration
func GetDefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		Enabled:        true,
		MaxRequestSize: 10 * 1024 * 1024, // 10MB
		AllowedContentTypes: []string{
			"application/json",
			"application/x-www-form-urlencoded",
			"multipart/form-data",
			"text/plain",
		},
		SanitizeHTML:       true,
		ValidateSQL:        true,
		ValidateXSS:        true,
		RequireContentType: true,
		CustomValidators: map[string]string{
			"password":   "Strong password required",
			"username":   "Valid username required",
			"safestring": "Safe string required",
		},
	}
}
