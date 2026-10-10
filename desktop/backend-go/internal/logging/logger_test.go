package logging

import (
	"strings"
	"testing"
)

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal email",
			input:    "john.doe@example.com",
			expected: "j***@example.com",
		},
		{
			name:     "Single char email",
			input:    "a@example.com",
			expected: "a***@example.com",
		},
		{
			name:     "Empty email",
			input:    "",
			expected: "",
		},
		{
			name:     "Invalid email",
			input:    "notanemail",
			expected: "***@***",
		},
		{
			name:     "Long email",
			input:    "very.long.email.address@corporate.example.com",
			expected: "v***@corporate.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskEmail(tt.input)
			if result != tt.expected {
				t.Errorf("MaskEmail(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // What the output should contain
	}{
		{
			name:     "JWT token",
			input:    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			contains: "eyJ***[JWT_REDACTED]",
		},
		{
			name:     "Bearer token",
			input:    "Bearer abc123def456ghi789",
			contains: "Bearer ***[TOKEN_REDACTED]",
		},
		{
			name:     "Long token",
			input:    "1234567890abcdefghijklmnop",
			contains: "1234***[TOKEN_REDACTED]",
		},
		{
			name:     "Short token",
			input:    "abc123",
			contains: "***[REDACTED]",
		},
		{
			name:     "Empty token",
			input:    "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskToken(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("MaskToken(%q) = %q, should contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestDetectAndRedactSecrets(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldDetect  bool
		shouldContain string
	}{
		{
			name:          "AWS access key",
			input:         "AKIAIOSFODNN7EXAMPLE",
			shouldDetect:  true,
			shouldContain: "[SECRET_REDACTED]",
		},
		{
			name:          "GitHub PAT",
			input:         "ghp_1234567890abcdefghijklmnopqrstuvwx",
			shouldDetect:  true,
			shouldContain: "[SECRET_REDACTED]",
		},
		{
			name:          "JWT in text",
			input:         "User logged in with token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.payload.signature",
			shouldDetect:  true,
			shouldContain: "[SECRET_REDACTED]",
		},
		{
			name:          "API key in assignment",
			input:         "api_key = 'sk-1234567890abcdefghijklmnopqrstuvwxyz'",
			shouldDetect:  true,
			shouldContain: "[SECRET_REDACTED]",
		},
		{
			name:          "Normal text",
			input:         "This is just normal text without secrets",
			shouldDetect:  false,
			shouldContain: "normal text",
		},
		{
			name:          "Private key header",
			input:         "-----BEGIN RSA PRIVATE KEY-----",
			shouldDetect:  true,
			shouldContain: "[SECRET_REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized, detected := DetectAndRedactSecrets(tt.input)

			if detected != tt.shouldDetect {
				t.Errorf("DetectAndRedactSecrets(%q) detected = %v, want %v", tt.input, detected, tt.shouldDetect)
			}

			if !strings.Contains(sanitized, tt.shouldContain) {
				t.Errorf("DetectAndRedactSecrets(%q) = %q, should contain %q", tt.input, sanitized, tt.shouldContain)
			}
		})
	}
}

func TestSanitizeSQL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "String literal",
			input:    "SELECT * FROM users WHERE email = 'test@example.com'",
			contains: "[REDACTED]",
		},
		{
			name:     "Token in WHERE",
			input:    "DELETE FROM sessions WHERE token = 'abc123def456'",
			contains: "[REDACTED]",
		},
		{
			name:     "Safe query",
			input:    "SELECT COUNT(*) FROM users",
			contains: "SELECT COUNT(*)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSQL(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("SanitizeSQL(%q) = %q, should contain %q", tt.input, result, tt.contains)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with query params",
			input:    "https://api.example.com/users?token=abc123&id=456",
			expected: "https://api.example.com/users?[PARAMS_REDACTED]",
		},
		{
			name:     "URL with session in path",
			input:    "https://api.example.com/session/abc-123-def/data",
			expected: "https://api.example.com/session/[REDACTED]/data",
		},
		{
			name:     "Safe URL",
			input:    "https://api.example.com/public/data",
			expected: "https://api.example.com/public/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeCookies(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "Session cookie",
			input:    "session_token=abc123def456; path=/",
			contains: []string{"session_token=[REDACTED]"},
		},
		{
			name:     "Multiple cookies",
			input:    "session=abc; user_id=123; tracking=xyz",
			contains: []string{"session=[REDACTED]", "user_id=[REDACTED]", "tracking=[REDACTED]"},
		},
		{
			name:     "Empty cookies",
			input:    "",
			contains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeCookies(tt.input)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("SanitizeCookies(%q) = %q, should contain %q", tt.input, result, expected)
				}
			}
		})
	}
}

func TestStructuredLog(t *testing.T) {
	fields := map[string]interface{}{
		"user_id":  "user-123",
		"action":   "login",
		"password": "secret", // Should be redacted
	}

	log := NewStructuredLog(LevelInfo, "User action", fields)

	// Test JSON output
	jsonStr := log.JSON()

	if !strings.Contains(jsonStr, "User action") {
		t.Error("JSON should contain message")
	}

	if !strings.Contains(jsonStr, "INFO") {
		t.Error("JSON should contain log level")
	}

	if strings.Contains(jsonStr, "secret") {
		t.Error("JSON should not contain raw password")
	}

	if !strings.Contains(jsonStr, "[REDACTED]") {
		t.Error("JSON should contain redacted password")
	}
}

func TestMaskUserID(t *testing.T) {
	userID := "user-abc-def-123-456"
	masked := MaskUserID(userID)

	// Should not show full ID
	if masked == userID {
		t.Error("MaskUserID should not return full user ID")
	}

	// Should contain asterisks
	if !strings.Contains(masked, "*") {
		t.Error("MaskUserID should contain asterisks")
	}
}

// Benchmark tests
func BenchmarkMaskEmail(b *testing.B) {
	email := "john.doe@example.com"
	for i := 0; i < b.N; i++ {
		MaskEmail(email)
	}
}

func BenchmarkDetectAndRedactSecrets(b *testing.B) {
	text := "API key is api_key=sk-1234567890abcdefghijklmnopqrstuvwxyz in this log"
	for i := 0; i < b.N; i++ {
		DetectAndRedactSecrets(text)
	}
}

func BenchmarkSanitizeURL(b *testing.B) {
	url := "https://api.example.com/session/abc-123-def/data?token=xyz"
	for i := 0; i < b.N; i++ {
		SanitizeURL(url)
	}
}
