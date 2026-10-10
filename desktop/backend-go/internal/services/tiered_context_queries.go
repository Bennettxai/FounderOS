package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// getUserFacts loads active user facts ordered by confidence score.
func (s *TieredContextService) getUserFacts(ctx context.Context, userID string, limit int) ([]UserFact, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, fact_key, fact_value, fact_type, confidence_score, is_active, created_at
		FROM user_facts
		WHERE user_id = $1 AND is_active = true
		ORDER BY confidence_score DESC, created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []UserFact
	for rows.Next() {
		var f UserFact
		if err := rows.Scan(&f.ID, &f.UserID, &f.FactKey, &f.FactValue, &f.FactType, &f.ConfidenceScore, &f.IsActive, &f.CreatedAt); err != nil {
			continue
		}
		facts = append(facts, f)
	}
	return facts, nil
}

// getNodeAncestry returns the node and its ancestors (root -> ... -> selected) up to maxDepth.
// It is resilient: returns an empty slice if the node is missing or archived.
func (s *TieredContextService) getNodeAncestry(ctx context.Context, nodeID uuid.UUID, userID string, maxDepth int) ([]nodeAncestryRow, error) {
	if maxDepth <= 0 {
		maxDepth = 8
	}

	query := `
		WITH RECURSIVE chain AS (
			SELECT n.id, n.parent_id, n.context_id, n.name,
			       n.type::text, COALESCE(n.health::text, ''), COALESCE(n.purpose, ''),
			       0 AS depth
			FROM nodes n
			WHERE n.id = $1 AND n.user_id = $2 AND n.is_archived = false
			UNION ALL
			SELECT p.id, p.parent_id, p.context_id, p.name,
			       p.type::text, COALESCE(p.health::text, ''), COALESCE(p.purpose, ''),
			       c.depth + 1 AS depth
			FROM nodes p
			JOIN chain c ON p.id = c.parent_id
			WHERE p.user_id = $2 AND p.is_archived = false AND c.depth < $3
		)
		SELECT id, parent_id, context_id, name, type, health, purpose, depth
		FROM chain
		ORDER BY depth DESC;
	`

	rows, err := s.pool.Query(ctx, query, nodeID, userID, maxDepth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chain []nodeAncestryRow
	for rows.Next() {
		var r nodeAncestryRow
		if err := rows.Scan(&r.ID, &r.ParentID, &r.ContextID, &r.Name, &r.Type, &r.Health, &r.Purpose, &r.Depth); err != nil {
			continue
		}
		chain = append(chain, r)
	}
	return chain, nil
}

func (s *TieredContextService) getProjectFull(ctx context.Context, projectID uuid.UUID, userID string) (*ProjectFullContext, error) {
	query := `
		SELECT p.id, p.name, COALESCE(p.description, ''), p.status, p.priority,
			   COALESCE(p.client_name, ''), COALESCE(p.project_type, '')
		FROM projects p
		WHERE p.id = $1 AND p.user_id = $2`

	var project ProjectFullContext
	err := s.pool.QueryRow(ctx, query, projectID, userID).Scan(
		&project.ID, &project.Name, &project.Description, &project.Status,
		&project.Priority, &project.ClientName, &project.ProjectType,
	)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *TieredContextService) getProjectTasks(ctx context.Context, projectID uuid.UUID, userID string) ([]TaskFullContext, error) {
	query := `
		SELECT t.id, t.title, COALESCE(t.description, ''), t.status, t.priority,
			   COALESCE(to_char(t.due_date, 'YYYY-MM-DD'), ''),
			   COALESCE(tm.name, '')
		FROM tasks t
		LEFT JOIN team_members tm ON tm.id = t.assignee_id
		WHERE t.project_id = $1 AND t.user_id = $2
		ORDER BY
			CASE t.priority WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
			t.due_date ASC NULLS LAST
		LIMIT 20`

	rows, err := s.pool.Query(ctx, query, projectID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []TaskFullContext
	for rows.Next() {
		var task TaskFullContext
		if err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status,
			&task.Priority, &task.DueDate, &task.AssigneeName); err != nil {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *TieredContextService) getProjectClient(ctx context.Context, projectID uuid.UUID, userID string) (*ClientFullContext, error) {
	query := `
		SELECT c.id, c.name, COALESCE(c.email, ''), COALESCE(c.industry, ''), c.status,
			   (SELECT COUNT(*) FROM client_contacts cc WHERE cc.client_id = c.id)
		FROM clients c
		JOIN projects p ON p.client_name = c.name
		WHERE p.id = $1 AND c.user_id = $2
		LIMIT 1`

	var client ClientFullContext
	err := s.pool.QueryRow(ctx, query, projectID, userID).Scan(
		&client.ID, &client.Name, &client.Email, &client.Industry, &client.Status, &client.ContactCount,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

func (s *TieredContextService) getProjectTeam(ctx context.Context, projectID uuid.UUID, userID string) ([]TeamMemberContext, error) {
	query := `
		SELECT DISTINCT tm.id, tm.name, tm.role, tm.status
		FROM team_members tm
		JOIN tasks t ON t.assignee_id = tm.id
		WHERE t.project_id = $1 AND tm.user_id = $2
		LIMIT 10`

	rows, err := s.pool.Query(ctx, query, projectID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var team []TeamMemberContext
	for rows.Next() {
		var tm TeamMemberContext
		if err := rows.Scan(&tm.ID, &tm.Name, &tm.Role, &tm.Status); err != nil {
			continue
		}
		team = append(team, tm)
	}
	return team, nil
}

func (s *TieredContextService) getContextFull(ctx context.Context, contextID uuid.UUID, userID string) (*ContextFullContext, error) {
	query := `
		SELECT c.id, c.name, c.type::text, COALESCE(c.content, ''),
			   COALESCE(c.system_prompt_template, ''), COALESCE(c.word_count, 0),
			   EXISTS(SELECT 1 FROM context_embeddings ce WHERE ce.context_id = c.id)
		FROM contexts c
		WHERE c.id = $1 AND c.user_id = $2 AND c.is_archived = false`

	var doc ContextFullContext
	err := s.pool.QueryRow(ctx, query, contextID, userID).Scan(
		&doc.ID, &doc.Name, &doc.Type, &doc.Content, &doc.SystemPrompt,
		&doc.WordCount, &doc.HasEmbeddings,
	)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *TieredContextService) getNodeSummary(ctx context.Context, nodeID uuid.UUID, userID string) (*NodeSummary, error) {
	query := `
		SELECT n.id, n.name, n.type::text, COALESCE(n.health::text, ''),
			   COALESCE(n.purpose, '')
		FROM nodes n
		WHERE n.id = $1 AND n.user_id = $2 AND n.is_archived = false`

	var node NodeSummary
	err := s.pool.QueryRow(ctx, query, nodeID, userID).Scan(
		&node.ID, &node.Name, &node.Type, &node.Health, &node.Purpose,
	)
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *TieredContextService) getOtherProjects(ctx context.Context, excludeID *uuid.UUID, userID string, limit int) ([]EntitySummary, error) {
	query := `
		SELECT p.id, p.name, p.status::text
		FROM projects p
		WHERE p.user_id = $1 AND ($2::uuid IS NULL OR p.id != $2)
		ORDER BY p.updated_at DESC
		LIMIT $3`

	rows, err := s.pool.Query(ctx, query, userID, excludeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []EntitySummary
	for rows.Next() {
		var p EntitySummary
		if err := rows.Scan(&p.ID, &p.Name, &p.Type); err != nil {
			continue
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *TieredContextService) getSiblingContexts(ctx context.Context, selectedIDs []uuid.UUID, userID string, limit int) ([]EntitySummary, error) {
	// Get parent IDs of selected contexts, then get their siblings
	query := `
		SELECT DISTINCT c2.id, c2.name, c2.type::text
		FROM contexts c1
		JOIN contexts c2 ON c2.parent_id = c1.parent_id
		WHERE c1.id = ANY($1)
		  AND c2.id != ALL($1)
		  AND c2.user_id = $2
		  AND c2.is_archived = false
		ORDER BY c2.updated_at DESC
		LIMIT $3`

	rows, err := s.pool.Query(ctx, query, selectedIDs, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var siblings []EntitySummary
	for rows.Next() {
		var s EntitySummary
		if err := rows.Scan(&s.ID, &s.Name, &s.Type); err != nil {
			continue
		}
		siblings = append(siblings, s)
	}
	return siblings, nil
}

func (s *TieredContextService) getRelatedClients(ctx context.Context, userID string, limit int) ([]EntitySummary, error) {
	query := `
		SELECT c.id, c.name, c.status::text
		FROM clients c
		WHERE c.user_id = $1
		ORDER BY c.updated_at DESC
		LIMIT $2`

	rows, err := s.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []EntitySummary
	for rows.Next() {
		var c EntitySummary
		if err := rows.Scan(&c.ID, &c.Name, &c.Type); err != nil {
			continue
		}
		clients = append(clients, c)
	}
	return clients, nil
}

func (s *TieredContextService) getAllProjectNames(ctx context.Context, userID string, limit int) ([]EntitySummary, error) {
	return s.getOtherProjects(ctx, nil, userID, limit)
}

func (s *TieredContextService) getAllContextNames(ctx context.Context, userID string, limit int) ([]EntitySummary, error) {
	query := `
		SELECT c.id, c.name, c.type::text
		FROM contexts c
		WHERE c.user_id = $1 AND c.is_archived = false
		ORDER BY c.updated_at DESC
		LIMIT $2`

	rows, err := s.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contexts []EntitySummary
	for rows.Next() {
		var c EntitySummary
		if err := rows.Scan(&c.ID, &c.Name, &c.Type); err != nil {
			continue
		}
		contexts = append(contexts, c)
	}
	return contexts, nil
}

func (s *TieredContextService) getAllClientNames(ctx context.Context, userID string, limit int) ([]EntitySummary, error) {
	return s.getRelatedClients(ctx, userID, limit)
}

// getRelevantMemories loads personal memories ordered by importance, scoped to project if available.
func (s *TieredContextService) getRelevantMemories(ctx context.Context, userID string, projectID *uuid.UUID, limit int) ([]MemoryFullContext, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, title, summary, memory_type, COALESCE(importance_score, 0.5)
		FROM memories
		WHERE user_id = $1
		  AND is_active = true
		  AND ($2::uuid IS NULL OR project_id = $2 OR project_id IS NULL)
		ORDER BY
			CASE WHEN is_pinned THEN 0 ELSE 1 END,
			importance_score DESC,
			updated_at DESC
		LIMIT $3
	`

	rows, err := s.pool.Query(ctx, query, userID, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var memories []MemoryFullContext
	for rows.Next() {
		var m MemoryFullContext
		if err := rows.Scan(&m.ID, &m.Title, &m.Summary, &m.MemoryType, &m.ImportanceScore); err != nil {
			continue
		}
		memories = append(memories, m)
	}
	return memories, nil
}

// touchMemories updates last_accessed_at and increments access_count for loaded memories.
// Called async (fire-and-forget) to avoid adding latency.
func (s *TieredContextService) touchMemories(ctx context.Context, memoryIDs []uuid.UUID) {
	if len(memoryIDs) == 0 {
		return
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE memories
		SET last_accessed_at = NOW(),
		    access_count = COALESCE(access_count, 0) + 1
		WHERE id = ANY($1)
	`, memoryIDs)
	if err != nil {
		slog.Warn("touchMemories failed", "error", err, "count", len(memoryIDs))
	}
}

// getDocumentFull retrieves full document content for context injection.
// Waits for document processing to complete if needed.
func (s *TieredContextService) getDocumentFull(ctx context.Context, docID uuid.UUID, userID string) (*DocumentContext, error) {
	// Query from uploaded_documents table with extracted_text column
	query := `
		SELECT d.id, d.original_filename, d.display_name, d.mime_type,
			   COALESCE(d.extracted_text, '') as content,
			   d.processing_status,
			   (SELECT COUNT(*) FROM document_chunks WHERE document_id = d.id) as chunk_count
		FROM uploaded_documents d
		WHERE d.id = $1 AND d.user_id = $2
	`

	// Poll for document processing completion (max 10 seconds)
	maxWait := 10
	for attempt := 0; attempt < maxWait; attempt++ {
		row := s.pool.QueryRow(ctx, query, docID, userID)

		var doc DocumentContext
		var content string
		var status string
		err := row.Scan(&doc.ID, &doc.Filename, &doc.DisplayName, &doc.MimeType, &content, &status, &doc.ChunkCount)
		if err != nil {
			if attempt < maxWait-1 {
				// Document might not be inserted yet, wait and retry
				slog.Info("Document not found, waiting...", "docID", docID, "attempt", attempt+1)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Second):
					continue
				}
			}
			slog.Error("Failed to get document for context", "docID", docID, "userID", userID, "error", err)
			return nil, err
		}

		// If still processing and content is empty, wait for extraction
		if status == "processing" && content == "" {
			if attempt < maxWait-1 {
				slog.Info("Document still processing, waiting...", "docID", docID, "status", status, "attempt", attempt+1)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Second):
					continue
				}
			}
		}

		slog.Info("Document retrieved for context injection",
			"docID", docID,
			"filename", doc.Filename,
			"status", status,
			"contentLength", len(content),
			"chunkCount", doc.ChunkCount)

		// Truncate content if too large (max 50KB for context injection)
		const maxContentLen = 50 * 1024
		if len(content) > maxContentLen {
			doc.Content = content[:maxContentLen] + "\n\n[Content truncated - document is too large]"
		} else {
			doc.Content = content
		}

		return &doc, nil
	}

	return nil, fmt.Errorf("timeout waiting for document processing")
}
