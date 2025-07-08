package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// APIKeyManager manages API keys for authentication
type APIKeyManager struct {
	db       *gorm.DB
	redis    *redis.Client
	logger   *logrus.Logger
	config   *APIKeyConfig
	useRedis bool

	// In-memory cache for API keys
	cache map[string]*APIKey
}

// APIKeyConfig holds API key configuration
type APIKeyConfig struct {
	// Key generation
	KeyLength    int
	PrefixLength int

	// Expiration
	DefaultTTL time.Duration
	MaxTTL     time.Duration

	// Rate limiting
	DefaultRateLimit int
	MaxRateLimit     int

	// Security
	HashKeys         bool
	RequireUserAgent bool
	RequireReferer   bool
	AllowedIPs       []string

	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int
}

// APIKey represents an API key
type APIKey struct {
	ID          string     `json:"id" gorm:"primaryKey"`
	UserID      string     `json:"user_id" gorm:"index"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"key_hash" gorm:"index"`
	Prefix      string     `json:"prefix" gorm:"index"`
	Permissions []string   `json:"permissions" gorm:"serializer:json"`
	RateLimit   int        `json:"rate_limit"`
	AllowedIPs  []string   `json:"allowed_ips" gorm:"serializer:json"`
	Referers    []string   `json:"referers" gorm:"serializer:json"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	IsActive    bool       `json:"is_active"`
	UsageCount  int64      `json:"usage_count"`
}

// APIKeyUsage tracks API key usage
type APIKeyUsage struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	APIKeyID  string    `json:"api_key_id" gorm:"index"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Endpoint  string    `json:"endpoint"`
	Method    string    `json:"method"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name        string     `json:"name" binding:"required"`
	Permissions []string   `json:"permissions"`
	RateLimit   int        `json:"rate_limit"`
	AllowedIPs  []string   `json:"allowed_ips"`
	Referers    []string   `json:"referers"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// CreateAPIKeyResponse represents the response when creating an API key
type CreateAPIKeyResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Key     string `json:"key"`
	Prefix  string `json:"prefix"`
	Message string `json:"message"`
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(db *gorm.DB, config *APIKeyConfig, logger *logrus.Logger) *APIKeyManager {
	akm := &APIKeyManager{
		db:     db,
		logger: logger,
		config: config,
		cache:  make(map[string]*APIKey),
	}

	// Initialize Redis if configured
	if config.RedisAddr != "" {
		akm.redis = redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})

		// Test Redis connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := akm.redis.Ping(ctx).Err(); err != nil {
			logger.WithError(err).Warn("Failed to connect to Redis, using in-memory cache")
			akm.useRedis = false
		} else {
			akm.useRedis = true
			logger.Info("Connected to Redis for API key caching")
		}
	}

	// Auto-migrate database tables
	if err := db.AutoMigrate(&APIKey{}, &APIKeyUsage{}); err != nil {
		logger.WithError(err).Error("Failed to migrate API key tables")
	}

	return akm
}

// DefaultAPIKeyConfig returns default API key configuration
func DefaultAPIKeyConfig() *APIKeyConfig {
	return &APIKeyConfig{
		KeyLength:        32,
		PrefixLength:     8,
		DefaultTTL:       365 * 24 * time.Hour,     // 1 year
		MaxTTL:           5 * 365 * 24 * time.Hour, // 5 years
		DefaultRateLimit: 1000,
		MaxRateLimit:     10000,
		HashKeys:         true,
		RequireUserAgent: false,
		RequireReferer:   false,
		AllowedIPs:       []string{},
	}
}

