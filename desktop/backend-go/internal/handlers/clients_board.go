package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// GetClientBoard returns the aggregated "client board" for a single client:
// the client record plus its projects, tasks (transitive via projects), team
// (distinct people on this client's projects), deals, interactions, and counts.
//
// One endpoint instead of frontend fan-out because tasks/team require
// project-ID resolution first (N+1 from the browser otherwise); each section
// is a single SQL round-trip server-side.
func (h *ClientHandler) GetClientBoard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "client_id")
		return
	}

	pgID := pgtype.UUID{Bytes: id, Valid: true}
	queries := sqlc.New(h.pool)

	// Verify client access (owner or active workspace member, same pattern as
	// the sibling client endpoints)
	client, err := h.getClientShared(c, pgID, user.ID)
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Client")
		return
	}

	ctx := c.Request.Context()

	projects, err := h.clientBoardProjects(ctx, id, user.ID)
	if err != nil {
		slog.Error("failed to load client board projects", "client_id", id, "error", err)
		utils.RespondInternalError(c, slog.Default(), "load client projects", nil)
		return
	}

	tasks, err := h.clientBoardTasks(ctx, id, user.ID)
	if err != nil {
		slog.Error("failed to load client board tasks", "client_id", id, "error", err)
		utils.RespondInternalError(c, slog.Default(), "load client tasks", nil)
		return
	}

	team, err := h.clientBoardTeam(ctx, id, user.ID)
	if err != nil {
		slog.Error("failed to load client board team", "client_id", id, "error", err)
		utils.RespondInternalError(c, slog.Default(), "load client team", nil)
		return
	}

	// Deals and interactions reuse the existing per-client queries.
	// Errors are non-fatal (mirrors GetClient): warn and return empty.
	deals, err := queries.ListClientDeals(ctx, pgID)
	if err != nil {
		slog.Warn("failed to load client deals for board", "client_id", id, "error", err)
		deals = nil
	}
	interactions, err := queries.ListClientInteractions(ctx, pgID)
	if err != nil {
		slog.Warn("failed to load client interactions for board", "client_id", id, "error", err)
		interactions = nil
	}

	tasksOpen, tasksDone := clientBoardTaskCounts(tasks)

	c.JSON(http.StatusOK, gin.H{
		"client":       TransformClient(client),
		"projects":     projects,
		"tasks":        tasks,
		"team":         team,
		"deals":        transformClientDealsFromRows(deals),
		"interactions": TransformInteractions(interactions),
		"counts": gin.H{
			"projects":   len(projects),
			"tasks_open": tasksOpen,
			"tasks_done": tasksDone,
			"team":       len(team),
			"deals":      len(deals),
		},
	})
}

// clientBoardProjects returns this client's projects, gated exactly like
// ListProjects (owner OR active workspace member). `progress` is computed
// from real task completion (done tasks / total tasks); 0 when no tasks.
func (h *ClientHandler) clientBoardProjects(ctx context.Context, clientID uuid.UUID, userID string) ([]gin.H, error) {
	const q = `
		SELECT p.id, p.name, p.status::text, p.priority::text, p.due_date, p.updated_at,
		       COALESCE(
		           ROUND(100.0 * COUNT(t.id) FILTER (WHERE t.status::text IN ('done', 'completed'))::numeric
		                 / NULLIF(COUNT(t.id), 0))::int,
		           0) AS progress
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.client_id = $1
		  AND (p.user_id = $2 OR p.workspace_id IN (
		        SELECT workspace_id FROM workspace_members WHERE user_id = $2 AND status = 'active'))
		GROUP BY p.id, p.name, p.status, p.priority, p.due_date, p.updated_at
		ORDER BY p.updated_at DESC`

	rows, err := h.pool.Query(ctx, q, clientID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]gin.H, 0)
	for rows.Next() {
		var (
			id        pgtype.UUID
			name      string
			status    pgtype.Text
			priority  pgtype.Text
			dueDate   pgtype.Date
			updatedAt pgtype.Timestamp
			progress  int32
		)
		if err := rows.Scan(&id, &name, &status, &priority, &dueDate, &updatedAt, &progress); err != nil {
			return nil, err
		}
		projects = append(projects, gin.H{
			"id":         projectUUIDToString(id),
			"name":       name,
			"status":     boardTextPtr(status),
			"priority":   boardTextPtr(priority),
			"due_date":   dateToString(dueDate),
			"progress":   progress,
			"updated_at": projectTimestampToString(updatedAt),
		})
	}
	return projects, rows.Err()
}

