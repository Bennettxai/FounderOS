package services

import "strings"

// ============================================================================
// QUERY OPTIMIZER
// ============================================================================

// QueryOptimizer transforms conversational input into optimized search queries
type QueryOptimizer struct{}

// NewQueryOptimizer creates a new query optimizer
func NewQueryOptimizer() *QueryOptimizer {
	return &QueryOptimizer{}
}

// stopWords are common words to remove from queries
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true, "is": true, "are": true, "was": true, "were": true,
	"be": true, "been": true, "being": true, "have": true, "has": true, "had": true,
	"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
	"should": true, "may": true, "might": true, "must": true, "shall": true,
	"i": true, "me": true, "my": true, "myself": true, "we": true, "our": true,
	"you": true, "your": true, "yourself": true, "he": true, "she": true, "it": true,
	"they": true, "them": true, "their": true, "this": true, "that": true, "these": true,
	"what": true, "which": true, "who": true, "whom": true, "where": true, "when": true,
	"why": true, "how": true, "can": true, "please": true, "help": true, "need": true,
	"want": true, "like": true, "know": true, "think": true, "just": true, "also": true,
	"about": true, "with": true, "from": true, "into": true, "some": true, "any": true,
	"tell": true, "explain": true, "describe": true, "give": true, "show": true,
}

// questionPrefixes are common question patterns to strip
var questionPrefixes = []string{
	"can you tell me",
	"could you explain",
	"what is the",
	"what are the",
	"how do i",
	"how can i",
	"how to",
	"why is",
	"why are",
	"why do",
	"where can i find",
	"where is",
	"when did",
	"when was",
	"who is",
	"who are",
	"tell me about",
	"explain to me",
	"i want to know",
	"i need to know",
	"i'm looking for",
	"looking for",
	"searching for",
	"find me",
	"help me with",
	"help me understand",
}

// OptimizeQuery transforms conversational input into a search-optimized query
func (o *QueryOptimizer) OptimizeQuery(input string) string {
	if input == "" {
		return ""
	}

	// Lowercase for processing
	query := strings.ToLower(strings.TrimSpace(input))

	// Remove question prefixes
	for _, prefix := range questionPrefixes {
		if strings.HasPrefix(query, prefix) {
			query = strings.TrimPrefix(query, prefix)
			query = strings.TrimSpace(query)
			break
		}
	}

	// Remove punctuation at the end
	query = strings.TrimRight(query, "?.!,;:")

	// Split into words and filter
	words := strings.Fields(query)
	var filteredWords []string

	for _, word := range words {
		// Clean the word
		cleanWord := strings.Trim(word, ".,!?;:'\"()[]{}")

		// Skip stop words unless it's a short query
		if len(words) > 3 && stopWords[cleanWord] {
			continue
		}

		// Skip very short words unless they're likely important (numbers, acronyms)
		if len(cleanWord) < 2 && !isNumeric(cleanWord) {
			continue
		}

		filteredWords = append(filteredWords, cleanWord)
	}

	// Reconstruct the query
	optimized := strings.Join(filteredWords, " ")

	// If we filtered too aggressively, use original minus question prefix
	if len(optimized) < 3 && len(input) > 3 {
		optimized = strings.TrimRight(query, "?.!,;:")
	}

	return optimized
}

// QueryType represents the type of search query
type QueryType int

const (
	QueryTypeGeneral QueryType = iota
	QueryTypeHowTo
	QueryTypeComparison
	QueryTypeNews
	QueryTypeResearch
	QueryTypeTechnical
)

// QueryContext provides additional context for query optimization
type QueryContext struct {
	QueryType       QueryType
	RequireRecent   bool
	PreferredDomain string
	Language        string
}

// DetectQueryType analyzes the input to determine the type of query
func (o *QueryOptimizer) DetectQueryType(input string) QueryType {
	lower := strings.ToLower(input)

	// Check for how-to patterns
	if strings.Contains(lower, "how to") || strings.Contains(lower, "how do i") ||
		strings.Contains(lower, "how can i") || strings.Contains(lower, "steps to") {
		return QueryTypeHowTo
	}

	// Check for comparison patterns
	if strings.Contains(lower, " vs ") || strings.Contains(lower, "versus") ||
		strings.Contains(lower, "compared to") || strings.Contains(lower, "difference between") ||
		strings.Contains(lower, "better than") {
		return QueryTypeComparison
	}

	// Check for news patterns
	if strings.Contains(lower, "latest") || strings.Contains(lower, "recent") ||
		strings.Contains(lower, "news") || strings.Contains(lower, "today") ||
		strings.Contains(lower, "announced") || strings.Contains(lower, "released") {
		return QueryTypeNews
	}

	// Check for research patterns
	if strings.Contains(lower, "research") || strings.Contains(lower, "study") ||
		strings.Contains(lower, "analysis") || strings.Contains(lower, "statistics") ||
		strings.Contains(lower, "data on") {
		return QueryTypeResearch
	}

	// Check for technical patterns
	if strings.Contains(lower, "error") || strings.Contains(lower, "code") ||
		strings.Contains(lower, "programming") || strings.Contains(lower, "api") ||
		strings.Contains(lower, "documentation") || strings.Contains(lower, "bug") ||
		strings.Contains(lower, "implement") {
		return QueryTypeTechnical
	}

	return QueryTypeGeneral
}

// OptimizeWithContext adds contextual modifiers based on query intent
func (o *QueryOptimizer) OptimizeWithContext(input string, context QueryContext) string {
	base := o.OptimizeQuery(input)
	if base == "" {
		return ""
	}

	var modifiers []string

	// Add recency modifier for current events
	if context.RequireRecent {
		modifiers = append(modifiers, "2024 2025")
	}

	// Add type-specific modifiers
	switch context.QueryType {
	case QueryTypeTechnical:
		modifiers = append(modifiers, "documentation tutorial guide")
	case QueryTypeNews:
		modifiers = append(modifiers, "news latest")
	case QueryTypeResearch:
		modifiers = append(modifiers, "research study analysis")
	case QueryTypeHowTo:
		modifiers = append(modifiers, "how to guide steps")
	case QueryTypeComparison:
		modifiers = append(modifiers, "vs comparison difference")
	}

	// Add domain restriction if specified
	if context.PreferredDomain != "" {
		modifiers = append(modifiers, "site:"+context.PreferredDomain)
	}

	if len(modifiers) > 0 {
		return base + " " + strings.Join(modifiers, " ")
	}

	return base
}

// GenerateSearchQueries creates multiple search queries for comprehensive results
func (o *QueryOptimizer) GenerateSearchQueries(input string, maxQueries int) []string {
	if maxQueries <= 0 {
		maxQueries = 3
	}

	queries := make([]string, 0, maxQueries)

	// Primary optimized query
	primary := o.OptimizeQuery(input)
	if primary != "" {
		queries = append(queries, primary)
	}

	// Detect query type and add contextual variant
	queryType := o.DetectQueryType(input)
	ctx := QueryContext{QueryType: queryType}

	contextual := o.OptimizeWithContext(input, ctx)
	if contextual != primary && contextual != "" {
		queries = append(queries, contextual)
	}

	// Add a more specific variant if query is long enough
	if len(queries) < maxQueries {
		words := strings.Fields(primary)
		if len(words) > 4 {
			// Take the most important words (skip first word if it's common)
			start := 0
			if stopWords[strings.ToLower(words[0])] {
				start = 1
			}
			end := min(start+4, len(words))
			specific := strings.Join(words[start:end], " ")
			if specific != primary {
				queries = append(queries, specific)
			}
		}
	}

	return queries
}
