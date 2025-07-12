package services

import (
	"errors"
	"fleet-backend/internal/models"
	"fleet-backend/internal/repository"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AlertService struct {
	alertRepo   *repository.AlertRepository
	vehicleRepo *repository.VehicleRepository
}

func NewAlertService(alertRepo *repository.AlertRepository) *AlertService {
	return &AlertService{
		alertRepo: alertRepo,
	}
}

// SetVehicleRepository allows setting the vehicle repository for vehicle updates
func (s *AlertService) SetVehicleRepository(vehicleRepo *repository.VehicleRepository) {
	s.vehicleRepo = vehicleRepo
}

type CreateAlertRequest struct {
	VehicleID string `json:"vehicleId" validate:"required"`
	Type      string `json:"type" validate:"required,oneof=fuel_theft maintenance speeding unauthorized low_fuel"`
	Message   string `json:"message" validate:"required,min=1,max=500"`
	Severity  string `json:"severity" validate:"required,oneof=low medium high critical"`
}

type UpdateAlertRequest struct {
	Message  string `json:"message,omitempty"`
	Severity string `json:"severity,omitempty" validate:"omitempty,oneof=low medium high critical"`
	Resolved bool   `json:"resolved,omitempty"`
}

func (s *AlertService) GetAllAlerts() ([]*models.Alert, error) {
	return s.alertRepo.FindAll()
}

func (s *AlertService) GetAlertByID(id string) (*models.Alert, error) {
	return s.alertRepo.FindByID(id)
}

func (s *AlertService) GetAlertsByVehicle(vehicleID string) ([]*models.Alert, error) {
	return s.alertRepo.FindByVehicleID(vehicleID)
}

func (s *AlertService) GetAlertsByType(alertType string) ([]*models.Alert, error) {
	return s.alertRepo.FindByType(alertType)
}

func (s *AlertService) GetAlertsBySeverity(severity string) ([]*models.Alert, error) {
	return s.alertRepo.FindBySeverity(severity)
}

func (s *AlertService) GetUnresolvedAlerts() ([]*models.Alert, error) {
	return s.alertRepo.FindUnresolved()
}

func (s *AlertService) CreateAlert(req *CreateAlertRequest) (*models.Alert, error) {
	// Verify vehicle exists
	if s.vehicleRepo != nil {
		_, err := s.vehicleRepo.FindByID(req.VehicleID)
		if err != nil {
			return nil, errors.New("vehicle not found")
		}
	}

	// Create alert model
	alert := &models.Alert{
		ID:        primitive.NewObjectID(),
		VehicleID: req.VehicleID,
		Type:      req.Type,
		Message:   req.Message,
		Severity:  req.Severity,
		Timestamp: time.Now(),
		Resolved:  false,
	}

	createdAlert, err := s.alertRepo.Create(alert)
	if err != nil {
		return nil, err
	}

	// Update vehicle with new alert if vehicle repo is available
	if s.vehicleRepo != nil {
		s.addAlertToVehicle(req.VehicleID, createdAlert)
	}

	return createdAlert, nil
}

func (s *AlertService) UpdateAlert(id string, req *UpdateAlertRequest) (*models.Alert, error) {
	// Find existing alert
	alert, err := s.alertRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("alert not found")
	}

	// Update fields if provided
	if req.Message != "" {
		alert.Message = req.Message
	}
	if req.Severity != "" {
		alert.Severity = req.Severity
	}
	
	// Handle resolution
	if req.Resolved && !alert.Resolved {
		alert.Resolved = true
		alert.ResolvedAt = &time.Time{}
		*alert.ResolvedAt = time.Now()
	} else if !req.Resolved && alert.Resolved {
		alert.Resolved = false
		alert.ResolvedAt = nil
	}

	updatedAlert, err := s.alertRepo.Update(id, alert)
	if err != nil {
		return nil, err
	}

	// Update vehicle alerts if vehicle repo is available
	if s.vehicleRepo != nil {
		s.updateVehicleAlert(alert.VehicleID, updatedAlert)
	}

	return updatedAlert, nil
}

func (s *AlertService) ResolveAlert(id string) (*models.Alert, error) {
	alert, err := s.alertRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("alert not found")
	}

	if alert.Resolved {
		return alert, nil // Already resolved
	}

	alert.Resolved = true
	alert.ResolvedAt = &time.Time{}
	*alert.ResolvedAt = time.Now()

	updatedAlert, err := s.alertRepo.Update(id, alert)
	if err != nil {
		return nil, err
	}

	// Update vehicle alerts if vehicle repo is available
	if s.vehicleRepo != nil {
		s.updateVehicleAlert(alert.VehicleID, updatedAlert)
	}

	return updatedAlert, nil
}

