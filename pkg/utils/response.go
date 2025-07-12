package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// APIResponse represents a standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

// SuccessResponse sends a successful response
func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// ErrorResponse sends an error response
func ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	response := APIResponse{
		Success: false,
		Message: message,
	}

	if err != nil {
		response.Error = err.Error()
	}

	c.JSON(statusCode, response)
}

// ValidationErrorResponse sends a validation error response
func ValidationErrorResponse(c *gin.Context, err error) {
	var errors []string
	
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errors = append(errors, getValidationErrorMessage(fieldError))
		}
	} else {
		errors = append(errors, err.Error())
	}

	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Message: "Validation failed",
		Error:   errors,
	})
}

// getValidationErrorMessage returns a user-friendly validation error message
func getValidationErrorMessage(fieldError validator.FieldError) string {
	field := fieldError.Field()
	tag := fieldError.Tag()

	switch tag {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "min":
		return field + " must be at least " + fieldError.Param() + " characters long"
	case "max":
		return field + " must be at most " + fieldError.Param() + " characters long"
	case "oneof":
		return field + " must be one of: " + fieldError.Param()
	default:
		return field + " is invalid"
	}
}

// PaginationResponse represents a paginated response
type PaginationResponse struct {
	Success    bool        `json:"success"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// PaginatedResponse sends a paginated response
func PaginatedResponse(c *gin.Context, statusCode int, message string, data interface{}, pagination Pagination) {
	c.JSON(statusCode, PaginationResponse{
		Success:    true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	})
}