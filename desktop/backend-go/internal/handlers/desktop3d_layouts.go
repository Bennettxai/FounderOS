package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ModulePosition represents a 3D module's position, rotation, and scale
type ModulePosition struct {
	ModuleID string             `json:"module_id"`
	Position map[string]float64 `json:"position"` // {x, y, z}
	Rotation map[string]float64 `json:"rotation"` // {x, y, z}
	Scale    float64            `json:"scale"`
}

// Desktop3DLayout represents a saved 3D Desktop layout
type Desktop3DLayout struct {
	ID        uuid.UUID        `json:"id"`
	UserID    string           `json:"user_id"`
	Name      string           `json:"name"`
	Type      string           `json:"type"` // "default" or "custom"
	IsActive  bool             `json:"is_active"`
	Modules   []ModulePosition `json:"modules"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// Desktop3DHandler handles all 3D desktop layout operations.
type Desktop3DHandler struct {
	pool *pgxpool.Pool
}

// NewDesktop3DHandler constructs a Desktop3DHandler with the given connection pool.
func NewDesktop3DHandler(pool *pgxpool.Pool) *Desktop3DHandler {
	return &Desktop3DHandler{pool: pool}
}

// RegisterDesktop3DRoutes mounts all desktop3d routes under the provided API group.
func RegisterDesktop3DRoutes(api *gin.RouterGroup, h *Desktop3DHandler, auth gin.HandlerFunc) {
	desktop3d := api.Group("/desktop3d")
	desktop3d.Use(auth, middleware.RequireAuth())
	{
		desktop3d.GET("/layouts", h.ListDesktop3DLayouts)
		desktop3d.POST("/layouts", h.CreateDesktop3DLayout)
		desktop3d.GET("/layouts/active", h.GetActiveDesktop3DLayout)
		desktop3d.POST("/layouts/:id/activate", h.ActivateDesktop3DLayout)
		desktop3d.DELETE("/layouts/:id", h.DeleteDesktop3DLayout)
	}
}

// ListDesktop3DLayouts returns all layouts for the current user.
func (h *Desktop3DHandler) ListDesktop3DLayouts(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		slog.Error("[Desktop3D Layouts] Unauthorized access attempt")
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	slog.Info("[Desktop3D Layouts] Listing layouts", "user_id", user.ID)

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, user_id, name, type, is_active, modules, created_at, updated_at
		FROM desktop3d_layouts
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, user.ID)

	if err != nil {
		slog.Error("[Desktop3D Layouts] Failed to query layouts", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "fetch layouts", err)
		return
	}
	defer rows.Close()

	layouts := []Desktop3DLayout{}
	for rows.Next() {
		var layout Desktop3DLayout
		var modulesJSON []byte

		err := rows.Scan(
			&layout.ID,
			&layout.UserID,
			&layout.Name,
			&layout.Type,
			&layout.IsActive,
			&modulesJSON,
			&layout.CreatedAt,
			&layout.UpdatedAt,
		)

		if err != nil {
			slog.Warn("[Desktop3D Layouts] Failed to scan layout row", "error", err)
			continue
		}

		// Parse modules JSON
		if err := json.Unmarshal(modulesJSON, &layout.Modules); err != nil {
			slog.Warn("[Desktop3D Layouts] Failed to unmarshal modules", "error", err, "layout_id", layout.ID)
			layout.Modules = []ModulePosition{} // Empty array as fallback
		}

		layouts = append(layouts, layout)
	}

	slog.Info("[Desktop3D Layouts] Successfully listed layouts", "user_id", user.ID, "count", len(layouts))
	c.JSON(http.StatusOK, layouts)
}

// CreateDesktop3DLayout creates a new custom layout.
func (h *Desktop3DHandler) CreateDesktop3DLayout(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		slog.Error("[Desktop3D Layouts] Unauthorized create attempt")
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var input struct {
		Name    string           `json:"name" binding:"required"`
		Modules []ModulePosition `json:"modules" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		slog.Warn("[Desktop3D Layouts] Invalid input", "error", err, "user_id", user.ID)
		utils.RespondBadRequest(c, slog.Default(), "Invalid input: name and modules are required")
		return
	}

	// Validate name length
	if len(input.Name) == 0 || len(input.Name) > 255 {
		slog.Warn("[Desktop3D Layouts] Invalid name length", "user_id", user.ID, "name", input.Name)
		utils.RespondBadRequest(c, slog.Default(), "Name must be between 1 and 255 characters")
		return
	}

	// Validate modules array
	if len(input.Modules) == 0 {
		slog.Warn("[Desktop3D Layouts] Empty modules array", "user_id", user.ID)
		utils.RespondBadRequest(c, slog.Default(), "Modules array cannot be empty")
		return
	}

	slog.Info("[Desktop3D Layouts] Creating new layout", "user_id", user.ID, "name", input.Name, "module_count", len(input.Modules))

	// Marshal modules to JSON
	modulesJSON, err := json.Marshal(input.Modules)
	if err != nil {
		slog.Error("[Desktop3D Layouts] Failed to marshal modules", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "process modules", err)
		return
	}

	layoutID := uuid.New()

	// Insert new layout
	_, err = h.pool.Exec(c.Request.Context(), `
		INSERT INTO desktop3d_layouts (id, user_id, name, type, modules)
		VALUES ($1, $2, $3, 'custom', $4)
	`, layoutID, user.ID, input.Name, modulesJSON)

	if err != nil {
		slog.Error("[Desktop3D Layouts] Failed to insert layout", "error", err, "user_id", user.ID, "name", input.Name)
		utils.RespondInternalError(c, slog.Default(), "save layout", err)
		return
	}

	// Fetch and return the created layout
	var layout Desktop3DLayout
	err = h.pool.QueryRow(c.Request.Context(), `
		SELECT id, user_id, name, type, is_active, modules, created_at, updated_at
		FROM desktop3d_layouts
		WHERE id = $1
	`, layoutID).Scan(
		&layout.ID,
		&layout.UserID,
		&layout.Name,
		&layout.Type,
		&layout.IsActive,
		&modulesJSON,
		&layout.CreatedAt,
		&layout.UpdatedAt,
	)

	if err != nil {
		slog.Error("[Desktop3D Layouts] Failed to fetch created layout", "error", err, "layout_id", layoutID)
		utils.RespondInternalError(c, slog.Default(), "fetch created layout", err)
		return
	}

	// Parse modules JSON
	if err := json.Unmarshal(modulesJSON, &layout.Modules); err != nil {
		slog.Warn("[Desktop3D Layouts] Failed to unmarshal modules on fetch", "error", err, "layout_id", layoutID)
		layout.Modules = []ModulePosition{}
	}

	slog.Info("[Desktop3D Layouts] Successfully created layout", "layout_id", layoutID, "user_id", user.ID, "name", input.Name)
	c.JSON(http.StatusOK, layout)
}

