package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GetContextTree retrieves the context tree for a user
func (s *ContextService) GetContextTree(ctx context.Context, userID string, projectID, nodeID *uuid.UUID) (*ContextTree, error) {
	tree := &ContextTree{
		RootNode: &ContextTreeNode{
			ID:       uuid.Nil,
			Type:     "root",
			Name:     "Context Tree",
			Children: make([]*ContextTreeNode, 0),
		},
		LastUpdated: time.Now(),
	}

	// Get projects
	projectNodes, err := s.getProjectNodes(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	tree.RootNode.Children = append(tree.RootNode.Children, projectNodes...)

	// Count total items
	for _, child := range tree.RootNode.Children {
		tree.TotalItems += countTreeItems(child)
	}
	tree.RootNode.ItemCount = tree.TotalItems

	return tree, nil
}

// getProjectNodes retrieves project nodes for the tree
func (s *ContextService) getProjectNodes(ctx context.Context, userID string, projectID *uuid.UUID) ([]*ContextTreeNode, error) {
	query := `
		SELECT id, name, description, status
		FROM projects
		WHERE user_id = $1 AND is_archived = false
	`
	args := []any{userID}

	if projectID != nil {
		query += " AND id = $2"
		args = append(args, *projectID)
	}

	query += " ORDER BY updated_at DESC LIMIT 50"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	var nodes []*ContextTreeNode
	for rows.Next() {
		var id uuid.UUID
		var name, description, status string

		if err := rows.Scan(&id, &name, &description, &status); err != nil {
			continue
		}

		node := &ContextTreeNode{
			ID:          id,
			Type:        "project",
			Name:        name,
			Description: description,
			Icon:        "folder",
			Children:    make([]*ContextTreeNode, 0),
			Metadata: map[string]any{
				"status": status,
			},
		}

		// Get project children (memories, documents, etc.)
		children, itemCount := s.getProjectChildren(ctx, userID, id)
		node.Children = children
		node.ItemCount = itemCount

		nodes = append(nodes, node)
	}

	return nodes, nil
}

// getProjectChildren retrieves child items for a project
func (s *ContextService) getProjectChildren(ctx context.Context, userID string, projectID uuid.UUID) ([]*ContextTreeNode, int) {
	children := make([]*ContextTreeNode, 0)
	totalItems := 0

	// Memories category
	var memoryCount int
	s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM memories WHERE user_id = $1 AND project_id = $2 AND is_active = true
	`, userID, projectID).Scan(&memoryCount)

	if memoryCount > 0 {
		children = append(children, &ContextTreeNode{
			ID:        uuid.New(),
			Type:      "category",
			Name:      "Memories",
			Icon:      "brain",
			ItemCount: memoryCount,
		})
		totalItems += memoryCount
	}

	// Documents category
	var docCount int
	s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM uploaded_documents WHERE user_id = $1 AND project_id = $2
	`, userID, projectID).Scan(&docCount)

	if docCount > 0 {
		children = append(children, &ContextTreeNode{
			ID:        uuid.New(),
			Type:      "category",
			Name:      "Documents",
			Icon:      "file-text",
			ItemCount: docCount,
		})
		totalItems += docCount
	}

	// Artifacts category
	var artifactCount int
	s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM artifacts WHERE user_id = $1 AND project_id = $2
	`, userID, projectID).Scan(&artifactCount)

	if artifactCount > 0 {
		children = append(children, &ContextTreeNode{
			ID:        uuid.New(),
			Type:      "category",
			Name:      "Artifacts",
			Icon:      "file-code",
			ItemCount: artifactCount,
		})
		totalItems += artifactCount
	}

	// Conversations category
	var convCount int
	s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM conversations WHERE user_id = $1 AND project_id = $2
	`, userID, projectID).Scan(&convCount)

	if convCount > 0 {
		children = append(children, &ContextTreeNode{
			ID:        uuid.New(),
			Type:      "category",
			Name:      "Conversations",
			Icon:      "message-square",
			ItemCount: convCount,
		})
		totalItems += convCount
	}

	return children, totalItems
}

// GetTreeStatistics returns statistics about the context tree
func (s *ContextService) GetTreeStatistics(ctx context.Context, userID string) (*TreeStatistics, error) {
	stats := &TreeStatistics{
		ByType: make(map[string]int),
	}

	// Count projects
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects WHERE user_id = $1 AND is_archived = false`, userID).Scan(&stats.TotalProjects)

	// Count nodes
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM nodes WHERE user_id = $1`, userID).Scan(&stats.TotalNodes)

	// Count memories
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM memories WHERE user_id = $1 AND is_active = true`, userID).Scan(&stats.TotalMemories)

	// Count contexts
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM contexts WHERE user_id = $1 AND is_archived = false`, userID).Scan(&stats.TotalContexts)

	// Count artifacts
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM artifacts WHERE user_id = $1`, userID).Scan(&stats.TotalArtifacts)

	// Count documents
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM uploaded_documents WHERE user_id = $1`, userID).Scan(&stats.TotalDocuments)

	// Count voice notes
	s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM voice_notes WHERE user_id = $1`, userID).Scan(&stats.TotalVoiceNotes)

	// Rough token estimate across common tree sources.
	// NOTE: This is an approximation (chars/4) to avoid loading all rows.
	var totalTokens int
	{
		var t int
		// Contexts
		s.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(LENGTH(COALESCE(content, '')) / 4), 0)
			FROM contexts
			WHERE user_id = $1 AND is_archived = false
		`, userID).Scan(&t)
		totalTokens += t
	}
	{
		var t int
		// Artifacts
		s.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(LENGTH(COALESCE(content, '')) / 4), 0)
			FROM artifacts
			WHERE user_id = $1
		`, userID).Scan(&t)
		totalTokens += t
	}
	{
		var t int
		// Documents (extracted_text)
		s.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(LENGTH(COALESCE(extracted_text, '')) / 4), 0)
			FROM uploaded_documents
			WHERE user_id = $1
		`, userID).Scan(&t)
		totalTokens += t
	}
	{
		var t int
		// Memories
		s.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(LENGTH(COALESCE(content, '')) / 4), 0)
			FROM memories
			WHERE user_id = $1 AND is_active = true
		`, userID).Scan(&t)
		totalTokens += t
	}
	{
		var t int
		// Voice notes
		s.pool.QueryRow(ctx, `
			SELECT COALESCE(SUM(LENGTH(COALESCE(transcript, '')) / 4), 0)
			FROM voice_notes
			WHERE user_id = $1
		`, userID).Scan(&t)
		totalTokens += t
	}
	stats.TotalTokens = totalTokens

	stats.ByType["projects"] = stats.TotalProjects
	stats.ByType["nodes"] = stats.TotalNodes
	stats.ByType["memories"] = stats.TotalMemories
	stats.ByType["contexts"] = stats.TotalContexts
	stats.ByType["artifacts"] = stats.TotalArtifacts
	stats.ByType["documents"] = stats.TotalDocuments
	stats.ByType["voice_notes"] = stats.TotalVoiceNotes

	return stats, nil
}

// countTreeItems recursively counts items in a tree node
func countTreeItems(node *ContextTreeNode) int {
	count := node.ItemCount
	for _, child := range node.Children {
		count += countTreeItems(child)
	}
	return count
}
