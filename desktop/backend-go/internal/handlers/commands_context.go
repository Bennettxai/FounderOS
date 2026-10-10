package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// ContextBundle contains all loaded context data for a command.
type ContextBundle struct {
	Documents     []ContextDocument     `json:"documents"`
	Conversations []ContextConversation `json:"conversations"`
	Artifacts     []ContextArtifact     `json:"artifacts"`
	Projects      []ContextProject      `json:"projects"`
	Clients       []ContextClient       `json:"clients"`
	Tasks         []ContextTask         `json:"tasks"`
}

// ContextDocument is a document included in the command context bundle.
type ContextDocument struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

// ContextConversation is a conversation summary included in the context bundle.
type ContextConversation struct {
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Messages int    `json:"messages"`
}

// ContextArtifact is an artifact included in the context bundle.
type ContextArtifact struct {
	Title   string `json:"title"`
	Type    string `json:"type"`
	Summary string `json:"summary"`
	Content string `json:"content"`
}

// ContextProject is a project summary included in the context bundle.
type ContextProject struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// ContextClient is a client summary included in the context bundle.
type ContextClient struct {
	Name     string `json:"name"`
	Company  string `json:"company"`
	Status   string `json:"status"`
	Industry string `json:"industry"`
}

// ContextTask is a task included in the context bundle.
type ContextTask struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
}

// loadContextBundle loads relevant context data for a command based on the
// command's declared context sources. It is a package-level function so it can
// be called from both CommandHandler and ChatHandler without coupling them.
func loadContextBundle(
	ctx context.Context,
	queries *sqlc.Queries,
	userID string,
	contextID *uuid.UUID,
	projectID *uuid.UUID,
	sources []string,
) *ContextBundle {
	bundle := &ContextBundle{}

	for _, source := range sources {
		switch source {
		case "documents":
			if contextID != nil {
				contextDoc, err := queries.GetContext(ctx, sqlc.GetContextParams{
					ID:     pgtype.UUID{Bytes: *contextID, Valid: true},
					UserID: userID,
				})
				if err == nil && contextDoc.Name != "" {
					content := extractBlocksContent(contextDoc.Blocks)
					if content != "" {
						docType := "document"
						if contextDoc.Type.Valid {
							docType = string(contextDoc.Type.Contexttype)
						}
						bundle.Documents = append(bundle.Documents, ContextDocument{
							Title:   contextDoc.Name,
							Content: content,
							Type:    docType,
						})
					}
				}
			}

		case "conversations":
			if contextID != nil {
				convs, err := queries.ListConversationsByContext(ctx, sqlc.ListConversationsByContextParams{
					UserID:    userID,
					ContextID: pgtype.UUID{Bytes: *contextID, Valid: true},
				})
				if err == nil {
					limit := len(convs)
					if limit > 5 {
						limit = 5
					}
					for i := 0; i < limit; i++ {
						conv := convs[i]
						title := "Untitled"
						if conv.Title != nil {
							title = *conv.Title
						}
						bundle.Conversations = append(bundle.Conversations, ContextConversation{
							Title:    title,
							Messages: int(conv.MessageCount),
						})
					}
				}
			}

		case "artifacts":
			if contextID != nil {
				artifacts, err := queries.ListArtifacts(ctx, sqlc.ListArtifactsParams{
					UserID:    userID,
					ContextID: pgtype.UUID{Bytes: *contextID, Valid: true},
				})
				if err == nil {
					limit := len(artifacts)
					if limit > 10 {
						limit = 10
					}
					for i := 0; i < limit; i++ {
						a := artifacts[i]
						summary := ""
						if a.Summary != nil {
							summary = *a.Summary
						}
						bundle.Artifacts = append(bundle.Artifacts, ContextArtifact{
							Title:   a.Title,
							Type:    string(a.Type),
							Summary: summary,
							Content: truncateString(a.Content, 2000),
						})
					}
				}
			}

		case "projects":
			if projectID != nil {
				project, err := queries.GetProject(ctx, sqlc.GetProjectParams{
					ID:     pgtype.UUID{Bytes: *projectID, Valid: true},
					UserID: userID,
				})
				if err == nil {
					desc := ""
					if project.Description != nil {
						desc = *project.Description
					}
					status := "active"
					if project.Status.Valid {
						status = string(project.Status.Projectstatus)
					}
					bundle.Projects = append(bundle.Projects, ContextProject{
						Name:        project.Name,
						Description: desc,
						Status:      status,
					})
				}
			}

		case "clients":
			if contextID != nil {
				contextDoc, err := queries.GetContext(ctx, sqlc.GetContextParams{
					ID:     pgtype.UUID{Bytes: *contextID, Valid: true},
					UserID: userID,
				})
				if err == nil && contextDoc.ClientID.Valid {
					client, err := queries.GetClient(ctx, sqlc.GetClientParams{
						ID:     contextDoc.ClientID,
						UserID: userID,
					})
					if err == nil {
						industry := ""
						if client.Industry != nil {
							industry = *client.Industry
						}
						status := "unknown"
						if client.Status.Valid {
							status = string(client.Status.Clientstatus)
						}
						clientType := "company"
						if client.Type.Valid {
							clientType = string(client.Type.Clienttype)
						}
						bundle.Clients = append(bundle.Clients, ContextClient{
							Name:     client.Name,
							Company:  clientType,
							Status:   status,
							Industry: industry,
						})
					}
				}
			}

		case "tasks":
			if projectID != nil {
				tasks, err := queries.ListTasks(ctx, sqlc.ListTasksParams{
					UserID:    userID,
					ProjectID: pgtype.UUID{Bytes: *projectID, Valid: true},
				})
				if err == nil {
					limit := len(tasks)
					if limit > 20 {
						limit = 20
					}
					for i := 0; i < limit; i++ {
						t := tasks[i]
						desc := ""
						if t.Description != nil {
							desc = *t.Description
						}
						status := "pending"
						if t.Status.Valid {
							status = string(t.Status.Taskstatus)
						}
						priority := "medium"
						if t.Priority.Valid {
							priority = string(t.Priority.Taskpriority)
						}
						bundle.Tasks = append(bundle.Tasks, ContextTask{
							Title:       t.Title,
							Description: desc,
							Priority:    priority,
							Status:      status,
						})
					}
				}
			} else {
				tasks, err := queries.ListTasks(ctx, sqlc.ListTasksParams{
					UserID: userID,
				})
				if err == nil {
					limit := len(tasks)
					if limit > 10 {
						limit = 10
					}
					for i := 0; i < limit; i++ {
						t := tasks[i]
						desc := ""
						if t.Description != nil {
							desc = *t.Description
						}
						status := "pending"
						if t.Status.Valid {
							status = string(t.Status.Taskstatus)
						}
						priority := "medium"
						if t.Priority.Valid {
							priority = string(t.Priority.Taskpriority)
						}
						bundle.Tasks = append(bundle.Tasks, ContextTask{
							Title:       t.Title,
							Description: desc,
							Priority:    priority,
							Status:      status,
						})
					}
				}
			}
		}
	}

	return bundle
}

