package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// Context Injection Methods
// ============================================================================

// InjectContextIntoConversation prepares context for injection into a conversation
func (s *ProjectContextService) InjectContextIntoConversation(
	ctx context.Context,
	userID string,
	projectID, nodeID *uuid.UUID,
	agentType, focusMode string,
	maxTokens int,
) (*InjectedContext, error) {
	ic := &InjectedContext{
		LoadedItems:    make([]ContextItem, 0),
		TokenBreakdown: make(map[string]int),
	}

	items := make([]BudgetItem, 0, 32)
	items = append(items, BudgetItem{
		Key:      "header",
		Type:     "meta",
		Content:  "\n## Loaded Context\n\n",
		Priority: 100,
		Pinned:   true,
	})

	// Load project context if project is selected
	if projectID != nil {
		pc, err := s.LoadProjectContext(ctx, userID, *projectID)
		if err == nil {
			// Add project info (pinned)
			var psb strings.Builder
			psb.WriteString(fmt.Sprintf("### Active Project: %s\n", pc.Project.Name))
			if pc.Project.Description != "" {
				psb.WriteString(fmt.Sprintf("%s\n", pc.Project.Description))
			}
			psb.WriteString("\n")
			items = append(items, BudgetItem{Key: "project", Type: "project", Content: psb.String(), Priority: 90, Pinned: true})

			// Add memories
			if len(pc.Memories) > 0 {
				for _, m := range pc.Memories {
					items = append(items, BudgetItem{
						Key:      "memory:" + m.ID.String(),
						Type:     "memory",
						Content:  fmt.Sprintf("### Relevant Memories\n- [%s] %s: %s\n\n", m.MemoryType, m.Title, m.Summary),
						Priority: 60,
						Pinned:   false,
					})
					ic.LoadedItems = append(ic.LoadedItems, ContextItem{
						ID:    m.ID,
						Type:  "memory",
						Title: m.Title,
					})
				}
			}

			// Add user facts
			if len(pc.UserFacts) > 0 {
				var fsb strings.Builder
				fsb.WriteString("### User Facts\n")
				for _, f := range pc.UserFacts {
					fsb.WriteString(fmt.Sprintf("- %s: %s\n", f.FactKey, f.FactValue))
				}
				fsb.WriteString("\n")
				items = append(items, BudgetItem{Key: "user_facts", Type: "user_facts", Content: fsb.String(), Priority: 80, Pinned: true})
			}

			// Add recent conversation context
			if len(pc.Conversations) > 0 {
				for _, c := range pc.Conversations {
					items = append(items, BudgetItem{
						Key:      "conversation:" + c.ID.String(),
						Type:     "conversation",
						Content:  fmt.Sprintf("### Recent Discussion Context\n- %s\n\n", c.Summary),
						Priority: 50,
						Pinned:   false,
					})
				}
			}
		}
	}

	// Load node context if node is selected
	if nodeID != nil {
		nc, err := s.LoadNodeContext(ctx, userID, *nodeID)
		if err == nil {
			var nsb strings.Builder
			nsb.WriteString(fmt.Sprintf("### Business Context: %s (%s)\n", nc.Node.Name, nc.Node.Type))
			if nc.Node.Description != "" {
				nsb.WriteString(fmt.Sprintf("%s\n", nc.Node.Description))
			}
			nsb.WriteString("\n")
			items = append(items, BudgetItem{Key: "node", Type: "node", Content: nsb.String(), Priority: 70, Pinned: true})

			// Add node memories
			if len(nc.Memories) > 0 {
				for _, m := range nc.Memories {
					items = append(items, BudgetItem{
						Key:      "node_memory:" + m.ID.String(),
						Type:     "node_memory",
						Content:  fmt.Sprintf("### Node-Specific Memories\n- [%s] %s\n\n", m.MemoryType, m.Summary),
						Priority: 40,
						Pinned:   false,
					})
				}
			}
		}
	}

	res := ApplyTokenBudget(items, maxTokens)

	// Rebuild prompt addition from kept items
	var sb strings.Builder
	for _, it := range res.Kept {
		if it.Content == "" {
			continue
		}
		sb.WriteString(it.Content)
	}
	ic.SystemPromptAddition = sb.String()
	ic.TotalTokens = res.UsedTokens

	// Token breakdown from kept items
	ic.TokenBreakdown = make(map[string]int)
	for _, it := range res.Kept {
		if it.Type == "meta" {
			continue
		}
		it = ensureTokenCount(it)
		ic.TokenBreakdown[it.Type] += it.TokenCount
	}

	return ic, nil
}