// CreateAPIKey creates a new API key
func (akm *APIKeyManager) CreateAPIKey(userID string, req *CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	// Validate request
	if err := akm.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Generate key
	key, err := akm.generateKey()
	if err != nil {
		return nil, err
	}

	// Create prefix
	prefix := key[:akm.config.PrefixLength]

	// Hash key if configured
	keyHash := key
	if akm.config.HashKeys {
		keyHash = akm.hashKey(key)
	}

	// Set default rate limit
	rateLimit := req.RateLimit
	if rateLimit == 0 {
		rateLimit = akm.config.DefaultRateLimit
	}

	// Create API key record
	apiKey := &APIKey{
		ID:          akm.generateID(),
		UserID:      userID,
		Name:        req.Name,
		KeyHash:     keyHash,
		Prefix:      prefix,
		Permissions: req.Permissions,
		RateLimit:   rateLimit,
		AllowedIPs:  req.AllowedIPs,
		Referers:    req.Referers,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
		IsActive:    true,
		UsageCount:  0,
	}

	// Save to database
	if err := akm.db.Create(apiKey).Error; err != nil {
		return nil, err
	}

	// Cache the key
	akm.cacheAPIKey(apiKey)

	akm.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"api_key_id": apiKey.ID,
		"name":       apiKey.Name,
	}).Info("API key created")

	return &CreateAPIKeyResponse{
		ID:      apiKey.ID,
		Name:    apiKey.Name,
		Key:     key,
		Prefix:  prefix,
		Message: "API key created successfully. Store this key securely as it won't be shown again.",
	}, nil
}

// ValidateAPIKey validates an API key and returns the associated key info
func (akm *APIKeyManager) ValidateAPIKey(key string) (*APIKey, error) {
	if key == "" {
		return nil, NewAuthenticationError("API key is required")
	}

	// Extract prefix
	if len(key) < akm.config.PrefixLength {
		return nil, NewAuthenticationError("Invalid API key format")
	}

	prefix := key[:akm.config.PrefixLength]

	// Get API key from cache or database
	apiKey, err := akm.getAPIKeyByPrefix(prefix)
	if err != nil {
		return nil, err
	}

	// Validate key
	if !akm.validateKey(key, apiKey.KeyHash) {
		return nil, NewAuthenticationError("Invalid API key")
	}

	// Check if key is active
	if !apiKey.IsActive {
		return nil, NewAuthenticationError("API key is inactive")
	}

	// Check if key is expired
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, NewAuthenticationError("API key has expired")
	}

	return apiKey, nil
}

// APIKeyMiddleware returns a Gin middleware for API key authentication
func (akm *APIKeyManager) APIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get API key from header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Try Authorization header
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}
		}

		if apiKey == "" {
			c.JSON(401, gin.H{
				"error": "API key is required",
			})
			c.Abort()
			return
		}

		// Validate API key
		key, err := akm.ValidateAPIKey(apiKey)
		if err != nil {
			if appErr, ok := err.(*AppError); ok {
				c.JSON(appErr.HTTPStatus, gin.H{
					"error": appErr.Message,
				})
			} else {
				c.JSON(401, gin.H{
					"error": "Invalid API key",
				})
			}
			c.Abort()
			return
		}

		// Additional validation
		if err := akm.validateAPIKeyConstraints(c, key); err != nil {
			if appErr, ok := err.(*AppError); ok {
				c.JSON(appErr.HTTPStatus, gin.H{
					"error": appErr.Message,
				})
			} else {
				c.JSON(403, gin.H{
					"error": "API key validation failed",
				})
			}
			c.Abort()
			return
		}

		// Set key info in context
		c.Set("api_key", key)
		c.Set("api_key_id", key.ID)
		c.Set("user_id", key.UserID)

		// Record usage
		go akm.recordUsage(key.ID, c)

		c.Next()
	}
}

