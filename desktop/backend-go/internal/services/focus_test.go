package services

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestPreflightLatency measures Pre-flight pipeline overhead
// Target: < 2 seconds for complete preflight context building
func TestPreflightLatency(t *testing.T) {
	// Create focus service without DB for unit testing
	service := NewFocusService(nil)
	ctx := context.Background()

	tests := []struct {
		name       string
		focusMode  string
		maxLatency time.Duration
	}{
		{"Quick mode preflight", "quick", 100 * time.Millisecond},
		{"Deep mode preflight", "deep", 200 * time.Millisecond},
		{"Creative mode preflight", "creative", 100 * time.Millisecond},
		{"Analyze mode preflight", "analyze", 150 * time.Millisecond},
		{"Write mode preflight", "write", 100 * time.Millisecond},
		{"Plan mode preflight", "plan", 100 * time.Millisecond},
		{"Code mode preflight", "code", 150 * time.Millisecond},
		{"Research mode preflight", "research", 200 * time.Millisecond},
		{"Build mode preflight", "build", 150 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			// Get effective settings - this is the core preflight operation
			settings, err := service.GetEffectiveSettings(ctx, "", tt.focusMode)

			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("GetEffectiveSettings failed for mode %s: %v", tt.focusMode, err)
				return
			}

			if settings == nil {
				t.Errorf("GetEffectiveSettings returned nil for mode %s", tt.focusMode)
				return
			}

			if elapsed > tt.maxLatency {
				t.Errorf("Preflight latency %v exceeds target %v for mode %s",
					elapsed, tt.maxLatency, tt.focusMode)
			}

			t.Logf("Mode %s: latency=%v (target: <%v)", tt.focusMode, elapsed, tt.maxLatency)
		})
	}
}

// TestPreflightLatencyBenchmark runs a benchmark for preflight operations
func BenchmarkPreflightGetSettings(b *testing.B) {
	service := NewFocusService(nil)
	ctx := context.Background()

	modes := []string{"quick", "deep", "creative", "analyze", "write", "plan", "code", "research", "build"}

	for _, mode := range modes {
		b.Run(mode, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = service.GetEffectiveSettings(ctx, "", mode)
			}
		})
	}
}

// TestOutputConstraintsGeneration tests output constraints building latency
func TestOutputConstraintsLatency(t *testing.T) {
	service := NewFocusService(nil)

	modes := []string{"quick", "deep", "creative", "analyze", "write", "plan", "code", "research", "build"}
	maxLatency := 10 * time.Millisecond

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			settings := &FocusSettings{
				Name:        mode,
				OutputStyle: "balanced",
			}

			start := time.Now()
			constraints := service.buildOutputConstraints(settings)
			elapsed := time.Since(start)

			if constraints.Style == "" {
				t.Error("Output constraints style is empty")
			}

			if elapsed > maxLatency {
				t.Errorf("Output constraints latency %v exceeds target %v", elapsed, maxLatency)
			}

			t.Logf("Mode %s: constraints built in %v", mode, elapsed)
		})
	}
}

// TestKeywordExtractionLatency tests keyword extraction performance
func TestKeywordExtractionLatency(t *testing.T) {
	queries := []struct {
		name  string
		query string
	}{
		{"Short query", "how to fix bug"},
		{"Medium query", "I need help with the authentication system in my application"},
		{"Long query", "Can you help me analyze the performance issues in our database layer and suggest optimizations for the query patterns we're using in the user management module"},
		{"With special chars", "What's the status of project #123 & how do we deploy?"},
	}

	maxLatency := 5 * time.Millisecond

	for _, tt := range queries {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			keywords := extractKeywords(tt.query)
			elapsed := time.Since(start)

			if len(keywords) == 0 {
				t.Error("No keywords extracted")
			}

			if elapsed > maxLatency {
				t.Errorf("Keyword extraction latency %v exceeds target %v", elapsed, maxLatency)
			}

			t.Logf("Query '%s...': %d keywords in %v", tt.query[:minIntTest(30, len(tt.query))], len(keywords), elapsed)
		})
	}
}

// TestContextTypesForFocusMode validates context type mapping
func TestContextTypesForFocusMode(t *testing.T) {
	service := NewFocusService(nil)

	// Tests should match actual implementation in getContextTypesForFocusMode
	tests := []struct {
		mode          string
		expectedTypes []string
	}{
		{"code", []string{"PROJECT", "DOCUMENT", "CUSTOM"}},
		{"build", []string{"PROJECT", "DOCUMENT", "CUSTOM"}},
		{"write", []string{"DOCUMENT", "BUSINESS", "CUSTOM"}},
		{"analyze", []string{"BUSINESS", "PROJECT", "DOCUMENT"}},
		{"research", []string{"DOCUMENT", "CUSTOM", "BUSINESS"}},
		{"deep", []string{"DOCUMENT", "CUSTOM", "BUSINESS"}},
		{"plan", []string{"PROJECT", "BUSINESS", "DOCUMENT"}},
		{"quick", []string{"DOCUMENT", "PROJECT", "BUSINESS", "CUSTOM"}},       // default
		{"creative", []string{"CUSTOM", "DOCUMENT", "PERSON"}},
		{"unknown", []string{"DOCUMENT", "PROJECT", "BUSINESS", "CUSTOM"}},     // default
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			types := service.getContextTypesForFocusMode(tt.mode)

			if len(types) != len(tt.expectedTypes) {
				t.Errorf("Expected %d types, got %d for mode %s",
					len(tt.expectedTypes), len(types), tt.mode)
				return
			}

			for i, expectedType := range tt.expectedTypes {
				if types[i] != expectedType {
					t.Errorf("Expected type %s at index %d, got %s for mode %s",
						expectedType, i, types[i], tt.mode)
				}
			}
		})
	}
}

