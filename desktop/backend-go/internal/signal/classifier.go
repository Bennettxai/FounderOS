package signal

import "strings"

// SignalEnvelope is the fast-path classification result.
// Produced by FastClassifier on every user message before agent execution.
type SignalEnvelope struct {
	Mode       Mode    // Detected OSA mode
	Genre      Genre   // Detected speech-act genre
	Weight     float64 // Information density [0,1]
	DocType    string  // "proposal", "sop", "report", "brief", "framework", ""
	Confidence float64 // Classification confidence [0,1]
}

// FastClassifier performs zero-LLM signal classification using keyword patterns.
// <1ms latency. Called synchronously on the hot path.
type FastClassifier struct{}

// NewFastClassifier creates a new FastClassifier.
func NewFastClassifier() *FastClassifier {
	return &FastClassifier{}
}

// Classify produces a SignalEnvelope from a user message.
// focusMode maps to OSA mode. hasProject/hasClient increase genre confidence.
func (fc *FastClassifier) Classify(msg, focusMode string, hasProject, hasClient bool) SignalEnvelope {
	lower := strings.ToLower(msg)

	mode := classifyMode(focusMode, lower)
	genre := classifyGenre(lower)
	docType := classifyDocType(lower)
	weight := classifyWeight(msg, lower)
	confidence := classifyConfidence(genre, docType, hasProject, hasClient)

	return SignalEnvelope{
		Mode:       mode,
		Genre:      genre,
		Weight:     weight,
		DocType:    docType,
		Confidence: confidence,
	}
}

func classifyMode(focusMode, lower string) Mode {
	// Focus mode takes precedence
	switch focusMode {
	case "write":
		return ModeBuild
	case "analyze", "research":
		return ModeAnalyze
	case "plan", "build":
		return ModeBuild
	case "maintain":
		return ModeMaintain
	}

	// Fallback to message content analysis
	directVerbs := []string{"create", "make", "build", "generate", "write", "draft", "produce", "send", "schedule", "update", "delete", "remove", "add"}
	for _, v := range directVerbs {
		if strings.Contains(lower, v) {
			return ModeExecute
		}
	}

	questionWords := []string{"what", "how", "why", "when", "where", "who", "which", "explain", "describe", "tell me"}
	for _, w := range questionWords {
		if strings.Contains(lower, w) {
			return ModeAssist
		}
	}

	analyzeWords := []string{"analyze", "compare", "evaluate", "assess", "review", "audit", "measure", "benchmark"}
	for _, w := range analyzeWords {
		if strings.Contains(lower, w) {
			return ModeAnalyze
		}
	}

	return ModeAssist // default
}

func classifyGenre(lower string) Genre {
	// DIRECT: imperative verbs (user wants action)
	directPatterns := []string{
		"create", "make", "build", "generate", "write", "draft", "produce",
		"send", "schedule", "update", "delete", "remove", "add", "set up",
		"implement", "deploy", "configure", "fix", "resolve",
	}
	for _, p := range directPatterns {
		if strings.Contains(lower, p) {
			return GenreDirect
		}
	}

	// DECIDE: choice/decision language
	decidePatterns := []string{
		"should i", "should we", "which one", "what's better", "choose between",
		"decide", "recommend", "pros and cons", "compare options", "trade-off",
		"best approach", "evaluate options",
	}
	for _, p := range decidePatterns {
		if strings.Contains(lower, p) {
			return GenreDecide
		}
	}

	// COMMIT: promise/commitment language
	commitPatterns := []string{
		"i will", "i'll", "we will", "we'll", "i commit", "let's plan",
		"schedule", "promise", "guarantee", "agree to", "sign off",
	}
	for _, p := range commitPatterns {
		if strings.Contains(lower, p) {
			return GenreCommit
		}
	}

	// EXPRESS: emotion/internal state
	expressPatterns := []string{
		"i feel", "frustrated", "happy", "concerned", "worried",
		"excited", "confused", "thank", "sorry", "appreciate",
		"love", "hate", "great job", "well done",
	}
	for _, p := range expressPatterns {
		if strings.Contains(lower, p) {
			return GenreExpress
		}
	}

	// INFORM: question words (default for information-seeking)
	informPatterns := []string{
		"what", "how", "why", "when", "where", "who", "which",
		"explain", "describe", "tell me", "show me", "list",
		"summarize", "overview", "status", "report",
	}
	for _, p := range informPatterns {
		if strings.Contains(lower, p) {
			return GenreInform
		}
	}

	return GenreInform // default: information-seeking
}

func classifyDocType(lower string) string {
	docPatterns := map[string][]string{
		"proposal":  {"proposal", "rfp", "bid", "pitch", "offer"},
		"sop":       {"sop", "standard operating procedure", "process document", "playbook", "runbook"},
		"report":    {"report", "analysis report", "status report", "quarterly", "monthly report"},
		"brief":     {"brief", "briefing", "executive summary", "one-pager"},
		"framework": {"framework", "methodology", "model", "architecture"},
		"guide":     {"guide", "tutorial", "how-to", "documentation", "manual"},
		"plan":      {"plan", "roadmap", "strategy", "timeline", "milestone"},
	}

	for docType, patterns := range docPatterns {
		for _, p := range patterns {
			if strings.Contains(lower, p) {
				return docType
			}
		}
	}
	return ""
}

func classifyWeight(msg, lower string) float64 {
	// Weight based on message complexity
	wordCount := len(strings.Fields(msg))

	// Short acknowledgments
	if wordCount <= 3 {
		return 0.2
	}

	// Questions
	if strings.Contains(lower, "?") {
		if wordCount <= 10 {
			return 0.4
		}
		return 0.6
	}

	// Medium complexity
	if wordCount <= 20 {
		return 0.5
	}

	// Detailed specifications
	if wordCount <= 50 {
		return 0.7
	}

	// Very detailed
	return 0.8
}

func classifyConfidence(genre Genre, docType string, hasProject, hasClient bool) float64 {
	confidence := 0.5 // base

	// DocType match boosts confidence
	if docType != "" {
		confidence += 0.2
	}

	// Context availability boosts confidence for DIRECT genre
	if genre == GenreDirect {
		if hasProject {
			confidence += 0.1
		}
		if hasClient {
			confidence += 0.1
		}
	}

	// Cap at 0.95
	if confidence > 0.95 {
		confidence = 0.95
	}

	return confidence
}
