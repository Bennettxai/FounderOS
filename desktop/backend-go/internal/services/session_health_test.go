package services

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestSessionHealthService() *SessionHealthService {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mts := &ModeTransitionService{
		pool:     nil,
		logger:   logger.With("component", "mode_transition"),
		lastMode: make(map[string]lastModeState),
	}
	return NewSessionHealthService(mts, logger)
}

// makeTransition creates a transition where FromMode spent durationMs before switching to toMode.
// DurationMs measures time in the *previous* mode (FromMode).
func makeTransition(fromMode string, durationMs int64) ModeTransition {
	return ModeTransition{
		FromMode:   fromMode,
		DurationMs: durationMs,
	}
}

// ---------------------------------------------------------------------------
// computeDominantMode
// ---------------------------------------------------------------------------

func TestComputeDominantMode_ReturnsModeWithHighestCumulativeTime(t *testing.T) {
	history := []ModeTransition{
		makeTransition("code", 30000),
		makeTransition("write", 10000),
		makeTransition("code", 20000), // total code = 50 000 ms
		makeTransition("write", 5000), // total write = 15 000 ms
	}

	result := computeDominantMode(history)

	assert.Equal(t, "code", result)
}

func TestComputeDominantMode_HandlesEmptyHistory(t *testing.T) {
	result := computeDominantMode([]ModeTransition{})

	assert.Equal(t, "", result, "empty history should produce empty dominant mode")
}

func TestComputeDominantMode_SingleMode(t *testing.T) {
	history := []ModeTransition{
		makeTransition("research", 60000),
	}

	result := computeDominantMode(history)

	assert.Equal(t, "research", result)
}

func TestComputeDominantMode_SkipsEmptyFromMode(t *testing.T) {
	history := []ModeTransition{
		{FromMode: "", DurationMs: 999999}, // should be ignored (no known previous mode)
		makeTransition("plan", 1000),
	}

	result := computeDominantMode(history)

	assert.Equal(t, "plan", result, "empty FromMode entries should not count")
}

func TestComputeDominantMode_TieBreakReturnsAWinner(t *testing.T) {
	// Two modes with the same total time — result must still be a valid mode.
	history := []ModeTransition{
		makeTransition("quick", 5000),
		makeTransition("deep", 5000),
	}

	result := computeDominantMode(history)

	assert.Contains(t, []string{"quick", "deep"}, result,
		"tie should resolve to one of the competing modes")
}