// ActivateDesktop3DLayout sets a layout as active (deactivates all others).
func (h *Desktop3DHandler) ActivateDesktop3DLayout(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		slog.Error("[Desktop3D Layouts] Unauthorized activate attempt")
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	layoutID := c.Param("id")
	parsedID, err := uuid.Parse(layoutID)
	if err != nil {
		slog.Warn("[Desktop3D Layouts] Invalid layout ID", "id", layoutID, "user_id", user.ID)
		utils.RespondInvalidID(c, slog.Default(), "layout ID")
		return
	}

	slog.Info("[Desktop3D Layouts] Activating layout", "layout_id", parsedID, "user_id", user.ID)

	// Start transaction
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		slog.Error("[Desktop3D Layouts] Failed to begin transaction", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "activate layout", err)
		return
	}
	defer tx.Rollback(c.Request.Context())

	// Deactivate all layouts for this user
	_, err = tx.Exec(c.Request.Context(), `
		UPDATE desktop3d_layouts
		SET is_active = false, updated_at = NOW()
		WHERE user_id = $1
	`, user.ID)

	if err != nil {
		slog.Error("[Desktop3D Layouts] Failed to deactivate layouts", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "update layouts", err)
		return
	}

	// Activate the specified layout
	result, err := tx.Exec(c.Request.Context(), `
		UPDATE desktop3d_layouts
		SET is_active = true, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`, parsedID, user.ID)

	if err != nil {
		slog.Error("[Desktop3D Layouts] Failed to activate layout", "error", err, "layout_id", parsedID, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "activate layout", err)
		return
	}

	// Check if the layout was found
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		slog.Warn("[Desktop3D Layouts] Layout not found or not owned by user", "layout_id", parsedID, "user_id", user.ID)
		utils.RespondNotFound(c, slog.Default(), "Layout")
		return
	}

	// Commit transaction
	if err := tx.Commit(c.Request.Context()); err != nil {
		slog.Error("[Desktop3D Layouts] Failed to commit transaction", "error", err, "layout_id", parsedID)
		utils.RespondInternalError(c, slog.Default(), "activate layout", err)
		return
	}

	slog.Info("[Desktop3D Layouts] Successfully activated layout", "layout_id", parsedID, "user_id", user.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Layout activated"})
}

