package services

import (
	"errors"
	"fleet-backend/internal/models"
	"fleet-backend/internal/repository"
	"fleet-backend/internal/websocket"
	"fleet-backend/pkg/batch"
	"fleet-backend/pkg/cache"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type VehicleService struct {
	vehicleRepo     *repository.VehicleRepository
	alertRepo       *repository.AlertRepository
	cacheManager    cache.CacheManager
	cacheConfig     cache.CacheConfig
	batchProcessor  batch.BatchProcessor
	wsManager       websocket.WebSocketManager
}

func NewVehicleService(vehicleRepo *repository.VehicleRepository) *VehicleService {
	return &VehicleService{
		vehicleRepo: vehicleRepo,
		cacheConfig: cache.DefaultCacheConfig(),
	}
}

// SetAlertRepository allows setting the alert repository for alert generation
func (s *VehicleService) SetAlertRepository(alertRepo *repository.AlertRepository) {
	s.alertRepo = alertRepo
}

// SetCacheManager allows setting the cache manager for caching operations
func (s *VehicleService) SetCacheManager(cacheManager cache.CacheManager) {
	s.cacheManager = cacheManager
}

// SetCacheConfig allows setting custom cache configuration
func (s *VehicleService) SetCacheConfig(config cache.CacheConfig) {
	s.cacheConfig = config
}

// SetBatchProcessor allows setting the batch processor for optimized updates
func (s *VehicleService) SetBatchProcessor(batchProcessor batch.BatchProcessor) {
	s.batchProcessor = batchProcessor
}

// SetWebSocketManager allows setting the WebSocket manager for real-time updates
func (s *VehicleService) SetWebSocketManager(wsManager websocket.WebSocketManager) {
	s.wsManager = wsManager
}

type CreateVehicleRequest struct {
	Name             string  `json:"name" validate:"required,min=1,max=100"`
	PlateNumber      string  `json:"plateNumber" validate:"required,min=1,max=20"`
	Driver           string  `json:"driver" validate:"required,min=1,max=100"`
	Make             string  `json:"make,omitempty"`
	Model            string  `json:"model,omitempty"`
	Year             int     `json:"year,omitempty" validate:"omitempty,min=1900,max=2030"`
	VIN              string  `json:"vin,omitempty"`
	MaxFuelCapacity  float64 `json:"maxFuelCapacity" validate:"required,min=1"`
	FuelConsumption  float64 `json:"fuelConsumption" validate:"required,min=0.1"`
}

type UpdateVehicleRequest struct {
	Name             string             `json:"name,omitempty"`
	PlateNumber      string             `json:"plateNumber,omitempty"`
	Driver           string             `json:"driver,omitempty"`
	FuelLevel        float64            `json:"fuelLevel,omitempty"`
	Location         *models.Location   `json:"location,omitempty"`
	Speed            int                `json:"speed,omitempty"`
	Status           string             `json:"status,omitempty" validate:"omitempty,oneof=active idle maintenance offline"`
	Odometer         int                `json:"odometer,omitempty"`
	Make             string             `json:"make,omitempty"`
	Model            string             `json:"model,omitempty"`
	Year             int                `json:"year,omitempty"`
	VIN              string             `json:"vin,omitempty"`
	MaxFuelCapacity  float64            `json:"maxFuelCapacity,omitempty"`
	FuelConsumption  float64            `json:"fuelConsumption,omitempty"`
}

func (s *VehicleService) GetAllVehicles() ([]*models.Vehicle, error) {
	// Try cache first if cache manager is available
	if s.cacheManager != nil {
		cacheKey := "all_vehicles"
		cachedVehicles, err := s.cacheManager.GetVehicleList(cacheKey)
		if err == nil && cachedVehicles != nil {
			return cachedVehicles, nil
		}
		// Log cache miss but continue to database
		if err != nil {
			fmt.Printf("Cache error for GetAllVehicles: %v\n", err)
		}
	}

	// Fallback to database
	vehicles, err := s.vehicleRepo.FindAll()
	if err != nil {
		return nil, err
	}

	// Cache the result if cache manager is available
	if s.cacheManager != nil {
		ttl := s.cacheConfig.GetTTLForDataType("vehicle_list")
		if cacheErr := s.cacheManager.SetVehicleList("all_vehicles", vehicles, ttl); cacheErr != nil {
			fmt.Printf("Failed to cache all vehicles: %v\n", cacheErr)
		}
	}

	return vehicles, nil
}

func (s *VehicleService) GetVehicleByID(id string) (*models.Vehicle, error) {
	// Try cache first if cache manager is available
	if s.cacheManager != nil {
		cachedVehicle, err := s.cacheManager.GetVehicle(id)
		if err == nil && cachedVehicle != nil {
			return cachedVehicle, nil
		}
		// Log cache miss but continue to database
		if err != nil {
			fmt.Printf("Cache error for GetVehicleByID(%s): %v\n", id, err)
		}
	}

	// Fallback to database
	vehicle, err := s.vehicleRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Cache the result if cache manager is available
	if s.cacheManager != nil {
		ttl := s.cacheConfig.GetTTLForDataType("vehicle")
		if cacheErr := s.cacheManager.SetVehicle(id, vehicle, ttl); cacheErr != nil {
			fmt.Printf("Failed to cache vehicle %s: %v\n", id, cacheErr)
		}
	}

	return vehicle, nil
}

func (s *VehicleService) CreateVehicle(req *CreateVehicleRequest) (*models.Vehicle, error) {
	// Check if plate number already exists
	existingVehicle, _ := s.vehicleRepo.FindByPlateNumber(req.PlateNumber)
	if existingVehicle != nil {
		return nil, errors.New("plate number already exists")
	}

	// Create vehicle model
	vehicle := &models.Vehicle{
		ID:               primitive.NewObjectID(),
		Name:             req.Name,
		PlateNumber:      req.PlateNumber,
		Driver:           req.Driver,
		FuelLevel:        100.0,
		MaxFuelCapacity:  req.MaxFuelCapacity,
		Location: models.Location{
			Lat:     40.7128, // Default to NYC coordinates
			Lng:     -74.0060,
			Address: "New York, NY",
		},
		Speed:           0,
		Status:          "idle",
		LastUpdate:      time.Now(),
		Odometer:        0,
		FuelConsumption: req.FuelConsumption,
		Alerts:          []models.Alert{},
		Make:            req.Make,
		Model:           req.Model,
		Year:            req.Year,
		VIN:             req.VIN,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	createdVehicle, err := s.vehicleRepo.Create(vehicle)
	if err != nil {
		return nil, err
	}

	// Invalidate relevant cache entries after successful creation
	if s.cacheManager != nil {
		s.invalidateCacheOnCreate(createdVehicle)
	}

	return createdVehicle, nil
}

func (s *VehicleService) UpdateVehicle(id string, req *UpdateVehicleRequest) (*models.Vehicle, error) {
	// Find existing vehicle
	vehicle, err := s.vehicleRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("vehicle not found")
	}

	// Store previous values for cache invalidation
	previousFuelLevel := vehicle.FuelLevel
	previousDriver := vehicle.Driver
	previousStatus := vehicle.Status

	// Update fields if provided
	if req.Name != "" {
		vehicle.Name = req.Name
	}
	if req.PlateNumber != "" {
		// Check if new plate number is already taken
		existingVehicle, _ := s.vehicleRepo.FindByPlateNumber(req.PlateNumber)
		if existingVehicle != nil && existingVehicle.ID.Hex() != id {
			return nil, errors.New("plate number already exists")
		}
		vehicle.PlateNumber = req.PlateNumber
	}
	if req.Driver != "" {
		vehicle.Driver = req.Driver
	}
	if req.FuelLevel > 0 {
		vehicle.FuelLevel = req.FuelLevel
	}
	if req.Location != nil {
		vehicle.Location = *req.Location
	}
	if req.Speed >= 0 {
		vehicle.Speed = req.Speed
	}
	if req.Status != "" {
		vehicle.Status = req.Status
	}
	if req.Odometer > 0 {
		vehicle.Odometer = req.Odometer
	}
	if req.Make != "" {
		vehicle.Make = req.Make
	}
	if req.Model != "" {
		vehicle.Model = req.Model
	}
	if req.Year > 0 {
		vehicle.Year = req.Year
	}
	if req.VIN != "" {
		vehicle.VIN = req.VIN
	}
	if req.MaxFuelCapacity > 0 {
		vehicle.MaxFuelCapacity = req.MaxFuelCapacity
	}
	if req.FuelConsumption > 0 {
		vehicle.FuelConsumption = req.FuelConsumption
	}

	vehicle.LastUpdate = time.Now()
	vehicle.UpdatedAt = time.Now()

	// Check for fuel theft if fuel level was updated
	if req.FuelLevel > 0 && s.alertRepo != nil {
		s.checkFuelTheft(vehicle, previousFuelLevel)
		s.checkLowFuel(vehicle)
		s.checkSpeeding(vehicle)
	}

	updatedVehicle, err := s.vehicleRepo.Update(id, vehicle)
	if err != nil {
		return nil, err
	}

	// Invalidate relevant cache entries after successful update
	if s.cacheManager != nil {
		s.invalidateCacheOnUpdate(updatedVehicle, previousDriver, previousStatus)
	}

	return updatedVehicle, nil
}

func (s *VehicleService) DeleteVehicle(id string) error {
	// Check if vehicle exists and get it for cache invalidation
	vehicle, err := s.vehicleRepo.FindByID(id)
	if err != nil {
		return errors.New("vehicle not found")
	}

	err = s.vehicleRepo.Delete(id)
	if err != nil {
		return err
	}

	// Invalidate relevant cache entries after successful deletion
	if s.cacheManager != nil {
		s.invalidateCacheOnDelete(vehicle)
	}

	return nil
}

func (s *VehicleService) GetVehicleUpdates() ([]*models.Vehicle, error) {
	// Simply return all vehicles without simulation
	// The optimized telemetry service handles updates separately
	vehicles, err := s.vehicleRepo.FindAll()
	if err != nil {
		return nil, err
	}

	return vehicles, nil
}

func (s *VehicleService) GetVehiclesByStatus(status string) ([]*models.Vehicle, error) {
	// Try cache first if cache manager is available
	if s.cacheManager != nil {
		cacheKey := fmt.Sprintf("vehicles_by_status_%s", status)
		cachedVehicles, err := s.cacheManager.GetVehicleList(cacheKey)
		if err == nil && cachedVehicles != nil {
			return cachedVehicles, nil
		}
		// Log cache miss but continue to database
		if err != nil {
			fmt.Printf("Cache error for GetVehiclesByStatus(%s): %v\n", status, err)
		}
	}

	// Fallback to database
	vehicles, err := s.vehicleRepo.FindByStatus(status)
	if err != nil {
		return nil, err
	}

	// Cache the result if cache manager is available
	if s.cacheManager != nil {
		cacheKey := fmt.Sprintf("vehicles_by_status_%s", status)
		ttl := s.cacheConfig.GetTTLForDataType("vehicle_list")
		if cacheErr := s.cacheManager.SetVehicleList(cacheKey, vehicles, ttl); cacheErr != nil {
			fmt.Printf("Failed to cache vehicles by status %s: %v\n", status, cacheErr)
		}
	}

	return vehicles, nil
}

func (s *VehicleService) GetVehiclesByDriver(driver string) ([]*models.Vehicle, error) {
	// Try cache first if cache manager is available
	if s.cacheManager != nil {
		cacheKey := fmt.Sprintf("vehicles_by_driver_%s", driver)
		cachedVehicles, err := s.cacheManager.GetVehicleList(cacheKey)
		if err == nil && cachedVehicles != nil {
			return cachedVehicles, nil
		}
		// Log cache miss but continue to database
		if err != nil {
			fmt.Printf("Cache error for GetVehiclesByDriver(%s): %v\n", driver, err)
		}
	}

	// Fallback to database
	vehicles, err := s.vehicleRepo.FindByDriver(driver)
	if err != nil {
		return nil, err
	}

	// Cache the result if cache manager is available
	if s.cacheManager != nil {
		cacheKey := fmt.Sprintf("vehicles_by_driver_%s", driver)
		ttl := s.cacheConfig.GetTTLForDataType("vehicle_list")
		if cacheErr := s.cacheManager.SetVehicleList(cacheKey, vehicles, ttl); cacheErr != nil {
			fmt.Printf("Failed to cache vehicles by driver %s: %v\n", driver, cacheErr)
		}
	}

	return vehicles, nil
}

// simulateVehicleUpdates simulates real-time vehicle data changes using batch processing
func (s *VehicleService) simulateVehicleUpdates(vehicle *models.Vehicle) {
	now := time.Now()
	
	// Only update if last update was more than 5 seconds ago
	if now.Sub(vehicle.LastUpdate) < 5*time.Second {
		return
	}

	previousFuelLevel := vehicle.FuelLevel
	var updateData batch.VehicleUpdateData
	hasUpdates := false

	// Simulate fuel consumption or theft
	random := rand.Float64()
	if random < 0.02 { // 2% chance of fuel theft
		fuelDrop := 10 + rand.Float64()*20
		newFuelLevel := math.Max(0, vehicle.FuelLevel-fuelDrop)
		updateData.FuelLevel = &newFuelLevel
		hasUpdates = true
		
		// Generate critical alert for fuel theft
		if s.alertRepo != nil && s.wsManager != nil {
			s.broadcastFuelTheftAlert(vehicle, previousFuelLevel, newFuelLevel)
		}
	} else if random < 0.7 { // 70% chance of normal consumption
		consumption := rand.Float64() * 0.5
		newFuelLevel := math.Max(0, vehicle.FuelLevel-consumption)
		updateData.FuelLevel = &newFuelLevel
		hasUpdates = true
	}

	// Simulate location changes for active vehicles
	if vehicle.Status == "active" {
		variation := 0.01
		newLocation := models.Location{
			Lat:     vehicle.Location.Lat + (rand.Float64()-0.5)*variation,
			Lng:     vehicle.Location.Lng + (rand.Float64()-0.5)*variation,
			Address: vehicle.Location.Address, // Keep existing address
		}
		updateData.Location = &newLocation
		hasUpdates = true
		
		// Simulate speed changes
		newSpeed := int(rand.Float64() * 80) // 0-80 km/h
		updateData.Speed = &newSpeed
		
		// Check for speeding alerts
		if newSpeed > 80 && s.alertRepo != nil && s.wsManager != nil {
			s.broadcastSpeedingAlert(vehicle, newSpeed)
		}
		
		// Simulate odometer increase
		newOdometer := vehicle.Odometer + int(rand.Float64()*5) // 0-5 km increase
		updateData.Odometer = &newOdometer
	} else {
		// Vehicle is not active, set speed to 0
		newSpeed := 0
		updateData.Speed = &newSpeed
		hasUpdates = true
	}

	// Set timestamp for the update
	updateData.Timestamp = now

	// Send update to batch processor if we have updates and batch processor is available
	if hasUpdates && s.batchProcessor != nil {
		if err := s.batchProcessor.AddUpdate(vehicle.ID.Hex(), updateData); err != nil {
			// Fallback to direct database update if batch processing fails
			fmt.Printf("Batch processing failed for vehicle %s, falling back to direct update: %v\n", vehicle.ID.Hex(), err)
			s.fallbackToDirectUpdate(vehicle, updateData)
		}
	} else if hasUpdates {
		// No batch processor available, use direct update
		s.fallbackToDirectUpdate(vehicle, updateData)
	}
}

// fallbackToDirectUpdate performs direct database update when batch processing is unavailable
func (s *VehicleService) fallbackToDirectUpdate(vehicle *models.Vehicle, updateData batch.VehicleUpdateData) {
	// Apply updates to vehicle model
	if updateData.FuelLevel != nil {
		vehicle.FuelLevel = *updateData.FuelLevel
	}
	if updateData.Location != nil {
		vehicle.Location = *updateData.Location
	}
	if updateData.Speed != nil {
		vehicle.Speed = *updateData.Speed
	}
	if updateData.Odometer != nil {
		vehicle.Odometer = *updateData.Odometer
	}
	
	vehicle.LastUpdate = updateData.Timestamp
	vehicle.UpdatedAt = updateData.Timestamp

	// Update in database directly
	if _, err := s.vehicleRepo.Update(vehicle.ID.Hex(), vehicle); err != nil {
		fmt.Printf("Failed to update vehicle %s directly: %v\n", vehicle.ID.Hex(), err)
		return
	}

	// Broadcast update via WebSocket if available
	if s.wsManager != nil {
		wsUpdate := s.convertToWebSocketUpdate(vehicle.ID.Hex(), updateData)
		if err := s.wsManager.BroadcastVehicleUpdate(vehicle.ID.Hex(), wsUpdate); err != nil {
			fmt.Printf("Failed to broadcast vehicle update via WebSocket: %v\n", err)
		}
	}
}

// convertToWebSocketUpdate converts batch update data to WebSocket update format
func (s *VehicleService) convertToWebSocketUpdate(vehicleID string, updateData batch.VehicleUpdateData) websocket.VehicleUpdate {
	data := make(map[string]interface{})
	updateType := "simulation"
	priority := websocket.PriorityLow // Default priority for simulation updates
	
	// Add fields that were updated
	if updateData.FuelLevel != nil {
		data["fuelLevel"] = *updateData.FuelLevel
		updateType = "fuel"
		
		// Check for critical fuel levels
		if *updateData.FuelLevel < 10 {
			priority = websocket.PriorityCritical
		} else if *updateData.FuelLevel < 20 {
			priority = websocket.PriorityHigh
		}
	}
	
	if updateData.Location != nil {
		data["location"] = *updateData.Location
		if updateType == "simulation" {
			updateType = "location"
		}
	}
	
	if updateData.Speed != nil {
		data["speed"] = *updateData.Speed
		if updateType == "simulation" {
			updateType = "speed"
		}
		
		// Check for speeding
		if *updateData.Speed > 80 {
			priority = websocket.PriorityHigh
		}
	}
	
	if updateData.Odometer != nil {
		data["odometer"] = *updateData.Odometer
		if updateType == "simulation" {
			updateType = "odometer"
		}
	}
	
	return websocket.VehicleUpdate{
		VehicleID:  vehicleID,
		UpdateType: updateType,
		Data:       data,
		Timestamp:  updateData.Timestamp,
		Priority:   priority,
	}
}

// broadcastFuelTheftAlert broadcasts a critical fuel theft alert
func (s *VehicleService) broadcastFuelTheftAlert(vehicle *models.Vehicle, previousLevel, newLevel float64) {
	fuelDrop := previousLevel - newLevel
	if fuelDrop > 15 { // Threshold for fuel theft detection
		// Create alert in database
		alert := &models.Alert{
			ID:        primitive.NewObjectID(),
			VehicleID: vehicle.ID.Hex(),
			Type:      "fuel_theft",
			Message:   fmt.Sprintf("Abnormal fuel drop detected: %.1fL lost - Possible theft", fuelDrop),
			Severity:  "critical",
			Timestamp: time.Now(),
			Resolved:  false,
		}
		
		if _, err := s.alertRepo.Create(alert); err != nil {
			fmt.Printf("Failed to create fuel theft alert: %v\n", err)
		}
		
		// Broadcast critical alert via WebSocket
		wsUpdate := websocket.VehicleUpdate{
			VehicleID:  vehicle.ID.Hex(),
			UpdateType: "alert",
			Data: map[string]interface{}{
				"alertType":    "fuel_theft",
				"alertId":      alert.ID.Hex(),
				"message":      alert.Message,
				"severity":     alert.Severity,
				"fuelLevel":    newLevel,
				"fuelDrop":     fuelDrop,
				"previousLevel": previousLevel,
			},
			Timestamp: alert.Timestamp,
			Priority:  websocket.PriorityCritical,
		}
		
		if err := s.wsManager.BroadcastVehicleUpdate(vehicle.ID.Hex(), wsUpdate); err != nil {
			fmt.Printf("Failed to broadcast fuel theft alert: %v\n", err)
		}
	}
}

// broadcastSpeedingAlert broadcasts a high priority speeding alert
func (s *VehicleService) broadcastSpeedingAlert(vehicle *models.Vehicle, speed int) {
	// Create alert in database
	alert := &models.Alert{
		ID:        primitive.NewObjectID(),
		VehicleID: vehicle.ID.Hex(),
		Type:      "speeding",
		Message:   fmt.Sprintf("Vehicle exceeding speed limit: %d km/h", speed),
		Severity:  "high",
		Timestamp: time.Now(),
		Resolved:  false,
	}
	
	if _, err := s.alertRepo.Create(alert); err != nil {
		fmt.Printf("Failed to create speeding alert: %v\n", err)
	}
	
	// Broadcast high priority alert via WebSocket
	wsUpdate := websocket.VehicleUpdate{
		VehicleID:  vehicle.ID.Hex(),
		UpdateType: "alert",
		Data: map[string]interface{}{
			"alertType": "speeding",
			"alertId":   alert.ID.Hex(),
			"message":   alert.Message,
			"severity":  alert.Severity,
			"speed":     speed,
			"speedLimit": 80,
		},
		Timestamp: alert.Timestamp,
		Priority:  websocket.PriorityHigh,
	}
	
	if err := s.wsManager.BroadcastVehicleUpdate(vehicle.ID.Hex(), wsUpdate); err != nil {
		fmt.Printf("Failed to broadcast speeding alert: %v\n", err)
	}
}

// Alert generation methods
func (s *VehicleService) checkFuelTheft(vehicle *models.Vehicle, previousLevel float64) {
	fuelDrop := previousLevel - vehicle.FuelLevel
	if fuelDrop > 15 { // Threshold for fuel theft detection
		alert := &models.Alert{
			ID:        primitive.NewObjectID(),
			VehicleID: vehicle.ID.Hex(),
			Type:      "fuel_theft",
			Message:   "Abnormal fuel drop detected - Possible theft",
			Severity:  "critical",
			Timestamp: time.Now(),
			Resolved:  false,
		}
		s.alertRepo.Create(alert)
		
		// Add alert to vehicle
		vehicle.Alerts = append(vehicle.Alerts, *alert)
	}
}

func (s *VehicleService) checkLowFuel(vehicle *models.Vehicle) {
	fuelPercentage := (vehicle.FuelLevel / vehicle.MaxFuelCapacity) * 100
	if fuelPercentage < 20 { // Low fuel threshold
		// Check if alert already exists
		hasLowFuelAlert := false
		for _, alert := range vehicle.Alerts {
			if alert.Type == "low_fuel" && !alert.Resolved {
				hasLowFuelAlert = true
				break
			}
		}
		
		if !hasLowFuelAlert {
			alert := &models.Alert{
				ID:        primitive.NewObjectID(),
				VehicleID: vehicle.ID.Hex(),
				Type:      "low_fuel",
				Message:   "Low fuel level detected",
				Severity:  "medium",
				Timestamp: time.Now(),
				Resolved:  false,
			}
			s.alertRepo.Create(alert)
			
			// Add alert to vehicle
			vehicle.Alerts = append(vehicle.Alerts, *alert)
		}
	}
}

func (s *VehicleService) checkSpeeding(vehicle *models.Vehicle) {
	if vehicle.Speed > 80 { // Speed limit threshold
		alert := &models.Alert{
			ID:        primitive.NewObjectID(),
			VehicleID: vehicle.ID.Hex(),
			Type:      "speeding",
			Message:   "Vehicle exceeding speed limit",
			Severity:  "high",
			Timestamp: time.Now(),
			Resolved:  false,
		}
		s.alertRepo.Create(alert)
		
		// Add alert to vehicle
		vehicle.Alerts = append(vehicle.Alerts, *alert)
	}
}

// Cache invalidation helper methods

// invalidateCacheOnCreate invalidates relevant cache entries when a vehicle is created
func (s *VehicleService) invalidateCacheOnCreate(vehicle *models.Vehicle) {
	// Invalidate all vehicles list
	if err := s.cacheManager.Delete("fleet:vehicle_list:all_vehicles"); err != nil {
		fmt.Printf("Failed to invalidate all vehicles cache: %v\n", err)
	}

	// Invalidate vehicles by status list
	statusCacheKey := fmt.Sprintf("fleet:vehicle_list:vehicles_by_status_%s", vehicle.Status)
	if err := s.cacheManager.Delete(statusCacheKey); err != nil {
		fmt.Printf("Failed to invalidate vehicles by status cache: %v\n", err)
	}

	// Invalidate vehicles by driver list
	driverCacheKey := fmt.Sprintf("fleet:vehicle_list:vehicles_by_driver_%s", vehicle.Driver)
	if err := s.cacheManager.Delete(driverCacheKey); err != nil {
		fmt.Printf("Failed to invalidate vehicles by driver cache: %v\n", err)
	}

	// Cache the new vehicle
	ttl := s.cacheConfig.GetTTLForDataType("vehicle")
	if err := s.cacheManager.SetVehicle(vehicle.ID.Hex(), vehicle, ttl); err != nil {
		fmt.Printf("Failed to cache new vehicle %s: %v\n", vehicle.ID.Hex(), err)
	}
}

// invalidateCacheOnUpdate invalidates relevant cache entries when a vehicle is updated
func (s *VehicleService) invalidateCacheOnUpdate(vehicle *models.Vehicle, previousDriver, previousStatus string) {
	vehicleID := vehicle.ID.Hex()

	// Invalidate the specific vehicle cache
	if err := s.cacheManager.InvalidateVehicle(vehicleID); err != nil {
		fmt.Printf("Failed to invalidate vehicle cache for %s: %v\n", vehicleID, err)
	}

	// Invalidate all vehicles list
	if err := s.cacheManager.Delete("fleet:vehicle_list:all_vehicles"); err != nil {
		fmt.Printf("Failed to invalidate all vehicles cache: %v\n", err)
	}

	// Invalidate current status cache
	statusCacheKey := fmt.Sprintf("fleet:vehicle_list:vehicles_by_status_%s", vehicle.Status)
	if err := s.cacheManager.Delete(statusCacheKey); err != nil {
		fmt.Printf("Failed to invalidate vehicles by status cache: %v\n", err)
	}

	// Invalidate previous status cache if status changed
	if previousStatus != vehicle.Status {
		prevStatusCacheKey := fmt.Sprintf("fleet:vehicle_list:vehicles_by_status_%s", previousStatus)
		if err := s.cacheManager.Delete(prevStatusCacheKey); err != nil {
			fmt.Printf("Failed to invalidate previous vehicles by status cache: %v\n", err)
		}
	}

	// Invalidate current driver cache
	driverCacheKey := fmt.Sprintf("fleet:vehicle_list:vehicles_by_driver_%s", vehicle.Driver)
	if err := s.cacheManager.Delete(driverCacheKey); err != nil {
		fmt.Printf("Failed to invalidate vehicles by driver cache: %v\n", err)
	}

	// Invalidate previous driver cache if driver changed
	if previousDriver != vehicle.Driver {
		prevDriverCacheKey := fmt.Sprintf("fleet:vehicle_list:vehicles_by_driver_%s", previousDriver)
		if err := s.cacheManager.Delete(prevDriverCacheKey); err != nil {
			fmt.Printf("Failed to invalidate previous vehicles by driver cache: %v\n", err)
		}
	}

	// Cache the updated vehicle
	ttl := s.cacheConfig.GetTTLForDataType("vehicle")
	if err := s.cacheManager.SetVehicle(vehicleID, vehicle, ttl); err != nil {
		fmt.Printf("Failed to cache updated vehicle %s: %v\n", vehicleID, err)
	}
}

// invalidateCacheOnDelete invalidates relevant cache entries when a vehicle is deleted
func (s *VehicleService) invalidateCacheOnDelete(vehicle *models.Vehicle) {
	vehicleID := vehicle.ID.Hex()

	// Invalidate the specific vehicle cache
	if err := s.cacheManager.InvalidateVehicle(vehicleID); err != nil {
		fmt.Printf("Failed to invalidate vehicle cache for %s: %v\n", vehicleID, err)
	}

	// Invalidate all vehicles list
	if err := s.cacheManager.Delete("fleet:vehicle_list:all_vehicles"); err != nil {
		fmt.Printf("Failed to invalidate all vehicles cache: %v\n", err)
	}

	// Invalidate vehicles by status list
	statusCacheKey := fmt.Sprintf("fleet:vehicle_list:vehicles_by_status_%s", vehicle.Status)
	if err := s.cacheManager.Delete(statusCacheKey); err != nil {
		fmt.Printf("Failed to invalidate vehicles by status cache: %v\n", err)
	}

	// Invalidate vehicles by driver list
	driverCacheKey := fmt.Sprintf("fleet:vehicle_list:vehicles_by_driver_%s", vehicle.Driver)
	if err := s.cacheManager.Delete(driverCacheKey); err != nil {
		fmt.Printf("Failed to invalidate vehicles by driver cache: %v\n", err)
	}
}