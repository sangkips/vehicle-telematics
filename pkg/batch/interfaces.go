package batch

import (
	"fmt"
	"time"

	"fleet-backend/internal/models"
)

// BatchProcessor defines the interface for batch processing vehicle updates
type BatchProcessor interface {
	AddUpdate(vehicleID string, update VehicleUpdateData) error
	ProcessBatch() error
	SetBatchSize(size int)
	SetBatchInterval(interval time.Duration)
	GetBatchStats() BatchStats
	Start() error
	Stop() error
}

// VehicleUpdateData represents the data for a vehicle update
type VehicleUpdateData struct {
	FuelLevel    *float64         `json:"fuelLevel,omitempty"`
	Location     *models.Location `json:"location,omitempty"`
	Speed        *int             `json:"speed,omitempty"`
	Status       *string          `json:"status,omitempty"`
	Odometer     *int             `json:"odometer,omitempty"`
	Timestamp    time.Time        `json:"timestamp"`
}

// BatchStats provides statistics about batch processing
type BatchStats struct {
	BatchesProcessed int           `json:"batchesProcessed"`
	AverageSize      float64       `json:"averageSize"`
	ProcessingTime   time.Duration `json:"processingTime"`
	ErrorRate        float64       `json:"errorRate"`
	TotalUpdates     int64         `json:"totalUpdates"`
	FailedUpdates    int64         `json:"failedUpdates"`
	LastProcessedAt  time.Time     `json:"lastProcessedAt"`
}

// BatchConfig holds configuration for batch processing
type BatchConfig struct {
	MaxBatchSize      int           `json:"maxBatchSize"`      // 50 vehicles per batch
	BatchInterval     time.Duration `json:"batchInterval"`     // 30 seconds
	MaxWaitTime       time.Duration `json:"maxWaitTime"`       // 5 minutes
	RetryAttempts     int           `json:"retryAttempts"`     // 3 attempts
	RetryBackoff      time.Duration `json:"retryBackoff"`      // exponential backoff
}

// VehicleRepository defines the interface for vehicle data persistence
type VehicleRepository interface {
	UpdateVehicle(vehicleID string, update VehicleUpdateData) error
	UpdateVehiclesBatch(updates map[string]VehicleUpdateData) error
}

// Error definitions for batch processing
var (
	ErrInvalidBatchSize     = fmt.Errorf("invalid batch size: must be greater than 0")
	ErrInvalidBatchInterval = fmt.Errorf("invalid batch interval: must be greater than 0")
	ErrInvalidMaxWaitTime   = fmt.Errorf("invalid max wait time: must be greater than 0")
	ErrInvalidRetryAttempts = fmt.Errorf("invalid retry attempts: must be greater than or equal to 0")
	ErrInvalidRetryBackoff  = fmt.Errorf("invalid retry backoff: must be greater than or equal to 0")
)