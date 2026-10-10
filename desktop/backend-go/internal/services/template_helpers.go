package services

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
)

// GetTemplatesDirectory returns the absolute path to the OSA templates directory
func GetTemplatesDirectory() string {
	// Get the current file's directory
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "internal/prompts/templates/osa"
	}

	// Navigate from internal/services to internal/prompts/templates/osa
	servicesDir := filepath.Dir(filename)
	internalDir := filepath.Dir(servicesDir)
	templatesDir := filepath.Join(internalDir, "prompts", "templates", "osa")

	return templatesDir
}

// ValidateVariableType checks if a variable value matches the expected type
func ValidateVariableType(value interface{}, expectedType string) error {
	if value == nil {
		return nil // nil is valid for any type
	}

	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected type 'string', got '%T'", value)
		}

	case "array":
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
			return fmt.Errorf("expected type 'array', got '%T'", value)
		}

	case "object":
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Map && rv.Kind() != reflect.Struct {
			return fmt.Errorf("expected type 'object', got '%T'", value)
		}

	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			// Valid number type
		default:
			return fmt.Errorf("expected type 'number', got '%T'", value)
		}

	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected type 'boolean', got '%T'", value)
		}

	default:
		return fmt.Errorf("unknown type '%s'", expectedType)
	}

	return nil
}

// CoerceToString attempts to convert a value to string for template rendering
func CoerceToString(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}
