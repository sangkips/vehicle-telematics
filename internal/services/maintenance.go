package services

import (
	"errors"
	"fleet-backend/internal/models"
	"fleet-backend/internal/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MaintenanceService struct {
	maintenanceRepo *repository.MaintenanceRepository
	vehicleRepo     *repository.VehicleRepository
}

func NewMaintenanceService(maintenanceRepo *repository.MaintenanceRepository, vehicleRepo *repository.VehicleRepository) *MaintenanceService {
	return &MaintenanceService{
		maintenanceRepo: maintenanceRepo,
		vehicleRepo:     vehicleRepo,
	}
}

// Maintenance Records
type CreateMaintenanceRequest struct {
	VehicleID       string    `json:"vehicleId" validate:"required"`
	Types           []string  `json:"types" validate:"required,min=1"`
	Description     string    `json:"description" validate:"required"`
	Cost            float64   `json:"cost" validate:"min=0"`
	Currency        string    `json:"currency" validate:"required"`
	ServiceCenter   string    `json:"serviceCenter" validate:"required"`
	PerformedAt     time.Time `json:"performedAt" validate:"required"`
	Odometer        int       `json:"odometer" validate:"min=0"`
	ServiceInterval *int      `json:"serviceInterval,omitempty"` // Optional custom interval
	PartsReplaced   []string  `json:"partsReplaced"`
	Notes           string    `json:"notes,omitempty"`
	Status          string    `json:"status" validate:"required"`
}

type UpdateMaintenanceRequest struct {
	Types               []string   `json:"types,omitempty"`
	Description         string     `json:"description,omitempty"`
	Cost                *float64   `json:"cost,omitempty"`
	Currency            string     `json:"currency,omitempty"`
	ServiceCenter       string     `json:"serviceCenter,omitempty"`
	PerformedAt         *time.Time `json:"performedAt,omitempty"`
	Odometer            *int       `json:"odometer,omitempty"`
	NextServiceOdometer *int       `json:"nextServiceOdometer,omitempty"`
	NextServiceDate     *time.Time `json:"nextServiceDate,omitempty"`
	PartsReplaced       []string   `json:"partsReplaced,omitempty"`
	Notes               string     `json:"notes,omitempty"`
	Status              string     `json:"status,omitempty"`
}

func (s *MaintenanceService) CreateMaintenanceRecord(req *CreateMaintenanceRequest) (*models.MaintenanceRecord, error) {
	// Validate vehicle exists
	vehicle, err := s.vehicleRepo.FindByID(req.VehicleID)
	if err != nil {
		return nil, errors.New("vehicle not found")
	}

	vehicleObjectID, err := primitive.ObjectIDFromHex(req.VehicleID)
	if err != nil {
		return nil, errors.New("invalid vehicle ID")
	}

	// Determine service interval (use custom or calculate from types)
	serviceInterval := s.getServiceIntervalForTypes(req.Types, req.ServiceInterval)
	
	// Calculate next service odometer
	nextServiceOdometer := req.Odometer + serviceInterval

	// Estimate next service date based on vehicle's average daily mileage
	nextServiceDate := s.estimateNextServiceDate(vehicle, req.Odometer, nextServiceOdometer)

	record := &models.MaintenanceRecord{
		VehicleID:           vehicleObjectID,
		Types:               req.Types,
		Description:         req.Description,
		Cost:                req.Cost,
		Currency:            req.Currency,
		ServiceCenter:       req.ServiceCenter,
		PerformedAt:         req.PerformedAt,
		Odometer:            req.Odometer,
		ServiceInterval:     serviceInterval,
		NextServiceOdometer: nextServiceOdometer,
		NextServiceDate:     nextServiceDate,
		PartsReplaced:       req.PartsReplaced,
		Notes:               req.Notes,
		Status:              req.Status,
	}

	err = s.maintenanceRepo.Create(record)
	if err != nil {
		return nil, err
	}

	// Create service reminder
	s.createServiceReminder(req.VehicleID, req.Types, nextServiceDate, &nextServiceOdometer, req.Odometer)

	return record, nil
}

