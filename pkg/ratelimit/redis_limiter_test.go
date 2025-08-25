package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*redis.Client, func()) {
	// Start miniredis server
	mr, err := miniredis.Run()
	require.NoError(t, err)
	
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	
	// Test connection
	err = client.Ping(context.Background()).Err()
	require.NoError(t, err)
	
	cleanup := func() {
		client.Close()
		mr.Close()
	}
	
	return client, cleanup
}

func TestNewRedisRateLimiter(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	config := DefaultConfig()
	limiter := NewRedisRateLimiter(client, config)
	
	assert.NotNil(t, limiter)
	assert.Equal(t, config, limiter.config)
	assert.NotNil(t, limiter.stats)
	assert.NotNil(t, limiter.customLimits)
}

func TestRedisRateLimiter_Allow_BasicFunctionality(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.DefaultLimits["default"] = RateLimit{
		RequestsPerMinute: 5,
		BurstSize:         3,
		WindowSize:        time.Minute,
	}
	
	limiter := NewRedisRateLimiter(client, config)
	
	clientID := "test-client"
	endpoint := "test-endpoint"
	
	// First 3 requests should be allowed (burst size)
	for i := 0; i < 3; i++ {
		allowed, resetTime, err := limiter.Allow(clientID, endpoint)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
		assert.Equal(t, time.Duration(0), resetTime)
	}
	
	// 4th request should be blocked (exceeded burst)
	allowed, resetTime, err := limiter.Allow(clientID, endpoint)
	assert.NoError(t, err)
	assert.False(t, allowed, "4th request should be blocked")
	assert.Greater(t, resetTime, time.Duration(0))
}

