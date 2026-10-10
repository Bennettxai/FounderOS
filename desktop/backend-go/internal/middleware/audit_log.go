package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditLogger returns middleware that logs all requests for compliance auditing.
// It logs to structured slog output which can be ingested by any log aggregator.
func AuditLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Capture user ID before the request runs (user may not be set yet)
		c.Next()

		// Resolve user ID after auth middleware has run
		userID := "anonymous"
		if user := GetCurrentUser(c); user != nil {
			userID = user.ID
		}

		slog.Info("AUDIT",
			"timestamp", start.UTC().Format(time.RFC3339),
			"user_id", userID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
			"latency_ms", time.Since(start).Milliseconds(),
		)
	}
}

// AuditSensitiveAccess logs access to sensitive data endpoints with extra detail.
// Apply this to route groups that serve PII or confidential data (e.g. memories, conversations).
func AuditSensitiveAccess(resourceType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		userID := "anonymous"
		if user := GetCurrentUser(c); user != nil {
			userID = user.ID
		}

		// Best-effort resource ID resolution across common param names
		resourceID := c.Param("id")
		if resourceID == "" {
			resourceID = c.Param("workspace_id")
		}

		slog.Warn("AUDIT_SENSITIVE_ACCESS",
			"user_id", userID,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"ip", c.ClientIP(),
			"timestamp", time.Now().UTC().Format(time.RFC3339),
		)
	}
}
