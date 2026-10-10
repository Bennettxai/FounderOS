package subconscious

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/feedback"
)

// TestFullPipeline exercises the entire observe → extract → emit → accumulate → inject chain
// using in-memory mocks. No DB, no LLM — pure pipeline validation.
func TestFullPipeline(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	// --- Setup: Build the full pipeline with in-memory components ---

	// Metrics (30-min windows)
	reEncoding := NewInMemoryReEncoding(30 * time.Minute)
	signalBounce := NewInMemorySignalBounce(30 * time.Minute)
	genreRecognition := NewInMemoryGenreRecognition(30 * time.Minute)
	feedbackClosure := NewInMemoryFeedbackClosure(30 * time.Minute)

	// AutopoieticMonitor
	autopoietic, err := feedback.NewAutopoieticMonitor(ctx, feedback.AutopoieticMonitorConfig{
		Logger: logger,
	})
	if err != nil {
		t.Fatalf("failed to create AutopoieticMonitor: %v", err)
	}

	// MetricEmitter
	emitter := NewMetricEmitter(reEncoding, signalBounce, genreRecognition, feedbackClosure, autopoietic, logger)

	// BlockStore (in-memory mock)
	store := NewMockBlockStore()

	// BlockAccumulator (no SelfImprovementEngine — nil is safe)
	accumulator := NewBlockAccumulator(store, nil, logger)

	// PatternExtractor (heuristic, no deps)
	extractor := NewPatternExtractor()

	// Observer (no classifier — nil skips LLM call)
	observer := NewObserver(nil, extractor, emitter, accumulator, logger)

	// SubconsciousHintProvider
	actuator := feedback.NewPromptActuator(10*time.Minute, logger)
	provider := NewSubconsciousHintProvider(actuator, store, logger)

	// --- Scenario 1: Normal turn (no patterns) ---
	t.Run("normal_turn_no_patterns", func(t *testing.T) {
		observer.ObserveSync(ctx, ObserveInput{
			UserID:      "user-1",
			UserMessage: "What's the weather today?",
			AgentType:   "orchestrator",
		})

		// Verify metrics recorded (one normal event)
		reRate, _ := reEncoding.Frequency(ctx, 0)
		if reRate != 0 {
			t.Errorf("expected 0 re-encoding rate, got %.2f", reRate)
		}

		// No blocks should be written (no patterns detected)
		blocks, _ := store.GetActiveBlocks(ctx, "user-1")
		if len(blocks) != 0 {
			t.Errorf("expected 0 blocks, got %d", len(blocks))
		}

		// Hints should be empty
		hints := provider.ActiveHintsForUser("user-1")
		if hints != "" {
			t.Errorf("expected empty hints, got %q", hints)
		}
	})

	// --- Scenario 2: Re-encoding (user rephrases) ---
	t.Run("re_encoding_detected", func(t *testing.T) {
		observer.ObserveSync(ctx, ObserveInput{
			UserID:          "user-1",
			UserMessage:     "can you show me all users in the system please",
			PreviousUserMsg: "list the users for me",
			AssistantResp:   "Here are the users...",
			AgentType:       "orchestrator",
		})

		// Re-encoding metric should now be > 0
		reRate, _ := reEncoding.Frequency(ctx, 0)
		if reRate == 0 {
			t.Error("expected non-zero re-encoding rate after rephrase")
		}

		// Session patterns block should exist
		block, _ := store.GetBlock(ctx, "user-1", BlockSessionPatterns)
		if block == nil {
			t.Fatal("expected session_patterns block")
		}
		if !strings.Contains(block.Content, "Re-encoding detected") {
			t.Errorf("expected re-encoding note in block, got: %s", block.Content)
		}

		// Now check injection
		hints := provider.ActiveHintsForUser("user-1")
		if !strings.Contains(hints, "Session Patterns") {
			t.Errorf("expected session patterns in hints, got: %s", hints)
		}
	})

	// --- Scenario 3: User states a preference ---
	t.Run("preference_detected", func(t *testing.T) {
		observer.ObserveSync(ctx, ObserveInput{
			UserID:      "user-1",
			UserMessage: "I always prefer dark mode and TypeScript",
			AgentType:   "orchestrator",
		})

		block, _ := store.GetBlock(ctx, "user-1", BlockUserPreferences)
		if block == nil {
			t.Fatal("expected user_preferences block")
		}
		if !strings.Contains(block.Content, "dark mode") {
			t.Errorf("expected preference in block, got: %s", block.Content)
		}

		hints := provider.ActiveHintsForUser("user-1")
		if !strings.Contains(hints, "User Preferences") {
			t.Errorf("expected user preferences in hints, got: %s", hints)
		}
	})

	// --- Scenario 4: Frustration signal ---
	t.Run("frustration_detected", func(t *testing.T) {
		observer.ObserveSync(ctx, ObserveInput{
			UserID:      "user-1",
			UserMessage: "No, I said I wanted the DELETE endpoint, not the GET endpoint",
			AgentType:   "orchestrator",
		})

		block, _ := store.GetBlock(ctx, "user-1", BlockSessionPatterns)
		if block == nil {
			t.Fatal("expected session_patterns block")
		}
		if !strings.Contains(block.Content, "Frustration signal") {
			t.Errorf("expected frustration in block, got: %s", block.Content)
		}
	})

	// --- Scenario 5: Feedback closure (positive) ---
	t.Run("feedback_closure_detected", func(t *testing.T) {
		observer.ObserveSync(ctx, ObserveInput{
			UserID:      "user-1",
			UserMessage: "Perfect, that's exactly what I needed, thanks!",
			AgentType:   "orchestrator",
		})

		closureRate, _ := feedbackClosure.ClosureRate(ctx, 0)
		if closureRate == 0 {
			t.Error("expected non-zero feedback closure rate")
		}
	})

	// --- Scenario 6: Bounce detection ---
	t.Run("bounce_detected", func(t *testing.T) {
		observer.ObserveSync(ctx, ObserveInput{
			UserID:        "user-1",
			UserMessage:   "Can you handle that?",
			AssistantResp: "I'm sorry, that's outside my expertise. Switching to the document agent...",
			AgentType:     "orchestrator",
		})

		bounceRate, _ := signalBounce.Rate(ctx, 0)
		if bounceRate == 0 {
			t.Error("expected non-zero bounce rate")
		}
	})

	// --- Scenario 7: Verify hints compose actuator + blocks ---
	t.Run("hints_compose_actuator_and_blocks", func(t *testing.T) {
		// Trigger a hint in the actuator
		_ = actuator.Act(ctx, feedback.ActionContextExpansion, "re_encoding_frequency", 0.25, 0.15)

		hints := provider.ActiveHintsForUser("user-1")
		if !strings.Contains(hints, "SIGNAL QUALITY CORRECTIONS") {
			t.Error("expected actuator corrections in hints")
		}
		if !strings.Contains(hints, "SUBCONSCIOUS OBSERVATIONS") {
			t.Error("expected subconscious observations in hints")
		}
	})
}

