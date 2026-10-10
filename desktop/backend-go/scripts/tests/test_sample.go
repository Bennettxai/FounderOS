//go:build ignore

// Test sample file for migrate_errors.go testing
// This file contains various error patterns that should be migrated
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// TestHandler demonstrates old error patterns
type TestHandler struct{}

func (h *TestHandler) ExampleUnauthorized(c *gin.Context) {
	// Should become: utils.RespondUnauthorized(c, slog.Default())
	c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
	return
}

func (h *TestHandler) ExampleBadRequest(c *gin.Context) {
	// Should become: utils.RespondBadRequest(c, slog.Default(), "Invalid request data")
	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
	return
}

func (h *TestHandler) ExampleInvalidID(c *gin.Context) {
	// Should become: utils.RespondInvalidID(c, slog.Default(), "user ID")
	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
	return
}

func (h *TestHandler) ExampleNotFound(c *gin.Context) {
	// Should become: utils.RespondNotFound(c, slog.Default(), "User")
	c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
	return
}

func (h *TestHandler) ExampleInternalError(c *gin.Context) {
	// Should become: utils.RespondInternalError(c, slog.Default(), "fetch user data", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
	return
}

func (h *TestHandler) ExampleInvalidRequest(c *gin.Context) {
	var req struct{}
	err := c.ShouldBindJSON(&req)
	if err != nil {
		// Should become: utils.RespondInvalidRequest(c, slog.Default(), err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}

func (h *TestHandler) ExampleForbidden(c *gin.Context) {
	// Should become: utils.RespondForbidden(c, slog.Default(), "Access denied")
	c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
	return
}

func (h *TestHandler) ExampleConflict(c *gin.Context) {
	// Should become: utils.RespondConflict(c, slog.Default(), "User already exists")
	c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
	return
}

func (h *TestHandler) ExampleServiceUnavailable(c *gin.Context) {
	// Should become: utils.RespondServiceUnavailable(c, slog.Default(), "Database")
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database is temporarily unavailable"})
	return
}

func (h *TestHandler) ExampleNotImplemented(c *gin.Context) {
	// Should become: utils.RespondNotImplemented(c, slog.Default(), "Feature")
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Feature is not implemented"})
	return
}

func (h *TestHandler) ExampleTooManyRequests(c *gin.Context) {
	// Should become: utils.RespondTooManyRequests(c, slog.Default(), "API")
	c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests to API"})
	return
}
