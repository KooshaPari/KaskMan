package repositories

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

// Config placeholder for tests
type Config struct {
	Logger *logrus.Logger
}

// MockCacheManager is a mock implementation of CacheManager
type MockCacheManager struct {
	mock.Mock
}

func NewMockCacheManager() *MockCacheManager {
	return &MockCacheManager{}
}

func (m *MockCacheManager) Get(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockCacheManager) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheManager) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheManager) Clear(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockCacheManager) SetMany(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	args := m.Called(ctx, items, expiration)
	return args.Error(0)
}

func (m *MockCacheManager) GetMany(ctx context.Context, keys []string) (map[string]interface{}, error) {
	args := m.Called(ctx, keys)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}