package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ContextBuilder builds hierarchical context for AI queries
type ContextBuilder struct {
	pool             *pgxpool.Pool
	embeddingService *EmbeddingService
}

// HierarchicalContext represents the full context for an AI query
type HierarchicalContext struct {
	Query          string           `json:"query"`
	ProfileContext *ProfileContext  `json:"profile_context,omitempty"`
	RelevantBlocks []RelevantBlock  `json:"relevant_blocks"`
	RelatedDocs    []RelatedDoc     `json:"related_docs,omitempty"`
	SiblingDocs    []RelatedDoc     `json:"sibling_docs,omitempty"`
}

// ProfileContext represents a parent profile's context
type ProfileContext struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	Content      string    `json:"content,omitempty"`
}

// RelevantBlock represents a matched content block
type RelevantBlock struct {
	ContextID    uuid.UUID `json:"context_id"`
	DocumentName string    `json:"document_name"`
	BlockContent string    `json:"block_content"`
	BlockType    string    `json:"block_type"`
	Similarity   float64   `json:"similarity"`
}

// RelatedDoc represents a related document
type RelatedDoc struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Type string    `json:"type"`
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(pool *pgxpool.Pool, embeddingService *EmbeddingService) *ContextBuilder {
	return &ContextBuilder{
		pool:             pool,
		embeddingService: embeddingService,
	}
}

// BuildContext creates hierarchical context for an AI query
func (cb *ContextBuilder) BuildContext(ctx context.Context, query string, userID string, limit int) (*HierarchicalContext, error) {
	if limit <= 0 {
		limit = 5
	}

	hc := &HierarchicalContext{
		Query:          query,
		RelevantBlocks: []RelevantBlock{},
		RelatedDocs:    []RelatedDoc{},
		SiblingDocs:    []RelatedDoc{},
	}

	// 1. Semantic search for relevant blocks
	results, err := cb.embeddingService.SimilaritySearch(ctx, query, limit, userID)
	if err != nil {
		return nil, fmt.Errorf("similarity search: %w", err)
	}

	// Track unique parent IDs and context types
	parentIDs := make(map[uuid.UUID]bool)
	contextTypes := make(map[string]bool)
	contextIDs := make(map[uuid.UUID]bool)

	for _, r := range results {
		hc.RelevantBlocks = append(hc.RelevantBlocks, RelevantBlock{
			ContextID:    r.ContextID,
			DocumentName: r.ContextName,
			BlockContent: r.Content,
			BlockType:    r.BlockType,
			Similarity:   r.Similarity,
		})
		contextIDs[r.ContextID] = true
		contextTypes[r.ContextType] = true
		if r.ParentID != nil {
			parentIDs[*r.ParentID] = true
		}
	}

	// 2. Get parent profile context (hierarchical expansion)
	for parentID := range parentIDs {
		profile, err := cb.getProfileContext(ctx, parentID, userID)
		if err == nil && profile != nil {
			hc.ProfileContext = profile
			break // Use first matching parent
		}
	}

	// 3. Get sibling documents (under same parent)
	if hc.ProfileContext != nil {
		siblings, err := cb.getSiblingDocs(ctx, hc.ProfileContext.ID, userID, contextIDs)
		if err == nil {
			hc.SiblingDocs = siblings
		}
	}

	// 4. Get related documents by type
	for ctxType := range contextTypes {
		related, err := cb.getRelatedByType(ctx, ctxType, userID, contextIDs, 3)
		if err == nil {
			hc.RelatedDocs = append(hc.RelatedDocs, related...)
		}
	}

	return hc, nil
}

// getProfileContext gets context for a parent profile
func (cb *ContextBuilder) getProfileContext(ctx context.Context, id uuid.UUID, userID string) (*ProfileContext, error) {
	var profile ProfileContext
	var systemPrompt, content *string
	var ctxType string

	row := cb.pool.QueryRow(ctx, `
		SELECT id, name, type::text, system_prompt_template, content
		FROM contexts
		WHERE id = $1 AND user_id = $2 AND is_archived = false
	`, id, userID)

	err := row.Scan(&profile.ID, &profile.Name, &ctxType, &systemPrompt, &content)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	profile.Type = ctxType
	if systemPrompt != nil {
		profile.SystemPrompt = *systemPrompt
	}
	if content != nil {
		profile.Content = *content
	}

	return &profile, nil
}

