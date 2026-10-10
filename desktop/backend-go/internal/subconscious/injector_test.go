package subconscious

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/feedback"
)

func TestSubconsciousHintProviderEmpty(t *testing.T) {
	store := NewMockBlockStore()
	actuator := feedback.NewPromptActuator(10*time.Minute, slog.Default())
	provider := NewSubconsciousHintProvider(actuator, store, nil)

	hints := provider.ActiveHints()
	if hints != "" {
		t.Errorf("expected empty hints, got %q", hints)
	}

	hints = provider.ActiveHintsForUser("user1")
	if hints != "" {
		t.Errorf("expected empty hints for user, got %q", hints)
	}
}

func TestSubconsciousHintProviderWithBlocks(t *testing.T) {
	store := NewMockBlockStore()
	actuator := feedback.NewPromptActuator(10*time.Minute, slog.Default())
	provider := NewSubconsciousHintProvider(actuator, store, nil)

	ctx := context.Background()

	// Add a high-weight block
	_ = store.UpsertBlock(ctx, Block{
		UserID:    "user1",
		BlockType: BlockUserPreferences,
		Content:   "- I prefer dark mode\n- Always use TypeScript",
		Weight:    0.7,
	})

	hints := provider.ActiveHintsForUser("user1")
	if !strings.Contains(hints, "SUBCONSCIOUS OBSERVATIONS") {
		t.Error("expected subconscious header in hints")
	}
	if !strings.Contains(hints, "User Preferences") {
		t.Error("expected User Preferences section")
	}
	if !strings.Contains(hints, "dark mode") {
		t.Error("expected preference content")
	}
}

func TestSubconsciousHintProviderSilencePolicy(t *testing.T) {
	store := NewMockBlockStore()
	actuator := feedback.NewPromptActuator(10*time.Minute, slog.Default())
	provider := NewSubconsciousHintProvider(actuator, store, nil)

	ctx := context.Background()

	// Add a low-weight block (below minInjectionWeight = 0.3)
	_ = store.UpsertBlock(ctx, Block{
		UserID:    "user1",
		BlockType: BlockPendingItems,
		Content:   "low priority item",
		Weight:    0.1,
	})

	hints := provider.ActiveHintsForUser("user1")
	if hints != "" {
		t.Errorf("expected empty hints for low-weight block, got %q", hints)
	}
}

func TestSubconsciousHintProviderComposesActuator(t *testing.T) {
	store := NewMockBlockStore()
	actuator := feedback.NewPromptActuator(10*time.Minute, slog.Default())
	provider := NewSubconsciousHintProvider(actuator, store, nil)

	// Trigger a hint in the actuator
	ctx := context.Background()
	_ = actuator.Act(ctx, feedback.ActionContextExpansion, "re_encoding_frequency", 0.25, 0.15)

	// Add a block
	_ = store.UpsertBlock(ctx, Block{
		UserID:    "user1",
		BlockType: BlockGuidance,
		Content:   "Focus on code quality",
		Weight:    0.5,
	})

	hints := provider.ActiveHintsForUser("user1")

	// Should contain both actuator hints AND subconscious blocks
	if !strings.Contains(hints, "SIGNAL QUALITY CORRECTIONS") {
		t.Error("expected actuator corrections in hints")
	}
	if !strings.Contains(hints, "SUBCONSCIOUS OBSERVATIONS") {
		t.Error("expected subconscious observations in hints")
	}
}

func TestSubconsciousHintProviderNilComponents(t *testing.T) {
	// Should not panic with nil actuator and store
	provider := NewSubconsciousHintProvider(nil, nil, nil)
	hints := provider.ActiveHints()
	if hints != "" {
		t.Errorf("expected empty hints, got %q", hints)
	}
}

func TestBlockTypeHeaders(t *testing.T) {
	for _, bt := range AllBlockTypes {
		header := blockTypeHeader(bt)
		if header == "" {
			t.Errorf("missing header for block type %s", bt)
		}
	}
}
