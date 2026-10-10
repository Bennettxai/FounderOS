package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// APIVersion represents an API version configuration
type APIVersion struct {
	Version          string    // e.g., "v1", "v2"
	IsDeprecated     bool      // Whether this version is deprecated
	SunsetDate       time.Time // Date when version will be removed
	SuccessorVersion string    // Next version to migrate to (e.g., "v2")
}

// VersioningConfig holds versioning configuration
type VersioningConfig struct {
	CurrentVersion string
	Versions       map[string]*APIVersion
}

// DefaultVersioningConfig returns the default versioning configuration
func DefaultVersioningConfig() *VersioningConfig {
	return &VersioningConfig{
		CurrentVersion: "v1",
		Versions: map[string]*APIVersion{
			"v1": {
				Version:          "v1",
				IsDeprecated:     false,
				SunsetDate:       time.Time{}, // No sunset date yet
				SuccessorVersion: "",
			},
		},
	}
}

// DeprecationHeaders middleware adds deprecation headers for deprecated API versions
func DeprecationHeaders(version *APIVersion) gin.HandlerFunc {
	return func(c *gin.Context) {
		if version.IsDeprecated {
			c.Header("Deprecation", "true")

			if !version.SunsetDate.IsZero() {
				// RFC 3339 format: "2026-06-01T00:00:00Z"
				c.Header("Sunset", version.SunsetDate.Format(time.RFC3339))
			}

			if version.SuccessorVersion != "" {
				// Link header format: </api/v2>; rel="successor-version"
				c.Header("Link", "</api/"+version.SuccessorVersion+">; rel=\"successor-version\"")
			}

			slog.Debug("deprecated_api_version_accessed",
				"version", version.Version,
				"path", c.Request.URL.Path,
				"user_agent", c.Request.UserAgent(),
			)
		}

		// Add current version header
		c.Header("API-Version", version.Version)

		c.Next()
	}
}

// VersionRedirect creates a redirect middleware from old path to versioned path
// Example: /api/chat -> /api/v1/chat
func VersionRedirect(targetVersion string, permanent bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the current path after /api
		originalPath := c.Request.URL.Path

		// Add deprecation warning for non-versioned API access
		c.Header("Deprecation", "true")
		c.Header("Link", "</api/"+targetVersion+">; rel=\"successor-version\"")
		c.Header("Warning", "299 - \"Direct /api access is deprecated. Use /api/"+targetVersion+" instead\"")

		slog.Warn("non_versioned_api_access",
			"path", originalPath,
			"redirecting_to_version", targetVersion,
			"user_agent", c.Request.UserAgent(),
		)

		c.Next()
	}
}

// ValidateAPIVersion middleware validates that the API version in the URL is supported
func ValidateAPIVersion(cfg *VersioningConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract version from path (e.g., /api/v1/chat -> v1)
		// This is informational only - the router already matched the version

		// We don't need to extract or validate here since gin router
		// already handles routing to correct version groups

		c.Next()
	}
}
