package handlers

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// stringToProjectStatus converts a string to sqlc.Projectstatus.
func stringToProjectStatus(s string) sqlc.Projectstatus {
	typeMap := map[string]sqlc.Projectstatus{
		"active":    sqlc.ProjectstatusACTIVE,
		"paused":    sqlc.ProjectstatusPAUSED,
		"completed": sqlc.ProjectstatusCOMPLETED,
		"archived":  sqlc.ProjectstatusARCHIVED,
	}
	if enum, ok := typeMap[strings.ToLower(s)]; ok {
		return enum
	}
	return sqlc.ProjectstatusACTIVE
}

// stringToProjectPriority converts a string to sqlc.Projectpriority.
func stringToProjectPriority(p string) sqlc.Projectpriority {
	typeMap := map[string]sqlc.Projectpriority{
		"critical": sqlc.ProjectpriorityCRITICAL,
		"high":     sqlc.ProjectpriorityHIGH,
		"medium":   sqlc.ProjectpriorityMEDIUM,
		"low":      sqlc.ProjectpriorityLOW,
	}
	if enum, ok := typeMap[strings.ToLower(p)]; ok {
		return enum
	}
	return sqlc.ProjectpriorityMEDIUM
}

// TransformProjectRows transforms ListProjectsRow to a clean JSON response.
func TransformProjectRows(rows []sqlc.ListProjectsRow) []map[string]interface{} {
	result := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		result[i] = map[string]interface{}{
			"id":                  projectUUIDToString(row.ID),
			"user_id":             row.UserID,
			"name":                row.Name,
			"description":         row.Description,
			// Enums are stored UPPERCASE (ACTIVE/MEDIUM) but the frontend's
			// Project type expects lowercase ('active'|'paused'|...). Return the
			// lowercase form so the kanban columns match — otherwise every project
			// lands in no column and the board renders empty.
			"status":              strings.ToLower(string(row.Status.Projectstatus)),
			"priority":            strings.ToLower(string(row.Priority.Projectpriority)),
			"client_name":         row.ClientName,
			"client_id":           projectUUIDToString(row.ClientID),
			"client_company_name": row.ClientCompanyName,
			"project_type":        row.ProjectType,
			"project_metadata":    row.ProjectMetadata,
			"start_date":          dateToString(row.StartDate),
			"due_date":            dateToString(row.DueDate),
			"completed_at":        projectTimestamptzToString(row.CompletedAt),
			"visibility":          row.Visibility,
			"owner_id":            row.OwnerID,
			"created_at":          projectTimestampToString(row.CreatedAt),
			"updated_at":          projectTimestampToString(row.UpdatedAt),
		}
	}
	return result
}

// Helper functions for type conversion (project-specific).
func projectUUIDToString(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes).String()
	return &id
}

func dateToString(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}

func projectTimestampToString(t pgtype.Timestamp) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}

func projectTimestamptzToString(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format(time.RFC3339)
	return &s
}
