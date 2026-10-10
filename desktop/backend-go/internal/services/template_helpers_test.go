package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTemplatesDirectory(t *testing.T) {
	dir := GetTemplatesDirectory()

	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "prompts")
	assert.Contains(t, dir, "templates")
	assert.Contains(t, dir, "osa")
}

func TestValidateVariableType_String(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectError bool
	}{
		{"valid string", "test", false},
		{"invalid int", 123, true},
		{"invalid bool", true, true},
		{"invalid array", []string{"a"}, true},
		{"nil is valid", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariableType(tt.value, "string")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVariableType_Array(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectError bool
	}{
		{"valid slice", []string{"a", "b"}, false},
		{"valid array", [3]int{1, 2, 3}, false},
		{"invalid string", "not-array", true},
		{"invalid int", 123, true},
		{"nil is valid", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariableType(tt.value, "array")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVariableType_Object(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectError bool
	}{
		{"valid map", map[string]interface{}{"key": "value"}, false},
		{"valid struct", struct{ Name string }{Name: "test"}, false},
		{"invalid string", "not-object", true},
		{"invalid int", 123, true},
		{"nil is valid", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariableType(tt.value, "object")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVariableType_Number(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectError bool
	}{
		{"valid int", 123, false},
		{"valid int32", int32(123), false},
		{"valid int64", int64(123), false},
		{"valid float32", float32(123.45), false},
		{"valid float64", float64(123.45), false},
		{"valid uint", uint(123), false},
		{"invalid string", "123", true},
		{"invalid bool", true, true},
		{"nil is valid", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariableType(tt.value, "number")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVariableType_Boolean(t *testing.T) {
	tests := []struct {
		name        string
		value       interface{}
		expectError bool
	}{
		{"valid true", true, false},
		{"valid false", false, false},
		{"invalid string", "true", true},
		{"invalid int", 1, true},
		{"nil is valid", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVariableType(tt.value, "boolean")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVariableType_UnknownType(t *testing.T) {
	err := ValidateVariableType("test", "unknown-type")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

func TestCoerceToString(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "test", "test"},
		{"int", 123, "123"},
		{"float", 123.45, "123.45"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, ""},
		{"slice", []string{"a", "b"}, "[a b]"},
		{"map", map[string]int{"a": 1}, "map[a:1]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CoerceToString(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}
