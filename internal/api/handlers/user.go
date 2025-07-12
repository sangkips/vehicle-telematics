package handlers

import (
	"fleet-backend/internal/services"
	"fleet-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	userService *services.UserService
	validator   *validator.Validate
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		validator:   validator.New(),
	}
}

// GetUsers retrieves all users
func (h *UserHandler) GetUsers(c *gin.Context) {
	users, err := h.userService.GetAllUsers()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve users", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Users retrieved successfully", users)
}

// GetUser retrieves a specific user by ID
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "User not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User retrieved successfully", user)
}

// CreateUser creates a new user
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	user, err := h.userService.CreateUser(&req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to create user", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "User created successfully", user)
}

// UpdateUser updates an existing user
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	var req services.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	user, err := h.userService.UpdateUser(userID, &req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to update user", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User updated successfully", user)
}

// DeleteUser deletes a user
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	err := h.userService.DeleteUser(userID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to delete user", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User deleted successfully", nil)
}

// ChangeUserStatus changes a user's status (active/inactive/suspended)
func (h *UserHandler) ChangeUserStatus(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	var req struct {
		Status string `json:"status" validate:"required,oneof=active inactive suspended"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	user, err := h.userService.ChangeUserStatus(userID, req.Status)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to change user status", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User status changed successfully", user)
}

// GetUsersByRole retrieves users by role
func (h *UserHandler) GetUsersByRole(c *gin.Context) {
	role := c.Query("role")
	if role == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Role parameter is required", nil)
		return
	}

	// This would need to be implemented in the user service
	// users, err := h.userService.GetUsersByRole(role)
	// if err != nil {
	//     utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve users", err)
	//     return
	// }

	utils.SuccessResponse(c, http.StatusOK, "Users retrieved successfully", []interface{}{})
}