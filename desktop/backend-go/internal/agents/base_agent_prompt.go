package agents

import (
	"context"
	"log/slog"

	"github.com/rhl/businessos-backend/internal/feedback"
	"github.com/rhl/businessos-backend/internal/prompts"
	"github.com/rhl/businessos-backend/internal/prompts/core"
	"github.com/rhl/businessos-backend/internal/services"
)

// buildMessages prepares messages for the LLM, including context
func (a *BaseAgent) buildMessages(input AgentInput) []services.ChatMessage {
	messages := make([]services.ChatMessage, 0, len(input.Messages)+1)

	// Capture last user message for personalization
	for i := len(input.Messages) - 1; i >= 0; i-- {
		if input.Messages[i].Role == "user" {
			a.lastUserMessage = input.Messages[i].Content
			break
		}
	}

	// Prepend context as system message if available.
	// Skip if tieredContext is already injected into the system prompt (L4).
	if input.Context != nil && a.tieredContext == nil {
		contextContent := ""
		if a.contextReqs.MaxContextTokens > 0 {
			contextContent = input.Context.FormatForAIWithTokenBudget(a.contextReqs.MaxContextTokens)
		} else {
			contextContent = input.Context.FormatForAI()
		}
		if contextContent != "" {
			contextMsg := services.ChatMessage{
				Role:    "system",
				Content: contextContent,
			}
			messages = append(messages, contextMsg)
		}
	}

	// Add conversation messages
	messages = append(messages, input.Messages...)

	return messages
}

// buildSystemPromptWithThinking returns the system prompt with thinking instructions if enabled
func (a *BaseAgent) buildSystemPromptWithThinking() string {
	slog.Debug("buildSystemPromptWithThinking called", "systemPrompt_len", len(a.systemPrompt))

	// CRITICAL: Start with profile context at the VERY BEGINNING
	// Profile context helps personalize responses based on user's business context
	var result string
	if a.profileContext != "" {
		result = a.profileContext
		slog.Default().Debug("[Agent] ✓ PROFILE CONTEXT placed at START of prompt",
			"chars", len(a.profileContext))
	}

	// Then add role context (for permissions)
	if a.roleContextPrompt != "" {
		if result != "" {
			result += "\n\n"
		}
		result += a.roleContextPrompt
		slog.Default().Debug("[Agent] ✓ ROLE CONTEXT added",
			"chars", len(a.roleContextPrompt))
	}

	// Then add focus mode prompt if set
	if a.focusModePrompt != "" {
		if result != "" {
			result += "\n\n"
		}
		result += a.focusModePrompt
		slog.Debug("applied focus mode prompt", "len", len(a.focusModePrompt))
	}

	// Then add output style prompt if set
	if a.outputStylePrompt != "" {
		if result != "" {
			result += "\n\n"
		}
		result += a.outputStylePrompt
		slog.Debug("applied output style prompt", "len", len(a.outputStylePrompt))
	}

	// Then add workspace memory context if set (Feature: Memory Hierarchy)
	if a.memoryContext != "" {
		if result != "" {
			result += "\n\n"
		}
		result += a.memoryContext
		slog.Default().Debug("[Agent] Applied memory context", "chars", len(a.memoryContext))
	}

	// Then add skills context if set (Agent Skills System)
	if a.skillsPrompt != "" {
		if result != "" {
			result += "\n\n"
		}
		result += a.skillsPrompt
		slog.Default().Debug("[Agent] Applied skills context", "chars", len(a.skillsPrompt))
	}

	// Genre-aware composition context (Signal Theory L3)
	if a.genreContext != "" {
		if result != "" {
			result += "\n\n"
		}
		result += a.genreContext
		slog.Default().Debug("[Agent] Applied genre context", "chars", len(a.genreContext))
	}

	// Now add the base system prompt
	if result != "" {
		result += "\n\n"
	}
	result += a.systemPrompt

	// Authoritative TieredContext injection (Signal Theory L4)
	// When tieredContext is set via SetTieredContext, inject it into the system prompt
	// with an adaptive token budget. This replaces the generic buildMessages() injection.
	if a.tieredContext != nil {
		budget := 4000 // default token budget
		ctxText := a.tieredContext.FormatForAIWithTokenBudget(budget)
		if ctxText != "" {
			result += "\n\n" + ctxText
			slog.Default().Debug("[Agent] Injected L4 TieredContext into system prompt",
				"chars", len(ctxText), "budget", budget)
		}
	}

	// Signal Theory stack: principles → genre enforcement → self-routing capabilities
	// Order matters: principles set the WHY, genre sets the WHAT, routing sets the HOW.
	result += "\n\n" + core.SignalTheoryPrinciples
	result += "\n\n" + core.GenreEnforcementStandards
	result += "\n\n" + core.SelfRoutingCapabilities

	// Inject homeostatic feedback corrections + subconscious observations.
	// Prefer user-scoped hints (includes memory blocks) over global hints.
	if a.signalHints != nil {
		var hints string
		if userScoped, ok := a.signalHints.(feedback.UserScopedHintProvider); ok && a.userID != "" {
			hints = userScoped.ActiveHintsForUser(a.userID)
		} else {
			hints = a.signalHints.ActiveHints()
		}
		if hints != "" {
			result += "\n\n" + hints
			slog.Default().Debug("[Agent] Injected signal hints",
				"chars", len(hints), "user_scoped", a.userID != "")
		}
	}

	// Apply personalization AFTER base prompt but before thinking
	// This allows personalization to enhance but not override role context
	if a.promptPersonalizer != nil && a.userID != "" {
		ctx := context.Background()
		personalizedPrompt, err := a.promptPersonalizer.BuildPersonalizedPrompt(ctx, a.userID, result, a.lastUserMessage)
		if err == nil && personalizedPrompt != "" {
			result = personalizedPrompt
			slog.Default().Debug("[Agent] Applied prompt personalization",
				"total_chars", len(result))
		}
	}

	// Finally, add thinking instructions if enabled
	if a.llmOptions.ThinkingEnabled {
		// Use custom thinking instruction from template if provided, otherwise use default
		thinkingInstruction := prompts.ThinkingInstruction
		if a.llmOptions.ThinkingInstruction != "" {
			thinkingInstruction = a.llmOptions.ThinkingInstruction
		}
		slog.Debug("thinking enabled", "instruction_len", len(thinkingInstruction))
		return result + "\n\n" + thinkingInstruction
	}
	return result
}
