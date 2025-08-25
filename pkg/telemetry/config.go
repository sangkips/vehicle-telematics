package telemetry

import (
	"os"
	"strconv"
	"time"
)

// LoadTelemetryConfig loads telemetry configuration from environment variables
func LoadTelemetryConfig() TelemetryConfig {
	config := TelemetryConfig{
		EnableAdaptiveScheduling: true,
		EnableDeltaUpdates:      true,
		EnableRateLimiting:      true,
		EnableBatching:          true,
		MaxConcurrentUpdates:    10,
		HealthCheckInterval:     5 * time.Minute,
	}
	
	// Load from environment variables
	if val := os.Getenv("TELEMETRY_ADAPTIVE_SCHEDULING"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.EnableAdaptiveScheduling = enabled
		}
	}
	
	if val := os.Getenv("TELEMETRY_DELTA_UPDATES"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.EnableDeltaUpdates = enabled
		}
	}
	
	if val := os.Getenv("TELEMETRY_RATE_LIMITING"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.EnableRateLimiting = enabled
		}
	}
	
	if val := os.Getenv("TELEMETRY_BATCHING"); val != "" {
		if enabled, err := strconv.ParseBool(val); err == nil {
			config.EnableBatching = enabled
		}
	}
	
	if val := os.Getenv("TELEMETRY_MAX_CONCURRENT"); val != "" {
		if maxConcurrent, err := strconv.Atoi(val); err == nil && maxConcurrent > 0 {
			config.MaxConcurrentUpdates = maxConcurrent
		}
	}
	
	if val := os.Getenv("TELEMETRY_HEALTH_CHECK_INTERVAL"); val != "" {
		if interval, err := time.ParseDuration(val); err == nil {
			config.HealthCheckInterval = interval
		}
	}
	
	return config
}

// GetOptimalUpdateIntervals returns recommended update intervals based on vehicle state
func GetOptimalUpdateIntervals() map[VehicleState]time.Duration {
	return map[VehicleState]time.Duration{
		StateActive:      30 * time.Second,  // Active vehicles - frequent updates
		StateIdle:        2 * time.Minute,   // Idle vehicles - moderate updates
		StateParked:      10 * time.Minute,  // Parked vehicles - infrequent updates
		StateMaintenance: 30 * time.Minute,  // Maintenance - minimal updates
		StateOffline:     1 * time.Hour,     // Offline - very rare updates
	}
}

// GetDeltaThresholds returns recommended delta thresholds for change detection
func GetDeltaThresholds() DeltaThresholds {
	return DeltaThresholds{
		FuelLevelPercent: 5.0,              // 5% fuel change
		LocationMeters:   100.0,            // 100 meter movement
		SpeedKmh:        10,                // 10 km/h speed change
		OdometerKm:      1,                 // 1 km odometer change
		TimeThreshold:   15 * time.Minute,  // Force update after 15 minutes
	}
}