// getSiblingDocs gets documents under the same parent
func (cb *ContextBuilder) getSiblingDocs(ctx context.Context, parentID uuid.UUID, userID string, exclude map[uuid.UUID]bool) ([]RelatedDoc, error) {
	rows, err := cb.pool.Query(ctx, `
		SELECT id, name, type::text
		FROM contexts
		WHERE parent_id = $1
		  AND user_id = $2
		  AND is_archived = false
		ORDER BY updated_at DESC
		LIMIT 5
	`, parentID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []RelatedDoc
	for rows.Next() {
		var doc RelatedDoc
		if err := rows.Scan(&doc.ID, &doc.Name, &doc.Type); err != nil {
			continue
		}
		if !exclude[doc.ID] {
			docs = append(docs, doc)
		}
	}

	return docs, rows.Err()
}

// getRelatedByType gets related documents of the same type
func (cb *ContextBuilder) getRelatedByType(ctx context.Context, ctxType string, userID string, exclude map[uuid.UUID]bool, limit int) ([]RelatedDoc, error) {
	rows, err := cb.pool.Query(ctx, `
		SELECT id, name, type::text
		FROM contexts
		WHERE type::text = $1
		  AND user_id = $2
		  AND is_archived = false
		ORDER BY updated_at DESC
		LIMIT $3
	`, ctxType, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []RelatedDoc
	for rows.Next() {
		var doc RelatedDoc
		if err := rows.Scan(&doc.ID, &doc.Name, &doc.Type); err != nil {
			continue
		}
		if !exclude[doc.ID] {
			docs = append(docs, doc)
		}
	}

	return docs, rows.Err()
}

// FormatForAI formats the hierarchical context as a prompt string
func (hc *HierarchicalContext) FormatForAI() string {
	var sb strings.Builder

	sb.WriteString("## Retrieved Knowledge Base Context\n\n")

	// Profile context
	if hc.ProfileContext != nil {
		sb.WriteString(fmt.Sprintf("### Active Profile: %s (%s)\n", hc.ProfileContext.Name, hc.ProfileContext.Type))
		if hc.ProfileContext.SystemPrompt != "" {
			sb.WriteString(fmt.Sprintf("**Context Instructions:** %s\n\n", truncateText(hc.ProfileContext.SystemPrompt, 500)))
		}
		if hc.ProfileContext.Content != "" {
			sb.WriteString(fmt.Sprintf("**Profile Summary:** %s\n\n", truncateText(hc.ProfileContext.Content, 300)))
		}
	}

	// Relevant content blocks
	if len(hc.RelevantBlocks) > 0 {
		sb.WriteString("### Relevant Information\n")
		for i, block := range hc.RelevantBlocks {
			matchPercent := int(block.Similarity * 100)
			sb.WriteString(fmt.Sprintf("%d. **%s** (%d%% match)\n", i+1, block.DocumentName, matchPercent))
			sb.WriteString(fmt.Sprintf("   > %s\n\n", truncateText(block.BlockContent, 300)))
		}
	}

	// Sibling documents
	if len(hc.SiblingDocs) > 0 {
		sb.WriteString("### Related Documents (Same Profile)\n")
		for _, doc := range hc.SiblingDocs {
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", doc.Name, doc.Type))
		}
		sb.WriteString("\n")
	}

	// Related documents by type
	if len(hc.RelatedDocs) > 0 {
		sb.WriteString("### Related Documents (Same Type)\n")
		for _, doc := range hc.RelatedDocs {
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", doc.Name, doc.Type))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// BuildContextForDocument builds context specific to a document
func (cb *ContextBuilder) BuildContextForDocument(ctx context.Context, contextID uuid.UUID, userID string) (*HierarchicalContext, error) {
	hc := &HierarchicalContext{
		RelevantBlocks: []RelevantBlock{},
		RelatedDocs:    []RelatedDoc{},
		SiblingDocs:    []RelatedDoc{},
	}

	// Get the document details
	var parentID pgtype.UUID
	var ctxType, name string

	row := cb.pool.QueryRow(ctx, `
		SELECT name, type::text, parent_id
		FROM contexts
		WHERE id = $1 AND user_id = $2 AND is_archived = false
	`, contextID, userID)

	if err := row.Scan(&name, &ctxType, &parentID); err != nil {
		return nil, err
	}

	hc.Query = fmt.Sprintf("Context for document: %s", name)

	// Get parent profile if exists
	if parentID.Valid {
		parentUUID := uuid.UUID(parentID.Bytes)
		profile, err := cb.getProfileContext(ctx, parentUUID, userID)
		if err == nil && profile != nil {
			hc.ProfileContext = profile
		}

		// Get siblings
		exclude := map[uuid.UUID]bool{contextID: true}
		siblings, err := cb.getSiblingDocs(ctx, parentUUID, userID, exclude)
		if err == nil {
			hc.SiblingDocs = siblings
		}
	}

	// Get related by type
	exclude := map[uuid.UUID]bool{contextID: true}
	related, err := cb.getRelatedByType(ctx, ctxType, userID, exclude, 5)
	if err == nil {
		hc.RelatedDocs = related
	}

	return hc, nil
}

// truncateText truncates text to a maximum length
func truncateText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
