package services

import (
	"context"
	"fmt"
	"strings"
)

// loadKBItemsByCategories loads contexts (KB items) matching the specified categories
func (s *FocusService) loadKBItemsByCategories(ctx context.Context, userID string, categories []string, limit int) ([]KBContextItem, error) {
	if s.pool == nil || len(categories) == 0 {
		return nil, nil
	}

	if limit <= 0 {
		limit = 5 // Default limit
	}

	// Build query with category filter
	// Categories map to context types (PERSON, BUSINESS, PROJECT, CUSTOM, document, DOCUMENT)
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, type, content
		FROM contexts
		WHERE user_id = $1
		  AND is_archived = false
		  AND type = ANY($2::text[])
		ORDER BY updated_at DESC
		LIMIT $3
	`, userID, categories, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load KB items: %w", err)
	}
	defer rows.Close()

	var items []KBContextItem
	for rows.Next() {
		var item KBContextItem
		var contextType string
		var content *string

		err := rows.Scan(&item.ID, &item.Title, &contextType, &content)
		if err != nil {
			continue
		}

		item.Category = contextType
		if content != nil {
			item.Content = *content
		}

		items = append(items, item)
	}

	return items, nil
}

// AutoLoadKBContext intelligently loads KB items based on focus mode and query content
func (s *FocusService) AutoLoadKBContext(ctx context.Context, userID string, focusMode string, userQuery string, limit int) ([]KBContextItem, error) {
	if s.pool == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 5
	}

	// Determine which context types to prioritize based on focus mode
	priorityTypes := s.getContextTypesForFocusMode(focusMode)

	// Extract keywords from query for relevance matching
	keywords := extractKeywords(userQuery)

	// Build query with smart relevance scoring
	query := `
		SELECT id, name, type, content,
		       (CASE WHEN type = ANY($3::text[]) THEN 2 ELSE 1 END) as type_score,
		       (CASE
		         WHEN name ILIKE ANY($4::text[]) THEN 3
		         WHEN content ILIKE ANY($4::text[]) THEN 2
		         ELSE 1
		       END) as relevance_score
		FROM contexts
		WHERE user_id = $1
		  AND is_archived = false
		  AND (name ILIKE ANY($4::text[]) OR content ILIKE ANY($4::text[]) OR type = ANY($3::text[]))
		ORDER BY (type_score * relevance_score) DESC, updated_at DESC
		LIMIT $2
	`

	// Build keyword patterns for ILIKE
	keywordPatterns := make([]string, len(keywords))
	for i, kw := range keywords {
		keywordPatterns[i] = "%" + kw + "%"
	}

	// If no keywords, fall back to category-based loading
	if len(keywordPatterns) == 0 {
		return s.loadKBItemsByCategories(ctx, userID, priorityTypes, limit)
	}

	rows, err := s.pool.Query(ctx, query, userID, limit, priorityTypes, keywordPatterns)
	if err != nil {
		// Fallback to simple category load
		return s.loadKBItemsByCategories(ctx, userID, priorityTypes, limit)
	}
	defer rows.Close()

	var items []KBContextItem
	for rows.Next() {
		var item KBContextItem
		var contextType string
		var content *string
		var typeScore, relevanceScore int

		err := rows.Scan(&item.ID, &item.Title, &contextType, &content, &typeScore, &relevanceScore)
		if err != nil {
			continue
		}

		item.Category = contextType
		if content != nil {
			item.Content = *content
		}

		items = append(items, item)
	}

	return items, nil
}

// getContextTypesForFocusMode returns priority context types based on focus mode
func (s *FocusService) getContextTypesForFocusMode(focusMode string) []string {
	switch focusMode {
	case "code", "build":
		return []string{"PROJECT", "DOCUMENT", "CUSTOM"}
	case "write":
		return []string{"DOCUMENT", "BUSINESS", "CUSTOM"}
	case "analyze":
		return []string{"BUSINESS", "PROJECT", "DOCUMENT"}
	case "plan", "planning":
		return []string{"PROJECT", "BUSINESS", "DOCUMENT"}
	case "research", "deep":
		return []string{"DOCUMENT", "CUSTOM", "BUSINESS"}
	case "creative":
		return []string{"CUSTOM", "DOCUMENT", "PERSON"}
	default:
		return []string{"DOCUMENT", "PROJECT", "BUSINESS", "CUSTOM"}
	}
}

// extractKeywords extracts important keywords from a query for matching
func extractKeywords(query string) []string {
	// Remove common stop words and extract meaningful terms
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "are": true, "was": true,
		"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "can": true, "may": true, "might": true, "must": true,
		"i": true, "me": true, "my": true, "we": true, "our": true, "you": true, "your": true,
		"he": true, "she": true, "it": true, "they": true, "them": true, "their": true,
		"this": true, "that": true, "these": true, "those": true,
		"what": true, "which": true, "who": true, "whom": true, "where": true, "when": true,
		"why": true, "how": true, "with": true, "about": true, "for": true, "from": true,
		"of": true, "on": true, "in": true, "to": true, "at": true, "by": true, "as": true,
		"and": true, "or": true, "but": true, "if": true, "then": true, "so": true,
		"please": true, "help": true, "want": true, "need": true, "like": true,
		"tell": true, "explain": true, "show": true, "give": true, "make": true,
	}

	words := strings.Fields(strings.ToLower(query))
	var keywords []string
	seen := make(map[string]bool)

	for _, word := range words {
		// Clean punctuation
		word = strings.Trim(word, ".,!?;:'\"()[]{}")

		// Skip short words, stop words, and duplicates
		if len(word) < 3 || stopWords[word] || seen[word] {
			continue
		}

		seen[word] = true
		keywords = append(keywords, word)
	}

	// Limit to most relevant keywords
	if len(keywords) > 5 {
		keywords = keywords[:5]
	}

	return keywords
}
