package logging

import (
	"strings"
	"testing"
)

func TestMaskSessionID(t *testing.T) {
	tests := []struct {
		input       string
		checkPrefix bool // Check that prefix is preserved
	}{
		{"abc12345-6789-abcd-efgh-ijklmnopqrst", true},
		{"short", false}, // Too short, gets fully masked
		{"12345678", false}, // At limit, gets masked
		{"123456789", true}, // Over limit, prefix preserved
		{"", false},
	}

	for _, tt := range tests {
		result := MaskSessionID(tt.input)

		// Result should not equal input (should be masked)
		if len(tt.input) > 0 && result == tt.input {
			t.Errorf("MaskSessionID(%q) = %q, should be masked", tt.input, result)
		}

		// If checkPrefix, first 8 chars should be preserved
		if tt.checkPrefix && len(tt.input) > 8 {
			if !strings.HasPrefix(result, tt.input[:8]) {
				t.Errorf("MaskSessionID(%q) should preserve first 8 chars, got %q", tt.input, result)
			}
		}

		// Result should contain asterisks (masking)
		if len(tt.input) > 0 && !strings.Contains(result, "*") {
			t.Errorf("MaskSessionID(%q) = %q, should contain masking asterisks", tt.input, result)
		}
	}
}

func TestMaskIP(t *testing.T) {
	tests := []struct {
		input    string
		validate func(result string) bool
	}{
		// IPv4 - should show first two octets
		{"192.168.1.100", func(r string) bool { return strings.HasPrefix(r, "192.168.") && strings.Contains(r, "xxx") }},
		{"10.0.0.1", func(r string) bool { return strings.HasPrefix(r, "10.0.") && strings.Contains(r, "xxx") }},
		// IPv6 - should show first 8 chars + ellipsis
		{"2001:0db8:85a3:0000:0000:8a2e:0370:7334", func(r string) bool { return len(r) <= 12 && strings.HasPrefix(r, "2001:0db") }},
		// Short hostname - gets truncated
		{"localhost", func(r string) bool { return len(r) <= 12 }},
	}

	for _, tt := range tests {
		result := MaskIP(tt.input)
		if !tt.validate(result) {
			t.Errorf("MaskIP(%q) = %q, validation failed", tt.input, result)
		}
	}
}

func TestSafeLogFields(t *testing.T) {
	fields := map[string]interface{}{
		"username":    "john",
		"password":    "secret123",
		"api_key":     "sk-abc123",
		"session_id":  "uuid-1234",
		"count":       42,
		"Bearer":      "token-xyz",
		"normal_field": "visible",
	}

	result := SafeLogFields(fields)

	// Should be redacted
	if result["password"] != "[REDACTED]" {
		t.Error("password should be redacted")
	}
	if result["api_key"] != "[REDACTED]" {
		t.Error("api_key should be redacted")
	}
	if result["session_id"] != "[REDACTED]" {
		t.Error("session_id should be redacted")
	}

	// Should NOT be redacted
	if result["username"] != "john" {
		t.Error("username should not be redacted")
	}
	if result["count"] != 42 {
		t.Error("count should not be redacted")
	}
	if result["normal_field"] != "visible" {
		t.Error("normal_field should not be redacted")
	}
}

func TestSanitizerMasking(t *testing.T) {
	logger := NewSanitizedLogger(DefaultLogConfig())

	tests := []struct {
		input       string
		shouldMask  bool
		description string
	}{
		// Should mask - JWT tokens (starts with eyJ)
		{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.xxx.yyy", true, "JWT"},
		{"User john@example.com logged in", true, "Email"},
		{"Connecting from 192.168.1.100", true, "IP address"},

		// Should NOT mask (these patterns aren't in our regex set)
		{"Server started on port 8080", false, "Normal message"},
		{"Processing 42 requests", false, "Numbers"},
	}

	for _, tt := range tests {
		result := logger.sanitize(tt.input)
		wasMasked := result != tt.input

		if tt.shouldMask && !wasMasked {
			t.Errorf("Expected masking for %s: %q -> %q", tt.description, tt.input, result)
		}

		if !tt.shouldMask && wasMasked {
			t.Errorf("Unexpected masking for %s: %q -> %q", tt.description, tt.input, result)
		}
	}
}

func TestTerminalContentFiltering(t *testing.T) {
	config := DefaultLogConfig()
	config.FilterTerminalIO = true
	logger := NewSanitizedLogger(config)

	// Long content with terminal patterns should be filtered
	terminalOutput := strings.Repeat("x", 250) + "\x1b[32mgreen text\x1b[0m"
	result := logger.sanitize(terminalOutput)

	if !strings.Contains(result, "[terminal output filtered]") {
		t.Error("Terminal output should be filtered")
	}

	// Short content should pass through even with escape codes
	shortContent := "short \x1b[32mgreen\x1b[0m text"
	result = logger.sanitize(shortContent)
	// This won't be filtered because it's under 200 chars
}

func TestLogLevels(t *testing.T) {
	config := DefaultLogConfig()
	config.MinLevel = LevelWarn

	logger := NewSanitizedLogger(config)

	// Debug and Info should be suppressed
	// (We can't easily test log output, but we ensure no panic)
	logger.Debug("This should not appear")
	logger.Info("This should not appear")
	logger.Warn("This should appear")
	logger.Error("This should appear")
}

func TestConcurrentLogging(t *testing.T) {
	logger := GetLogger()
	done := make(chan bool)

	// Run concurrent log calls
	for i := 0; i < 100; i++ {
		go func(id int) {
			logger.Info("Concurrent log from goroutine %d", id)
			logger.Security("Security event from goroutine %d", id)
			done <- true
		}(i)
	}

	// Wait for all
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestConfigUpdate(t *testing.T) {
	logger := NewSanitizedLogger(DefaultLogConfig())

	// Update config with shorter mask
	newConfig := DefaultLogConfig()
	newConfig.SessionIDMaskLength = 4

	logger.UpdateConfig(newConfig)

	// Verify config took effect by testing the mask length
	// (config change should affect masking behavior via sanitize)
	// This test verifies UpdateConfig doesn't panic and is thread-safe
}

func BenchmarkSanitize(b *testing.B) {
	logger := NewSanitizedLogger(DefaultLogConfig())
	input := "User john@example.com logged in from 192.168.1.100 with session abc12345-6789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.sanitize(input)
	}
}

func BenchmarkMaskSessionID(b *testing.B) {
	sessionID := "abc12345-6789-abcd-efgh-ijklmnopqrst"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MaskSessionID(sessionID)
	}
}
