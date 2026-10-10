package utils

// StringPtr returns a pointer to the string value.
// This is useful when you need to pass a string literal to a function that expects *string.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the int value.
// This is useful when you need to pass an int literal to a function that expects *int.
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr returns a pointer to the int64 value.
// This is useful when you need to pass an int64 literal to a function that expects *int64.
func Int64Ptr(i int64) *int64 {
	return &i
}

// BoolPtr returns a pointer to the bool value.
// This is useful when you need to pass a bool literal to a function that expects *bool.
func BoolPtr(b bool) *bool {
	return &b
}

// Float64Ptr returns a pointer to the float64 value.
// This is useful when you need to pass a float64 literal to a function that expects *float64.
func Float64Ptr(f float64) *float64 {
	return &f
}
