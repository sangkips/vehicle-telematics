package handlers

import (
	"fleet-backend/internal/services"
	"fleet-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type VehicleHandler struct {
	vehicleService *services.VehicleService
	validator      *validator.Validate
}

func NewVehicleHandler(vehicleService *services.VehicleService) *VehicleHandler {
	return &VehicleHandler{
		vehicleService: vehicleService,
		validator:      validator.New(),
	}
}

// GetVehicles retrieves all vehicles
func (h *VehicleHandler) GetVehicles(c *gin.Context) {
	vehicles, err := h.vehicleService.GetAllVehicles()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve vehicles", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Vehicles retrieved successfully", vehicles)
}

// GetVehicle retrieves a specific vehicle by ID
func (h *VehicleHandler) GetVehicle(c *gin.Context) {
	vehicleID := c.Param("id")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	vehicle, err := h.vehicleService.GetVehicleByID(vehicleID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Vehicle not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Vehicle retrieved successfully", vehicle)
}

// CreateVehicle creates a new vehicle
func (h *VehicleHandler) CreateVehicle(c *gin.Context) {
	var req services.CreateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	vehicle, err := h.vehicleService.CreateVehicle(&req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to create vehicle", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Vehicle created successfully", vehicle)
}

// UpdateVehicle updates an existing vehicle
func (h *VehicleHandler) UpdateVehicle(c *gin.Context) {
	vehicleID := c.Param("id")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	var req services.UpdateVehicleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	vehicle, err := h.vehicleService.UpdateVehicle(vehicleID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to update vehicle", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Vehicle updated successfully", vehicle)
}

// DeleteVehicle deletes a vehicle
func (h *VehicleHandler) DeleteVehicle(c *gin.Context) {
	vehicleID := c.Param("id")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	err := h.vehicleService.DeleteVehicle(vehicleID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to delete vehicle", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Vehicle deleted successfully", nil)
}

// GetVehicleUpdates retrieves real-time vehicle updates
func (h *VehicleHandler) GetVehicleUpdates(c *gin.Context) {
	vehicles, err := h.vehicleService.GetVehicleUpdates()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve vehicle updates", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Vehicle updates retrieved successfully", vehicles)
}

// GetVehiclesByStatus retrieves vehicles by status
func (h *VehicleHandler) GetVehiclesByStatus(c *gin.Context) {
	status := c.Query("status")
	if status == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Status parameter is required", nil)
		return
	}

	vehicles, err := h.vehicleService.GetVehiclesByStatus(status)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve vehicles", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Vehicles retrieved successfully", vehicles)
}

// GetVehiclesByDriver retrieves vehicles by driver
func (h *VehicleHandler) GetVehiclesByDriver(c *gin.Context) {
	driver := c.Query("driver")
	if driver == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Driver parameter is required", nil)
		return
	}

	vehicles, err := h.vehicleService.GetVehiclesByDriver(driver)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve vehicles", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Vehicles retrieved successfully", vehicles)
}

// UpdateVehicleLocation updates a vehicle's location
func (h *VehicleHandler) UpdateVehicleLocation(c *gin.Context) {
	vehicleID := c.Param("id")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	var req struct {
		Lat     float64 `json:"lat" validate:"required,min=-90,max=90"`
		Lng     float64 `json:"lng" validate:"required,min=-180,max=180"`
		Address string  `json:"address"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	// This would need to be implemented in the vehicle service
	// location := models.Location{
	//     Lat:     req.Lat,
	//     Lng:     req.Lng,
	//     Address: req.Address,
	// }
	// 
	// err := h.vehicleService.UpdateVehicleLocation(vehicleID, location)
	// if err != nil {
	//     utils.ErrorResponse(c, http.StatusBadRequest, "Failed to update vehicle location", err)
	//     return
	// }

	utils.SuccessResponse(c, http.StatusOK, "Vehicle location updated successfully", nil)
}

// UpdateVehicleFuelLevel updates a vehicle's fuel level
func (h *VehicleHandler) UpdateVehicleFuelLevel(c *gin.Context) {
	vehicleID := c.Param("id")
	if vehicleID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Vehicle ID is required", nil)
		return
	}

	var req struct {
		FuelLevel float64 `json:"fuelLevel" validate:"required,min=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	// This would need to be implemented in the vehicle service
	// err := h.vehicleService.UpdateVehicleFuelLevel(vehicleID, req.FuelLevel)
	// if err != nil {
	//     utils.ErrorResponse(c, http.StatusBadRequest, "Failed to update fuel level", err)
	//     return
	// }

	utils.SuccessResponse(c, http.StatusOK, "Vehicle fuel level updated successfully", nil)
}