// TestPipelineConcurrency verifies the pipeline is safe under concurrent observation.
func TestPipelineConcurrency(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	reEncoding := NewInMemoryReEncoding(30 * time.Minute)
	signalBounce := NewInMemorySignalBounce(30 * time.Minute)
	genreRecognition := NewInMemoryGenreRecognition(30 * time.Minute)
	feedbackClosure := NewInMemoryFeedbackClosure(30 * time.Minute)

	emitter := NewMetricEmitter(reEncoding, signalBounce, genreRecognition, feedbackClosure, nil, logger)
	store := NewMockBlockStore()
	accumulator := NewBlockAccumulator(store, nil, logger)
	extractor := NewPatternExtractor()
	observer := NewObserver(nil, extractor, emitter, accumulator, logger)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			observer.ObserveSync(ctx, ObserveInput{
				UserID:      "user-concurrent",
				UserMessage: "I prefer dark mode",
				AgentType:   "orchestrator",
			})
		}(i)
	}
	wg.Wait()

	// Should not panic, and preferences block should exist
	block, _ := store.GetBlock(ctx, "user-concurrent", BlockUserPreferences)
	if block == nil {
		t.Fatal("expected user_preferences block after concurrent writes")
	}
}

// TestHintProviderUserScoping verifies user isolation in hint injection.
func TestHintProviderUserScoping(t *testing.T) {
	ctx := context.Background()
	store := NewMockBlockStore()
	provider := NewSubconsciousHintProvider(nil, store, nil)

	// User A has a preference
	_ = store.UpsertBlock(ctx, Block{
		UserID:    "user-a",
		BlockType: BlockUserPreferences,
		Content:   "- Always use Go",
		Weight:    0.7,
	})

	// User B has a different preference
	_ = store.UpsertBlock(ctx, Block{
		UserID:    "user-b",
		BlockType: BlockUserPreferences,
		Content:   "- Always use Python",
		Weight:    0.7,
	})

	hintsA := provider.ActiveHintsForUser("user-a")
	hintsB := provider.ActiveHintsForUser("user-b")

	if !strings.Contains(hintsA, "Go") {
		t.Error("user-a should see Go preference")
	}
	if strings.Contains(hintsA, "Python") {
		t.Error("user-a should NOT see Python preference")
	}
	if !strings.Contains(hintsB, "Python") {
		t.Error("user-b should see Python preference")
	}
	if strings.Contains(hintsB, "Go") {
		t.Error("user-b should NOT see Go preference")
	}
}

