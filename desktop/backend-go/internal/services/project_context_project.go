package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ============================================================================
// Project Context Methods
// ============================================================================

// LoadProjectContext loads all relevant context when a project is selected
func (s *ProjectContextService) LoadProjectContext(ctx context.Context, userID string, projectID uuid.UUID) (*ProjectContext, error) {
	pc := &ProjectContext{
		Memories:              make([]Memory, 0),
		Documents:             make([]Document, 0),
		Artifacts:             make([]Artifact, 0),
		VoiceNotes:            make([]VoiceNote, 0),
		Conversations:         make([]ConversationSummary, 0),
		KnowledgeBaseContexts: make([]Context, 0),
		UserFacts:             make([]UserFact, 0),
	}

	// 1. Get project details
	project, err := s.getProject(ctx, userID, projectID)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	pc.Project = project

	// 2. Get project's context profile (if exists)
	if s.contextService != nil {
		profile, _ := s.contextService.GetContextProfile(ctx, userID, "project", projectID)
		pc.Profile = profile
	}

	// 3. Load memories associated with project
	memories, err := s.getProjectMemories(ctx, userID, projectID, 10)
	if err == nil {
		pc.Memories = memories
	}

	// 4. Load documents linked to project
	documents, err := s.getProjectDocuments(ctx, userID, projectID, 5)
	if err == nil {
		pc.Documents = documents
	}

	// 5. Load artifacts for project
	artifacts, err := s.getProjectArtifacts(ctx, userID, projectID, 5)
	if err == nil {
		pc.Artifacts = artifacts
	}

	// 6. Get recent voice notes for project
	voiceNotes, err := s.getProjectVoiceNotes(ctx, userID, projectID, 3)
	if err == nil {
		pc.VoiceNotes = voiceNotes
	}

	// 7. Get recent conversations in project
	conversations, err := s.getProjectConversations(ctx, userID, projectID, 3)
	if err == nil {
		pc.Conversations = conversations
	}

	// 8. Get KB contexts linked to project
	kbContexts, err := s.getProjectKBContexts(ctx, userID, projectID, 5)
	if err == nil {
		pc.KnowledgeBaseContexts = kbContexts
	}

	// 9. Get user facts
	userFacts, err := s.getUserFacts(ctx, userID)
	if err == nil {
		pc.UserFacts = userFacts
	}

	// Estimate total tokens
	pc.TotalTokenEstimate = s.estimateProjectTokens(pc)

	return pc, nil
}

// getProject retrieves project details
func (s *ProjectContextService) getProject(ctx context.Context, userID string, projectID uuid.UUID) (*Project, error) {
	var p Project
	var clientID *uuid.UUID
	var clientName *string

	err := s.pool.QueryRow(ctx, `
		SELECT p.id, p.user_id, p.name, p.description, p.status, p.priority, p.client_id,
		       c.name as client_name, p.created_at, p.updated_at
		FROM projects p
		LEFT JOIN clients c ON c.id = p.client_id
		WHERE p.id = $1 AND p.user_id = $2
	`, projectID, userID).Scan(
		&p.ID, &p.UserID, &p.Name, &p.Description, &p.Status, &p.Priority, &clientID,
		&clientName, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("project not found")
	}
	if err != nil {
		return nil, err
	}

	p.ClientID = clientID
	if clientName != nil {
		p.ClientName = *clientName
	}

	return &p, nil
}