func TestRedisRateLimiter_Allow_WindowReset(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.DefaultLimits["default"] = RateLimit{
		RequestsPerMinute: 10,
		BurstSize:         1,
		WindowSize:        200 * time.Millisecond, // Short window for testing
	}
	
	limiter := NewRedisRateLimiter(client, config)
	
	clientID := "test-client"
	endpoint := "test-endpoint"
	
	// First request should be allowed
	allowed, _, err := limiter.Allow(clientID, endpoint)
	assert.NoError(t, err)
	assert.True(t, allowed)
	
	// Second request should be blocked (burst size = 1)
	allowed, resetTime, err := limiter.Allow(clientID, endpoint)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Greater(t, resetTime, time.Duration(0))
	
	// Wait for window to reset (add some buffer)
	time.Sleep(250 * time.Millisecond)
	
	// Request should be allowed again after window reset
	allowed, _, err = limiter.Allow(clientID, endpoint)
	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestRedisRateLimiter_Allow_DifferentClients(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.DefaultLimits["default"] = RateLimit{
		RequestsPerMinute: 5,
		BurstSize:         1,
		WindowSize:        time.Minute,
	}
	
	limiter := NewRedisRateLimiter(client, config)
	
	endpoint := "test-endpoint"
	
	// Both clients should be allowed their first request
	allowed1, _, err := limiter.Allow("client1", endpoint)
	assert.NoError(t, err)
	assert.True(t, allowed1)
	
	allowed2, _, err := limiter.Allow("client2", endpoint)
	assert.NoError(t, err)
	assert.True(t, allowed2)
	
	// Both clients should be blocked on second request
	allowed1, _, err = limiter.Allow("client1", endpoint)
	assert.NoError(t, err)
	assert.False(t, allowed1)
	
	allowed2, _, err = limiter.Allow("client2", endpoint)
	assert.NoError(t, err)
	assert.False(t, allowed2)
}

func TestRedisRateLimiter_Allow_DifferentEndpoints(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	limiter := NewRedisRateLimiter(client, DefaultConfig())
	
	clientID := "test-client"
	
	// Set custom limits for different endpoints
	err := limiter.SetCustomLimit(clientID, "endpoint1", RateLimit{
		RequestsPerMinute: 5,
		BurstSize:         1,
		WindowSize:        time.Minute,
	})
	assert.NoError(t, err)
	
	err = limiter.SetCustomLimit(clientID, "endpoint2", RateLimit{
		RequestsPerMinute: 5,
		BurstSize:         2,
		WindowSize:        time.Minute,
	})
	assert.NoError(t, err)
	
	// First endpoint allows 1 request
	allowed, _, err := limiter.Allow(clientID, "endpoint1")
	assert.NoError(t, err)
	assert.True(t, allowed)
	
	allowed, _, err = limiter.Allow(clientID, "endpoint1")
	assert.NoError(t, err)
	assert.False(t, allowed)
	
	// Second endpoint allows 2 requests
	allowed, _, err = limiter.Allow(clientID, "endpoint2")
	assert.NoError(t, err)
	assert.True(t, allowed)
	
	allowed, _, err = limiter.Allow(clientID, "endpoint2")
	assert.NoError(t, err)
	assert.True(t, allowed)
	
	allowed, _, err = limiter.Allow(clientID, "endpoint2")
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestRedisRateLimiter_Allow_DisabledLimiter(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.Enabled = false
	
	limiter := NewRedisRateLimiter(client, config)
	
	// All requests should be allowed when limiter is disabled
	for i := 0; i < 10; i++ {
		allowed, resetTime, err := limiter.Allow("client", "endpoint")
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, time.Duration(0), resetTime)
	}
}

func TestRedisRateLimiter_SetCustomLimit(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	limiter := NewRedisRateLimiter(client, DefaultConfig())
	
	clientID := "test-client"
	endpoint := "test-endpoint"
	customLimit := RateLimit{
		RequestsPerMinute: 10,
		BurstSize:         5,
		WindowSize:        time.Minute,
	}
	
	// Set custom limit
	err := limiter.SetCustomLimit(clientID, endpoint, customLimit)
	assert.NoError(t, err)
	
	// Verify custom limit is applied
	limits := limiter.GetLimits(clientID)
	assert.Equal(t, customLimit, limits[endpoint])
	
	// Test that custom limit is used
	for i := 0; i < 5; i++ {
		allowed, _, err := limiter.Allow(clientID, endpoint)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed with custom limit", i+1)
	}
	
	// 6th request should be blocked
	allowed, _, err := limiter.Allow(clientID, endpoint)
	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestRedisRateLimiter_GetLimits(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	config := DefaultConfig()
	limiter := NewRedisRateLimiter(client, config)
	
	clientID := "test-client"
	
	// Get default limits
	limits := limiter.GetLimits(clientID)
	assert.Equal(t, len(config.DefaultLimits), len(limits))
	
	// Set custom limit
	customLimit := RateLimit{RequestsPerMinute: 100, BurstSize: 50, WindowSize: time.Minute}
	err := limiter.SetCustomLimit(clientID, "custom-endpoint", customLimit)
	assert.NoError(t, err)
	
	// Get limits with custom override
	limits = limiter.GetLimits(clientID)
	assert.Equal(t, customLimit, limits["custom-endpoint"])
}

func TestRedisRateLimiter_GetStats(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	limiter := NewRedisRateLimiter(client, DefaultConfig())
	
	// Initial stats
	stats := limiter.GetStats()
	assert.Equal(t, int64(0), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.BlockedRequests)
	
	// Make some requests
	limiter.Allow("client1", "endpoint1")
	limiter.Allow("client1", "endpoint1") // This might be blocked
	limiter.Allow("client2", "endpoint1")
	
	// Check updated stats
	stats = limiter.GetStats()
	assert.Equal(t, int64(3), stats.TotalRequests)
}

func TestRedisRateLimiter_ConcurrentAccess(t *testing.T) {
	client, cleanup := setupTestRedis(t)
	defer cleanup()
	
	config := DefaultConfig()
	config.DefaultLimits["test"] = RateLimit{
		RequestsPerMinute: 100,
		BurstSize:         10,
		WindowSize:        time.Minute,
	}
	
	limiter := NewRedisRateLimiter(client, config)
	
	// Run concurrent requests
	const numGoroutines = 10
	const requestsPerGoroutine = 5
	
	results := make(chan bool, numGoroutines*requestsPerGoroutine)
	
	for i := 0; i < numGoroutines; i++ {
		go func(clientID string) {
			for j := 0; j < requestsPerGoroutine; j++ {
				allowed, _, err := limiter.Allow(clientID, "test")
				if err == nil {
					results <- allowed
				} else {
					results <- false
				}
			}
		}(string(rune('A' + i)))
	}
	
	// Collect results
	allowedCount := 0
	for i := 0; i < numGoroutines*requestsPerGoroutine; i++ {
		if <-results {
			allowedCount++
		}
	}
	
	// Should allow at least some requests (burst size * number of clients)
	assert.Greater(t, allowedCount, 0)
	assert.LessOrEqual(t, allowedCount, numGoroutines*requestsPerGoroutine)
}

func TestConfig_GetEndpointKey(t *testing.T) {
	config := DefaultConfig()
	
	tests := []struct {
		endpoint string
		method   string
		expected string
	}{
		{"/api/v1/auth/login", "POST", "auth_login"},
		{"/api/v1/auth/logout", "POST", "auth_logout"},
		{"/api/v1/vehicles", "GET", "vehicles"},
		{"/api/v1/vehicles", "POST", "vehicles_create"},
		{"/api/v1/vehicles/123", "PATCH", "vehicles_update"},
		{"/api/v1/vehicles/abc", "DELETE", "vehicles_delete"},
		{"/api/v1/health", "GET", "health"},
		{"/api/v1/unknown", "GET", "default"},
	}
	
	for _, tt := range tests {
		t.Run(tt.endpoint+"_"+tt.method, func(t *testing.T) {
			result := config.GetEndpointKey(tt.endpoint, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		key     string
		pattern string
		matches bool
	}{
		{"POST:/api/v1/vehicles/123", "POST:/api/v1/vehicles/*", true},
		{"PATCH:/api/v1/vehicles/abc", "PATCH:/api/v1/vehicles/*", true},
		{"GET:/api/v1/vehicles", "POST:/api/v1/vehicles/*", false},
		{"POST:/api/v1/auth/login", "POST:/api/v1/auth/login", true},
		{"POST:/api/v1/auth/logout", "POST:/api/v1/auth/login", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.key+"_"+tt.pattern, func(t *testing.T) {
			result := matchesPattern(tt.key, tt.pattern)
			assert.Equal(t, tt.matches, result)
		})
	}
}