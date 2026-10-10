package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DelegationTarget represents an agent that can receive delegated tasks
type DelegationTarget struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	DisplayName    string    `json:"display_name"`
	Description    string    `json:"description"`
	Capabilities   []string  `json:"capabilities,omitempty"`
	Category       string    `json:"category,omitempty"`
	IsSystemAgent  bool      `json:"is_system_agent"`
	ModelOverride  *string   `json:"model_override,omitempty"`
	SystemPrompt   string    `json:"system_prompt,omitempty"`
}

// DelegationRequest represents a request to delegate to another agent
type DelegationRequest struct {
	FromAgent      string              `json:"from_agent"`
	ToAgent        string              `json:"to_agent"`
	Reason         string              `json:"reason"`
	Context        string              `json:"context"`
	OriginalQuery  string              `json:"original_query"`
	ConversationID uuid.UUID           `json:"conversation_id"`
	UserID         string              `json:"user_id"`
	Metadata       map[string]string   `json:"metadata,omitempty"`
}

// DelegationResult represents the result of a delegation
type DelegationResult struct {
	Success       bool              `json:"success"`
	TargetAgent   *DelegationTarget `json:"target_agent,omitempty"`
	Error         string            `json:"error,omitempty"`
	TraceID       uuid.UUID         `json:"trace_id"`
}

// MentionedAgent represents an @mentioned agent in a message
type MentionedAgent struct {
	Mention    string            `json:"mention"`    // The @mention text (e.g., "@code-reviewer")
	Agent      *DelegationTarget `json:"agent"`      // The resolved agent
	Position   int               `json:"position"`   // Position in the message
	Resolved   bool              `json:"resolved"`   // Whether the agent was found
}

// DelegationService handles agent delegation and @mention resolution
type DelegationService struct {
	pool *pgxpool.Pool
}

// NewDelegationService creates a new delegation service
func NewDelegationService(pool *pgxpool.Pool) *DelegationService {
	return &DelegationService{pool: pool}
}

// ResolveAgentMention resolves an @mention to an agent
func (s *DelegationService) ResolveAgentMention(ctx context.Context, userID string, mention string) (*DelegationTarget, error) {
	// Normalize the mention (remove @ if present, lowercase)
	mention = strings.TrimPrefix(strings.ToLower(mention), "@")

	// First try custom agents
	agent, err := s.getCustomAgentByMention(ctx, userID, mention)
	if err == nil && agent != nil {
		return agent, nil
	}

	// Then try system presets
	agent, err = s.getPresetByMention(ctx, mention)
	if err == nil && agent != nil {
		return agent, nil
	}

	// Check built-in core agents
	agent = s.getCoreAgentByMention(mention)
	if agent != nil {
		return agent, nil
	}

	return nil, fmt.Errorf("agent not found: @%s", mention)
}

// ExtractMentions extracts all @mentions from a message
func (s *DelegationService) ExtractMentions(ctx context.Context, userID string, message string) []MentionedAgent {
	var mentions []MentionedAgent

	// Simple @mention extraction (word starting with @)
	words := strings.Fields(message)
	position := 0

	for _, word := range words {
		if strings.HasPrefix(word, "@") && len(word) > 1 {
			mentionText := strings.TrimSuffix(word, ",")
			mentionText = strings.TrimSuffix(mentionText, ":")
			mentionText = strings.TrimSuffix(mentionText, ".")

			agent, err := s.ResolveAgentMention(ctx, userID, mentionText)
			mention := MentionedAgent{
				Mention:  mentionText,
				Position: strings.Index(message, word),
				Resolved: err == nil && agent != nil,
			}
			if agent != nil {
				mention.Agent = agent
			}
			mentions = append(mentions, mention)
		}
		position += len(word) + 1
	}

	return mentions
}

