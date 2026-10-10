package utils

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	ErrCodeValidation         ErrorCode = "VALIDATION_ERROR"
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrCodeConflict           ErrorCode = "CONFLICT"
	ErrCodeRateLimited        ErrorCode = "RATE_LIMITED"
	ErrCodeInternal           ErrorCode = "INTERNAL_ERROR"
	ErrCodeBadRequest         ErrorCode = "BAD_REQUEST"
	ErrCodeUnprocessable      ErrorCode = "UNPROCESSABLE_ENTITY"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeNotImplemented     ErrorCode = "NOT_IMPLEMENTED"
	ErrCodeTooManyRequests    ErrorCode = "TOO_MANY_REQUESTS"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// ErrorBuilder provides a fluent API for building error responses
type ErrorBuilder struct {
	statusCode int
	code       ErrorCode
	message    string
	details    map[string]interface{}
	logger     *slog.Logger
}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder(logger *slog.Logger) *ErrorBuilder {
	return &ErrorBuilder{
		logger:  logger,
		details: make(map[string]interface{}),
	}
}

// Unauthorized creates a 401 error builder
func Unauthorized(logger *slog.Logger, message string) *ErrorBuilder {
	if message == "" {
		message = "Not authenticated"
	}
	return &ErrorBuilder{
		statusCode: http.StatusUnauthorized,
		code:       ErrCodeUnauthorized,
		message:    message,
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// BadRequest creates a 400 error builder
func BadRequest(logger *slog.Logger, message string) *ErrorBuilder {
	return &ErrorBuilder{
		statusCode: http.StatusBadRequest,
		code:       ErrCodeBadRequest,
		message:    message,
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// NotFound creates a 404 error builder
func NotFound(logger *slog.Logger, resource string) *ErrorBuilder {
	return &ErrorBuilder{
		statusCode: http.StatusNotFound,
		code:       ErrCodeNotFound,
		message:    resource + " not found",
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// InternalError creates a 500 error builder
func InternalError(logger *slog.Logger, message string) *ErrorBuilder {
	return &ErrorBuilder{
		statusCode: http.StatusInternalServerError,
		code:       ErrCodeInternal,
		message:    message,
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// ValidationError creates a validation error builder
func ValidationError(logger *slog.Logger, field string) *ErrorBuilder {
	return &ErrorBuilder{
		statusCode: http.StatusBadRequest,
		code:       ErrCodeValidation,
		message:    "Validation failed",
		details:    map[string]interface{}{"field": field},
		logger:     logger,
	}
}

// Forbidden creates a 403 error builder
func Forbidden(logger *slog.Logger, message string) *ErrorBuilder {
	if message == "" {
		message = "Access denied"
	}
	return &ErrorBuilder{
		statusCode: http.StatusForbidden,
		code:       ErrCodeForbidden,
		message:    message,
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// Conflict creates a 409 error builder
func Conflict(logger *slog.Logger, message string) *ErrorBuilder {
	return &ErrorBuilder{
		statusCode: http.StatusConflict,
		code:       ErrCodeConflict,
		message:    message,
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// ServiceUnavailable creates a 503 error builder
func ServiceUnavailable(logger *slog.Logger, service string) *ErrorBuilder {
	return &ErrorBuilder{
		statusCode: http.StatusServiceUnavailable,
		code:       ErrCodeServiceUnavailable,
		message:    service + " is temporarily unavailable",
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// NotImplemented creates a 501 error builder
func NotImplemented(logger *slog.Logger, feature string) *ErrorBuilder {
	return &ErrorBuilder{
		statusCode: http.StatusNotImplemented,
		code:       ErrCodeNotImplemented,
		message:    feature + " is not implemented",
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// TooManyRequests creates a 429 error builder
func TooManyRequests(logger *slog.Logger, resource string) *ErrorBuilder {
	return &ErrorBuilder{
		statusCode: http.StatusTooManyRequests,
		code:       ErrCodeTooManyRequests,
		message:    "Too many requests to " + resource,
		details:    make(map[string]interface{}),
		logger:     logger,
	}
}

// WithDetails adds a detail field to the error
func (b *ErrorBuilder) WithDetails(key string, value interface{}) *ErrorBuilder {
	b.details[key] = value
	return b
}

// WithError adds the error message to details
func (b *ErrorBuilder) WithError(err error) *ErrorBuilder {
	if err != nil {
		b.details["error"] = err.Error()
	}
	return b
}

// WithMessage overrides the default message
func (b *ErrorBuilder) WithMessage(msg string) *ErrorBuilder {
	b.message = msg
	return b
}

// Respond sends the error response and logs it
func (b *ErrorBuilder) Respond(c *gin.Context) {
	// Log with structured fields
	if b.logger != nil {
		logAttrs := []any{
			"status", b.statusCode,
			"code", b.code,
			"message", b.message,
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
		}

		if userID, exists := c.Get("user_id"); exists {
			logAttrs = append(logAttrs, "user_id", userID)
		}

		if len(b.details) > 0 {
			logAttrs = append(logAttrs, "details", b.details)
		}

		b.logger.Error("API error", logAttrs...)
	}

	// Send response
	response := ErrorResponse{
		Error: ErrorDetail{
			Code:    b.code,
			Message: b.message,
			Details: b.details,
		},
	}

	c.JSON(b.statusCode, response)
}

// =====================================================
// Simple helper functions for the most common cases
// =====================================================

// RespondUnauthorized sends a 401 Unauthorized response
func RespondUnauthorized(c *gin.Context, logger *slog.Logger) {
	Unauthorized(logger, "Not authenticated").Respond(c)
}

// RespondInvalidID sends a 400 Bad Request for invalid ID
func RespondInvalidID(c *gin.Context, logger *slog.Logger, field string) {
	ValidationError(logger, field).
		WithMessage("Invalid " + field).
		Respond(c)
}

// RespondNotFound sends a 404 Not Found response
func RespondNotFound(c *gin.Context, logger *slog.Logger, resource string) {
	NotFound(logger, resource).Respond(c)
}

// RespondInternalError sends a 500 Internal Server Error
func RespondInternalError(c *gin.Context, logger *slog.Logger, operation string, err error) {
	InternalError(logger, "Failed to "+operation).
		WithError(err).
		Respond(c)
}

// RespondBadRequest sends a 400 Bad Request response
func RespondBadRequest(c *gin.Context, logger *slog.Logger, message string) {
	BadRequest(logger, message).Respond(c)
}

// RespondForbidden sends a 403 Forbidden response
func RespondForbidden(c *gin.Context, logger *slog.Logger, message string) {
	Forbidden(logger, message).Respond(c)
}

// RespondConflict sends a 409 Conflict response
func RespondConflict(c *gin.Context, logger *slog.Logger, message string) {
	Conflict(logger, message).Respond(c)
}

// RespondInvalidRequest sends a 400 for invalid request body
func RespondInvalidRequest(c *gin.Context, logger *slog.Logger, err error) {
	BadRequest(logger, "Invalid request body").
		WithError(err).
		Respond(c)
}

// RespondServiceUnavailable sends a 503 Service Unavailable response
func RespondServiceUnavailable(c *gin.Context, logger *slog.Logger, service string) {
	ServiceUnavailable(logger, service).Respond(c)
}

// RespondNotImplemented sends a 501 Not Implemented response
func RespondNotImplemented(c *gin.Context, logger *slog.Logger, feature string) {
	NotImplemented(logger, feature).Respond(c)
}

// RespondTooManyRequests sends a 429 Too Many Requests response
func RespondTooManyRequests(c *gin.Context, logger *slog.Logger, resource string) {
	TooManyRequests(logger, resource).Respond(c)
}
