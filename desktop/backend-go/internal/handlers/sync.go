package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// SyncHandler handles table-level sync endpoints.
type SyncHandler struct {
	pool *pgxpool.Pool
}

// NewSyncHandler creates a new SyncHandler.
func NewSyncHandler(pool *pgxpool.Pool) *SyncHandler {
	return &SyncHandler{pool: pool}
}

// RegisterSyncRoutes registers all sync routes.
// The /api/sync group and the per-table /{table}/sync convenience routes are
// both registered here so ownership is fully contained in SyncHandler.
func RegisterSyncRoutes(api *gin.RouterGroup, h *SyncHandler, auth gin.HandlerFunc) {
	// /api/sync group
	syncRoutes := api.Group("/sync")
	syncRoutes.Use(auth, middleware.RequireAuth())
	{
		syncRoutes.GET("/status", h.GetSyncStatus)
		syncRoutes.GET("/full", h.FullSync)
		syncRoutes.GET("/:table", h.GetSyncChanges)
	}

	// Per-table convenience endpoints for the sync engine
	api.GET("/contexts/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("contexts"))
	api.GET("/conversations/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("conversations"))
	api.GET("/projects/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("projects"))
	api.GET("/tasks/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("tasks"))
	api.GET("/nodes/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("nodes"))
	api.GET("/clients/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("clients"))
	api.GET("/calendar_events/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("calendar_events"))
	api.GET("/daily_logs/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("daily_logs"))
	api.GET("/team_members/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("team_members"))
	api.GET("/artifacts/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("artifacts"))
	api.GET("/focus_items/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("focus_items"))
	api.GET("/user_settings/sync", auth, middleware.RequireAuth(), h.createTableSyncHandler("user_settings"))
}

// SyncRequest represents a sync request
type SyncRequest struct {
	Since    string   `form:"since"`
	Tables   []string `form:"tables"`
	FullSync bool     `form:"full_sync"`
}

// SyncableTables defines tables that support sync
var SyncableTables = map[string]string{
	"contexts":            "SELECT * FROM contexts WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"conversations":       "SELECT * FROM conversations WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"messages":            "SELECT m.* FROM messages m JOIN conversations c ON m.conversation_id = c.id WHERE c.user_id = $1 AND m.created_at > $2 ORDER BY m.created_at DESC",
	"projects":            "SELECT * FROM projects WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"artifacts":           "SELECT * FROM artifacts WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"nodes":               "SELECT * FROM nodes WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"team_members":        "SELECT * FROM team_members WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"tasks":               "SELECT * FROM tasks WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"focus_items":         "SELECT * FROM focus_items WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"daily_logs":          "SELECT * FROM daily_logs WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"user_settings":       "SELECT * FROM user_settings WHERE user_id = $1 AND updated_at > $2",
	"clients":             "SELECT * FROM clients WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
	"client_contacts":     "SELECT cc.* FROM client_contacts cc JOIN clients c ON cc.client_id = c.id WHERE c.user_id = $1 AND cc.updated_at > $2 ORDER BY cc.updated_at DESC",
	"client_interactions": "SELECT ci.* FROM client_interactions ci JOIN clients c ON ci.client_id = c.id WHERE c.user_id = $1 AND ci.created_at > $2 ORDER BY ci.created_at DESC",
	"calendar_events":     "SELECT * FROM calendar_events WHERE user_id = $1 AND updated_at > $2 ORDER BY updated_at DESC",
}

// SyncableWorkspaceTables holds the workspace-aware sync query for tables that
// carry a workspace_id (BusinessOSSync). When a sync request includes the
// X-Workspace-ID header for a workspace the user actively belongs to, these
// queries return the user's own records PLUS everything shared into that
// workspace — so a doc one teammate marks shared shows up for everyone.
// Params: $1 = userID, $2 = since, $3 = workspaceID.
var SyncableWorkspaceTables = map[string]string{
	"contexts": "SELECT * FROM contexts WHERE (user_id = $1 OR workspace_id = $3) AND updated_at > $2 ORDER BY updated_at DESC",
	"projects": "SELECT * FROM projects WHERE (user_id = $1 OR workspace_id = $3) AND updated_at > $2 ORDER BY updated_at DESC",
	"tasks":    "SELECT * FROM tasks WHERE (user_id = $1 OR workspace_id = $3) AND updated_at > $2 ORDER BY updated_at DESC",
}

// GetSyncChanges returns changes since a given timestamp.
// GET /api/sync/:table
func (h *SyncHandler) GetSyncChanges(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := user.ID

	table := c.Param("table")
	since := c.Query("since")

	// Validate table is syncable
	query, ok := SyncableTables[table]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Table not syncable: " + table})
		return
	}

	// Parse since timestamp, default to epoch
	sinceTime := time.Time{}
	if since != "" {
		var err error
		sinceTime, err = time.Parse(time.RFC3339, since)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid since timestamp"})
			return
		}
	}

	// BusinessOSSync: if the client sends X-Workspace-ID for a workspace the
	// user actively belongs to, use the workspace-scoped query for tables that
	// carry workspace_id so teammates see shared rows.
	var args []interface{}
	if wsQuery, hasWS := SyncableWorkspaceTables[table]; hasWS {
		if wsHeader := c.GetHeader("X-Workspace-ID"); wsHeader != "" {
			var isMember bool
			if err := h.pool.QueryRow(c,
				`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
				wsHeader, userID).Scan(&isMember); err == nil && isMember {
				query = wsQuery
				args = []interface{}{userID, sinceTime, wsHeader}
			}
		}
	}
	if args == nil {
		args = []interface{}{userID, sinceTime}
	}

	// Execute query
	rows, err := h.pool.Query(c, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query changes"})
		return
	}
	defer rows.Close()

	// Collect results
	var results []map[string]interface{}
	fieldDescriptions := rows.FieldDescriptions()

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, fd := range fieldDescriptions {
			row[string(fd.Name)] = values[i]
		}
		results = append(results, row)
	}

	if results == nil {
		results = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, results)
}

// FullSync returns all data for initial sync.
// GET /api/sync/full
func (h *SyncHandler) FullSync(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := user.ID

	// BusinessOSSync: resolve workspace membership once for the whole full-sync
	// so workspace-aware tables return shared rows for members.
	wsHeader := c.GetHeader("X-Workspace-ID")
	var wsID string
	if wsHeader != "" {
		var isMember bool
		if err := h.pool.QueryRow(c,
			`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
			wsHeader, userID).Scan(&isMember); err == nil && isMember {
			wsID = wsHeader
		}
	}

	epoch := time.Time{}
	result := make(map[string][]map[string]interface{})

	for table, query := range SyncableTables {
		// Use the workspace-scoped query when the user is an active member and
		// this table supports workspace_id scoping.
		var args []interface{}
		if wsID != "" {
			if wsQuery, hasWS := SyncableWorkspaceTables[table]; hasWS {
				query = wsQuery
				args = []interface{}{userID, epoch, wsID}
			}
		}
		if args == nil {
			args = []interface{}{userID, epoch}
		}

		rows, err := h.pool.Query(c, query, args...)
		if err != nil {
			continue
		}

		var records []map[string]interface{}
		fieldDescriptions := rows.FieldDescriptions()

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				continue
			}

			row := make(map[string]interface{})
			for i, fd := range fieldDescriptions {
				row[string(fd.Name)] = values[i]
			}
			records = append(records, row)
		}
		rows.Close()

		if records == nil {
			records = []map[string]interface{}{}
		}
		result[table] = records
	}

	c.JSON(http.StatusOK, gin.H{
		"sync_timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":           result,
	})
}

// GetSyncStatus returns sync status and server timestamp.
// GET /api/sync/status
func (h *SyncHandler) GetSyncStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":           "healthy",
		"server_timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":          "1.0.0",
	})
}

// createTableSyncHandler creates a handler for a specific table's sync endpoint.
func (h *SyncHandler) createTableSyncHandler(table string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetCurrentUser(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		userID := user.ID

		since := c.Query("since")

		// Validate table is syncable
		query, ok := SyncableTables[table]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Table not syncable: " + table})
			return
		}

		// Parse since timestamp, default to epoch
		sinceTime := time.Time{}
		if since != "" {
			var err error
			sinceTime, err = time.Parse(time.RFC3339, since)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid since timestamp"})
				return
			}
		}

		// BusinessOSSync: if the desktop sends X-Workspace-ID for a workspace the
		// user actively belongs to, pull workspace-shared rows too (not just the
		// user's own) for tables that carry workspace_id.
		var args []interface{}
		if wsQuery, hasWS := SyncableWorkspaceTables[table]; hasWS {
			if wsHeader := c.GetHeader("X-Workspace-ID"); wsHeader != "" {
				var isMember bool
				if err := h.pool.QueryRow(c,
					`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
					wsHeader, userID).Scan(&isMember); err == nil && isMember {
					query = wsQuery
					args = []interface{}{userID, sinceTime, wsHeader}
				}
			}
		}
		if args == nil {
			args = []interface{}{userID, sinceTime}
		}

		// Execute query
		rows, err := h.pool.Query(c, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query changes"})
			return
		}
		defer rows.Close()

		// Collect results
		var results []map[string]interface{}
		fieldDescriptions := rows.FieldDescriptions()

		for rows.Next() {
			values, err := rows.Values()
			if err != nil {
				continue
			}

			row := make(map[string]interface{})
			for i, fd := range fieldDescriptions {
				row[string(fd.Name)] = values[i]
			}
			results = append(results, row)
		}

		if results == nil {
			results = []map[string]interface{}{}
		}

		c.JSON(http.StatusOK, results)
	}
}
