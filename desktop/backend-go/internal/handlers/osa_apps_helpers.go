package handlers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"

	"github.com/google/uuid"
)

// mapStatusForFrontend converts backend status strings to frontend-expected values.
// Frontend expects: pending, starting, in_progress, completed, failed
func mapStatusForFrontend(status string) string {
	switch status {
	case "started", "starting":
		return "starting"
	case "planning", "generating", "running", "processing", "in_progress":
		return "in_progress"
	case "completed", "done", "success":
		return "completed"
	case "failed", "error":
		return "failed"
	case "pending", "queued", "waiting":
		return "pending"
	default:
		// Default to in_progress for unknown statuses
		return "in_progress"
	}
}

func parseIntParam(s string) (int32, error) {
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return 0, err
	}
	return int32(val), nil
}

func convertToAppDetail(row sqlc.ListOSAModuleInstancesByUserRow) AppDetail {
	return AppDetail{
		ID:               row.ID.Bytes,
		WorkspaceID:      row.WorkspaceID.Bytes,
		ModuleID:         uuidPtrFromPgtype(row.ModuleID),
		Name:             row.Name,
		DisplayName:      row.DisplayName,
		Description:      row.Description,
		OSAWorkflowID:    row.OsaWorkflowID,
		OSASandboxID:     row.OsaSandboxID,
		CodeRepository:   row.CodeRepository,
		DeploymentURL:    row.DeploymentUrl,
		Status:           row.Status,
		FilesCreated:     row.FilesCreated,
		TestsPassed:      row.TestsPassed,
		BuildStatus:      row.BuildStatus,
		CreatedAt:        row.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		GeneratedAt:      timestampPtrToString(row.GeneratedAt),
		DeployedAt:       timestampPtrToString(row.DeployedAt),
		LastBuildAt:      timestampPtrToString(row.LastBuildAt),
		BuildEventsCount: row.BuildEventsCount,
	}
}

func convertToAppDetailFromRow(row sqlc.GetOSAModuleInstanceByIDWithAuthRow) AppDetail {
	return AppDetail{
		ID:             row.ID.Bytes,
		WorkspaceID:    row.WorkspaceID.Bytes,
		ModuleID:       uuidPtrFromPgtype(row.ModuleID),
		Name:           row.Name,
		DisplayName:    row.DisplayName,
		Description:    row.Description,
		OSAWorkflowID:  row.OsaWorkflowID,
		OSASandboxID:   row.OsaSandboxID,
		CodeRepository: row.CodeRepository,
		DeploymentURL:  row.DeploymentUrl,
		Status:         row.Status,
		FilesCreated:   row.FilesCreated,
		TestsPassed:    row.TestsPassed,
		BuildStatus:    row.BuildStatus,
		CreatedAt:      row.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      row.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		GeneratedAt:    timestampPtrToString(row.GeneratedAt),
		DeployedAt:     timestampPtrToString(row.DeployedAt),
		LastBuildAt:    timestampPtrToString(row.LastBuildAt),
	}
}

func convertToAppDetailFromGeneratedApp(app sqlc.OsaModuleInstance) AppDetail {
	return AppDetail{
		ID:             app.ID.Bytes,
		WorkspaceID:    app.WorkspaceID.Bytes,
		ModuleID:       uuidPtrFromPgtype(app.ModuleID),
		Name:           app.Name,
		DisplayName:    app.DisplayName,
		Description:    app.Description,
		OSAWorkflowID:  app.OsaWorkflowID,
		OSASandboxID:   app.OsaSandboxID,
		CodeRepository: app.CodeRepository,
		DeploymentURL:  app.DeploymentUrl,
		Status:         app.Status,
		FilesCreated:   app.FilesCreated,
		TestsPassed:    app.TestsPassed,
		BuildStatus:    app.BuildStatus,
		CreatedAt:      app.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      app.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		GeneratedAt:    timestampPtrToString(app.GeneratedAt),
		DeployedAt:     timestampPtrToString(app.DeployedAt),
		LastBuildAt:    timestampPtrToString(app.LastBuildAt),
	}
}

func uuidPtrFromPgtype(u pgtype.UUID) *uuid.UUID {
	if !u.Valid {
		return nil
	}
	id := uuid.UUID(u.Bytes)
	return &id
}

func timestampPtrToString(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	s := t.Time.Format("2006-01-02T15:04:05Z")
	return &s
}

// isValidPath validates path parameters to prevent directory traversal attacks
func isValidPath(path string) bool {
	// URL decode first to catch encoded attacks
	decoded, err := url.QueryUnescape(path)
	if err != nil {
		// If we can't decode it, reject it
		return false
	}

	// Reject backslashes (Windows path separator used for traversal)
	if strings.Contains(path, "\\") || strings.Contains(decoded, "\\") {
		return false
	}

	// Prevent directory traversal (check both original and decoded)
	if strings.Contains(path, "..") || strings.Contains(decoded, "..") {
		return false
	}

	// Prevent null bytes
	if strings.Contains(path, "\x00") || strings.Contains(decoded, "\x00") {
		return false
	}

	// Prevent absolute paths (except single "/" which means root)
	if strings.HasPrefix(path, "/") && len(path) > 1 && path != "/" {
		// Allow paths like "/src" but not "//src" or "/../../etc"
		if strings.HasPrefix(path, "//") {
			return false
		}
	}

	// Same check for decoded version
	if strings.HasPrefix(decoded, "/") && len(decoded) > 1 && decoded != "/" {
		if strings.HasPrefix(decoded, "//") {
			return false
		}
	}

	return true
}

// formatTimestamp converts pgtype.Timestamptz to ISO 8601 string pointer
func formatTimestamp(t pgtype.Timestamptz) *string {
	if !t.Valid {
		return nil
	}
	formatted := t.Time.Format("2006-01-02T15:04:05Z")
	return &formatted
}

// getInstallationStatus safely extracts installation status from nullable field
func getInstallationStatus(status *string) string {
	if status == nil {
		return "pending"
	}
	return *status
}
