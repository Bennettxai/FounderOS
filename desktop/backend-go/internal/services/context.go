package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContextService handles context profiles, loading rules, and tree operations
type ContextService struct {
	pool             *pgxpool.Pool
	embeddingService *EmbeddingService
	logger           *slog.Logger
}

// NewContextService creates a new context service
func NewContextService(pool *pgxpool.Pool, embeddingService *EmbeddingService) *ContextService {
	return &ContextService{
		pool:             pool,
		embeddingService: embeddingService,
		logger:           slog.Default().With("service", "context"),
	}
}

// ============================================================================
// Context Profile Methods
// ============================================================================

// CreateContextProfile creates a new context profile
func (s *ContextService) CreateContextProfile(ctx context.Context, userID, entityType string, entityID uuid.UUID, name, description string) (*ContextProfile, error) {
	profile := &ContextProfile{
		ID:          uuid.New(),
		UserID:      userID,
		EntityType:  entityType,
		EntityID:    entityID,
		Name:        name,
		Description: description,
		ContextTree: make(map[string]any),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	contextTree, _ := json.Marshal(profile.ContextTree)

	_, err := s.pool.Exec(ctx, `
		INSERT INTO context_profiles (id, user_id, entity_type, entity_id, name, description, context_tree, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, profile.ID, userID, entityType, entityID, name, description, contextTree, profile.CreatedAt, profile.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("create context profile: %w", err)
	}

	return profile, nil
}

// GetContextProfile retrieves a context profile by entity
func (s *ContextService) GetContextProfile(ctx context.Context, userID, entityType string, entityID uuid.UUID) (*ContextProfile, error) {
	var profile ContextProfile
	var contextTreeJSON []byte
	var keyFacts, documentTypes []string

	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, entity_type, entity_id, name, description, context_tree, summary,
		       key_facts, document_types, total_documents, total_file_size_bytes,
		       total_contexts, total_memories, total_artifacts, total_tasks, created_at, updated_at
		FROM context_profiles
		WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, entityType, entityID).Scan(
		&profile.ID, &profile.UserID, &profile.EntityType, &profile.EntityID,
		&profile.Name, &profile.Description, &contextTreeJSON, &profile.Summary,
		&keyFacts, &documentTypes, &profile.TotalDocuments, &profile.TotalFileSizeBytes,
		&profile.TotalContexts, &profile.TotalMemories, &profile.TotalArtifacts, &profile.TotalTasks,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get context profile: %w", err)
	}

	if contextTreeJSON != nil {
		json.Unmarshal(contextTreeJSON, &profile.ContextTree)
	}
	profile.KeyFacts = keyFacts
	profile.DocumentTypes = documentTypes

	return &profile, nil
}

// UpdateContextProfile updates a context profile
func (s *ContextService) UpdateContextProfile(ctx context.Context, profile *ContextProfile) error {
	contextTree, _ := json.Marshal(profile.ContextTree)

	_, err := s.pool.Exec(ctx, `
		UPDATE context_profiles SET
			name = $2, description = $3, context_tree = $4, summary = $5,
			key_facts = $6, document_types = $7, total_documents = $8, total_file_size_bytes = $9,
			total_contexts = $10, total_memories = $11, total_artifacts = $12, total_tasks = $13,
			updated_at = NOW()
		WHERE id = $1
	`, profile.ID, profile.Name, profile.Description, contextTree, profile.Summary,
		profile.KeyFacts, profile.DocumentTypes, profile.TotalDocuments, profile.TotalFileSizeBytes,
		profile.TotalContexts, profile.TotalMemories, profile.TotalArtifacts, profile.TotalTasks)

	if err != nil {
		return fmt.Errorf("update context profile: %w", err)
	}

	return nil
}

// ============================================================================
// Context Loading Rules Methods
// ============================================================================

// GetLoadingRules retrieves context loading rules for a user
func (s *ContextService) GetLoadingRules(ctx context.Context, userID, triggerType, triggerValue string) ([]ContextLoadingRule, error) {
	query := `
		SELECT id, user_id, name, description, trigger_type, trigger_value,
		       load_memories, memory_types, memory_limit,
		       load_contexts, context_categories, context_limit,
		       load_artifacts, artifact_types, artifact_limit,
		       load_recent_conversations, conversation_limit,
		       priority, is_active, created_at, updated_at
		FROM context_loading_rules
		WHERE user_id = $1 AND is_active = true
	`
	args := []any{userID}

	if triggerType != "" {
		query += " AND (trigger_type = $2 OR trigger_type = 'always')"
		args = append(args, triggerType)
	}

	query += " ORDER BY priority DESC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query loading rules: %w", err)
	}
	defer rows.Close()

	var rules []ContextLoadingRule
	for rows.Next() {
		var r ContextLoadingRule
		err := rows.Scan(
			&r.ID, &r.UserID, &r.Name, &r.Description, &r.TriggerType, &r.TriggerValue,
			&r.LoadMemories, &r.MemoryTypes, &r.MemoryLimit,
			&r.LoadContexts, &r.ContextCategories, &r.ContextLimit,
			&r.LoadArtifacts, &r.ArtifactTypes, &r.ArtifactLimit,
			&r.LoadRecentConversations, &r.ConversationLimit,
			&r.Priority, &r.IsActive, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			continue
		}
		rules = append(rules, r)
	}

	return rules, nil
}

// CreateLoadingRule creates a new context loading rule
func (s *ContextService) CreateLoadingRule(ctx context.Context, rule *ContextLoadingRule) error {
	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	_, err := s.pool.Exec(ctx, `
		INSERT INTO context_loading_rules (
			id, user_id, name, description, trigger_type, trigger_value,
			load_memories, memory_types, memory_limit,
			load_contexts, context_categories, context_limit,
			load_artifacts, artifact_types, artifact_limit,
			load_recent_conversations, conversation_limit,
			priority, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
	`, rule.ID, rule.UserID, rule.Name, rule.Description, rule.TriggerType, rule.TriggerValue,
		rule.LoadMemories, rule.MemoryTypes, rule.MemoryLimit,
		rule.LoadContexts, rule.ContextCategories, rule.ContextLimit,
		rule.LoadArtifacts, rule.ArtifactTypes, rule.ArtifactLimit,
		rule.LoadRecentConversations, rule.ConversationLimit,
		rule.Priority, rule.IsActive, rule.CreatedAt, rule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create loading rule: %w", err)
	}

	return nil
}

// ============================================================================
// Context Item Loading
// ============================================================================

// LoadContextItem loads a specific context item by ID and type
func (s *ContextService) LoadContextItem(ctx context.Context, userID string, itemID uuid.UUID, itemType string) (*ContextItem, error) {
	item := &ContextItem{
		ID:   itemID,
		Type: itemType,
	}

	switch itemType {
	case "memory":
		err := s.pool.QueryRow(ctx, `
			SELECT title, content FROM memories WHERE id = $1 AND user_id = $2
		`, itemID, userID).Scan(&item.Title, &item.Content)
		if err != nil {
			return nil, fmt.Errorf("load memory: %w", err)
		}

	case "document":
		err := s.pool.QueryRow(ctx, `
			SELECT COALESCE(display_name, filename), extracted_text FROM uploaded_documents WHERE id = $1 AND user_id = $2
		`, itemID, userID).Scan(&item.Title, &item.Content)
		if err != nil {
			return nil, fmt.Errorf("load document: %w", err)
		}

	case "artifact":
		err := s.pool.QueryRow(ctx, `
			SELECT title, content FROM artifacts WHERE id = $1 AND user_id = $2
		`, itemID, userID).Scan(&item.Title, &item.Content)
		if err != nil {
			return nil, fmt.Errorf("load artifact: %w", err)
		}

	default:
		return nil, fmt.Errorf("unknown item type: %s", itemType)
	}

	// Estimate tokens (rough estimate: 4 chars per token)
	item.TokenCount = len(item.Content) / 4

	return item, nil
}

// ============================================================================
// Agent Context Session Methods
// ============================================================================

// CreateContextSession creates a new agent context session
func (s *ContextService) CreateContextSession(ctx context.Context, userID string, conversationID uuid.UUID, agentType string, maxTokens int) (*AgentContextSession, error) {
	session := &AgentContextSession{
		ID:               uuid.New(),
		UserID:           userID,
		ConversationID:   conversationID,
		AgentType:        agentType,
		MaxContextTokens: maxTokens,
		AvailableTokens:  maxTokens,
		LoadedMemories:   []uuid.UUID{},
		LoadedContexts:   []uuid.UUID{},
		LoadedArtifacts:  []uuid.UUID{},
		LoadedDocuments:  []uuid.UUID{},
		StartedAt:        time.Now(),
		LastActivityAt:   time.Now(),
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO agent_context_sessions (
			id, user_id, conversation_id, agent_type, max_context_tokens,
			used_context_tokens, available_tokens, loaded_memories, loaded_contexts,
			loaded_artifacts, loaded_documents, started_at, last_activity_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, session.ID, userID, conversationID, agentType, maxTokens,
		0, maxTokens, session.LoadedMemories, session.LoadedContexts,
		session.LoadedArtifacts, session.LoadedDocuments, session.StartedAt, session.LastActivityAt)

	if err != nil {
		return nil, fmt.Errorf("create context session: %w", err)
	}

	return session, nil
}

// GetContextSession retrieves an agent context session
func (s *ContextService) GetContextSession(ctx context.Context, sessionID uuid.UUID) (*AgentContextSession, error) {
	var session AgentContextSession

	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, conversation_id, agent_type, agent_id, max_context_tokens,
		       used_context_tokens, available_tokens, loaded_memories, loaded_contexts,
		       loaded_artifacts, loaded_documents, base_system_prompt, injected_context,
		       total_system_prompt_tokens, project_id, node_id, focus_mode,
		       started_at, last_activity_at, ended_at
		FROM agent_context_sessions WHERE id = $1
	`, sessionID).Scan(
		&session.ID, &session.UserID, &session.ConversationID, &session.AgentType,
		&session.AgentID, &session.MaxContextTokens, &session.UsedContextTokens,
		&session.AvailableTokens, &session.LoadedMemories, &session.LoadedContexts,
		&session.LoadedArtifacts, &session.LoadedDocuments, &session.BaseSystemPrompt,
		&session.InjectedContext, &session.TotalSystemPromptTokens, &session.ProjectID,
		&session.NodeID, &session.FocusMode, &session.StartedAt, &session.LastActivityAt,
		&session.EndedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get context session: %w", err)
	}

	return &session, nil
}

// UpdateSessionTokenUsage updates the token usage for a session
func (s *ContextService) UpdateSessionTokenUsage(ctx context.Context, sessionID uuid.UUID, usedTokens int) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE agent_context_sessions
		SET used_context_tokens = $2, available_tokens = max_context_tokens - $2, last_activity_at = NOW()
		WHERE id = $1
	`, sessionID, usedTokens)

	if err != nil {
		return fmt.Errorf("update session token usage: %w", err)
	}

	return nil
}
