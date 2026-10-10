package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// =============================================================================
// MOBILE ERROR RESPONSE STANDARDIZATION
// =============================================================================
// All mobile API errors follow a consistent format:
//
//	{
//	    "error": {
//	        "code": "VALIDATION_ERROR",
//	        "message": "Invalid due_date format",
//	        "details": {
//	            "field": "due_date",
//	            "expected": "ISO 8601 date string"
//	        }
//	    }
//	}
//
// This makes it easy for mobile clients to:
// 1. Check for errors consistently (response.error != null)
// 2. Display user-friendly messages (error.message)
// 3. Handle specific error types (switch on error.code)
// 4. Show field-specific validation errors (error.details.field)
// =============================================================================

// MobileError represents a standardized error response
type MobileError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// MobileErrorResponse wraps the error in an "error" key
type MobileErrorResponse struct {
	Error MobileError `json:"error"`
}

// =============================================================================
// ERROR CODES
// =============================================================================
// These codes are documented in MOBILE_API.md and should be consistent
// across all mobile endpoints. Clients use these for programmatic handling.
// =============================================================================

const (
	// 400 Bad Request
	ErrCodeValidation    = "VALIDATION_ERROR"
	ErrCodeInvalidCursor = "INVALID_CURSOR"

	// 401 Unauthorized
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeTokenExpired = "TOKEN_EXPIRED"

	// 403 Forbidden
	ErrCodeForbidden       = "FORBIDDEN"
	ErrCodeWorkspaceAccess = "WORKSPACE_ACCESS_DENIED"

	// 404 Not Found
	ErrCodeNotFound = "NOT_FOUND"

	// 409 Conflict
	ErrCodeConflict = "CONFLICT"

	// 422 Unprocessable Entity
	ErrCodeCaptureFailed = "CAPTURE_FAILED"

	// 429 Rate Limited
	ErrCodeRateLimited = "RATE_LIMITED"

	// 500 Internal Server Error
	ErrCodeInternal = "INTERNAL_ERROR"

	// 503 Service Unavailable
	ErrCodeUnavailable = "SERVICE_UNAVAILABLE"
)

// =============================================================================
// ERROR RESPONSE HELPERS
// =============================================================================
// These functions make it easy to return consistent error responses.
// Usage: MobileRespondError(c, 400, ErrCodeValidation, "Invalid input")
// =============================================================================

// MobileRespondError sends a standardized error response
func MobileRespondError(c *gin.Context, status int, code string, message string) {
	c.JSON(status, MobileErrorResponse{
		Error: MobileError{
			Code:    code,
			Message: message,
		},
	})
}

// MobileRespondErrorWithDetails sends an error with additional details
func MobileRespondErrorWithDetails(c *gin.Context, status int, code string, message string, details map[string]interface{}) {
	c.JSON(status, MobileErrorResponse{
		Error: MobileError{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// =============================================================================
// COMMON ERROR RESPONSES
// =============================================================================
// Pre-built helpers for the most common error cases
// =============================================================================

// MobileRespondValidationError returns a 400 with field-specific validation details
// Example: MobileRespondValidationError(c, "due_date", "ISO 8601 date string")
func MobileRespondValidationError(c *gin.Context, field string, expected string) {
	MobileRespondErrorWithDetails(c, http.StatusBadRequest, ErrCodeValidation,
		"Invalid "+field+" format",
		map[string]interface{}{
			"field":    field,
			"expected": expected,
		},
	)
}

// MobileRespondNotFound returns a 404 for missing resources
// Example: MobileRespondNotFound(c, "task")
func MobileRespondNotFound(c *gin.Context, resource string) {
	MobileRespondError(c, http.StatusNotFound, ErrCodeNotFound,
		resource+" not found",
	)
}

// MobileRespondUnauthorized returns a 401 for auth failures
func MobileRespondUnauthorized(c *gin.Context) {
	MobileRespondError(c, http.StatusUnauthorized, ErrCodeUnauthorized,
		"Authentication required",
	)
}

// MobileRespondForbidden returns a 403 for permission failures
func MobileRespondForbidden(c *gin.Context, reason string) {
	MobileRespondError(c, http.StatusForbidden, ErrCodeForbidden, reason)
}

// MobileRespondInvalidCursor returns a 400 for bad pagination cursors
func MobileRespondInvalidCursor(c *gin.Context) {
	MobileRespondError(c, http.StatusBadRequest, ErrCodeInvalidCursor,
		"Invalid pagination cursor",
	)
}

// MobileRespondRateLimited returns a 429 with retry information
func MobileRespondRateLimited(c *gin.Context, retryAfterSeconds int) {
	c.Header("Retry-After", string(rune(retryAfterSeconds)))
	MobileRespondErrorWithDetails(c, http.StatusTooManyRequests, ErrCodeRateLimited,
		"Too many requests",
		map[string]interface{}{
			"retry_after": retryAfterSeconds,
		},
	)
}

// MobileRespondInternalError returns a 500 for server errors
// Note: Don't expose internal details to clients
func MobileRespondInternalError(c *gin.Context) {
	MobileRespondError(c, http.StatusInternalServerError, ErrCodeInternal,
		"An unexpected error occurred",
	)
}

// MobileRespondConflict returns a 409 for version conflicts
func MobileRespondConflict(c *gin.Context, message string) {
	MobileRespondError(c, http.StatusConflict, ErrCodeConflict, message)
}
