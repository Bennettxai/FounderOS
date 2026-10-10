package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// View API Types
// ============================================================================

// ViewResponse represents a view in API responses
type ViewResponse struct {
	ID           string                 `json:"id"`
	TableID      string                 `json:"table_id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	Description  *string                `json:"description,omitempty"`
	IsDefault    bool                   `json:"is_default"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Filters      []interface{}          `json:"filters,omitempty"`
	Sorts        []interface{}          `json:"sorts,omitempty"`
	GroupBy      *string                `json:"group_by,omitempty"`
	ViewSettings map[string]interface{} `json:"view_settings,omitempty"`
	Position     int                    `json:"position"`
	CreatedAt    string                 `json:"created_at"`
	UpdatedAt    string                 `json:"updated_at"`
}

// CreateViewRequest represents the request to create a view
type CreateViewRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Type         string                 `json:"type" binding:"required"`
	Description  *string                `json:"description,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Filters      []interface{}          `json:"filters,omitempty"`
	Sorts        []interface{}          `json:"sorts,omitempty"`
	GroupBy      *string                `json:"group_by,omitempty"`
	ViewSettings map[string]interface{} `json:"view_settings,omitempty"`
	Position     *int                   `json:"position,omitempty"`
}

// UpdateViewRequest represents the request to update a view
type UpdateViewRequest struct {
	Name         *string                `json:"name,omitempty"`
	Description  *string                `json:"description,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Filters      []interface{}          `json:"filters,omitempty"`
	Sorts        []interface{}          `json:"sorts,omitempty"`
	GroupBy      *string                `json:"group_by,omitempty"`
	ViewSettings map[string]interface{} `json:"view_settings,omitempty"`
	IsDefault    *bool                  `json:"is_default,omitempty"`
}

// ============================================================================
// View Handlers
// ============================================================================

// ListViews returns all views for a table
// GET /api/tables/:id/views
func (h *TableHandler) ListViews(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	tableID, err := tableParseUUID(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "table_id")
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Verify user owns table
	_, err = queries.GetCustomTable(ctx, sqlc.GetCustomTableParams{
		ID:     tableID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Table")
		return
	}

	views, err := queries.ListCustomViews(ctx, tableID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list views", nil)
		return
	}

	response := make([]ViewResponse, len(views))
	for i, v := range views {
		response[i] = viewToResponse(v)
	}

	c.JSON(http.StatusOK, response)
}

// CreateView creates a new view for a table
// POST /api/tables/:id/views
func (h *TableHandler) CreateView(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	tableID, err := tableParseUUID(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "table_id")
		return
	}

	var req CreateViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Verify user owns table
	_, err = queries.GetCustomTable(ctx, sqlc.GetCustomTableParams{
		ID:     tableID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Table")
		return
	}

	// Get next position
	position := int32(0)
	if req.Position != nil {
		position = int32(*req.Position)
	} else {
		existingViews, _ := queries.ListCustomViews(ctx, tableID)
		position = int32(len(existingViews))
	}

	// Parse group_by if provided
	var groupBy pgtype.UUID
	if req.GroupBy != nil {
		parsed, err := tableParseUUID(*req.GroupBy)
		if err == nil {
			groupBy = parsed
		}
	}

	view, err := queries.CreateCustomView(ctx, sqlc.CreateCustomViewParams{
		TableID:      tableID,
		Name:         req.Name,
		ViewType:     req.Type,
		Description:  req.Description,
		Config:       tableMapToJSONBytes(req.Config),
		Filters:      tableSliceToJSONBytes(req.Filters),
		Sorts:        tableSliceToJSONBytes(req.Sorts),
		GroupBy:      groupBy,
		ViewSettings: tableMapToJSONBytes(req.ViewSettings),
		Position:     position,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create view", nil)
		return
	}

	c.JSON(http.StatusCreated, viewToResponse(view))
}

// UpdateView updates an existing view
// PUT /api/tables/:tableId/views/:viewId
func (h *TableHandler) UpdateView(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	tableID, err := tableParseUUID(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "table_id")
		return
	}

	viewID, err := tableParseUUID(c.Param("viewId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "view_id")
		return
	}

	var req UpdateViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Verify user owns table
	_, err = queries.GetCustomTable(ctx, sqlc.GetCustomTableParams{
		ID:     tableID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Table")
		return
	}

	// Get existing view
	existing, err := queries.GetCustomView(ctx, viewID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "View")
		return
	}

	// Verify view belongs to table
	if tableUUIDToString(existing.TableID) != tableUUIDToString(tableID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "View does not belong to this table"})
		return
	}

	// Merge updates
	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}
	description := existing.Description
	if req.Description != nil {
		description = req.Description
	}
	config := existing.Config
	if req.Config != nil {
		config = tableMapToJSONBytes(req.Config)
	}
	filters := existing.Filters
	if req.Filters != nil {
		filters = tableSliceToJSONBytes(req.Filters)
	}
	sorts := existing.Sorts
	if req.Sorts != nil {
		sorts = tableSliceToJSONBytes(req.Sorts)
	}
	groupBy := existing.GroupBy
	if req.GroupBy != nil {
		parsed, err := tableParseUUID(*req.GroupBy)
		if err == nil {
			groupBy = parsed
		}
	}
	viewSettings := existing.ViewSettings
	if req.ViewSettings != nil {
		viewSettings = tableMapToJSONBytes(req.ViewSettings)
	}

	// Handle is_default separately
	if req.IsDefault != nil && *req.IsDefault {
		queries.SetDefaultView(ctx, sqlc.SetDefaultViewParams{
			TableID: tableID,
			ID:      viewID,
		})
	}

	view, err := queries.UpdateCustomView(ctx, sqlc.UpdateCustomViewParams{
		ID:           viewID,
		Name:         name,
		Description:  description,
		Config:       config,
		Filters:      filters,
		Sorts:        sorts,
		GroupBy:      groupBy,
		ViewSettings: viewSettings,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update view", nil)
		return
	}

	c.JSON(http.StatusOK, viewToResponse(view))
}

// DeleteView deletes a view
// DELETE /api/tables/:tableId/views/:viewId
func (h *TableHandler) DeleteView(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	tableID, err := tableParseUUID(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "table_id")
		return
	}

	viewID, err := tableParseUUID(c.Param("viewId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "view_id")
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Verify user owns table
	_, err = queries.GetCustomTable(ctx, sqlc.GetCustomTableParams{
		ID:     tableID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Table")
		return
	}

	// Verify view belongs to table
	existing, err := queries.GetCustomView(ctx, viewID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "View")
		return
	}
	if tableUUIDToString(existing.TableID) != tableUUIDToString(tableID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "View does not belong to this table"})
		return
	}

	// Don't allow deleting the last view
	views, _ := queries.ListCustomViews(ctx, tableID)
	if len(views) <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete the last view"})
		return
	}

	err = queries.DeleteCustomView(ctx, viewID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete view", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
