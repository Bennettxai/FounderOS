package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ============================================================================
// Node Context Methods
// ============================================================================

// LoadNodeContext loads context when a specific node is selected
func (s *ProjectContextService) LoadNodeContext(ctx context.Context, userID string, nodeID uuid.UUID) (*NodeContext, error) {
	nc := &NodeContext{
		Ancestors: make([]*Node, 0),
		Memories:  make([]Memory, 0),
		Projects:  make([]Project, 0),
	}

	// 1. Get node details
	node, err := s.getNode(ctx, userID, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	nc.Node = node

	// 2. Get node ancestors (path to root)
	ancestors, err := s.getNodeAncestors(ctx, userID, nodeID)
	if err == nil {
		nc.Ancestors = ancestors
	}

	// 3. Get node's context profile
	if s.contextService != nil {
		profile, _ := s.contextService.GetContextProfile(ctx, userID, "node", nodeID)
		nc.Profile = profile
	}

	// 4. Load memories for this specific node
	memories, err := s.getNodeMemories(ctx, userID, nodeID, 10)
	if err == nil {
		nc.Memories = memories
	}

	// 5. Get projects under this node
	projects, err := s.getNodeProjects(ctx, userID, nodeID, 10)
	if err == nil {
		nc.Projects = projects
	}

	return nc, nil
}

// getNode retrieves node details
func (s *ProjectContextService) getNode(ctx context.Context, userID string, nodeID uuid.UUID) (*Node, error) {
	var n Node

	err := s.pool.QueryRow(ctx, `
		SELECT id, user_id, name, type, description, parent_id, created_at, updated_at
		FROM nodes
		WHERE id = $1 AND user_id = $2
	`, nodeID, userID).Scan(
		&n.ID, &n.UserID, &n.Name, &n.Type, &n.Description, &n.ParentID, &n.CreatedAt, &n.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("node not found")
	}
	if err != nil {
		return nil, err
	}

	return &n, nil
}

// getNodeAncestors retrieves ancestor nodes (path to root)
func (s *ProjectContextService) getNodeAncestors(ctx context.Context, userID string, nodeID uuid.UUID) ([]*Node, error) {
	// Use recursive CTE to get ancestors
	rows, err := s.pool.Query(ctx, `
		WITH RECURSIVE ancestors AS (
			SELECT id, user_id, name, type, description, parent_id, created_at, updated_at, 0 as depth
			FROM nodes
			WHERE id = $1 AND user_id = $2

			UNION ALL

			SELECT n.id, n.user_id, n.name, n.type, n.description, n.parent_id, n.created_at, n.updated_at, a.depth + 1
			FROM nodes n
			JOIN ancestors a ON n.id = a.parent_id
			WHERE n.user_id = $2
		)
		SELECT id, user_id, name, type, description, parent_id, created_at, updated_at
		FROM ancestors
		WHERE depth > 0
		ORDER BY depth DESC
	`, nodeID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ancestors []*Node
	for rows.Next() {
		var n Node
		err := rows.Scan(&n.ID, &n.UserID, &n.Name, &n.Type, &n.Description, &n.ParentID, &n.CreatedAt, &n.UpdatedAt)
		if err != nil {
			continue
		}
		ancestors = append(ancestors, &n)
	}

	return ancestors, nil
}

// getNodeMemories retrieves memories for a node
func (s *ProjectContextService) getNodeMemories(ctx context.Context, userID string, nodeID uuid.UUID, limit int) ([]Memory, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, title, summary, content, memory_type, category, source_type,
		       source_id, project_id, node_id, importance_score, access_count, is_pinned,
		       tags, created_at, updated_at
		FROM memories
		WHERE user_id = $1 AND node_id = $2 AND is_active = true
		ORDER BY is_pinned DESC, importance_score DESC, created_at DESC
		LIMIT $3
	`, userID, nodeID, limit)
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

// getNodeProjects retrieves projects under a node
func (s *ProjectContextService) getNodeProjects(ctx context.Context, userID string, nodeID uuid.UUID, limit int) ([]Project, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, name, description, status, priority, client_id, created_at, updated_at
		FROM projects
		WHERE user_id = $1 AND node_id = $2 AND is_archived = false
		ORDER BY updated_at DESC
		LIMIT $3
	`, userID, nodeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Status, &p.Priority, &p.ClientID, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			continue
		}
		projects = append(projects, p)
	}

	return projects, nil
}
