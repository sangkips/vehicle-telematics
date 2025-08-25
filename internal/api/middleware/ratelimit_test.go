package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fleet-backend/pkg/ratelimit"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestMiddleware(t *testing.T) (*gin.Engine, func()) {
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
	
	// Create rate limiter with test config
	config := ratelimit.DefaultConfig()
	config.RedisKeyPrefix = "test_ratelimit:"
	config.DefaultLimits["default"] = ratelimit.RateLimit{
		RequestsPerMinute: 5,
		BurstSize:         2,
		WindowSize:        time.Minute,
	}
	config.DefaultLimits["auth_login"] = ratelimit.RateLimit{
		RequestsPerMinute: 1,
		BurstSize:         1,
		WindowSize:        100 * time.Millisecond, // Very short window for testing
	}
	
	limiter := ratelimit.NewRedisRateLimiter(client, config)
	
	// Set up Gin router with rate limiting middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimitMiddleware(limiter))
	
	// Add test routes
	router.POST("/api/v1/auth/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "login successful"})
	})
	
	router.GET("/api/v1/vehicles", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"vehicles": []string{}})
	})
	
	cleanup := func() {
		client.Close()
		mr.Close()
	}
	
	return router, cleanup
}

func TestRateLimitMiddleware_BasicFunctionality(t *testing.T) {
	router, cleanup := setupTestMiddleware(t)
	defer cleanup()
	
	// First request should be allowed
	req1 := httptest.NewRequest("GET", "/api/v1/vehicles", nil)
	req1.Header.Set("X-Forwarded-For", "192.168.1.1")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Header().Get("X-RateLimit-Limit"), "100") // vehicles endpoint limit from default config
	
	// Second request should be allowed
	req2 := httptest.NewRequest("GET", "/api/v1/vehicles", nil)
	req2.Header.Set("X-Forwarded-For", "192.168.1.1")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRateLimitMiddleware_RateLimitExceeded(t *testing.T) {
	router, cleanup := setupTestMiddleware(t)
	defer cleanup()
	
	clientIP := "192.168.1.2"
	
	// Make the first request - should be allowed
	req1 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req1.Header.Set("X-Forwarded-For", clientIP)
	req1.Header.Set("User-Agent", "TestAgent/1.0") // Set consistent User-Agent
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	// Check that the correct rate limit is being applied
	limit := w1.Header().Get("X-RateLimit-Limit")
	burst := w1.Header().Get("X-RateLimit-Burst")
	t.Logf("Rate limit headers: Limit=%s, Burst=%s", limit, burst)
	assert.Equal(t, "1", limit)  // auth_login limit
	assert.Equal(t, "1", burst)  // auth_login burst
	
	// Make the second request immediately - should be blocked due to burst size = 1
	req2 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req2.Header.Set("X-Forwarded-For", clientIP)
	req2.Header.Set("User-Agent", "TestAgent/1.0") // Set same User-Agent
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	t.Logf("Second request status: %d, body: %s", w2.Code, w2.Body.String())
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
	assert.NotEmpty(t, w2.Header().Get("Retry-After"))
	assert.Contains(t, w2.Body.String(), "Rate limit exceeded")
}

func TestRateLimitMiddleware_DifferentClients(t *testing.T) {
	router, cleanup := setupTestMiddleware(t)
	defer cleanup()
	
	// Client 1 makes a request
	req1 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req1.Header.Set("X-Forwarded-For", "192.168.1.3")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	
	// Client 2 should still be able to make a request
	req2 := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	req2.Header.Set("X-Forwarded-For", "192.168.1.4")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRateLimitMiddleware_AuthenticatedUser(t *testing.T) {
	router, cleanup := setupTestMiddleware(t)
	defer cleanup()
	
	// Add middleware to set user context (simulating authentication)
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user123")
		c.Next()
	})
	
	// Add a test route that uses the authenticated user context
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "authenticated"})
	})
	
	// Make request as authenticated user
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// The rate limit should be applied per user, not per IP
	// Make another request with same user but different IP
	req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
	req2.Header.Set("X-Forwarded-For", "192.168.1.100")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestRateLimitMiddleware_Headers(t *testing.T) {
	router, cleanup := setupTestMiddleware(t)
	defer cleanup()
	
	req := httptest.NewRequest("GET", "/api/v1/vehicles", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.5")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	// Check that rate limit headers are set
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Window"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Burst"))
}

func TestGetClientID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		setupContext   func(*gin.Context)
		expectedPrefix string
	}{
		{
			name: "authenticated user",
			setupContext: func(c *gin.Context) {
				c.Set("user_id", "user123")
			},
			expectedPrefix: "user:",
		},
		{
			name: "api key",
			setupContext: func(c *gin.Context) {
				c.Request.Header.Set("X-API-Key", "api123")
			},
			expectedPrefix: "api:",
		},
		{
			name: "anonymous user",
			setupContext: func(c *gin.Context) {
				c.Request.Header.Set("X-Forwarded-For", "192.168.1.1")
				c.Request.Header.Set("User-Agent", "TestAgent/1.0")
			},
			expectedPrefix: "anon:",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/test", nil)
			
			tt.setupContext(c)
			
			clientID := getClientID(c)
			assert.Contains(t, clientID, tt.expectedPrefix)
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/api/v1/vehicles", "/api/v1/vehicles"},
		{"/api/v1/vehicles/123", "/api/v1/vehicles/*"},
		{"/api/v1/vehicles/abc123def", "/api/v1/vehicles/*"},
		{"/api/v1/users/user-123/profile", "/api/v1/users/*/profile"},
		{"/api/v1/alerts/64f1a2b3c4d5e6f7a8b9c0d1", "/api/v1/alerts/*"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizePath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},                                    // numeric
		{"64f1a2b3c4d5e6f7a8b9c0d1", true},              // MongoDB ObjectID
		{"550e8400-e29b-41d4-a716-446655440000", true},  // UUID
		{"abc", false},                                   // regular string
		{"user-profile", false},                          // hyphenated string
		{"", false},                                      // empty string
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}