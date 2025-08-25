package telemetry

import (
	"context"
	"fleet-backend/internal/models"
	"fleet-backend/internal/services"
	"fleet-backend/pkg/batch"
	"fmt"
	"log"
	"sync"
	"time"
)

// OptimizedTelemetryService combines all optimization strategies
type OptimizedTelemetryService struct {
	vehicleService    *services.VehicleService
	scheduler         *AdaptiveScheduler
	deltaTracker      *DeltaTracker
	rateLimiter       *SmartRateLimiter
	batchProcessor    batch.BatchProcessor
	
	// Configuration
	config            TelemetryConfig
	
	// State management
	activeVehicles    map[string]bool
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	
	// Statistics
	stats             TelemetryStats
	statsMux          sync.RWMutex
}

type TelemetryConfig struct {
	EnableAdaptiveScheduling bool
	EnableDeltaUpdates      bool
	EnableRateLimiting      bool
	EnableBatching          bool
	MaxConcurrentUpdates    int
	HealthCheckInterval     time.Duration
}

type TelemetryStats struct {
	TotalUpdatesRequested   int64
	UpdatesSkipped         int64
	UpdatesSent            int64
	RateLimitRejects       int64
	DeltaSkips            int64
	AverageUpdateSize     float64
	LastUpdateTime        time.Time
	ActiveVehicleCount    int
}