// RecordMention records an @mention in the database
func (s *DelegationService) RecordMention(ctx context.Context, userID string, conversationID, messageID uuid.UUID, mention MentionedAgent) error {
	if s.pool == nil {
		return nil
	}

	var agentID *uuid.UUID
	if mention.Agent != nil {
		agentID = &mention.Agent.ID
	}

	resolutionNote := ""
	if !mention.Resolved {
		resolutionNote = fmt.Sprintf("Agent not found: %s", mention.Mention)
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO agent_mentions (
			user_id, conversation_id, message_id,
			mentioned_agent_id, mention_text, position_in_message,
			resolved, resolution_note
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, userID, conversationID, messageID, agentID, mention.Mention, mention.Position, mention.Resolved, resolutionNote)

	if err != nil {
		slog.Warn("Failed to record mention", "error", err)
	}
	return err
}

// Delegate initiates a delegation to another agent
func (s *DelegationService) Delegate(ctx context.Context, req DelegationRequest) (*DelegationResult, error) {
	traceID := uuid.New()

	slog.Info("Agent delegation",
		"trace_id", traceID,
		"from", req.FromAgent,
		"to", req.ToAgent,
		"reason", req.Reason,
		"conversation_id", req.ConversationID,
	)

	// Resolve the target agent
	target, err := s.ResolveAgentMention(ctx, req.UserID, req.ToAgent)
	if err != nil {
		return &DelegationResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to resolve agent %s: %v", req.ToAgent, err),
			TraceID: traceID,
		}, nil
	}

	return &DelegationResult{
		Success:     true,
		TargetAgent: target,
		TraceID:     traceID,
	}, nil
}

// ListAvailableAgents returns all agents available for delegation
func (s *DelegationService) ListAvailableAgents(ctx context.Context, userID string) ([]DelegationTarget, error) {
	var agents []DelegationTarget

	// Add core/built-in agents
	agents = append(agents, s.getCoreAgents()...)

	// Add system presets
	presets, err := s.getSystemPresets(ctx)
	if err == nil {
		agents = append(agents, presets...)
	}

	// Add user's custom agents
	if s.pool != nil {
		customAgents, err := s.getUserCustomAgents(ctx, userID)
		if err == nil {
			agents = append(agents, customAgents...)
		}
	}

	return agents, nil
}

// getCustomAgentByMention looks up a custom agent by mention
func (s *DelegationService) getCustomAgentByMention(ctx context.Context, userID string, mention string) (*DelegationTarget, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	var agent DelegationTarget
	var capabilities []string

	err := s.pool.QueryRow(ctx, `
		SELECT id, name, display_name, description, capabilities, category, model_preference, system_prompt
		FROM custom_agents
		WHERE (user_id = $1 OR is_system = TRUE) AND LOWER(name) = $2 AND is_active = TRUE
	`, userID, mention).Scan(
		&agent.ID, &agent.Name, &agent.DisplayName, &agent.Description,
		&capabilities, &agent.Category, &agent.ModelOverride, &agent.SystemPrompt,
	)
	if err != nil {
		return nil, err
	}

	agent.Capabilities = capabilities
	agent.IsSystemAgent = false
	return &agent, nil
}

// getPresetByMention looks up a preset agent by mention
func (s *DelegationService) getPresetByMention(ctx context.Context, mention string) (*DelegationTarget, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	var agent DelegationTarget
	var capabilities []string

	err := s.pool.QueryRow(ctx, `
		SELECT id, name, display_name, description, capabilities, category, model_preference, system_prompt
		FROM agent_presets
		WHERE LOWER(name) = $1 AND is_active = TRUE
	`, mention).Scan(
		&agent.ID, &agent.Name, &agent.DisplayName, &agent.Description,
		&capabilities, &agent.Category, &agent.ModelOverride, &agent.SystemPrompt,
	)
	if err != nil {
		return nil, err
	}

	agent.Capabilities = capabilities
	agent.IsSystemAgent = true
	return &agent, nil
}

