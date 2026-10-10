package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

func generateShareToken() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func transformDashboard(d sqlc.UserDashboard) gin.H {
	result := gin.H{
		"id":          dashboardUuidToString(d.ID),
		"user_id":     d.UserID,
		"name":        d.Name,
		"description": d.Description,
		"is_default":  d.IsDefault,
		"layout":      json.RawMessage(d.Layout),
		"visibility":  d.Visibility,
		"share_token": d.ShareToken,
		"created_via": d.CreatedVia,
		"created_at":  d.CreatedAt.Time,
		"updated_at":  d.UpdatedAt.Time,
	}
	if d.WorkspaceID.Valid {
		result["workspace_id"] = dashboardUuidToString(d.WorkspaceID)
	}
	return result
}

func transformDashboards(dashboards []sqlc.UserDashboard) []gin.H {
	result := make([]gin.H, len(dashboards))
	for i, d := range dashboards {
		result[i] = transformDashboard(d)
	}
	return result
}

func transformWidgetTypes(widgets []sqlc.DashboardWidget) []gin.H {
	result := make([]gin.H, len(widgets))
	for i, w := range widgets {
		result[i] = gin.H{
			"widget_type":    w.WidgetType,
			"name":           w.Name,
			"description":    w.Description,
			"category":       w.Category,
			"config_schema":  json.RawMessage(w.ConfigSchema),
			"default_config": json.RawMessage(w.DefaultConfig),
			"default_size":   json.RawMessage(w.DefaultSize),
			"min_size":       json.RawMessage(w.MinSize),
			"sse_events":     w.SseEvents,
			"is_enabled":     w.IsEnabled,
		}
	}
	return result
}

func transformTemplates(templates []sqlc.DashboardTemplate) []gin.H {
	result := make([]gin.H, len(templates))
	for i, t := range templates {
		result[i] = gin.H{
			"id":            dashboardUuidToString(t.ID),
			"name":          t.Name,
			"description":   t.Description,
			"category":      t.Category,
			"layout":        json.RawMessage(t.Layout),
			"thumbnail_url": t.ThumbnailUrl,
			"is_default":    t.IsDefault,
			"sort_order":    t.SortOrder,
		}
	}
	return result
}

func dashboardUuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}
