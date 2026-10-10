package subconscious

import (
	"strings"
	"unicode/utf8"
)

// ExtractedPatterns holds the heuristic detection results for a single turn.
type ExtractedPatterns struct {
	IsReEncoding     bool    // user rephrased previous message
	ReEncodingSim    float64 // similarity score [0, 1]
	IsBounce         bool    // response switched modes
	IsGenreMatch     bool    // classifier genre matches expected
	IsFeedbackClosed bool    // user acknowledged/confirmed
	IsFrustration    bool    // user shows frustration signals
	HasPreference    bool    // user stated a preference
	PreferenceText   string  // extracted preference text (if any)
}

// PatternExtractor performs heuristic pattern detection on conversation turns.
// No LLM calls — all detection is string-based for $0 cost.
type PatternExtractor struct{}

// NewPatternExtractor creates a new extractor.
func NewPatternExtractor() *PatternExtractor {
	return &PatternExtractor{}
}

// Extract detects patterns from the current turn's messages.
func (pe *PatternExtractor) Extract(currentUserMsg, previousUserMsg, assistantResponse string) ExtractedPatterns {
	p := ExtractedPatterns{}

	// 1. Re-encoding detection: user rephrased previous message
	if previousUserMsg != "" {
		p.ReEncodingSim = normalizedLevenshteinSimilarity(
			strings.ToLower(previousUserMsg),
			strings.ToLower(currentUserMsg),
		)
		// Similarity > 0.25 AND < 0.95 = rephrase (not identical, but similar enough)
		p.IsReEncoding = p.ReEncodingSim > 0.25 && p.ReEncodingSim < 0.95
	}

	// 2. Bounce detection: mode-switching keywords in response
	p.IsBounce = detectBounce(assistantResponse)

	// 3. Genre match: assume correct unless explicitly wrong (classifier handles this)
	p.IsGenreMatch = true

	// 4. Feedback closure: user acknowledged/confirmed
	p.IsFeedbackClosed = detectFeedbackClosure(currentUserMsg)

	// 5. Frustration detection
	p.IsFrustration = detectFrustration(currentUserMsg)

	// 6. Preference detection
	p.HasPreference, p.PreferenceText = detectPreference(currentUserMsg)

	return p
}

// detectBounce checks if the response contains mode-switching signals.
func detectBounce(response string) bool {
	lower := strings.ToLower(response)
	bounceSignals := []string{
		"let me switch to",
		"switching to",
		"that's outside my",
		"i'll hand this off",
		"this would be better handled",
		"not the right mode",
		"let me redirect",
	}
	for _, sig := range bounceSignals {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

// detectFeedbackClosure checks for acknowledgment/confirmation tokens.
func detectFeedbackClosure(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))

	// Short acknowledgments
	closureTokens := []string{
		"thanks", "thank you", "perfect", "great", "got it",
		"looks good", "that works", "exactly", "nice", "awesome",
		"understood", "makes sense", "yes", "yep", "yeah",
		"correct", "right", "ok", "okay",
	}
	for _, tok := range closureTokens {
		if lower == tok || strings.HasPrefix(lower, tok+".") ||
			strings.HasPrefix(lower, tok+"!") || strings.HasPrefix(lower, tok+",") {
			return true
		}
	}
	return false
}

// detectFrustration checks for frustration markers.
func detectFrustration(msg string) bool {
	lower := strings.ToLower(msg)
	frustrationMarkers := []string{
		"i said", "i already", "i told you", "no, i meant",
		"that's not what", "wrong", "not what i asked",
		"you're not understanding", "please listen",
		"try again", "that's incorrect",
	}
	for _, marker := range frustrationMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// detectPreference checks if the user is stating a preference.
func detectPreference(msg string) (bool, string) {
	lower := strings.ToLower(msg)
	prefixMarkers := []string{
		"i prefer", "i always prefer", "i always want", "please always",
		"from now on", "i like", "don't use", "never use",
		"always use", "i want you to", "i'd prefer",
	}
	for _, prefix := range prefixMarkers {
		if idx := strings.Index(lower, prefix); idx != -1 {
			// Extract the preference text (up to 200 chars)
			prefText := msg[idx:]
			if len(prefText) > 200 {
				prefText = prefText[:200]
			}
			return true, strings.TrimSpace(prefText)
		}
	}
	return false, ""
}

// normalizedLevenshteinSimilarity computes 1 - (editDistance / max(len(a), len(b))).
// Returns 0 for completely different strings, 1 for identical.
func normalizedLevenshteinSimilarity(a, b string) float64 {
	la := utf8.RuneCountInString(a)
	lb := utf8.RuneCountInString(b)
	if la == 0 && lb == 0 {
		return 1.0
	}
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}

	dist := levenshteinDistance([]rune(a), []rune(b))
	return 1.0 - float64(dist)/float64(maxLen)
}

// levenshteinDistance computes the edit distance between two rune slices.
// Optimized to use O(min(m,n)) space.
func levenshteinDistance(a, b []rune) int {
	if len(a) > len(b) {
		a, b = b, a
	}
	// Limit computation for very long strings (performance guard)
	if len(a) > 500 {
		a = a[:500]
	}
	if len(b) > 500 {
		b = b[:500]
	}

	prev := make([]int, len(a)+1)
	curr := make([]int, len(a)+1)
	for i := range prev {
		prev[i] = i
	}

	for j := 1; j <= len(b); j++ {
		curr[0] = j
		for i := 1; i <= len(a); i++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := curr[i-1] + 1
			del := prev[i] + 1
			sub := prev[i-1] + cost
			curr[i] = min3(ins, del, sub)
		}
		prev, curr = curr, prev
	}
	return prev[len(a)]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
