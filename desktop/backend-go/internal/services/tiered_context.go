package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// BuildTieredContext creates the full tiered context for an AI request
func (s *TieredContextService) BuildTieredContext(
	ctx context.Context,
	req TieredContextRequest,
) (*TieredContext, error) {
	tc := &TieredContext{
		Level1: &FullContext{},
		Level2: &AwarenessContext{},
		Level3: &OnDemandRegistry{},
	}

	// Build Level 1: Full context for selected items
	if err := s.buildLevel1(ctx, req, tc.Level1); err != nil {
		return nil, fmt.Errorf("build level 1: %w", err)
	}

	// Build Level 2: Awareness of related items
	if err := s.buildLevel2(ctx, req, tc.Level1, tc.Level2); err != nil {
		return nil, fmt.Errorf("build level 2: %w", err)
	}

	// Build Level 3: Registry of on-demand entities
	if err := s.buildLevel3(ctx, req, tc.Level3); err != nil {
		return nil, fmt.Errorf("build level 3: %w", err)
	}

	return tc, nil
}

// CompressConversation uses the summarizer to compress old messages if the list is too long
func (s *TieredContextService) CompressConversation(ctx context.Context, messages []ChatMessage, threshold int) ([]ChatMessage, string, error) {
	if s.summarizer == nil || len(messages) <= threshold {
		return messages, "", nil
	}

	// Keep last threshold/2 messages as recent context, summarize the rest
	recentCount := threshold / 2
	if recentCount < 5 {
		recentCount = 5
	}
	if recentCount > 15 {
		recentCount = 15
	}

	return s.summarizer.HierarchicalSummarize(ctx, messages, recentCount)
}

// buildLevel1 populates full context for selected items
func (s *TieredContextService) buildLevel1(
	ctx context.Context,
	req TieredContextRequest,
	level1 *FullContext,
) error {
	// Track contexts already included to avoid duplicates.
	seenContexts := make(map[uuid.UUID]struct{}, len(req.ContextIDs)+8)

	// 0. If a business node is selected, include its context plus ancestor contexts (inheritance).
	if req.NodeID != nil {
		chain, err := s.getNodeAncestry(ctx, *req.NodeID, req.UserID, 8)
		if err == nil {
			for _, nc := range chain {
				if nc.ContextID == nil {
					continue
				}
				if _, ok := seenContexts[*nc.ContextID]; ok {
					continue
				}
				// Skip if the user already explicitly selected it; we'll load it in the normal pass.
				alreadySelected := false
				for _, selected := range req.ContextIDs {
					if selected == *nc.ContextID {
						alreadySelected = true
						break
					}
				}
				if alreadySelected {
					continue
				}

				doc, err := s.getContextFull(ctx, *nc.ContextID, req.UserID)
				if err == nil {
					level1.Contexts = append(level1.Contexts, *doc)
					seenContexts[*nc.ContextID] = struct{}{}
				}
			}
		}
	}

	// 1. Get selected project with full details
	if req.ProjectID != nil {
		project, err := s.getProjectFull(ctx, *req.ProjectID, req.UserID)
		if err == nil {
			level1.Project = project

			// Get project tasks
			tasks, err := s.getProjectTasks(ctx, *req.ProjectID, req.UserID)
			if err == nil {
				level1.Tasks = tasks
			}

			// Get linked client if project has one
			client, err := s.getProjectClient(ctx, *req.ProjectID, req.UserID)
			if err == nil && client != nil {
				level1.LinkedClient = client
			}

			// Get assigned team members
			team, err := s.getProjectTeam(ctx, *req.ProjectID, req.UserID)
			if err == nil {
				level1.TeamMembers = team
			}
		}
	}

	// 2. Get selected contexts with full details
	for _, ctxID := range req.ContextIDs {
		if _, ok := seenContexts[ctxID]; ok {
			continue
		}
		doc, err := s.getContextFull(ctx, ctxID, req.UserID)
		if err == nil {
			level1.Contexts = append(level1.Contexts, *doc)
			seenContexts[ctxID] = struct{}{}
		}
	}

	// 3. Get attached documents with full content
	for _, docID := range req.DocumentIDs {
		doc, err := s.getDocumentFull(ctx, docID, req.UserID)
		if err == nil {
			level1.Documents = append(level1.Documents, *doc)
		}
	}

	// 4. Load relevant personal memories for this context
	memories, err := s.getRelevantMemories(ctx, req.UserID, req.ProjectID, 10)
	if err == nil && len(memories) > 0 {
		level1.Memories = memories
		// Async: touch access timestamps for loaded memories (zero latency)
		memIDs := make([]uuid.UUID, len(memories))
		for i, m := range memories {
			memIDs[i] = m.ID
		}
		go s.touchMemories(context.Background(), memIDs)
	}

	return nil
}

