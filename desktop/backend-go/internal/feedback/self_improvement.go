package feedback

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ImprovementType classifies the kind of improvement being suggested.
type ImprovementType string

const (
	ImprovementTypePromptRefinement    ImprovementType = "prompt_refinement"
	ImprovementTypeContextExpansion    ImprovementType = "context_expansion"
	ImprovementTypeReasoningDepth      ImprovementType = "reasoning_depth"
	ImprovementTypeToolSelection       ImprovementType = "tool_selection"
	ImprovementTypeResponseFormat      ImprovementType = "response_format"
	ImprovementTypeErrorRecovery       ImprovementType = "error_recovery"
	ImprovementTypeLatencyOptimization ImprovementType = "latency_optimization"
)

// allImprovementTypes is the ordered set used when iterating over Q-values.
var allImprovementTypes = []ImprovementType{
	ImprovementTypePromptRefinement,
	ImprovementTypeContextExpansion,
	ImprovementTypeReasoningDepth,
	ImprovementTypeToolSelection,
	ImprovementTypeResponseFormat,
	ImprovementTypeErrorRecovery,
	ImprovementTypeLatencyOptimization,
}

// RewardWeights controls per-type scaling of the base Q-value reward.
// Loaded hot from Redis; falls back to defaults when Redis is unavailable.
type RewardWeights struct {
	PromptRefinement    float64 `json:"prompt_refinement"`
	ContextExpansion    float64 `json:"context_expansion"`
	ReasoningDepth      float64 `json:"reasoning_depth"`
	ToolSelection       float64 `json:"tool_selection"`
	ResponseFormat      float64 `json:"response_format"`
	ErrorRecovery       float64 `json:"error_recovery"`
	LatencyOptimization float64 `json:"latency_optimization"`
}

// weight returns the weight for a given ImprovementType.
func (w RewardWeights) weight(t ImprovementType) float64 {
	switch t {
	case ImprovementTypePromptRefinement:
		return w.PromptRefinement
	case ImprovementTypeContextExpansion:
		return w.ContextExpansion
	case ImprovementTypeReasoningDepth:
		return w.ReasoningDepth
	case ImprovementTypeToolSelection:
		return w.ToolSelection
	case ImprovementTypeResponseFormat:
		return w.ResponseFormat
	case ImprovementTypeErrorRecovery:
		return w.ErrorRecovery
	case ImprovementTypeLatencyOptimization:
		return w.LatencyOptimization
	default:
		return 1.0
	}
}

// ImprovementSuggestion is a single actionable recommendation produced by the
// SelfImprovementEngine.
type ImprovementSuggestion struct {
	ID          string          `json:"id"`
	Type        ImprovementType `json:"type"`
	Description string          `json:"description"`
	Confidence  float64         `json:"confidence"` // 0-10 scale
	Impact      float64         `json:"impact"`     // 0-1 scale
	AutoApply   bool            `json:"auto_apply"`
	TenantID    string          `json:"tenant_id"`
	CreatedAt   time.Time       `json:"created_at"`
}

// QState is a (context, agentType, improvementType) triple used as the
// Q-table key.
type QState struct {
	Context         string
	AgentType       string
	ImprovementType ImprovementType
}

// SelfImprovementEngine implements Q-learning to select and rank improvement
// suggestions.  Learning rate alpha=0.15, discount factor gamma=0.85.
// A suggestion is flagged for automatic application when confidence >= 9.0
// AND impact >= 0.25.
type SelfImprovementEngine struct {
	mu     sync.RWMutex
	qTable map[string]map[ImprovementType]float64 // stateKey -> improvement -> Q-value
	alpha  float64                                 // learning rate
	gamma  float64                                 // discount factor
	redis  redis.UniversalClient
	logger *slog.Logger
}

