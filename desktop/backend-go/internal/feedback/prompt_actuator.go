package feedback

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// PromptHint is a corrective directive injected into the next LLM prompt.
type PromptHint struct {
	Action    ActuatorAction `json:"action"`
	Metric    string         `json:"metric"`
	Directive string         `json:"directive"`
	ExpiresAt time.Time      `json:"expires_at"`
}

// SignalHintProvider is the read interface consumed by BaseAgent.
// Decouples the agent from the actuator implementation.
type SignalHintProvider interface {
	// ActiveHints returns the current prompt injection text.
	// Returns "" if no active corrections.
	ActiveHints() string
}

// UserScopedHintProvider extends SignalHintProvider with user-scoped hints.
// Implementations that store per-user memory blocks (e.g. SubconsciousHintProvider)
// should implement this interface. BaseAgent checks for it via type assertion.
type UserScopedHintProvider interface {
	SignalHintProvider
	// ActiveHintsForUser returns hints scoped to a specific user.
	ActiveHintsForUser(userID string) string
}

// PromptActuator implements Actuator (from homeostatic_loop.go) by
// translating metric deviations into prompt correction hints that
// the agent reads on each request via the SignalHintProvider interface.
//
// Architecture:
//
//	HomeostaticLoop → Actuator.Act() → PromptActuator stores hint
//	BaseAgent.buildSystemPromptWithThinking() → SignalHintProvider.ActiveHints()
//
// All methods are safe for concurrent use.
type PromptActuator struct {
	mu     sync.RWMutex
	hints  []PromptHint
	ttl    time.Duration
	logger *slog.Logger
}

// NewPromptActuator creates a PromptActuator with the given hint TTL.
// Hints expire after TTL to prevent stale corrections from persisting.
func NewPromptActuator(ttl time.Duration, logger *slog.Logger) *PromptActuator {
	if ttl == 0 {
		ttl = 10 * time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &PromptActuator{
		ttl:    ttl,
		logger: logger.With("component", "prompt_actuator"),
	}
}

// Act implements Actuator. Called by HomeostaticLoop when a metric
// deviates beyond its setpoint tolerance. Translates the deviation
// into a human-readable directive the LLM can understand.
func (pa *PromptActuator) Act(_ context.Context, action ActuatorAction, metricName string, currentValue, targetValue float64) error {
	directive := buildDirective(action, metricName, currentValue, targetValue)
	if directive == "" {
		return nil
	}

	hint := PromptHint{
		Action:    action,
		Metric:    metricName,
		Directive: directive,
		ExpiresAt: time.Now().UTC().Add(pa.ttl),
	}

	pa.mu.Lock()
	defer pa.mu.Unlock()

	// Replace existing hint for the same metric to avoid duplicates.
	for i, h := range pa.hints {
		if h.Metric == metricName {
			pa.hints[i] = hint
			pa.logger.Info("prompt hint updated",
				"action", action, "metric", metricName,
				"current", currentValue, "target", targetValue)
			return nil
		}
	}
	pa.hints = append(pa.hints, hint)
	pa.logger.Info("prompt hint created",
		"action", action, "metric", metricName,
		"current", currentValue, "target", targetValue)
	return nil
}

// ActiveHints returns the current prompt injection text. Returns ""
// if no active (non-expired) hints exist. Called by
// BaseAgent.buildSystemPromptWithThinking() on every request.
func (pa *PromptActuator) ActiveHints() string {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	now := time.Now().UTC()
	var active []string
	for _, h := range pa.hints {
		if now.Before(h.ExpiresAt) {
			active = append(active, "- "+h.Directive)
		}
	}
	if len(active) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"## SIGNAL QUALITY CORRECTIONS (auto-detected)\n\nThe following issues were detected in recent responses. Adjust accordingly:\n\n%s",
		strings.Join(active, "\n"),
	)
}

// Prune removes expired hints. Called periodically by the homeostatic
// loop or a background ticker to prevent unbounded growth.
func (pa *PromptActuator) Prune() int {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	now := time.Now().UTC()
	kept := pa.hints[:0]
	for _, h := range pa.hints {
		if now.Before(h.ExpiresAt) {
			kept = append(kept, h)
		}
	}
	pruned := len(pa.hints) - len(kept)
	pa.hints = kept
	return pruned
}

// HintCount returns the number of active (non-expired) hints.
func (pa *PromptActuator) HintCount() int {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	now := time.Now().UTC()
	count := 0
	for _, h := range pa.hints {
		if now.Before(h.ExpiresAt) {
			count++
		}
	}
	return count
}

func buildDirective(action ActuatorAction, metricName string, current, target float64) string {
	switch action {
	case ActionModeRebalance:
		return fmt.Sprintf(
			"Signal bounce rate is %.0f%% (target: <%.0f%%). Too many responses are misrouted. "+
				"Pay closer attention to user intent before selecting tools or response format.",
			current*100, target*100,
		)
	case ActionContextExpansion:
		return fmt.Sprintf(
			"Re-encoding frequency is %.0f%% (target: <%.0f%%). Users are rephrasing too often. "+
				"Proactively load more context (tree_search, get_project) before responding.",
			current*100, target*100,
		)
	case ActionPromptRefinement:
		return fmt.Sprintf(
			"Genre recognition is %.0f%% (target: >%.0f%%). Response format isn't matching expectations. "+
				"More carefully classify: does the user want DIRECT action, INFORMATION, a COMMITMENT, a DECISION, or EXPRESSION?",
			current*100, target*100,
		)
	case ActionAlert:
		return fmt.Sprintf(
			"Metric %s at %.2f (target: %.2f). Monitor and adjust response quality.",
			metricName, current, target,
		)
	default:
		return ""
	}
}