func (s *MaintenanceService) GetMaintenanceRecord(id string) (*models.MaintenanceRecord, error) {
	return s.maintenanceRepo.FindByID(id)
}

func (s *MaintenanceService) GetMaintenanceRecordsByVehicle(vehicleID string) ([]*models.MaintenanceRecord, error) {
	// Validate vehicle exists
	_, err := s.vehicleRepo.FindByID(vehicleID)
	if err != nil {
		return nil, errors.New("vehicle not found")
	}

	return s.maintenanceRepo.FindByVehicleID(vehicleID)
}

func (s *MaintenanceService) GetAllMaintenanceRecords(limit, offset int) ([]*models.MaintenanceRecord, error) {
	return s.maintenanceRepo.FindAll(limit, offset)
}

func (s *MaintenanceService) UpdateMaintenanceRecord(id string, req *UpdateMaintenanceRequest) (*models.MaintenanceRecord, error) {
	record, err := s.maintenanceRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("maintenance record not found")
	}

	// Update fields if provided
	if len(req.Types) > 0 {
		record.Types = req.Types
	}
	if req.Description != "" {
		record.Description = req.Description
	}
	if req.Cost != nil {
		record.Cost = *req.Cost
	}
	if req.Currency != "" {
		record.Currency = req.Currency
	}
	if req.ServiceCenter != "" {
		record.ServiceCenter = req.ServiceCenter
	}
	if req.PerformedAt != nil {
		record.PerformedAt = *req.PerformedAt
	}
	if req.Odometer != nil {
		record.Odometer = *req.Odometer
	}
	if req.NextServiceOdometer != nil {
		record.NextServiceOdometer = *req.NextServiceOdometer
	}
	if req.NextServiceDate != nil {
		record.NextServiceDate = req.NextServiceDate
	}
	if req.PartsReplaced != nil {
		record.PartsReplaced = req.PartsReplaced
	}
	if req.Notes != "" {
		record.Notes = req.Notes
	}
	if req.Status != "" {
		record.Status = req.Status
	}

	err = s.maintenanceRepo.Update(id, record)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *MaintenanceService) DeleteMaintenanceRecord(id string) error {
	_, err := s.maintenanceRepo.FindByID(id)
	if err != nil {
		return errors.New("maintenance record not found")
	}

	return s.maintenanceRepo.Delete(id)
}

// Maintenance Schedules
type CreateScheduleRequest struct {
	VehicleID           string    `json:"vehicleId" validate:"required"`
	Types               []string  `json:"types" validate:"required,min=1"`
	Description         string    `json:"description" validate:"required"`
	IntervalKm          int       `json:"intervalKm" validate:"required,min=1"`
	IntervalDays        *int      `json:"intervalDays,omitempty"`
	LastServiceOdometer int       `json:"lastServiceOdometer" validate:"required,min=0"`
	LastServiceDate     time.Time `json:"lastServiceDate" validate:"required"`
	ServiceCenterName   string    `json:"serviceCenterName" validate:"required"`
}

type UpdateScheduleRequest struct {
	Description         string     `json:"description,omitempty"`
	IntervalKm          *int       `json:"intervalKm,omitempty"`
	IntervalDays        *int       `json:"intervalDays,omitempty"`
	LastServiceOdometer *int       `json:"lastServiceOdometer,omitempty"`
	LastServiceDate     *time.Time `json:"lastServiceDate,omitempty"`
	ServiceCenterName   string     `json:"serviceCenterName,omitempty"`
	IsActive            *bool      `json:"isActive,omitempty"`
}

