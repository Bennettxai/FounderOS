package security_test

import (
	"flag"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Parse flags so testing.Short() works
	flag.Parse()

	// Security integration tests require a fully configured environment.
	// Skip in short mode - run explicitly with: go test ./tests/security/
	if testing.Short() {
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// Shared test helper functions

// limitString truncates a string to a maximum length
func limitString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// detectXSSPattern checks if a string contains common XSS patterns
func detectXSSPattern(input string) bool {
	xssPatterns := []string{
		"<script",
		"javascript:",
		"onerror=",
		"onload=",
		"onfocus=",
		"onclick=",
		"onmouseover=",
		"onstart=",
		"ontoggle=",
		"<svg",
		"<iframe",
		"<img",
		"<body",
		"<input",
		"<select",
		"<textarea",
		"<marquee",
		"<details",
	}

	// Simple case-insensitive substring check
	for _, pattern := range xssPatterns {
		if len(input) >= len(pattern) {
			for i := 0; i <= len(input)-len(pattern); i++ {
				match := true
				for j := 0; j < len(pattern); j++ {
					c1 := input[i+j]
					c2 := pattern[j]
					// Convert to lowercase for comparison
					if c1 >= 'A' && c1 <= 'Z' {
						c1 = c1 + 32
					}
					if c2 >= 'A' && c2 <= 'Z' {
						c2 = c2 + 32
					}
					if c1 != c2 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}