// DeleteDesktop3DLayout deletes a custom layout.
func (h *Desktop3DHandler) DeleteDesktop3DLayout(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		slog.Error("[Desktop3D Layouts] Unauthorized delete attempt")
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	layoutID := c.Param("id")
	parsedID, err := uuid.Parse(layoutID)
	if err != nil {
		slog.Warn("[Desktop3D Layouts] Invalid layout ID", "id", layoutID, "user_id", user.ID)
		utils.RespondInvalidID(c, slog.Default(), "layout ID")
		return
	}

	slog.Info("[Desktop3D Layouts] Deleting layout", "layout_id", parsedID, "user_id", user.ID)

	// Delete the layout (only if owned by user)
	result, err := h.pool.Exec(c.Request.Context(), `
		DELETE FROM desktop3d_layouts
		WHERE id = $1 AND user_id = $2
	`, parsedID, user.ID)

	if err != nil {
		slog.Error("[Desktop3D Layouts] Failed to delete layout", "error", err, "layout_id", parsedID, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "delete layout", err)
		return
	}

	// Check if the layout was found
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		slog.Warn("[Desktop3D Layouts] Layout not found or not owned by user", "layout_id", parsedID, "user_id", user.ID)
		utils.RespondNotFound(c, slog.Default(), "Layout")
		return
	}

	slog.Info("[Desktop3D Layouts] Successfully deleted layout", "layout_id", parsedID, "user_id", user.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Layout deleted"})
}

// GetActiveDesktop3DLayout returns the currently active layout for the user.
func (h *Desktop3DHandler) GetActiveDesktop3DLayout(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		slog.Error("[Desktop3D Layouts] Unauthorized access to active layout")
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	slog.Info("[Desktop3D Layouts] Getting active layout", "user_id", user.ID)

	var layout Desktop3DLayout
	var modulesJSON []byte

	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT id, user_id, name, type, is_active, modules, created_at, updated_at
		FROM desktop3d_layouts
		WHERE user_id = $1 AND is_active = true
		LIMIT 1
	`, user.ID).Scan(
		&layout.ID,
		&layout.UserID,
		&layout.Name,
		&layout.Type,
		&layout.IsActive,
		&modulesJSON,
		&layout.CreatedAt,
		&layout.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			slog.Info("[Desktop3D Layouts] No active layout found", "user_id", user.ID)
			c.JSON(http.StatusOK, nil)
			return
		}
		slog.Error("[Desktop3D Layouts] Failed to fetch active layout", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "fetch active layout", err)
		return
	}

	// Parse modules JSON
	if err := json.Unmarshal(modulesJSON, &layout.Modules); err != nil {
		slog.Warn("[Desktop3D Layouts] Failed to unmarshal modules", "error", err, "layout_id", layout.ID)
		layout.Modules = []ModulePosition{}
	}

	slog.Info("[Desktop3D Layouts] Successfully fetched active layout", "layout_id", layout.ID, "user_id", user.ID)
	c.JSON(http.StatusOK, layout)
}