// buildLevel2 populates awareness context for related items
func (s *TieredContextService) buildLevel2(
	ctx context.Context,
	req TieredContextRequest,
	level1 *FullContext,
	level2 *AwarenessContext,
) error {
	// 1. Get node info if provided
	if req.NodeID != nil {
		node, err := s.getNodeSummary(ctx, *req.NodeID, req.UserID)
		if err == nil {
			level2.NodeInfo = node
		}

		// Also include node lineage (parents) for situational awareness.
		if chain, err := s.getNodeAncestry(ctx, *req.NodeID, req.UserID, 8); err == nil {
			for _, nc := range chain {
				if nc.ID == *req.NodeID {
					continue
				}
				level2.ParentNodes = append(level2.ParentNodes, NodeSummary{
					ID:      nc.ID,
					Name:    nc.Name,
					Type:    nc.Type,
					Health:  nc.Health,
					Purpose: nc.Purpose,
				})
			}
		}

		// Get other projects (not the selected one)
		projects, err := s.getOtherProjects(ctx, req.ProjectID, req.UserID, 10)
		if err == nil {
			level2.OtherProjects = projects
		}
	}

	// 2. Get sibling contexts (same parent as selected contexts)
	if len(req.ContextIDs) > 0 {
		siblings, err := s.getSiblingContexts(ctx, req.ContextIDs, req.UserID, 10)
		if err == nil {
			level2.SiblingContexts = siblings
		}
	}

	// 3. Get related clients
	clients, err := s.getRelatedClients(ctx, req.UserID, 5)
	if err == nil {
		// Filter out the linked client if already in Level 1
		for _, c := range clients {
			if level1.LinkedClient == nil || c.ID != level1.LinkedClient.ID {
				level2.RelatedClients = append(level2.RelatedClients, c)
			}
		}
	}

	// 4. User facts (global personal/contextual preferences)
	if req.UserID != "" {
		facts, err := s.getUserFacts(ctx, req.UserID, 20)
		if err == nil {
			level2.UserFacts = facts
		}
	}

	return nil
}

// buildLevel3 builds the on-demand entity registry
func (s *TieredContextService) buildLevel3(
	ctx context.Context,
	req TieredContextRequest,
	level3 *OnDemandRegistry,
) error {
	// Get a lightweight registry of all fetchable entities
	// This tells the AI what it CAN fetch if needed

	// Projects
	projects, _ := s.getAllProjectNames(ctx, req.UserID, 20)
	for _, p := range projects {
		level3.AvailableEntities = append(level3.AvailableEntities, OnDemandEntity{
			Type: "project",
			ID:   p.ID,
			Name: p.Name,
		})
	}

	// Contexts (documents)
	contexts, _ := s.getAllContextNames(ctx, req.UserID, 30)
	for _, c := range contexts {
		level3.AvailableEntities = append(level3.AvailableEntities, OnDemandEntity{
			Type: "context",
			ID:   c.ID,
			Name: c.Name,
		})
	}

	// Clients
	clients, _ := s.getAllClientNames(ctx, req.UserID, 10)
	for _, c := range clients {
		level3.AvailableEntities = append(level3.AvailableEntities, OnDemandEntity{
			Type: "client",
			ID:   c.ID,
			Name: c.Name,
		})
	}

	return nil
}

// ScopedRAGSearch performs embedding search ONLY within specified contexts
func (s *TieredContextService) ScopedRAGSearch(
	ctx context.Context,
	query string,
	contextIDs []uuid.UUID,
	userID string,
	limit int,
) ([]RelevantBlock, error) {
	if len(contextIDs) == 0 || s.embeddingService == nil {
		return nil, nil
	}

	// Use the embedding service's scoped search
	return s.embeddingService.ScopedSimilaritySearch(ctx, query, contextIDs, userID, limit)
}
