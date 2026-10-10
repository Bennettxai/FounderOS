// Package middleware provides HTTP middleware for the BusinessOS API server.
package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/services"
)

// PlanEnforcementMiddleware checks whether the incoming request is permitted
// by the workspace's current plan.
//
// It gates:
//   - AI/chat endpoints  → daily AI-call limit
//   - Module endpoints   → max module count
//   - Team endpoints     → max team-member count
//
// When a limit is exceeded the handler is aborted with HTTP 429 and a JSON
// body that includes the upgrade prompt.  The workspace_id must already be
// present in the Gin context (set by WorkspaceMiddleware or similar).
func PlanEnforcementMiddleware(usageService *services.UsageMeteringService) gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceID := GetWorkspaceID(c)
		if workspaceID == nil {
			// No workspace context — let the handler decide how to handle this.
			c.Next()
			return
		}

		path := c.FullPath()

		switch {
		case isAIEndpoint(path):
			allowed, remaining, err := usageService.CheckAILimit(c.Request.Context(), *workspaceID)
			if err != nil {
				slog.ErrorContext(c.Request.Context(), "plan_enforcement: ai limit check failed",
					"workspace_id", workspaceID, "error", err)
				// Fail open: do not block on internal errors.
				c.Next()
				return
			}
			if !allowed {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":     "Daily AI call limit reached for your plan.",
					"code":      "ai_limit_exceeded",
					"remaining": remaining,
					"action":    "Upgrade your plan to continue using AI features today.",
				})
				return
			}

		case isModuleEndpoint(path):
			if c.Request.Method != http.MethodPost {
				c.Next()
				return
			}
			summary, err := usageService.GetUsageSummary(c.Request.Context(), *workspaceID)
			if err != nil {
				slog.ErrorContext(c.Request.Context(), "plan_enforcement: module count check failed",
					"workspace_id", workspaceID, "error", err)
				c.Next()
				return
			}
			limit := summary.ModulesLimit
			if limit != -1 && summary.ModulesCount >= limit {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":   "Module limit reached for your plan.",
					"code":    "module_limit_exceeded",
					"current": summary.ModulesCount,
					"limit":   limit,
					"action":  "Upgrade your plan to create additional modules.",
				})
				return
			}

		case isTeamEndpoint(path):
			if c.Request.Method != http.MethodPost {
				c.Next()
				return
			}
			summary, err := usageService.GetUsageSummary(c.Request.Context(), *workspaceID)
			if err != nil {
				slog.ErrorContext(c.Request.Context(), "plan_enforcement: team member check failed",
					"workspace_id", workspaceID, "error", err)
				c.Next()
				return
			}
			limit := summary.TeamMembersLimit
			if limit != -1 && summary.TeamMembers >= limit {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":   "Team member limit reached for your plan.",
					"code":    "team_limit_exceeded",
					"current": summary.TeamMembers,
					"limit":   limit,
					"action":  "Upgrade your plan to invite more team members.",
				})
				return
			}
		}

		c.Next()
	}
}

// AIRateLimitMiddleware is a focused middleware for AI/chat endpoints only.
// It records each successful pass as an AI call and enforces the daily limit.
// Use this instead of PlanEnforcementMiddleware when you need per-endpoint
// granularity (e.g., on the /api/chat route group).
func AIRateLimitMiddleware(usageService *services.UsageMeteringService) gin.HandlerFunc {
	return func(c *gin.Context) {
		workspaceID := GetWorkspaceID(c)
		if workspaceID == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		allowed, remaining, err := usageService.CheckAILimit(ctx, *workspaceID)
		if err != nil {
			slog.ErrorContext(ctx, "ai_rate_limit: limit check failed",
				"workspace_id", workspaceID, "error", err)
			c.Next()
			return
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":     "Daily AI call limit reached for your plan.",
				"code":      "ai_limit_exceeded",
				"remaining": remaining,
				"action":    "Upgrade your plan to continue using AI features today.",
			})
			return
		}

		// Record the call before passing to the handler. Fire-and-forget errors
		// are logged but do not block the request.
		if err := usageService.RecordAICall(ctx, *workspaceID); err != nil {
			slog.ErrorContext(ctx, "ai_rate_limit: failed to record ai call",
				"workspace_id", workspaceID, "error", err)
		}

		c.Next()
	}
}

// isAIEndpoint returns true for paths that consume AI tokens.
func isAIEndpoint(path string) bool {
	aiPaths := []string{
		"/api/chat",
		"/api/v1/chat",
		"/api/osa/",
		"/api/v1/osa/",
	}
	for _, p := range aiPaths {
		if len(path) >= len(p) && path[:len(p)] == p {
			return true
		}
	}
	return false
}

// isModuleEndpoint returns true for paths that create/manage OSA modules.
func isModuleEndpoint(path string) bool {
	modulePaths := []string{
		"/api/modules",
		"/api/v1/modules",
		"/api/osa/apps",
		"/api/v1/osa/apps",
	}
	for _, p := range modulePaths {
		if len(path) >= len(p) && path[:len(p)] == p {
			return true
		}
	}
	return false
}

// isTeamEndpoint returns true for paths that manage team membership.
func isTeamEndpoint(path string) bool {
	teamPaths := []string{
		"/api/team",
		"/api/v1/team",
		"/api/workspaces/members",
		"/api/v1/workspaces/members",
	}
	for _, p := range teamPaths {
		if len(path) >= len(p) && path[:len(p)] == p {
			return true
		}
	}
	return false
}
