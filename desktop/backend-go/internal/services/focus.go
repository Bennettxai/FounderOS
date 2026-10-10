package services

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FocusSettings represents the effective settings for a focus mode
type FocusSettings struct {
	Name                 string
	DisplayName          string
	EffectiveModel       *string
	Temperature          float64
	MaxTokens            int
	OutputStyle          string // concise, balanced, detailed, structured
	ResponseFormat       string // markdown, plain, json, artifact
	MaxResponseLength    *int
	RequireSources       bool
	AutoSearch           bool
	SearchDepth          string // quick, standard, deep
	KBContextLimit       int
	IncludeHistoryCount  int
	ThinkingEnabled      bool
	ThinkingStyle        *string
	SystemPromptPrefix   string
	SystemPromptSuffix   string
	CustomSystemPrompt   string
	AutoLoadKBCategories []string
}

// FocusContext represents the pre-flight context to inject
type FocusContext struct {
	SystemPrompt      string               // Combined system prompt
	KBContext         []KBContextItem      // Knowledge base items to include
	SearchContext     []SearchContextItem  // Web search results to include
	ProjectContext    []ProjectContextItem // Project context to include
	OutputConstraints OutputConstraints    // Server-side output constraints
	LLMOptions        LLMOptions           // LLM configuration
}

// KBContextItem represents a knowledge base item to inject
type KBContextItem struct {
	ID       uuid.UUID
	Title    string
	Content  string
	Category string
}

// SearchContextItem represents a search result to inject
type SearchContextItem struct {
	Title   string
	URL     string
	Snippet string
	Source  string
}

// ProjectContextItem represents project context to inject
type ProjectContextItem struct {
	ID          uuid.UUID
	Name        string
	Description string
	Status      string
}

// OutputConstraints defines server-side constraints for focus modes
type OutputConstraints struct {
	MaxLength       *int   // Maximum response length in chars
	Style           string // concise, balanced, detailed, structured
	Format          string // markdown, plain, json, artifact
	RequireSources  bool   // Must include sources/citations
	RequireArtifact bool   // Should generate artifact for long content
}

// FocusService handles focus mode configuration and context injection
type FocusService struct {
	pool *pgxpool.Pool
}

// NewFocusService creates a new focus service
func NewFocusService(pool *pgxpool.Pool) *FocusService {
	return &FocusService{pool: pool}
}