func (s *MaintenanceService) CreateSchedule(req *CreateScheduleRequest) (*models.MaintenanceSchedule, error) {
	// Validate vehicle exists
	vehicle, err := s.vehicleRepo.FindByID(req.VehicleID)
	if err != nil {
		return nil, errors.New("vehicle not found")
	}

	vehicleObjectID, err := primitive.ObjectIDFromHex(req.VehicleID)
	if err != nil {
		return nil, errors.New("invalid vehicle ID")
	}

	// Calculate next service odometer
	nextServiceOdometer := req.LastServiceOdometer + req.IntervalKm

	// Estimate next service date based on vehicle usage and interval days
	var nextServiceDate *time.Time
	if req.IntervalDays != nil {
		estimatedDate := req.LastServiceDate.AddDate(0, 0, *req.IntervalDays)
		nextServiceDate = &estimatedDate
	} else {
		// Estimate based on vehicle usage patterns
		nextServiceDate = s.estimateNextServiceDate(vehicle, req.LastServiceOdometer, nextServiceOdometer)
	}

	schedule := &models.MaintenanceSchedule{
		VehicleID:           vehicleObjectID,
		Types:               req.Types,
		Description:         req.Description,
		IntervalKm:          req.IntervalKm,
		IntervalDays:        req.IntervalDays,
		LastServiceOdometer: req.LastServiceOdometer,
		LastServiceDate:     req.LastServiceDate,
		NextServiceOdometer: nextServiceOdometer,
		NextServiceDate:     nextServiceDate,
		ServiceCenterName:   req.ServiceCenterName,
		IsActive:            true,
	}

	err = s.maintenanceRepo.CreateSchedule(schedule)
	if err != nil {
		return nil, err
	}

	return schedule, nil
}

func (s *MaintenanceService) GetSchedulesByVehicle(vehicleID string) ([]*models.MaintenanceSchedule, error) {
	// Validate vehicle exists
	_, err := s.vehicleRepo.FindByID(vehicleID)
	if err != nil {
		return nil, errors.New("vehicle not found")
	}

	return s.maintenanceRepo.FindSchedulesByVehicleID(vehicleID)
}

func (s *MaintenanceService) GetUpcomingSchedules(days int) ([]*models.MaintenanceSchedule, error) {
	return s.maintenanceRepo.FindUpcomingSchedules(days)
}

func (s *MaintenanceService) GetAllSchedules() ([]*models.MaintenanceSchedule, error) {
	return s.maintenanceRepo.FindAllSchedules()
}

func (s *MaintenanceService) UpdateSchedule(id string, req *UpdateScheduleRequest) (*models.MaintenanceSchedule, error) {
	schedule, err := s.maintenanceRepo.FindScheduleByID(id)
	if err != nil {
		return nil, errors.New("maintenance schedule not found")
	}

	// Update fields if provided
	if req.Description != "" {
		schedule.Description = req.Description
	}
	if req.IntervalKm != nil {
		schedule.IntervalKm = *req.IntervalKm
		// Recalculate next service odometer if interval changed
		schedule.NextServiceOdometer = schedule.LastServiceOdometer + schedule.IntervalKm
	}
	if req.IntervalDays != nil {
		schedule.IntervalDays = req.IntervalDays
	}
	if req.LastServiceOdometer != nil {
		schedule.LastServiceOdometer = *req.LastServiceOdometer
		// Recalculate next service odometer
		schedule.NextServiceOdometer = schedule.LastServiceOdometer + schedule.IntervalKm
	}
	if req.LastServiceDate != nil {
		schedule.LastServiceDate = *req.LastServiceDate
		// Recalculate next service date if interval days is set
		if schedule.IntervalDays != nil {
			estimatedDate := schedule.LastServiceDate.AddDate(0, 0, *schedule.IntervalDays)
			schedule.NextServiceDate = &estimatedDate
		}
	}
	if req.ServiceCenterName != "" {
		schedule.ServiceCenterName = req.ServiceCenterName
	}
	if req.IsActive != nil {
		schedule.IsActive = *req.IsActive
	}

	err = s.maintenanceRepo.UpdateSchedule(id, schedule)
	if err != nil {
		return nil, err
	}

	return schedule, nil
}

func (s *MaintenanceService) DeleteSchedule(id string) error {
	_, err := s.maintenanceRepo.FindScheduleByID(id)
	if err != nil {
		return errors.New("maintenance schedule not found")
	}

	return s.maintenanceRepo.DeleteSchedule(id)
}

func (s *MaintenanceService) GetSchedule(id string) (*models.MaintenanceSchedule, error) {
	return s.maintenanceRepo.FindScheduleByID(id)
}

