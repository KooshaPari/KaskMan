package security

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	// Redis configuration
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Rate limit rules
	GlobalRPS       int           // Global requests per second
	GlobalBurst     int           // Global burst capacity
	PerIPRPS        int           // Per-IP requests per second
	PerIPBurst      int           // Per-IP burst capacity
	PerUserRPS      int           // Per-user requests per second
	PerUserBurst    int           // Per-user burst capacity
	WindowSize      time.Duration // Time window for rate limiting
	CleanupInterval time.Duration // Cleanup interval for memory store

	// Endpoint-specific limits
	EndpointLimits map[string]EndpointLimit

	// Security settings
	BlockDuration       time.Duration // How long to block IPs that exceed limits
	SuspiciousThreshold int           // Threshold for marking IP as suspicious

	// Exemptions
	WhitelistedIPs   []string // IPs exempt from rate limiting
	WhitelistedUsers []string // Users exempt from rate limiting
}

// EndpointLimit defines rate limits for specific endpoints
type EndpointLimit struct {
	RPS   int // Requests per second
	Burst int // Burst capacity
}

// RateLimiter manages rate limiting for different clients
type RateLimiter struct {
	config      *RateLimitConfig
	redis       *redis.Client
	memoryStore *MemoryStore
	logger      *logrus.Logger
	useRedis    bool

	// IP-based limiters
	ipLimiters map[string]*rate.Limiter
	ipMutex    sync.RWMutex

	// Global limiter
	globalLimiter *rate.Limiter

	// Blocked IPs
	blockedIPs map[string]time.Time
	blockMutex sync.RWMutex
}

// MemoryStore provides in-memory rate limiting storage
type MemoryStore struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(config *RateLimitConfig, logger *logrus.Logger) *RateLimiter {
	rl := &RateLimiter{
		config:        config,
		logger:        logger,
		ipLimiters:    make(map[string]*rate.Limiter),
		blockedIPs:    make(map[string]time.Time),
		globalLimiter: rate.NewLimiter(rate.Limit(config.GlobalRPS), config.GlobalBurst),
		memoryStore:   &MemoryStore{requests: make(map[string][]time.Time)},
	}

	// Try to initialize Redis if configured
	if config.RedisAddr != "" {
		rl.redis = redis.NewClient(&redis.Options{
			Addr:     config.RedisAddr,
			Password: config.RedisPassword,
			DB:       config.RedisDB,
		})

		// Test Redis connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := rl.redis.Ping(ctx).Err(); err != nil {
			logger.WithError(err).Warn("Failed to connect to Redis, falling back to memory store")
			rl.useRedis = false
		} else {
			rl.useRedis = true
			logger.Info("Connected to Redis for rate limiting")
		}
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Middleware returns a Gin middleware for rate limiting
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get client IP
		clientIP := rl.getClientIP(c)

		// Check if IP is blocked
		if rl.isBlocked(clientIP) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "IP temporarily blocked due to rate limit violations",
				"retry_after": rl.config.BlockDuration.Seconds(),
			})
			c.Abort()
			return
		}

		// Check if IP is whitelisted
		if rl.isWhitelisted(clientIP) {
			c.Next()
			return
		}

		// Get user ID if authenticated
		userID, _ := c.Get("user_id")
		userIDStr := ""
		if userID != nil {
			userIDStr = userID.(string)
		}

		// Check if user is whitelisted
		if userIDStr != "" && rl.isUserWhitelisted(userIDStr) {
			c.Next()
			return
		}

		// Get endpoint-specific limits
		endpoint := c.Request.URL.Path
		endpointLimit, hasEndpointLimit := rl.config.EndpointLimits[endpoint]

		// Check global rate limit
		if !rl.globalLimiter.Allow() {
			rl.logger.WithFields(logrus.Fields{
				"client_ip": clientIP,
				"endpoint":  endpoint,
				"user_id":   userIDStr,
			}).Warn("Global rate limit exceeded")

			c.Header("X-RateLimit-Global", fmt.Sprintf("%d", rl.config.GlobalRPS))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", "1")

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Global rate limit exceeded",
				"retry_after": 1,
			})
			c.Abort()
			return
		}

		// Check per-IP rate limit
		if !rl.allowIP(clientIP) {
			rl.handleRateLimit(c, clientIP, "IP", rl.config.PerIPRPS)
			return
		}

		// Check per-user rate limit (if authenticated)
		if userIDStr != "" && !rl.allowUser(userIDStr) {
			rl.handleRateLimit(c, userIDStr, "User", rl.config.PerUserRPS)
			return
		}

		// Check endpoint-specific rate limit
		if hasEndpointLimit {
			key := fmt.Sprintf("endpoint:%s:%s", endpoint, clientIP)
			if !rl.allowEndpoint(key, endpointLimit) {
				rl.handleRateLimit(c, endpoint, "Endpoint", endpointLimit.RPS)
				return
			}
		}

		// Set rate limit headers
		rl.setRateLimitHeaders(c, clientIP, userIDStr)

		c.Next()
	}
}

