package services

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CommandActionType defines the type of action a command performs
type CommandActionType string

const (
	CommandActionAgent    CommandActionType = "agent"
	CommandActionTemplate CommandActionType = "template"
	CommandActionTool     CommandActionType = "tool"
)

// Command represents a custom /slash command
type Command struct {
	ID               uuid.UUID         `json:"id"`
	UserID           string            `json:"user_id"`
	Trigger          string            `json:"trigger"`           // e.g., "/review"
	DisplayName      string            `json:"display_name"`
	Description      string            `json:"description"`
	ActionType       CommandActionType `json:"action_type"`
	TargetAgentID    *uuid.UUID        `json:"target_agent_id,omitempty"`
	PromptTemplate   string            `json:"prompt_template,omitempty"`
	ToolName         string            `json:"tool_name,omitempty"`
	RequiresInput    bool              `json:"requires_input"`
	InputPlaceholder string            `json:"input_placeholder,omitempty"`
	Parameters       map[string]string `json:"parameters,omitempty"`
	StreamingEnabled bool              `json:"streaming_enabled"`
	ThinkingEnabled  bool              `json:"thinking_enabled"`
	Category         string            `json:"category"`
	IsSystem         bool              `json:"is_system"`
}

// CommandExecutionResult represents the result of executing a command
type CommandExecutionResult struct {
	Success         bool              `json:"success"`
	Command         *Command          `json:"command,omitempty"`
	ProcessedPrompt string            `json:"processed_prompt,omitempty"`
	TargetAgent     *DelegationTarget `json:"target_agent,omitempty"`
	Error           string            `json:"error,omitempty"`
	NeedsInput      bool              `json:"needs_input"`
	InputPrompt     string            `json:"input_prompt,omitempty"`
}

// CommandService handles /slash command resolution and execution
type CommandService struct {
	pool              *pgxpool.Pool
	delegationService *DelegationService
}

// NewCommandService creates a new command service
func NewCommandService(pool *pgxpool.Pool) *CommandService {
	return &CommandService{
		pool:              pool,
		delegationService: NewDelegationService(pool),
	}
}

// ParseCommand extracts a command from the beginning of a message
// Returns the command trigger, any arguments, and the remaining message
func (s *CommandService) ParseCommand(message string) (trigger string, args string, remaining string, isCommand bool) {
	message = strings.TrimSpace(message)

	// Check if message starts with /
	if !strings.HasPrefix(message, "/") {
		return "", "", message, false
	}

	// Extract the command (first word)
	parts := strings.SplitN(message, " ", 2)
	trigger = strings.ToLower(parts[0])

	if len(parts) > 1 {
		remaining = strings.TrimSpace(parts[1])
		args = remaining
	}

	return trigger, args, remaining, true
}

// ResolveCommand finds and returns the command definition for a trigger
func (s *CommandService) ResolveCommand(ctx context.Context, userID string, trigger string) (*Command, error) {
	// Ensure trigger starts with /
	if !strings.HasPrefix(trigger, "/") {
		trigger = "/" + trigger
	}
	trigger = strings.ToLower(trigger)

	// First check user's custom commands
	cmd, err := s.getUserCommand(ctx, userID, trigger)
	if err == nil && cmd != nil {
		return cmd, nil
	}

	// Then check system commands
	cmd, err = s.getSystemCommand(ctx, trigger)
	if err == nil && cmd != nil {
		return cmd, nil
	}

	// Check built-in commands
	cmd = s.getBuiltInCommand(trigger)
	if cmd != nil {
		return cmd, nil
	}

	return nil, fmt.Errorf("command not found: %s", trigger)
}