// TestMetricRatesAfterMultipleTurns verifies metric sliding windows accumulate correctly.
func TestMetricRatesAfterMultipleTurns(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	reEncoding := NewInMemoryReEncoding(30 * time.Minute)
	signalBounce := NewInMemorySignalBounce(30 * time.Minute)
	genreRecognition := NewInMemoryGenreRecognition(30 * time.Minute)
	feedbackClosure := NewInMemoryFeedbackClosure(30 * time.Minute)

	emitter := NewMetricEmitter(reEncoding, signalBounce, genreRecognition, feedbackClosure, nil, logger)
	store := NewMockBlockStore()
	accumulator := NewBlockAccumulator(store, nil, logger)
	extractor := NewPatternExtractor()
	observer := NewObserver(nil, extractor, emitter, accumulator, logger)

	// 10 normal turns
	for i := 0; i < 10; i++ {
		observer.ObserveSync(ctx, ObserveInput{
			UserID:      "user-rates",
			UserMessage: "normal message",
			AgentType:   "orchestrator",
		})
	}

	// Verify all metrics at 0% (no re-encoding, no bouncing, 100% genre match, no closure)
	reRate, _ := reEncoding.Frequency(ctx, 0)
	if reRate != 0 {
		t.Errorf("expected 0%% re-encoding after normal turns, got %.2f", reRate)
	}
	genreRate, _ := genreRecognition.Rate(ctx, 0)
	if genreRate != 1.0 {
		t.Errorf("expected 100%% genre recognition after normal turns, got %.2f", genreRate)
	}

	// Now inject 2 re-encoding turns
	observer.ObserveSync(ctx, ObserveInput{
		UserID:          "user-rates",
		UserMessage:     "can you display all the users please",
		PreviousUserMsg: "show me the list of users",
		AgentType:       "orchestrator",
	})
	observer.ObserveSync(ctx, ObserveInput{
		UserID:          "user-rates",
		UserMessage:     "please list out every user account",
		PreviousUserMsg: "can you display all the users please",
		AgentType:       "orchestrator",
	})

	// Rate should be at least 1/12 ≈ 0.083 (at least 1 re-encoding detected)
	reRate, _ = reEncoding.Frequency(ctx, 0)
	if reRate == 0 {
		t.Errorf("expected non-zero re-encoding rate after rephrase turns, got 0")
	}
	// Rate should be at most 2/12 ≈ 0.167 (at most 2 re-encodings)
	if reRate > 0.2 {
		t.Errorf("expected re-encoding rate <= 0.20, got %.4f", reRate)
	}
}
