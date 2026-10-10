package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a standardized API error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Standard error codes
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeValidation          = "VALIDATION_ERROR"
	ErrCodeInternalServer      = "INTERNAL_SERVER_ERROR"
	ErrCodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
	ErrCodeDatabaseError       = "DATABASE_ERROR"
	ErrCodeInvalidInput        = "INVALID_INPUT"
)

// RespondWithError sends a standardized error response
func RespondWithError(c *gin.Context, statusCode int, errorMessage string) {
	code := getErrorCode(statusCode)
	c.JSON(statusCode, ErrorResponse{
		Error: errorMessage,
		Code:  code,
	})
}

// RespondWithDetailedError sends a detailed error response with custom code and details
func RespondWithDetailedError(c *gin.Context, statusCode int, errorMessage, customCode string, details map[string]interface{}) {
	code := customCode
	if code == "" {
		code = getErrorCode(statusCode)
	}

	c.JSON(statusCode, ErrorResponse{
		Error:   errorMessage,
		Code:    code,
		Details: details,
	})
}

// RespondWithValidationError sends a validation error response
func RespondWithValidationError(c *gin.Context, fieldErrors map[string]string) {
	details := make(map[string]interface{})
	for field, err := range fieldErrors {
		details[field] = err
	}

	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error:   "Validation failed",
		Code:    ErrCodeValidation,
		Details: details,
	})
}

// RespondForbidden sends a 403 forbidden response
func RespondForbidden(c *gin.Context, message string) {
	if message == "" {
		message = "Forbidden"
	}
	c.JSON(http.StatusForbidden, ErrorResponse{
		Error: message,
		Code:  ErrCodeForbidden,
	})
}

// RespondNotFound sends a 404 not found response
func RespondNotFound(c *gin.Context, resource string) {
	message := "Not found"
	if resource != "" {
		message = resource + " not found"
	}
	c.JSON(http.StatusNotFound, ErrorResponse{
		Error: message,
		Code:  ErrCodeNotFound,
	})
}

// RespondBadRequest sends a 400 bad request response
func RespondBadRequest(c *gin.Context, message string) {
	if message == "" {
		message = "Bad request"
	}
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Error: message,
		Code:  ErrCodeBadRequest,
	})
}

// RespondInternalError sends a 500 internal server error response
func RespondInternalError(c *gin.Context, message string) {
	if message == "" {
		message = "Internal server error"
	}
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: message,
		Code:  ErrCodeInternalServer,
	})
}

// RespondDatabaseError sends a database error response
func RespondDatabaseError(c *gin.Context, operation string) {
	message := "Database error"
	if operation != "" {
		message = "Database error during " + operation
	}
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: message,
		Code:  ErrCodeDatabaseError,
	})
}

// getErrorCode maps HTTP status codes to error codes
func getErrorCode(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return ErrCodeBadRequest
	case http.StatusUnauthorized:
		return ErrCodeUnauthorized
	case http.StatusForbidden:
		return ErrCodeForbidden
	case http.StatusNotFound:
		return ErrCodeNotFound
	case http.StatusConflict:
		return ErrCodeConflict
	case http.StatusInternalServerError:
		return ErrCodeInternalServer
	case http.StatusServiceUnavailable:
		return ErrCodeServiceUnavailable
	default:
		return ErrCodeInternalServer
	}
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    interface{}            `json:"data,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// RespondSuccess sends a standardized success response
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// RespondCreated sends a 201 created response
func RespondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// RespondWithMessage sends a success response with a message
func RespondWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// RespondNoContent sends a 204 no content response
func RespondNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}
