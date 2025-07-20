package handlers

import (
	"fleet-backend/internal/models"
	"fleet-backend/internal/services"
	"fleet-backend/pkg/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type MaintenanceHandler struct {
	maintenanceService *services.MaintenanceService
	validator          *validator.Validate
}

func NewMaintenanceHandler(maintenanceService *services.MaintenanceService) *MaintenanceHandler {
	return &MaintenanceHandler{
		maintenanceService: maintenanceService,
		validator:          validator.New(),
	}
}

// Maintenance Records
func (h *MaintenanceHandler) CreateMaintenanceRecord(c *gin.Context) {
	var req services.CreateMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	record, err := h.maintenanceService.CreateMaintenanceRecord(&req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to create maintenance record", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Maintenance record created successfully", record)
}

func (h *MaintenanceHandler) GetMaintenanceRecord(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Maintenance record ID is required", nil)
		return
	}

	record, err := h.maintenanceService.GetMaintenanceRecord(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Maintenance record not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Maintenance record retrieved successfully", record)
}

func (h *MaintenanceHandler) GetMaintenanceRecords(c *gin.Context) {
	vehicleID := c.Query("vehicleId")
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	var records []*models.MaintenanceRecord
	if vehicleID != "" {
		records, err = h.maintenanceService.GetMaintenanceRecordsByVehicle(vehicleID)
	} else {
		records, err = h.maintenanceService.GetAllMaintenanceRecords(limit, offset)
	}

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve maintenance records", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Maintenance records retrieved successfully", records)
}

func (h *MaintenanceHandler) UpdateMaintenanceRecord(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Maintenance record ID is required", nil)
		return
	}

	var req services.UpdateMaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	record, err := h.maintenanceService.UpdateMaintenanceRecord(id, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to update maintenance record", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Maintenance record updated successfully", record)
}

func (h *MaintenanceHandler) DeleteMaintenanceRecord(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Maintenance record ID is required", nil)
		return
	}

	err := h.maintenanceService.DeleteMaintenanceRecord(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to delete maintenance record", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Maintenance record deleted successfully", nil)
}

// Maintenance Schedules
func (h *MaintenanceHandler) CreateSchedule(c *gin.Context) {
	var req services.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	schedule, err := h.maintenanceService.CreateSchedule(&req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to create maintenance schedule", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Maintenance schedule created successfully", schedule)
}

func (h *MaintenanceHandler) GetSchedulesByVehicle(c *gin.Context) {
	vehicleID := c.Param("vehicleId")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	schedules, err := h.maintenanceService.GetSchedulesByVehicle(vehicleID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to retrieve maintenance schedules", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Maintenance schedules retrieved successfully", schedules)
}

func (h *MaintenanceHandler) GetUpcomingSchedules(c *gin.Context) {
	daysStr := c.Query("days")
	var days int
	
	if daysStr != "" {
		var err error
		days, err = strconv.Atoi(daysStr)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid days parameter", err)
			return
		}
	} else {
		// If no days parameter provided, get all future schedules (use 0 to indicate no limit)
		days = 0
	}

	schedules, err := h.maintenanceService.GetUpcomingSchedules(days)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve upcoming schedules", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Upcoming schedules retrieved successfully", schedules)
}

func (h *MaintenanceHandler) GetAllSchedules(c *gin.Context) {
	schedules, err := h.maintenanceService.GetAllSchedules()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve schedules", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Schedules retrieved successfully", schedules)
}

func (h *MaintenanceHandler) GetSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Schedule ID is required", nil)
		return
	}

	schedule, err := h.maintenanceService.GetSchedule(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Schedule not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Schedule retrieved successfully", schedule)
}

func (h *MaintenanceHandler) UpdateSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Schedule ID is required", nil)
		return
	}

	var req services.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	schedule, err := h.maintenanceService.UpdateSchedule(id, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to update schedule", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Schedule updated successfully", schedule)
}

func (h *MaintenanceHandler) DeleteSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Schedule ID is required", nil)
		return
	}

	err := h.maintenanceService.DeleteSchedule(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to delete schedule", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Schedule deleted successfully", nil)
}

// Service Reminders
func (h *MaintenanceHandler) GetServiceReminders(c *gin.Context) {
	vehicleID := c.Param("vehicleId")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	reminders, err := h.maintenanceService.GetServiceReminders(vehicleID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to retrieve service reminders", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Service reminders retrieved successfully", reminders)
}

func (h *MaintenanceHandler) GetOverdueReminders(c *gin.Context) {
	reminders, err := h.maintenanceService.GetOverdueReminders()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve overdue reminders", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Overdue reminders retrieved successfully", reminders)
}

func (h *MaintenanceHandler) GetNextServiceDue(c *gin.Context) {
	thresholdStr := c.DefaultQuery("threshold", "2000") // Default 2000km threshold
	threshold, err := strconv.Atoi(thresholdStr)
	if err != nil {
		threshold = 2000
	}

	reminders, err := h.maintenanceService.GetNextServiceDue(threshold)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve vehicles due for service", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Vehicles due for service retrieved successfully", reminders)
}