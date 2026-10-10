package subconscious

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/feedback"
	"github.com/rhl/businessos-backend/internal/signal"
)

// BlockAccumulator updates subconscious memory blocks from observed patterns.
// Wires into SelfImprovementEngine for Q-learning updates.
type BlockAccumulator struct {
	store       BlockStore
	selfImprove *feedback.SelfImprovementEngine
	logger      *slog.Logger
}

// NewBlockAccumulator creates a new accumulator.
func NewBlockAccumulator(
	store BlockStore,
	selfImprove *feedback.SelfImprovementEngine,
	logger *slog.Logger,
) *BlockAccumulator {
	if logger == nil {
		logger = slog.Default()
	}
	return &BlockAccumulator{
		store:       store,
		selfImprove: selfImprove,
		logger:      logger.With("component", "block_accumulator"),
	}
}

// Observation is the data produced by the observer pipeline for a single turn.
type Observation struct {
	UserID         string
	UserMessage    string
	Response       string
	Patterns       ExtractedPatterns
	Classification ClassificationResult
	AgentType      string
	ConversationID string
}

// Accumulate updates memory blocks based on the observation.
func (ba *BlockAccumulator) Accumulate(ctx context.Context, obs Observation) {
	// 1. Session patterns: track re-encoding and frustration
	if obs.Patterns.IsReEncoding || obs.Patterns.IsFrustration {
		ba.updateSessionPatterns(ctx, obs)
	}

	// 2. User preferences: persist stated preferences
	if obs.Patterns.HasPreference {
		ba.updateUserPreferences(ctx, obs)
	}

	// 3. Feedback closure: positive closure → Q-learning reward
	if obs.Patterns.IsFeedbackClosed && ba.selfImprove != nil {
		ba.rewardSelfImprovement(ctx, obs)
	}

	// 4. Frustration → negative Q-learning reward
	if obs.Patterns.IsFrustration && ba.selfImprove != nil {
		ba.penalizeSelfImprovement(ctx, obs)
	}

	// 5. Self-improvement: generate suggestions on every turn (cheap)
	if ba.selfImprove != nil {
		ba.generateSelfImprovementBlock(ctx, obs)
	}
}

func (ba *BlockAccumulator) updateSessionPatterns(ctx context.Context, obs Observation) {
	existing, _ := ba.store.GetBlock(ctx, obs.UserID, BlockSessionPatterns)

	var content string
	if existing != nil {
		content = existing.Content
	}

	timestamp := time.Now().UTC().Format("15:04")
	if obs.Patterns.IsReEncoding {
		line := fmt.Sprintf("[%s] Re-encoding detected (sim=%.2f): user rephrased message",
			timestamp, obs.Patterns.ReEncodingSim)
		content = appendLine(content, line)
	}
	if obs.Patterns.IsFrustration {
		preview := obs.UserMessage
		if len(preview) > 100 {
			preview = preview[:100]
		}
		line := fmt.Sprintf("[%s] Frustration signal: %q", timestamp, preview)
		content = appendLine(content, line)
	}

	// Session patterns expire after 4 hours
	expiry := time.Now().UTC().Add(4 * time.Hour)
	if err := ba.store.UpsertBlock(ctx, Block{
		UserID:    obs.UserID,
		BlockType: BlockSessionPatterns,
		Content:   content,
		Weight:    0.6,
		ExpiresAt: &expiry,
	}); err != nil {
		ba.logger.Warn("failed to update session_patterns block", "error", err)
	} else {
		ba.logger.Info("session_patterns block updated",
			"user_id", obs.UserID,
			"re_encoding", obs.Patterns.IsReEncoding,
			"frustration", obs.Patterns.IsFrustration,
			"content_len", len(content))
	}
}

func (ba *BlockAccumulator) updateUserPreferences(ctx context.Context, obs Observation) {
	existing, _ := ba.store.GetBlock(ctx, obs.UserID, BlockUserPreferences)

	var content string
	if existing != nil {
		content = existing.Content
	}

	line := fmt.Sprintf("- %s", obs.Patterns.PreferenceText)
	// Avoid duplicating the same preference
	if !strings.Contains(content, obs.Patterns.PreferenceText) {
		content = appendLine(content, line)
	}

	// Preferences are permanent (no expiry)
	if err := ba.store.UpsertBlock(ctx, Block{
		UserID:    obs.UserID,
		BlockType: BlockUserPreferences,
		Content:   content,
		Weight:    0.7,
	}); err != nil {
		ba.logger.Warn("failed to update user_preferences block", "error", err)
	} else {
		ba.logger.Info("user_preferences block updated",
			"user_id", obs.UserID,
			"preference", obs.Patterns.PreferenceText)
	}
}

func (ba *BlockAccumulator) rewardSelfImprovement(ctx context.Context, obs Observation) {
	// Positive feedback closure = reward the agent's behavior
	state := feedback.QState{
		Context:         truncate(obs.UserMessage, 64),
		AgentType:       obs.AgentType,
		ImprovementType: feedback.ImprovementTypeResponseFormat,
	}
	if err := ba.selfImprove.UpdateQValue(ctx, state, 1.0); err != nil {
		ba.logger.Warn("Q-value reward failed", "error", err)
	}
}

func (ba *BlockAccumulator) penalizeSelfImprovement(ctx context.Context, obs Observation) {
	// Frustration = penalize current behavior
	improvType := feedback.ImprovementTypePromptRefinement
	if obs.Patterns.IsReEncoding {
		improvType = feedback.ImprovementTypeContextExpansion
	}
	state := feedback.QState{
		Context:         truncate(obs.UserMessage, 64),
		AgentType:       obs.AgentType,
		ImprovementType: improvType,
	}
	if err := ba.selfImprove.UpdateQValue(ctx, state, -0.5); err != nil {
		ba.logger.Warn("Q-value penalty failed", "error", err)
	}
}

func (ba *BlockAccumulator) generateSelfImprovementBlock(ctx context.Context, obs Observation) {
	suggestions, err := ba.selfImprove.GenerateSuggestions(
		ctx, obs.UserID, obs.AgentType, obs.UserMessage, "")
	if err != nil {
		ba.logger.Warn("self-improvement suggestions failed", "error", err)
		return
	}

	// Only persist high-confidence suggestions (confidence >= 5.0)
	var lines []string
	for _, s := range suggestions {
		if s.Confidence >= 5.0 {
			lines = append(lines, fmt.Sprintf("- [%.1f] %s", s.Confidence, s.Description))
		}
	}
	if len(lines) == 0 {
		return
	}

	content := strings.Join(lines, "\n")
	if err := ba.store.UpsertBlock(ctx, Block{
		UserID:    obs.UserID,
		BlockType: BlockSelfImprovement,
		Content:   content,
		Weight:    0.4,
	}); err != nil {
		ba.logger.Warn("failed to update self_improvement block", "error", err)
	}
}

// appendLine appends a line to content, respecting MaxBlockContentLength.
func appendLine(content, line string) string {
	if content == "" {
		return line
	}
	combined := content + "\n" + line
	if len(combined) > MaxBlockContentLength {
		// Remove oldest lines to make room
		lines := strings.Split(combined, "\n")
		for len(strings.Join(lines, "\n")) > MaxBlockContentLength && len(lines) > 1 {
			lines = lines[1:]
		}
		return strings.Join(lines, "\n")
	}
	return combined
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// GenreToString converts a signal.Genre to string for display.
func GenreToString(g signal.Genre) string {
	return string(g)
}
