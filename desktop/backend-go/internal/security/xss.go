package security

import (
	"html"
	"regexp"
	"strings"
)

// Common XSS attack patterns
var (
	// Event handlers (onclick=, onerror=, onload=, etc.)
	eventHandlerRegex = regexp.MustCompile(`(?i)\s*on\w+\s*=`)

	// Dangerous URL protocols (matches anywhere in string, not just at start)
	dangerousProtocolRegex = regexp.MustCompile(`(?i)(javascript|data|vbscript|file):`)
)

// SanitizeForJSON sanitizes user input to prevent stored XSS attacks
// This function should be called on user-generated content before returning it in JSON responses
func SanitizeForJSON(input string) string {
	if input == "" {
		return input
	}

	// First remove dangerous patterns BEFORE HTML escaping
	sanitized := input

	// Remove/escape event handlers (onerror=, onclick=, etc.)
	sanitized = eventHandlerRegex.ReplaceAllString(sanitized, " on___BLOCKED___=")

	// Escape dangerous URL protocols (must happen before HTML escaping)
	sanitized = dangerousProtocolRegex.ReplaceAllString(sanitized, "blocked___:")

	// Finally apply HTML escaping (handles <, >, &, ", ')
	sanitized = html.EscapeString(sanitized)

	return sanitized
}

// SanitizeURL checks and sanitizes URLs to prevent javascript: and data: protocol attacks
func SanitizeURL(url string) string {
	if url == "" {
		return url
	}

	trimmed := strings.TrimSpace(url)

	// Check for dangerous protocols
	if dangerousProtocolRegex.MatchString(trimmed) {
		return "" // Return empty string for dangerous URLs
	}

	return url
}

// ContainsXSSPattern checks if input contains common XSS patterns
func ContainsXSSPattern(input string) bool {
	if input == "" {
		return false
	}

	patterns := []string{
		"<script",
		"javascript:",
		"onerror=",
		"onload=",
		"onfocus=",
		"onclick=",
		"<iframe",
		"<svg",
		"<embed",
		"<object",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range patterns {
		if strings.Contains(lowerInput, pattern) {
			return true
		}
	}

	return false
}