// focusModeDefaults contains hardcoded defaults for focus modes
var focusModeDefaults = map[string]*FocusSettings{
	"quick": {
		Name:               "quick",
		DisplayName:        "Quick",
		Temperature:        0.5,
		MaxTokens:          2048,
		OutputStyle:        "concise",
		ResponseFormat:     "markdown",
		ThinkingEnabled:    false,
		SystemPromptPrefix: "You are in Quick Mode. Provide brief, direct answers. Be concise and to the point. Avoid unnecessary elaboration.",
	},
	"deep": {
		Name:            "deep",
		DisplayName:     "Deep Research",
		Temperature:     0.7,
		MaxTokens:       8192,
		OutputStyle:     "detailed",
		ResponseFormat:  "markdown",
		AutoSearch:      true,
		SearchDepth:     "deep",
		RequireSources:  true,
		ThinkingEnabled: true,
		SystemPromptPrefix: `You are in Deep Research Mode with LIVE WEB SEARCH results provided below.

CRITICAL INSTRUCTIONS:
1. You MUST base your response primarily on the Search Results provided below
2. DO NOT make up or hallucinate information - only use data from the search results
3. Reference sources inline when making claims using [Source Name](URL) format
4. If the search results don't contain enough information, clearly state what is missing
5. ALWAYS end your response with a "## Sources" section listing ALL sources used

RESPONSE FORMAT:
- Start with a clear, comprehensive answer
- Use markdown formatting for readability
- End with:

## Sources
- [Source 1 Title](url1)
- [Source 2 Title](url2)
...

Your response should synthesize information from the search results to answer the user's question comprehensively.`,
	},
	"creative": {
		Name:               "creative",
		DisplayName:        "Creative",
		Temperature:        0.9,
		MaxTokens:          4096,
		OutputStyle:        "balanced",
		ResponseFormat:     "markdown",
		ThinkingEnabled:    true,
		SystemPromptPrefix: "You are in Creative Mode. Think outside the box. Explore unconventional ideas and approaches. Be imaginative and innovative in your responses.",
	},
	"analyze": {
		Name:            "analyze",
		DisplayName:     "Analysis",
		Temperature:     0.6,
		MaxTokens:       6144,
		OutputStyle:     "structured",
		ResponseFormat:  "markdown",
		ThinkingEnabled: true,
		SystemPromptPrefix: `You are in Analyst Mode.

Your job is to produce correct, decision-grade analysis.

Process (follow every time):
1) Clarify the objective: what decision will this inform?
2) Identify inputs and assumptions: what is known vs unknown?
3) Choose an analysis approach: break the problem into parts and analyze each.
4) Synthesize: connect findings into a coherent narrative.
5) Recommend: propose specific next actions, trade-offs, and risks.

Output requirements:
- Use clear section headers.
- Quantify when possible; label estimates and confidence.
- If data is missing, state exactly what's missing and provide a best-effort path forward.
- Prefer actionable conclusions over generic commentary.`,
	},
	"write": {
		Name:               "write",
		DisplayName:        "Writing",
		Temperature:        0.7,
		MaxTokens:          8192,
		OutputStyle:        "detailed",
		ResponseFormat:     "artifact",
		ThinkingEnabled:    false,
		SystemPromptPrefix: "You are in Writing Mode. Create well-structured, polished content. Focus on clarity, flow, and appropriate tone. Generate artifacts for longer documents.",
	},
	"plan": {
		Name:               "plan",
		DisplayName:        "Planning",
		Temperature:        0.6,
		MaxTokens:          6144,
		OutputStyle:        "structured",
		ResponseFormat:     "markdown",
		ThinkingEnabled:    true,
		SystemPromptPrefix: "You are in Planning Mode. Create actionable plans with clear steps. Consider dependencies and timelines. Structure output as organized lists or project artifacts.",
	},
	"code": {
		Name:            "code",
		DisplayName:     "Coding",
		Temperature:     0.4,
		MaxTokens:       8192,
		OutputStyle:     "structured",
		ResponseFormat:  "artifact",
		ThinkingEnabled: true,
		SystemPromptPrefix: `You are in Coder Mode.

Your job is to deliver working, production-quality code changes.

Rules:
1) Make the smallest correct change that fully solves the request.
2) Respect the existing codebase style and conventions.
3) Prefer fixing the root cause over patching symptoms.
4) When requirements are ambiguous, ask 1–3 precise clarifying questions OR pick the simplest reasonable interpretation and state assumptions.
5) Verify your work logically: consider edge cases, error handling, and backward compatibility.

Output requirements:
- Provide the implementation directly.
- If code is substantial, organize by file/area.
- Generate a code artifact when delivering multi-file implementations.`,
	},
	"research": {
		Name:            "research",
		DisplayName:     "Research",
		Temperature:     0.7,
		MaxTokens:       8192,
		OutputStyle:     "detailed",
		ResponseFormat:  "markdown",
		AutoSearch:      true,
		SearchDepth:     "deep",
		RequireSources:  true,
		ThinkingEnabled: true,
		SystemPromptPrefix: `You are in Researcher Mode.

You will receive LIVE WEB SEARCH results.

Hard rules:
1) Base claims on the provided search results; do not invent facts.
2) When information conflicts, explain the discrepancy and weigh sources.
3) If evidence is insufficient, say what's missing and how to verify.

Method:
- Extract key claims from sources.
- Group claims by sub-question.
- Synthesize into a concise answer first, then add details.

Output requirements:
- Use markdown.
- Include inline citations as [Source Title](URL) for key claims.
- End with a "## Sources" section listing all sources used.`,
	},
	"build": {
		Name:               "build",
		DisplayName:        "Build",
		Temperature:        0.5,
		MaxTokens:          8192,
		OutputStyle:        "structured",
		ResponseFormat:     "artifact",
		ThinkingEnabled:    true,
		SystemPromptPrefix: "You are in Build Mode. Focus on implementation and construction. Create concrete deliverables. Generate artifacts for documents, code, or plans.",
	},
}
