package services

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ============================================================================
// INTENT CLASSIFICATION TYPES
// ============================================================================

// IntentType represents the classified intent of a user message
type IntentType string

const (
	IntentTypeChat       IntentType = "chat"       // Standard LLM response
	IntentTypeSearch     IntentType = "search"     // Web search required
	IntentTypeAgent      IntentType = "agent"      // Agent delegation via @mention
	IntentTypeCommand    IntentType = "command"    // Slash command execution
	IntentTypeCode       IntentType = "code"       // Code generation/review
	IntentTypeAnalysis   IntentType = "analysis"   // Data analysis task
	IntentTypeWriting    IntentType = "writing"    // Writing/document task
	IntentTypePlanning   IntentType = "planning"   // Planning/strategy task
	IntentTypeResearch   IntentType = "research"   // Research task (search + synthesis)
)

// IntentConfidence represents how confident we are in the classification
type IntentConfidence string

const (
	ConfidenceHigh   IntentConfidence = "high"
	ConfidenceMedium IntentConfidence = "medium"
	ConfidenceLow    IntentConfidence = "low"
)

// RoutingDecision contains the full routing analysis
type RoutingDecision struct {
	// Primary intent
	Intent     IntentType       `json:"intent"`
	Confidence IntentConfidence `json:"confidence"`

	// Extracted entities
	Mentions    []string `json:"mentions,omitempty"`    // @agent mentions
	Command     string   `json:"command,omitempty"`     // /command if present
	CommandArgs string   `json:"command_args,omitempty"`

	// Search indicators
	RequiresSearch   bool   `json:"requires_search"`
	SearchQuery      string `json:"search_query,omitempty"`
	SearchDepth      string `json:"search_depth,omitempty"` // basic, deep

	// Routing recommendations
	SuggestedAgent   string `json:"suggested_agent,omitempty"`
	SuggestedFocus   string `json:"suggested_focus,omitempty"`
	RequiresThinking bool   `json:"requires_thinking"`

	// Original message
	OriginalMessage string `json:"original_message"`
	CleanedMessage  string `json:"cleaned_message"` // Without @mentions and /commands
}

// ============================================================================
// ROUTER SERVICE
// ============================================================================

// RouterService handles intent classification and routing decisions
type RouterService struct {
	pool              *pgxpool.Pool
	delegationService *DelegationService
	commandService    *CommandService
	searchPatterns    []*searchPattern
	intentPatterns    map[IntentType][]*regexp.Regexp
}

// searchPattern defines a pattern that triggers search
type searchPattern struct {
	pattern    *regexp.Regexp
	depth      string // basic or deep
	confidence IntentConfidence
}

// NewRouterService creates a new router service
func NewRouterService(pool *pgxpool.Pool) *RouterService {
	r := &RouterService{
		pool:           pool,
		searchPatterns: buildSearchPatterns(),
		intentPatterns: buildIntentPatterns(),
	}

	// Initialize delegation and command services if pool is available
	if pool != nil {
		r.delegationService = NewDelegationService(pool)
		r.commandService = NewCommandService(pool)
	}

	return r
}

// buildSearchPatterns creates regex patterns that indicate search is needed
func buildSearchPatterns() []*searchPattern {
	return []*searchPattern{
		// Current events / news (high confidence, deep search)
		{regexp.MustCompile(`(?i)\b(latest|recent|current|today|yesterday|this week|this month|news|update|announced)\b`), "deep", ConfidenceHigh},

		// Explicit search requests (high confidence)
		{regexp.MustCompile(`(?i)\b(search|look up|find|google|find out|search for)\b`), "basic", ConfidenceHigh},

		// Factual questions (medium confidence)
		{regexp.MustCompile(`(?i)^(what is|what are|who is|who are|when did|when was|where is|where are)\b`), "basic", ConfidenceMedium},
		{regexp.MustCompile(`(?i)\b(how much|how many|how long|how old)\b`), "basic", ConfidenceMedium},

		// Research indicators (high confidence, deep search)
		{regexp.MustCompile(`(?i)\b(research|study|statistics|data on|report on|analysis of)\b`), "deep", ConfidenceHigh},

		// Comparison requests (medium confidence)
		{regexp.MustCompile(`(?i)\b(compare|comparison|vs|versus|difference between|better than)\b`), "basic", ConfidenceMedium},

		// Technical documentation (medium confidence)
		{regexp.MustCompile(`(?i)\b(documentation|docs|api reference|official guide)\b`), "basic", ConfidenceMedium},

		// Price/availability (high confidence)
		{regexp.MustCompile(`(?i)\b(price|cost|buy|purchase|available|stock|where to buy)\b`), "basic", ConfidenceHigh},

		// External knowledge (medium confidence)
		{regexp.MustCompile(`(?i)\b(according to|based on|sources say|experts say)\b`), "basic", ConfidenceMedium},
	}
}

