package handlers

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// Field API Types
// ============================================================================

// FieldResponse represents a column/field in API responses
type FieldResponse struct {
	ID          string                 `json:"id"`
	TableID     string                 `json:"table_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description *string                `json:"description,omitempty"`
	Position    int                    `json:"position"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Required    bool                   `json:"required"`
	Unique      bool                   `json:"unique"`
	Hidden      bool                   `json:"hidden"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	// Options for select fields
	Options []FieldOptionResponse `json:"options,omitempty"`
}

// CreateFieldRequest represents the request to create a field
type CreateFieldRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Description *string                `json:"description,omitempty"`
	Position    *int                   `json:"position,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Required    *bool                  `json:"required,omitempty"`
	Unique      *bool                  `json:"unique,omitempty"`
	// Options for select fields (single_select, multi_select)
	Options []CreateFieldOptionRequest `json:"options,omitempty"`
}

// UpdateFieldRequest represents the request to update a field
type UpdateFieldRequest struct {
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Required    *bool                  `json:"required,omitempty"`
	Hidden      *bool                  `json:"hidden,omitempty"`
}

// FieldOptionResponse represents a select option in API responses
type FieldOptionResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Color    *string `json:"color,omitempty"`
	Position int     `json:"position"`
}

// CreateFieldOptionRequest represents the request to create a field option
type CreateFieldOptionRequest struct {
	Name     string  `json:"name" binding:"required"`
	Color    *string `json:"color,omitempty"`
	Position *int    `json:"position,omitempty"`
}

// ============================================================================
// Field (Column) Handlers
// ============================================================================

// ListFields returns all fields for a table
// GET /api/tables/:id/fields
func (h *TableHandler) ListFields(c *gin.Context) {
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

	fields, err := queries.ListCustomFields(ctx, tableID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list fields", nil)
		return
	}

	response := make([]FieldResponse, len(fields))
	for i, f := range fields {
		response[i] = fieldToResponse(f)
		// Load options for select fields
		if f.FieldType == "single_select" || f.FieldType == "multi_select" {
			options, _ := queries.ListFieldOptions(ctx, f.ID)
			response[i].Options = make([]FieldOptionResponse, len(options))
			for j, o := range options {
				response[i].Options[j] = optionToResponse(o)
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

// CreateField creates a new field in a table
// POST /api/tables/:id/fields
func (h *TableHandler) CreateField(c *gin.Context) {
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

	var req CreateFieldRequest
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
		existingFields, _ := queries.ListCustomFields(ctx, tableID)
		position = int32(len(existingFields))
	}

	field, err := queries.CreateCustomField(ctx, sqlc.CreateCustomFieldParams{
		TableID:      tableID,
		Name:         req.Name,
		FieldType:    req.Type,
		Description:  req.Description,
		Position:     position,
		Config:       tableMapToJSONBytes(req.Config),
		Required:     req.Required,
		UniqueValues: req.Unique,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create field", nil)
		return
	}

	// Create options for select fields
	response := fieldToResponse(field)
	if (req.Type == "single_select" || req.Type == "multi_select") && len(req.Options) > 0 {
		response.Options = make([]FieldOptionResponse, len(req.Options))
		for i, opt := range req.Options {
			pos := int32(i)
			if opt.Position != nil {
				pos = int32(*opt.Position)
			}
			option, err := queries.CreateFieldOption(ctx, sqlc.CreateFieldOptionParams{
				FieldID:  field.ID,
				Name:     opt.Name,
				Color:    opt.Color,
				Position: pos,
			})
			if err == nil {
				response.Options[i] = optionToResponse(option)
			}
		}
	}

	c.JSON(http.StatusCreated, response)
}

// UpdateField updates an existing field
// PUT /api/tables/:tableId/fields/:fieldId
func (h *TableHandler) UpdateField(c *gin.Context) {
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

	fieldID, err := tableParseUUID(c.Param("columnId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "column_id")
		return
	}

	var req UpdateFieldRequest
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

	// Get existing field
	existing, err := queries.GetCustomField(ctx, fieldID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Field")
		return
	}

	// Verify field belongs to table
	if tableUUIDToString(existing.TableID) != tableUUIDToString(tableID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Field does not belong to this table"})
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
	required := existing.Required
	if req.Required != nil {
		required = req.Required
	}
	hidden := existing.Hidden
	if req.Hidden != nil {
		hidden = req.Hidden
	}

	field, err := queries.UpdateCustomField(ctx, sqlc.UpdateCustomFieldParams{
		ID:          fieldID,
		Name:        name,
		Description: description,
		Config:      config,
		Required:    required,
		Hidden:      hidden,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update field", nil)
		return
	}

	c.JSON(http.StatusOK, fieldToResponse(field))
}

// DeleteField deletes a field from a table
// DELETE /api/tables/:tableId/fields/:fieldId
func (h *TableHandler) DeleteField(c *gin.Context) {
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

	fieldID, err := tableParseUUID(c.Param("columnId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "column_id")
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

	// Verify field belongs to table
	existing, err := queries.GetCustomField(ctx, fieldID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Field")
		return
	}
	if tableUUIDToString(existing.TableID) != tableUUIDToString(tableID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Field does not belong to this table"})
		return
	}

	err = queries.DeleteCustomField(ctx, fieldID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete field", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ReorderFields reorders fields in a table
// POST /api/tables/:id/fields/reorder
func (h *TableHandler) ReorderFields(c *gin.Context) {
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

	var req struct {
		ColumnIDs []string `json:"column_ids" binding:"required"`
	}
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

	// Update positions
	for i, fid := range req.ColumnIDs {
		fieldID, err := tableParseUUID(fid)
		if err != nil {
			continue
		}
		queries.UpdateCustomFieldPosition(ctx, sqlc.UpdateCustomFieldPositionParams{
			ID:       fieldID,
			Position: int32(i),
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
