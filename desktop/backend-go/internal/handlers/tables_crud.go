package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// TableHandler handles flexible table operations (NocoDB-inspired)
type TableHandler struct {
	pool *pgxpool.Pool
}

// NewTableHandler creates a new TableHandler
func NewTableHandler(pool *pgxpool.Pool) *TableHandler {
	return &TableHandler{pool: pool}
}

// ============================================================================
// API Request/Response Types for Tables Module
// ============================================================================

// TableResponse represents a table in API responses
type TableResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Icon        *string                `json:"icon,omitempty"`
	Color       *string                `json:"color,omitempty"`
	WorkspaceID *string                `json:"workspace_id,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	RowCount    int64                  `json:"row_count"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	// Nested data for detail view
	Columns []FieldResponse `json:"columns,omitempty"`
	Views   []ViewResponse  `json:"views,omitempty"`
}

// CreateTableRequest represents the request to create a table
type CreateTableRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description *string                `json:"description,omitempty"`
	Icon        *string                `json:"icon,omitempty"`
	Color       *string                `json:"color,omitempty"`
	WorkspaceID *string                `json:"workspace_id,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// UpdateTableRequest represents the request to update a table
type UpdateTableRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Icon        *string                `json:"icon,omitempty"`
	Color       *string                `json:"color,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// ============================================================================
// Table Handlers
// ============================================================================

// ListTables returns all tables for the current user
// GET /api/tables
func (h *TableHandler) ListTables(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Optional workspace filter
	var workspaceID pgtype.UUID
	if wsID := c.Query("workspace_id"); wsID != "" {
		parsed, err := tableParseUUID(wsID)
		if err == nil {
			workspaceID = parsed
		}
	}

	tables, err := queries.ListCustomTables(ctx, sqlc.ListCustomTablesParams{
		UserID:      user.ID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		slog.Error("Failed to list tables", "error", err, "user_id", user.ID, "workspace_id", workspaceID)
		utils.RespondInternalError(c, slog.Default(), "list tables", nil)
		return
	}

	// Get row counts for each table
	response := make([]TableResponse, len(tables))
	for i, t := range tables {
		rowCount, err := queries.CountCustomRecords(ctx, t.ID)
		if err != nil {
			slog.Warn("Failed to count records for table", "error", err, "table_id", t.ID)
			rowCount = 0 // Use 0 as fallback
		}
		response[i] = tableToResponse(t, rowCount)
	}

	c.JSON(http.StatusOK, response)
}

// GetTable returns a single table with its columns and views
// GET /api/tables/:id
func (h *TableHandler) GetTable(c *gin.Context) {
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

	table, err := queries.GetCustomTable(ctx, sqlc.GetCustomTableParams{
		ID:     tableID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Table")
		return
	}

	// Get columns, views, and row count
	fields, err := queries.ListCustomFields(ctx, tableID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list table fields", err)
		return
	}
	views, err := queries.ListCustomViews(ctx, tableID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list table views", err)
		return
	}
	rowCount, err := queries.CountCustomRecords(ctx, tableID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "count table records", err)
		return
	}

	// Get options for select fields
	fieldResponses := make([]FieldResponse, len(fields))
	for i, f := range fields {
		fieldResponses[i] = fieldToResponse(f)
		// Load options for select fields
		if f.FieldType == "single_select" || f.FieldType == "multi_select" {
			options, _ := queries.ListFieldOptions(ctx, f.ID)
			fieldResponses[i].Options = make([]FieldOptionResponse, len(options))
			for j, o := range options {
				fieldResponses[i].Options[j] = optionToResponse(o)
			}
		}
	}

	viewResponses := make([]ViewResponse, len(views))
	for i, v := range views {
		viewResponses[i] = viewToResponse(v)
	}

	response := tableToResponse(table, rowCount)
	response.Columns = fieldResponses
	response.Views = viewResponses

	c.JSON(http.StatusOK, response)
}

// CreateTable creates a new table
// POST /api/tables
func (h *TableHandler) CreateTable(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req CreateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Parse workspace ID if provided
	var workspaceID pgtype.UUID
	if req.WorkspaceID != nil {
		parsed, err := tableParseUUID(*req.WorkspaceID)
		if err == nil {
			workspaceID = parsed
		}
	}

	table, err := queries.CreateCustomTable(ctx, sqlc.CreateCustomTableParams{
		UserID:      user.ID,
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		Color:       req.Color,
		WorkspaceID: workspaceID,
		Settings:    tableMapToJSONBytes(req.Settings),
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create table", nil)
		return
	}

	// Create default view (Grid)
	_, err = queries.CreateCustomView(ctx, sqlc.CreateCustomViewParams{
		TableID:      table.ID,
		Name:         "Grid View",
		ViewType:     "grid",
		Config:       nil,
		Filters:      nil,
		Sorts:        nil,
		ViewSettings: nil,
		Position:     0,
	})
	if err != nil {
		// Log error but don't fail - table was created successfully
	}

	// Create default primary field (Name)
	_, err = queries.CreateCustomField(ctx, sqlc.CreateCustomFieldParams{
		TableID:      table.ID,
		Name:         "Name",
		FieldType:    "text",
		Position:     0,
		Config:       []byte(`{"is_primary": true}`),
		Required:     tableBoolPtr(true),
		UniqueValues: tableBoolPtr(false),
	})
	if err != nil {
		// Log error but don't fail
	}

	c.JSON(http.StatusCreated, tableToResponse(table, 0))
}

// UpdateTable updates an existing table
// PUT /api/tables/:id
func (h *TableHandler) UpdateTable(c *gin.Context) {
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

	var req UpdateTableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	// Get existing table
	existing, err := queries.GetCustomTable(ctx, sqlc.GetCustomTableParams{
		ID:     tableID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Table")
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
	icon := existing.Icon
	if req.Icon != nil {
		icon = req.Icon
	}
	color := existing.Color
	if req.Color != nil {
		color = req.Color
	}
	settings := existing.Settings
	if req.Settings != nil {
		settings = tableMapToJSONBytes(req.Settings)
	}

	table, err := queries.UpdateCustomTable(ctx, sqlc.UpdateCustomTableParams{
		ID:          tableID,
		Name:        name,
		Description: description,
		Icon:        icon,
		Color:       color,
		Settings:    settings,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update table", nil)
		return
	}

	rowCount, _ := queries.CountCustomRecords(ctx, tableID)
	c.JSON(http.StatusOK, tableToResponse(table, rowCount))
}

// DeleteTable deletes a table and all its data
// DELETE /api/tables/:id
func (h *TableHandler) DeleteTable(c *gin.Context) {
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

	err = queries.DeleteCustomTable(ctx, sqlc.DeleteCustomTableParams{
		ID:     tableID,
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete table", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
