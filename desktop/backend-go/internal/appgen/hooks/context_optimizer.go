package hooks

import (
	"context"
	"fmt"
	"log/slog"
)

// ContextOptimizerHook implements OSA hierarchical context management
// It monitors token usage and triggers compaction at thresholds
type ContextOptimizerHook struct {
	maxTokens int
	logger    *slog.Logger
}

// NewContextOptimizerHook creates a new context optimizer hook
func NewContextOptimizerHook(maxTokens int, logger *slog.Logger) *ContextOptimizerHook {
	return &ContextOptimizerHook{
		maxTokens: maxTokens,
		logger:    logger,
	}
}

func (h *ContextOptimizerHook) Name() string {
	return "context-optimizer"
}

func (h *ContextOptimizerHook) Execute(ctx context.Context, hookCtx HookContext) error {
	utilization := float64(hookCtx.TokensUsed) / float64(h.maxTokens)

	h.logger.InfoContext(ctx, "context utilization check",
		"tokens_used", hookCtx.TokensUsed,
		"max_tokens", h.maxTokens,
		"utilization", fmt.Sprintf("%.1f%%", utilization*100))

	switch {
	case utilization < 0.50:
		// < 50%: No action needed
		return nil

	case utilization < 0.80:
		// 50-79%: Log warning
		h.logger.WarnContext(ctx, "context approaching threshold",
			"utilization", utilization,
			"tokens", hookCtx.TokensUsed,
			"threshold", "50-80%")
		return nil

	case utilization < 0.90:
		// 80-89%: Recommend summarization (~12% reduction)
		h.logger.WarnContext(ctx, "context threshold exceeded - summarization recommended",
			"utilization", utilization,
			"tokens", hookCtx.TokensUsed,
			"threshold", "80-90%",
			"action", "summarize_tool_calls")
		// In production, this would trigger actual summarization
		return nil

	case utilization < 0.95:
		// 90-94%: Recommend full compaction (~40% reduction)
		h.logger.ErrorContext(ctx, "context critical - compaction required",
			"utilization", utilization,
			"tokens", hookCtx.TokensUsed,
			"threshold", "90-95%",
			"action", "full_compaction")
		// In production, this would trigger compaction
		return nil

	default:
		// ≥95%: Hard stop
		err := fmt.Errorf("context overflow: %d tokens exceeds safe threshold (%d max)",
			hookCtx.TokensUsed, h.maxTokens)
		h.logger.ErrorContext(ctx, "context overflow - halting execution",
			"error", err,
			"tokens", hookCtx.TokensUsed,
			"max_tokens", h.maxTokens)
		return err
	}
}

// CompactionLevel represents the level of context compaction needed
type CompactionLevel int

const (
	CompactionNone      CompactionLevel = 0 // < 50%: No action
	CompactionBreak     CompactionLevel = 1 // 50-79%: Mark breakpoint
	CompactionSummarize CompactionLevel = 2 // 80-89%: Summarize tool calls
	CompactionCompact   CompactionLevel = 3 // 90-94%: Full compaction
	CompactionHardStop  CompactionLevel = 4 // ≥95%: Hard stop
)

// GetCompactionLevel determines the compaction level based on utilization
func GetCompactionLevel(utilization float64) CompactionLevel {
	switch {
	case utilization < 0.50:
		return CompactionNone
	case utilization < 0.80:
		return CompactionBreak
	case utilization < 0.90:
		return CompactionSummarize
	case utilization < 0.95:
		return CompactionCompact
	default:
		return CompactionHardStop
	}
}
