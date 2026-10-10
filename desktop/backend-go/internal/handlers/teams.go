package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// TeamsHandler owns teams — the layer inside a workspace. Users are added to teams
// and moved between them. Managing teams requires a manager+ role in the workspace.
type TeamsHandler struct {
	pool *pgxpool.Pool
}

func NewTeamsHandler(pool *pgxpool.Pool) *TeamsHandler {
	return &TeamsHandler{pool: pool}
}

func (h *Handlers) registerTeamRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	th := NewTeamsHandler(h.pool)

	// Workspace-scoped: list + create teams.
	ws := api.Group("/workspaces/:id/teams")
	ws.Use(auth, middleware.RequireAuth())
	{
		ws.GET("", th.ListTeams)
		ws.POST("", th.CreateTeam)
	}

	// Team-scoped: update/delete + membership.
	t := api.Group("/teams/:teamId")
	t.Use(auth, middleware.RequireAuth())
	{
		t.PUT("", th.UpdateTeam)
		t.DELETE("", th.DeleteTeam)
		t.GET("/members", th.ListTeamMembers)
		t.POST("/members", th.AddTeamMember)
		t.DELETE("/members/:userId", th.RemoveTeamMember)
		t.POST("/members/:userId/move", th.MoveTeamMember)
	}
}

// workspaceRole returns the caller's role in a workspace, or "".
func (h *TeamsHandler) workspaceRole(ctx context.Context, workspaceID, userID string) string {
	var role string
	err := h.pool.QueryRow(ctx,
		`SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active'`,
		workspaceID, userID).Scan(&role)
	if err != nil {
		return ""
	}
	return role
}

func isWsManager(role string) bool { return role == "owner" || role == "admin" || role == "manager" }

// teamWorkspace returns the workspace id that owns a team.
func (h *TeamsHandler) teamWorkspace(ctx context.Context, teamID string) (string, error) {
	var wsID string
	err := h.pool.QueryRow(ctx, `SELECT workspace_id FROM teams WHERE id=$1`, teamID).Scan(&wsID)
	return wsID, err
}