// Service Reminders
func (s *MaintenanceService) GetServiceReminders(vehicleID string) ([]*models.ServiceReminder, error) {
	// Validate vehicle exists
	_, err := s.vehicleRepo.FindByID(vehicleID)
	if err != nil {
		return nil, errors.New("vehicle not found")
	}

	reminders, err := s.maintenanceRepo.FindRemindersByVehicleID(vehicleID)
	if err != nil {
		return nil, err
	}

	// Update reminder status
	for _, reminder := range reminders {
		s.updateReminderStatus(reminder)
	}

	return reminders, nil
}

func (s *MaintenanceService) GetOverdueReminders() ([]*models.ServiceReminder, error) {
	return s.maintenanceRepo.FindOverdueReminders()
}

// Helper functions
func (s *MaintenanceService) createServiceReminder(vehicleID string, maintenanceTypes []string, nextServiceDate *time.Time, nextServiceOdometer *int, currentOdometer int) error {
	vehicleObjectID, err := primitive.ObjectIDFromHex(vehicleID)
	if err != nil {
		return err
	}

	reminder := &models.ServiceReminder{
		VehicleID:       vehicleObjectID,
		Types:           maintenanceTypes,
		DueDate:         nextServiceDate,
		DueOdometer:     nextServiceOdometer,
		CurrentOdometer: currentOdometer,
		Priority:        s.calculatePriority(nextServiceDate, nextServiceOdometer, currentOdometer),
		IsOverdue:       false,
	}

	s.updateReminderStatus(reminder)
	return s.maintenanceRepo.CreateReminder(reminder)
}

func (s *MaintenanceService) updateReminderStatus(reminder *models.ServiceReminder) {
	now := time.Now()

	// Calculate days until due
	if reminder.DueDate != nil {
		daysUntil := int(reminder.DueDate.Sub(now).Hours() / 24)
		reminder.DaysUntilDue = &daysUntil
		reminder.IsOverdue = daysUntil < 0
	}

	// Calculate odometer until due
	if reminder.DueOdometer != nil {
		odometerUntil := *reminder.DueOdometer - reminder.CurrentOdometer
		reminder.OdometerUntilDue = &odometerUntil
		if !reminder.IsOverdue {
			reminder.IsOverdue = odometerUntil <= 0
		}
	}

	// Update priority based on urgency
	reminder.Priority = s.calculatePriority(reminder.DueDate, reminder.DueOdometer, reminder.CurrentOdometer)
}

func (s *MaintenanceService) calculatePriority(dueDate *time.Time, dueOdometer *int, currentOdometer int) string {
	if dueDate != nil {
		daysUntil := int(dueDate.Sub(time.Now()).Hours() / 24)
		if daysUntil < 0 {
			return models.PriorityUrgent
		} else if daysUntil <= 7 {
			return models.PriorityHigh
		} else if daysUntil <= 30 {
			return models.PriorityMedium
		}
	}

	if dueOdometer != nil {
		odometerUntil := *dueOdometer - currentOdometer
		if odometerUntil <= 0 {
			return models.PriorityUrgent
		} else if odometerUntil <= 1000 {
			return models.PriorityHigh
		} else if odometerUntil <= 5000 {
			return models.PriorityMedium
		}
	}

	return models.PriorityLow
}

// getServiceIntervalForTypes returns the service interval for multiple maintenance types
// Uses the shortest interval among all types to ensure no service is missed
func (s *MaintenanceService) getServiceIntervalForTypes(maintenanceTypes []string, customInterval *int) int {
	if customInterval != nil && *customInterval > 0 {
		return *customInterval
	}

	if len(maintenanceTypes) == 0 {
		return 10000 // Default interval
	}

	// Find the shortest interval among all types
	shortestInterval := 100000 // Start with a large number
	found := false

	for _, maintenanceType := range maintenanceTypes {
		if interval, exists := models.DefaultServiceIntervals[maintenanceType]; exists {
			if interval < shortestInterval {
				shortestInterval = interval
				found = true
			}
		}
	}

	if found {
		return shortestInterval
	}

	// Default interval if no types found
	return 10000 // 10,000 km default
}

