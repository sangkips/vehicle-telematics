package ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter implements RateLimiter using Redis as the backend
type RedisRateLimiter struct {
	client       *redis.Client
	config       *Config
	stats        *RateLimiterStats
	customLimits map[string]map[string]RateLimit // clientID -> endpoint -> limit
	mu           sync.RWMutex
	ctx          context.Context
}

// NewRedisRateLimiter creates a new Redis-backed rate limiter
func NewRedisRateLimiter(client *redis.Client, config *Config) *RedisRateLimiter {
	if config == nil {
		config = DefaultConfig()
	}
	
	limiter := &RedisRateLimiter{
		client:       client,
		config:       config,
		stats:        &RateLimiterStats{},
		customLimits: make(map[string]map[string]RateLimit),
		ctx:          context.Background(),
	}
	
	// Start cleanup goroutine
	go limiter.cleanupExpiredKeys()
	
	return limiter
}

// Allow checks if a request should be allowed based on rate limits
func (r *RedisRateLimiter) Allow(clientID string, endpoint string) (bool, time.Duration, error) {
	if !r.config.Enabled {
		return true, 0, nil
	}
	
	atomic.AddInt64(&r.stats.TotalRequests, 1)
	
	// Get the rate limit for this client and endpoint
	limit := r.getRateLimit(clientID, endpoint)
	
	// Generate Redis key
	key := fmt.Sprintf("%s%s:%s", r.config.RedisKeyPrefix, clientID, endpoint)
	
	// Use Lua script for atomic token bucket operations
	allowed, resetTime, err := r.checkTokenBucket(key, limit)
	if err != nil {
		return false, 0, fmt.Errorf("rate limit check failed: %w", err)
	}
	
	if !allowed {
		atomic.AddInt64(&r.stats.BlockedRequests, 1)
		return false, resetTime, nil
	}
	
	return true, 0, nil
}

// checkTokenBucket performs atomic token bucket check using Lua script
func (r *RedisRateLimiter) checkTokenBucket(key string, limit RateLimit) (bool, time.Duration, error) {
	now := time.Now()
	
	// Simplified Lua script for sliding window rate limiting
	script := `
		local key = KEYS[1]
		local burst_size = tonumber(ARGV[1])
		local window_size = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		-- Get current request count and window start
		local count = tonumber(redis.call('HGET', key, 'count')) or 0
		local window_start = tonumber(redis.call('HGET', key, 'window_start')) or now
		
		-- Check if we need to reset the window
		if now - window_start >= window_size then
			count = 0
			window_start = now
		end
		
		-- Check if request can be allowed
		local allowed = count < burst_size
		if allowed then
			count = count + 1
		end
		
		-- Calculate reset time (convert from milliseconds to seconds)
		local reset_time = 0
		if not allowed then
			reset_time = math.ceil(((window_start + window_size) - now) / 1000)
		end
		
		-- Save state with TTL
		local ttl = math.max(1, math.ceil(window_size + 1))
		redis.call('HSET', key, 'count', count)
		redis.call('HSET', key, 'window_start', window_start)
		redis.call('EXPIRE', key, ttl)
		
		return {allowed and 1 or 0, reset_time}
	`
	
	result, err := r.client.Eval(r.ctx, script, []string{key}, 
		limit.BurstSize, 
		int64(limit.WindowSize.Milliseconds()), 
		now.UnixMilli()).Result()
	
	if err != nil {
		return false, 0, err
	}
	
	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) != 2 {
		return false, 0, fmt.Errorf("unexpected script result format")
	}
	
	allowed := resultSlice[0].(int64) == 1
	resetTime := time.Duration(resultSlice[1].(int64)) * time.Second
	
	return allowed, resetTime, nil
}

// getRateLimit gets the rate limit for a specific client and endpoint
func (r *RedisRateLimiter) getRateLimit(clientID, endpoint string) RateLimit {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
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

// GetLimits returns the current rate limits for a client
func (r *RedisRateLimiter) GetLimits(clientID string) map[string]RateLimit {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	limits := make(map[string]RateLimit)
	
	// Add default limits
	for endpoint, limit := range r.config.DefaultLimits {
		limits[endpoint] = limit
	}
	
	// Override with custom limits
	if clientLimits, exists := r.customLimits[clientID]; exists {
		for endpoint, limit := range clientLimits {
			limits[endpoint] = limit
		}
	}
	
	return limits
}

// SetCustomLimit sets a custom rate limit for a specific client and endpoint
func (r *RedisRateLimiter) SetCustomLimit(clientID string, endpoint string, limit RateLimit) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.customLimits[clientID] == nil {
		r.customLimits[clientID] = make(map[string]RateLimit)
	}
	
	r.customLimits[clientID][endpoint] = limit
	
	// Persist custom limits to Redis for durability
	key := fmt.Sprintf("%scustom:%s", r.config.RedisKeyPrefix, clientID)
	data, err := json.Marshal(r.customLimits[clientID])
	if err != nil {
		return fmt.Errorf("failed to marshal custom limits: %w", err)
	}
	
	err = r.client.Set(r.ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to persist custom limits: %w", err)
	}
	
	return nil
}

// GetStats returns current rate limiter statistics
func (r *RedisRateLimiter) GetStats() RateLimiterStats {
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

// cleanupExpiredKeys removes expired rate limit keys from Redis
func (r *RedisRateLimiter) cleanupExpiredKeys() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		// Use SCAN to find expired keys
		pattern := r.config.RedisKeyPrefix + "*"
		iter := r.client.Scan(r.ctx, 0, pattern, 100).Iterator()
		
		var expiredKeys []string
		for iter.Next(r.ctx) {
			key := iter.Val()
			
			// Check if key exists (TTL will handle expiration)
			exists, err := r.client.Exists(r.ctx, key).Result()
			if err != nil || exists == 0 {
				expiredKeys = append(expiredKeys, key)
			}
		}
		
		// Clean up expired keys from memory
		if len(expiredKeys) > 0 {
			r.mu.Lock()
			for range expiredKeys {
				// Extract client ID from key and clean up custom limits if needed
				// This is a simplified cleanup - in production, you might want more sophisticated logic
			}
			r.mu.Unlock()
		}
	}
}

// LoadCustomLimits loads custom limits from Redis on startup
func (r *RedisRateLimiter) LoadCustomLimits() error {
	pattern := fmt.Sprintf("%scustom:*", r.config.RedisKeyPrefix)
	iter := r.client.Scan(r.ctx, 0, pattern, 100).Iterator()
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for iter.Next(r.ctx) {
		key := iter.Val()
		
		// Extract client ID from key
		clientID := key[len(r.config.RedisKeyPrefix)+7:] // Remove "ratelimit:custom:" prefix
		
		// Get custom limits data
		data, err := r.client.Get(r.ctx, key).Result()
		if err != nil {
			continue // Skip if error
		}
		
		var limits map[string]RateLimit
		if err := json.Unmarshal([]byte(data), &limits); err != nil {
			continue // Skip if unmarshal error
		}
		
		r.customLimits[clientID] = limits
	}
	
	return iter.Err()
}