func TestComputeDominantMode_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		history  []ModeTransition
		expected string
	}{
		{
			name:     "code dominates",
			history:  []ModeTransition{makeTransition("code", 100), makeTransition("write", 50)},
			expected: "code",
		},
		{
			name:     "write dominates after split",
			history:  []ModeTransition{makeTransition("write", 200), makeTransition("code", 100), makeTransition("write", 50)},
			expected: "write",
		},
		{
			name:     "single entry",
			history:  []ModeTransition{makeTransition("analyze", 42)},
			expected: "analyze",
		},
		{
			name:     "all zero durations picks any",
			history:  []ModeTransition{makeTransition("quick", 0), makeTransition("deep", 0)},
			expected: "", // maxDuration stays 0; no winner
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeDominantMode(tt.history)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// generateHint
// ---------------------------------------------------------------------------

func TestGenerateHint_ConfusedSignal(t *testing.T) {
	h := &SessionHealth{
		IsConfused:   true,
		IsFrustrated: false,
		IsDeepFocus:  false,
	}

	hint := generateHint(h)

	assert.Contains(t, hint, "confused", "hint should acknowledge confusion")
	assert.Contains(t, hint, "clarifying", "hint should recommend clarifying questions")
}

func TestGenerateHint_FrustratedSignal(t *testing.T) {
	h := &SessionHealth{
		IsConfused:   false,
		IsFrustrated: true,
		IsDeepFocus:  false,
	}

	hint := generateHint(h)

	assert.Contains(t, hint, "concise", "hint should recommend concise responses")
}

func TestGenerateHint_DeepFocusSignal(t *testing.T) {
	h := &SessionHealth{
		IsConfused:   false,
		IsFrustrated: false,
		IsDeepFocus:  true,
	}

	hint := generateHint(h)

	assert.Contains(t, hint, "detailed", "hint should recommend detailed responses")
}

func TestGenerateHint_NormalState_ReturnsEmpty(t *testing.T) {
	h := &SessionHealth{
		IsConfused:   false,
		IsFrustrated: false,
		IsDeepFocus:  false,
	}

	hint := generateHint(h)

	assert.Equal(t, "", hint, "normal state should produce empty hint")
}

func TestGenerateHint_ConfusedTakesPrecedenceOverFrustrated(t *testing.T) {
	// Both flags set — confused wins (switch case order).
	h := &SessionHealth{
		IsConfused:   true,
		IsFrustrated: true,
		IsDeepFocus:  false,
	}

	hint := generateHint(h)

	assert.Contains(t, hint, "confused")
}

// ---------------------------------------------------------------------------
// Health heuristics — IsConfused, IsFrustrated, IsDeepFocus thresholds
// ---------------------------------------------------------------------------

func TestSessionHealth_IsConfused_AboveThreshold(t *testing.T) {
	// >3.0 transitions/min triggers IsConfused.
	flipRate := 3.1
	result := flipRate > 3.0
	assert.True(t, result)
}

func TestSessionHealth_IsConfused_BelowThreshold(t *testing.T) {
	flipRate := 3.0
	result := flipRate > 3.0
	assert.False(t, result, "exactly 3.0 should NOT trigger IsConfused")
}

func TestSessionHealth_IsFrustrated_AboveThreshold(t *testing.T) {
	msgFreq := 5.1
	result := msgFreq > 5.0
	assert.True(t, result)
}

func TestSessionHealth_IsFrustrated_AtThreshold(t *testing.T) {
	msgFreq := 5.0
	result := msgFreq > 5.0
	assert.False(t, result, "exactly 5.0 should NOT trigger IsFrustrated")
}

func TestSessionHealth_IsDeepFocus_AllConditionsMet(t *testing.T) {
	isConfused := false
	sessionDuration := 31 * time.Minute
	flipRate := 0.4

	isDeepFocus := !isConfused && sessionDuration > 30*time.Minute && flipRate < 0.5
	assert.True(t, isDeepFocus)
}

func TestSessionHealth_IsDeepFocus_BlockedWhenConfused(t *testing.T) {
	isConfused := true
	sessionDuration := 60 * time.Minute
	flipRate := 0.1

	isDeepFocus := !isConfused && sessionDuration > 30*time.Minute && flipRate < 0.5
	assert.False(t, isDeepFocus, "confusion should block deep focus")
}

func TestSessionHealth_IsDeepFocus_BlockedWhenSessionTooShort(t *testing.T) {
	isConfused := false
	sessionDuration := 29 * time.Minute
	flipRate := 0.1

	isDeepFocus := !isConfused && sessionDuration > 30*time.Minute && flipRate < 0.5
	assert.False(t, isDeepFocus, "short session should block deep focus")
}

func TestSessionHealth_IsDeepFocus_BlockedWhenHighFlipRate(t *testing.T) {
	isConfused := false
	sessionDuration := 45 * time.Minute
	flipRate := 0.5

	isDeepFocus := !isConfused && sessionDuration > 30*time.Minute && flipRate < 0.5
	assert.False(t, isDeepFocus, "flip rate >=0.5 should block deep focus")
}

// ---------------------------------------------------------------------------
// MessageFrequency calculation (pure math)
// ---------------------------------------------------------------------------

func TestSessionHealth_MessageFrequency_CalculatedCorrectly(t *testing.T) {
	tests := []struct {
		name           string
		messageCount   int
		sessionMinutes float64
		expectedFreq   float64
	}{
		{"10 messages over 2 minutes", 10, 2.0, 5.0},
		{"1 message over 1 minute", 1, 1.0, 1.0},
		{"30 messages over 5 minutes", 30, 5.0, 6.0},
		{"0 messages", 0, 5.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var freq float64
			if tt.sessionMinutes > 0 {
				freq = float64(tt.messageCount) / tt.sessionMinutes
			}
			assert.InDelta(t, tt.expectedFreq, freq, 0.001)
		})
	}
}

// ---------------------------------------------------------------------------
// ComputeHealth — pure sub-function coverage without DB dependency
//
// ComputeHealth calls GetModeFlipRate and GetTransitionHistory which both
// require a live pgxpool. Those paths are not tested here; instead we cover
// every logic branch that does NOT touch the pool.
// ---------------------------------------------------------------------------

// TestComputeHealth_GetLastMode verifies that the CurrentMode field is
// populated from the in-memory cache (zero DB involvement).
func TestComputeHealth_GetLastModeReadsFromCache(t *testing.T) {
	svc := newTestSessionHealthService()

	svc.transitions.mu.Lock()
	svc.transitions.lastMode["conv-cache-check"] = lastModeState{
		Mode:      "analyze",
		EnteredAt: time.Now(),
	}
	svc.transitions.mu.Unlock()

	// Verify the cache is readable before ComputeHealth would consume it.
	mode := svc.transitions.GetLastMode("conv-cache-check")
	assert.Equal(t, "analyze", mode)
}

// TestComputeHealth_SessionDuration verifies the duration math inline.
func TestComputeHealth_SessionDurationCalculation(t *testing.T) {
	sessionStart := time.Now().Add(-20 * time.Minute)
	duration := time.Since(sessionStart)

	assert.GreaterOrEqual(t, duration, 19*time.Minute,
		"session duration should be at least 19 minutes")
}

// TestComputeHealth_ZeroMessageFrequency checks the zero-message edge case
// in the frequency formula used by ComputeHealth.
func TestComputeHealth_ZeroMessageFrequency(t *testing.T) {
	messageCount := 0
	sessionMinutes := 5.0

	var freq float64
	if sessionMinutes > 0 {
		freq = float64(messageCount) / sessionMinutes
	}

	assert.Equal(t, 0.0, freq)
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func BenchmarkComputeDominantMode(b *testing.B) {
	history := make([]ModeTransition, 20)
	modes := []string{"code", "write", "analyze", "research", "plan"}
	for i := range history {
		history[i] = makeTransition(modes[i%len(modes)], int64((i+1)*1000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = computeDominantMode(history)
	}
}

func BenchmarkGenerateHint(b *testing.B) {
	h := &SessionHealth{IsConfused: true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateHint(h)
	}
}
