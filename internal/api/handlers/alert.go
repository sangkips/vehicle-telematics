package handlers

import (
	"fleet-backend/internal/services"
	"fleet-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)


type AlertHandler struct {
	alertService *services.AlertService
	validator    *validator.Validate
}

func NewAlertHandler(alertService *services.AlertService) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
		validator:    validator.New(),
	}
}

// GetAlerts retrieves all alerts
func (h *AlertHandler) GetAlerts(c *gin.Context) {
	alerts, err := h.alertService.GetAllAlerts()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve alerts", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alerts retrieved successfully", alerts)
}

// GetAlert retrieves a specific alert by ID
func (h *AlertHandler) GetAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Alert ID is required", nil)
		return
	}

	alert, err := h.alertService.GetAlertByID(alertID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Alert not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alert retrieved successfully", alert)
}

// CreateAlert creates a new alert
func (h *AlertHandler) CreateAlert(c *gin.Context) {
	var req services.CreateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	alert, err := h.alertService.CreateAlert(&req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to create alert", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Alert created successfully", alert)
}

// UpdateAlert updates an existing alert
func (h *AlertHandler) UpdateAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Alert ID is required", nil)
		return
	}

	var req services.UpdateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	alert, err := h.alertService.UpdateAlert(alertID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to update alert", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alert updated successfully", alert)
}

// ResolveAlert marks an alert as resolved
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Alert ID is required", nil)
		return
	}

	alert, err := h.alertService.ResolveAlert(alertID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to resolve alert", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alert resolved successfully", alert)
}

// DismissAlert dismisses (deletes) an alert
func (h *AlertHandler) DismissAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Alert ID is required", nil)
		return
	}

	err := h.alertService.DismissAlert(alertID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to dismiss alert", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alert dismissed successfully", nil)
}

// GetAlertsByVehicle retrieves alerts for a specific vehicle
func (h *AlertHandler) GetAlertsByVehicle(c *gin.Context) {
	vehicleID := c.Param("vehicleId")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	alerts, err := h.alertService.GetAlertsByVehicle(vehicleID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve alerts", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alerts retrieved successfully", alerts)
}

// GetAlertsByType retrieves alerts by type
func (h *AlertHandler) GetAlertsByType(c *gin.Context) {
	alertType := c.Query("type")
	if alertType == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Alert type parameter is required", nil)
		return
	}

	alerts, err := h.alertService.GetAlertsByType(alertType)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve alerts", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alerts retrieved successfully", alerts)
}

// GetAlertsBySeverity retrieves alerts by severity
func (h *AlertHandler) GetAlertsBySeverity(c *gin.Context) {
	severity := c.Query("severity")
	if severity == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Severity parameter is required", nil)
		return
	}

	alerts, err := h.alertService.GetAlertsBySeverity(severity)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve alerts", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alerts retrieved successfully", alerts)
}

// GetUnresolvedAlerts retrieves all unresolved alerts
func (h *AlertHandler) GetUnresolvedAlerts(c *gin.Context) {
	alerts, err := h.alertService.GetUnresolvedAlerts()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve unresolved alerts", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Unresolved alerts retrieved successfully", alerts)
}

// GetAlertStatistics retrieves alert statistics
func (h *AlertHandler) GetAlertStatistics(c *gin.Context) {
	stats, err := h.alertService.GetAlertStatistics()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve alert statistics", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Alert statistics retrieved successfully", stats)
}

// ResolveAlertsByVehicle resolves all alerts for a specific vehicle
func (h *AlertHandler) ResolveAlertsByVehicle(c *gin.Context) {
	vehicleID := c.Param("vehicleId")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	err := h.alertService.ResolveAlertsByVehicle(vehicleID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to resolve alerts", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "All vehicle alerts resolved successfully", nil)
}

// ResolveAlertsByType resolves all alerts of a specific type
func (h *AlertHandler) ResolveAlertsByType(c *gin.Context) {
	alertType := c.Query("type")
	if alertType == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Alert type parameter is required", nil)
		return
	}

	err := h.alertService.ResolveAlertsByType(alertType)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to resolve alerts", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "All alerts of type resolved successfully", nil)
}
