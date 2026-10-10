package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/cache"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// TeamHandler handles all team member HTTP requests.
type TeamHandler struct {
	EngineSyncHook
	pool       *pgxpool.Pool
	queryCache *cache.QueryCache
}

// NewTeamHandler constructs a TeamHandler.
func NewTeamHandler(pool *pgxpool.Pool, queryCache *cache.QueryCache) *TeamHandler {
	return &TeamHandler{pool: pool, queryCache: queryCache}
}

// RegisterTeamRoutes mounts team member endpoints under /api/team.
func RegisterTeamRoutes(api *gin.RouterGroup, h *TeamHandler, auth gin.HandlerFunc) {
	team := api.Group("/team")
	team.Use(auth, middleware.RequireAuth())
	{
		team.GET("", h.ListTeamMembers)
		team.POST("", h.CreateTeamMember)
		team.GET("/:id", h.GetTeamMember)
		team.PUT("/:id", h.UpdateTeamMember)
		team.PATCH("/:id/status", h.UpdateTeamMemberStatus)
		team.PATCH("/:id/capacity", h.UpdateTeamMemberCapacity)
		team.POST("/:id/activity", h.AddTeamMemberActivity)
		team.DELETE("/:id", h.DeleteTeamMember)
	}
}

// listWorkspaceTeamMembersSQL mirrors the sqlc ListTeamMembers query (same
// columns, same order) but scopes by workspace instead of owning user, so
// every ACTIVE member of the workspace sees the full shared roster.
const listWorkspaceTeamMembersSQL = `SELECT tm.id, tm.user_id, tm.name, tm.email, tm.role, tm.avatar_url, tm.status, tm.capacity, tm.manager_id, tm.skills, tm.hourly_rate, tm.share_calendar, tm.calendar_user_id, tm.joined_at, tm.created_at, tm.updated_at,
       (SELECT COUNT(*) FROM tasks t WHERE t.assignee_id = tm.id AND t.status != 'done') as active_task_count,
       (SELECT COUNT(*) FROM project_members pm JOIN projects p ON pm.project_id = p.id WHERE pm.team_member_id = tm.id AND p.status = 'ACTIVE') as active_project_count
FROM team_members tm
WHERE tm.workspace_id = $1
ORDER BY tm.name ASC`

