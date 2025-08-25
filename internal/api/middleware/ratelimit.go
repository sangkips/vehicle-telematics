package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fleet-backend/pkg/ratelimit"

	"github.com/gin-gonic/gin"
)

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter ratelimit.RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health checks in development
		if c.Request.URL.Path == "/api/v1/health" && gin.Mode() == gin.DebugMode {
			c.Next()
			return
		}
		
		// Get client identifier
		clientID := getClientID(c)
		
		// Get endpoint identifier
		endpoint := getEndpointID(c)
		
		// Check rate limit
		allowed, resetTime, err := limiter.Allow(clientID, endpoint)
		if err != nil {
			// Log error but don't block request on rate limiter failure
			c.Header("X-RateLimit-Error", "Rate limiter unavailable")
			c.Next()
			return
		}
		
		// Get current limits for headers
		limits := limiter.GetLimits(clientID)
		endpointKey := getEndpointKey(endpoint, c.Request.Method)
		
		var currentLimit ratelimit.RateLimit
		if limit, exists := limits[endpointKey]; exists {
			currentLimit = limit
		} else if limit, exists := limits["default"]; exists {
			currentLimit = limit
		} else {
			// Fallback default
			currentLimit = ratelimit.RateLimit{
				RequestsPerMinute: 60,
				BurstSize:         15,
				WindowSize:        time.Minute,
			}
		}
		
		// Set rate limit headers
		setRateLimitHeaders(c, currentLimit, allowed, resetTime)
		
		if !allowed {
			// Request blocked by rate limiter
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": fmt.Sprintf("Too many requests. Try again in %v", resetTime),
				"code":    "RATE_LIMIT_EXCEEDED",
				"retryAfter": int(resetTime.Seconds()),
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// getClientID extracts a unique client identifier from the request
func getClientID(c *gin.Context) string {
	// Priority order for client identification:
	// 1. User ID from JWT token (for authenticated requests)
	// 2. API key (if implemented)
	// 3. IP address + User-Agent (for anonymous requests)
	
	// Check for authenticated user
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok && uid != "" {
			return fmt.Sprintf("user:%s", uid)
		}
	}
	
	// Check for API key in headers
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		return fmt.Sprintf("api:%s", apiKey)
	}
	
	// Fallback to IP + User-Agent hash for anonymous requests
	ip := getClientIP(c)
	userAgent := c.GetHeader("User-Agent")
	
	// Create a simple hash of IP + User-Agent for anonymous identification
	return fmt.Sprintf("anon:%s:%s", ip, hashString(userAgent))
}

// getClientIP extracts the real client IP address
func getClientIP(c *gin.Context) string {
	// Check for forwarded headers first
	if forwarded := c.GetHeader("X-Forwarded-For"); forwarded != "" {
		// Take the first IP in the chain
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}
	
	if realIP := c.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}
	
	// Fallback to remote address
	return c.ClientIP()
}

// hashString creates a simple hash of a string for client identification
func hashString(s string) string {
	if s == "" {
		return "unknown"
	}
	
	// Simple hash function for demonstration
	// In production, consider using a proper hash function
	hash := uint32(0)
	for _, c := range s {
		hash = hash*31 + uint32(c)
	}
	
	return fmt.Sprintf("%x", hash)[:8] // Take first 8 characters
}

// getEndpointID creates a unique identifier for the endpoint
func getEndpointID(c *gin.Context) string {
	method := c.Request.Method
	path := c.Request.URL.Path
	
	// Normalize path by replacing IDs with placeholders
	normalizedPath := normalizePath(path)
	
	return fmt.Sprintf("%s:%s", method, normalizedPath)
}

// normalizePath replaces dynamic segments with placeholders
func normalizePath(path string) string {
	// Replace common ID patterns with placeholders
	// This helps group similar endpoints together for rate limiting
	
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		// Check if segment looks like an ID (UUID, ObjectID, or numeric)
		if isID(segment) {
			segments[i] = "*"
		}
	}
	
	return strings.Join(segments, "/")
}

// isID checks if a string looks like an ID
func isID(s string) bool {
	if s == "" {
		return false
	}
	
	// Check for MongoDB ObjectID (24 hex characters)
	if len(s) == 24 {
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				break
			}
		}
		return true
	}
	
	// Check for UUID pattern (8-4-4-4-12 hex characters)
	if len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-' {
		return true
	}
	
	// Check for numeric ID
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	
	return false
}

// getEndpointKey maps an endpoint to a rate limit category
func getEndpointKey(endpoint, method string) string {
	// This should match the logic in config.go
	endpointMap := map[string]string{
		"POST:/api/v1/auth/login":    "auth_login",
		"POST:/api/v1/auth/logout":   "auth_logout",
		"POST:/api/v1/auth/register": "auth",
		"GET:/api/v1/auth/profile":   "auth",
		
		"GET:/api/v1/vehicles":      "vehicles",
		"POST:/api/v1/vehicles":     "vehicles_create",
		"PATCH:/api/v1/vehicles/*":  "vehicles_update",
		"DELETE:/api/v1/vehicles/*": "vehicles_delete",
		
		"GET:/api/v1/alerts":     "alerts",
		"POST:/api/v1/alerts":    "alerts_create",
		
		"GET:/api/v1/users":      "users",
		"POST:/api/v1/users":     "users_create",
		"PATCH:/api/v1/users/*":  "users_update",
		"DELETE:/api/v1/users/*": "users_delete",
		
		"GET:/api/v1/maintenance":  "maintenance",
		"POST:/api/v1/maintenance": "maintenance_create",
		
		"GET:/api/v1/health": "health",
	}
	
	if category, exists := endpointMap[endpoint]; exists {
		return category
	}
	
	// Check for wildcard matches
	for pattern, category := range endpointMap {
		if matchesEndpointPattern(endpoint, pattern) {
			return category
		}
	}
	
	return "default"
}

// matchesEndpointPattern checks if an endpoint matches a pattern with wildcards
func matchesEndpointPattern(endpoint, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return len(endpoint) >= len(prefix) && endpoint[:len(prefix)] == prefix
	}
	return endpoint == pattern
}

// setRateLimitHeaders sets standard rate limiting headers
func setRateLimitHeaders(c *gin.Context, limit ratelimit.RateLimit, allowed bool, resetTime time.Duration) {
	// Set standard rate limit headers
	c.Header("X-RateLimit-Limit", strconv.Itoa(limit.RequestsPerMinute))
	c.Header("X-RateLimit-Window", strconv.Itoa(int(limit.WindowSize.Seconds())))
	c.Header("X-RateLimit-Burst", strconv.Itoa(limit.BurstSize))
	
	if !allowed {
		c.Header("Retry-After", strconv.Itoa(int(resetTime.Seconds())))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(resetTime).Unix(), 10))
	}
	
	// Add custom headers for debugging (only in debug mode)
	if gin.Mode() == gin.DebugMode {
		c.Header("X-RateLimit-Allowed", strconv.FormatBool(allowed))
		if resetTime > 0 {
			c.Header("X-RateLimit-Reset-Time", resetTime.String())
		}
	}
}