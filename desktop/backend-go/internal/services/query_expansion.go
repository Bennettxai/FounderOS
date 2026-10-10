package services

import (
	"context"
	"fmt"
	"strings"
)

// QueryExpansionService enhances queries with synonyms and rewrites
type QueryExpansionService struct {
	synonyms   map[string][]string
	llmService LLMService // Optional: for advanced query rewriting (uses existing LLMService interface)
}

// NewQueryExpansionService creates a new query expansion service
func NewQueryExpansionService(llmService LLMService) *QueryExpansionService {
	return &QueryExpansionService{
		synonyms:   buildDefaultSynonymMap(),
		llmService: llmService,
	}
}

// ExpandedQuery represents an expanded query with variants
type ExpandedQuery struct {
	Original    string   `json:"original"`
	Expanded    []string `json:"expanded"`    // Synonym-expanded versions
	Rewritten   string   `json:"rewritten"`   // LLM-rewritten version
	AllVariants []string `json:"all_variants"` // All query variants
}

// Expand expands a query with synonyms and optionally rewrites it
func (q *QueryExpansionService) Expand(ctx context.Context, query string, useRewrite bool) (*ExpandedQuery, error) {
	expanded := &ExpandedQuery{
		Original:    query,
		Expanded:    make([]string, 0),
		AllVariants: make([]string, 0),
	}

	// 1. Synonym expansion
	expanded.Expanded = q.expandWithSynonyms(query)

	// 2. Query rewriting (if LLM available and requested)
	if useRewrite && q.llmService != nil {
		rewritten, err := q.rewriteQuery(ctx, query)
		if err == nil && rewritten != "" && rewritten != query {
			expanded.Rewritten = rewritten
		}
	}

	// 3. Combine all variants
	expanded.AllVariants = append(expanded.AllVariants, query)
	expanded.AllVariants = append(expanded.AllVariants, expanded.Expanded...)
	if expanded.Rewritten != "" {
		expanded.AllVariants = append(expanded.AllVariants, expanded.Rewritten)
	}

	// Deduplicate
	expanded.AllVariants = deduplicate(expanded.AllVariants)

	return expanded, nil
}

// expandWithSynonyms expands query terms with synonyms
func (q *QueryExpansionService) expandWithSynonyms(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	variants := make([]string, 0)

	// Generate variants by replacing each word with its synonyms
	for i, word := range words {
		if synonyms, exists := q.synonyms[word]; exists {
			for _, synonym := range synonyms {
				// Create variant with synonym
				variantWords := make([]string, len(words))
				copy(variantWords, words)
				variantWords[i] = synonym
				variant := strings.Join(variantWords, " ")
				variants = append(variants, variant)
			}
		}
	}

	return variants
}

// rewriteQuery uses LLM to rewrite the query for better search
func (q *QueryExpansionService) rewriteQuery(ctx context.Context, query string) (string, error) {
	if q.llmService == nil {
		return "", nil
	}

	systemPrompt := "You are a query rewriter. Rewrite search queries to be more effective for semantic search. Keep the core meaning but make it clearer and more specific. Return ONLY the rewritten query, nothing else."

	messages := []ChatMessage{
		{
			Role:    "user",
			Content: fmt.Sprintf("Original query: %s\n\nRewritten query:", query),
		},
	}

	rewritten, err := q.llmService.ChatComplete(ctx, messages, systemPrompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(rewritten), nil
}

// ExtractKeyTerms extracts key terms from a query for highlighting
func (q *QueryExpansionService) ExtractKeyTerms(query string) []string {
	// Remove common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true,
		"should": true, "could": true, "may": true, "might": true, "must": true,
		"can": true, "what": true, "how": true, "when": true, "where": true,
		"why": true, "which": true, "who": true, "whom": true, "whose": true,
	}

	words := strings.Fields(strings.ToLower(query))
	keyTerms := make([]string, 0)

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,;:!?()[]{}\"'")

		if word != "" && !stopWords[word] && len(word) > 2 {
			keyTerms = append(keyTerms, word)
		}
	}

	return keyTerms
}

