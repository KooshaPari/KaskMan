package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RateLimiter interface for different rate limiting implementations
type RateLimiter interface {
	Allow(key string) (bool, error)
	Reset(key string) error
	GetStats(key string) (*RateLimitStats, error)
}

// RateLimitStats holds statistics for a rate limit key
type RateLimitStats struct {
	Key          string    `json:"key"`
	RequestCount int64     `json:"request_count"`
	WindowStart  time.Time `json:"window_start"`
	WindowEnd    time.Time `json:"window_end"`
	Limit        int64     `json:"limit"`
	Remaining    int64     `json:"remaining"`
	ResetTime    time.Time `json:"reset_time"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool          `mapstructure:"enabled" json:"enabled"`
	DefaultLimit      int64         `mapstructure:"default_limit" json:"default_limit"`
	DefaultWindow     time.Duration `mapstructure:"default_window" json:"default_window"`
	SkipPaths         []string      `mapstructure:"skip_paths" json:"skip_paths"`
	SkipIPs           []string      `mapstructure:"skip_ips" json:"skip_ips"`
	TrustedProxies    []string      `mapstructure:"trusted_proxies" json:"trusted_proxies"`
	HeaderKeyResolver string        `mapstructure:"header_key_resolver" json:"header_key_resolver"`

	// Per-endpoint limits
	EndpointLimits map[string]EndpointLimit `mapstructure:"endpoint_limits" json:"endpoint_limits"`

	// Per-user limits (authenticated)
	UserLimits map[string]UserLimit `mapstructure:"user_limits" json:"user_limits"`
}

// EndpointLimit defines rate limits for specific endpoints
type EndpointLimit struct {
	Method string        `mapstructure:"method" json:"method"`
	Path   string        `mapstructure:"path" json:"path"`
	Limit  int64         `mapstructure:"limit" json:"limit"`
	Window time.Duration `mapstructure:"window" json:"window"`
}

// UserLimit defines rate limits for specific user roles or IDs
type UserLimit struct {
	Role   string        `mapstructure:"role" json:"role"`
	UserID string        `mapstructure:"user_id" json:"user_id"`
	Limit  int64         `mapstructure:"limit" json:"limit"`
	Window time.Duration `mapstructure:"window" json:"window"`
}

// InMemoryRateLimiter provides an in-memory rate limiting implementation
type InMemoryRateLimiter struct {
	buckets map[string]*TokenBucket
	mutex   sync.RWMutex
	logger  *logrus.Logger
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	Capacity     int64
	Tokens       int64
	RefillRate   int64
	LastRefill   time.Time
	WindowStart  time.Time
	WindowEnd    time.Time
	RequestCount int64
	mutex        sync.Mutex
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter(logger *logrus.Logger) *InMemoryRateLimiter {
	limiter := &InMemoryRateLimiter{
		buckets: make(map[string]*TokenBucket),
		logger:  logger,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// Allow checks if a request is allowed for the given key
func (rl *InMemoryRateLimiter) Allow(key string) (bool, error) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		// Create new bucket with default settings
		bucket = &TokenBucket{
			Capacity:     100, // Default limit
			Tokens:       100,
			RefillRate:   100, // Refill rate per minute
			LastRefill:   time.Now(),
			WindowStart:  time.Now(),
			WindowEnd:    time.Now().Add(time.Minute),
			RequestCount: 0,
		}
		rl.buckets[key] = bucket
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	now := time.Now()

	// Check if we need to start a new window
	if now.After(bucket.WindowEnd) {
		bucket.WindowStart = now
		bucket.WindowEnd = now.Add(time.Minute)
		bucket.RequestCount = 0
		bucket.Tokens = bucket.Capacity
	}

	// Refill tokens based on time elapsed
	timeSinceRefill := now.Sub(bucket.LastRefill)
	tokensToAdd := int64(timeSinceRefill.Minutes()) * bucket.RefillRate
	if tokensToAdd > 0 {
		bucket.Tokens = minInt64(bucket.Capacity, bucket.Tokens+tokensToAdd)
		bucket.LastRefill = now
	}

	// Check if request is allowed
	if bucket.Tokens > 0 {
		bucket.Tokens--
		bucket.RequestCount++
		return true, nil
	}

	bucket.RequestCount++
	return false, nil
}

// Reset resets the rate limit for a key
func (rl *InMemoryRateLimiter) Reset(key string) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.buckets, key)
	return nil
}

// GetStats returns statistics for a rate limit key
func (rl *InMemoryRateLimiter) GetStats(key string) (*RateLimitStats, error) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	return &RateLimitStats{
		Key:          key,
		RequestCount: bucket.RequestCount,
		WindowStart:  bucket.WindowStart,
		WindowEnd:    bucket.WindowEnd,
		Limit:        bucket.Capacity,
		Remaining:    bucket.Tokens,
		ResetTime:    bucket.WindowEnd,
	}, nil
}

// cleanup removes expired buckets
func (rl *InMemoryRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mutex.Lock()
			now := time.Now()
			for key, bucket := range rl.buckets {
				bucket.mutex.Lock()
				if now.Sub(bucket.WindowEnd) > 10*time.Minute {
					delete(rl.buckets, key)
				}
				bucket.mutex.Unlock()
			}
			rl.mutex.Unlock()
		}
	}
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(config RateLimitConfig, limiter RateLimiter, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		// Skip rate limiting for certain paths
		path := c.Request.URL.Path
		for _, skipPath := range config.SkipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		// Skip rate limiting for trusted IPs
		clientIP := c.ClientIP()
		for _, skipIP := range config.SkipIPs {
			if clientIP == skipIP {
				c.Next()
				return
			}
		}

		// Determine rate limit key
		key := getRateLimitKey(c, config)

		// Get limit configuration for this request
		_, window := getRateLimitConfig(c, config)

		// Check rate limit
		allowed, err := limiter.Allow(key)
		if err != nil {
			logger.WithError(err).WithField("key", key).Error("Rate limit check failed")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Rate limit check failed",
			})
			c.Abort()
			return
		}

		// Get current stats
		stats, _ := limiter.GetStats(key)

		// Set rate limit headers
		if stats != nil {
			c.Header("X-RateLimit-Limit", strconv.FormatInt(stats.Limit, 10))
			c.Header("X-RateLimit-Remaining", strconv.FormatInt(stats.Remaining, 10))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(stats.ResetTime.Unix(), 10))
			c.Header("X-RateLimit-Window", window.String())
		}

		if !allowed {
			logger.WithFields(logrus.Fields{
				"ip":     clientIP,
				"path":   path,
				"method": c.Request.Method,
				"key":    key,
			}).Warn("Rate limit exceeded")

			retryAfter := time.Until(stats.ResetTime)
			c.Header("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"limit":       stats.Limit,
				"remaining":   stats.Remaining,
				"reset_time":  stats.ResetTime,
				"retry_after": retryAfter.String(),
			})
			c.Abort()
			return
		}

		// Log rate limit info
		logger.WithFields(logrus.Fields{
			"ip":        clientIP,
			"path":      path,
			"method":    c.Request.Method,
			"key":       key,
			"remaining": stats.Remaining,
		}).Debug("Rate limit check passed")

		c.Next()
	}
}

// getRateLimitKey determines the key for rate limiting
func getRateLimitKey(c *gin.Context, config RateLimitConfig) string {
	// Check for custom header key resolver
	if config.HeaderKeyResolver != "" {
		if headerValue := c.GetHeader(config.HeaderKeyResolver); headerValue != "" {
			return fmt.Sprintf("header:%s", headerValue)
		}
	}

	// Use authenticated user ID if available
	if userID, exists := c.Get("user_id"); exists {
		return fmt.Sprintf("user:%v", userID)
	}

	// Use client IP as fallback
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// getRateLimitConfig determines the rate limit configuration for a request
func getRateLimitConfig(c *gin.Context, config RateLimitConfig) (int64, time.Duration) {
	path := c.Request.URL.Path
	method := c.Request.Method

	// Check for endpoint-specific limits
	for _, endpointLimit := range config.EndpointLimits {
		if (endpointLimit.Method == "" || endpointLimit.Method == method) &&
			(endpointLimit.Path == "" || endpointLimit.Path == path) {
			return endpointLimit.Limit, endpointLimit.Window
		}
	}

	// Check for user-specific limits
	if userID, exists := c.Get("user_id"); exists {
		userIDStr := fmt.Sprintf("%v", userID)
		for _, userLimit := range config.UserLimits {
			if userLimit.UserID == userIDStr {
				return userLimit.Limit, userLimit.Window
			}
		}

		// Check for role-specific limits
		if role, exists := c.Get("user_role"); exists {
			roleStr := fmt.Sprintf("%v", role)
			for _, userLimit := range config.UserLimits {
				if userLimit.Role == roleStr {
					return userLimit.Limit, userLimit.Window
				}
			}
		}
	}

	// Return default limits
	return config.DefaultLimit, config.DefaultWindow
}

// Helper functions
func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// GetDefaultRateLimitConfig returns default rate limiting configuration
func GetDefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:       true,
		DefaultLimit:  100,
		DefaultWindow: time.Minute,
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
		},
		SkipIPs: []string{
			"127.0.0.1",
			"::1",
		},
		EndpointLimits: map[string]EndpointLimit{
			"auth_login": {
				Method: "POST",
				Path:   "/api/auth/login",
				Limit:  10,
				Window: time.Minute,
			},
			"auth_register": {
				Method: "POST",
				Path:   "/api/auth/register",
				Limit:  5,
				Window: time.Minute,
			},
			"rnd_operations": {
				Method: "POST",
				Path:   "/api/rnd/*",
				Limit:  20,
				Window: time.Minute,
			},
		},
		UserLimits: map[string]UserLimit{
			"admin": {
				Role:   "admin",
				Limit:  1000,
				Window: time.Minute,
			},
			"user": {
				Role:   "user",
				Limit:  200,
				Window: time.Minute,
			},
		},
	}
}
