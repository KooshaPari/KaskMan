package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value     interface{}
	ExpiresAt time.Time
}

// IsExpired checks if the cache item has expired
func (c *CacheItem) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// InMemoryCacheManager provides an in-memory cache implementation
type InMemoryCacheManager struct {
	cache  map[string]*CacheItem
	mutex  sync.RWMutex
	logger *logrus.Logger
}

// NewInMemoryCacheManager creates a new in-memory cache manager
func NewInMemoryCacheManager(logger *logrus.Logger) *InMemoryCacheManager {
	cm := &InMemoryCacheManager{
		cache:  make(map[string]*CacheItem),
		logger: logger,
	}

	// Start cleanup goroutine
	go cm.startCleanup()

	return cm
}

// Set stores a value in the cache with an expiration time
func (cm *InMemoryCacheManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	expiresAt := time.Now().Add(expiration)
	if expiration == 0 {
		expiresAt = time.Now().Add(24 * time.Hour) // Default 24 hours
	}

	cm.cache[key] = &CacheItem{
		Value:     value,
		ExpiresAt: expiresAt,
	}

	cm.logger.WithContext(ctx).WithFields(logrus.Fields{
		"key":        key,
		"expires_at": expiresAt,
	}).Debug("Cache item stored")

	return nil
}

// Get retrieves a value from the cache
func (cm *InMemoryCacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	item, exists := cm.cache[key]
	if !exists {
		return fmt.Errorf("cache key not found: %s", key)
	}

	if item.IsExpired() {
		// Remove expired item
		delete(cm.cache, key)
		return fmt.Errorf("cache key expired: %s", key)
	}

	// Convert value to destination type
	if err := cm.convertValue(item.Value, dest); err != nil {
		return fmt.Errorf("failed to convert cache value: %w", err)
	}

	cm.logger.WithContext(ctx).WithField("key", key).Debug("Cache hit")
	return nil
}

// Delete removes a value from the cache
func (cm *InMemoryCacheManager) Delete(ctx context.Context, key string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	delete(cm.cache, key)
	cm.logger.WithContext(ctx).WithField("key", key).Debug("Cache item deleted")
	return nil
}

// Clear removes all cache items matching a pattern
func (cm *InMemoryCacheManager) Clear(ctx context.Context, pattern string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	keysToDelete := make([]string, 0)
	for key := range cm.cache {
		if cm.matchesPattern(key, pattern) {
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(cm.cache, key)
	}

	cm.logger.WithContext(ctx).WithFields(logrus.Fields{
		"pattern": pattern,
		"count":   len(keysToDelete),
	}).Debug("Cache items cleared")

	return nil
}

// SetMany stores multiple values in the cache
func (cm *InMemoryCacheManager) SetMany(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	expiresAt := time.Now().Add(expiration)
	if expiration == 0 {
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	for key, value := range items {
		cm.cache[key] = &CacheItem{
			Value:     value,
			ExpiresAt: expiresAt,
		}
	}

	cm.logger.WithContext(ctx).WithFields(logrus.Fields{
		"count":      len(items),
		"expires_at": expiresAt,
	}).Debug("Multiple cache items stored")

	return nil
}

// GetMany retrieves multiple values from the cache
func (cm *InMemoryCacheManager) GetMany(ctx context.Context, keys []string) (map[string]interface{}, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	result := make(map[string]interface{})
	expiredKeys := make([]string, 0)

	for _, key := range keys {
		item, exists := cm.cache[key]
		if !exists {
			continue
		}

		if item.IsExpired() {
			expiredKeys = append(expiredKeys, key)
			continue
		}

		result[key] = item.Value
	}

	// Remove expired items
	for _, key := range expiredKeys {
		delete(cm.cache, key)
	}

	cm.logger.WithContext(ctx).WithFields(logrus.Fields{
		"requested": len(keys),
		"found":     len(result),
		"expired":   len(expiredKeys),
	}).Debug("Multiple cache items retrieved")

	return result, nil
}

// convertValue converts a cached value to the destination type
func (cm *InMemoryCacheManager) convertValue(value interface{}, dest interface{}) error {
	// Convert to JSON and back to handle type conversion
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	if err := json.Unmarshal(jsonData, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return nil
}

// matchesPattern checks if a key matches a pattern (simple wildcard support)
func (cm *InMemoryCacheManager) matchesPattern(key, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Simple prefix matching for now
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return key == pattern
}

// startCleanup starts a background goroutine to clean up expired cache items
func (cm *InMemoryCacheManager) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute) // Cleanup every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.cleanup()
		}
	}
}

// cleanup removes expired cache items
func (cm *InMemoryCacheManager) cleanup() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	expiredKeys := make([]string, 0)
	for key, item := range cm.cache {
		if item.IsExpired() {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(cm.cache, key)
	}

	if len(expiredKeys) > 0 {
		cm.logger.WithField("expired_count", len(expiredKeys)).Debug("Cleaned up expired cache items")
	}
}

// GetSize returns the current cache size
func (cm *InMemoryCacheManager) GetSize() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return len(cm.cache)
}

// GetStats returns cache statistics
func (cm *InMemoryCacheManager) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	total := len(cm.cache)
	expired := 0

	for _, item := range cm.cache {
		if item.IsExpired() {
			expired++
		}
	}

	return map[string]interface{}{
		"total_items":   total,
		"expired_items": expired,
		"active_items":  total - expired,
	}
}

// NoCacheManager provides a no-op cache implementation for when caching is disabled
type NoCacheManager struct{}

// NewNoCacheManager creates a no-op cache manager
func NewNoCacheManager() *NoCacheManager {
	return &NoCacheManager{}
}

// Set is a no-op for the no-cache manager
func (nc *NoCacheManager) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

// Get always returns a "not found" error for the no-cache manager
func (nc *NoCacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	return fmt.Errorf("cache disabled")
}

// Delete is a no-op for the no-cache manager
func (nc *NoCacheManager) Delete(ctx context.Context, key string) error {
	return nil
}

// Clear is a no-op for the no-cache manager
func (nc *NoCacheManager) Clear(ctx context.Context, pattern string) error {
	return nil
}

// SetMany is a no-op for the no-cache manager
func (nc *NoCacheManager) SetMany(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	return nil
}

// GetMany always returns an empty map for the no-cache manager
func (nc *NoCacheManager) GetMany(ctx context.Context, keys []string) (map[string]interface{}, error) {
	return make(map[string]interface{}), nil
}
