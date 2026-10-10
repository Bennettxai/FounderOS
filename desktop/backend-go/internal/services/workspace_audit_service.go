package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkspaceAuditService handles audit logging for workspace activities
type WorkspaceAuditService struct {
	pool *pgxpool.Pool
}

// AuditLog represents a workspace audit log entry
type AuditLog struct {
	ID           uuid.UUID              `json:"id"`
	WorkspaceID  uuid.UUID              `json:"workspace_id"`
	UserID       string                 `json:"user_id"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   *string                `json:"resource_id,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	IPAddress    *string                `json:"ip_address,omitempty"`
	UserAgent    *string                `json:"user_agent,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

// AuditLogFilter represents filters for querying audit logs
type AuditLogFilter struct {
	WorkspaceID  uuid.UUID
	UserID       *string
	Action       *string
	ResourceType *string
	ResourceID   *string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}

// NewWorkspaceAuditService creates a new workspace audit service
func NewWorkspaceAuditService(pool *pgxpool.Pool) *WorkspaceAuditService {
	return &WorkspaceAuditService{pool: pool}
}

// LogAction creates a new audit log entry
func (s *WorkspaceAuditService) LogAction(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID string,
	action string,
	resourceType string,
	resourceID *string,
	details map[string]interface{},
	ipAddress *string,
	userAgent *string,
) (*AuditLog, error) {
	log := &AuditLog{}

	var detailsJSON []byte
	var err error
	if details != nil {
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return nil, fmt.Errorf("marshal details: %w", err)
		}
	}

	err = s.pool.QueryRow(ctx, `
		INSERT INTO workspace_audit_logs (workspace_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, workspace_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at
	`, workspaceID, userID, action, resourceType, resourceID, detailsJSON, ipAddress, userAgent).Scan(
		&log.ID,
		&log.WorkspaceID,
		&log.UserID,
		&log.Action,
		&log.ResourceType,
		&log.ResourceID,
		&log.Details,
		&log.IPAddress,
		&log.UserAgent,
		&log.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("create audit log: %w", err)
	}

	return log, nil
}

// GetLogs retrieves audit logs with filters
func (s *WorkspaceAuditService) GetLogs(ctx context.Context, filter AuditLogFilter) ([]AuditLog, error) {
	query := `
		SELECT id, workspace_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at
		FROM workspace_audit_logs
		WHERE workspace_id = $1
	`
	args := []interface{}{filter.WorkspaceID}
	argCount := 1

	if filter.UserID != nil {
		argCount++
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
	}

	if filter.Action != nil {
		argCount++
		query += fmt.Sprintf(" AND action = $%d", argCount)
		args = append(args, *filter.Action)
	}

	if filter.ResourceType != nil {
		argCount++
		query += fmt.Sprintf(" AND resource_type = $%d", argCount)
		args = append(args, *filter.ResourceType)
	}

	if filter.ResourceID != nil {
		argCount++
		query += fmt.Sprintf(" AND resource_id = $%d", argCount)
		args = append(args, *filter.ResourceID)
	}

	if filter.StartDate != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.StartDate)
	}

	if filter.EndDate != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.EndDate)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	} else {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, 100) // Default limit
	}

	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.WorkspaceID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&detailsJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, fmt.Errorf("unmarshal details: %w", err)
			}
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// GetLogByID retrieves a specific audit log by ID
func (s *WorkspaceAuditService) GetLogByID(ctx context.Context, logID uuid.UUID) (*AuditLog, error) {
	log := &AuditLog{}
	var detailsJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, workspace_id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, created_at
		FROM workspace_audit_logs
		WHERE id = $1
	`, logID).Scan(
		&log.ID,
		&log.WorkspaceID,
		&log.UserID,
		&log.Action,
		&log.ResourceType,
		&log.ResourceID,
		&detailsJSON,
		&log.IPAddress,
		&log.UserAgent,
		&log.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}

	if len(detailsJSON) > 0 {
		if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
			return nil, fmt.Errorf("unmarshal details: %w", err)
		}
	}

	return log, nil
}

// GetUserActivity retrieves recent activity for a specific user
func (s *WorkspaceAuditService) GetUserActivity(ctx context.Context, workspaceID uuid.UUID, userID string, limit int) ([]AuditLog, error) {
	filter := AuditLogFilter{
		WorkspaceID: workspaceID,
		UserID:      &userID,
		Limit:       limit,
	}
	return s.GetLogs(ctx, filter)
}

// GetResourceHistory retrieves history for a specific resource
func (s *WorkspaceAuditService) GetResourceHistory(ctx context.Context, workspaceID uuid.UUID, resourceType string, resourceID string, limit int) ([]AuditLog, error) {
	filter := AuditLogFilter{
		WorkspaceID:  workspaceID,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Limit:        limit,
	}
	return s.GetLogs(ctx, filter)
}

// GetActionCount returns count of actions by type within a time period
func (s *WorkspaceAuditService) GetActionCount(ctx context.Context, workspaceID uuid.UUID, startDate, endDate time.Time) (map[string]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT action, COUNT(*) as count
		FROM workspace_audit_logs
		WHERE workspace_id = $1
		AND created_at >= $2
		AND created_at <= $3
		GROUP BY action
		ORDER BY count DESC
	`, workspaceID, startDate, endDate)

	if err != nil {
		return nil, fmt.Errorf("query action counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return nil, fmt.Errorf("scan action count: %w", err)
		}
		counts[action] = count
	}

	return counts, nil
}

// GetMostActiveUsers returns users with most actions within a time period
func (s *WorkspaceAuditService) GetMostActiveUsers(ctx context.Context, workspaceID uuid.UUID, startDate, endDate time.Time, limit int) ([]struct {
	UserID string `json:"user_id"`
	Count  int    `json:"count"`
}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT user_id, COUNT(*) as count
		FROM workspace_audit_logs
		WHERE workspace_id = $1
		AND created_at >= $2
		AND created_at <= $3
		GROUP BY user_id
		ORDER BY count DESC
		LIMIT $4
	`, workspaceID, startDate, endDate, limit)

	if err != nil {
		return nil, fmt.Errorf("query active users: %w", err)
	}
	defer rows.Close()

	var users []struct {
		UserID string `json:"user_id"`
		Count  int    `json:"count"`
	}

	for rows.Next() {
		var user struct {
			UserID string `json:"user_id"`
			Count  int    `json:"count"`
		}
		if err := rows.Scan(&user.UserID, &user.Count); err != nil {
			return nil, fmt.Errorf("scan user activity: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}
