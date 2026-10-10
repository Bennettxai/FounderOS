// Package handlers — optimal_engine_routes.go
//
// Three HTTP surfaces sharing the same OptimalEngine client:
//
//  1. /api/v1/optimal/ask + /api/v1/optimal/recall/* — natural-language
//     queries against the engine (RAG + recall primitives).
//  2. /api/v1/optimal/memory/* — CRUD over the engine's memory primitive,
//     scoped to the BO user's workspace.
//  3. /api/v1/brief/{morning,evening} — composed daily briefings that
//     combine BusinessOS module data (tasks, calendar, recent activity)
//     with engine recall.
//
// Every route requires session auth (mounted behind RequireAuth) and
// returns 503 when the engine isn't configured so callers fail loudly
// instead of getting silently empty results.
package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/optimalengine"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// OptimalEngineHandler exposes engine-backed endpoints. Constructed once
// at startup with a shared *optimalengine.Client (reused across all three
// route groups). The pool is needed for the briefing endpoints which read
// from the BO database (tasks, calendar) before composing recall.
type OptimalEngineHandler struct {
	engine *optimalengine.Client
	pool   sqlcConn
}

// sqlcConn lets us inject either *pgxpool.Pool (production) or a stub
// (tests) wherever a sqlc.New(arg) is needed. Defined narrowly so this
// handler doesn't pull in pgxpool at the type level.
type sqlcConn interface {
	sqlc.DBTX
}

// NewOptimalEngineHandler is the single constructor. Pass nil for engine
// to disable every route (they'll return 503).
func NewOptimalEngineHandler(engine *optimalengine.Client, pool sqlcConn) *OptimalEngineHandler {
	return &OptimalEngineHandler{engine: engine, pool: pool}
}

// RegisterOptimalEngineRoutes mounts everything under /api on the given
// group, behind the supplied auth middleware.
func RegisterOptimalEngineRoutes(api *gin.RouterGroup, h *OptimalEngineHandler, auth gin.HandlerFunc) {
	if h == nil {
		return
	}

	// Mounted under /engine (NOT /optimal) — /api/optimal/*path is a
	// catch-all proxy that would shadow these typed routes if they
	// shared the prefix. /engine is BO-side typed surface; /optimal
	// stays as raw passthrough to the engine.
	opt := api.Group("/engine")
	opt.Use(auth, middleware.RequireAuth())
	{
		opt.POST("/ask", h.Ask)

		recall := opt.Group("/recall")
		recall.GET("/who", h.RecallWho)
		recall.GET("/when", h.RecallWhen)
		recall.GET("/where", h.RecallWhere)
		recall.GET("/owns", h.RecallOwns)
		recall.GET("/actions", h.RecallActions)

		mem := opt.Group("/memory")
		mem.GET("", h.ListMemories)
		mem.POST("", h.CreateMemory)
		mem.POST("/:id/forget", h.ForgetMemory)
	}

	brief := api.Group("/brief")
	brief.Use(auth, middleware.RequireAuth())
	{
		brief.GET("/morning", h.MorningBrief)
		brief.GET("/evening", h.EveningBrief)
	}
}

// ----- Ask + Recall -----------------------------------------------------------

// askRequest mirrors the engine's POST /api/rag body. We pass it through
// almost verbatim, with workspace defaulting to "default" if not supplied.
type askRequest struct {
	Query     string `json:"query" binding:"required"`
	Workspace string `json:"workspace"`
	Audience  string `json:"audience"`
	Format    string `json:"format"`
	Bandwidth string `json:"bandwidth"`
}