// SuggestQueries generates query suggestions based on intent
func (q *QueryExpansionService) SuggestQueries(query string, intent QueryIntent) []string {
	suggestions := make([]string, 0)

	switch intent {
	case IntentFactualLookup:
		// Suggest more specific factual queries
		suggestions = append(suggestions, "definition of "+query)
		suggestions = append(suggestions, "what is "+query)
		suggestions = append(suggestions, query+" explanation")

	case IntentProcedural:
		// Suggest procedural variations
		suggestions = append(suggestions, "how to "+query)
		suggestions = append(suggestions, "steps for "+query)
		suggestions = append(suggestions, query+" tutorial")
		suggestions = append(suggestions, query+" guide")

	case IntentComparison:
		// Already a comparison, keep as is
		suggestions = append(suggestions, query)

	case IntentRecent:
		// Suggest time-based variations
		suggestions = append(suggestions, "latest "+query)
		suggestions = append(suggestions, "recent "+query)
		suggestions = append(suggestions, query+" updates")

	case IntentExhaustive:
		// Suggest comprehensive variations
		suggestions = append(suggestions, "complete guide to "+query)
		suggestions = append(suggestions, "everything about "+query)
		suggestions = append(suggestions, query+" overview")

	default:
		suggestions = append(suggestions, query)
	}

	return suggestions
}

// buildDefaultSynonymMap creates a default synonym dictionary
func buildDefaultSynonymMap() map[string][]string {
	return map[string][]string{
		// Programming terms
		"function":      {"method", "procedure", "routine"},
		"method":        {"function", "procedure"},
		"class":         {"type", "struct"},
		"variable":      {"var", "field", "property"},
		"array":         {"list", "slice", "collection"},
		"object":        {"instance", "entity"},
		"error":         {"exception", "failure", "bug"},
		"fix":           {"resolve", "repair", "correct"},
		"implement":     {"create", "build", "develop"},
		"optimize":      {"improve", "enhance", "speed up"},
		"refactor":      {"restructure", "reorganize", "clean up"},

		// Database terms
		"database":      {"db", "datastore", "data store"},
		"table":         {"relation", "entity"},
		"query":         {"search", "select", "find"},
		"index":         {"key", "lookup"},
		"join":          {"merge", "combine"},

		// Web/API terms
		"api":           {"endpoint", "service", "interface"},
		"endpoint":      {"route", "path", "api"},
		"request":       {"call", "query"},
		"response":      {"reply", "result", "output"},
		"authentication": {"auth", "login", "sign-in"},
		"authorization":  {"access control", "permissions"},

		// General tech terms
		"bug":           {"error", "issue", "defect"},
		"feature":       {"functionality", "capability"},
		"performance":   {"speed", "efficiency", "throughput"},
		"security":      {"safety", "protection"},
		"configuration": {"config", "settings", "setup"},
		"documentation": {"docs", "manual", "guide"},
		"test":          {"testing", "validation", "verification"},
		"deploy":        {"deployment", "release", "publish"},
		"container":     {"docker", "pod"},
		"server":        {"host", "instance", "node"},
		"client":        {"frontend", "user interface", "ui"},

		// Action verbs
		"create":        {"make", "build", "generate"},
		"delete":        {"remove", "destroy", "drop"},
		"update":        {"modify", "change", "edit"},
		"retrieve":      {"get", "fetch", "obtain"},
		"save":          {"store", "persist", "write"},
		"load":          {"read", "fetch", "retrieve"},

		// Common adjectives
		"fast":          {"quick", "rapid", "speedy"},
		"slow":          {"sluggish", "delayed"},
		"big":           {"large", "huge", "massive"},
		"small":         {"tiny", "little", "minimal"},
		"new":           {"recent", "latest", "fresh"},
		"old":           {"legacy", "deprecated", "outdated"},
		"simple":        {"easy", "basic", "straightforward"},
		"complex":       {"complicated", "advanced", "sophisticated"},
		"good":          {"better", "optimal", "best"},
		"bad":           {"poor", "suboptimal", "problematic"},
	}
}

// AddSynonym adds a custom synonym mapping
func (q *QueryExpansionService) AddSynonym(word string, synonyms []string) {
	word = strings.ToLower(word)
	if _, exists := q.synonyms[word]; !exists {
		q.synonyms[word] = make([]string, 0)
	}
	q.synonyms[word] = append(q.synonyms[word], synonyms...)
}

// GetSynonyms returns synonyms for a word
func (q *QueryExpansionService) GetSynonyms(word string) []string {
	word = strings.ToLower(word)
	if synonyms, exists := q.synonyms[word]; exists {
		return synonyms
	}
	return nil
}

// Helper functions

func deduplicate(strings []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, str := range strings {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}
