package security

import (
	"testing"
)

func TestSanitizeForJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantNot  []string // strings that should NOT appear in output
	}{
		{
			name:    "Event handler in img tag",
			input:   "<img src=x onerror=alert(1)>",
			wantNot: []string{"onerror=", "<img"},
		},
		{
			name:    "JavaScript protocol in URL",
			input:   "javascript:void(0)",
			wantNot: []string{"javascript:"},
		},
		{
			name:    "JavaScript protocol in iframe",
			input:   "<iframe src=javascript:alert(1)>",
			wantNot: []string{"javascript:", "<iframe"},
		},
		{
			name:    "Script tag",
			input:   "<script>alert('xss')</script>",
			wantNot: []string{"<script"},
		},
		{
			name:    "SVG with onload",
			input:   "<svg onload=alert('xss')>",
			wantNot: []string{"onload=", "<svg"},
		},
		{
			name:    "Data URL protocol",
			input:   "data:text/html,<script>alert(1)</script>",
			wantNot: []string{"data:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeForJSON(tt.input)

			// Verify dangerous patterns are removed/escaped
			for _, pattern := range tt.wantNot {
				if contains(got, pattern) {
					t.Errorf("SanitizeForJSON() = %q, should not contain %q", got, pattern)
				}
			}

			// Verify output is different from input (sanitization occurred)
			if got == tt.input {
				t.Errorf("SanitizeForJSON() = %q, should be different from input %q", got, tt.input)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "JavaScript protocol",
			input: "javascript:alert(1)",
			want:  "",
		},
		{
			name:  "Data protocol",
			input: "data:text/html,<script>",
			want:  "",
		},
		{
			name:  "Safe HTTP URL",
			input: "https://example.com",
			want:  "https://example.com",
		},
		{
			name:  "Safe relative URL",
			input: "/path/to/resource",
			want:  "/path/to/resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeURL(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContainsXSSPattern(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "Script tag",
			input: "<script>alert(1)</script>",
			want:  true,
		},
		{
			name:  "JavaScript protocol",
			input: "javascript:void(0)",
			want:  true,
		},
		{
			name:  "Event handler",
			input: "<img onerror=alert(1)>",
			want:  true,
		},
		{
			name:  "Safe content",
			input: "Hello world",
			want:  false,
		},
		{
			name:  "Safe HTML",
			input: "<p>This is safe</p>",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsXSSPattern(tt.input)
			if got != tt.want {
				t.Errorf("ContainsXSSPattern() = %v, want %v for input %q", got, tt.want, tt.input)
			}
		})
	}
}

// Helper function to check if string contains substring (case-insensitive would be better)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (
		s[:len(substr)] == substr ||
		s[len(s)-len(substr):] == substr ||
		containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
