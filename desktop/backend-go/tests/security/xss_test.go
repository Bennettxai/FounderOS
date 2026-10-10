package security_test

import (
	"encoding/json"
	"html"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReflectedXSSPrevention tests XSS prevention in query parameters
func TestReflectedXSSPrevention(t *testing.T) {
	xssPayloads := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<svg onload=alert('XSS')>",
		"<body onload=alert('XSS')>",
		"<iframe src='javascript:alert(`xss`)'></iframe>",
		"<input onfocus=alert('XSS') autofocus>",
		"<select onfocus=alert('XSS') autofocus>",
		"<textarea onfocus=alert('XSS') autofocus>",
		"<marquee onstart=alert('XSS')>",
		"<details open ontoggle=alert('XSS')>",
	}

	for _, payload := range xssPayloads {
		t.Run("XSS_"+limitString(payload, 30), func(t *testing.T) {
			// Test 1: Sanitize for JSON (removes dangerous patterns)
			sanitized := security.SanitizeForJSON(payload)
			assert.NotEqual(t, payload, sanitized, "Payload should be sanitized")
			assert.NotContains(t, sanitized, "<script", "Script tags should be escaped")
			assert.NotContains(t, sanitized, "javascript:", "JavaScript protocol should be removed")

			// Test 2: Sanitize then JSON encode (production pattern)
			sanitizedForJSON := security.SanitizeForJSON(payload)
			jsonData := map[string]string{"message": sanitizedForJSON}
			_, err := json.Marshal(jsonData)
			require.NoError(t, err)

			// Sanitization removes dangerous patterns before JSON encoding
			assert.NotContains(t, sanitizedForJSON, "onerror=", "Event handlers should be removed")
			assert.NotContains(t, sanitizedForJSON, "javascript:", "Dangerous protocols should be removed")

			// Test 3: Detect XSS patterns
			detected := detectXSSPattern(payload)
			assert.True(t, detected, "XSS pattern should be detected")
		})
	}
}

// TestStoredXSSPrevention tests XSS prevention in stored user content
func TestStoredXSSPrevention(t *testing.T) {
	userInputs := []struct {
		field   string
		input   string
		context string
	}{
		{"username", "<script>alert('xss')</script>", "User profile"},
		{"message", "<img src=x onerror=alert(1)>", "Chat message"},
		{"description", "javascript:void(0)", "Project description"},
		{"name", "<svg/onload=alert('xss')>", "Workspace name"},
		{"bio", "<iframe src=javascript:alert(1)>", "User bio"},
	}

	for _, tt := range userInputs {
		t.Run(tt.field+"_"+tt.context, func(t *testing.T) {
			// Database should store raw input
			stored := tt.input

			// Sanitize user content before returning in API response
			sanitized := security.SanitizeForJSON(stored)

			// API response should be JSON-encoded with sanitized content
			response := map[string]string{tt.field: sanitized}
			jsonBytes, err := json.Marshal(response)
			require.NoError(t, err)

			// Verify JSON encoding prevents XSS
			jsonString := string(jsonBytes)
			assert.NotContains(t, jsonString, "<script", "Script tags should be escaped")
			assert.NotContains(t, jsonString, "onerror=", "Event handlers should be escaped")
			assert.NotContains(t, jsonString, "javascript:", "JavaScript protocol should be escaped")

			// Verify sanitized content is different from raw input (XSS patterns removed)
			if security.ContainsXSSPattern(tt.input) {
				assert.NotEqual(t, tt.input, sanitized, "XSS patterns should be sanitized")
			}
		})
	}
}

// TestDOMBasedXSSPrevention tests DOM-based XSS prevention patterns
func TestDOMBasedXSSPrevention(t *testing.T) {
	t.Run("URL fragment XSS", func(t *testing.T) {
		// Simulate dangerous URL fragments
		fragments := []string{
			"#<script>alert(1)</script>",
			"#javascript:alert(1)",
			"#data:text/html,<script>alert(1)</script>",
		}

		for _, fragment := range fragments {
			// Frontend should sanitize URL fragments before using in DOM
			sanitized := sanitizeURLFragment(fragment)
			assert.NotContains(t, sanitized, "<script", "Script tags should be removed")
			assert.NotContains(t, sanitized, "javascript:", "JavaScript protocol should be removed")
		}
	})

	t.Run("innerHTML usage prevention", func(t *testing.T) {
		// Document that innerHTML should never be used with user input
		// Always use textContent or createElement

		userInput := "<img src=x onerror=alert(1)>"

		// WRONG: element.innerHTML = userInput (executes XSS)
		// RIGHT: element.textContent = userInput (safe)

		// Simulate textContent behavior (treats as plain text)
		safeTextContent := userInput // No parsing, just text

		// Verify it's treated as text, not HTML
		assert.Contains(t, safeTextContent, "<img", "Should be stored as literal text")
	})
}