// getClientIP extracts the real client IP from the request
func (rl *RateLimiter) getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header (for proxies)
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	return c.ClientIP()
}

// isBlocked checks if an IP is temporarily blocked
func (rl *RateLimiter) isBlocked(ip string) bool {
	rl.blockMutex.RLock()
	defer rl.blockMutex.RUnlock()

	blockedUntil, exists := rl.blockedIPs[ip]
	if !exists {
		return false
	}

	if time.Now().After(blockedUntil) {
		// Block has expired, remove it
		delete(rl.blockedIPs, ip)
		return false
	}

	return true
}

// blockIP temporarily blocks an IP address
func (rl *RateLimiter) blockIP(ip string) {
	rl.blockMutex.Lock()
	defer rl.blockMutex.Unlock()

	rl.blockedIPs[ip] = time.Now().Add(rl.config.BlockDuration)

	rl.logger.WithFields(logrus.Fields{
		"ip":       ip,
		"duration": rl.config.BlockDuration,
	}).Warn("IP blocked due to rate limit violations")
}

// isWhitelisted checks if an IP is whitelisted
func (rl *RateLimiter) isWhitelisted(ip string) bool {
	for _, whitelistedIP := range rl.config.WhitelistedIPs {
		if ip == whitelistedIP {
			return true
		}
	}
	return false
}

// isUserWhitelisted checks if a user is whitelisted
func (rl *RateLimiter) isUserWhitelisted(userID string) bool {
	for _, whitelistedUser := range rl.config.WhitelistedUsers {
		if userID == whitelistedUser {
			return true
		}
	}
	return false
}

// allowIP checks if an IP is allowed to make a request
func (rl *RateLimiter) allowIP(ip string) bool {
	rl.ipMutex.RLock()
	limiter, exists := rl.ipLimiters[ip]
	rl.ipMutex.RUnlock()

	if !exists {
		rl.ipMutex.Lock()
		limiter = rate.NewLimiter(rate.Limit(rl.config.PerIPRPS), rl.config.PerIPBurst)
		rl.ipLimiters[ip] = limiter
		rl.ipMutex.Unlock()
	}

	allowed := limiter.Allow()

	// If not allowed, check if we should block this IP
	if !allowed {
		rl.checkSuspiciousActivity(ip)
	}

	return allowed
}

// allowUser checks if a user is allowed to make a request
func (rl *RateLimiter) allowUser(userID string) bool {
	if rl.useRedis {
		return rl.allowUserRedis(userID)
	}
	return rl.allowUserMemory(userID)
}

// allowUserRedis checks user rate limit using Redis
func (rl *RateLimiter) allowUserRedis(userID string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:user:%s", userID)

	// Use Redis sliding window rate limiting
	now := time.Now()
	window := now.Truncate(rl.config.WindowSize)

	pipe := rl.redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(window.Unix()-int64(rl.config.WindowSize.Seconds()), 10))
	pipe.ZCard(ctx, key)
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.Unix()), Member: now.UnixNano()})
	pipe.Expire(ctx, key, rl.config.WindowSize*2)

	results, err := pipe.Exec(ctx)
	if err != nil {
		rl.logger.WithError(err).Error("Redis rate limit check failed")
		return true // Allow request on Redis failure
	}

	count := results[1].(*redis.IntCmd).Val()
	return count < int64(rl.config.PerUserRPS*int(rl.config.WindowSize.Seconds()))
}

// allowUserMemory checks user rate limit using memory store
func (rl *RateLimiter) allowUserMemory(userID string) bool {
	return rl.memoryStore.Allow(fmt.Sprintf("user:%s", userID), rl.config.PerUserRPS, rl.config.WindowSize)
}

// allowEndpoint checks endpoint-specific rate limits
func (rl *RateLimiter) allowEndpoint(key string, limit EndpointLimit) bool {
	if rl.useRedis {
		return rl.allowEndpointRedis(key, limit)
	}
	return rl.memoryStore.Allow(key, limit.RPS, rl.config.WindowSize)
}

// allowEndpointRedis checks endpoint rate limit using Redis
func (rl *RateLimiter) allowEndpointRedis(key string, limit EndpointLimit) bool {
	ctx := context.Background()
	redisKey := fmt.Sprintf("ratelimit:endpoint:%s", key)

	now := time.Now()
	window := now.Truncate(rl.config.WindowSize)

	pipe := rl.redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, redisKey, "-inf", strconv.FormatInt(window.Unix()-int64(rl.config.WindowSize.Seconds()), 10))
	pipe.ZCard(ctx, redisKey)
	pipe.ZAdd(ctx, redisKey, redis.Z{Score: float64(now.Unix()), Member: now.UnixNano()})
	pipe.Expire(ctx, redisKey, rl.config.WindowSize*2)

	results, err := pipe.Exec(ctx)
	if err != nil {
		rl.logger.WithError(err).Error("Redis endpoint rate limit check failed")
		return true
	}

	count := results[1].(*redis.IntCmd).Val()
	return count < int64(limit.RPS*int(rl.config.WindowSize.Seconds()))
}

