package subconscious

import (
	"testing"
)

func TestParseClassification(t *testing.T) {
	tests := []struct {
		name     string
		response string
		wantG    string
		wantW    float64
	}{
		{
			name:     "standard response",
			response: "GENRE: DIRECT\nWEIGHT: 0.8",
			wantG:    "DIRECT",
			wantW:    0.8,
		},
		{
			name:     "lowercase genre",
			response: "genre: express\nweight: 0.3",
			wantG:    "EXPRESS",
			wantW:    0.3,
		},
		{
			name:     "extra whitespace",
			response: "  GENRE:  COMMIT  \n  WEIGHT:  0.9  ",
			wantG:    "COMMIT",
			wantW:    0.9,
		},
		{
			name:     "garbage response returns defaults",
			response: "I don't know what you mean",
			wantG:    "INFORM",
			wantW:    0.5,
		},
		{
			name:     "partial response (genre only)",
			response: "GENRE: DECIDE",
			wantG:    "DECIDE",
			wantW:    0.5, // default
		},
		{
			name:     "out of range weight clamps to default",
			response: "GENRE: INFORM\nWEIGHT: 2.5",
			wantG:    "INFORM",
			wantW:    0.5, // 2.5 > 1.0 so stays default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseClassification(tt.response)
			if string(result.Genre) != tt.wantG {
				t.Errorf("genre = %s, want %s", result.Genre, tt.wantG)
			}
			if result.Weight != tt.wantW {
				t.Errorf("weight = %f, want %f", result.Weight, tt.wantW)
			}
		})
	}
}

func TestPatternExtractorReEncoding(t *testing.T) {
	pe := NewPatternExtractor()

	// Identical messages should NOT be flagged (sim > 0.95)
	p := pe.Extract("hello world", "hello world", "")
	if p.IsReEncoding {
		t.Error("identical messages should not be re-encoding")
	}

	// Completely different messages should NOT be flagged
	p = pe.Extract("what is the weather", "implement authentication", "")
	if p.IsReEncoding {
		t.Error("completely different messages should not be re-encoding")
	}

	// Similar but not identical = rephrase (rephrased request)
	p = pe.Extract("can you display all the users please", "show me the list of users", "")
	if !p.IsReEncoding {
		t.Error("similar messages should be detected as re-encoding")
	}
}

func TestPatternExtractorFrustration(t *testing.T) {
	pe := NewPatternExtractor()

	tests := []struct {
		msg  string
		want bool
	}{
		{"I said I want the blue one", true},
		{"No, I meant the other file", true},
		{"That's not what I asked for", true},
		{"Can you help me with this?", false},
		{"Try again please", true},
	}

	for _, tt := range tests {
		p := pe.Extract(tt.msg, "", "")
		if p.IsFrustration != tt.want {
			t.Errorf("frustration(%q) = %v, want %v", tt.msg, p.IsFrustration, tt.want)
		}
	}
}

func TestPatternExtractorFeedbackClosure(t *testing.T) {
	pe := NewPatternExtractor()

	closureMessages := []string{
		"thanks", "Thank you!", "perfect", "great, that works",
		"got it", "looks good", "exactly!", "yes",
	}

	for _, msg := range closureMessages {
		p := pe.Extract(msg, "", "")
		if !p.IsFeedbackClosed {
			t.Errorf("expected feedback closure for %q", msg)
		}
	}

	nonClosureMessages := []string{
		"can you also add tests", "what about error handling",
		"now implement the second part",
	}

	for _, msg := range nonClosureMessages {
		p := pe.Extract(msg, "", "")
		if p.IsFeedbackClosed {
			t.Errorf("unexpected feedback closure for %q", msg)
		}
	}
}

func TestPatternExtractorPreference(t *testing.T) {
	pe := NewPatternExtractor()

	tests := []struct {
		msg      string
		wantPref bool
	}{
		{"I prefer dark mode", true},
		{"Please always use TypeScript", true},
		{"From now on, write tests first", true},
		{"Never use var in JavaScript", true},
		{"Can you fix this bug?", false},
	}

	for _, tt := range tests {
		p := pe.Extract(tt.msg, "", "")
		if p.HasPreference != tt.wantPref {
			t.Errorf("preference(%q) = %v, want %v", tt.msg, p.HasPreference, tt.wantPref)
		}
		if tt.wantPref && p.PreferenceText == "" {
			t.Errorf("preference text should not be empty for %q", tt.msg)
		}
	}
}

func TestPatternExtractorBounce(t *testing.T) {
	pe := NewPatternExtractor()

	bounceResp := "I think this would be better handled by the build agent. Let me switch to build mode."
	p := pe.Extract("build the project", "", bounceResp)
	if !p.IsBounce {
		t.Error("expected bounce detection")
	}

	normalResp := "Here is the list of files in the project."
	p = pe.Extract("show files", "", normalResp)
	if p.IsBounce {
		t.Error("unexpected bounce detection")
	}
}

func TestLevenshteinSimilarity(t *testing.T) {
	tests := []struct {
		a, b   string
		minSim float64
		maxSim float64
	}{
		{"hello", "hello", 1.0, 1.0},
		{"", "", 1.0, 1.0},
		{"abc", "xyz", 0.0, 0.1},
		{"kitten", "sitting", 0.4, 0.6},
		{"show me users", "show me the users", 0.7, 0.9},
	}

	for _, tt := range tests {
		sim := normalizedLevenshteinSimilarity(tt.a, tt.b)
		if sim < tt.minSim || sim > tt.maxSim {
			t.Errorf("similarity(%q, %q) = %f, want [%f, %f]",
				tt.a, tt.b, sim, tt.minSim, tt.maxSim)
		}
	}
}

func TestObserverNilComponents(t *testing.T) {
	// Observer should not panic with nil components
	obs := NewObserver(nil, nil, nil, nil, nil)
	obs.observe(nil, ObserveInput{
		UserID:      "test",
		UserMessage: "hello",
	})
	// No panic = pass
}

func TestAppendLine(t *testing.T) {
	// Basic append
	result := appendLine("", "first line")
	if result != "first line" {
		t.Errorf("expected 'first line', got %q", result)
	}

	result = appendLine("first line", "second line")
	if result != "first line\nsecond line" {
		t.Errorf("expected two lines, got %q", result)
	}
}

func TestTruncate(t *testing.T) {
	if truncate("hello", 3) != "hel" {
		t.Error("truncate failed")
	}
	if truncate("hi", 10) != "hi" {
		t.Error("truncate should not extend")
	}
}