// getProjectMemories retrieves memories for a project
func (s *ProjectContextService) getProjectMemories(ctx context.Context, userID string, projectID uuid.UUID, limit int) ([]Memory, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, title, summary, content, memory_type, category, source_type,
		       source_id, project_id, node_id, importance_score, access_count, is_pinned,
		       tags, created_at, updated_at
		FROM memories
		WHERE user_id = $1 AND project_id = $2 AND is_active = true
		ORDER BY is_pinned DESC, importance_score DESC, created_at DESC
		LIMIT $3
	`, userID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		err := rows.Scan(
			&m.ID, &m.UserID, &m.Title, &m.Summary, &m.Content, &m.MemoryType, &m.Category,
			&m.SourceType, &m.SourceID, &m.ProjectID, &m.NodeID, &m.ImportanceScore,
			&m.AccessCount, &m.IsPinned, &m.Tags, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			continue
		}
		memories = append(memories, m)
	}

	return memories, nil
}

// getProjectDocuments retrieves documents for a project
func (s *ProjectContextService) getProjectDocuments(ctx context.Context, userID string, projectID uuid.UUID, limit int) ([]Document, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, filename, display_name, description, file_type, document_type,
		       LEFT(extracted_text, 1000), word_count, created_at
		FROM uploaded_documents
		WHERE user_id = $1 AND project_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var documents []Document
	for rows.Next() {
		var d Document
		err := rows.Scan(
			&d.ID, &d.UserID, &d.Filename, &d.DisplayName, &d.Description, &d.FileType,
			&d.DocumentType, &d.ExtractedText, &d.WordCount, &d.CreatedAt,
		)
		if err != nil {
			continue
		}
		documents = append(documents, d)
	}

	return documents, nil
}

// getProjectArtifacts retrieves artifacts for a project
func (s *ProjectContextService) getProjectArtifacts(ctx context.Context, userID string, projectID uuid.UUID, limit int) ([]Artifact, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, title, LEFT(content, 1000), type, project_id, created_at
		FROM artifacts
		WHERE user_id = $1 AND project_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []Artifact
	for rows.Next() {
		var a Artifact
		err := rows.Scan(&a.ID, &a.UserID, &a.Title, &a.Content, &a.ArtifactType, &a.ProjectID, &a.CreatedAt)
		if err != nil {
			continue
		}
		artifacts = append(artifacts, a)
	}

	return artifacts, nil
}

// getProjectVoiceNotes retrieves voice notes for a project
func (s *ProjectContextService) getProjectVoiceNotes(ctx context.Context, userID string, projectID uuid.UUID, limit int) ([]VoiceNote, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, title, transcript, duration, project_id, node_id, key_topics, created_at
		FROM voice_notes
		WHERE user_id = $1 AND project_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var voiceNotes []VoiceNote
	for rows.Next() {
		var v VoiceNote
		err := rows.Scan(&v.ID, &v.UserID, &v.Title, &v.Transcript, &v.Duration, &v.ProjectID, &v.NodeID, &v.KeyTopics, &v.CreatedAt)
		if err != nil {
			continue
		}
		voiceNotes = append(voiceNotes, v)
	}

	return voiceNotes, nil
}

// getProjectConversations retrieves recent conversation summaries for a project
func (s *ProjectContextService) getProjectConversations(ctx context.Context, userID string, projectID uuid.UUID, limit int) ([]ConversationSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT cs.id, cs.conversation_id, cs.summary, cs.key_points, cs.decisions_made,
		       cs.topics, cs.message_count, cs.created_at
		FROM conversation_summaries cs
		JOIN conversations c ON c.id = cs.conversation_id
		WHERE cs.user_id = $1 AND c.project_id = $2
		ORDER BY cs.created_at DESC
		LIMIT $3
	`, userID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ConversationSummary
	for rows.Next() {
		var s ConversationSummary
		var convID uuid.UUID
		err := rows.Scan(&s.ID, &convID, &s.Summary, &s.KeyPoints, &s.DecisionsMade, &s.Topics, &s.MessageCount, &s.CreatedAt)
		if err != nil {
			continue
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}

// getProjectKBContexts retrieves KB contexts linked to a project
func (s *ProjectContextService) getProjectKBContexts(ctx context.Context, userID string, projectID uuid.UUID, limit int) ([]Context, error) {
	// Get contexts that are linked to this project via context_profile_items
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.user_id, c.name, c.type, LEFT(c.content, 500), c.system_prompt, c.created_at
		FROM contexts c
		JOIN context_profile_items cpi ON cpi.item_id = c.id AND cpi.item_type = 'kb_context'
		JOIN context_profiles cp ON cp.id = cpi.context_profile_id
		WHERE c.user_id = $1 AND cp.entity_type = 'project' AND cp.entity_id = $2 AND c.is_archived = false
		ORDER BY cpi.sort_order, c.created_at DESC
		LIMIT $3
	`, userID, projectID, limit)
	if err != nil {
		// Fallback: get contexts directly if profile items don't exist yet
		return s.getDirectProjectContexts(ctx, userID, projectID, limit)
	}
	defer rows.Close()

	var contexts []Context
	for rows.Next() {
		var c Context
		err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Type, &c.Content, &c.SystemPrompt, &c.CreatedAt)
		if err != nil {
			continue
		}
		contexts = append(contexts, c)
	}

	if len(contexts) == 0 {
		return s.getDirectProjectContexts(ctx, userID, projectID, limit)
	}

	return contexts, nil
}

// getDirectProjectContexts retrieves contexts that might be related to the project name
func (s *ProjectContextService) getDirectProjectContexts(ctx context.Context, userID string, projectID uuid.UUID, limit int) ([]Context, error) {
	// Get project name first
	var projectName string
	s.pool.QueryRow(ctx, `SELECT name FROM projects WHERE id = $1`, projectID).Scan(&projectName)

	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, name, type, LEFT(content, 500), system_prompt, created_at
		FROM contexts
		WHERE user_id = $1 AND is_archived = false
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contexts []Context
	for rows.Next() {
		var c Context
		err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.Type, &c.Content, &c.SystemPrompt, &c.CreatedAt)
		if err != nil {
			continue
		}
		contexts = append(contexts, c)
	}

	return contexts, nil
}

// getUserFacts retrieves user facts
func (s *ProjectContextService) getUserFacts(ctx context.Context, userID string) ([]UserFact, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, fact_key, fact_value, fact_type, confidence_score, is_active, created_at
		FROM user_facts
		WHERE user_id = $1 AND is_active = true
		ORDER BY confidence_score DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []UserFact
	for rows.Next() {
		var f UserFact
		err := rows.Scan(&f.ID, &f.UserID, &f.FactKey, &f.FactValue, &f.FactType, &f.ConfidenceScore, &f.IsActive, &f.CreatedAt)
		if err != nil {
			continue
		}
		facts = append(facts, f)
	}

	return facts, nil
}

// estimateProjectTokens estimates total tokens for project context
func (s *ProjectContextService) estimateProjectTokens(pc *ProjectContext) int {
	total := 0

	// Project info (~100 tokens)
	if pc.Project != nil {
		total += 100
	}

	// Memories (average ~200 tokens each)
	total += len(pc.Memories) * 200

	// Documents (average ~250 tokens each for summary)
	total += len(pc.Documents) * 250

	// Artifacts (average ~250 tokens each for summary)
	total += len(pc.Artifacts) * 250

	// Voice notes (average ~150 tokens each)
	total += len(pc.VoiceNotes) * 150

	// Conversations (average ~100 tokens each)
	total += len(pc.Conversations) * 100

	// KB Contexts (average ~150 tokens each)
	total += len(pc.KnowledgeBaseContexts) * 150

	// User facts (average ~20 tokens each)
	total += len(pc.UserFacts) * 20

	return total
}