// checkSuspiciousActivity checks if an IP should be blocked for suspicious activity
func (rl *RateLimiter) checkSuspiciousActivity(ip string) {
	// Simple implementation: block IP after threshold violations
	// In production, you might want more sophisticated detection
	rl.blockIP(ip)
}

// handleRateLimit handles rate limit violations
func (rl *RateLimiter) handleRateLimit(c *gin.Context, identifier, limitType string, limit int) {
	rl.logger.WithFields(logrus.Fields{
		"identifier": identifier,
		"limit_type": limitType,
		"limit":      limit,
		"endpoint":   c.Request.URL.Path,
	}).Warn("Rate limit exceeded")

	retryAfter := int(rl.config.WindowSize.Seconds())

	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	c.Header("X-RateLimit-Remaining", "0")
	c.Header("Retry-After", fmt.Sprintf("%d", retryAfter))

	c.JSON(http.StatusTooManyRequests, gin.H{
		"error":       fmt.Sprintf("%s rate limit exceeded", limitType),
		"limit":       limit,
		"retry_after": retryAfter,
	})
	c.Abort()
}

// setRateLimitHeaders sets rate limit headers in the response
func (rl *RateLimiter) setRateLimitHeaders(c *gin.Context, clientIP, userID string) {
	c.Header("X-RateLimit-Global-Limit", fmt.Sprintf("%d", rl.config.GlobalRPS))
	c.Header("X-RateLimit-IP-Limit", fmt.Sprintf("%d", rl.config.PerIPRPS))

	if userID != "" {
		c.Header("X-RateLimit-User-Limit", fmt.Sprintf("%d", rl.config.PerUserRPS))
	}
}

// Allow checks if a request is allowed for the given key
func (ms *MemoryStore) Allow(key string, limit int, window time.Duration) bool {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-window)

	// Get existing requests for this key
	requests, exists := ms.requests[key]
	if !exists {
		requests = []time.Time{}
	}

	// Remove old requests
	validRequests := []time.Time{}
	for _, req := range requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}

	// Check if we're within the limit
	if len(validRequests) >= limit {
		ms.requests[key] = validRequests
		return false
	}

	// Add this request
	validRequests = append(validRequests, now)
	ms.requests[key] = validRequests

	return true
}

// cleanup periodically cleans up old rate limiters and blocked IPs
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.cleanupIPLimiters()
			rl.cleanupBlockedIPs()
			rl.cleanupMemoryStore()
		}
	}
}

// cleanupIPLimiters removes unused IP limiters
func (rl *RateLimiter) cleanupIPLimiters() {
	rl.ipMutex.Lock()
	defer rl.ipMutex.Unlock()

	// Remove limiters that haven't been used recently
	// This is a simple implementation - in production you might want more sophisticated cleanup
	if len(rl.ipLimiters) > 1000 { // Arbitrary threshold
		rl.ipLimiters = make(map[string]*rate.Limiter)
	}
}

// cleanupBlockedIPs removes expired blocked IPs
func (rl *RateLimiter) cleanupBlockedIPs() {
	rl.blockMutex.Lock()
	defer rl.blockMutex.Unlock()

	now := time.Now()
	for ip, blockedUntil := range rl.blockedIPs {
		if now.After(blockedUntil) {
			delete(rl.blockedIPs, ip)
		}
	}
}

// cleanupMemoryStore removes old requests from memory store
func (rl *RateLimiter) cleanupMemoryStore() {
	rl.memoryStore.mutex.Lock()
	defer rl.memoryStore.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.config.WindowSize * 2) // Keep some buffer

	for key, requests := range rl.memoryStore.requests {
		validRequests := []time.Time{}
		for _, req := range requests {
			if req.After(cutoff) {
				validRequests = append(validRequests, req)
			}
		}

		if len(validRequests) == 0 {
			delete(rl.memoryStore.requests, key)
		} else {
			rl.memoryStore.requests[key] = validRequests
		}
	}
}

// GetStats returns rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.ipMutex.RLock()
	ipLimiterCount := len(rl.ipLimiters)
	rl.ipMutex.RUnlock()

	rl.blockMutex.RLock()
	blockedIPCount := len(rl.blockedIPs)
	rl.blockMutex.RUnlock()

	rl.memoryStore.mutex.RLock()
	memoryStoreKeys := len(rl.memoryStore.requests)
	rl.memoryStore.mutex.RUnlock()

	return map[string]interface{}{
		"ip_limiters":       ipLimiterCount,
		"blocked_ips":       blockedIPCount,
		"memory_store_keys": memoryStoreKeys,
		"using_redis":       rl.useRedis,
	}
}