// buildIntentPatterns creates patterns for intent classification
func buildIntentPatterns() map[IntentType][]*regexp.Regexp {
	return map[IntentType][]*regexp.Regexp{
		IntentTypeCode: {
			regexp.MustCompile(`(?i)\b(write code|code|implement|function|class|method|bug|debug|fix|error|exception|compile|build|test|unit test)\b`),
			regexp.MustCompile(`(?i)\b(javascript|typescript|python|go|golang|rust|java|c\+\+|react|svelte|node)\b`),
			regexp.MustCompile(`(?i)\b(refactor|optimize|improve performance|code review)\b`),
		},
		IntentTypeAnalysis: {
			regexp.MustCompile(`(?i)\b(analyze|analysis|data|metrics|statistics|trends|patterns|insights)\b`),
			regexp.MustCompile(`(?i)\b(chart|graph|visualization|dashboard|report)\b`),
			regexp.MustCompile(`(?i)\b(calculate|compute|evaluate|assess|measure)\b`),
		},
		IntentTypeWriting: {
			regexp.MustCompile(`(?i)\b(write|draft|compose|create|document|article|blog|post|email|letter)\b`),
			regexp.MustCompile(`(?i)\b(rewrite|edit|proofread|summarize|paraphrase)\b`),
			regexp.MustCompile(`(?i)\b(proposal|report|memo|brief|announcement)\b`),
		},
		IntentTypePlanning: {
			regexp.MustCompile(`(?i)\b(plan|planning|strategy|roadmap|timeline|schedule|milestone)\b`),
			regexp.MustCompile(`(?i)\b(project plan|action items|next steps|priorities|goals)\b`),
			regexp.MustCompile(`(?i)\b(brainstorm|ideas|options|alternatives|approach)\b`),
		},
		IntentTypeResearch: {
			regexp.MustCompile(`(?i)\b(research|investigate|explore|deep dive|comprehensive|thorough)\b`),
			regexp.MustCompile(`(?i)\b(find out everything|tell me everything|all about|in depth)\b`),
		},
	}
}

// Route analyzes a message and returns routing decision
func (r *RouterService) Route(ctx context.Context, message string, conversationID string) (*RoutingDecision, error) {
	decision := &RoutingDecision{
		Intent:          IntentTypeChat,
		Confidence:      ConfidenceMedium,
		OriginalMessage: message,
		CleanedMessage:  message,
	}

	// Step 1: Check for slash command (highest priority)
	if strings.HasPrefix(strings.TrimSpace(message), "/") {
		r.parseCommand(message, decision)
		if decision.Command != "" {
			decision.Intent = IntentTypeCommand
			decision.Confidence = ConfidenceHigh
			return decision, nil
		}
	}

	// Step 2: Extract @mentions
	r.extractMentions(message, decision)
	if len(decision.Mentions) > 0 {
		decision.Intent = IntentTypeAgent
		decision.Confidence = ConfidenceHigh
		// Suggest the first mentioned agent
		decision.SuggestedAgent = decision.Mentions[0]
	}

	// Step 3: Check for search requirement
	r.analyzeSearchNeed(message, decision)

	// Step 4: Classify intent based on patterns (if not already determined)
	if decision.Intent == IntentTypeChat {
		r.classifyIntent(message, decision)
	}

	// Step 5: Determine suggested focus mode
	r.suggestFocusMode(decision)

	// Step 6: Determine if thinking/reasoning would help
	r.analyzeThinkingNeed(message, decision)

	// Clean message (remove mentions and commands)
	decision.CleanedMessage = r.cleanMessage(message, decision)

	slog.Debug("Routing decision made",
		"intent", decision.Intent,
		"confidence", decision.Confidence,
		"requires_search", decision.RequiresSearch,
		"suggested_agent", decision.SuggestedAgent,
		"suggested_focus", decision.SuggestedFocus,
	)

	return decision, nil
}

