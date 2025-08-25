package telemetry

import (
	"fleet-backend/internal/models"
	"math"
	"sync"
	"time"
)

// DeltaTracker tracks changes in vehicle data and only sends significant updates
type DeltaTracker struct {
	lastStates map[string]*VehicleSnapshot
	thresholds DeltaThresholds
	mu         sync.RWMutex
}

type VehicleSnapshot struct {
	FuelLevel    float64
	Location     models.Location
	Speed        int
	Status       string
	Odometer     int
	LastUpdate   time.Time
}

type DeltaThresholds struct {
	FuelLevelPercent   float64 // 5% change
	LocationMeters     float64 // 100 meters
	SpeedKmh          int     // 10 km/h change
	OdometerKm        int     // 1 km change
	TimeThreshold     time.Duration // Force update after 15 minutes
}

func NewDeltaTracker() *DeltaTracker {
	return &DeltaTracker{
		lastStates: make(map[string]*VehicleSnapshot),
		thresholds: DeltaThresholds{
			FuelLevelPercent: 5.0,
			LocationMeters:   100.0,
			SpeedKmh:        10,
			OdometerKm:      1,
			TimeThreshold:   15 * time.Minute,
		},
	}
}

// ShouldUpdate determines if a vehicle update should be sent based on changes
func (dt *DeltaTracker) ShouldUpdate(vehicleID string, current *models.Vehicle) (bool, map[string]interface{}) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	
	lastState, exists := dt.lastStates[vehicleID]
	if !exists {
		// First update for this vehicle
		dt.lastStates[vehicleID] = dt.createSnapshot(current)
		return true, dt.createFullUpdate(current)
	}
	
	// Check time threshold - force update after time limit
	if time.Since(lastState.LastUpdate) > dt.thresholds.TimeThreshold {
		dt.lastStates[vehicleID] = dt.createSnapshot(current)
		return true, dt.createFullUpdate(current)
	}
	
	// Check for significant changes
	changes := make(map[string]interface{})
	hasSignificantChange := false
	
	// Fuel level change
	fuelChange := math.Abs(current.FuelLevel - lastState.FuelLevel)
	if fuelChange >= dt.thresholds.FuelLevelPercent {
		changes["fuelLevel"] = current.FuelLevel
		changes["fuelChange"] = current.FuelLevel - lastState.FuelLevel
		hasSignificantChange = true
	}
	
	// Location change
	distance := dt.calculateDistance(lastState.Location, current.Location)
	if distance >= dt.thresholds.LocationMeters {
		changes["location"] = current.Location
		changes["distanceMoved"] = distance
		hasSignificantChange = true
	}
	
	// Speed change
	speedChange := int(math.Abs(float64(current.Speed - lastState.Speed)))
	if speedChange >= dt.thresholds.SpeedKmh {
		changes["speed"] = current.Speed
		changes["speedChange"] = current.Speed - lastState.Speed
		hasSignificantChange = true
	}
	
	// Status change (always significant)
	if current.Status != lastState.Status {
		changes["status"] = current.Status
		changes["previousStatus"] = lastState.Status
		hasSignificantChange = true
	}
	
	// Odometer change
	odometerChange := int(math.Abs(float64(current.Odometer - lastState.Odometer)))
	if odometerChange >= dt.thresholds.OdometerKm {
		changes["odometer"] = current.Odometer
		changes["odometerChange"] = current.Odometer - lastState.Odometer
		hasSignificantChange = true
	}
	
	if hasSignificantChange {
		changes["vehicleId"] = vehicleID
		changes["timestamp"] = time.Now()
		dt.lastStates[vehicleID] = dt.createSnapshot(current)
	}
	
	return hasSignificantChange, changes
}

// createSnapshot creates a snapshot of current vehicle state
func (dt *DeltaTracker) createSnapshot(vehicle *models.Vehicle) *VehicleSnapshot {
	return &VehicleSnapshot{
		FuelLevel:  vehicle.FuelLevel,
		Location:   vehicle.Location,
		Speed:      vehicle.Speed,
		Status:     vehicle.Status,
		Odometer:   vehicle.Odometer,
		LastUpdate: time.Now(),
	}
}

// createFullUpdate creates a full update payload
func (dt *DeltaTracker) createFullUpdate(vehicle *models.Vehicle) map[string]interface{} {
	return map[string]interface{}{
		"vehicleId":  vehicle.ID.Hex(),
		"fuelLevel":  vehicle.FuelLevel,
		"location":   vehicle.Location,
		"speed":      vehicle.Speed,
		"status":     vehicle.Status,
		"odometer":   vehicle.Odometer,
		"timestamp":  time.Now(),
		"updateType": "full",
	}
}

// calculateDistance calculates distance between two locations in meters
func (dt *DeltaTracker) calculateDistance(loc1, loc2 models.Location) float64 {
	const earthRadius = 6371000 // Earth radius in meters
	
	lat1Rad := loc1.Lat * math.Pi / 180
	lat2Rad := loc2.Lat * math.Pi / 180
	deltaLat := (loc2.Lat - loc1.Lat) * math.Pi / 180
	deltaLng := (loc2.Lng - loc1.Lng) * math.Pi / 180
	
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
		math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return earthRadius * c
}

// SetThresholds allows customizing delta thresholds
func (dt *DeltaTracker) SetThresholds(thresholds DeltaThresholds) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.thresholds = thresholds
}

// GetLastState returns the last known state for a vehicle
func (dt *DeltaTracker) GetLastState(vehicleID string) (*VehicleSnapshot, bool) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	state, exists := dt.lastStates[vehicleID]
	return state, exists
}