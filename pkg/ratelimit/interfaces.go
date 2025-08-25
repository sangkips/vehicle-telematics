package ratelimit

import (
	"time"
)

// RateLimiter defines the interface for rate limiting functionality
type RateLimiter interface {
	Allow(clientID string, endpoint string) (bool, time.Duration, error)
	GetLimits(clientID string) map[string]RateLimit
	SetCustomLimit(clientID string, endpoint string, limit RateLimit) error
	GetStats() RateLimiterStats
}

// RateLimit defines the configuration for rate limiting
type RateLimit struct {
	RequestsPerMinute int           `json:"requestsPerMinute"`
	BurstSize         int           `json:"burstSize"`
	WindowSize        time.Duration `json:"windowSize"`
}

// RateLimiterStats provides statistics about rate limiting
type RateLimiterStats struct {
	TotalRequests     int64   `json:"totalRequests"`
	BlockedRequests   int64   `json:"blockedRequests"`
	AverageLatency    float64 `json:"averageLatency"`
	ActiveClients     int     `json:"activeClients"`
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	Capacity     int       `json:"capacity"`
	Tokens       int       `json:"tokens"`
	RefillRate   int       `json:"refillRate"`
	LastRefill   time.Time `json:"lastRefill"`
	WindowStart  time.Time `json:"windowStart"`
}