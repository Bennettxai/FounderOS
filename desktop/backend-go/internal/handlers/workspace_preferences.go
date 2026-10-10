package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// WorkspacePreferencesHandler manages the workspace-level settings primitive:
// the fundamental business settings (time zone, date/time format, week start,
// working hours) that every module reads, plus a calendar-specific JSONB block.
// Workspace-scoped via X-Workspace-ID + active membership.
type WorkspacePreferencesHandler struct {
	pool *pgxpool.Pool
}

func NewWorkspacePreferencesHandler(pool *pgxpool.Pool) *WorkspacePreferencesHandler {
	return &WorkspacePreferencesHandler{pool: pool}
}

type workspacePreferences struct {
	WorkspaceID         string          `json:"workspace_id"`
	Timezone            string          `json:"timezone"`
	DateFormat          string          `json:"date_format"`
	TimeFormat          string          `json:"time_format"`
	WeekStart           int             `json:"week_start"`
	WorkingHoursStart   int             `json:"working_hours_start"`
	WorkingHoursEnd     int             `json:"working_hours_end"`
	DefaultEventMinutes int             `json:"default_event_minutes"`
	Language            string          `json:"language"`
	Calendar            json.RawMessage `json:"calendar"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

func (h *WorkspacePreferencesHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		return uuid.Nil, false
	}
	var member bool
	if err := h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member); err != nil || !member {
		return uuid.Nil, false
	}
	return wsID, true
}

// GetPreferences returns the active workspace's preferences, creating sane
// defaults on first read so the client always has a complete object.
// GET /api/workspace/preferences
func (h *WorkspacePreferencesHandler) GetPreferences(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		RespondBadRequestErr(c, "no active workspace")
		return
	}

	if _, err := h.pool.Exec(c.Request.Context(),
		`INSERT INTO workspace_preferences (workspace_id) VALUES ($1)
		 ON CONFLICT (workspace_id) DO NOTHING`, wsID); err != nil {
		RespondInternalErr(c, "init workspace preferences", err)
		return
	}

	p, err := h.scan(c, wsID)
	if err != nil {
		RespondInternalErr(c, "read workspace preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

type prefsInput struct {
	Timezone            *string          `json:"timezone"`
	DateFormat          *string          `json:"date_format"`
	TimeFormat          *string          `json:"time_format"`
	WeekStart           *int             `json:"week_start"`
	WorkingHoursStart   *int             `json:"working_hours_start"`
	WorkingHoursEnd     *int             `json:"working_hours_end"`
	DefaultEventMinutes *int             `json:"default_event_minutes"`
	Language            *string          `json:"language"`
	Calendar            *json.RawMessage `json:"calendar"`
}

// UpdatePreferences upserts the active workspace's preferences (partial update).
// PUT /api/workspace/preferences
func (h *WorkspacePreferencesHandler) UpdatePreferences(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		RespondBadRequestErr(c, "no active workspace")
		return
	}

	var in prefsInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid preferences payload")
		return
	}

	// Ensure a row exists, then apply COALESCE-style partial update.
	if _, err := h.pool.Exec(c.Request.Context(),
		`INSERT INTO workspace_preferences (workspace_id) VALUES ($1)
		 ON CONFLICT (workspace_id) DO NOTHING`, wsID); err != nil {
		RespondInternalErr(c, "init workspace preferences", err)
		return
	}

	var calArg interface{}
	if in.Calendar != nil {
		calArg = []byte(*in.Calendar)
	}
	if _, err := h.pool.Exec(c.Request.Context(), `
		UPDATE workspace_preferences SET
			timezone              = COALESCE($2, timezone),
			date_format           = COALESCE($3, date_format),
			time_format           = COALESCE($4, time_format),
			week_start            = COALESCE($5, week_start),
			working_hours_start   = COALESCE($6, working_hours_start),
			working_hours_end     = COALESCE($7, working_hours_end),
			default_event_minutes = COALESCE($8, default_event_minutes),
			language              = COALESCE($9, language),
			calendar              = COALESCE($10, calendar),
			updated_at            = now()
		WHERE workspace_id = $1
	`, wsID, in.Timezone, in.DateFormat, in.TimeFormat, in.WeekStart,
		in.WorkingHoursStart, in.WorkingHoursEnd, in.DefaultEventMinutes, in.Language, calArg); err != nil {
		RespondInternalErr(c, "update workspace preferences", err)
		return
	}

	p, err := h.scan(c, wsID)
	if err != nil {
		RespondInternalErr(c, "read workspace preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *WorkspacePreferencesHandler) scan(c *gin.Context, wsID uuid.UUID) (*workspacePreferences, error) {
	var p workspacePreferences
	var cal []byte
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT workspace_id, timezone, date_format, time_format, week_start,
		       working_hours_start, working_hours_end, default_event_minutes,
		       language, calendar, updated_at
		FROM workspace_preferences WHERE workspace_id = $1
	`, wsID).Scan(&p.WorkspaceID, &p.Timezone, &p.DateFormat, &p.TimeFormat, &p.WeekStart,
		&p.WorkingHoursStart, &p.WorkingHoursEnd, &p.DefaultEventMinutes, &p.Language, &cal, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if len(cal) == 0 {
		cal = []byte("{}")
	}
	p.Calendar = cal
	return &p, nil
}