// validateAPIKeyConstraints validates API key constraints
func (akm *APIKeyManager) validateAPIKeyConstraints(c *gin.Context, key *APIKey) error {
	// Check IP restrictions
	if len(key.AllowedIPs) > 0 {
		clientIP := c.ClientIP()
		allowed := false
		for _, ip := range key.AllowedIPs {
			if ip == clientIP {
				allowed = true
				break
			}
		}
		if !allowed {
			return NewAuthorizationError("API key not allowed from this IP address")
		}
	}

	// Check referer restrictions
	if len(key.Referers) > 0 {
		referer := c.GetHeader("Referer")
		if referer == "" && akm.config.RequireReferer {
			return NewAuthorizationError("Referer header is required")
		}

		if referer != "" {
			allowed := false
			for _, allowedReferer := range key.Referers {
				if strings.Contains(referer, allowedReferer) {
					allowed = true
					break
				}
			}
			if !allowed {
				return NewAuthorizationError("API key not allowed from this referer")
			}
		}
	}

	// Check user agent (if required)
	if akm.config.RequireUserAgent {
		userAgent := c.GetHeader("User-Agent")
		if userAgent == "" {
			return NewAuthorizationError("User-Agent header is required")
		}
	}

	return nil
}

// GetAPIKeys returns all API keys for a user
func (akm *APIKeyManager) GetAPIKeys(userID string) ([]*APIKey, error) {
	var keys []*APIKey

	if err := akm.db.Where("user_id = ?", userID).Find(&keys).Error; err != nil {
		return nil, err
	}

	// Don't return the actual key hash
	for _, key := range keys {
		key.KeyHash = ""
	}

	return keys, nil
}

// RevokeAPIKey revokes an API key
func (akm *APIKeyManager) RevokeAPIKey(keyID, userID string) error {
	var key APIKey
	if err := akm.db.Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error; err != nil {
		return err
	}

	key.IsActive = false
	key.UpdatedAt = time.Now()

	if err := akm.db.Save(&key).Error; err != nil {
		return err
	}

	// Remove from cache
	akm.removeCachedAPIKey(key.Prefix)

	akm.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"api_key_id": keyID,
	}).Info("API key revoked")

	return nil
}

// UpdateAPIKey updates an API key
func (akm *APIKeyManager) UpdateAPIKey(keyID, userID string, updates map[string]interface{}) error {
	var key APIKey
	if err := akm.db.Where("id = ? AND user_id = ?", keyID, userID).First(&key).Error; err != nil {
		return err
	}

	updates["updated_at"] = time.Now()

	if err := akm.db.Model(&key).Updates(updates).Error; err != nil {
		return err
	}

	// Update cache
	akm.removeCachedAPIKey(key.Prefix)

	return nil
}

