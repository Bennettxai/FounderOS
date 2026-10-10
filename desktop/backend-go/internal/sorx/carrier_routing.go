// Package sorx provides CARRIER AMQP routing for Tier 3-4 skill execution.
// Tier 3 (ReasoningAI) and Tier 4 (GenerativeAI) AI calls are routed through
// the CARRIER bridge to the Elixir SorxMain reasoning engine. When CARRIER is
// unavailable, these calls fall back to the local Groq LLM transparently.
package sorx

import (
	"context"
	"fmt"
	"sync"

	"github.com/rhl/businessos-backend/internal/carrier"
)

// ============================================================================
// Package-level carrier handle
// ============================================================================

// engineCarrier is the package-level CARRIER client injected by the Engine.
// It mirrors the actionHandlers registry pattern used elsewhere in this package.
// Access is protected by engineCarrierMu.
var (
	engineCarrier   *carrier.Client
	engineCarrierMu sync.RWMutex
)

// setEngineCarrier stores the carrier client for use by action handlers.
// Called only by Engine.SetCarrierClient.
func setEngineCarrier(c *carrier.Client) {
	engineCarrierMu.Lock()
	engineCarrier = c
	engineCarrierMu.Unlock()
}

// getEngineCarrier returns the carrier client, or nil if not configured.
func getEngineCarrier() *carrier.Client {
	engineCarrierMu.RLock()
	defer engineCarrierMu.RUnlock()
	return engineCarrier
}

// ============================================================================
// Tier → CARRIER routing helpers
// ============================================================================

// tierToRoutingKey returns the AMQP routing key for the given tier and action.
// Returns an empty string for tiers that do not route through CARRIER.
func tierToRoutingKey(tier SkillTier, action string) string {
	switch tier {
	case TierReasoningAI:
		// Sonnet-class reasoning → Boardroom deliberation subsystem.
		return carrier.RoutingKey(carrier.TopicBoardroom, action)
	case TierGenerativeAI:
		// Opus-class generation → Boardroom deliberation subsystem.
		return carrier.RoutingKey(carrier.TopicBoardroom, action)
	default:
		return ""
	}
}

// tierToPriority maps a SkillTier to a CARRIER message priority.
// Tier 4 (Generative) gets the highest priority because it represents
// novel, user-initiated reasoning tasks that tolerate the least latency.
func tierToPriority(tier SkillTier) int {
	switch tier {
	case TierGenerativeAI:
		return carrier.MaxPriority // 10
	case TierReasoningAI:
		return 7
	default:
		return carrier.DefaultPriority // 5
	}
}

// extractCarrierResult pulls a string result out of a *carrier.Response.
// SorxMain returns results in resp.Result; we coerce to string for LLM-style
// callers that expect a text response. Structured callers can use the raw
// *carrier.Response instead.
func extractCarrierResult(resp *carrier.Response) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("nil carrier response")
	}
	if resp.Error != nil {
		return "", resp.Error
	}
	switch v := resp.Result.(type) {
	case string:
		return v, nil
	case map[string]interface{}:
		// SorxMain may wrap the text in {"output": "..."} or {"result": "..."}.
		for _, key := range []string{"output", "result", "text", "content"} {
			if s, ok := v[key].(string); ok && s != "" {
				return s, nil
			}
		}
		// Fall through to fmt.Sprintf so callers always get a non-empty string.
		return fmt.Sprintf("%v", v), nil
	default:
		if v == nil {
			return "", fmt.Errorf("empty result from SorxMain")
		}
		return fmt.Sprintf("%v", v), nil
	}
}

// ============================================================================
// routeAICall — central dispatch for AI text generation
// ============================================================================

// routeAICall is the single entry point for all AI text generation in SORX.
//
// Routing logic:
//   - Tier 1-2: Local LLM directly (no CARRIER overhead).
//   - Tier 3-4: CARRIER first → SorxMain (Elixir reasoning engine).
//     If CARRIER is unavailable (disabled, disconnected, timeout), falls back
//     to the local Groq LLM. Hard errors from SorxMain are propagated.
//
// Parameters:
//   - tier:          the SkillTier of the executing skill.
//   - carrierAction: the CARRIER action name (e.g. "reason", "generate").
//   - userID:        forwarded in the MessageContext for multi-tenancy.
//   - systemPrompt:  the system prompt for both CARRIER and local fallback.
//   - userPrompt:    the user message for both CARRIER and local fallback.
//   - opts:          optional MessageContext; if provided, opts[0] is merged
//     with userID (userID always wins for the UserID field).
//     Callers that omit opts get a minimal backward-compatible
//     context containing only UserID.
//
// Returns the model's text output or an error.
func routeAICall(
	ctx context.Context,
	tier SkillTier,
	carrierAction string,
	userID string,
	systemPrompt string,
	userPrompt string,
	opts ...carrier.MessageContext,
) (string, error) {
	// Tier 1-2: bypass CARRIER entirely.
	if tier < TierReasoningAI {
		return callGroqLLM(ctx, systemPrompt, userPrompt)
	}

	// Tier 3-4: attempt CARRIER first.
	c := getEngineCarrier()
	if c != nil && c.IsConnected() {
		routingKey := tierToRoutingKey(tier, carrierAction)

		// Build MessageContext: start with caller-supplied context if present,
		// then ensure UserID is always set from the explicit parameter.
		var msgCtx carrier.MessageContext
		if len(opts) > 0 {
			msgCtx = opts[0]
		}
		msgCtx.UserID = userID

		resp, err := c.Send(ctx, carrier.Request{
			Method:     "deliberate",
			RoutingKey: routingKey,
			Priority:   tierToPriority(tier),
			Params: map[string]any{
				"system_prompt": systemPrompt,
				"user_prompt":   userPrompt,
				"tier":          int(tier),
				"model":         TierToModel(tier),
			},
			Context: msgCtx,
		})

		if err == nil {
			return extractCarrierResult(resp)
		}

		// IsFallback covers: disabled, disconnected, timeout.
		// Any other error is a real SorxMain failure — propagate it.
		if !carrier.IsFallback(err) {
			return "", fmt.Errorf("sorxmain error: %w", err)
		}

		// Degraded mode: fall through to local LLM.
		// The caller logs this; we keep routeAICall side-effect-free.
	}

	// Local fallback (CARRIER nil, not connected, or fallback error).
	return callGroqLLM(ctx, systemPrompt, userPrompt)
}
