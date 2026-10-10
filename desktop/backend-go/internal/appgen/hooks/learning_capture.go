package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// LearningCaptureHook implements the OSA learning capture pattern
// It automatically saves successful patterns and error resolutions
type LearningCaptureHook struct {
	storage LearningStorage
	logger  *slog.Logger
}

// NewLearningCaptureHook creates a new learning capture hook
func NewLearningCaptureHook(storage LearningStorage, logger *slog.Logger) *LearningCaptureHook {
	return &LearningCaptureHook{
		storage: storage,
		logger:  logger,
	}
}

func (h *LearningCaptureHook) Name() string {
	return "learning-capture"
}

func (h *LearningCaptureHook) Execute(ctx context.Context, hookCtx HookContext) error {
	duration := hookCtx.EndTime.Sub(hookCtx.StartTime)

	// Convert AgentType to string
	agentTypeStr := fmt.Sprintf("%v", hookCtx.AgentType)

	// Create learning record
	record := LearningRecord{
		Timestamp:  time.Now(),
		AgentType:  agentTypeStr,
		TaskID:     hookCtx.TaskID,
		Success:    hookCtx.Error == nil,
		Duration:   duration,
		TokensUsed: hookCtx.TokensUsed,
	}

	// Classify the interaction
	record.Classification = map[string]string{
		"domain":     classifyDomain(hookCtx.AgentType),
		"complexity": classifyComplexity(duration, hookCtx.TokensUsed),
		"value":      classifyLearningValue(hookCtx),
	}

	// Extract patterns if successful
	if hookCtx.Error == nil && hookCtx.Output != nil {
		record.Patterns = extractPatterns(hookCtx.Output)
	}

	// Save to storage
	if err := h.storage.Save(ctx, record); err != nil {
		h.logger.WarnContext(ctx, "failed to save learning record",
			"error", err,
			"agent_type", record.AgentType)
		return err
	}

	h.logger.InfoContext(ctx, "learning record captured",
		"agent_type", record.AgentType,
		"success", record.Success,
		"duration_ms", record.Duration.Milliseconds())

	return nil
}

// classifyDomain determines the domain from agent type
func classifyDomain(agentType interface{}) string {
	switch agentType {
	case "frontend":
		return "ui"
	case "backend":
		return "api"
	case "database":
		return "data"
	case "test":
		return "quality"
	default:
		return "general"
	}
}

// classifyComplexity determines complexity based on duration and tokens
func classifyComplexity(duration time.Duration, tokens int) string {
	if duration < 10*time.Second && tokens < 1000 {
		return "simple"
	} else if duration < 60*time.Second && tokens < 5000 {
		return "moderate"
	}
	return "complex"
}

// classifyLearningValue determines if the interaction has learning value
func classifyLearningValue(hookCtx HookContext) string {
	// High value: successful complex operations or error resolutions
	if hookCtx.Error == nil && hookCtx.Metadata != nil {
		if complexity, ok := hookCtx.Metadata["complexity"].(string); ok {
			if complexity == "complex" {
				return "high"
			}
		}
	}

	// Medium value: successful moderate operations
	if hookCtx.Error == nil {
		return "medium"
	}

	// Low value: failures (but still captured for error pattern analysis)
	return "low"
}

// extractPatterns extracts code patterns from successful operations
func extractPatterns(output interface{}) []string {
	patterns := []string{}

	// Convert output to JSON for pattern extraction
	if jsonBytes, err := json.Marshal(output); err == nil {
		// Basic pattern: just record that output was generated
		patterns = append(patterns, "code_generated")

		// Could add more sophisticated pattern extraction here
		// For now, keep it simple
		if len(jsonBytes) > 1000 {
			patterns = append(patterns, "large_output")
		}
	}

	return patterns
}

// LearningRecord represents a captured learning interaction
type LearningRecord struct {
	Timestamp      time.Time
	AgentType      string
	TaskID         string
	Success        bool
	Duration       time.Duration
	Patterns       []string
	TokensUsed     int
	Classification map[string]string
}

// LearningStorage defines the interface for persisting learning records
type LearningStorage interface {
	Save(ctx context.Context, record LearningRecord) error
	GetPatterns(ctx context.Context, agentType string, limit int) ([]string, error)
}
