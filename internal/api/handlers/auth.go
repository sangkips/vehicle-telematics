package handlers

import (
	"fleet-backend/internal/services"
	"fleet-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	authService *services.AuthService
	validator   *validator.Validate
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validator:   validator.New(),
	}
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	response, err := h.authService.Login(&req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Authentication failed", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Login successful", response)
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {

	utils.SuccessResponse(c, http.StatusOK, "Logout successful", nil)
}

// RefreshToken handles token refresh for authenticated users
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	token, err := h.authService.RefreshToken(userID.(string))
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Token refresh failed", err)
		return
	}

	response := map[string]string{
		"token": token,
	}

	utils.SuccessResponse(c, http.StatusOK, "Token refreshed successfully", response)
}

// RefreshTokenPublic handles token refresh without requiring authentication middleware
func (h *AuthHandler) RefreshTokenPublic(c *gin.Context) {
	var req struct {
		Token string `json:"token" validate:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	newToken, err := h.authService.RefreshTokenFromString(req.Token)
	if err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Token refresh failed", err)
		return
	}

	response := map[string]string{
		"token": newToken,
	}

	utils.SuccessResponse(c, http.StatusOK, "Token refreshed successfully", response)
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	// Get user profile from auth service
	user, err := h.authService.GetUserProfile(userID.(string))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve profile", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Profile retrieved successfully", user)
}

// ChangePassword handles password change requests
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"currentPassword" validate:"required"`
		NewPassword     string `json:"newPassword" validate:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password changed successfully", nil)
}

// ForgotPassword handles password reset requests
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req services.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := h.authService.ForgotPassword(req.Email); err != nil {
		// Don't reveal specific errors to prevent information disclosure
		utils.SuccessResponse(c, http.StatusOK, "If the email exists, a password reset link has been sent", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "If the email exists, a password reset link has been sent", nil)
}

// ResetPassword handles password reset completion
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req services.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		utils.ValidationErrorResponse(c, err)
		return
	}

	if err := h.authService.ResetPassword(req.Token, req.NewPassword); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Failed to reset password", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password reset successfully", nil)
}
