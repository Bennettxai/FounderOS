package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============================================================================
// Helper Functions for Integration Stats
// ============================================================================

// getIntegrationSyncStats returns detailed sync statistics for an integration.
func (h *IntegrationsHandler) getIntegrationSyncStats(c *gin.Context, integrationID uuid.UUID, providerID, userID string) map[string]interface{} {
	stats := map[string]interface{}{
		"total_items":      0,
		"items_by_type":    map[string]int{},
		"date_range":       nil,
		"last_sync":        nil,
		"last_sync_status": nil,
		"sync_count":       0,
	}

	switch providerID {
	case "google_calendar":
		// Get calendar event stats
		var eventCount int
		var minDate, maxDate *time.Time
		h.pool.QueryRow(c.Request.Context(), `
			SELECT COUNT(*), MIN(start_time), MAX(start_time)
			FROM calendar_events WHERE user_id = $1 AND source = 'google'
		`, userID).Scan(&eventCount, &minDate, &maxDate)

		stats["total_items"] = eventCount
		stats["items_by_type"] = map[string]int{
			"events": eventCount,
		}

		if minDate != nil && maxDate != nil {
			stats["date_range"] = map[string]interface{}{
				"from": minDate,
				"to":   maxDate,
			}
		}

		// Get last sync info - use started_at as fallback if completed_at is null
		var lastSync, startedAt *time.Time
		var lastStatus *string
		var syncCount int
		h.pool.QueryRow(c.Request.Context(), `
			SELECT COALESCE(completed_at, started_at), started_at, status FROM integration_sync_log
			WHERE user_integration_id = $1
			ORDER BY started_at DESC LIMIT 1
		`, integrationID).Scan(&lastSync, &startedAt, &lastStatus)

		h.pool.QueryRow(c.Request.Context(), `
			SELECT COUNT(*) FROM integration_sync_log
			WHERE user_integration_id = $1
		`, integrationID).Scan(&syncCount)

		// Use started_at if completed_at is null
		if lastSync == nil && startedAt != nil {
			lastSync = startedAt
		}

		stats["last_sync"] = lastSync
		stats["last_sync_status"] = lastStatus
		stats["sync_count"] = syncCount

	case "gmail":
		// Get email stats (for future)
		var emailCount int
		var unreadCount int
		h.pool.QueryRow(c.Request.Context(), `
			SELECT COUNT(*), COUNT(*) FILTER (WHERE is_read = false)
			FROM emails WHERE user_id = $1 AND provider = 'gmail'
		`, userID).Scan(&emailCount, &unreadCount)

		stats["total_items"] = emailCount
		stats["items_by_type"] = map[string]int{
			"emails": emailCount,
			"unread": unreadCount,
		}

	case "slack":
		// Get channel stats (for future)
		var channelCount int
		var messageCount int
		h.pool.QueryRow(c.Request.Context(), `
			SELECT COUNT(*) FROM channels WHERE user_id = $1 AND provider = 'slack'
		`, userID).Scan(&channelCount)
		h.pool.QueryRow(c.Request.Context(), `
			SELECT COUNT(*) FROM channel_messages cm
			JOIN channels c ON cm.channel_id = c.id
			WHERE c.user_id = $1 AND c.provider = 'slack'
		`, userID).Scan(&messageCount)

		stats["total_items"] = channelCount + messageCount
		stats["items_by_type"] = map[string]int{
			"channels": channelCount,
			"messages": messageCount,
		}
	}

	return stats
}

// getSyncHistory returns the last 10 sync operations for an integration.
func (h *IntegrationsHandler) getSyncHistory(c *gin.Context, integrationID uuid.UUID) []map[string]interface{} {
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, sync_type, direction, status, started_at, completed_at, records_synced, error_message
		FROM integration_sync_log
		WHERE user_integration_id = $1
		ORDER BY started_at DESC
		LIMIT 10
	`, integrationID)
	if err != nil {
		return []map[string]interface{}{}
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var (
			id            uuid.UUID
			syncType      string
			direction     string
			status        string
			startedAt     *time.Time
			completedAt   *time.Time
			recordsSynced *int
			errorMessage  *string
		)
		if err := rows.Scan(&id, &syncType, &direction, &status, &startedAt, &completedAt, &recordsSynced, &errorMessage); err != nil {
			continue
		}
		history = append(history, map[string]interface{}{
			"id":             id,
			"sync_type":      syncType,
			"direction":      direction,
			"status":         status,
			"started_at":     startedAt,
			"completed_at":   completedAt,
			"records_synced": recordsSynced,
			"error_message":  errorMessage,
		})
	}
	return history
}

// getAvailablePermissions returns all available permissions for a provider.
func getAvailablePermissions(providerID string) []map[string]interface{} {
	permissions := map[string][]map[string]interface{}{
		"google_calendar": {
			{"scope": "calendar", "name": "Calendar Access", "description": "View and manage your calendars", "granted": true},
			{"scope": "calendar.readonly", "name": "Calendar Read-Only", "description": "View your calendars", "granted": true},
			{"scope": "calendar.events", "name": "Calendar Events", "description": "Create and edit events", "granted": true},
			{"scope": "calendar.settings.readonly", "name": "Calendar Settings", "description": "View calendar settings", "granted": false},
		},
		"gmail": {
			{"scope": "gmail.readonly", "name": "Gmail Read-Only", "description": "View your emails", "granted": false},
			{"scope": "gmail.send", "name": "Gmail Send", "description": "Send emails on your behalf", "granted": false},
			{"scope": "gmail.compose", "name": "Gmail Compose", "description": "Compose new emails", "granted": false},
			{"scope": "gmail.modify", "name": "Gmail Full Access", "description": "Read, send, and manage emails", "granted": false},
		},
		"slack": {
			{"scope": "channels:read", "name": "View Channels", "description": "View public channels", "granted": false},
			{"scope": "channels:history", "name": "Channel History", "description": "View messages in channels", "granted": false},
			{"scope": "chat:write", "name": "Send Messages", "description": "Send messages as the app", "granted": false},
			{"scope": "users:read", "name": "View Users", "description": "View workspace members", "granted": false},
		},
		"notion": {
			{"scope": "read_content", "name": "Read Content", "description": "View pages and databases", "granted": false},
			{"scope": "insert_content", "name": "Insert Content", "description": "Create new pages", "granted": false},
			{"scope": "update_content", "name": "Update Content", "description": "Edit existing pages", "granted": false},
		},
	}

	if perms, ok := permissions[providerID]; ok {
		return perms
	}
	return []map[string]interface{}{}
}
