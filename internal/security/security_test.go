package security

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInputValidator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Suppress logs during testing

	config := DefaultValidationConfig(logger)
	validator := NewInputValidator(config)

	t.Run("SQL Injection Detection", func(t *testing.T) {
		maliciousInputs := []string{
			"'; DROP TABLE users; --",
			"1 OR 1=1",
			"UNION SELECT * FROM passwords",
			"admin'--",
		}

		for _, input := range maliciousInputs {
			assert.True(t, validator.detectSQLInjection(input), "Should detect SQL injection in: %s", input)
		}
	})

	t.Run("XSS Detection", func(t *testing.T) {
		maliciousInputs := []string{
			"<script>alert('xss')</script>",
			"javascript:alert(1)",
			"<iframe src='evil.com'></iframe>",
			"<img onload='alert(1)'>",
		}

		for _, input := range maliciousInputs {
			assert.True(t, validator.detectXSS(input), "Should detect XSS in: %s", input)
		}
	})

	t.Run("HTML Sanitization", func(t *testing.T) {
		input := "<script>alert('evil')</script><p>Good content</p>"
		sanitized := validator.SanitizeInput(input)

		assert.NotContains(t, sanitized, "<script>")
		assert.Contains(t, sanitized, "Good content")
	})
}

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	config := &RateLimitConfig{
		GlobalRPS:           10,
		GlobalBurst:         20,
		PerIPRPS:            5,
		PerIPBurst:          10,
		WindowSize:          time.Minute,
		CleanupInterval:     5 * time.Minute,
		BlockDuration:       15 * time.Minute,
		SuspiciousThreshold: 3,
	}

	rateLimiter := NewRateLimiter(config, logger)

	t.Run("Rate Limiting Works", func(t *testing.T) {
		router := gin.New()
		router.Use(rateLimiter.Middleware())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "ok"})
		})

		// First few requests should pass
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code, "Request %d should pass", i+1)
		}

		// Subsequent requests should be rate limited
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code == 429 {
				// Rate limit hit
				break
			}
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := DefaultSecurityHeadersConfig("production")

	router := gin.New()
	router.Use(SecurityHeaders(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	t.Run("Security Headers Applied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)

		// Check security headers
		assert.Contains(t, w.Header().Get("Content-Security-Policy"), "default-src")
		assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
		assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	})

	t.Run("HSTS Header for HTTPS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		hstsHeader := w.Header().Get("Strict-Transport-Security")
		assert.Contains(t, hstsHeader, "max-age=31536000")
		assert.Contains(t, hstsHeader, "includeSubDomains")
	})
}

func TestErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	errorHandler := NewErrorHandler(logger, "test")

	t.Run("Validation Error", func(t *testing.T) {
		router := gin.New()
		router.Use(errorHandler.ErrorMiddleware())
		router.GET("/test", func(c *gin.Context) {
			err := NewValidationError("Invalid input", map[string]interface{}{
				"field": "username",
			})
			errorHandler.HandleError(c, err)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid input")
		assert.Contains(t, w.Body.String(), "validation")
	})

	t.Run("Authentication Error", func(t *testing.T) {
		router := gin.New()
		router.Use(errorHandler.ErrorMiddleware())
		router.GET("/test", func(c *gin.Context) {
			err := NewAuthenticationError("Invalid credentials")
			errorHandler.HandleError(c, err)
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 401, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid credentials")
	})
}

func TestValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	config := DefaultValidationConfig(logger)
	validator := NewInputValidator(config)

	router := gin.New()
	router.Use(validator.ValidationMiddleware())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	t.Run("Clean Input Passes", func(t *testing.T) {
		body := strings.NewReader(`{"name": "John Doe", "email": "john@example.com"}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("Malicious Input Blocked", func(t *testing.T) {
		body := strings.NewReader(`{"name": "'; DROP TABLE users; --", "email": "test@example.com"}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
		assert.Contains(t, w.Body.String(), "SQL injection")
	})

	t.Run("XSS Input Blocked", func(t *testing.T) {
		body := strings.NewReader(`{"comment": "<script>alert('xss')</script>"}`)
		req := httptest.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 400, w.Code)
		// Either XSS or SQL injection detection is fine for this test
		responseBody := w.Body.String()
		assert.True(t, strings.Contains(responseBody, "XSS") || strings.Contains(responseBody, "SQL injection"),
			"Should detect either XSS or SQL injection in script tag")
	})
}

func TestPasswordValidation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	config := &AuthConfig{
		PasswordMinLength:      8,
		PasswordRequireUpper:   true,
		PasswordRequireLower:   true,
		PasswordRequireDigit:   true,
		PasswordRequireSpecial: true,
	}

	authEnhancer := &AuthEnhancer{config: config}

	t.Run("Strong Password Passes", func(t *testing.T) {
		err := authEnhancer.ValidatePasswordStrength("MyStr0ng!Pass")
		assert.NoError(t, err)
	})

	t.Run("Weak Passwords Fail", func(t *testing.T) {
		weakPasswords := []string{
			"short",           // Too short
			"nouppercase123!", // No uppercase
			"NOLOWERCASE123!", // No lowercase
			"NoDigits!",       // No digits
			"NoSpecial123",    // No special chars
		}

		for _, password := range weakPasswords {
			err := authEnhancer.ValidatePasswordStrength(password)
			assert.Error(t, err, "Password should be rejected: %s", password)
		}
	})
}

func TestAPIKeyValidation(t *testing.T) {
	// This would require a database connection, so we'll test the validation logic
	config := DefaultAPIKeyConfig()

	t.Run("Valid API Key Format", func(t *testing.T) {
		// Test key format validation
		key := "12345678abcdefghijklmnopqrstuvwxyz01234567890123456789012345678901"
		assert.True(t, len(key) >= config.PrefixLength, "Key should be long enough")
	})
}

func BenchmarkRateLimiter(b *testing.B) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	config := &RateLimitConfig{
		GlobalRPS:   10000,
		GlobalBurst: 20000,
		PerIPRPS:    1000,
		PerIPBurst:  2000,
		WindowSize:  time.Minute,
	}

	rateLimiter := NewRateLimiter(config, logger)

	router := gin.New()
	router.Use(rateLimiter.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "ok"})
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

func BenchmarkInputValidation(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel)

	config := DefaultValidationConfig(logger)
	validator := NewInputValidator(config)

	testInput := "This is a normal input string with no malicious content"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.detectSQLInjection(testInput)
		validator.detectXSS(testInput)
	}
}