// TestOutputConstraintsInstructionsGeneration tests instruction generation
func TestOutputConstraintsInstructionsGeneration(t *testing.T) {
	service := NewFocusService(nil)

	maxLen500 := 500              // 500/4 = 125 words -> brief
	maxLen4000 := 4000            // 4000/4 = 1000 words -> moderate-length
	maxLen16000 := 16000          // 16000/4 = 4000 words -> comprehensive (>3000)

	tests := []struct {
		name        string
		constraints OutputConstraints
		shouldHave  []string
	}{
		{
			name: "Brief response",
			constraints: OutputConstraints{
				MaxLength: &maxLen500,
				Style:     "concise",
			},
			shouldHave: []string{"brief", "focused"},
		},
		{
			name: "Moderate response",
			constraints: OutputConstraints{
				MaxLength: &maxLen4000,
				Style:     "balanced",
			},
			shouldHave: []string{"moderate-length"},
		},
		{
			name: "Comprehensive response with sources",
			constraints: OutputConstraints{
				MaxLength:      &maxLen16000,
				Style:          "detailed",
				RequireSources: true,
			},
			shouldHave: []string{"comprehensive", "Sources"},
		},
		{
			name: "With artifact requirement",
			constraints: OutputConstraints{
				Style:           "structured",
				RequireArtifact: true,
			},
			shouldHave: []string{"artifact"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			instructions := service.GetOutputConstraintsInstructions(tt.constraints)
			elapsed := time.Since(start)

			for _, keyword := range tt.shouldHave {
				if !containsIgnoreCase(instructions, keyword) {
					t.Errorf("Instructions should contain '%s', got: %s", keyword, instructions)
				}
			}

			if elapsed > 5*time.Millisecond {
				t.Errorf("Instruction generation too slow: %v", elapsed)
			}

			t.Logf("Generated %d chars in %v", len(instructions), elapsed)
		})
	}
}

// TestDefaultMaxLengthValues validates max length defaults
func TestDefaultMaxLengthValues(t *testing.T) {
	service := NewFocusService(nil)

	tests := []struct {
		mode          string
		style         string
		expectedRange [2]int // min, max expected values
	}{
		{"quick", "concise", [2]int{1000, 3000}},
		{"deep", "detailed", [2]int{12000, 20000}},
		{"creative", "balanced", [2]int{6000, 10000}},
		{"analyze", "structured", [2]int{10000, 15000}},
		{"write", "detailed", [2]int{16000, 25000}},
		{"code", "balanced", [2]int{12000, 20000}},
	}

	for _, tt := range tests {
		t.Run(tt.mode+"_"+tt.style, func(t *testing.T) {
			maxLen := service.getDefaultMaxLength(tt.mode, tt.style)

			if maxLen == nil {
				t.Error("getDefaultMaxLength returned nil")
				return
			}

			if *maxLen < tt.expectedRange[0] || *maxLen > tt.expectedRange[1] {
				t.Errorf("Expected max length in range [%d, %d], got %d",
					tt.expectedRange[0], tt.expectedRange[1], *maxLen)
			}

			t.Logf("Mode %s/%s: maxLength=%d", tt.mode, tt.style, *maxLen)
		})
	}
}

// BenchmarkKeywordExtraction benchmarks keyword extraction
func BenchmarkKeywordExtraction(b *testing.B) {
	queries := []string{
		"how to fix bug",
		"I need help with the authentication system",
		"Can you help me analyze the performance issues in our database",
	}

	for _, q := range queries {
		b.Run(q[:minIntTest(20, len(q))], func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = extractKeywords(q)
			}
		})
	}
}

// BenchmarkOutputConstraintsInstructions benchmarks instruction generation
func BenchmarkOutputConstraintsInstructions(b *testing.B) {
	service := NewFocusService(nil)
	maxLen := 4000
	constraints := OutputConstraints{
		MaxLength:       &maxLen,
		Style:           "balanced",
		RequireSources:  true,
		RequireArtifact: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetOutputConstraintsInstructions(constraints)
	}
}

// Helper function
func containsIgnoreCase(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

func minIntTest(a, b int) int {
	if a < b {
		return a
	}
	return b
}