func NewOptimizedTelemetryService(vehicleService *services.VehicleService, batchProcessor batch.BatchProcessor) *OptimizedTelemetryService {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &OptimizedTelemetryService{
		vehicleService: vehicleService,
		scheduler:      NewAdaptiveScheduler(),
		deltaTracker:   NewDeltaTracker(),
		rateLimiter:    NewSmartRateLimiter(),
		batchProcessor: batchProcessor,
		config: TelemetryConfig{
			EnableAdaptiveScheduling: true,
			EnableDeltaUpdates:      true,
			EnableRateLimiting:      true,
			EnableBatching:          true,
			MaxConcurrentUpdates:    10,
			HealthCheckInterval:     5 * time.Minute,
		},
		activeVehicles: make(map[string]bool),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start initializes the optimized telemetry service
func (ots *OptimizedTelemetryService) Start() error {
	log.Println("Starting optimized telemetry service...")
	
	// Start batch processor if enabled
	if ots.config.EnableBatching && ots.batchProcessor != nil {
		if err := ots.batchProcessor.Start(); err != nil {
			return fmt.Errorf("failed to start batch processor: %v", err)
		}
	}
	
	// Start health check routine
	go ots.healthCheckLoop()
	
	// Initialize vehicle schedules
	if err := ots.initializeVehicleSchedules(); err != nil {
		return fmt.Errorf("failed to initialize vehicle schedules: %v", err)
	}
	
	log.Println("Optimized telemetry service started successfully")
	return nil
}

// Stop gracefully shuts down the telemetry service
func (ots *OptimizedTelemetryService) Stop() error {
	log.Println("Stopping optimized telemetry service...")
	
	ots.cancel()
	
	if ots.scheduler != nil {
		ots.scheduler.Stop()
	}
	
	if ots.config.EnableBatching && ots.batchProcessor != nil {
		if err := ots.batchProcessor.Stop(); err != nil {
			log.Printf("Error stopping batch processor: %v", err)
		}
	}
	
	log.Println("Optimized telemetry service stopped")
	return nil
}

// ProcessVehicleUpdate processes a vehicle update with all optimizations
func (ots *OptimizedTelemetryService) ProcessVehicleUpdate(vehicleID string, vehicle *models.Vehicle) error {
	ots.incrementTotalRequests()
	
	// 1. Check rate limiting if enabled
	if ots.config.EnableRateLimiting {
		priority := ots.determinePriority(vehicle)
		allowed, retryAfter := ots.rateLimiter.CanMakeRequest(vehicleID, priority)
		
		if !allowed {
			ots.incrementRateLimitRejects()
			log.Printf("Rate limit exceeded for vehicle %s, retry after %v", vehicleID, retryAfter)
			return fmt.Errorf("rate limit exceeded, retry after %v", retryAfter)
		}
	}
	
	// 2. Check if update is significant using delta tracking
	if ots.config.EnableDeltaUpdates {
		shouldUpdate, changes := ots.deltaTracker.ShouldUpdate(vehicleID, vehicle)
		if !shouldUpdate {
			ots.incrementDeltaSkips()
			return nil // Skip insignificant update
		}
		
		// Use delta changes for more efficient updates
		return ots.processDeltaUpdate(vehicleID, changes)
	}
	
	// 3. Process full update
	return ots.processFullUpdate(vehicleID, vehicle)
}

// processDeltaUpdate processes only the changed fields
func (ots *OptimizedTelemetryService) processDeltaUpdate(vehicleID string, changes map[string]interface{}) error {
	if ots.config.EnableBatching && ots.batchProcessor != nil {
		// Convert changes to batch update format
		updateData := ots.convertToBatchUpdate(changes)
		return ots.batchProcessor.AddUpdate(vehicleID, updateData)
	}
	
	// Fallback to direct update
	return ots.directDeltaUpdate(vehicleID, changes)
}

// processFullUpdate processes a complete vehicle update
func (ots *OptimizedTelemetryService) processFullUpdate(vehicleID string, vehicle *models.Vehicle) error {
	if ots.config.EnableBatching && ots.batchProcessor != nil {
		updateData := batch.VehicleUpdateData{
			FuelLevel: &vehicle.FuelLevel,
			Location:  &vehicle.Location,
			Speed:     &vehicle.Speed,
			Status:    &vehicle.Status,
			Odometer:  &vehicle.Odometer,
			Timestamp: time.Now(),
		}
		return ots.batchProcessor.AddUpdate(vehicleID, updateData)
	}
	
	// Fallback to direct service update
	updateReq := &services.UpdateVehicleRequest{
		FuelLevel: vehicle.FuelLevel,
		Location:  &vehicle.Location,
		Speed:     vehicle.Speed,
		Status:    vehicle.Status,
		Odometer:  vehicle.Odometer,
	}
	
	_, err := ots.vehicleService.UpdateVehicle(vehicleID, updateReq)
	if err == nil {
		ots.incrementUpdatesSent()
	}
	return err
}

// UpdateVehicleState updates vehicle state and adjusts scheduling
func (ots *OptimizedTelemetryService) UpdateVehicleState(vehicleID string, state VehicleState) {
	if ots.config.EnableAdaptiveScheduling {
		ots.scheduler.UpdateVehicleState(vehicleID, state, func(id string) {
			// This callback is triggered when it's time to update the vehicle
			ots.scheduleVehicleUpdate(id)
		})
	}
	
	// Update active vehicles tracking
	ots.mu.Lock()
	ots.activeVehicles[vehicleID] = (state == StateActive || state == StateIdle)
	ots.mu.Unlock()
}

// scheduleVehicleUpdate is called by the adaptive scheduler
func (ots *OptimizedTelemetryService) scheduleVehicleUpdate(vehicleID string) {
	// Get current vehicle data
	vehicle, err := ots.vehicleService.GetVehicleByID(vehicleID)
	if err != nil {
		log.Printf("Failed to get vehicle %s for scheduled update: %v", vehicleID, err)
		return
	}
	
	// Process the update with all optimizations
	if err := ots.ProcessVehicleUpdate(vehicleID, vehicle); err != nil {
		log.Printf("Failed to process scheduled update for vehicle %s: %v", vehicleID, err)
	}
}

// initializeVehicleSchedules sets up initial schedules for all vehicles
func (ots *OptimizedTelemetryService) initializeVehicleSchedules() error {
	vehicles, err := ots.vehicleService.GetAllVehicles()
	if err != nil {
		return err
	}
	
	for _, vehicle := range vehicles {
		state := ots.mapStatusToState(vehicle.Status)
		ots.UpdateVehicleState(vehicle.ID.Hex(), state)
	}
	
	return nil
}

// mapStatusToState maps vehicle status to telemetry state
func (ots *OptimizedTelemetryService) mapStatusToState(status string) VehicleState {
	switch status {
	case "active":
		return StateActive
	case "idle":
		return StateIdle
	case "maintenance":
		return StateMaintenance
	case "offline":
		return StateOffline
	default:
		return StateParked
	}
}

// determinePriority determines update priority based on vehicle data
func (ots *OptimizedTelemetryService) determinePriority(vehicle *models.Vehicle) Priority {
	// Critical: Low fuel, alerts, or maintenance status
	if vehicle.FuelLevel < 10 || vehicle.Status == "maintenance" || len(vehicle.Alerts) > 0 {
		return PriorityCritical
	}
	
	// High: Speeding or active status
	if vehicle.Speed > 80 || vehicle.Status == "active" {
		return PriorityHigh
	}
	
	// Medium: Normal active operations
	if vehicle.Status == "idle" {
		return PriorityMedium
	}
	
	// Low: Parked or offline
	return PriorityLow
}

// convertToBatchUpdate converts delta changes to batch update format
func (ots *OptimizedTelemetryService) convertToBatchUpdate(changes map[string]interface{}) batch.VehicleUpdateData {
	updateData := batch.VehicleUpdateData{
		Timestamp: time.Now(),
	}
	
	if fuelLevel, ok := changes["fuelLevel"].(float64); ok {
		updateData.FuelLevel = &fuelLevel
	}
	
	if location, ok := changes["location"].(models.Location); ok {
		updateData.Location = &location
	}
	
	if speed, ok := changes["speed"].(int); ok {
		updateData.Speed = &speed
	}
	
	if status, ok := changes["status"].(string); ok {
		updateData.Status = &status
	}
	
	if odometer, ok := changes["odometer"].(int); ok {
		updateData.Odometer = &odometer
	}
	
	return updateData
}

// directDeltaUpdate performs direct update with only changed fields
func (ots *OptimizedTelemetryService) directDeltaUpdate(vehicleID string, changes map[string]interface{}) error {
	updateReq := &services.UpdateVehicleRequest{}
	
	if fuelLevel, ok := changes["fuelLevel"].(float64); ok {
		updateReq.FuelLevel = fuelLevel
	}
	
	if location, ok := changes["location"].(models.Location); ok {
		updateReq.Location = &location
	}
	
	if speed, ok := changes["speed"].(int); ok {
		updateReq.Speed = speed
	}
	
	if status, ok := changes["status"].(string); ok {
		updateReq.Status = status
	}
	
	if odometer, ok := changes["odometer"].(int); ok {
		updateReq.Odometer = odometer
	}
	
	_, err := ots.vehicleService.UpdateVehicle(vehicleID, updateReq)
	if err == nil {
		ots.incrementUpdatesSent()
	}
	return err
}

// healthCheckLoop performs periodic health checks
func (ots *OptimizedTelemetryService) healthCheckLoop() {
	ticker := time.NewTicker(ots.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ots.performHealthCheck()
		case <-ots.ctx.Done():
			return
		}
	}
}

// performHealthCheck checks system health and adjusts parameters
func (ots *OptimizedTelemetryService) performHealthCheck() {
	stats := ots.GetStats()
	
	// Log current statistics
	log.Printf("Telemetry Stats - Total: %d, Sent: %d, Skipped: %d, Rate Limited: %d, Delta Skips: %d, Active Vehicles: %d",
		stats.TotalUpdatesRequested, stats.UpdatesSent, stats.UpdatesSkipped,
		stats.RateLimitRejects, stats.DeltaSkips, stats.ActiveVehicleCount)
	
	// Adjust thresholds based on performance
	if stats.RateLimitRejects > stats.UpdatesSent/2 {
		log.Println("High rate limit rejection rate detected, consider adjusting thresholds")
	}
}

// Statistics methods
func (ots *OptimizedTelemetryService) incrementTotalRequests() {
	ots.statsMux.Lock()
	defer ots.statsMux.Unlock()
	ots.stats.TotalUpdatesRequested++
}

func (ots *OptimizedTelemetryService) incrementUpdatesSent() {
	ots.statsMux.Lock()
	defer ots.statsMux.Unlock()
	ots.stats.UpdatesSent++
	ots.stats.LastUpdateTime = time.Now()
}

func (ots *OptimizedTelemetryService) incrementRateLimitRejects() {
	ots.statsMux.Lock()
	defer ots.statsMux.Unlock()
	ots.stats.RateLimitRejects++
}

func (ots *OptimizedTelemetryService) incrementDeltaSkips() {
	ots.statsMux.Lock()
	defer ots.statsMux.Unlock()
	ots.stats.DeltaSkips++
}

// GetStats returns current telemetry statistics
func (ots *OptimizedTelemetryService) GetStats() TelemetryStats {
	ots.statsMux.RLock()
	defer ots.statsMux.RUnlock()
	
	ots.mu.RLock()
	activeCount := len(ots.activeVehicles)
	ots.mu.RUnlock()
	
	stats := ots.stats
	stats.ActiveVehicleCount = activeCount
	return stats
}