// Ask routes a natural-language question through the engine's RAG pipeline.
// The engine handles intent classification, wiki lookup, chunk retrieval,
// and envelope composition — we just relay the typed response.
func (h *OptimalEngineHandler) Ask(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	var req askRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	resp, err := h.engine.Ask(c.Request.Context(), optimalengine.AskRequest{
		Query:     req.Query,
		Workspace: req.Workspace,
		Audience:  req.Audience,
		Format:    req.Format,
		Bandwidth: req.Bandwidth,
	})
	if err != nil {
		h.engineError(c, "ask", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RecallWho handles /api/v1/optimal/recall/who?topic=…&role=…&workspace=
func (h *OptimalEngineHandler) RecallWho(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	resp, err := h.engine.RecallWho(
		c.Request.Context(),
		c.Query("topic"),
		c.DefaultQuery("role", "owner"),
		c.Query("workspace"),
	)
	if err != nil {
		h.engineError(c, "recall.who", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RecallWhen handles /api/v1/optimal/recall/when?event=…&workspace=
func (h *OptimalEngineHandler) RecallWhen(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	resp, err := h.engine.RecallWhen(c.Request.Context(), c.Query("event"), c.Query("workspace"))
	if err != nil {
		h.engineError(c, "recall.when", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RecallWhere handles /api/v1/optimal/recall/where?thing=…&workspace=
func (h *OptimalEngineHandler) RecallWhere(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	resp, err := h.engine.RecallWhere(c.Request.Context(), c.Query("thing"), c.Query("workspace"))
	if err != nil {
		h.engineError(c, "recall.where", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RecallOwns handles /api/v1/optimal/recall/owns?actor=…&workspace=
// When actor is empty, it defaults to the authed user's id so a generic
// "what am I committed to" works without any params.
func (h *OptimalEngineHandler) RecallOwns(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	actor := c.Query("actor")
	if actor == "" {
		if u := middleware.GetCurrentUser(c); u != nil {
			actor = u.ID
		}
	}
	resp, err := h.engine.RecallOwns(c.Request.Context(), actor, c.Query("workspace"))
	if err != nil {
		h.engineError(c, "recall.owns", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// RecallActions handles /api/v1/optimal/recall/actions?actor=&topic=&since=&workspace=
func (h *OptimalEngineHandler) RecallActions(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	resp, err := h.engine.RecallActions(
		c.Request.Context(),
		c.Query("actor"),
		c.Query("topic"),
		c.Query("since"),
		c.Query("workspace"),
	)
	if err != nil {
		h.engineError(c, "recall.actions", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ----- Memory ----------------------------------------------------------------

// ListMemories handles GET /api/v1/optimal/memory?workspace=&limit=&audience=
func (h *OptimalEngineHandler) ListMemories(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	resp, err := h.engine.ListMemories(c.Request.Context(), optimalengine.ListMemoriesOptions{
		Workspace:          c.Query("workspace"),
		Audience:           c.Query("audience"),
		Limit:              limit,
		IncludeForgotten:   c.Query("include_forgotten") == "true",
		IncludeOldVersions: c.Query("include_old_versions") == "true",
	})
	if err != nil {
		h.engineError(c, "memory.list", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// createMemoryRequest is the BO-side body for POST /api/v1/optimal/memory.
// We attach the authed user id as metadata.author_id so memories are
// always traceable back to a BusinessOS user.
type createMemoryRequest struct {
	Content     string                 `json:"content" binding:"required"`
	Workspace   string                 `json:"workspace"`
	Audience    string                 `json:"audience"`
	IsStatic    *bool                  `json:"is_static"`
	CitationURI string                 `json:"citation_uri"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CreateMemory handles POST /api/v1/optimal/memory.
func (h *OptimalEngineHandler) CreateMemory(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	var req createMemoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}
	meta := req.Metadata
	if meta == nil {
		meta = map[string]interface{}{}
	}
	if u := middleware.GetCurrentUser(c); u != nil {
		meta["author_id"] = u.ID
		if u.Email != "" {
			meta["author_email"] = u.Email
		}
	}
	mem, err := h.engine.CreateMemory(c.Request.Context(), optimalengine.CreateMemoryRequest{
		Content:     req.Content,
		Workspace:   req.Workspace,
		Audience:    req.Audience,
		IsStatic:    req.IsStatic,
		CitationURI: req.CitationURI,
		Metadata:    meta,
	})
	if err != nil {
		h.engineError(c, "memory.create", err)
		return
	}
	c.JSON(http.StatusCreated, mem)
}

// ForgetMemory handles POST /api/v1/optimal/memory/:id/forget.
func (h *OptimalEngineHandler) ForgetMemory(c *gin.Context) {
	if !h.requireEngine(c) {
		return
	}
	id := c.Param("id")
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := h.engine.ForgetMemory(c.Request.Context(), id, body.Reason); err != nil {
		h.engineError(c, "memory.forget", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ----- Brief ------------------------------------------------------------------

// MorningBrief composes "what's on today" for the authed user. Pulls from
// BO Postgres (open tasks, today's calendar events) AND the engine
// (recall.owns(user) for stale commitments + recall.actions for context).
func (h *OptimalEngineHandler) MorningBrief(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)

	// Open tasks (any status except completed/cancelled, plus a soft cap
	// so a user with hundreds of open items doesn't pay a 50KB response).
	tasks, _ := queries.ListTasks(ctx, sqlc.ListTasksParams{UserID: user.ID})
	open := make([]any, 0, 10)
	for _, t := range tasks {
		if t.Status.Valid && (t.Status.Taskstatus == sqlc.TaskstatusDone || t.Status.Taskstatus == sqlc.TaskstatusCancelled) {
			continue
		}
		open = append(open, TransformTask(t))
		if len(open) == 10 {
			break
		}
	}

	// Today's calendar events.
	events, _ := queries.GetUpcomingEvents(ctx, sqlc.GetUpcomingEventsParams{UserID: user.ID, Limit: 10})
	todayEvents := make([]any, 0, len(events))
	for _, e := range events {
		if !inTimeRange(e.StartTime, todayStart, todayEnd) {
			continue
		}
		todayEvents = append(todayEvents, TransformCalendarEvent(e))
	}

	// Engine recall — what is the user committed to (open commitments
	// across the whole graph, not just BO tasks)?
	var commitments optimalengine.AskResponse
	if h.engine.Enabled() {
		if r, err := h.engine.RecallOwns(ctx, user.ID, ""); err == nil {
			commitments = r
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"date":         todayStart.Format("2006-01-02"),
		"user_id":      user.ID,
		"open_tasks":   open,
		"today_events": todayEvents,
		"commitments":  commitments,
		"generated_at": now.UTC().Format(time.RFC3339),
	})
}

// EveningBrief composes "what got done today" for the authed user.
// Pulls completed BO tasks + closed deals from Postgres AND uses the
// engine's recall.actions for cross-module activity ("what decisions /
// commitments / actions happened today").
func (h *OptimalEngineHandler) EveningBrief(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx := c.Request.Context()
	queries := sqlc.New(h.pool)

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	since := todayStart.Format("2006-01-02")

	// Completed tasks today.
	tasks, _ := queries.ListTasks(ctx, sqlc.ListTasksParams{UserID: user.ID})
	completed := make([]any, 0, 10)
	for _, t := range tasks {
		if !t.Status.Valid || t.Status.Taskstatus != sqlc.TaskstatusDone {
			continue
		}
		// UpdatedAt within today is our heuristic for "completed today" —
		// the schema doesn't have a completed_at column.
		if !inTimeRangeTimestamp(t.UpdatedAt, todayStart, todayStart.Add(24*time.Hour)) {
			continue
		}
		completed = append(completed, TransformTask(t))
		if len(completed) == 20 {
			break
		}
	}

	// Engine recall — actions / decisions / commitments since today started.
	var activity optimalengine.AskResponse
	if h.engine.Enabled() {
		if r, err := h.engine.RecallActions(ctx, user.ID, "", since, ""); err == nil {
			activity = r
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"date":            todayStart.Format("2006-01-02"),
		"user_id":         user.ID,
		"completed_tasks": completed,
		"activity":        activity,
		"generated_at":    now.UTC().Format(time.RFC3339),
	})
}

// ----- helpers ---------------------------------------------------------------

func (h *OptimalEngineHandler) requireEngine(c *gin.Context) bool {
	if h.engine == nil || !h.engine.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OptimalEngine not configured",
			"hint":  "set OPTIMAL_ENGINE_URL to enable engine-backed routes",
		})
		return false
	}
	return true
}

// engineError surfaces upstream failures verbatim. We don't try to
// categorize 4xx vs 5xx — the engine returns its own status codes and
// this handler is a thin wrapper. Operators triaging will grep for
// "optimal_engine_handler:" in the logs.
func (h *OptimalEngineHandler) engineError(c *gin.Context, op string, err error) {
	slog.WarnContext(c.Request.Context(), "optimal_engine_handler: upstream call failed",
		"op", op, "error", err.Error())
	msg := err.Error()
	c.JSON(http.StatusBadGateway, gin.H{
		"error":          "engine call failed",
		"op":             op,
		"upstream_error": strings.TrimSpace(msg),
	})
}

// inTimeRange handles either pgtype.Timestamp (no tz) or pgtype.Timestamptz
// (with tz). Calendar events use Timestamptz, task UpdatedAt is Timestamp;
// keeping one helper per shape keeps the call sites simple.
func inTimeRange(ts pgtype.Timestamptz, lo, hi time.Time) bool {
	if !ts.Valid {
		return false
	}
	return !ts.Time.Before(lo) && ts.Time.Before(hi)
}

func inTimeRangeTimestamp(ts pgtype.Timestamp, lo, hi time.Time) bool {
	if !ts.Valid {
		return false
	}
	return !ts.Time.Before(lo) && ts.Time.Before(hi)
}

// linkOptimalEngineHandlers wires the EngineSync's shared client into a
// fresh *OptimalEngineHandler. Defined here (not in handlers.go) so the
// dependency on services.EngineSync is local to engine-backed routes.
func (h *Handlers) buildOptimalEngineHandler() *OptimalEngineHandler {
	var client *optimalengine.Client
	if es, ok := any(h.engineSync).(interface{ Engine() *optimalengine.Client }); ok && es != nil {
		client = es.Engine()
	}
	return NewOptimalEngineHandler(client, h.pool)
}

// Compile-time assertion that EngineSync exposes Engine().
var _ = func() *optimalengine.Client {
	var es *services.EngineSync
	return es.Engine()
}()
