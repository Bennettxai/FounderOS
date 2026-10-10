package services

// DIKWLevel represents the Data-Information-Knowledge-Wisdom hierarchy.
type DIKWLevel string

const (
	DIKWData        DIKWLevel = "DATA"        // Raw facts, observations
	DIKWInformation DIKWLevel = "INFORMATION" // Organized, contextualized data
	DIKWKnowledge   DIKWLevel = "KNOWLEDGE"   // Patterns, decisions, understanding
	DIKWWisdom      DIKWLevel = "WISDOM"      // Principles, policies, meta-knowledge
)

// ClassifyDIKW maps a memory type and content to a DIKW level.
// Used when inserting workspace memories to classify their knowledge tier.
func ClassifyDIKW(memoryType, content string) DIKWLevel {
	// Primary classification by memory_type
	switch memoryType {
	case "fact", "observation", "log", "event":
		return DIKWData
	case "process", "pattern", "summary", "context":
		return DIKWInformation
	case "knowledge", "decision", "lesson", "insight":
		return DIKWKnowledge
	case "policy", "principle", "strategy", "value":
		return DIKWWisdom
	}

	// Content-based heuristics for edge cases
	return classifyByContent(content)
}

// classifyByContent uses keyword heuristics to classify content.
func classifyByContent(content string) DIKWLevel {
	if len(content) == 0 {
		return DIKWData
	}

	// Convert to lowercase for matching (avoid importing strings for a simple check)
	lower := toLowerASCII(content)

	// Wisdom indicators: principles, policies, meta-knowledge
	wisdomKeywords := []string{
		"always", "never", "principle", "policy", "must", "standard",
		"governance", "rule", "best practice", "guideline",
	}
	for _, kw := range wisdomKeywords {
		if containsASCII(lower, kw) {
			return DIKWWisdom
		}
	}

	// Knowledge indicators: decisions, patterns, understanding
	knowledgeKeywords := []string{
		"decided", "learned", "pattern", "because", "reason",
		"conclusion", "lesson", "insight", "approach", "strategy",
	}
	for _, kw := range knowledgeKeywords {
		if containsASCII(lower, kw) {
			return DIKWKnowledge
		}
	}

	// Information indicators: organized, contextualized
	infoKeywords := []string{
		"summary", "overview", "process", "workflow", "context",
		"description", "explanation", "how to",
	}
	for _, kw := range infoKeywords {
		if containsASCII(lower, kw) {
			return DIKWInformation
		}
	}

	return DIKWData // Default to most basic level
}

// toLowerASCII converts ASCII uppercase letters to lowercase.
// Avoids importing strings package for a trivial helper.
func toLowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// containsASCII checks if s contains substr (case-insensitive ASCII).
func containsASCII(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