// clientBoardTasks returns tasks belonging to this client's projects
// (transitive linkage: task -> project -> client), same project gating as
// ListProjects. Tasks without a project are not attributable to a client.
func (h *ClientHandler) clientBoardTasks(ctx context.Context, clientID uuid.UUID, userID string) ([]gin.H, error) {
	const q = `
		SELECT t.id, t.title, t.status::text, t.priority::text, t.due_date, t.project_id, t.assignee_id
		FROM tasks t
		WHERE t.project_id IN (
		    SELECT p.id FROM projects p
		    WHERE p.client_id = $1
		      AND (p.user_id = $2 OR p.workspace_id IN (
		            SELECT workspace_id FROM workspace_members WHERE user_id = $2 AND status = 'active')))
		ORDER BY t.status, t.priority`

	rows, err := h.pool.Query(ctx, q, clientID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]gin.H, 0)
	for rows.Next() {
		var (
			id         pgtype.UUID
			title      string
			status     pgtype.Text
			priority   pgtype.Text
			dueDate    pgtype.Timestamp
			projectID  pgtype.UUID
			assigneeID pgtype.UUID
		)
		if err := rows.Scan(&id, &title, &status, &priority, &dueDate, &projectID, &assigneeID); err != nil {
			return nil, err
		}
		tasks = append(tasks, gin.H{
			"id":          projectUUIDToString(id),
			"title":       title,
			"status":      boardTextPtr(status),
			"priority":    boardTextPtr(priority),
			"due_date":    projectTimestampToString(dueDate),
			"project_id":  projectUUIDToString(projectID),
			"assignee_id": projectUUIDToString(assigneeID),
		})
	}
	return tasks, rows.Err()
}

// clientBoardTeam returns the distinct people assigned (via project_members)
// to this client's projects. Name/email prefer the team_members record and
// fall back to the "user" record; role prefers the team member's job role and
// falls back to their project role.
func (h *ClientHandler) clientBoardTeam(ctx context.Context, clientID uuid.UUID, userID string) ([]gin.H, error) {
	const q = `
		SELECT DISTINCT ON (pm.user_id, pm.team_member_id)
		       pm.team_member_id, pm.user_id,
		       COALESCE(tm.name, u.name) AS name,
		       COALESCE(tm.email, u.email) AS email,
		       COALESCE(tm.role, pm.role::text) AS role
		FROM project_members pm
		JOIN projects p ON pm.project_id = p.id
		 AND p.client_id = $1
		 AND (p.user_id = $2 OR p.workspace_id IN (
		       SELECT workspace_id FROM workspace_members WHERE user_id = $2 AND status = 'active'))
		LEFT JOIN team_members tm ON pm.team_member_id = tm.id
		LEFT JOIN "user" u ON pm.user_id = u.id
		ORDER BY pm.user_id, pm.team_member_id`

	rows, err := h.pool.Query(ctx, q, clientID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	team := make([]gin.H, 0)
	for rows.Next() {
		var (
			teamMemberID pgtype.UUID
			memberUserID string
			name         pgtype.Text
			email        pgtype.Text
			role         pgtype.Text
		)
		if err := rows.Scan(&teamMemberID, &memberUserID, &name, &email, &role); err != nil {
			return nil, err
		}
		team = append(team, gin.H{
			"team_member_id": projectUUIDToString(teamMemberID),
			"user_id":        memberUserID,
			"name":           boardTextPtr(name),
			"email":          boardTextPtr(email),
			"role":           boardTextPtr(role),
		})
	}
	return team, rows.Err()
}

// clientBoardTaskCounts derives open/done counts from the board's task rows.
// Open = status NOT IN (done, completed, cancelled). Done = done or completed.
func clientBoardTaskCounts(tasks []gin.H) (open, done int) {
	for _, t := range tasks {
		status := ""
		if s, ok := t["status"].(*string); ok && s != nil {
			status = strings.ToLower(*s)
		}
		switch status {
		case "done", "completed":
			done++
		case "cancelled":
			// neither open nor done
		default:
			open++
		}
	}
	return open, done
}

// boardTextPtr converts a nullable pgtype.Text to *string (nil when NULL).
func boardTextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}
