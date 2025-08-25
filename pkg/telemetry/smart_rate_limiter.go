package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SmartRateLimiter implements adaptive rate limiting with exponential backoff
type SmartRateLimiter struct {
	vehicleLimits map[string]*VehicleRateLimit
	globalConfig  RateLimitConfig
	mu            sync.RWMutex
}

type VehicleRateLimit struct {
	VehicleID        string
	RequestsPerHour  int
	CurrentRequests  int
	WindowStart      time.Time
	BackoffLevel     int
	LastRejection    time.Time
	ConsecutiveRejects int
}

type RateLimitConfig struct {
	BaseRequestsPerHour    int           // 120 requests/hour (2 per minute)
	MaxRequestsPerHour     int           // 720 requests/hour (12 per minute)
	BackoffMultiplier      float64       // 1.5x backoff
	BackoffResetDuration   time.Duration // Reset after 30 minutes
	BurstAllowance         int           // Allow 5 burst requests
	WindowDuration         time.Duration // 1 hour window
}

func NewSmartRateLimiter() *SmartRateLimiter {
	return &SmartRateLimiter{
		vehicleLimits: make(map[string]*VehicleRateLimit),
		globalConfig: RateLimitConfig{
			BaseRequestsPerHour:  120,  // 2 per minute
			MaxRequestsPerHour:   720,  // 12 per minute max
			BackoffMultiplier:    1.5,
			BackoffResetDuration: 30 * time.Minute,
			BurstAllowance:       5,
			WindowDuration:       time.Hour,
		},
	}
}

// CanMakeRequest checks if a vehicle can make a request
func (srl *SmartRateLimiter) CanMakeRequest(vehicleID string, priority Priority) (bool, time.Duration) {
	srl.mu.Lock()
	defer srl.mu.Unlock()
	
	limit, exists := srl.vehicleLimits[vehicleID]
	if !exists {
		limit = &VehicleRateLimit{
			VehicleID:       vehicleID,
			RequestsPerHour: srl.globalConfig.BaseRequestsPerHour,
			WindowStart:     time.Now(),
		}
		srl.vehicleLimits[vehicleID] = limit
	}
	
	now := time.Now()
	
	// Reset window if needed
	if now.Sub(limit.WindowStart) >= srl.globalConfig.WindowDuration {
		limit.CurrentRequests = 0
		limit.WindowStart = now
	}
	
	// Reset backoff if enough time has passed
	if now.Sub(limit.LastRejection) >= srl.globalConfig.BackoffResetDuration {
		limit.BackoffLevel = 0
		limit.ConsecutiveRejects = 0
	}
	
	// Calculate current limit with backoff
	currentLimit := srl.calculateCurrentLimit(limit)
	
	// Allow critical requests with higher priority
	if priority == PriorityCritical {
		currentLimit += srl.globalConfig.BurstAllowance
	}
	
	// Check if request is allowed
	if limit.CurrentRequests < currentLimit {
		limit.CurrentRequests++
		return true, 0
	}
	
	// Request rejected - apply backoff
	limit.ConsecutiveRejects++
	limit.LastRejection = now
	if limit.BackoffLevel < 5 { // Max backoff level
		limit.BackoffLevel++
	}
	
	// Calculate retry after duration
	retryAfter := srl.calculateRetryAfter(limit)
	return false, retryAfter
}

// calculateCurrentLimit calculates the current rate limit with backoff applied
func (srl *SmartRateLimiter) calculateCurrentLimit(limit *VehicleRateLimit) int {
	baseLimit := srl.globalConfig.BaseRequestsPerHour
	
	// Apply backoff reduction
	if limit.BackoffLevel > 0 {
		reduction := 1.0
		for i := 0; i < limit.BackoffLevel; i++ {
			reduction /= srl.globalConfig.BackoffMultiplier
		}
		return int(float64(baseLimit) * reduction)
	}
	
	return baseLimit
}

// calculateRetryAfter calculates when the next request can be made
func (srl *SmartRateLimiter) calculateRetryAfter(limit *VehicleRateLimit) time.Duration {
	// Base retry time increases with consecutive rejects
	baseRetry := time.Duration(limit.ConsecutiveRejects) * 30 * time.Second
	
	// Apply exponential backoff
	backoffMultiplier := 1.0
	for i := 0; i < limit.BackoffLevel; i++ {
		backoffMultiplier *= srl.globalConfig.BackoffMultiplier
	}
	
	retryDuration := time.Duration(float64(baseRetry) * backoffMultiplier)
	
	// Cap at maximum retry time
	maxRetry := 15 * time.Minute
	if retryDuration > maxRetry {
		retryDuration = maxRetry
	}
	
	return retryDuration
}

// GetVehicleStats returns rate limiting stats for a vehicle
func (srl *SmartRateLimiter) GetVehicleStats(vehicleID string) *VehicleRateLimit {
	srl.mu.RLock()
	defer srl.mu.RUnlock()
	
	if limit, exists := srl.vehicleLimits[vehicleID]; exists {
		// Return a copy to avoid race conditions
		return &VehicleRateLimit{
			VehicleID:          limit.VehicleID,
			RequestsPerHour:    limit.RequestsPerHour,
			CurrentRequests:    limit.CurrentRequests,
			WindowStart:        limit.WindowStart,
			BackoffLevel:       limit.BackoffLevel,
			LastRejection:      limit.LastRejection,
			ConsecutiveRejects: limit.ConsecutiveRejects,
		}
	}
	return nil
}

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// RequestWithBackoff makes a request with automatic retry and backoff
func (srl *SmartRateLimiter) RequestWithBackoff(ctx context.Context, vehicleID string, priority Priority, requestFunc func() error) error {
	maxRetries := 3
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		allowed, retryAfter := srl.CanMakeRequest(vehicleID, priority)
		
		if allowed {
			return requestFunc()
		}
		
		// Wait for retry period or context cancellation
		select {
		case <-time.After(retryAfter):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	return fmt.Errorf("rate limit exceeded after %d attempts for vehicle %s", maxRetries, vehicleID)
}