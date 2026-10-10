package services

import "testing"

func TestClassifyDIKW_ByMemoryType(t *testing.T) {
	tests := []struct {
		memType  string
		expected DIKWLevel
	}{
		{"fact", DIKWData},
		{"observation", DIKWData},
		{"process", DIKWInformation},
		{"pattern", DIKWInformation},
		{"knowledge", DIKWKnowledge},
		{"decision", DIKWKnowledge},
		{"policy", DIKWWisdom},
		{"principle", DIKWWisdom},
	}

	for _, tt := range tests {
		result := ClassifyDIKW(tt.memType, "")
		if result != tt.expected {
			t.Errorf("ClassifyDIKW(%q, \"\") = %s, want %s", tt.memType, result, tt.expected)
		}
	}
}

func TestClassifyDIKW_ByContent(t *testing.T) {
	tests := []struct {
		content  string
		expected DIKWLevel
	}{
		{"We always use TLS for all API connections", DIKWWisdom},
		{"The team decided to use PostgreSQL for the main database", DIKWKnowledge},
		{"This is a summary of the Q3 results", DIKWInformation},
		{"Server responded with 200 OK", DIKWData},
		{"", DIKWData},
	}

	for _, tt := range tests {
		result := ClassifyDIKW("", tt.content)
		if result != tt.expected {
			t.Errorf("ClassifyDIKW(\"\", %q) = %s, want %s", tt.content[:min(40, len(tt.content))], result, tt.expected)
		}
	}
}

func TestClassifyDIKW_MemoryTypeTakesPriority(t *testing.T) {
	// Even if content suggests Wisdom, memory type should take priority
	result := ClassifyDIKW("fact", "This is a principle we always follow")
	if result != DIKWData {
		t.Errorf("memory type should take priority, got %s", result)
	}
}

func TestClassifyDIKW_UnknownType(t *testing.T) {
	// Unknown memory type falls through to content analysis
	result := ClassifyDIKW("custom_type", "We decided to use a new approach because it works better")
	if result != DIKWKnowledge {
		t.Errorf("expected KNOWLEDGE from content, got %s", result)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