// NewSelfImprovementEngine constructs a SelfImprovementEngine.
// redisClient may be nil; in that case reward-weight loads fall back to
// defaults and Q-table updates are in-memory only.
func NewSelfImprovementEngine(redisClient redis.UniversalClient, logger *slog.Logger) *SelfImprovementEngine {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "self_improvement_engine")
	return &SelfImprovementEngine{
		qTable: make(map[string]map[ImprovementType]float64),
		alpha:  0.15,
		gamma:  0.85,
		redis:  redisClient,
		logger: logger,
	}
}

// GenerateSuggestions produces a ranked list of ImprovementSuggestions for
// the given tenant and agent context.  Q-values drive confidence; reward
// weights from Redis scale the raw score into an impact estimate.
func (e *SelfImprovementEngine) GenerateSuggestions(
	ctx context.Context,
	tenantID, agentType, inputContext, feedback string,
) ([]ImprovementSuggestion, error) {
	weights, err := e.LoadRewardWeights(ctx, tenantID)
	if err != nil {
		// Non-fatal: fall back to defaults and log at warn.
		e.logger.WarnContext(ctx, "reward weights load failed, using defaults",
			"tenant_id", tenantID, "error", err)
		weights = defaultRewardWeights()
	}

	// Derive a short context token to reduce Q-table fan-out.
	ctxToken := contextToken(inputContext)

	e.mu.RLock()
	suggestions := make([]ImprovementSuggestion, 0, len(allImprovementTypes))
	for _, impType := range allImprovementTypes {
		state := QState{
			Context:         ctxToken,
			AgentType:       agentType,
			ImprovementType: impType,
		}
		key := stateKey(state)
		qVal := 0.0
		if row, ok := e.qTable[key]; ok {
			qVal = row[impType]
		}

		// Confidence: normalise Q-value to [0, 10].
		// Q-values may start at zero and grow; cap at 10.
		confidence := clamp(qVal*10.0, 0.0, 10.0)

		// Impact: weighted confidence relative to type weight, capped at 1.
		impact := clamp((confidence/10.0)*weights.weight(impType), 0.0, 1.0)

		// Inject a small entropy term so cold-start suggestions are
		// distinguishable from each other rather than all identical.
		if confidence == 0 {
			// #nosec G404 — non-crypto use, entropy for tie-breaking only
			confidence = rand.Float64() * 0.5 // [0, 0.5)
			impact = rand.Float64() * 0.1     // [0, 0.1)
		}

		s := ImprovementSuggestion{
			ID:          uuid.New().String(),
			Type:        impType,
			Description: describeImprovement(impType, agentType, feedback),
			Confidence:  confidence,
			Impact:      impact,
			TenantID:    tenantID,
			CreatedAt:   time.Now().UTC(),
		}
		s.AutoApply = shouldAutoApply(s)
		suggestions = append(suggestions, s)
	}
	e.mu.RUnlock()

	e.logger.InfoContext(ctx, "suggestions generated",
		"tenant_id", tenantID,
		"agent_type", agentType,
		"count", len(suggestions))

	return suggestions, nil
}

// UpdateQValue performs a Bellman-equation update for the given state-action
// pair:
//
//	Q(s,a) ← Q(s,a) + α·(r + γ·maxQ(s,·) - Q(s,a))
func (e *SelfImprovementEngine) UpdateQValue(ctx context.Context, state QState, reward float64) error {
	key := stateKey(state)

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.qTable[key] == nil {
		e.qTable[key] = make(map[ImprovementType]float64)
	}

	current := e.qTable[key][state.ImprovementType]
	maxNext := e.maxQ(key)

	updated := current + e.alpha*(reward+e.gamma*maxNext-current)
	e.qTable[key][state.ImprovementType] = updated

	e.logger.DebugContext(ctx, "Q-value updated",
		"state_key", key,
		"improvement_type", state.ImprovementType,
		"old_value", current,
		"new_value", updated,
		"reward", reward)

	return nil
}

