package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/agents"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/services"
	bossignal "github.com/rhl/businessos-backend/internal/signal"
	"github.com/rhl/businessos-backend/internal/tools"
)

// agentContextResult holds values produced by injectAgentContext that SendMessage needs.
type agentContextResult struct {
	roleContextStr   string
	memoryContextStr string
	model            string
	llmOptions       services.LLMOptions
	cotOrchestrator  *agents.OrchestratorCOT
}

// injectAgentContext configures agent with all runtime context layers (role, memory, profile,
// skills, focus, custom agent, COT, output style, signal annotation, tiered context)
// and returns values that SendMessage uses downstream.
func (h *ChatHandler) injectAgentContext(
	ctx context.Context,
	req SendMessageRequest,
	userID string,
	userName string,
	workspaceIDStr string,
	agent agents.Agent,
	registry *agents.AgentRegistry,
	tieredCtx *services.TieredContext,
	signalEnvelope bossignal.SignalEnvelope,
	agentType agents.AgentType,
	focusModeStr string,
	customAgent *sqlc.CustomAgent,
	customAgentSystemPrompt string,
	focusSystemPrompt string,
	searchContextText string,
	useCOT bool,
	model string,
	llmOptions services.LLMOptions,
) agentContextResult {
	// Register OSA escalation tool — the Orchestrator decides when to use it.
	// If OSA client is nil, the tool gracefully explains OSA is unavailable.
	if baseAgent, ok := agent.(*agents.BaseAgent); ok {
		var wsID *uuid.UUID
		if workspaceIDStr != "" {
			if parsed, err := uuid.Parse(workspaceIDStr); err == nil {
				wsID = &parsed
			}
		}
		baseAgent.RegisterExternalTool(tools.NewEscalateToOSATool(h.osaClient, userID, wsID))
	}

	var roleContextStr string
	var memoryContextStr string

	// Inject role context if workspace_id is provided (Feature 1: Role-based permissions)
	if workspaceIDStr != "" && h.roleContextService != nil {
		workspaceID, err := uuid.Parse(workspaceIDStr)
		if err == nil {
			roleCtx, err := h.roleContextService.GetUserRoleContext(ctx, userID, workspaceID)
			if err == nil {
				roleContextStr = roleCtx.GetRoleContextPrompt()
				agent.SetRoleContextPrompt(roleContextStr)
				slog.Info("[ChatV2] Injected role context",
					"role", roleCtx.RoleName, "level", roleCtx.HierarchyLevel, "permissions", len(roleCtx.Permissions))
			} else {
				slog.Info("[ChatV2] Failed to get role context", "error", err)
			}
		}
	}

	// Inject user onboarding profile context for personalization
	slog.Info("[ChatV2] Profile injection check", "user_id", userID)
	onboardingService := services.NewOnboardingService(h.pool, nil, nil, nil)
	userProfile, err := onboardingService.GetUserProfile(ctx, userID)
	if err == nil && userProfile != nil {
		profileContextStr := services.BuildProfilePrefix(userProfile)
		agent.SetProfileContext(profileContextStr)
		slog.Info("[ChatV2] Injected user profile context",
			"chars", len(profileContextStr),
			"business", userProfile.BusinessType, "team", userProfile.TeamSize,
			"role", userProfile.OwnerRole, "challenge", userProfile.MainChallenge)
	} else {
		if err != nil {
			slog.Info("[ChatV2] No profile found for user (expected if onboarding not completed)", "error", err)
		}
	}

	// Inject workspace memories if workspace_id is provided (Feature: Memory Hierarchy)
	slog.Info("[ChatV2] Memory injection check:,", "workspace_id", workspaceIDStr != "", "workspace_id", h.memoryHierarchyService != nil)

	if workspaceIDStr != "" && h.memoryHierarchyService != nil {
		workspaceID, err := uuid.Parse(workspaceIDStr)
		if err == nil {
			slog.Info("[ChatV2] Attempting to get accessible memories for workspace", "value", workspaceID)
			memories, err := h.memoryHierarchyService.GetAccessibleMemories(ctx, workspaceID, userID, nil, 20)
			slog.Info("[ChatV2] GetAccessibleMemories returned", "count", len(memories), "error", err)
			if err == nil && len(memories) > 0 {
				var memoryContext strings.Builder
				memoryContext.WriteString("\n## 🧠 WORKSPACE MEMORY BANK\n\n")
				memoryContext.WriteString("**CRITICAL INSTRUCTION**: The following memories contain factual information about this workspace. When answering questions, you MUST prioritize and use information from these memories. These are authoritative sources of truth for workspace-specific knowledge.\n\n")

				for _, mem := range memories {
					memoryContext.WriteString(fmt.Sprintf("### 📌 %s\n", mem.Title))
					if mem.Content != "" {
						memoryContext.WriteString(fmt.Sprintf("%s\n\n", mem.Content))
					}
				}

				memoryContext.WriteString("\n**REMINDER**: Always check these workspace memories first before providing general knowledge. If a question relates to information in these memories, use that information directly in your response.\n")

				memoryContextStr = memoryContext.String()
				agent.SetMemoryContext(memoryContextStr)
				slog.Info("[ChatV2] Injected workspace memories",
					"count", len(memories), "chars", len(memoryContextStr))
			} else if err != nil {
				slog.Info("[ChatV2] Failed to get workspace memories", "error", err)
			}
		}
	}

	// Inject skills context if skills loader is available
	if h.skillsLoader != nil && h.skillsLoader.IsLoaded() {
		skillsPrompt := h.skillsLoader.GetSkillsPromptXML()
		if skillsPrompt != "" {
			agent.SetSkillsPrompt(skillsPrompt)
			slog.Info("[ChatV2] Injected skills context", "chars", len(skillsPrompt))
		}
	}

	// Apply focus mode system prompt prefix if set
	if focusSystemPrompt != "" {
		fullFocusPrompt := focusSystemPrompt
		if searchContextText != "" {
			fullFocusPrompt = focusSystemPrompt + "\n\n" + searchContextText
			slog.Info("[ChatV2] Injected search context into focus prompt", "chars", len(searchContextText))
		}
		agent.SetFocusModePrompt(fullFocusPrompt)
		slog.Info("[ChatV2] Applied focus mode prompt prefix", "chars", len(fullFocusPrompt))
	} else if searchContextText != "" {
		agent.SetFocusModePrompt(searchContextText)
		slog.Info("[ChatV2] Injected search context only", "chars", len(searchContextText))
	}

	// If custom agent found, override the system prompt
	slog.Info("[ChatV2] Custom agent check:,", "customAgent", customAgent != nil, "promptLen", len(customAgentSystemPrompt))
	if customAgent != nil && customAgentSystemPrompt != "" {
		slog.Info("[ChatV2] APPLYING custom prompt", "value", customAgentSystemPrompt[:min(100, len(customAgentSystemPrompt))])
		agent.SetCustomSystemPrompt(customAgentSystemPrompt)
		if customAgent.ModelPreference != nil && *customAgent.ModelPreference != "" {
			agent.SetModel(*customAgent.ModelPreference)
			model = *customAgent.ModelPreference
		}
		if customAgent.ThinkingEnabled != nil && *customAgent.ThinkingEnabled {
			llmOptions.ThinkingEnabled = true
			agent.SetOptions(llmOptions)
		}
		slog.Info("[ChatV2] Using custom agent", "agent", customAgent.DisplayName, "model", model, "prompt_chars", len(customAgentSystemPrompt))
	} else {
		slog.Info("[ChatV2] NOT using custom agent -,", "customAgent", customAgent != nil, "promptLen", len(customAgentSystemPrompt))
	}

	// Create COT orchestrator if enabled (but NOT when using custom agents)
	var cotOrchestrator *agents.OrchestratorCOT
	if useCOT && customAgent == nil {
		cotOrchestrator = agents.NewOrchestratorCOT(h.pool, h.cfg, registry)
	} else if customAgent != nil && useCOT {
		slog.Info("[ChatV2] COT mode disabled for custom agent", "value", customAgent.DisplayName)
	}

	// Determine output style
	styleName := ""
	if req.OutputStyle != nil && *req.OutputStyle != "" {
		styleName = *req.OutputStyle
	} else {
		styleName = h.getUserStylePreference(ctx, userID, focusModeStr, string(agentType))
		if styleName == "" {
			styleName = detectStyleFromContext(focusModeStr, string(agentType))
		}
	}

	if styleName != "" {
		stylePrompt := h.applyOutputStyle(ctx, styleName, "")
		if stylePrompt != "" {
			agent.SetOutputStylePrompt(stylePrompt)
		}
	}

	// L3: Genre-aware composition — signal annotation + structure hints.
	genreCtx := agents.BuildSignalAnnotation(&signalEnvelope, tieredCtx)
	if genreCtx != "" {
		agent.SetGenreContext(genreCtx)
		slog.Debug("ChatV2: Injected signal annotation",
			"genre", signalEnvelope.Genre, "doctype", signalEnvelope.DocType, "len", len(genreCtx))
	}

	// L4: Inject TieredContext into authoritative system prompt (adaptive token budget)
	if tieredCtx != nil {
		agent.SetTieredContext(tieredCtx)
	}

	return agentContextResult{
		roleContextStr:   roleContextStr,
		memoryContextStr: memoryContextStr,
		model:            model,
		llmOptions:       llmOptions,
		cotOrchestrator:  cotOrchestrator,
	}
}