// buildCommandPrompt creates an enhanced prompt combining the user message with
// all loaded context sections.
func buildCommandPrompt(cmdInfo CommandInfo, userMessage string, bundle *ContextBundle) string {
	var sb strings.Builder

	sb.WriteString("USER REQUEST:\n")
	sb.WriteString(userMessage)
	sb.WriteString("\n\n")

	if len(bundle.Documents) > 0 {
		sb.WriteString("=== RELEVANT DOCUMENTS ===\n")
		for _, doc := range bundle.Documents {
			sb.WriteString(fmt.Sprintf("\n[%s: %s]\n", doc.Type, doc.Title))
			sb.WriteString(truncateString(doc.Content, 4000))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(bundle.Artifacts) > 0 {
		sb.WriteString("=== RELATED ARTIFACTS ===\n")
		for _, a := range bundle.Artifacts {
			sb.WriteString(fmt.Sprintf("\n[%s: %s]\n", a.Type, a.Title))
			if a.Summary != "" {
				sb.WriteString(fmt.Sprintf("Summary: %s\n", a.Summary))
			}
			if a.Content != "" {
				sb.WriteString(truncateString(a.Content, 1000))
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(bundle.Projects) > 0 {
		sb.WriteString("=== PROJECT CONTEXT ===\n")
		for _, p := range bundle.Projects {
			sb.WriteString(fmt.Sprintf("Project: %s\n", p.Name))
			sb.WriteString(fmt.Sprintf("Status: %s\n", p.Status))
			if p.Description != "" {
				sb.WriteString(fmt.Sprintf("Description: %s\n", p.Description))
			}
		}
		sb.WriteString("\n")
	}

	if len(bundle.Clients) > 0 {
		sb.WriteString("=== CLIENT CONTEXT ===\n")
		for _, cl := range bundle.Clients {
			sb.WriteString(fmt.Sprintf("Client: %s\n", cl.Name))
			if cl.Company != "" {
				sb.WriteString(fmt.Sprintf("Company: %s\n", cl.Company))
			}
			sb.WriteString(fmt.Sprintf("Status: %s\n", cl.Status))
			if cl.Industry != "" {
				sb.WriteString(fmt.Sprintf("Industry: %s\n", cl.Industry))
			}
		}
		sb.WriteString("\n")
	}

	if len(bundle.Tasks) > 0 {
		sb.WriteString("=== TASKS ===\n")
		for _, t := range bundle.Tasks {
			sb.WriteString(fmt.Sprintf("- [%s] %s (%s priority)\n", t.Status, t.Title, t.Priority))
		}
		sb.WriteString("\n")
	}

	if len(bundle.Conversations) > 0 {
		sb.WriteString("=== RECENT CONVERSATIONS ===\n")
		for _, conv := range bundle.Conversations {
			sb.WriteString(fmt.Sprintf("- %s (%d messages)\n", conv.Title, conv.Messages))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// truncateString returns s truncated to maxLen characters, appending "..." when
// truncation occurs.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractBlocksContent does a best-effort plain-text extraction from a Tiptap
// JSON blocks value. A proper parser should replace this in production.
func extractBlocksContent(blocksJSON []byte) string {
	content := string(blocksJSON)
	if content == "null" || content == "[]" || content == "" {
		return ""
	}
	content = strings.ReplaceAll(content, `"type":"paragraph"`, "")
	content = strings.ReplaceAll(content, `"content":[`, "")
	content = strings.ReplaceAll(content, `"text":"`, " ")
	content = strings.ReplaceAll(content, `"}]`, "")
	content = strings.ReplaceAll(content, `[{`, "")
	content = strings.ReplaceAll(content, `}]`, "")
	content = strings.ReplaceAll(content, `},{`, " ")
	content = strings.ReplaceAll(content, `"`, "")
	return strings.TrimSpace(content)
}
