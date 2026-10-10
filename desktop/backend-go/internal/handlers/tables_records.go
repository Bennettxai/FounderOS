package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// ============================================================================
// Record API Types
// ============================================================================

// RecordResponse represents a row/record in API responses
type RecordResponse struct {
	ID         string                 `json:"id"`
	TableID    string                 `json:"table_id"`
	Data       map[string]interface{} `json:"data"`
	Position   *int                   `json:"position,omitempty"`
	CreatedBy  *string                `json:"created_by,omitempty"`
	ModifiedBy *string                `json:"modified_by,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// CreateRecordRequest represents the request to create a record
type CreateRecordRequest struct {
	Data map[string]interface{} `json:"data" binding:"required"`
}

// UpdateRecordRequest represents the request to update a record
type UpdateRecordRequest struct {
	Data map[string]interface{} `json:"data" binding:"required"`
}

// ============================================================================
// Record (Row) Handlers
// ============================================================================

// ListRecords returns all records for a table
// GET /api/tables/:id/rows
func (h *TableHandler) ListRecords(c *gin.Context) {
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

	// Parse pagination params (frontend uses page/page_size, convert to limit/offset)
	page := 1
	pageSize := 100
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}
	// Also support limit/offset for backwards compatibility
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			page = (parsed / pageSize) + 1
		}
	}

	limit := int32(pageSize)
	offset := int32((page - 1) * pageSize)

	records, err := queries.ListCustomRecords(ctx, sqlc.ListCustomRecordsParams{
		TableID:   tableID,
		LimitVal:  limit,
		OffsetVal: offset,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list rows", nil)
		return
	}

	response := make([]RecordResponse, len(records))
	for i, r := range records {
		response[i] = recordToResponse(r)
	}

	// Get total count for pagination
	total, _ := queries.CountCustomRecords(ctx, tableID)
	hasMore := int64(page*pageSize) < total

	c.JSON(http.StatusOK, gin.H{
		"rows":      response,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"has_more":  hasMore,
	})
}

// GetRecord returns a single record
// GET /api/tables/:id/records/:recordId
func (h *TableHandler) GetRecord(c *gin.Context) {
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

	recordID, err := tableParseUUID(c.Param("rowId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "row_id")
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

	record, err := queries.GetCustomRecord(ctx, recordID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Record")
		return
	}

	// Verify record belongs to table
	if tableUUIDToString(record.TableID) != tableUUIDToString(tableID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Record does not belong to this table"})
		return
	}

	c.JSON(http.StatusOK, recordToResponse(record))
}

// CreateRecord creates a new record in a table
// POST /api/tables/:id/records
func (h *TableHandler) CreateRecord(c *gin.Context) {
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

	var req CreateRecordRequest
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

	record, err := queries.CreateCustomRecord(ctx, sqlc.CreateCustomRecordParams{
		TableID:   tableID,
		Data:      tableMapToJSONBytes(req.Data),
		CreatedBy: &user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create record", nil)
		return
	}

	c.JSON(http.StatusCreated, recordToResponse(record))
}

// UpdateRecord updates an existing record
// PUT /api/tables/:id/records/:recordId
func (h *TableHandler) UpdateRecord(c *gin.Context) {
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

	recordID, err := tableParseUUID(c.Param("rowId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "row_id")
		return
	}

	var req UpdateRecordRequest
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

	// Verify record belongs to table
	existing, err := queries.GetCustomRecord(ctx, recordID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Record")
		return
	}
	if tableUUIDToString(existing.TableID) != tableUUIDToString(tableID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Record does not belong to this table"})
		return
	}

	record, err := queries.UpdateCustomRecord(ctx, sqlc.UpdateCustomRecordParams{
		ID:         recordID,
		Data:       tableMapToJSONBytes(req.Data),
		ModifiedBy: &user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update record", nil)
		return
	}

	c.JSON(http.StatusOK, recordToResponse(record))
}

// UpdateRecordField updates a single field in a record
// PATCH /api/tables/:id/records/:recordId/fields/:fieldId
func (h *TableHandler) UpdateRecordField(c *gin.Context) {
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

	recordID, err := tableParseUUID(c.Param("rowId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "row_id")
		return
	}

	fieldID := c.Param("fieldId")

	var req struct {
		Value interface{} `json:"value"`
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

	// Verify record belongs to table
	existing, err := queries.GetCustomRecord(ctx, recordID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Record")
		return
	}
	if tableUUIDToString(existing.TableID) != tableUUIDToString(tableID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Record does not belong to this table"})
		return
	}

	// Marshal value to JSON
	valueJSON, err := json.Marshal(req.Value)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid value"})
		return
	}

	record, err := queries.UpdateCustomRecordField(ctx, sqlc.UpdateCustomRecordFieldParams{
		ID:         recordID,
		Column2:    fieldID,
		Column3:    valueJSON,
		ModifiedBy: &user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update record field", nil)
		return
	}

	c.JSON(http.StatusOK, recordToResponse(record))
}

// DeleteRecord deletes a record
// DELETE /api/tables/:id/records/:recordId
func (h *TableHandler) DeleteRecord(c *gin.Context) {
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

	recordID, err := tableParseUUID(c.Param("rowId"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "row_id")
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

	// Verify record belongs to table
	existing, err := queries.GetCustomRecord(ctx, recordID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Record")
		return
	}
	if tableUUIDToString(existing.TableID) != tableUUIDToString(tableID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Record does not belong to this table"})
		return
	}

	err = queries.DeleteCustomRecord(ctx, recordID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete record", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// BulkDeleteRecords deletes multiple records
// POST /api/tables/:id/records/bulk-delete
func (h *TableHandler) BulkDeleteRecords(c *gin.Context) {
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
		RowIDs []string `json:"row_ids" binding:"required"`
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

	// Delete each row
	deleted := 0
	for _, rid := range req.RowIDs {
		recordID, err := tableParseUUID(rid)
		if err != nil {
			continue
		}
		// Verify record belongs to table before deleting
		record, err := queries.GetCustomRecord(ctx, recordID)
		if err != nil {
			continue
		}
		if tableUUIDToString(record.TableID) != tableUUIDToString(tableID) {
			continue
		}
		if err := queries.DeleteCustomRecord(ctx, recordID); err == nil {
			deleted++
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "deleted": deleted})
}