// getCoreAgentByMention returns a built-in core agent
func (s *DelegationService) getCoreAgentByMention(mention string) *DelegationTarget {
	coreAgents := s.getCoreAgents()
	for _, agent := range coreAgents {
		if strings.ToLower(agent.Name) == strings.ToLower(mention) {
			return &agent
		}
	}
	return nil
}

// getCoreAgents returns the built-in core agents
func (s *DelegationService) getCoreAgents() []DelegationTarget {
	return []DelegationTarget{
		{
			ID:            uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Name:          "orchestrator",
			DisplayName:   "OSA Orchestrator",
			Description:   "Primary interface that handles general requests and routes to specialists",
			Capabilities:  []string{"routing", "general", "coordination"},
			Category:      "core",
			IsSystemAgent: true,
		},
		{
			ID:            uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Name:          "document",
			DisplayName:   "Document Specialist",
			Description:   "Creates formal business documents: proposals, SOPs, reports, frameworks",
			Capabilities:  []string{"writing", "documents", "proposals", "reports"},
			Category:      "core",
			IsSystemAgent: true,
		},
		{
			ID:            uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Name:          "project",
			DisplayName:   "Project Specialist",
			Description:   "Project management and planning specialist",
			Capabilities:  []string{"projects", "planning", "tasks", "management"},
			Category:      "core",
			IsSystemAgent: true,
		},
		{
			ID:            uuid.MustParse("00000000-0000-0000-0000-000000000004"),
			Name:          "task",
			DisplayName:   "Task Specialist",
			Description:   "Task management, prioritization, and scheduling",
			Capabilities:  []string{"tasks", "scheduling", "prioritization"},
			Category:      "core",
			IsSystemAgent: true,
		},
		{
			ID:            uuid.MustParse("00000000-0000-0000-0000-000000000005"),
			Name:          "client",
			DisplayName:   "Client Specialist",
			Description:   "Client relationship and pipeline specialist",
			Capabilities:  []string{"clients", "crm", "relationships"},
			Category:      "core",
			IsSystemAgent: true,
		},
		{
			ID:            uuid.MustParse("00000000-0000-0000-0000-000000000006"),
			Name:          "analyst",
			DisplayName:   "Analyst Specialist",
			Description:   "Data analysis and insights specialist",
			Capabilities:  []string{"analysis", "data", "insights", "research"},
			Category:      "core",
			IsSystemAgent: true,
		},
	}
}

// getSystemPresets gets agent presets from database
func (s *DelegationService) getSystemPresets(ctx context.Context) ([]DelegationTarget, error) {
	if s.pool == nil {
		return nil, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, name, display_name, description, capabilities, category, model_preference, system_prompt
		FROM agent_presets
		WHERE is_active = TRUE
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []DelegationTarget
	for rows.Next() {
		var agent DelegationTarget
		var capabilities []string

		err := rows.Scan(
			&agent.ID, &agent.Name, &agent.DisplayName, &agent.Description,
			&capabilities, &agent.Category, &agent.ModelOverride, &agent.SystemPrompt,
		)
		if err != nil {
			continue
		}

		agent.Capabilities = capabilities
		agent.IsSystemAgent = true
		agents = append(agents, agent)
	}

	return agents, nil
}

// getUserCustomAgents gets user's custom agents
func (s *DelegationService) getUserCustomAgents(ctx context.Context, userID string) ([]DelegationTarget, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, display_name, description, capabilities, category, model_preference, system_prompt
		FROM custom_agents
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []DelegationTarget
	for rows.Next() {
		var agent DelegationTarget
		var capabilities []string

		err := rows.Scan(
			&agent.ID, &agent.Name, &agent.DisplayName, &agent.Description,
			&capabilities, &agent.Category, &agent.ModelOverride, &agent.SystemPrompt,
		)
		if err != nil {
			continue
		}

		agent.Capabilities = capabilities
		agent.IsSystemAgent = false
		agents = append(agents, agent)
	}

	return agents, nil
}