// ListTeams — GET /api/workspaces/:id/teams
func (h *TeamsHandler) ListTeams(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	wsID := c.Param("id")
	if h.workspaceRole(c.Request.Context(), wsID, user.ID) == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT t.id, t.name, COALESCE(t.description,''), COALESCE(t.color,''),
		        (SELECT count(*) FROM team_memberships tm WHERE tm.team_id=t.id)
		 FROM teams t WHERE t.workspace_id=$1 ORDER BY t.created_at`, wsID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list teams", err)
		return
	}
	defer rows.Close()
	teams := []gin.H{}
	for rows.Next() {
		var id, name, desc, color string
		var count int
		if err := rows.Scan(&id, &name, &desc, &color, &count); err != nil {
			continue
		}
		teams = append(teams, gin.H{"id": id, "name": name, "description": desc, "color": color, "member_count": count})
	}
	c.JSON(http.StatusOK, gin.H{"teams": teams})
}

// CreateTeam — POST /api/workspaces/:id/teams
func (h *TeamsHandler) CreateTeam(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	wsID := c.Param("id")
	if !isWsManager(h.workspaceRole(c.Request.Context(), wsID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owners, admins, and managers can create teams"})
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	var id string
	err := h.pool.QueryRow(c.Request.Context(),
		`INSERT INTO teams (workspace_id, name, description, color, created_by) VALUES ($1,$2,$3,$4,$5) RETURNING id`,
		wsID, req.Name, req.Description, req.Color, user.ID).Scan(&id)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "create team", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "description": req.Description, "color": req.Color, "member_count": 0})
}

// UpdateTeam — PUT /api/teams/:teamId
func (h *TeamsHandler) UpdateTeam(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	teamID := c.Param("teamId")
	wsID, err := h.teamWorkspace(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}
	if !isWsManager(h.workspaceRole(c.Request.Context(), wsID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed"})
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Color       *string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	_, err = h.pool.Exec(c.Request.Context(),
		`UPDATE teams SET name=COALESCE($2,name), description=COALESCE($3,description), color=COALESCE($4,color), updated_at=NOW() WHERE id=$1`,
		teamID, req.Name, req.Description, req.Color)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "update team", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DeleteTeam — DELETE /api/teams/:teamId
func (h *TeamsHandler) DeleteTeam(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	teamID := c.Param("teamId")
	wsID, err := h.teamWorkspace(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}
	if !isWsManager(h.workspaceRole(c.Request.Context(), wsID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed"})
		return
	}
	if _, err := h.pool.Exec(c.Request.Context(), `DELETE FROM teams WHERE id=$1`, teamID); err != nil {
		utils.RespondInternalError(c, slog.Default(), "delete team", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ListTeamMembers — GET /api/teams/:teamId/members
func (h *TeamsHandler) ListTeamMembers(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	teamID := c.Param("teamId")
	wsID, err := h.teamWorkspace(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}
	if h.workspaceRole(c.Request.Context(), wsID, user.ID) == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a member of this workspace"})
		return
	}
	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT tm.user_id, tm.role, u.name, u.email, u.image
		 FROM team_memberships tm LEFT JOIN "user" u ON u.id=tm.user_id
		 WHERE tm.team_id=$1 ORDER BY tm.created_at`, teamID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "list team members", err)
		return
	}
	defer rows.Close()
	members := []gin.H{}
	for rows.Next() {
		var uid, role string
		var name, email, image *string
		if err := rows.Scan(&uid, &role, &name, &email, &image); err != nil {
			continue
		}
		members = append(members, gin.H{"user_id": uid, "role": role, "name": name, "email": email, "image": image})
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

// AddTeamMember — POST /api/teams/:teamId/members {user_id, role}
func (h *TeamsHandler) AddTeamMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	teamID := c.Param("teamId")
	wsID, err := h.teamWorkspace(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}
	if !isWsManager(h.workspaceRole(c.Request.Context(), wsID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed"})
		return
	}
	var req struct {
		UserID string `json:"user_id" binding:"required"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	// Target must be a member of the workspace.
	if h.workspaceRole(c.Request.Context(), wsID, req.UserID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User must be a member of the workspace first"})
		return
	}
	role := req.Role
	if role == "" {
		role = "member"
	}
	_, err = h.pool.Exec(c.Request.Context(),
		`INSERT INTO team_memberships (team_id, user_id, role, added_by) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (team_id, user_id) DO UPDATE SET role=$3`, teamID, req.UserID, role, user.ID)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "add team member", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"team_id": teamID, "user_id": req.UserID, "role": role})
}

// RemoveTeamMember — DELETE /api/teams/:teamId/members/:userId
func (h *TeamsHandler) RemoveTeamMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	teamID := c.Param("teamId")
	wsID, err := h.teamWorkspace(c.Request.Context(), teamID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}
	if !isWsManager(h.workspaceRole(c.Request.Context(), wsID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed"})
		return
	}
	_, err = h.pool.Exec(c.Request.Context(),
		`DELETE FROM team_memberships WHERE team_id=$1 AND user_id=$2`, teamID, c.Param("userId"))
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "remove team member", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed"})
}

// MoveTeamMember — POST /api/teams/:teamId/members/:userId/move {to_team_id}
// Moves a user from this team to another team in the same workspace.
func (h *TeamsHandler) MoveTeamMember(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	fromTeam := c.Param("teamId")
	target := c.Param("userId")
	wsID, err := h.teamWorkspace(c.Request.Context(), fromTeam)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
		return
	}
	if !isWsManager(h.workspaceRole(c.Request.Context(), wsID, user.ID)) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed"})
		return
	}
	var req struct {
		ToTeamID string `json:"to_team_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	// Destination team must be in the same workspace.
	toWs, err := h.teamWorkspace(c.Request.Context(), req.ToTeamID)
	if err != nil || toWs != wsID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Destination team must be in the same workspace"})
		return
	}
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "move member", err)
		return
	}
	defer tx.Rollback(c.Request.Context())
	if _, err := tx.Exec(c.Request.Context(), `DELETE FROM team_memberships WHERE team_id=$1 AND user_id=$2`, fromTeam, target); err != nil {
		utils.RespondInternalError(c, slog.Default(), "move member", err)
		return
	}
	if _, err := tx.Exec(c.Request.Context(),
		`INSERT INTO team_memberships (team_id, user_id, role, added_by) VALUES ($1,$2,'member',$3)
		 ON CONFLICT (team_id, user_id) DO NOTHING`, req.ToTeamID, target, user.ID); err != nil {
		utils.RespondInternalError(c, slog.Default(), "move member", err)
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		utils.RespondInternalError(c, slog.Default(), "move member", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "moved", "from_team": fromTeam, "to_team": req.ToTeamID, "user_id": target})
}
