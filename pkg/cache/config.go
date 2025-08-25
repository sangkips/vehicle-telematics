package cache

import "time"

// CacheConfig holds configuration for cache TTL values and behavior
type CacheConfig struct {
	VehicleDataTTL    time.Duration `json:"vehicleDataTTL"`    // 30 seconds for critical data
	VehicleListTTL    time.Duration `json:"vehicleListTTL"`    // 2 minutes for list data
	AlertDataTTL      time.Duration `json:"alertDataTTL"`      // 10 seconds for alerts
	HistoricalDataTTL time.Duration `json:"historicalDataTTL"` // 10 minutes for historical data
	MaxMemoryUsage    int64         `json:"maxMemoryUsage"`    // 100MB limit
	EvictionPolicy    string        `json:"evictionPolicy"`    // "lru"
	KeyPrefix         string        `json:"keyPrefix"`         // prefix for all cache keys
	TagPrefix         string        `json:"tagPrefix"`         // prefix for tag keys
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		VehicleDataTTL:    30 * time.Second,
		VehicleListTTL:    2 * time.Minute,
		AlertDataTTL:      10 * time.Second,
		HistoricalDataTTL: 10 * time.Minute,
		MaxMemoryUsage:    100 * 1024 * 1024, // 100MB
		EvictionPolicy:    "lru",
		KeyPrefix:         "fleet:",
		TagPrefix:         "tag:",
	}
}

// GetTTLForDataType returns appropriate TTL based on data type
func (c CacheConfig) GetTTLForDataType(dataType string) time.Duration {
	switch dataType {
	case "vehicle":
		return c.VehicleDataTTL
	case "vehicle_list":
		return c.VehicleListTTL
	case "alert":
		return c.AlertDataTTL
	case "historical":
		return c.HistoricalDataTTL
	default:
		return c.VehicleDataTTL
	}
}