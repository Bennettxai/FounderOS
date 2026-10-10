// Package handlers provides HTTP handlers for the BusinessOS API.
// This file implements centralized error handling to prevent raw error details
// from leaking to API clients.
package handlers

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Error response types
// ============================================================================

// AppError is a safe, structured error that distinguishes the user-facing
// message from the internal diagnostic detail.
//
//   - Code: machine-readable string (e.g. "INTERNAL_ERROR")
//   - Message: user-safe, never contains stack traces or DB internals
//   - Internal: full diagnostic string, written only to slog — never sent to client
type AppError struct {
	Code     string
	Message  string
	Internal error // logged, never serialized
}

func (e *AppError) Error() string { return e.Message }

// apiErrorBody is the JSON shape returned on every error response.
// It intentionally carries no "details" field to prevent accidental leakage.
type apiErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// ============================================================================
// Gin middleware
// ============================================================================

// ErrorRecoveryMiddleware is a Gin middleware that:
//  1. Recovers from panics and returns a generic 500 (no stack trace to client)
//  2. Logs the panic with full details via slog
//
// Register this as the first middleware on the engine so it wraps all handlers.
func ErrorRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				slog.Error("panic recovered in HTTP handler",
					"panic", r,
					"stack", string(stack),
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
				)
				writeInternalError(c)
				c.Abort()
			}
		}()
		c.Next()
	}
}

// ============================================================================
// Safe response helpers
// ============================================================================
// These functions are the ONLY correct way to return errors from handlers.
// They guarantee that internal error details never reach the client.

// RespondInternalErr logs err via slog and sends a generic 500 to the client.
// The operation parameter is used for the log message (e.g. "list modules").
// Never call c.JSON(500, ...) directly in handler code.
func RespondInternalErr(c *gin.Context, operation string, err error) {
	slog.ErrorContext(c.Request.Context(), "internal error",
		"operation", operation,
		"error", err,
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)
	writeInternalError(c)
}

// RespondBadRequestErr sends a 400 with the user-facing message.
// msg must be safe to expose — do not pass err.Error() here.
func RespondBadRequestErr(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, apiErrorBody{
		Error: msg,
		Code:  "BAD_REQUEST",
	})
}

// RespondNotFoundErr sends a 404 with a resource label, e.g. "module".
func RespondNotFoundErr(c *gin.Context, resource string) {
	c.JSON(http.StatusNotFound, apiErrorBody{
		Error: resource + " not found",
		Code:  "NOT_FOUND",
	})
}

// RespondUnauthorizedErr sends a 401. Pass a safe message like
// "authentication required" or "invalid token".
func RespondUnauthorizedErr(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Authentication required"
	}
	c.JSON(http.StatusUnauthorized, apiErrorBody{
		Error: msg,
		Code:  "UNAUTHORIZED",
	})
}

// RespondForbiddenErr sends a 403 with a safe, user-facing message.
func RespondForbiddenErr(c *gin.Context, msg string) {
	if msg == "" {
		msg = "Access denied"
	}
	c.JSON(http.StatusForbidden, apiErrorBody{
		Error: msg,
		Code:  "FORBIDDEN",
	})
}

// writeInternalError sends the canonical 500 body.
// This is the single source of truth for 500 responses.
func writeInternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, apiErrorBody{
		Error: "Internal server error",
		Code:  "INTERNAL_ERROR",
	})
}
