package services

import (
	"errors"
	"fleet-backend/internal/models"
	"fleet-backend/internal/repository"
	"math"
	"math/rand/v2"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type VehicleService struct {
	vehicleRepo *repository.VehicleRepository
	alertRepo   *repository.AlertRepository
}

func NewVehicleService(vehicleRepo *repository.VehicleRepository) *VehicleService {
	return &VehicleService{
		vehicleRepo: vehicleRepo,
	}
}

// SetAlertRepository allows setting the alert repository for alert generation
func (s *VehicleService) SetAlertRepository(alertRepo *repository.AlertRepository) {
	s.alertRepo = alertRepo
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
	return s.vehicleRepo.FindAll()
}

func (s *VehicleService) GetVehicleByID(id string) (*models.Vehicle, error) {
	return s.vehicleRepo.FindByID(id)
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
		FuelLevel:        100.0, // Start with full tank
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

	return s.vehicleRepo.Create(vehicle)
}

func (s *VehicleService) UpdateVehicle(id string, req *UpdateVehicleRequest) (*models.Vehicle, error) {
	// Find existing vehicle
	vehicle, err := s.vehicleRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("vehicle not found")
	}

	// Store previous fuel level for theft detection
	previousFuelLevel := vehicle.FuelLevel

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

	return s.vehicleRepo.Update(id, vehicle)
}

func (s *VehicleService) DeleteVehicle(id string) error {
	// Check if vehicle exists
	_, err := s.vehicleRepo.FindByID(id)
	if err != nil {
		return errors.New("vehicle not found")
	}

	return s.vehicleRepo.Delete(id)
}

func (s *VehicleService) GetVehicleUpdates() ([]*models.Vehicle, error) {
	vehicles, err := s.vehicleRepo.FindAll()
	if err != nil {
		return nil, err
	}

	// Simulate real-time updates for demo purposes
	for _, vehicle := range vehicles {
		s.simulateVehicleUpdates(vehicle)
	}

	return vehicles, nil
}

func (s *VehicleService) GetVehiclesByStatus(status string) ([]*models.Vehicle, error) {
	return s.vehicleRepo.FindByStatus(status)
}

func (s *VehicleService) GetVehiclesByDriver(driver string) ([]*models.Vehicle, error) {
	return s.vehicleRepo.FindByDriver(driver)
}

// simulateVehicleUpdates simulates real-time vehicle data changes
func (s *VehicleService) simulateVehicleUpdates(vehicle *models.Vehicle) {
	now := time.Now()
	
	// Only update if last update was more than 5 seconds ago
	if now.Sub(vehicle.LastUpdate) < 5*time.Second {
		return
	}

	previousFuelLevel := vehicle.FuelLevel

	// Simulate fuel consumption or theft
	random := rand.Float64()
	if random < 0.02 { // 2% chance of fuel theft
		fuelDrop := 10 + rand.Float64()*20
		vehicle.FuelLevel = math.Max(0, vehicle.FuelLevel-fuelDrop)
		if s.alertRepo != nil {
			s.checkFuelTheft(vehicle, previousFuelLevel)
		}
	} else if random < 0.7 { // 70% chance of normal consumption
		consumption := rand.Float64() * 0.5
		vehicle.FuelLevel = math.Max(0, vehicle.FuelLevel-consumption)
	}

	// Simulate location changes for active vehicles
	if vehicle.Status == "active" {
		variation := 0.01
		vehicle.Location.Lat += (rand.Float64() - 0.5) * variation
		vehicle.Location.Lng += (rand.Float64() - 0.5) * variation
		
		// Simulate speed changes
		vehicle.Speed = int(rand.Float64() * 80) // 0-80 km/h
		
		// Simulate odometer increase
		vehicle.Odometer += int(rand.Float64() * 5) // 0-5 km increase
	} else {
		vehicle.Speed = 0
	}

	// Check for alerts
	if s.alertRepo != nil {
		s.checkLowFuel(vehicle)
		s.checkSpeeding(vehicle)
	}

	vehicle.LastUpdate = now
	vehicle.UpdatedAt = now

	// Update in database
	s.vehicleRepo.Update(vehicle.ID.Hex(), vehicle)
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