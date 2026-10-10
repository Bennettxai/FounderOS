package handlers

import (
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// CURSOR PAGINATION
// =============================================================================
// Cursor-based pagination is better for mobile because:
// 1. Works correctly when data changes between requests (offset can skip/duplicate)
// 2. More efficient for large datasets (no COUNT query needed)
// 3. Consistent performance regardless of page number
//
// The cursor encodes the last item's ID and timestamp, allowing the next query
// to efficiently fetch items after that point.
// =============================================================================

// PaginationCursor represents the encoded pagination state
type PaginationCursor struct {
	ID        uuid.UUID `json:"id"`
	UpdatedAt int64     `json:"t"` // Unix timestamp for compact encoding
}

// EncodeCursor creates a base64-encoded cursor from ID and timestamp
// Example output: "eyJpZCI6IjEyMzQ1Njc4LTEyMzQtMTIzNC0xMjM0LTEyMzQ1Njc4OTBhYiIsInQiOjE3MDQzODQwMDB9"
func EncodeCursor(id uuid.UUID, updatedAt time.Time) string {
	cursor := PaginationCursor{
		ID:        id,
		UpdatedAt: updatedAt.Unix(),
	}
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor parses a base64 cursor back to ID and timestamp
// Returns zero values and error if cursor is invalid
func DecodeCursor(cursor string) (uuid.UUID, time.Time, error) {
	if cursor == "" {
		return uuid.Nil, time.Time{}, nil
	}

	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return uuid.Nil, time.Time{}, err
	}

	var parsed PaginationCursor
	if err := json.Unmarshal(data, &parsed); err != nil {
		return uuid.Nil, time.Time{}, err
	}

	return parsed.ID, time.Unix(parsed.UpdatedAt, 0), nil
}

// =============================================================================
// FIELD SELECTION
// =============================================================================
// Field selection allows clients to request only the fields they need:
//   GET /tasks?fields=id,title,status
//
// This reduces bandwidth significantly on mobile:
// - Full task: ~2KB
// - Selected fields: ~100 bytes
//
// The implementation uses reflection to dynamically filter struct fields.
// =============================================================================

// ParseFieldsParam parses a comma-separated fields string into a slice
// Example: "id,title,status" -> ["id", "title", "status"]
func ParseFieldsParam(fieldsParam string) []string {
	if fieldsParam == "" {
		return nil // nil means "all fields"
	}

	fields := strings.Split(fieldsParam, ",")
	result := make([]string, 0, len(fields))

	for _, f := range fields {
		trimmed := strings.TrimSpace(f)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// SelectFields filters a struct/map to only include requested fields
// If fields is nil or empty, returns the original data unchanged
//
// For structs: uses json tags to match field names
// For maps: uses map keys directly
func SelectFields(data interface{}, fields []string) interface{} {
	if len(fields) == 0 {
		return data
	}

	// Create a set for O(1) lookup
	fieldSet := make(map[string]bool, len(fields))
	for _, f := range fields {
		fieldSet[strings.ToLower(f)] = true
	}

	return selectFieldsRecursive(data, fieldSet)
}

// selectFieldsRecursive handles the actual field selection logic
func selectFieldsRecursive(data interface{}, fieldSet map[string]bool) interface{} {
	if data == nil {
		return nil
	}

	val := reflect.ValueOf(data)

	// Handle pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Struct:
		return selectFromStruct(val, fieldSet)
	case reflect.Map:
		return selectFromMap(val, fieldSet)
	case reflect.Slice:
		return selectFromSlice(val, fieldSet)
	default:
		return data
	}
}

// selectFromStruct filters struct fields based on json tags
func selectFromStruct(val reflect.Value, fieldSet map[string]bool) map[string]interface{} {
	result := make(map[string]interface{})
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)

		// Get the json tag name
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Parse json tag (handle "name,omitempty" format)
		jsonName := strings.Split(jsonTag, ",")[0]

		// Check if this field is requested
		if fieldSet[strings.ToLower(jsonName)] {
			fieldVal := val.Field(i)

			// Handle nil pointers
			if fieldVal.Kind() == reflect.Ptr && fieldVal.IsNil() {
				result[jsonName] = nil
				continue
			}

			result[jsonName] = fieldVal.Interface()
		}
	}

	return result
}

// selectFromMap filters map entries based on keys
func selectFromMap(val reflect.Value, fieldSet map[string]bool) map[string]interface{} {
	result := make(map[string]interface{})

	for _, key := range val.MapKeys() {
		keyStr := key.String()
		if fieldSet[strings.ToLower(keyStr)] {
			result[keyStr] = val.MapIndex(key).Interface()
		}
	}

	return result
}

// selectFromSlice applies field selection to each element in a slice
func selectFromSlice(val reflect.Value, fieldSet map[string]bool) []interface{} {
	result := make([]interface{}, val.Len())

	for i := 0; i < val.Len(); i++ {
		result[i] = selectFieldsRecursive(val.Index(i).Interface(), fieldSet)
	}

	return result
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// ClampInt ensures a value is within min/max bounds
// Used for pagination limits: ClampInt(limit, 1, 50)
func ClampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// TimeToUnix converts time.Time to Unix timestamp (int64)
// Used for lean mobile responses where timestamps are integers
func TimeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

// TimeToUnixPtr converts *time.Time to *int64
// Returns nil if input is nil
func TimeToUnixPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	unix := t.Unix()
	return &unix
}

// FormatDateOnly formats time as "2006-01-02" string (date only, no time)
// Used for due dates in mobile responses
func FormatDateOnly(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatDateOnlyPtr formats *time.Time as *string, returns nil if nil
func FormatDateOnlyPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}
