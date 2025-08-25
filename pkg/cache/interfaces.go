package cache

import (
	"time"

	"fleet-backend/internal/models"
)

// CacheManager defines the interface for caching operations
type CacheManager interface {
	// Vehicle operations
	GetVehicle(vehicleID string) (*models.Vehicle, error)
	SetVehicle(vehicleID string, vehicle *models.Vehicle, ttl time.Duration) error
	InvalidateVehicle(vehicleID string) error
	InvalidateVehiclesByTag(tag string) error
	
	// Vehicle list operations
	GetVehicleList(key string) ([]*models.Vehicle, error)
	SetVehicleList(key string, vehicles []*models.Vehicle, ttl time.Duration) error
	
	// Generic operations
	Get(key string, dest interface{}) error
	Set(key string, value interface{}, ttl time.Duration) error
	Delete(key string) error
	
	// Tag operations for intelligent invalidation
	TagKey(key string, tags ...string) error
	InvalidateByTag(tag string) error
	
	// Statistics and health
	GetCacheStats() CacheStats
	HealthCheck() error
	Close() error
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	HitRate       float64 `json:"hitRate"`
	MissRate      float64 `json:"missRate"`
	MemoryUsage   int64   `json:"memoryUsage"`
	KeyCount      int     `json:"keyCount"`
	EvictionCount int     `json:"evictionCount"`
	TotalHits     int64   `json:"totalHits"`
	TotalMisses   int64   `json:"totalMisses"`
}

// VehicleFilters defines filtering criteria for vehicle lists
type VehicleFilters struct {
	VehicleIDs []string `json:"vehicleIds,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	Drivers    []string `json:"drivers,omitempty"`
	AlertTypes []string `json:"alertTypes,omitempty"`
}