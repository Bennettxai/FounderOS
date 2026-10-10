package services

import (
	"regexp"
	"strings"
)

// extractTags extracts relevant tags from text
func (s *MemoryExtractorService) extractTags(text string) []string {
	tags := make([]string, 0)
	lower := strings.ToLower(text)

	// Technology tags
	techTags := map[string][]string{
		"go":         {"go", "golang"},
		"javascript": {"javascript", "js", "node"},
		"typescript": {"typescript", "ts"},
		"react":      {"react", "nextjs"},
		"svelte":     {"svelte", "sveltekit"},
		"python":     {"python", "py"},
		"database":   {"database", "sql", "postgresql", "postgres", "mysql", "redis"},
		"api":        {"api", "rest", "graphql", "endpoint"},
		"docker":     {"docker", "container", "kubernetes", "k8s"},
		"aws":        {"aws", "s3", "ec2", "lambda"},
		"git":        {"git", "github", "gitlab"},
	}

	for tag, keywords := range techTags {
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				tags = append(tags, tag)
				break
			}
		}
	}

	return tags
}

// extractEntitiesFromText extracts entities from text
func (s *MemoryExtractorService) extractEntitiesFromText(text string) []string {
	entities := make([]string, 0)

	// File paths
	filePattern := regexp.MustCompile(`[\w/-]+\.(go|ts|js|svelte|py|sql|json|yaml|yml|md)`)
	files := filePattern.FindAllString(text, -1)
	entities = append(entities, files...)

	// URLs
	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlPattern.FindAllString(text, -1)
	entities = append(entities, urls...)

	return entities
}

// inferTaskImportance infers task importance from description
func (s *MemoryExtractorService) inferTaskImportance(description string) int {
	lower := strings.ToLower(description)

	if strings.Contains(lower, "urgent") || strings.Contains(lower, "critical") ||
		strings.Contains(lower, "asap") || strings.Contains(lower, "blocker") {
		return 9
	}

	if strings.Contains(lower, "important") || strings.Contains(lower, "must") ||
		strings.Contains(lower, "required") {
		return 7
	}

	if strings.Contains(lower, "nice to have") || strings.Contains(lower, "maybe") ||
		strings.Contains(lower, "could") {
		return 3
	}

	return 5
}

// deduplicateMemories removes duplicate memories
func (s *MemoryExtractorService) deduplicateMemories(memories []ExtractedMemory) []ExtractedMemory {
	seen := make(map[string]bool)
	unique := make([]ExtractedMemory, 0)

	for _, m := range memories {
		key := strings.ToLower(m.Summary)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, m)
		}
	}

	return unique
}

// isDuplicate checks if a memory is a duplicate of existing memories
func (s *MemoryExtractorService) isDuplicate(m ExtractedMemory, existing []ExtractedMemory) bool {
	mLower := strings.ToLower(m.Summary)
	for _, e := range existing {
		eLower := strings.ToLower(e.Summary)
		// Check for exact match or high similarity
		if mLower == eLower {
			return true
		}
		// Check for substring match (one contains the other)
		if len(mLower) > 10 && len(eLower) > 10 {
			if strings.Contains(mLower, eLower) || strings.Contains(eLower, mLower) {
				return true
			}
		}
	}
	return false
}
