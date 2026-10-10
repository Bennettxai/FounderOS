package services

import (
	"fmt"
	"strings"
)

// FormatForAI formats the tiered context as a system prompt string
func (tc *TieredContext) FormatForAI() string {
	var sb strings.Builder

	sb.WriteString("## Context Overview\n\n")

	// Level 1: Full Context
	if tc.Level1 != nil {
		sb.WriteString("### Primary Focus (Full Details)\n\n")

		if tc.Level1.Project != nil {
			sb.WriteString(fmt.Sprintf("**Active Project: %s**\n", tc.Level1.Project.Name))
			sb.WriteString(fmt.Sprintf("- Status: %s | Priority: %s\n",
				tc.Level1.Project.Status, tc.Level1.Project.Priority))
			if tc.Level1.Project.Description != "" {
				sb.WriteString(fmt.Sprintf("- Description: %s\n", tc.Level1.Project.Description))
			}
			if tc.Level1.Project.ClientName != "" {
				sb.WriteString(fmt.Sprintf("- Client: %s\n", tc.Level1.Project.ClientName))
			}
			sb.WriteString("\n")

			// Tasks
			if len(tc.Level1.Tasks) > 0 {
				sb.WriteString("**Project Tasks:**\n")
				for _, task := range tc.Level1.Tasks {
					sb.WriteString(fmt.Sprintf("- [%s] %s (%s)", task.Status, task.Title, task.Priority))
					if task.DueDate != "" {
						sb.WriteString(fmt.Sprintf(" - Due: %s", task.DueDate))
					}
					if task.AssigneeName != "" {
						sb.WriteString(fmt.Sprintf(" - Assignee: %s", task.AssigneeName))
					}
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}
		}

		// Personal Memories
		if len(tc.Level1.Memories) > 0 {
			sb.WriteString("**Personal Memories:**\n")
			for _, mem := range tc.Level1.Memories {
				sb.WriteString(fmt.Sprintf("- [%s] **%s**: %s\n", mem.MemoryType, mem.Title, mem.Summary))
			}
			sb.WriteString("\n")
		}

		// Selected Documents
		if len(tc.Level1.Contexts) > 0 {
			sb.WriteString("**Selected Documents:**\n")
			for _, doc := range tc.Level1.Contexts {
				sb.WriteString(fmt.Sprintf("- **%s** (%s, %d words)\n", doc.Name, doc.Type, doc.WordCount))
				if doc.SystemPrompt != "" {
					sb.WriteString(fmt.Sprintf("  System context: %s\n", truncateText(doc.SystemPrompt, 200)))
				}
				if doc.Content != "" {
					content := truncateText(doc.Content, 1500)
					sb.WriteString(fmt.Sprintf("  Content:\n  > %s\n", content))
				}
			}
			sb.WriteString("\n")
		}

		// Linked Client
		if tc.Level1.LinkedClient != nil {
			sb.WriteString(fmt.Sprintf("**Linked Client: %s**\n", tc.Level1.LinkedClient.Name))
			sb.WriteString(fmt.Sprintf("- Status: %s | Industry: %s\n",
				tc.Level1.LinkedClient.Status, tc.Level1.LinkedClient.Industry))
			sb.WriteString("\n")
		}

		// Team Members
		if len(tc.Level1.TeamMembers) > 0 {
			sb.WriteString("**Team Members:**\n")
			for _, tm := range tc.Level1.TeamMembers {
				sb.WriteString(fmt.Sprintf("- %s (%s) - %s\n", tm.Name, tm.Role, tm.Status))
			}
			sb.WriteString("\n")
		}

		// RAG Results
		if len(tc.Level1.RelevantRAG) > 0 {
			sb.WriteString("**Relevant Knowledge (from selected documents):**\n")
			for i, block := range tc.Level1.RelevantRAG {
				sb.WriteString(fmt.Sprintf("%d. From \"%s\" (%.0f%% match):\n",
					i+1, block.DocumentName, block.Similarity*100))
				sb.WriteString(fmt.Sprintf("   > %s\n", truncateText(block.BlockContent, 300)))
			}
			sb.WriteString("\n")
		}

		// Attached Documents (uploaded by user)
		if len(tc.Level1.Documents) > 0 {
			sb.WriteString("**Attached Documents (uploaded by user):**\n")
			for _, doc := range tc.Level1.Documents {
				displayName := doc.DisplayName
				if displayName == "" {
					displayName = doc.Filename
				}
				sb.WriteString(fmt.Sprintf("\n--- Document: %s ---\n", displayName))
				if doc.Content != "" {
					sb.WriteString(fmt.Sprintf("%s\n", doc.Content))
				} else {
					sb.WriteString(fmt.Sprintf("[Document has %d chunks - use RAG search for content]\n", doc.ChunkCount))
				}
			}
			sb.WriteString("\n")
		}
	}

	// Level 2: Awareness
	if tc.Level2 != nil && tc.hasLevel2Content() {
		sb.WriteString("### Context Awareness (Summaries Only)\n\n")

		if tc.Level2.NodeInfo != nil {
			sb.WriteString(fmt.Sprintf("**Business Node: %s** (%s)\n",
				tc.Level2.NodeInfo.Name, tc.Level2.NodeInfo.Type))
			if tc.Level2.NodeInfo.Purpose != "" {
				sb.WriteString(fmt.Sprintf("- Purpose: %s\n", tc.Level2.NodeInfo.Purpose))
			}
			if tc.Level2.NodeInfo.Health != "" {
				sb.WriteString(fmt.Sprintf("- Health: %s\n", tc.Level2.NodeInfo.Health))
			}
			sb.WriteString("\n")
		}

		if len(tc.Level2.ParentNodes) > 0 {
			sb.WriteString("**Node Lineage (parents):**\n")
			for _, pn := range tc.Level2.ParentNodes {
				sb.WriteString(fmt.Sprintf("- %s (%s)\n", pn.Name, pn.Type))
			}
			sb.WriteString("\n")
		}

		if len(tc.Level2.UserFacts) > 0 {
			sb.WriteString("**User Facts (confirmed):**\n")
			for _, f := range tc.Level2.UserFacts {
				if strings.TrimSpace(f.FactKey) == "" || strings.TrimSpace(f.FactValue) == "" {
					continue
				}
				sb.WriteString(fmt.Sprintf("- %s: %s\n", f.FactKey, f.FactValue))
			}
			sb.WriteString("\n")
		}

		if len(tc.Level2.OtherProjects) > 0 {
			sb.WriteString("**Other Projects in Scope:** ")
			names := make([]string, len(tc.Level2.OtherProjects))
			for i, p := range tc.Level2.OtherProjects {
				names[i] = p.Name
			}
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString("\n\n")
		}

		if len(tc.Level2.SiblingContexts) > 0 {
			sb.WriteString("**Related Documents:** ")
			names := make([]string, len(tc.Level2.SiblingContexts))
			for i, c := range tc.Level2.SiblingContexts {
				names[i] = c.Name
			}
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString("\n\n")
		}

		if len(tc.Level2.RelatedClients) > 0 {
			sb.WriteString("**Other Clients:** ")
			names := make([]string, len(tc.Level2.RelatedClients))
			for i, c := range tc.Level2.RelatedClients {
				names[i] = c.Name
			}
			sb.WriteString(strings.Join(names, ", "))
			sb.WriteString("\n\n")
		}
	}

	// Level 3: On-Demand Notice
	if tc.Level3 != nil && len(tc.Level3.AvailableEntities) > 0 {
		sb.WriteString("### On-Demand Context\n")
		sb.WriteString("You can use the `get_entity_context` tool to retrieve full details for any entity mentioned above or from the user's workspace.\n\n")
	}

	return sb.String()
}

// FormatForAIWithTokenBudget formats tiered context but enforces a strict token budget
// across context types (documents, RAG blocks, awareness summaries, etc) using
// priority-based LRU eviction.
//
// If maxTokens <= 0, it falls back to FormatForAI().
func (tc *TieredContext) FormatForAIWithTokenBudget(maxTokens int) string {
	if maxTokens <= 0 {
		return tc.FormatForAI()
	}

	items := tc.buildBudgetItems()
	res := ApplyTokenBudget(items, maxTokens)

	var sb strings.Builder
	for _, it := range res.Kept {
		if it.Content == "" {
			continue
		}
		sb.WriteString(it.Content)
		if !strings.HasSuffix(it.Content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

func (tc *TieredContext) buildBudgetItems() []BudgetItem {
	items := make([]BudgetItem, 0, 16)

	// Global header (pinned)
	items = append(items, BudgetItem{
		Key:      "header",
		Type:     "meta",
		Content:  "## Context Overview\n",
		Priority: 100,
		Pinned:   true,
	})

	// Level 1: primary focus
	if tc.Level1 != nil {
		// User facts (high priority)
		if tc.Level2 != nil && len(tc.Level2.UserFacts) > 0 {
			var sb strings.Builder
			sb.WriteString("### User Facts (confirmed)\n\n")
			for _, f := range tc.Level2.UserFacts {
				if f.FactKey == "" || f.FactValue == "" {
					continue
				}
				sb.WriteString(fmt.Sprintf("- %s: %s\n", f.FactKey, f.FactValue))
			}
			items = append(items, BudgetItem{Key: "user_facts", Type: "user_facts", Content: sb.String(), Priority: 95, Pinned: true})
		}

		// Project + tasks + client + team (high priority)
		if tc.Level1.Project != nil {
			var sb strings.Builder
			sb.WriteString("### Primary Focus (Full Details)\n\n")
			sb.WriteString(fmt.Sprintf("**Active Project: %s**\n", tc.Level1.Project.Name))
			sb.WriteString(fmt.Sprintf("- Status: %s | Priority: %s\n", tc.Level1.Project.Status, tc.Level1.Project.Priority))
			if tc.Level1.Project.Description != "" {
				sb.WriteString(fmt.Sprintf("- Description: %s\n", tc.Level1.Project.Description))
			}
			if tc.Level1.Project.ClientName != "" {
				sb.WriteString(fmt.Sprintf("- Client: %s\n", tc.Level1.Project.ClientName))
			}
			sb.WriteString("\n")

			if len(tc.Level1.Tasks) > 0 {
				sb.WriteString("**Project Tasks:**\n")
				for _, task := range tc.Level1.Tasks {
					sb.WriteString(fmt.Sprintf("- [%s] %s (%s)", task.Status, task.Title, task.Priority))
					if task.DueDate != "" {
						sb.WriteString(fmt.Sprintf(" - Due: %s", task.DueDate))
					}
					if task.AssigneeName != "" {
						sb.WriteString(fmt.Sprintf(" - Assignee: %s", task.AssigneeName))
					}
					sb.WriteString("\n")
				}
				sb.WriteString("\n")
			}

			if tc.Level1.LinkedClient != nil {
				sb.WriteString(fmt.Sprintf("**Linked Client: %s**\n", tc.Level1.LinkedClient.Name))
				sb.WriteString(fmt.Sprintf("- Status: %s | Industry: %s\n\n", tc.Level1.LinkedClient.Status, tc.Level1.LinkedClient.Industry))
			}

			if len(tc.Level1.TeamMembers) > 0 {
				sb.WriteString("**Team Members:**\n")
				for _, tm := range tc.Level1.TeamMembers {
					sb.WriteString(fmt.Sprintf("- %s (%s) - %s\n", tm.Name, tm.Role, tm.Status))
				}
				sb.WriteString("\n")
			}

			items = append(items, BudgetItem{Key: "project", Type: "project", Content: sb.String(), Priority: 90, Pinned: true})
		}

		// Selected documents (medium priority; each document is a separate block for eviction)
		if len(tc.Level1.Contexts) > 0 {
			for _, doc := range tc.Level1.Contexts {
				var sb strings.Builder
				sb.WriteString("**Selected Documents:**\n")
				sb.WriteString(fmt.Sprintf("- **%s** (%s, %d words)\n", doc.Name, doc.Type, doc.WordCount))
				if doc.SystemPrompt != "" {
					sb.WriteString(fmt.Sprintf("  System context: %s\n", truncateText(doc.SystemPrompt, 200)))
				}
				if doc.Content != "" {
					content := truncateText(doc.Content, 1500)
					sb.WriteString(fmt.Sprintf("  Content:\n  > %s\n", content))
				}
				sb.WriteString("\n")
				items = append(items, BudgetItem{Key: "selected_doc:" + doc.ID.String(), Type: "selected_document", Content: sb.String(), Priority: 80, Pinned: false})
			}
		}

		// Personal memories (high-medium priority)
		if len(tc.Level1.Memories) > 0 {
			var sb strings.Builder
			sb.WriteString("**Personal Memories:**\n")
			for _, mem := range tc.Level1.Memories {
				sb.WriteString(fmt.Sprintf("- [%s] **%s**: %s\n", mem.MemoryType, mem.Title, mem.Summary))
			}
			sb.WriteString("\n")
			items = append(items, BudgetItem{Key: "memories", Type: "memories", Content: sb.String(), Priority: 85, Pinned: false})
		}

		// RAG blocks (medium/low priority)
		if len(tc.Level1.RelevantRAG) > 0 {
			var sb strings.Builder
			sb.WriteString("**Relevant Knowledge (from selected documents):**\n")
			for i, block := range tc.Level1.RelevantRAG {
				sb.WriteString(fmt.Sprintf("%d. From \"%s\" (%.0f%% match):\n", i+1, block.DocumentName, block.Similarity*100))
				sb.WriteString(fmt.Sprintf("   > %s\n", truncateText(block.BlockContent, 300)))
			}
			sb.WriteString("\n")
			items = append(items, BudgetItem{Key: "rag", Type: "rag", Content: sb.String(), Priority: 65, Pinned: false})
		}

		// Attached documents (often large; lower than selected docs)
		if len(tc.Level1.Documents) > 0 {
			for _, doc := range tc.Level1.Documents {
				var sb strings.Builder
				displayName := doc.DisplayName
				if displayName == "" {
					displayName = doc.Filename
				}
				sb.WriteString(fmt.Sprintf("--- Document: %s ---\n", displayName))
				if doc.Content != "" {
					sb.WriteString(doc.Content)
					sb.WriteString("\n")
				} else {
					sb.WriteString(fmt.Sprintf("[Document has %d chunks - use RAG search for content]\n", doc.ChunkCount))
				}
				sb.WriteString("\n")
				items = append(items, BudgetItem{Key: "attached_doc:" + doc.ID.String(), Type: "attached_document", Content: sb.String(), Priority: 60, Pinned: false})
			}
		}
	}

	// Level 2: awareness (low priority)
	if tc.Level2 != nil && tc.hasLevel2Content() {
		var sb strings.Builder
		sb.WriteString("### Context Awareness (Summaries Only)\n\n")

		if tc.Level2.NodeInfo != nil {
			sb.WriteString(fmt.Sprintf("**Business Node: %s** (%s)\n", tc.Level2.NodeInfo.Name, tc.Level2.NodeInfo.Type))
			if tc.Level2.NodeInfo.Purpose != "" {
				sb.WriteString(fmt.Sprintf("- Purpose: %s\n", tc.Level2.NodeInfo.Purpose))
			}
			if tc.Level2.NodeInfo.Health != "" {
				sb.WriteString(fmt.Sprintf("- Health: %s\n", tc.Level2.NodeInfo.Health))
			}
			sb.WriteString("\n")
		}

		if len(tc.Level2.ParentNodes) > 0 {
			sb.WriteString("**Node Lineage (parents):**\n")
			for _, pn := range tc.Level2.ParentNodes {
				sb.WriteString(fmt.Sprintf("- %s (%s)\n", pn.Name, pn.Type))
			}
			sb.WriteString("\n")
		}

		if len(tc.Level2.OtherProjects) > 0 {
			sb.WriteString("**Other Active Projects:**\n")
			for _, p := range tc.Level2.OtherProjects {
				sb.WriteString(fmt.Sprintf("- %s\n", p.Name))
			}
			sb.WriteString("\n")
		}

		if len(tc.Level2.SiblingContexts) > 0 {
			sb.WriteString("**Related Documents:**\n")
			for _, c := range tc.Level2.SiblingContexts {
				sb.WriteString(fmt.Sprintf("- %s (%s)\n", c.Name, c.Type))
			}
			sb.WriteString("\n")
		}

		if len(tc.Level2.RelatedClients) > 0 {
			sb.WriteString("**Related Clients:**\n")
			for _, cl := range tc.Level2.RelatedClients {
				sb.WriteString(fmt.Sprintf("- %s\n", cl.Name))
			}
			sb.WriteString("\n")
		}

		items = append(items, BudgetItem{Key: "awareness", Type: "awareness", Content: sb.String(), Priority: 30, Pinned: false})
	}

	// Level 3: on-demand registry (very low priority)
	if tc.Level3 != nil && len(tc.Level3.AvailableEntities) > 0 {
		var sb strings.Builder
		sb.WriteString("### On-Demand Available Context\n\n")
		sb.WriteString("The following entities are available via tool calls (not automatically loaded):\n")
		for _, e := range tc.Level3.AvailableEntities {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.Type, e.Name))
		}
		sb.WriteString("\n")
		items = append(items, BudgetItem{Key: "on_demand", Type: "on_demand", Content: sb.String(), Priority: 10, Pinned: false})
	}

	return items
}

func (tc *TieredContext) hasLevel2Content() bool {
	if tc.Level2 == nil {
		return false
	}
	return tc.Level2.NodeInfo != nil ||
		len(tc.Level2.ParentNodes) > 0 ||
		len(tc.Level2.UserFacts) > 0 ||
		len(tc.Level2.OtherProjects) > 0 ||
		len(tc.Level2.SiblingContexts) > 0 ||
		len(tc.Level2.RelatedClients) > 0
}
