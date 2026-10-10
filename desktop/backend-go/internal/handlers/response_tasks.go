package handlers

import (
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// Task response transformation
type TaskResponse struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Status      string  `json:"status"`
	Priority    string  `json:"priority"`
	DueDate     *string `json:"due_date"`
	CompletedAt *string `json:"completed_at"`
	ProjectID   *string `json:"project_id"`
	AssigneeID  *string `json:"assignee_id"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func TransformTask(t sqlc.Task) TaskResponse {
	status := "todo"
	if t.Status.Valid {
		status = string(t.Status.Taskstatus)
	}

	priority := "medium"
	if t.Priority.Valid {
		priority = string(t.Priority.Taskpriority)
	}

	return TaskResponse{
		ID:          pgtypeUUIDToStringRequired(t.ID),
		UserID:      t.UserID,
		Title:       t.Title,
		Description: t.Description,
		Status:      status,
		Priority:    priority,
		DueDate:     pgtypeTimestampToString(t.DueDate),
		CompletedAt: pgtypeTimestampToString(t.CompletedAt),
		ProjectID:   pgtypeUUIDToString(t.ProjectID),
		AssigneeID:  pgtypeUUIDToString(t.AssigneeID),
		CreatedAt:   t.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformTasks(tasks []sqlc.Task) []TaskResponse {
	result := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = TransformTask(t)
	}
	return result
}

// FocusItem response transformation
type FocusItemResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
	FocusDate string `json:"focus_date"`
	CreatedAt string `json:"created_at"`
}

func TransformFocusItem(f sqlc.FocusItem) FocusItemResponse {
	completed := false
	if f.Completed != nil {
		completed = *f.Completed
	}

	return FocusItemResponse{
		ID:        pgtypeUUIDToStringRequired(f.ID),
		UserID:    f.UserID,
		Text:      f.Text,
		Completed: completed,
		FocusDate: f.FocusDate.Time.Format("2006-01-02"),
		CreatedAt: f.CreatedAt.Time.Format(time.RFC3339),
	}
}

func TransformFocusItems(items []sqlc.FocusItem) []FocusItemResponse {
	result := make([]FocusItemResponse, len(items))
	for i, f := range items {
		result[i] = TransformFocusItem(f)
	}
	return result
}

// Artifact response transformation
type ArtifactResponse struct {
	ID             string  `json:"id"`
	UserID         string  `json:"user_id"`
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	Type           string  `json:"type"`
	Language       *string `json:"language"`
	Summary        *string `json:"summary"`
	ConversationID *string `json:"conversation_id"`
	MessageID      *string `json:"message_id"`
	ProjectID      *string `json:"project_id"`
	ContextID      *string `json:"context_id"`
	ContextName    *string `json:"context_name"`
	Version        int32   `json:"version"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func TransformArtifact(a sqlc.Artifact) ArtifactResponse {
	// ArtifactType is not nullable in the model
	artifactType := strings.ToLower(string(a.Type))

	version := int32(1)
	if a.Version != nil {
		version = *a.Version
	}

	return ArtifactResponse{
		ID:             pgtypeUUIDToStringRequired(a.ID),
		UserID:         a.UserID,
		Title:          a.Title,
		Content:        a.Content,
		Type:           artifactType,
		Language:       a.Language,
		Summary:        a.Summary,
		ConversationID: pgtypeUUIDToString(a.ConversationID),
		MessageID:      pgtypeUUIDToString(a.MessageID),
		ProjectID:      pgtypeUUIDToString(a.ProjectID),
		ContextID:      pgtypeUUIDToString(a.ContextID),
		ContextName:    nil, // Populated separately if needed
		Version:        version,
		CreatedAt:      a.CreatedAt.Time.Format(time.RFC3339),
		UpdatedAt:      a.UpdatedAt.Time.Format(time.RFC3339),
	}
}

func TransformArtifacts(artifacts []sqlc.Artifact) []ArtifactResponse {
	result := make([]ArtifactResponse, len(artifacts))
	for i, a := range artifacts {
		result[i] = TransformArtifact(a)
	}
	return result
}
