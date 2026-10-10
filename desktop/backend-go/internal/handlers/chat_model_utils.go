package handlers

import (
	"regexp"
	"strings"
)

// normalizeModelName fixes common model name issues
// Maps display names to actual API model IDs
func normalizeModelName(model string) string {
	// Common mappings from display names to API IDs
	modelMappings := map[string]string{
		// Groq models - fix spaces and case issues
		"llama 3.3 70b":           "llama-3.3-70b-versatile",
		"llama 3.3 70b versatile": "llama-3.3-70b-versatile",
		"llama 3.1 70b":           "llama-3.1-70b-versatile",
		"llama 3.1 70b versatile": "llama-3.1-70b-versatile",
		"llama 3.1 8b":            "llama-3.1-8b-instant",
		"llama 3.1 8b instant":    "llama-3.1-8b-instant",
		"llama 3 70b":             "llama3-70b-8192",
		"llama 3 8b":              "llama3-8b-8192",
		"mixtral 8x7b":            "mixtral-8x7b-32768",
		"gemma 2 9b":              "gemma2-9b-it",
		"gemma2 9b":               "gemma2-9b-it",
	}

	// Check for exact match (case-insensitive)
	lowerModel := strings.ToLower(strings.TrimSpace(model))
	if mapped, ok := modelMappings[lowerModel]; ok {
		return mapped
	}

	// Return original if no mapping found
	return model
}

// stripThinkingTags removes <thinking>...</thinking> tags and variations from the response
func stripThinkingTags(content string) string {
	// Use a more flexible regex that matches any tag starting with <think
	re := regexp.MustCompile(`<think[^>]*>[\s\S]*?</think[^>]*>\s*`)
	result := re.ReplaceAllString(content, "")
	return strings.TrimSpace(result)
}
