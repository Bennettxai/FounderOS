package subconscious

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rhl/businessos-backend/internal/feedback"
)

// minInjectionWeight is the minimum block weight for injection.
// Blocks below this weight are silent — the agent operates normally.
const minInjectionWeight = 0.3

// SubconsciousHintProvider wraps the existing PromptActuator and composes
// its output with subconscious memory blocks. Implements feedback.SignalHintProvider.
//
// This is the key integration point: swap PromptActuator → SubconsciousHintProvider
// in main.go. Zero changes to BaseAgent, AgentRegistry, or agent constructors.
type SubconsciousHintProvider struct {
	actuator *feedback.PromptActuator
	store    BlockStore
	logger   *slog.Logger
}

// NewSubconsciousHintProvider creates a new provider that composes
// PromptActuator hints with subconscious memory blocks.
func NewSubconsciousHintProvider(
	actuator *feedback.PromptActuator,
	store BlockStore,
	logger *slog.Logger,
) *SubconsciousHintProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &SubconsciousHintProvider{
		actuator: actuator,
		store:    store,
		logger:   logger.With("component", "subconscious_hint_provider"),
	}
}

// ActiveHints implements feedback.SignalHintProvider.
// Returns composed output: homeostatic corrections + subconscious observations.
func (p *SubconsciousHintProvider) ActiveHints() string {
	var sections []string

	// 1. Homeostatic corrections from PromptActuator (existing behavior)
	if p.actuator != nil {
		if hints := p.actuator.ActiveHints(); hints != "" {
			sections = append(sections, hints)
		}
	}

	// 2. Subconscious memory blocks
	if p.store != nil {
		blockText := p.formatBlocks()
		if blockText != "" {
			sections = append(sections, blockText)
		}
	}

	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n")
}

// ActiveHintsForUser returns hints scoped to a specific user.
// Falls back to ActiveHints() if userID is empty.
func (p *SubconsciousHintProvider) ActiveHintsForUser(userID string) string {
	var sections []string

	// 1. Homeostatic corrections (global, not user-scoped)
	if p.actuator != nil {
		if hints := p.actuator.ActiveHints(); hints != "" {
			sections = append(sections, hints)
		}
	}

	// 2. User-scoped subconscious blocks
	if p.store != nil && userID != "" {
		blockText := p.formatBlocksForUser(userID)
		if blockText != "" {
			sections = append(sections, blockText)
		}
	}

	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n")
}

// formatBlocks formats blocks without user scoping (uses empty context).
// This is the fallback when we don't have user context.
func (p *SubconsciousHintProvider) formatBlocks() string {
	// Without user context, we can't fetch blocks.
	// The user-scoped path (ActiveHintsForUser) should be preferred.
	return ""
}

// formatBlocksForUser fetches and formats blocks for a specific user.
func (p *SubconsciousHintProvider) formatBlocksForUser(userID string) string {
	ctx := context.Background()
	blocks, err := p.store.GetActiveBlocks(ctx, userID)
	if err != nil {
		p.logger.Warn("failed to fetch subconscious blocks",
			"user_id", userID, "error", err)
		return ""
	}

	var lines []string
	for _, b := range blocks {
		if b.Weight < minInjectionWeight {
			continue
		}
		if strings.TrimSpace(b.Content) == "" {
			continue
		}
		header := blockTypeHeader(b.BlockType)
		lines = append(lines, fmt.Sprintf("### %s\n%s", header, b.Content))
	}

	if len(lines) == 0 {
		return ""
	}

	return fmt.Sprintf("## SUBCONSCIOUS OBSERVATIONS (cross-session patterns)\n\n%s",
		strings.Join(lines, "\n\n"))
}

// Actuator returns the wrapped PromptActuator for direct access when needed.
func (p *SubconsciousHintProvider) Actuator() *feedback.PromptActuator {
	return p.actuator
}

func blockTypeHeader(bt BlockType) string {
	headers := map[BlockType]string{
		BlockCoreDirectives:  "Core Directives",
		BlockGuidance:        "Guidance",
		BlockSessionPatterns: "Session Patterns",
		BlockUserPreferences: "User Preferences",
		BlockToolGuidelines:  "Tool Guidelines",
		BlockSelfImprovement: "Self-Improvement",
		BlockProjectContext:  "Project Context",
		BlockPendingItems:    "Pending Items",
	}
	if h, ok := headers[bt]; ok {
		return h
	}
	return string(bt)
}

// Compile-time interface checks.
var _ feedback.SignalHintProvider = (*SubconsciousHintProvider)(nil)
var _ feedback.UserScopedHintProvider = (*SubconsciousHintProvider)(nil)