// generateKey generates a secure API key
func (akm *APIKeyManager) generateKey() (string, error) {
	bytes := make([]byte, akm.config.KeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateID generates a unique ID
func (akm *APIKeyManager) generateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("key_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// hashKey hashes an API key
func (akm *APIKeyManager) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// validateKey validates a key against its hash
func (akm *APIKeyManager) validateKey(key, hash string) bool {
	if akm.config.HashKeys {
		return akm.hashKey(key) == hash
	}
	return key == hash
}

// getAPIKeyByPrefix gets an API key by prefix
func (akm *APIKeyManager) getAPIKeyByPrefix(prefix string) (*APIKey, error) {
	// Check cache first
	if cached, exists := akm.cache[prefix]; exists {
		return cached, nil
	}

	// Check Redis cache
	if akm.useRedis {
		if cached, err := akm.getCachedAPIKeyRedis(prefix); err == nil && cached != nil {
			return cached, nil
		}
	}

	// Get from database
	var key APIKey
	if err := akm.db.Where("prefix = ? AND is_active = true", prefix).First(&key).Error; err != nil {
		return nil, NewAuthenticationError("API key not found")
	}

	// Cache the key
	akm.cacheAPIKey(&key)

	return &key, nil
}

// cacheAPIKey caches an API key
func (akm *APIKeyManager) cacheAPIKey(key *APIKey) {
	// In-memory cache
	akm.cache[key.Prefix] = key

	// Redis cache
	if akm.useRedis {
		akm.setCachedAPIKeyRedis(key)
	}
}

// setCachedAPIKeyRedis caches an API key in Redis
func (akm *APIKeyManager) setCachedAPIKeyRedis(key *APIKey) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("api_key:%s", key.Prefix)

	// Store as hash
	akm.redis.HSet(ctx, cacheKey, map[string]interface{}{
		"id":          key.ID,
		"user_id":     key.UserID,
		"name":        key.Name,
		"key_hash":    key.KeyHash,
		"prefix":      key.Prefix,
		"rate_limit":  key.RateLimit,
		"is_active":   key.IsActive,
		"usage_count": key.UsageCount,
	})

	// Set expiration
	akm.redis.Expire(ctx, cacheKey, 1*time.Hour)
}

// getCachedAPIKeyRedis gets a cached API key from Redis
func (akm *APIKeyManager) getCachedAPIKeyRedis(prefix string) (*APIKey, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("api_key:%s", prefix)

	result, err := akm.redis.HGetAll(ctx, cacheKey).Result()
	if err != nil || len(result) == 0 {
		return nil, err
	}

	key := &APIKey{
		ID:       result["id"],
		UserID:   result["user_id"],
		Name:     result["name"],
		KeyHash:  result["key_hash"],
		Prefix:   result["prefix"],
		IsActive: result["is_active"] == "true",
	}

	// Parse numeric fields
	if rateLimit, exists := result["rate_limit"]; exists {
		fmt.Sscanf(rateLimit, "%d", &key.RateLimit)
	}

	if usageCount, exists := result["usage_count"]; exists {
		fmt.Sscanf(usageCount, "%d", &key.UsageCount)
	}

	return key, nil
}

// removeCachedAPIKey removes an API key from cache
func (akm *APIKeyManager) removeCachedAPIKey(prefix string) {
	// Remove from in-memory cache
	delete(akm.cache, prefix)

	// Remove from Redis cache
	if akm.useRedis {
		ctx := context.Background()
		cacheKey := fmt.Sprintf("api_key:%s", prefix)
		akm.redis.Del(ctx, cacheKey)
	}
}

// recordUsage records API key usage
func (akm *APIKeyManager) recordUsage(keyID string, c *gin.Context) {
	usage := &APIKeyUsage{
		ID:        akm.generateID(),
		APIKeyID:  keyID,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Endpoint:  c.Request.URL.Path,
		Method:    c.Request.Method,
		Timestamp: time.Now(),
		Success:   true,
	}

	// Save to database (in background)
	if err := akm.db.Create(usage).Error; err != nil {
		akm.logger.WithError(err).Error("Failed to record API key usage")
	}

	// Update usage count
	akm.db.Model(&APIKey{}).Where("id = ?", keyID).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + 1"),
		"last_used_at": time.Now(),
	})
}

// validateCreateRequest validates a create API key request
func (akm *APIKeyManager) validateCreateRequest(req *CreateAPIKeyRequest) error {
	if req.Name == "" {
		return NewValidationError("Name is required", nil)
	}

	if req.RateLimit > akm.config.MaxRateLimit {
		return NewValidationError("Rate limit exceeds maximum", map[string]interface{}{
			"max_rate_limit": akm.config.MaxRateLimit,
		})
	}

	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now()) {
		return NewValidationError("Expiration date must be in the future", nil)
	}

	if req.ExpiresAt != nil && req.ExpiresAt.After(time.Now().Add(akm.config.MaxTTL)) {
		return NewValidationError("Expiration date exceeds maximum TTL", map[string]interface{}{
			"max_ttl": akm.config.MaxTTL,
		})
	}

	return nil
}

// GetStats returns API key manager statistics
func (akm *APIKeyManager) GetStats() map[string]interface{} {
	var totalKeys int64
	var activeKeys int64

	akm.db.Model(&APIKey{}).Count(&totalKeys)
	akm.db.Model(&APIKey{}).Where("is_active = true").Count(&activeKeys)

	return map[string]interface{}{
		"total_keys":  totalKeys,
		"active_keys": activeKeys,
		"cached_keys": len(akm.cache),
		"using_redis": akm.useRedis,
	}
}
