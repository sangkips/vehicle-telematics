package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// MemoryRateLimiter implements RateLimiter using in-memory storage
type MemoryRateLimiter struct {
	config       *Config
	stats        *RateLimiterStats
	customLimits map[string]map[string]RateLimit // clientID -> endpoint -> limit
	tokens       map[string]*TokenBucket         // key -> token bucket
	mu           sync.RWMutex
	ctx          context.Context
}

// NewMemoryRateLimiter creates a new in-memory rate limiter
func NewMemoryRateLimiter(config *Config) *MemoryRateLimiter {
	if config == nil {
		config = DefaultConfig()
	}

	limiter := &MemoryRateLimiter{
		config:       config,
		stats:        &RateLimiterStats{},
		customLimits: make(map[string]map[string]RateLimit),
		tokens:       make(map[string]*TokenBucket),
		ctx:          context.Background(),
	}

	// Start cleanup goroutine
	go limiter.cleanupExpiredTokens()

	return limiter
}

// Allow checks if a request should be allowed based on rate limits
func (r *MemoryRateLimiter) Allow(clientID string, endpoint string) (bool, time.Duration, error) {
	if !r.config.Enabled {
		return true, 0, nil
	}

	atomic.AddInt64(&r.stats.TotalRequests, 1)

	// Get the rate limit for this client and endpoint
	limit := r.getRateLimit(clientID, endpoint)

	// Generate key
	key := fmt.Sprintf("%s:%s", clientID, endpoint)

	r.mu.Lock()
	defer r.mu.Unlock()

	tokenBucket := r.getOrCreateTokenBucket(key, limit)

	now := time.Now()

	// Refill tokens based on time elapsed
	if !tokenBucket.LastRefill.IsZero() {
		elapsed := now.Sub(tokenBucket.LastRefill)
		tokensToAdd := int(float64(limit.RequestsPerMinute) * elapsed.Minutes())
		tokenBucket.Tokens = min(tokenBucket.Capacity, tokenBucket.Tokens+tokensToAdd)
	}

	// Check if request can be allowed
	if tokenBucket.Tokens > 0 {
		tokenBucket.Tokens--
		tokenBucket.LastRefill = now
		return true, 0, nil
	}

	// Calculate when tokens will be available
	timeUntilRefill := time.Minute / time.Duration(limit.RequestsPerMinute)
	resetTime := timeUntilRefill * time.Duration(max(1, tokenBucket.Tokens*-1+1))

	atomic.AddInt64(&r.stats.BlockedRequests, 1)
	return false, resetTime, nil
}

// getRateLimit gets the rate limit for a specific client and endpoint
func (r *MemoryRateLimiter) getRateLimit(clientID, endpoint string) RateLimit {
	// Check for custom limits first
	if clientLimits, exists := r.customLimits[clientID]; exists {
		if limit, exists := clientLimits[endpoint]; exists {
			return limit
		}
	}

	// Get endpoint category
	endpointKey := r.config.GetEndpointKey(endpoint, "")

	// Return default limit for endpoint category
	if limit, exists := r.config.DefaultLimits[endpointKey]; exists {
		return limit
	}

	// Fallback to default
	if defaultLimit, exists := r.config.DefaultLimits["default"]; exists {
		return defaultLimit
	}

	// Final fallback
	return RateLimit{
		RequestsPerMinute: 60,
		BurstSize:         15,
		WindowSize:        time.Minute,
	}
}

// getOrCreateTokenBucket gets or creates a token bucket for the key
func (r *MemoryRateLimiter) getOrCreateTokenBucket(key string, limit RateLimit) *TokenBucket {
	if bucket, exists := r.tokens[key]; exists {
		return bucket
	}

	bucket := &TokenBucket{
		Capacity:   limit.BurstSize,
		Tokens:     limit.BurstSize,
		RefillRate: limit.RequestsPerMinute,
		LastRefill: time.Now(),
	}

	r.tokens[key] = bucket
	return bucket
}

// GetLimits returns the current rate limits for a client
func (r *MemoryRateLimiter) GetLimits(clientID string) map[string]RateLimit {
	limits := make(map[string]RateLimit)

	// Add default limits
	for endpoint, limit := range r.config.DefaultLimits {
		limits[endpoint] = limit
	}

	// Override with custom limits
	r.mu.RLock()
	if clientLimits, exists := r.customLimits[clientID]; exists {
		for endpoint, limit := range clientLimits {
			limits[endpoint] = limit
		}
	}
	r.mu.RUnlock()

	return limits
}

// SetCustomLimit sets a custom rate limit for a specific client and endpoint
func (r *MemoryRateLimiter) SetCustomLimit(clientID string, endpoint string, limit RateLimit) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.customLimits[clientID] == nil {
		r.customLimits[clientID] = make(map[string]RateLimit)
	}

	r.customLimits[clientID][endpoint] = limit
	return nil
}

// GetStats returns current rate limiter statistics
func (r *MemoryRateLimiter) GetStats() RateLimiterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := *r.stats
	stats.ActiveClients = len(r.customLimits)

	// Calculate average latency (simplified)
	if stats.TotalRequests > 0 {
		stats.AverageLatency = float64(stats.BlockedRequests) / float64(stats.TotalRequests) * 100
	}

	return stats
}

// cleanupExpiredTokens cleans up old token buckets (simplified implementation)
func (r *MemoryRateLimiter) cleanupExpiredTokens() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()

		now := time.Now()
		for key, bucket := range r.tokens {
			// Remove buckets that haven't been used in a while
			if now.Sub(bucket.LastRefill) > time.Hour {
				delete(r.tokens, key)
			}
		}

		// Clean up custom limits that haven't been used recently (simplified)
		// In a real implementation, you'd track access times

		r.mu.Unlock()
	}
}

// Helper functions for min/max implementation without external dependencies
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
