package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// Sync & Module Integration Endpoints
// ============================================================================

// TriggerSync handles POST /api/integrations/:id/sync
// Triggers a manual sync for an integration.
func (h *IntegrationsHandler) TriggerSync(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	idStr := c.Param("id")
	module := c.Query("module")

	id, err := uuid.Parse(idStr)
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "integration ID")
		return
	}

	// Get the integration to check provider
	var providerID string
	err = h.pool.QueryRow(c.Request.Context(), `
		SELECT provider_id FROM user_integrations WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&providerID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Integration")
		return
	}

	// Create sync log entry
	var syncLogID uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO integration_sync_log (
			user_integration_id, module_id, sync_type, direction, status
		) VALUES ($1, $2, 'manual', 'bidirectional', 'in_progress')
		RETURNING id
	`, id, module).Scan(&syncLogID)

	if err != nil {
		slog.Warn("Failed to create sync log", "error", err)
		// Continue anyway - sync log is nice to have but not critical
	}

	// Perform actual sync based on provider
	var syncedCount int
	var syncError error
	syncStatusCode := http.StatusInternalServerError

	switch providerID {
	case "google_calendar", "google":
		// Sync calendar events from Google using new integration infrastructure
		calendarService := h.getCalendarService()
		if calendarService == nil {
			syncStatusCode = http.StatusServiceUnavailable
			syncError = fmt.Errorf("google calendar sync service is not configured")
		} else {
			timeMin := time.Now().AddDate(0, -1, 0) // Last month (recent past is enough)
			timeMax := time.Now().AddDate(0, 6, 0)  // Next 6 months (prioritize upcoming)
			_, syncError = calendarService.SyncEvents(c.Request.Context(), userID, timeMin, timeMax)
			if syncError == nil {
				// Count events synced (approximate)
				h.pool.QueryRow(c.Request.Context(),
					"SELECT COUNT(*) FROM calendar_events WHERE user_id = $1 AND source = 'google'",
					userID).Scan(&syncedCount)
			}
		}
	case "slack":
		syncStatusCode = http.StatusNotImplemented
		syncError = fmt.Errorf("slack sync must be handled by the integration provider service")
	case "notion":
		syncStatusCode = http.StatusNotImplemented
		syncError = fmt.Errorf("notion sync must be handled by the integration provider service")
	default:
		syncStatusCode = http.StatusNotImplemented
		syncError = fmt.Errorf("manual sync is not implemented for provider %q", providerID)
	}

	// Update sync log status
	if syncLogID != uuid.Nil {
		status := "completed"
		if syncError != nil {
			status = "failed"
		}
		h.pool.Exec(c.Request.Context(), `
			UPDATE integration_sync_log
			SET status = $1, completed_at = NOW(), records_synced = $2
			WHERE id = $3
		`, status, syncedCount, syncLogID)
	}

	if syncError != nil {
		slog.Warn("Sync unavailable for integration",
			"integration_id", idStr,
			"provider_id", providerID,
			"status", syncStatusCode,
			"error", syncError,
		)
		c.JSON(syncStatusCode, gin.H{
			"success":     false,
			"sync_log_id": syncLogID,
			"provider_id": providerID,
			"error":       syncError.Error(),
		})
		return
	}

	// Get detailed sync info for Google Calendar
	var syncDetails map[string]interface{}
	if providerID == "google_calendar" {
		var minDate, maxDate *time.Time
		var eventCount int
		h.pool.QueryRow(c.Request.Context(), `
			SELECT COUNT(*), MIN(start_time), MAX(start_time)
			FROM calendar_events WHERE user_id = $1 AND source = 'google'
		`, userID).Scan(&eventCount, &minDate, &maxDate)

		syncDetails = map[string]interface{}{
			"total_events": eventCount,
			"date_range": map[string]interface{}{
				"from": minDate,
				"to":   maxDate,
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"sync_log_id":  syncLogID,
		"message":      "Sync completed successfully",
		"synced_count": syncedCount,
		"details":      syncDetails,
	})
}

// GetModuleIntegrations handles GET /api/modules/:id/integrations
// Returns available and connected integrations for a specific module.
func (h *IntegrationsHandler) GetModuleIntegrations(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}
	userID := user.ID
	module := c.Param("id")

	// Get available providers for this module
	providerRows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, name, description, category, icon_url, skills, status
		FROM integration_providers
		WHERE $1 = ANY(modules) AND status != 'deprecated'
		ORDER BY status = 'available' DESC, name
	`, module)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "fetch providers", err)
		return
	}
	defer providerRows.Close()

	var availableProviders []map[string]interface{}
	for providerRows.Next() {
		var p struct {
			ID          string
			Name        string
			Description *string
			Category    string
			IconURL     *string
			Skills      []string
			Status      string
		}
		if err := providerRows.Scan(&p.ID, &p.Name, &p.Description, &p.Category, &p.IconURL, &p.Skills, &p.Status); err != nil {
			continue
		}
		availableProviders = append(availableProviders, map[string]interface{}{
			"id":          p.ID,
			"name":        p.Name,
			"description": p.Description,
			"category":    p.Category,
			"icon_url":    p.IconURL,
			"skills":      p.Skills,
			"status":      p.Status,
		})
	}

	// Get user's connected integrations for this module
	var connectedIntegrations []map[string]interface{}
	if userID != "" {
		connRows, err := h.pool.Query(c.Request.Context(), `
			SELECT ui.id, ui.provider_id, ui.status, ui.last_used_at,
			       ui.external_account_name, ui.settings,
			       ip.name, ip.icon_url
			FROM user_integrations ui
			JOIN integration_providers ip ON ui.provider_id = ip.id
			WHERE ui.user_id = $1 AND ui.status = 'connected' AND $2 = ANY(ip.modules)
		`, userID, module)
		if err == nil {
			defer connRows.Close()
			for connRows.Next() {
				var i struct {
					ID                  uuid.UUID
					ProviderID          string
					Status              string
					LastUsedAt          interface{}
					ExternalAccountName *string
					Settings            interface{}
					ProviderName        string
					IconURL             *string
				}
				if err := connRows.Scan(&i.ID, &i.ProviderID, &i.Status, &i.LastUsedAt,
					&i.ExternalAccountName, &i.Settings,
					&i.ProviderName, &i.IconURL); err != nil {
					continue
				}
				connectedIntegrations = append(connectedIntegrations, map[string]interface{}{
					"id":                    i.ID,
					"provider_id":           i.ProviderID,
					"provider_name":         i.ProviderName,
					"icon_url":              i.IconURL,
					"status":                i.Status,
					"last_used_at":          i.LastUsedAt,
					"external_account_name": i.ExternalAccountName,
					"settings":              i.Settings,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":                true,
		"module":                 module,
		"available_providers":    availableProviders,
		"connected_integrations": connectedIntegrations,
	})
}