// ExecuteCommand processes and executes a command
func (s *CommandService) ExecuteCommand(ctx context.Context, userID string, trigger string, args string, contextVars map[string]string) (*CommandExecutionResult, error) {
	cmd, err := s.ResolveCommand(ctx, userID, trigger)
	if err != nil {
		return &CommandExecutionResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Check if command requires input and none was provided
	if cmd.RequiresInput && args == "" {
		return &CommandExecutionResult{
			Success:     true,
			Command:     cmd,
			NeedsInput:  true,
			InputPrompt: cmd.InputPlaceholder,
		}, nil
	}

	// Process based on action type
	switch cmd.ActionType {
	case CommandActionTemplate:
		processed := s.processTemplate(cmd.PromptTemplate, args, contextVars)
		return &CommandExecutionResult{
			Success:         true,
			Command:         cmd,
			ProcessedPrompt: processed,
		}, nil

	case CommandActionAgent:
		if cmd.TargetAgentID == nil {
			return &CommandExecutionResult{
				Success: false,
				Command: cmd,
				Error:   "command has no target agent configured",
			}, nil
		}

		// Resolve the target agent
		agent, err := s.getAgentByID(ctx, *cmd.TargetAgentID)
		if err != nil {
			return &CommandExecutionResult{
				Success: false,
				Command: cmd,
				Error:   fmt.Sprintf("failed to resolve target agent: %v", err),
			}, nil
		}

		// Process template if present
		processed := args
		if cmd.PromptTemplate != "" {
			processed = s.processTemplate(cmd.PromptTemplate, args, contextVars)
		}

		return &CommandExecutionResult{
			Success:         true,
			Command:         cmd,
			ProcessedPrompt: processed,
			TargetAgent:     agent,
		}, nil

	case CommandActionTool:
		return &CommandExecutionResult{
			Success:         true,
			Command:         cmd,
			ProcessedPrompt: args,
		}, nil

	default:
		return &CommandExecutionResult{
			Success: false,
			Command: cmd,
			Error:   fmt.Sprintf("unknown action type: %s", cmd.ActionType),
		}, nil
	}
}

// ListCommands returns all available commands for a user
func (s *CommandService) ListCommands(ctx context.Context, userID string) ([]Command, error) {
	var commands []Command

	// Add built-in commands
	commands = append(commands, s.getBuiltInCommands()...)

	// Add system commands from database
	if s.pool != nil {
		systemCmds, err := s.getSystemCommands(ctx)
		if err == nil {
			commands = append(commands, systemCmds...)
		}

		// Add user's custom commands
		userCmds, err := s.getUserCommands(ctx, userID)
		if err == nil {
			commands = append(commands, userCmds...)
		}
	}

	return commands, nil
}

// IncrementUsage increments the usage counter for a command
func (s *CommandService) IncrementUsage(ctx context.Context, commandID uuid.UUID) {
	if s.pool == nil {
		return
	}

	_, err := s.pool.Exec(ctx, `
		UPDATE custom_commands
		SET times_used = times_used + 1, last_used_at = NOW()
		WHERE id = $1
	`, commandID)

	if err != nil {
		slog.Warn("Failed to increment command usage", "error", err)
	}
}

// processTemplate replaces placeholders in a template
func (s *CommandService) processTemplate(template string, input string, contextVars map[string]string) string {
	result := template

	// Replace {{input}} with the user's input
	result = strings.ReplaceAll(result, "{{input}}", input)
	result = strings.ReplaceAll(result, "{{ input }}", input)

	// Replace context variables
	for key, value := range contextVars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
		placeholder = fmt.Sprintf("{{ %s }}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Clean up any remaining unreplaced placeholders
	re := regexp.MustCompile(`\{\{[^}]+\}\}`)
	result = re.ReplaceAllString(result, "")

	return strings.TrimSpace(result)
}

// getUserCommand gets a user's custom command
func (s *CommandService) getUserCommand(ctx context.Context, userID string, trigger string) (*Command, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	var cmd Command
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, trigger, display_name, description, action_type,
		       target_agent_id, prompt_template, tool_name, requires_input,
		       input_placeholder, parameters, streaming_enabled, thinking_enabled,
		       category, is_system
		FROM custom_commands
		WHERE user_id = $1 AND trigger = $2 AND is_active = TRUE
	`, userID, trigger).Scan(
		&cmd.ID, &cmd.UserID, &cmd.Trigger, &cmd.DisplayName, &cmd.Description,
		&cmd.ActionType, &cmd.TargetAgentID, &cmd.PromptTemplate, &cmd.ToolName,
		&cmd.RequiresInput, &cmd.InputPlaceholder, &cmd.Parameters,
		&cmd.StreamingEnabled, &cmd.ThinkingEnabled, &cmd.Category, &cmd.IsSystem,
	)
	if err != nil {
		return nil, err
	}

	return &cmd, nil
}

// getSystemCommand gets a system command
func (s *CommandService) getSystemCommand(ctx context.Context, trigger string) (*Command, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	var cmd Command
	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, trigger, display_name, description, action_type,
		       target_agent_id, prompt_template, tool_name, requires_input,
		       input_placeholder, parameters, streaming_enabled, thinking_enabled,
		       category, is_system
		FROM custom_commands
		WHERE user_id = 'SYSTEM' AND trigger = $1 AND is_active = TRUE
	`, trigger).Scan(
		&cmd.ID, &cmd.UserID, &cmd.Trigger, &cmd.DisplayName, &cmd.Description,
		&cmd.ActionType, &cmd.TargetAgentID, &cmd.PromptTemplate, &cmd.ToolName,
		&cmd.RequiresInput, &cmd.InputPlaceholder, &cmd.Parameters,
		&cmd.StreamingEnabled, &cmd.ThinkingEnabled, &cmd.Category, &cmd.IsSystem,
	)
	if err != nil {
		return nil, err
	}

	return &cmd, nil
}

// getBuiltInCommand returns a built-in command
func (s *CommandService) getBuiltInCommand(trigger string) *Command {
	builtIn := s.getBuiltInCommands()
	for _, cmd := range builtIn {
		if cmd.Trigger == trigger {
			return &cmd
		}
	}
	return nil
}

// getBuiltInCommands returns hardcoded built-in commands
func (s *CommandService) getBuiltInCommands() []Command {
	return []Command{
		{
			ID:               uuid.MustParse("00000000-0000-0000-0001-000000000001"),
			UserID:           "SYSTEM",
			Trigger:          "/help",
			DisplayName:      "Help",
			Description:      "Show available commands",
			ActionType:       CommandActionTemplate,
			PromptTemplate:   "List all available slash commands and @agents with their descriptions.",
			StreamingEnabled: true,
			Category:         "system",
			IsSystem:         true,
		},
		{
			ID:               uuid.MustParse("00000000-0000-0000-0001-000000000002"),
			UserID:           "SYSTEM",
			Trigger:          "/agents",
			DisplayName:      "List Agents",
			Description:      "Show available agents for @mention",
			ActionType:       CommandActionTemplate,
			PromptTemplate:   "List all available @agents that can be mentioned.",
			StreamingEnabled: true,
			Category:         "system",
			IsSystem:         true,
		},
		{
			ID:               uuid.MustParse("00000000-0000-0000-0001-000000000003"),
			UserID:           "SYSTEM",
			Trigger:          "/clear",
			DisplayName:      "Clear Context",
			Description:      "Clear conversation context",
			ActionType:       CommandActionTool,
			ToolName:         "clear_context",
			StreamingEnabled: false,
			Category:         "system",
			IsSystem:         true,
		},
		{
			ID:               uuid.MustParse("00000000-0000-0000-0001-000000000004"),
			UserID:           "SYSTEM",
			Trigger:          "/summarize",
			DisplayName:      "Summarize",
			Description:      "Summarize the conversation",
			ActionType:       CommandActionTemplate,
			PromptTemplate: `Please provide a concise summary of this conversation, highlighting:
1. Key topics discussed
2. Decisions made
3. Action items identified
4. Open questions remaining`,
			StreamingEnabled: true,
			ThinkingEnabled:  true,
			Category:         "productivity",
			IsSystem:         true,
		},
		{
			ID:               uuid.MustParse("00000000-0000-0000-0001-000000000005"),
			UserID:           "SYSTEM",
			Trigger:          "/search",
			DisplayName:      "Web Search",
			Description:      "Search the web for information",
			ActionType:       CommandActionTool,
			ToolName:         "web_search",
			RequiresInput:    true,
			InputPlaceholder: "Enter your search query...",
			StreamingEnabled: true,
			Category:         "research",
			IsSystem:         true,
		},
		{
			ID:               uuid.MustParse("00000000-0000-0000-0001-000000000006"),
			UserID:           "SYSTEM",
			Trigger:          "/write",
			DisplayName:      "Write Mode",
			Description:      "Switch to document writing focus",
			ActionType:       CommandActionAgent,
			PromptTemplate:   "{{input}}",
			StreamingEnabled: true,
			Category:         "modes",
			IsSystem:         true,
		},
		{
			ID:               uuid.MustParse("00000000-0000-0000-0001-000000000007"),
			UserID:           "SYSTEM",
			Trigger:          "/analyze",
			DisplayName:      "Analyze Mode",
			Description:      "Switch to analysis focus",
			ActionType:       CommandActionAgent,
			PromptTemplate:   "{{input}}",
			StreamingEnabled: true,
			ThinkingEnabled:  true,
			Category:         "modes",
			IsSystem:         true,
		},
		{
			ID:               uuid.MustParse("00000000-0000-0000-0001-000000000008"),
			UserID:           "SYSTEM",
			Trigger:          "/plan",
			DisplayName:      "Plan Mode",
			Description:      "Switch to planning focus",
			ActionType:       CommandActionAgent,
			PromptTemplate:   "{{input}}",
			StreamingEnabled: true,
			ThinkingEnabled:  true,
			Category:         "modes",
			IsSystem:         true,
		},
	}
}

// getSystemCommands gets all system commands from database
func (s *CommandService) getSystemCommands(ctx context.Context) ([]Command, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, trigger, display_name, description, action_type,
		       target_agent_id, prompt_template, tool_name, requires_input,
		       input_placeholder, parameters, streaming_enabled, thinking_enabled,
		       category, is_system
		FROM custom_commands
		WHERE user_id = 'SYSTEM' AND is_active = TRUE
		ORDER BY trigger
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []Command
	for rows.Next() {
		var cmd Command
		err := rows.Scan(
			&cmd.ID, &cmd.UserID, &cmd.Trigger, &cmd.DisplayName, &cmd.Description,
			&cmd.ActionType, &cmd.TargetAgentID, &cmd.PromptTemplate, &cmd.ToolName,
			&cmd.RequiresInput, &cmd.InputPlaceholder, &cmd.Parameters,
			&cmd.StreamingEnabled, &cmd.ThinkingEnabled, &cmd.Category, &cmd.IsSystem,
		)
		if err != nil {
			continue
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

// getUserCommands gets all user's custom commands
func (s *CommandService) getUserCommands(ctx context.Context, userID string) ([]Command, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, trigger, display_name, description, action_type,
		       target_agent_id, prompt_template, tool_name, requires_input,
		       input_placeholder, parameters, streaming_enabled, thinking_enabled,
		       category, is_system
		FROM custom_commands
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY trigger
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []Command
	for rows.Next() {
		var cmd Command
		err := rows.Scan(
			&cmd.ID, &cmd.UserID, &cmd.Trigger, &cmd.DisplayName, &cmd.Description,
			&cmd.ActionType, &cmd.TargetAgentID, &cmd.PromptTemplate, &cmd.ToolName,
			&cmd.RequiresInput, &cmd.InputPlaceholder, &cmd.Parameters,
			&cmd.StreamingEnabled, &cmd.ThinkingEnabled, &cmd.Category, &cmd.IsSystem,
		)
		if err != nil {
			continue
		}
		commands = append(commands, cmd)
	}

	return commands, nil
}

// getAgentByID gets an agent by ID (from custom_agents or presets)
func (s *CommandService) getAgentByID(ctx context.Context, agentID uuid.UUID) (*DelegationTarget, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no database connection")
	}

	var agent DelegationTarget
	var capabilities []string

	// Try custom_agents first
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, display_name, description, capabilities, category, model_preference, system_prompt
		FROM custom_agents
		WHERE id = $1 AND is_active = TRUE
	`, agentID).Scan(
		&agent.ID, &agent.Name, &agent.DisplayName, &agent.Description,
		&capabilities, &agent.Category, &agent.ModelOverride, &agent.SystemPrompt,
	)
	if err == nil {
		agent.Capabilities = capabilities
		agent.IsSystemAgent = false
		return &agent, nil
	}

	// Try agent_presets
	err = s.pool.QueryRow(ctx, `
		SELECT id, name, display_name, description, capabilities, category, model_preference, system_prompt
		FROM agent_presets
		WHERE id = $1 AND is_active = TRUE
	`, agentID).Scan(
		&agent.ID, &agent.Name, &agent.DisplayName, &agent.Description,
		&capabilities, &agent.Category, &agent.ModelOverride, &agent.SystemPrompt,
	)
	if err == nil {
		agent.Capabilities = capabilities
		agent.IsSystemAgent = true
		return &agent, nil
	}

	return nil, fmt.Errorf("agent not found: %s", agentID)
}