// parseCommand extracts slash command from message
func (r *RouterService) parseCommand(message string, decision *RoutingDecision) {
	// Match /command pattern
	cmdPattern := regexp.MustCompile(`^/(\w+)(?:\s+(.*))?$`)
	trimmed := strings.TrimSpace(message)

	matches := cmdPattern.FindStringSubmatch(trimmed)
	if len(matches) >= 2 {
		decision.Command = matches[1]
		if len(matches) >= 3 {
			decision.CommandArgs = strings.TrimSpace(matches[2])
		}
	}
}

// extractMentions finds all @agent mentions in the message
func (r *RouterService) extractMentions(message string, decision *RoutingDecision) {
	// Match @agent-name pattern (lowercase, hyphens allowed)
	mentionPattern := regexp.MustCompile(`@([a-z][a-z0-9-]*)`)
	matches := mentionPattern.FindAllStringSubmatch(message, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 && !seen[match[1]] {
			decision.Mentions = append(decision.Mentions, match[1])
			seen[match[1]] = true
		}
	}
}

// analyzeSearchNeed determines if web search is required
func (r *RouterService) analyzeSearchNeed(message string, decision *RoutingDecision) {
	for _, sp := range r.searchPatterns {
		if sp.pattern.MatchString(message) {
			decision.RequiresSearch = true
			decision.SearchDepth = sp.depth

			// Use higher confidence if found
			if sp.confidence == ConfidenceHigh || decision.Confidence == ConfidenceLow {
				decision.Confidence = sp.confidence
			}

			// Build search query from message
			decision.SearchQuery = r.buildSearchQuery(message)
			break
		}
	}

	// Also trigger search for research intent
	if decision.Intent == IntentTypeResearch {
		decision.RequiresSearch = true
		decision.SearchDepth = "deep"
		if decision.SearchQuery == "" {
			decision.SearchQuery = r.buildSearchQuery(message)
		}
	}
}

// buildSearchQuery extracts a search query from the message
func (r *RouterService) buildSearchQuery(message string) string {
	optimizer := NewQueryOptimizer()
	return optimizer.OptimizeQuery(message)
}

// classifyIntent determines the primary intent based on patterns
func (r *RouterService) classifyIntent(message string, decision *RoutingDecision) {
	// Score each intent type
	scores := make(map[IntentType]int)

	for intentType, patterns := range r.intentPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(message) {
				scores[intentType]++
			}
		}
	}

	// Find highest scoring intent
	maxScore := 0
	for intentType, score := range scores {
		if score > maxScore {
			maxScore = score
			decision.Intent = intentType
		}
	}

	// Set confidence based on score
	if maxScore >= 3 {
		decision.Confidence = ConfidenceHigh
	} else if maxScore >= 2 {
		decision.Confidence = ConfidenceMedium
	} else if maxScore >= 1 {
		decision.Confidence = ConfidenceLow
	}
}

// suggestFocusMode recommends a focus mode based on intent
func (r *RouterService) suggestFocusMode(decision *RoutingDecision) {
	switch decision.Intent {
	case IntentTypeCode:
		decision.SuggestedFocus = "code"
	case IntentTypeAnalysis:
		decision.SuggestedFocus = "analyze"
	case IntentTypeWriting:
		decision.SuggestedFocus = "write"
	case IntentTypePlanning:
		decision.SuggestedFocus = "plan"
	case IntentTypeResearch:
		decision.SuggestedFocus = "deep"
	case IntentTypeSearch:
		decision.SuggestedFocus = "quick"
	default:
		// Don't suggest if unclear
	}
}

