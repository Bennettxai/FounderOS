package services

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
)

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// getRandomUserAgent returns a random user agent from the list
func getRandomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
}

// extractDomain extracts the domain from a URL
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}

// cleanDDGURL decodes DuckDuckGo's redirect URL
func cleanDDGURL(ddgURL string) string {
	// DuckDuckGo URLs are in format: //duckduckgo.com/l/?uddg=ENCODED_URL&...
	if strings.Contains(ddgURL, "uddg=") {
		parsed, err := url.Parse(ddgURL)
		if err != nil {
			return ddgURL
		}
		uddg := parsed.Query().Get("uddg")
		if uddg != "" {
			decoded, err := url.QueryUnescape(uddg)
			if err == nil {
				return decoded
			}
		}
	}

	// Handle direct URLs
	if strings.HasPrefix(ddgURL, "http") {
		return ddgURL
	}
	if strings.HasPrefix(ddgURL, "//") {
		return "https:" + ddgURL
	}

	return ddgURL
}

// cleanHTMLText removes HTML tags and decodes entities
func cleanHTMLText(text string) string {
	// Remove HTML tags
	tagPattern := regexp.MustCompile(`<[^>]+>`)
	text = tagPattern.ReplaceAllString(text, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&#x27;", "'")
	text = strings.ReplaceAll(text, "&#x2F;", "/")

	return strings.TrimSpace(text)
}

// extractTitleFromText extracts a title from DuckDuckGo's text format
func extractTitleFromText(text string) string {
	// DuckDuckGo often has format "Title - Description"
	if idx := strings.Index(text, " - "); idx > 0 && idx < 100 {
		return text[:idx]
	}
	// Truncate if too long
	if len(text) > 100 {
		return text[:97] + "..."
	}
	return text
}

// isNumeric checks if a string is numeric
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// hashQueryString creates a SHA256 hash of the normalized query string
func hashQueryString(query string) string {
	normalized := strings.ToLower(strings.TrimSpace(query))
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:])
}
