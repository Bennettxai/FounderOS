package handlers

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// ============================================================================
// Helper Functions (table-specific to avoid conflicts)
// ============================================================================

// tableParseUUID parses a string UUID into pgtype.UUID
func tableParseUUID(s string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

// tableUUIDToString converts a pgtype.UUID to string
func tableUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return uuid.UUID(u.Bytes).String()
}

// tableUUIDToPtr converts a pgtype.UUID to a string pointer
func tableUUIDToPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}

// tableJSONBytesToMap converts JSON bytes to a map
func tableJSONBytesToMap(data []byte) map[string]interface{} {
	if data == nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// tableJSONBytesToSlice converts JSON bytes to a slice
func tableJSONBytesToSlice(data []byte) []interface{} {
	if data == nil {
		return nil
	}
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

// tableMapToJSONBytes converts a map to JSON bytes
func tableMapToJSONBytes(m map[string]interface{}) []byte {
	if m == nil {
		return nil
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return data
}

// tableSliceToJSONBytes converts a slice to JSON bytes (nil for empty — SimpleProtocol compatibility)
func tableSliceToJSONBytes(s []interface{}) []byte {
	if s == nil {
		return nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return nil
	}
	return data
}

// tableBoolPtr returns a pointer to a bool
func tableBoolPtr(b bool) *bool {
	return &b
}

// tableGetBool returns the value of a bool pointer or default
func tableGetBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// ============================================================================
// Response Transformers
// ============================================================================

func tableToResponse(t sqlc.CustomTable, rowCount int64) TableResponse {
	return TableResponse{
		ID:          tableUUIDToString(t.ID),
		Name:        t.Name,
		Description: t.Description,
		Icon:        t.Icon,
		Color:       t.Color,
		WorkspaceID: tableUUIDToPtr(t.WorkspaceID),
		Settings:    tableJSONBytesToMap(t.Settings),
		RowCount:    rowCount,
		CreatedAt:   t.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   t.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func fieldToResponse(f sqlc.CustomField) FieldResponse {
	return FieldResponse{
		ID:          tableUUIDToString(f.ID),
		TableID:     tableUUIDToString(f.TableID),
		Name:        f.Name,
		Type:        f.FieldType,
		Description: f.Description,
		Position:    int(f.Position),
		Config:      tableJSONBytesToMap(f.Config),
		Required:    tableGetBool(f.Required),
		Unique:      tableGetBool(f.UniqueValues),
		Hidden:      tableGetBool(f.Hidden),
		CreatedAt:   f.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   f.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func optionToResponse(o sqlc.CustomFieldOption) FieldOptionResponse {
	return FieldOptionResponse{
		ID:       tableUUIDToString(o.ID),
		Name:     o.Name,
		Color:    o.Color,
		Position: int(o.Position),
	}
}

func recordToResponse(r sqlc.CustomRecord) RecordResponse {
	var position *int
	if r.Position != nil {
		p := int(*r.Position)
		position = &p
	}
	return RecordResponse{
		ID:         tableUUIDToString(r.ID),
		TableID:    tableUUIDToString(r.TableID),
		Data:       tableJSONBytesToMap(r.Data),
		Position:   position,
		CreatedBy:  r.CreatedBy,
		ModifiedBy: r.ModifiedBy,
		CreatedAt:  r.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  r.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

func viewToResponse(v sqlc.CustomView) ViewResponse {
	viewType := ""
	if vt, ok := v.ViewType.(string); ok {
		viewType = vt
	}
	return ViewResponse{
		ID:           tableUUIDToString(v.ID),
		TableID:      tableUUIDToString(v.TableID),
		Name:         v.Name,
		Type:         viewType,
		Description:  v.Description,
		IsDefault:    tableGetBool(v.IsDefault),
		Config:       tableJSONBytesToMap(v.Config),
		Filters:      tableJSONBytesToSlice(v.Filters),
		Sorts:        tableJSONBytesToSlice(v.Sorts),
		GroupBy:      tableUUIDToPtr(v.GroupBy),
		ViewSettings: tableJSONBytesToMap(v.ViewSettings),
		Position:     int(v.Position),
		CreatedAt:    v.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    v.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

// ============================================================================
// Route Registration
// ============================================================================

// RegisterTableRoutes registers all table management routes on the given router group.
func RegisterTableRoutes(api *gin.RouterGroup, h *TableHandler, auth gin.HandlerFunc) {
	tables := api.Group("/tables")
	tables.Use(auth, middleware.RequireAuth())
	{
		// Tables CRUD
		tables.GET("", h.ListTables)
		tables.POST("", h.CreateTable)
		tables.GET("/:id", h.GetTable)
		tables.PUT("/:id", h.UpdateTable)
		tables.DELETE("/:id", h.DeleteTable)
		// Columns (Fields) CRUD - matches frontend API naming
		tables.GET("/:id/columns", h.ListFields)
		tables.POST("/:id/columns", h.CreateField)
		tables.PUT("/:id/columns/:columnId", h.UpdateField)
		tables.DELETE("/:id/columns/:columnId", h.DeleteField)
		tables.POST("/:id/columns/reorder", h.ReorderFields)
		// Rows (Records) CRUD - matches frontend API naming
		tables.GET("/:id/rows", h.ListRecords)
		tables.POST("/:id/rows", h.CreateRecord)
		tables.GET("/:id/rows/:rowId", h.GetRecord)
		tables.PUT("/:id/rows/:rowId", h.UpdateRecord)
		tables.DELETE("/:id/rows/:rowId", h.DeleteRecord)
		tables.POST("/:id/rows/bulk-delete", h.BulkDeleteRecords)
		// Views CRUD
		tables.GET("/:id/views", h.ListViews)
		tables.POST("/:id/views", h.CreateView)
		tables.PUT("/:id/views/:viewId", h.UpdateView)
		tables.DELETE("/:id/views/:viewId", h.DeleteView)
	}
}