// listWorkspaceTeamMembers fetches the workspace roster directly via pgx and
// scans into sqlc.ListTeamMembersRow so the existing transform is reused.
func (h *TeamHandler) listWorkspaceTeamMembers(ctx context.Context, wsID uuid.UUID) ([]sqlc.ListTeamMembersRow, error) {
	rows, err := h.pool.Query(ctx, listWorkspaceTeamMembersSQL, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []sqlc.ListTeamMembersRow{}
	for rows.Next() {
		var i sqlc.ListTeamMembersRow
		if err := rows.Scan(
			&i.ID,
			&i.UserID,
			&i.Name,
			&i.Email,
			&i.Role,
			&i.AvatarUrl,
			&i.Status,
			&i.Capacity,
			&i.ManagerID,
			&i.Skills,
			&i.HourlyRate,
			&i.ShareCalendar,
			&i.CalendarUserID,
			&i.JoinedAt,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.ActiveTaskCount,
			&i.ActiveProjectCount,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// ListTeamMembers returns the team roster.
// Workspace context (X-Workspace-ID with verified ACTIVE membership): the full
// workspace roster, fetched directly so teammates see it too, not just the
// creator. No workspace context: the caller's own members (owner path), with
// Redis caching (10-minute TTL).
func (h *TeamHandler) ListTeamMembers(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	pg := ParsePagination(c)

	var allMembers []TeamMemberListResponse

	if wsID, ok := h.teamWorkspaceScope(c, user.ID); ok {
		// Workspace path: query by workspace, uncached. The write handlers only
		// invalidate team:user:* keys, so caching here would serve stale rosters.
		members, err := h.listWorkspaceTeamMembers(c.Request.Context(), wsID)
		if err != nil {
			slog.Error("ListTeamMembers workspace error", "workspace_id", wsID, "user_id", user.ID, "error", err)
			utils.RespondInternalError(c, slog.Default(), "list team members", err)
			return
		}
		allMembers = TransformTeamMemberListRows(members)
		slog.Debug("ListTeamMembers fetched workspace roster", "workspace_id", wsID, "count", len(allMembers))
	} else {
		// Owner path (no workspace header): per-user cache + sqlc, unchanged.
		cacheKey := fmt.Sprintf("team:user:%s:all", user.ID)

		if h.queryCache != nil {
			if hit, err := h.queryCache.Get(c.Request.Context(), cacheKey, &allMembers); !hit || err != nil {
				allMembers = nil // cache miss, query below
			} else {
				slog.Debug("ListTeamMembers cache hit", "user_id", user.ID)
			}
		}

		if allMembers == nil {
			queries := sqlc.New(h.pool)
			members, err := queries.ListTeamMembers(c.Request.Context(), user.ID)
			if err != nil {
				slog.Error("ListTeamMembers error", "user_id", user.ID, "error", err)
				utils.RespondInternalError(c, slog.Default(), "list team members", err)
				return
			}
			allMembers = TransformTeamMemberListRows(members)
			slog.Debug("ListTeamMembers fetched from DB", "user_id", user.ID, "count", len(allMembers))

			if h.queryCache != nil {
				_ = h.queryCache.Set(c.Request.Context(), cacheKey, allMembers, 10*time.Minute)
			}
		}
	}

	total := int64(len(allMembers))
	start := int(pg.Offset)
	end := start + int(pg.Limit)
	if start > len(allMembers) {
		start = len(allMembers)
	}
	if end > len(allMembers) {
		end = len(allMembers)
	}

	c.JSON(http.StatusOK, NewPaginatedResponse(allMembers[start:end], total, pg))
}

// teamWorkspaceScope resolves the X-Workspace-ID header and confirms active
// membership (same pattern as the clients + calendar modules). Returns
// (workspaceID, true) when a valid workspace context is active.
func (h *TeamHandler) teamWorkspaceScope(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

// CreateTeamMember creates a new team member.
func (h *TeamHandler) CreateTeamMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req struct {
		Name       string   `json:"name" binding:"required"`
		Email      string   `json:"email" binding:"required"`
		Role       string   `json:"role" binding:"required"`
		AvatarUrl  *string  `json:"avatar_url"`
		Status     *string  `json:"status"`
		Capacity   *int32   `json:"capacity"`
		ManagerID  *string  `json:"manager_id"`
		Skills     []string `json:"skills"`
		HourlyRate *float64 `json:"hourly_rate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	var status sqlc.NullMemberstatus
	if req.Status != nil {
		status = sqlc.NullMemberstatus{
			Memberstatus: stringToMemberStatus(*req.Status),
			Valid:        true,
		}
	}

	var managerID pgtype.UUID
	if req.ManagerID != nil {
		if parsed, err := uuid.Parse(*req.ManagerID); err == nil {
			managerID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	var skills []byte
	if req.Skills != nil && len(req.Skills) > 0 {
		if skillsJSON, err := json.Marshal(req.Skills); err == nil {
			skills = skillsJSON
		}
	}

	var hourlyRate pgtype.Numeric
	if req.HourlyRate != nil {
		hourlyRate = pgtype.Numeric{Valid: true}
		hourlyRate.Scan(*req.HourlyRate)
	}

	member, err := queries.CreateTeamMember(c.Request.Context(), sqlc.CreateTeamMemberParams{
		UserID:     user.ID,
		Name:       req.Name,
		Email:      req.Email,
		Role:       req.Role,
		AvatarUrl:  req.AvatarUrl,
		Status:     status,
		Capacity:   req.Capacity,
		ManagerID:  managerID,
		Skills:     skills,
		HourlyRate: hourlyRate,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create team member", err)
		return
	}

	// Stamp the new member with the active workspace so the roster belongs to
	// one business context (see migration 128; mirrors the clients module).
	if wsID, ok := h.teamWorkspaceScope(c, user.ID); ok {
		if _, err := h.pool.Exec(c.Request.Context(),
			`UPDATE team_members SET workspace_id = $1 WHERE id = $2`, wsID, member.ID); err != nil {
			slog.Warn("CreateTeamMember: workspace stamp failed", "error", err)
		}
	}

	if h.queryCache != nil {
		pattern := fmt.Sprintf("team:user:%s*", user.ID)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if deleted, err := h.queryCache.DeleteByPattern(ctx, pattern); err == nil {
				slog.Info("CreateTeamMember: invalidated  cache entries for user", "id", deleted, "id", user.ID)
			}
		}()
	}

	memberMeta := map[string]string{"role": req.Role, "email": req.Email}
	h.enqueue(c.Request.Context(), services.Signal{
		Module:     services.ModuleTeam,
		ID:         "member-" + uuidString(member.ID.Bytes),
		AuthorID:   user.ID,
		Title:      req.Name,
		Body:       fmt.Sprintf("%s — %s", req.Name, req.Role),
		Genre:      "team-member",
		ModifiedAt: pgPlainTimestampToTime(member.UpdatedAt),
		Metadata:   memberMeta,
	})

	c.JSON(http.StatusCreated, member)
}

// GetTeamMember returns a single team member.
func (h *TeamHandler) GetTeamMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "team_member_id")
		return
	}

	queries := sqlc.New(h.pool)
	member, err := queries.GetTeamMember(c.Request.Context(), sqlc.GetTeamMemberParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Team member")
		return
	}

	if c.Query("include_activities") == "true" {
		limit := int32(20)
		activities, err := queries.GetTeamMemberActivities(c.Request.Context(), sqlc.GetTeamMemberActivitiesParams{
			MemberID: pgtype.UUID{Bytes: id, Valid: true},
			Limit:    limit,
		})
		if err == nil {
			c.JSON(http.StatusOK, gin.H{
				"member":     member,
				"activities": activities,
			})
			return
		}
	}

	c.JSON(http.StatusOK, member)
}

// UpdateTeamMember updates a team member.
func (h *TeamHandler) UpdateTeamMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "team_member_id")
		return
	}

	var req struct {
		Name       *string  `json:"name"`
		Email      *string  `json:"email"`
		Role       *string  `json:"role"`
		AvatarUrl  *string  `json:"avatar_url"`
		Status     *string  `json:"status"`
		Capacity   *int32   `json:"capacity"`
		ManagerID  *string  `json:"manager_id"`
		Skills     []string `json:"skills"`
		HourlyRate *float64 `json:"hourly_rate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	queries := sqlc.New(h.pool)

	existing, err := queries.GetTeamMember(c.Request.Context(), sqlc.GetTeamMemberParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondNotFound(c, slog.Default(), "Team member")
		return
	}

	name := existing.Name
	if req.Name != nil {
		name = *req.Name
	}

	email := existing.Email
	if req.Email != nil {
		email = *req.Email
	}

	role := existing.Role
	if req.Role != nil {
		role = *req.Role
	}

	avatarUrl := existing.AvatarUrl
	if req.AvatarUrl != nil {
		avatarUrl = req.AvatarUrl
	}

	status := existing.Status
	if req.Status != nil {
		status = sqlc.NullMemberstatus{
			Memberstatus: stringToMemberStatus(*req.Status),
			Valid:        true,
		}
	}

	capacity := existing.Capacity
	if req.Capacity != nil {
		capacity = req.Capacity
	}

	managerID := existing.ManagerID
	if req.ManagerID != nil {
		if parsed, err := uuid.Parse(*req.ManagerID); err == nil {
			managerID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	skills := existing.Skills
	if req.Skills != nil {
		if skillsJSON, err := json.Marshal(req.Skills); err == nil {
			skills = skillsJSON
		}
	}

	hourlyRate := existing.HourlyRate
	if req.HourlyRate != nil {
		hourlyRate = pgtype.Numeric{Valid: true}
		hourlyRate.Scan(*req.HourlyRate)
	}

	member, err := queries.UpdateTeamMember(c.Request.Context(), sqlc.UpdateTeamMemberParams{
		ID:         pgtype.UUID{Bytes: id, Valid: true},
		Name:       name,
		Email:      email,
		Role:       role,
		AvatarUrl:  avatarUrl,
		Status:     status,
		Capacity:   capacity,
		ManagerID:  managerID,
		Skills:     skills,
		HourlyRate: hourlyRate,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update team member", err)
		return
	}

	if h.queryCache != nil {
		pattern := fmt.Sprintf("team:user:%s*", user.ID)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if deleted, err := h.queryCache.DeleteByPattern(ctx, pattern); err == nil {
				slog.Info("UpdateTeamMember: invalidated  cache entries for user", "id", deleted, "id", user.ID)
			}
		}()
	}

	c.JSON(http.StatusOK, member)
}

// DeleteTeamMember deletes a team member.
func (h *TeamHandler) DeleteTeamMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.RespondInvalidID(c, slog.Default(), "team_member_id")
		return
	}

	queries := sqlc.New(h.pool)
	err = queries.DeleteTeamMember(c.Request.Context(), sqlc.DeleteTeamMemberParams{
		ID:     pgtype.UUID{Bytes: id, Valid: true},
		UserID: user.ID,
	})
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete team member", err)
		return
	}

	if h.queryCache != nil {
		pattern := fmt.Sprintf("team:user:%s*", user.ID)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if deleted, err := h.queryCache.DeleteByPattern(ctx, pattern); err == nil {
				slog.Info("DeleteTeamMember: invalidated  cache entries for user", "id", deleted, "id", user.ID)
			}
		}()
	}

	c.JSON(http.StatusOK, gin.H{"message": "Team member deleted"})
}
