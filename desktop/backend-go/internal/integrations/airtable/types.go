// Package airtable provides the Airtable integration (Bases, Tables, Records).
package airtable

// ============================================================================
// Airtable API Types
// ============================================================================

// Base represents an Airtable base
type Base struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PermissionLevel string `json:"permissionLevel"` // none, read, comment, edit, create
}

// Table represents an Airtable table within a base
type Table struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	PrimaryFieldID string  `json:"primaryFieldId"`
	Fields         []Field `json:"fields"`
	Views          []View  `json:"views"`
}

// Field represents a field (column) in an Airtable table
type Field struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// View represents a view in an Airtable table
type View struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // grid, form, calendar, gallery, kanban
}

// Record represents a record (row) in an Airtable table
type Record struct {
	ID          string                 `json:"id"`
	CreatedTime string                 `json:"createdTime"`
	Fields      map[string]interface{} `json:"fields"`
}

// RecordList represents a paginated list of records
type RecordList struct {
	Records []Record `json:"records"`
	Offset  string   `json:"offset,omitempty"` // For pagination
}

// ============================================================================
// Field Type Constants
// ============================================================================

// FieldTypes constants
const (
	FieldTypeSingleLineText      = "singleLineText"
	FieldTypeMultilineText       = "multilineText"
	FieldTypeEmail               = "email"
	FieldTypeURL                 = "url"
	FieldTypeNumber              = "number"
	FieldTypeCurrency            = "currency"
	FieldTypePercent             = "percent"
	FieldTypeDuration            = "duration"
	FieldTypeCheckbox            = "checkbox"
	FieldTypeSingleSelect        = "singleSelect"
	FieldTypeMultipleSelects     = "multipleSelects"
	FieldTypeDate                = "date"
	FieldTypeDateTime            = "dateTime"
	FieldTypePhoneNumber         = "phoneNumber"
	FieldTypeMultipleAttachments = "multipleAttachments"
	FieldTypeMultipleRecordLinks = "multipleRecordLinks"
	FieldTypeRating              = "rating"
	FieldTypeRichText            = "richText"
	FieldTypeFormula             = "formula"
	FieldTypeRollup              = "rollup"
	FieldTypeCount               = "count"
	FieldTypeLookup              = "lookup"
	FieldTypeCreatedTime         = "createdTime"
	FieldTypeLastModifiedTime    = "lastModifiedTime"
	FieldTypeCreatedBy           = "createdBy"
	FieldTypeLastModifiedBy      = "lastModifiedBy"
	FieldTypeAutoNumber          = "autoNumber"
	FieldTypeBarcode             = "barcode"
	FieldTypeButton              = "button"
)

// ============================================================================
// Internal Types
// ============================================================================

// airtableUser represents user information from Airtable
type airtableUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}