func (s *AlertService) DismissAlert(id string) error {
	// Check if alert exists
	alert, err := s.alertRepo.FindByID(id)
	if err != nil {
		return errors.New("alert not found")
	}

	// Remove from vehicle alerts if vehicle repo is available
	if s.vehicleRepo != nil {
		s.removeAlertFromVehicle(alert.VehicleID, id)
	}

	return s.alertRepo.Delete(id)
}

func (s *AlertService) DeleteAlert(id string) error {
	// Check if alert exists
	alert, err := s.alertRepo.FindByID(id)
	if err != nil {
		return errors.New("alert not found")
	}

	// Remove from vehicle alerts if vehicle repo is available
	if s.vehicleRepo != nil {
		s.removeAlertFromVehicle(alert.VehicleID, id)
	}

	return s.alertRepo.Delete(id)
}

func (s *AlertService) GetAlertStatistics() (map[string]interface{}, error) {
	allAlerts, err := s.alertRepo.FindAll()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total":      len(allAlerts),
		"resolved":   0,
		"unresolved": 0,
		"by_severity": map[string]int{
			"low":      0,
			"medium":   0,
			"high":     0,
			"critical": 0,
		},
		"by_type": map[string]int{
			"fuel_theft":   0,
			"maintenance":  0,
			"speeding":     0,
			"unauthorized": 0,
			"low_fuel":     0,
		},
	}

	for _, alert := range allAlerts {
		if alert.Resolved {
			stats["resolved"] = stats["resolved"].(int) + 1
		} else {
			stats["unresolved"] = stats["unresolved"].(int) + 1
		}

		// Count by severity
		severityMap := stats["by_severity"].(map[string]int)
		severityMap[alert.Severity]++

		// Count by type
		typeMap := stats["by_type"].(map[string]int)
		typeMap[alert.Type]++
	}

	return stats, nil
}

// Helper methods for vehicle alert synchronization
func (s *AlertService) addAlertToVehicle(vehicleID string, alert *models.Alert) {
	vehicle, err := s.vehicleRepo.FindByID(vehicleID)
	if err != nil {
		return
	}

	// Add alert to vehicle's alerts array
	vehicle.Alerts = append(vehicle.Alerts, *alert)
	s.vehicleRepo.Update(vehicleID, vehicle)
}

func (s *AlertService) updateVehicleAlert(vehicleID string, alert *models.Alert) {
	vehicle, err := s.vehicleRepo.FindByID(vehicleID)
	if err != nil {
		return
	}

	// Update the alert in vehicle's alerts array
	for i, vehicleAlert := range vehicle.Alerts {
		if vehicleAlert.ID.Hex() == alert.ID.Hex() {
			vehicle.Alerts[i] = *alert
			break
		}
	}

	s.vehicleRepo.Update(vehicleID, vehicle)
}

func (s *AlertService) removeAlertFromVehicle(vehicleID string, alertID string) {
	vehicle, err := s.vehicleRepo.FindByID(vehicleID)
	if err != nil {
		return
	}

	// Remove alert from vehicle's alerts array
	for i, alert := range vehicle.Alerts {
		if alert.ID.Hex() == alertID {
			vehicle.Alerts = append(vehicle.Alerts[:i], vehicle.Alerts[i+1:]...)
			break
		}
	}

	s.vehicleRepo.Update(vehicleID, vehicle)
}

// Bulk operations
func (s *AlertService) ResolveAlertsByVehicle(vehicleID string) error {
	alerts, err := s.alertRepo.FindByVehicleID(vehicleID)
	if err != nil {
		return err
	}

	for _, alert := range alerts {
		if !alert.Resolved {
			s.ResolveAlert(alert.ID.Hex())
		}
	}

	return nil
}

func (s *AlertService) ResolveAlertsByType(alertType string) error {
	alerts, err := s.alertRepo.FindByType(alertType)
	if err != nil {
		return err
	}

	for _, alert := range alerts {
		if !alert.Resolved {
			s.ResolveAlert(alert.ID.Hex())
		}
	}

	return nil
}

func (s *AlertService) CleanupOldResolvedAlerts(daysOld int) error {
	cutoffDate := time.Now().AddDate(0, 0, -daysOld)
	return s.alertRepo.DeleteResolvedBefore(cutoffDate)
}