// estimateNextServiceDate estimates when the next service will be due based on vehicle usage patterns
func (s *MaintenanceService) estimateNextServiceDate(vehicle *models.Vehicle, currentOdometer, nextServiceOdometer int) *time.Time {
	// Calculate kilometers until next service
	kmUntilService := nextServiceOdometer - currentOdometer
	
	// Get vehicle's maintenance history to calculate average daily mileage
	avgDailyMileage := s.calculateAverageDailyMileage(vehicle.ID.Hex())
	
	// If we can't calculate average, use a default (50 km/day for fleet vehicles)
	if avgDailyMileage <= 0 {
		avgDailyMileage = 50
	}
	
	// Calculate estimated days until next service
	daysUntilService := float64(kmUntilService) / avgDailyMileage
	
	// Add some buffer (10% extra time)
	daysUntilService *= 1.1
	
	nextServiceDate := time.Now().AddDate(0, 0, int(daysUntilService))
	return &nextServiceDate
}

// calculateAverageDailyMileage calculates the vehicle's average daily mileage based on maintenance history
func (s *MaintenanceService) calculateAverageDailyMileage(vehicleID string) float64 {
	// Get maintenance records for this vehicle
	records, err := s.maintenanceRepo.FindByVehicleID(vehicleID)
	if err != nil || len(records) < 2 {
		return 0 // Not enough data
	}

	// Sort records by date (most recent first)
	// Calculate mileage between records
	var totalMileage int
	var totalDays float64
	
	for i := 0; i < len(records)-1; i++ {
		currentRecord := records[i]
		previousRecord := records[i+1]
		
		mileageDiff := currentRecord.Odometer - previousRecord.Odometer
		daysDiff := currentRecord.PerformedAt.Sub(previousRecord.PerformedAt).Hours() / 24
		
		if mileageDiff > 0 && daysDiff > 0 {
			totalMileage += mileageDiff
			totalDays += daysDiff
		}
	}
	
	if totalDays > 0 {
		return float64(totalMileage) / totalDays
	}
	
	return 0
}

// GetNextServiceDue returns vehicles that are due or approaching their next service
func (s *MaintenanceService) GetNextServiceDue(thresholdKm int) ([]*models.ServiceReminder, error) {
	// Get all vehicles
	vehicles, err := s.vehicleRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var dueReminders []*models.ServiceReminder
	
	for _, vehicle := range vehicles {
		// Get latest maintenance record for each vehicle
		records, err := s.maintenanceRepo.FindByVehicleID(vehicle.ID.Hex())
		if err != nil {
			continue
		}
		
		if len(records) == 0 {
			// No maintenance history - vehicle might need initial service
			reminder := &models.ServiceReminder{
				VehicleID:       vehicle.ID,
				Types:           []string{"inspection"},
				CurrentOdometer: vehicle.Odometer,
				Priority:        models.PriorityMedium,
				IsOverdue:       false,
			}
			dueReminders = append(dueReminders, reminder)
			continue
		}
		
		// Check each maintenance type for this vehicle
		latestRecord := records[0] // Assuming sorted by date desc
		
		// Calculate if service is due based on current odometer vs next service odometer
		if vehicle.Odometer >= latestRecord.NextServiceOdometer-thresholdKm {
			kmUntilService := latestRecord.NextServiceOdometer - vehicle.Odometer
			
			reminder := &models.ServiceReminder{
				VehicleID:        vehicle.ID,
				Types:            latestRecord.Types,
				DueOdometer:      &latestRecord.NextServiceOdometer,
				CurrentOdometer:  vehicle.Odometer,
				OdometerUntilDue: &kmUntilService,
				Priority:         s.calculatePriority(latestRecord.NextServiceDate, &latestRecord.NextServiceOdometer, vehicle.Odometer),
				IsOverdue:        kmUntilService <= 0,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			
			dueReminders = append(dueReminders, reminder)
		}
	}
	
	return dueReminders, nil
}