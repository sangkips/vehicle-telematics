package ratelimit

import (
	"time"
)

// Config holds the configuration for rate limiting
type Config struct {
	// Default rate limits for different endpoint types
	DefaultLimits map[string]RateLimit `json:"defaultLimits"`
	
	// Redis key prefix for rate limiting data
	RedisKeyPrefix string `json:"redisKeyPrefix"`
	
	// Cleanup interval for expired rate limit data
	CleanupInterval time.Duration `json:"cleanupInterval"`
	
	// Enable/disable rate limiting
	Enabled bool `json:"enabled"`
}

// DefaultConfig returns a default rate limiting configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultLimits: map[string]RateLimit{
			// Authentication endpoints - more restrictive
			"auth":        {RequestsPerMinute: 10, BurstSize: 5, WindowSize: time.Minute},
			"auth_login":  {RequestsPerMinute: 5, BurstSize: 2, WindowSize: time.Minute},
			"auth_logout": {RequestsPerMinute: 10, BurstSize: 5, WindowSize: time.Minute},
			
			// Vehicle endpoints - moderate limits
			"vehicles":        {RequestsPerMinute: 100, BurstSize: 20, WindowSize: time.Minute},
			"vehicles_create": {RequestsPerMinute: 20, BurstSize: 5, WindowSize: time.Minute},
			"vehicles_update": {RequestsPerMinute: 50, BurstSize: 10, WindowSize: time.Minute},
			"vehicles_delete": {RequestsPerMinute: 10, BurstSize: 3, WindowSize: time.Minute},
			
			// Alert endpoints - higher limits for monitoring
			"alerts":        {RequestsPerMinute: 200, BurstSize: 50, WindowSize: time.Minute},
			"alerts_create": {RequestsPerMinute: 100, BurstSize: 20, WindowSize: time.Minute},
			
			// User management - moderate limits
			"users":        {RequestsPerMinute: 50, BurstSize: 10, WindowSize: time.Minute},
			"users_create": {RequestsPerMinute: 10, BurstSize: 3, WindowSize: time.Minute},
			"users_update": {RequestsPerMinute: 20, BurstSize: 5, WindowSize: time.Minute},
			"users_delete": {RequestsPerMinute: 5, BurstSize: 2, WindowSize: time.Minute},
			
			// Maintenance endpoints
			"maintenance":        {RequestsPerMinute: 100, BurstSize: 20, WindowSize: time.Minute},
			"maintenance_create": {RequestsPerMinute: 30, BurstSize: 10, WindowSize: time.Minute},
			
			// Health check - very permissive
			"health": {RequestsPerMinute: 1000, BurstSize: 100, WindowSize: time.Minute},
			
			// Default fallback
			"default": {RequestsPerMinute: 60, BurstSize: 15, WindowSize: time.Minute},
		},
		RedisKeyPrefix:  "ratelimit:",
		CleanupInterval: 5 * time.Minute,
		Enabled:         true,
	}
}

// GetEndpointKey generates a rate limit key for a specific endpoint
func (c *Config) GetEndpointKey(endpoint, method string) string {
	// Map specific endpoints to rate limit categories
	endpointMap := map[string]string{
		"POST:/api/v1/auth/login":    "auth_login",
		"POST:/api/v1/auth/logout":   "auth_logout",
		"POST:/api/v1/auth/register": "auth",
		"GET:/api/v1/auth/profile":   "auth",
		
		"GET:/api/v1/vehicles":     "vehicles",
		"POST:/api/v1/vehicles":    "vehicles_create",
		"PATCH:/api/v1/vehicles/*": "vehicles_update",
		"DELETE:/api/v1/vehicles/*": "vehicles_delete",
		
		"GET:/api/v1/alerts":     "alerts",
		"POST:/api/v1/alerts":    "alerts_create",
		
		"GET:/api/v1/users":     "users",
		"POST:/api/v1/users":    "users_create",
		"PATCH:/api/v1/users/*": "users_update",
		"DELETE:/api/v1/users/*": "users_delete",
		
		"GET:/api/v1/maintenance":  "maintenance",
		"POST:/api/v1/maintenance": "maintenance_create",
		
		"GET:/api/v1/health": "health",
	}
	
	key := method + ":" + endpoint
	if category, exists := endpointMap[key]; exists {
		return category
	}
	
	// Check for wildcard matches
	for pattern, category := range endpointMap {
		if matchesPattern(key, pattern) {
			return category
		}
	}
	
	return "default"
}

// matchesPattern checks if a key matches a pattern with wildcards
func matchesPattern(key, pattern string) bool {
	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}
	return key == pattern
}