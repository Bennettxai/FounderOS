// Package handlers provides HTTP handlers for BusinessOS.
package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/integrations/google"
	"github.com/rhl/businessos-backend/internal/utils"
)

// IntegrationsHandler handles integration management endpoints.
type IntegrationsHandler struct {
	pool              *pgxpool.Pool
	integrationRouter *IntegrationRouter
}

// NewIntegrationsHandler creates a new integrations handler.
func NewIntegrationsHandler(pool *pgxpool.Pool, integrationRouter *IntegrationRouter) *IntegrationsHandler {
	return &IntegrationsHandler{
		pool:              pool,
		integrationRouter: integrationRouter,
	}
}

// getCalendarService returns the Google Calendar service from the integration router.
func (h *IntegrationsHandler) getCalendarService() *google.CalendarService {
	if h.integrationRouter != nil {
		return h.integrationRouter.GetGoogleCalendarService()
	}
	return nil
}

// ============================================================================
// Provider Endpoints
// ============================================================================

// GetProviders handles GET /api/integrations/providers
// Returns all available integration providers.
func (h *IntegrationsHandler) GetProviders(c *gin.Context) {
	category := c.Query("category")
	module := c.Query("module")
	status := c.Query("status")

	query := `
		SELECT id, name, description, category, icon_url,
		       oauth_config, modules, skills, status,
		       auto_live_sync, est_nodes, initial_sync, tooltip,
		       created_at, updated_at
		FROM integration_providers
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1

	if category != "" {
		query += ` AND category = $` + string(rune('0'+argNum))
		args = append(args, category)
		argNum++
	}

	if module != "" {
		query += ` AND $` + string(rune('0'+argNum)) + ` = ANY(modules)`
		args = append(args, module)
		argNum++
	}

	if status != "" {
		query += ` AND status = $` + string(rune('0'+argNum))
		args = append(args, status)
	} else {
		query += ` AND status != 'deprecated'`
	}

	query += ` ORDER BY category, name`

	var providers []map[string]interface{}

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		// Database table may not exist yet - use fallback providers
		providers = getDefaultProviders()
	} else {
		defer rows.Close()

		for rows.Next() {
			var p struct {
				ID           string
				Name         string
				Description  *string
				Category     string
				IconURL      *string
				OAuthConfig  interface{}
				Modules      []string
				Skills       []string
				Status       string
				AutoLiveSync *bool
				EstNodes     *string
				InitialSync  *string
				Tooltip      *string
				CreatedAt    interface{}
				UpdatedAt    interface{}
			}
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.IconURL,
				&p.OAuthConfig, &p.Modules, &p.Skills, &p.Status,
				&p.AutoLiveSync, &p.EstNodes, &p.InitialSync, &p.Tooltip,
				&p.CreatedAt, &p.UpdatedAt); err != nil {
				slog.Warn("[GetProviders] Scan error", "error", err)
				continue
			}

			provider := map[string]interface{}{
				"id":             p.ID,
				"name":           p.Name,
				"description":    p.Description,
				"category":       p.Category,
				"icon_url":       p.IconURL,
				"modules":        p.Modules,
				"skills":         p.Skills,
				"status":         p.Status,
				"oauth_provider": getOAuthProvider(p.ID),
			}
			// Add optional fields if present
			if p.AutoLiveSync != nil {
				provider["auto_live_sync"] = *p.AutoLiveSync
			}
			if p.EstNodes != nil {
				provider["est_nodes"] = *p.EstNodes
			}
			if p.InitialSync != nil {
				provider["initial_sync"] = *p.InitialSync
			}
			if p.Tooltip != nil {
				provider["tooltip"] = *p.Tooltip
			}
			providers = append(providers, provider)
		}

		// If no providers found (table empty), return defaults
		if len(providers) == 0 {
			providers = getDefaultProviders()
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"providers": providers,
		"count":     len(providers),
	})
}

// GetProvider handles GET /api/integrations/providers/:id
// Returns a single provider with full details.
func (h *IntegrationsHandler) GetProvider(c *gin.Context) {
	providerID := c.Param("id")

	var p struct {
		ID          string
		Name        string
		Description *string
		Category    string
		IconURL     *string
		OAuthConfig interface{}
		Modules     []string
		Skills      []string
		Status      string
	}

	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT id, name, description, category, icon_url,
		       oauth_config, modules, skills, status
		FROM integration_providers
		WHERE id = $1
	`, providerID).Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.IconURL,
		&p.OAuthConfig, &p.Modules, &p.Skills, &p.Status)

	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Provider")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"provider": map[string]interface{}{
			"id":           p.ID,
			"name":         p.Name,
			"description":  p.Description,
			"category":     p.Category,
			"icon_url":     p.IconURL,
			"oauth_config": p.OAuthConfig,
			"modules":      p.Modules,
			"skills":       p.Skills,
			"status":       p.Status,
		},
	})
}
