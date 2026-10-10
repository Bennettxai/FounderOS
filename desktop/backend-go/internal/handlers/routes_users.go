package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// registerUserRoutes wires up user profile, username, and MCP routes:
// /api/profile, /api/account, /api/users, /api/mcp.
func (h *Handlers) registerUserRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	// Profile + Account management routes - /api/profile and /api/account
	RegisterProfileRoutes(api, NewProfileHandler(h.pool, h.sessionCache), auth)

	// Username routes - /api/users
	usernameHandler := NewUsernameHandler(h.pool)
	users := api.Group("/users")
	{
		// Public route - check username availability (no auth required for UX)
		users.GET("/check-username/:username", usernameHandler.CheckUsernameAvailability)

		// Protected routes
		users.GET("/me", auth, usernameHandler.GetCurrentUser)
		users.PATCH("/me/username", auth, usernameHandler.SetUsername)
		users.POST("/me/complete-onboarding", auth, usernameHandler.CompleteOnboarding)
	}

	// MCP routes - /api/mcp
	mcpH := NewMCPHandler(h.pool)
	mcp := api.Group("/mcp")
	mcp.Use(auth, middleware.RequireAuth())
	{
		mcp.GET("/tools", mcpH.ListMCPTools)
		mcp.POST("/execute", mcpH.ExecuteMCPTool)
		mcp.GET("/health", mcpH.MCPHealth)
	}
}