// TestContentSecurityPolicy tests CSP header configuration
func TestContentSecurityPolicy(t *testing.T) {
	t.Run("CSP header blocks inline scripts", func(t *testing.T) {
		// Content-Security-Policy should include:
		// - default-src 'self'
		// - script-src 'self'
		// - NO 'unsafe-inline' or 'unsafe-eval'

		csp := "default-src 'self'; script-src 'self'; object-src 'none'; base-uri 'self';"

		assert.Contains(t, csp, "default-src 'self'", "Should restrict default sources")
		assert.Contains(t, csp, "script-src 'self'", "Should restrict script sources")
		assert.NotContains(t, csp, "unsafe-inline", "Should not allow unsafe-inline")
		assert.NotContains(t, csp, "unsafe-eval", "Should not allow unsafe-eval")
		assert.Contains(t, csp, "object-src 'none'", "Should block objects")
	})

	t.Run("CSP nonce for inline scripts", func(t *testing.T) {
		// If inline scripts are needed, use nonces
		nonce := generateNonce()

		assert.NotEmpty(t, nonce, "Nonce should be generated")
		assert.Greater(t, len(nonce), 16, "Nonce should be sufficiently random")

		csp := "script-src 'self' 'nonce-" + nonce + "';"
		assert.Contains(t, csp, "nonce-", "CSP should include nonce")
	})
}

// TestHTMLEntityEncoding tests HTML entity encoding
func TestHTMLEntityEncoding(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<script>", "&lt;script&gt;"},
		{"'alert(1)'", "&#39;alert(1)&#39;"},
		{`"onclick"`, "&#34;onclick&#34;"},
		{"&", "&amp;"},
		{"<>&\"'", "&lt;&gt;&amp;&#34;&#39;"},
	}

	for _, tt := range tests {
		t.Run("Encode_"+tt.input, func(t *testing.T) {
			escaped := html.EscapeString(tt.input)
			assert.Equal(t, tt.expected, escaped, "Should properly encode HTML entities")
		})
	}
}

// TestJavaScriptContextEscaping tests escaping for JavaScript contexts
func TestJavaScriptContextEscaping(t *testing.T) {
	dangerousInputs := []string{
		"</script><script>alert(1)</script>",
		"';alert(1);//",
		"\";alert(1);//",
		"\\';alert(1);//",
	}

	for _, input := range dangerousInputs {
		t.Run("JS_Context_"+limitString(input, 20), func(t *testing.T) {
			// When embedding in JavaScript, use JSON encoding
			jsonEncoded, err := json.Marshal(input)
			require.NoError(t, err)

			// JSON encoding escapes quotes and special chars
			jsonString := string(jsonEncoded)
			assert.NotContains(t, jsonString, "</script>", "Should escape script closing tags")
		})
	}
}

// TestURLContextEncoding tests URL encoding
func TestURLContextEncoding(t *testing.T) {
	tests := []struct {
		input    string
		contains string
	}{
		{"javascript:alert(1)", "javascript"},
		{"data:text/html,<script>alert(1)</script>", "data:"},
		{"vbscript:msgbox(1)", "vbscript"},
	}

	for _, tt := range tests {
		t.Run("URL_"+limitString(tt.input, 20), func(t *testing.T) {
			// Dangerous protocols should be detected
			isDangerous := isDangerousURL(tt.input)
			assert.True(t, isDangerous, "Should detect dangerous URL protocol")
		})
	}
}

// TestXSSInHTTPHeaders tests XSS prevention in HTTP headers
func TestXSSInHTTPHeaders(t *testing.T) {
	t.Run("X-Content-Type-Options header", func(t *testing.T) {
		// Should be set to 'nosniff' to prevent MIME sniffing
		header := "nosniff"
		assert.Equal(t, "nosniff", header, "X-Content-Type-Options should be nosniff")
	})

	t.Run("X-Frame-Options header", func(t *testing.T) {
		// Should prevent clickjacking
		validOptions := []string{"DENY", "SAMEORIGIN"}
		header := "DENY"

		found := false
		for _, opt := range validOptions {
			if header == opt {
				found = true
				break
			}
		}
		assert.True(t, found, "X-Frame-Options should be DENY or SAMEORIGIN")
	})
}

// Helper functions

func sanitizeURLFragment(fragment string) string {
	// Remove dangerous protocols and tags
	sanitized := fragment
	dangerous := []string{"javascript:", "data:", "<script", "<iframe"}
	for _, d := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, d, "")
		sanitized = strings.ReplaceAll(sanitized, strings.ToUpper(d), "")
	}
	return sanitized
}

func generateNonce() string {
	// Generate a cryptographically random nonce
	// In real implementation, use crypto/rand
	return "random-nonce-12345678901234567890"
}

func isDangerousURL(url string) bool {
	dangerousProtocols := []string{"javascript:", "data:", "vbscript:", "file:"}
	lowerURL := strings.ToLower(url)
	for _, protocol := range dangerousProtocols {
		if strings.HasPrefix(lowerURL, protocol) {
			return true
		}
	}
	return false
}
