package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// User Integration Endpoints
// ============================================================================

// GetConnectedIntegrations handles GET /api/integrations/connected
// Returns the user's connected integrations.
func (h *IntegrationsHandler) GetConnectedIntegrations(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID

	slog.Info("[GetConnectedIntegrations] Querying for userID", "userID", userID)
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT ui.id, ui.provider_id, ui.status, ui.connected_at, ui.last_used_at,
		       ui.external_account_name, ui.external_workspace_name,
		       ui.scopes, ui.settings,
		       ip.name as provider_name, ip.category, ip.icon_url, ip.skills
		FROM user_integrations ui
		JOIN integration_providers ip ON ui.provider_id = ip.id
		WHERE ui.user_id = $1
		ORDER BY ui.connected_at DESC
	`, userID)
	if err != nil {
		slog.Error("[GetConnectedIntegrations] Query error", "error", err)
		utils.RespondInternalError(c, slog.Default(), "fetch integrations", err)
		return
	}
	defer rows.Close()

	var integrations []map[string]interface{}
	for rows.Next() {
		var i struct {
			ID                    uuid.UUID
			ProviderID            string
			Status                string
			ConnectedAt           interface{}
			LastUsedAt            interface{}
			ExternalAccountName   *string
			ExternalWorkspaceName *string
			Scopes                []string
			Settings              interface{}
			ProviderName          string
			Category              string
			IconURL               *string
			Skills                []string
		}
		if err := rows.Scan(&i.ID, &i.ProviderID, &i.Status, &i.ConnectedAt, &i.LastUsedAt,
			&i.ExternalAccountName, &i.ExternalWorkspaceName,
			&i.Scopes, &i.Settings,
			&i.ProviderName, &i.Category, &i.IconURL, &i.Skills); err != nil {
			continue
		}

		integration := map[string]interface{}{
			"id":                      i.ID,
			"provider_id":             i.ProviderID,
			"provider_name":           i.ProviderName,
			"category":                i.Category,
			"icon_url":                i.IconURL,
			"status":                  i.Status,
			"connected_at":            i.ConnectedAt,
			"last_used_at":            i.LastUsedAt,
			"external_account_name":   i.ExternalAccountName,
			"external_workspace_name": i.ExternalWorkspaceName,
			"scopes":                  i.Scopes,
			"settings":                i.Settings,
			"skills":                  i.Skills,
		}
		integrations = append(integrations, integration)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"integrations": integrations,
		"count":        len(integrations),
	})
}

// GetIntegration handles GET /api/integrations/:id
// Returns details of a specific user integration.
func (h *IntegrationsHandler) GetIntegration(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "integration ID")
		return
	}

	var i struct {
		ID                    uuid.UUID
		ProviderID            string
		Status                string
		ConnectedAt           interface{}
		LastUsedAt            interface{}
		ExternalAccountID     *string
		ExternalAccountName   *string
		ExternalWorkspaceID   *string
		ExternalWorkspaceName *string
		Scopes                []string
		Settings              interface{}
		Metadata              interface{}
		ProviderName          string
		Category              string
		IconURL               *string
		Skills                []string
		Modules               []string
	}

	err = h.pool.QueryRow(c.Request.Context(), `
		SELECT ui.id, ui.provider_id, ui.status, ui.connected_at, ui.last_used_at,
		       ui.external_account_id, ui.external_account_name,
		       ui.external_workspace_id, ui.external_workspace_name,
		       ui.scopes, ui.settings, ui.metadata,
		       ip.name, ip.category, ip.icon_url, ip.skills, ip.modules
		FROM user_integrations ui
		JOIN integration_providers ip ON ui.provider_id = ip.id
		WHERE ui.id = $1 AND ui.user_id = $2
	`, id, userID).Scan(&i.ID, &i.ProviderID, &i.Status, &i.ConnectedAt, &i.LastUsedAt,
		&i.ExternalAccountID, &i.ExternalAccountName,
		&i.ExternalWorkspaceID, &i.ExternalWorkspaceName,
		&i.Scopes, &i.Settings, &i.Metadata,
		&i.ProviderName, &i.Category, &i.IconURL, &i.Skills, &i.Modules)

	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Integration")
		return
	}

	// Get comprehensive sync stats for this integration
	syncStats := h.getIntegrationSyncStats(c, id, i.ProviderID, userID)

	// Get available permissions for this provider
	availablePermissions := getAvailablePermissions(i.ProviderID)

	// Get sync history (last 10 syncs)
	syncHistory := h.getSyncHistory(c, id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"integration": map[string]interface{}{
			"id":                      i.ID,
			"provider_id":             i.ProviderID,
			"provider_name":           i.ProviderName,
			"category":                i.Category,
			"icon_url":                i.IconURL,
			"status":                  i.Status,
			"connected_at":            i.ConnectedAt,
			"last_used_at":            i.LastUsedAt,
			"external_account_id":     i.ExternalAccountID,
			"external_account_name":   i.ExternalAccountName,
			"external_workspace_id":   i.ExternalWorkspaceID,
			"external_workspace_name": i.ExternalWorkspaceName,
			"scopes":                  i.Scopes,
			"available_permissions":   availablePermissions,
			"settings":                i.Settings,
			"metadata":                i.Metadata,
			"skills":                  i.Skills,
			"modules":                 i.Modules,
			"sync_stats":              syncStats,
			"sync_history":            syncHistory,
		},
	})
}

// UpdateIntegrationSettingsRequest represents the request body for updating settings.
type UpdateIntegrationSettingsRequest struct {
	EnabledSkills []string               `json:"enabled_skills"`
	Notifications bool                   `json:"notifications"`
	SyncSettings  map[string]interface{} `json:"sync_settings"`
}

// UpdateIntegrationSettings handles PATCH /api/integrations/:id/settings
// Updates settings for a user's integration.
func (h *IntegrationsHandler) UpdateIntegrationSettings(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "integration ID")
		return
	}

	var req UpdateIntegrationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	settings := map[string]interface{}{
		"enabledSkills": req.EnabledSkills,
		"notifications": req.Notifications,
		"syncSettings":  req.SyncSettings,
	}

	_, err = h.pool.Exec(c.Request.Context(), `
		UPDATE user_integrations SET
			settings = $3,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, userID, settings)

	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update settings", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Settings updated",
	})
}

// DisconnectIntegration handles DELETE /api/integrations/:id
// Disconnects a user's integration.
func (h *IntegrationsHandler) DisconnectIntegration(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "integration ID")
		return
	}

	_, err = h.pool.Exec(c.Request.Context(), `
		UPDATE user_integrations SET
			status = 'disconnected',
			access_token_encrypted = NULL,
			refresh_token_encrypted = NULL,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, id, userID)

	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "disconnect integration", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Integration disconnected",
	})
}