// LoadRewardWeights fetches the reward weights for tenantID from Redis.
// The key is "feedback:rewards:{tenantID}".  Returns defaults when Redis is
// nil or the key does not exist.
func (e *SelfImprovementEngine) LoadRewardWeights(ctx context.Context, tenantID string) (RewardWeights, error) {
	if e.redis == nil {
		return defaultRewardWeights(), nil
	}

	redisKey := fmt.Sprintf("feedback:rewards:%s", tenantID)
	raw, err := e.redis.Get(ctx, redisKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			// Key doesn't exist — return defaults, not an error.
			return defaultRewardWeights(), nil
		}
		return defaultRewardWeights(), fmt.Errorf("load reward weights: %w", err)
	}

	var weights RewardWeights
	if err := json.Unmarshal(raw, &weights); err != nil {
		return defaultRewardWeights(), fmt.Errorf("load reward weights: unmarshal: %w", err)
	}

	e.logger.DebugContext(ctx, "reward weights loaded from Redis",
		"tenant_id", tenantID, "key", redisKey)

	return weights, nil
}

// maxQ returns the maximum Q-value across all improvement types for key.
// Called with e.mu held (at least read-locked).
func (e *SelfImprovementEngine) maxQ(key string) float64 {
	row, ok := e.qTable[key]
	if !ok {
		return 0.0
	}
	max := 0.0
	for _, v := range row {
		if v > max {
			max = v
		}
	}
	return max
}

// defaultRewardWeights returns conservative defaults.  All types are 1.0
// except LatencyOptimization which is 0.8 (deprioritised vs correctness).
func defaultRewardWeights() RewardWeights {
	return RewardWeights{
		PromptRefinement:    1.0,
		ContextExpansion:    1.0,
		ReasoningDepth:      1.0,
		ToolSelection:       1.0,
		ResponseFormat:      1.0,
		ErrorRecovery:       1.0,
		LatencyOptimization: 0.8,
	}
}

// shouldAutoApply returns true when confidence >= 9.0 AND impact >= 0.25.
func shouldAutoApply(s ImprovementSuggestion) bool {
	return s.Confidence >= 9.0 && s.Impact >= 0.25
}

// stateKey produces a stable string key from a QState.
func stateKey(state QState) string {
	return fmt.Sprintf("%s|%s|%s", state.Context, state.AgentType, state.ImprovementType)
}

// contextToken derives a short token from an input context string to bound
// Q-table growth.
func contextToken(inputContext string) string {
	// Use first 64 characters (lowercased, whitespace-trimmed) as the token.
	trimmed := strings.TrimSpace(strings.ToLower(inputContext))
	if len(trimmed) > 64 {
		return trimmed[:64]
	}
	return trimmed
}

// clamp returns v clamped to [min, max].
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// describeImprovement builds a human-readable description for a suggestion.
func describeImprovement(impType ImprovementType, agentType, feedback string) string {
	base := map[ImprovementType]string{
		ImprovementTypePromptRefinement:    "Refine prompt phrasing to reduce ambiguity",
		ImprovementTypeContextExpansion:    "Expand context window to include additional relevant history",
		ImprovementTypeReasoningDepth:      "Increase chain-of-thought reasoning steps",
		ImprovementTypeToolSelection:       "Re-evaluate tool selection order for efficiency",
		ImprovementTypeResponseFormat:      "Adjust response structure for clarity",
		ImprovementTypeErrorRecovery:       "Improve error-handling and retry strategy",
		ImprovementTypeLatencyOptimization: "Reduce processing latency through early exits",
	}
	desc, ok := base[impType]
	if !ok {
		desc = fmt.Sprintf("Apply %s improvement", impType)
	}
	if agentType != "" {
		desc = fmt.Sprintf("[%s] %s", agentType, desc)
	}
	if feedback != "" && len(feedback) < 120 {
		desc = fmt.Sprintf("%s — triggered by: %s", desc, feedback)
	}
	return desc
}