// analyzeThinkingNeed determines if extended thinking would help
func (r *RouterService) analyzeThinkingNeed(message string, decision *RoutingDecision) {
	// Patterns that benefit from extended thinking
	thinkingPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(complex|complicated|difficult|tricky|challenging)\b`),
		regexp.MustCompile(`(?i)\b(step by step|walk me through|explain your reasoning)\b`),
		regexp.MustCompile(`(?i)\b(analyze|evaluate|compare|contrast|pros and cons)\b`),
		regexp.MustCompile(`(?i)\b(design|architect|plan|strategy)\b`),
		regexp.MustCompile(`(?i)\b(debug|troubleshoot|diagnose|fix)\b`),
		regexp.MustCompile(`(?i)\b(optimize|improve|refactor)\b`),
		regexp.MustCompile(`(?i)\b(why|how|explain|reasoning)\b`),
	}

	for _, pattern := range thinkingPatterns {
		if pattern.MatchString(message) {
			decision.RequiresThinking = true
			break
		}
	}

	// Also enable thinking for complex intents
	switch decision.Intent {
	case IntentTypeAnalysis, IntentTypePlanning, IntentTypeResearch:
		decision.RequiresThinking = true
	case IntentTypeCode:
		// Enable for debugging/refactoring
		if regexp.MustCompile(`(?i)\b(debug|refactor|optimize|fix)\b`).MatchString(message) {
			decision.RequiresThinking = true
		}
	}
}

// cleanMessage removes @mentions and /commands from the message
func (r *RouterService) cleanMessage(message string, decision *RoutingDecision) string {
	cleaned := message

	// Remove @mentions
	mentionPattern := regexp.MustCompile(`@[a-z][a-z0-9-]*`)
	cleaned = mentionPattern.ReplaceAllString(cleaned, "")

	// Remove /command if present
	if decision.Command != "" {
		cmdPattern := regexp.MustCompile(`^/\w+\s*`)
		cleaned = cmdPattern.ReplaceAllString(cleaned, "")
	}

	// Clean up whitespace
	cleaned = strings.TrimSpace(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")

	return cleaned
}

// ============================================================================
// CONVENIENCE METHODS
// ============================================================================

// ShouldSearch returns true if the routing indicates search is needed
func (d *RoutingDecision) ShouldSearch() bool {
	return d.RequiresSearch
}

// ShouldDelegate returns true if the message should be delegated to an agent
func (d *RoutingDecision) ShouldDelegate() bool {
	return d.Intent == IntentTypeAgent && len(d.Mentions) > 0
}

// ShouldExecuteCommand returns true if a slash command was found
func (d *RoutingDecision) ShouldExecuteCommand() bool {
	return d.Intent == IntentTypeCommand && d.Command != ""
}

// ShouldUseThinking returns true if extended thinking is recommended
func (d *RoutingDecision) ShouldUseThinking() bool {
	return d.RequiresThinking
}

// IsHighConfidence returns true if we're confident in the routing
func (d *RoutingDecision) IsHighConfidence() bool {
	return d.Confidence == ConfidenceHigh
}

// GetPrimaryMention returns the first @mention, or empty string
func (d *RoutingDecision) GetPrimaryMention() string {
	if len(d.Mentions) > 0 {
		return d.Mentions[0]
	}
	return ""
}

// ============================================================================
// AUTO-AGENT SELECTION
// ============================================================================

// agentMapping maps intent types to suggested agents
var agentMapping = map[IntentType]string{
	IntentTypeCode:     "coder",
	IntentTypeAnalysis: "analyst",
	IntentTypeWriting:  "writer",
	IntentTypePlanning: "planner",
	IntentTypeResearch: "researcher",
}

// SuggestAgent returns a suggested agent based on intent
func (r *RouterService) SuggestAgent(decision *RoutingDecision) string {
	// If already has a mention, use that
	if len(decision.Mentions) > 0 {
		return decision.Mentions[0]
	}

	// Otherwise suggest based on intent
	if agent, ok := agentMapping[decision.Intent]; ok {
		return agent
	}

	return ""
}

// ============================================================================
// BULK ROUTING
// ============================================================================

// RouteMultiple analyzes multiple messages and returns routing decisions
func (r *RouterService) RouteMultiple(ctx context.Context, messages []string, conversationID string) ([]*RoutingDecision, error) {
	decisions := make([]*RoutingDecision, len(messages))

	for i, msg := range messages {
		decision, err := r.Route(ctx, msg, conversationID)
		if err != nil {
			return nil, err
		}
		decisions[i] = decision
	}

	return decisions, nil
}

// QuickRoute provides a simplified routing result
func (r *RouterService) QuickRoute(ctx context.Context, message string) (IntentType, bool, string) {
	decision, err := r.Route(ctx, message, "")
	if err != nil {
		return IntentTypeChat, false, ""
	}

	return decision.Intent, decision.RequiresSearch, decision.SuggestedAgent
}
