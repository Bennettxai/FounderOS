package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// Boards - the composition layer (Views & Boards, Phase B).
//
// A board is a workspace-scoped definition: several views of the existing
// module primitives (projects, tasks, team, deals, clients) laid out on one
// surface, usually sharing one context filter (typically a client). The board
// DEFINITION is workspace-scoped (any active member can list/read/edit it,
// same model as custom_modules), but the board DATA stays USER-scoped: the
// module data model is user_id-scoped today (projects, tasks, deals, clients
// all hang off user_id), so /data resolves each view against the CALLER's
// rows. Sharing module data across a workspace is a separate, explicit
// decision - not silently changed here.
//
// Expected boards table shape (modeled on custom_modules):
//
//	boards (
//	    id UUID PK, workspace_id UUID -> workspaces, created_by VARCHAR,
//	    name VARCHAR, kind VARCHAR, subject_type VARCHAR, subject_id UUID,
//	    layout JSONB,           -- [{"view":"tasks","filters":{"client_id":"..."}}, ...]
//	    is_pinned BOOLEAN, position INT, created_at, updated_at
//	)
//
// The table may not exist yet in a given environment; every handler degrades
// to 503 "boards table not migrated" instead of a raw 500 in that case.

// BoardHandler handles board CRUD and server-side board data resolution.
type BoardHandler struct {
	pool *pgxpool.Pool
}

// NewBoardHandler returns a handler backed by the given pool.
func NewBoardHandler(pool *pgxpool.Pool) *BoardHandler {
	return &BoardHandler{pool: pool}
}

// registerBoardRoutes wires /api/boards. All routes require auth; workspace
// membership is enforced per-request via the X-Workspace-ID header.
func registerBoardRoutes(api *gin.RouterGroup, auth gin.HandlerFunc, h *BoardHandler) {
	boards := api.Group("/boards")
	boards.Use(auth, middleware.RequireAuth())
	{
		boards.GET("", h.ListBoards)
		boards.POST("", h.CreateBoard)
		boards.GET("/:id", h.GetBoard)
		boards.PUT("/:id", h.UpdateBoard)
		boards.DELETE("/:id", h.DeleteBoard)
		boards.POST("/:id/pin", h.PinBoard)
		boards.GET("/:id/data", h.GetBoardData)
	}
}

// boardCols is the canonical SELECT column list for the boards table.
const boardCols = `
	id, workspace_id, created_by, name, kind, subject_type, subject_id,
	layout, is_pinned, position, created_at, updated_at`

// boardResponse is the JSON shape returned for a board definition.
type boardResponse struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	CreatedBy   string          `json:"created_by"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	SubjectType *string         `json:"subject_type"`
	SubjectID   *string         `json:"subject_id"`
	Layout      json.RawMessage `json:"layout"`
	IsPinned    bool            `json:"is_pinned"`
	Position    int             `json:"position"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

// boardLayoutEntry is one view placement inside a board layout. Only the
// filters the resolver understands are decoded; unknown filter keys are
// ignored rather than rejected so layouts can grow without breaking older
// backends.
type boardLayoutEntry struct {
	View    string `json:"view"`
	Filters struct {
		ClientID string `json:"client_id"`
	} `json:"filters"`
}

// boardWorkspaceScope resolves the X-Workspace-ID header and confirms active
// membership (same check as calendar_events.go workspaceScope). Boards are
// always workspace-scoped, so unlike the calendar there is no personal
// fallback: a missing/invalid header or non-membership is a hard failure.
func (h *BoardHandler) boardWorkspaceScope(c *gin.Context, userID string) (uuid.UUID, bool) {
	hdr := c.GetHeader("X-Workspace-ID")
	if hdr == "" {
		RespondBadRequestErr(c, "X-Workspace-ID header required")
		return uuid.Nil, false
	}
	wsID, err := uuid.Parse(hdr)
	if err != nil {
		RespondBadRequestErr(c, "invalid X-Workspace-ID header")
		return uuid.Nil, false
	}
	var member bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&member)
	if err != nil || !member {
		RespondForbiddenErr(c, "not a member of this workspace")
		return uuid.Nil, false
	}
	return wsID, true
}

// respondBoardsNotMigrated returns true (and writes a 503) when err indicates
// the boards relation does not exist in this environment yet. Everything else
// is left to the caller's normal error handling.
func respondBoardsNotMigrated(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" && strings.Contains(pgErr.Message, "boards") {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "boards table not migrated"})
		return true
	}
	if strings.Contains(err.Error(), `relation "boards"`) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "boards table not migrated"})
		return true
	}
	return false
}

// scanBoard maps a boards row (boardCols order) into the response shape.
func scanBoard(row scanRow) (boardResponse, error) {
	var (
		b         boardResponse
		id, wsID  uuid.UUID
		subjectID *uuid.UUID
		layout    []byte
		createdAt time.Time
		updatedAt time.Time
	)
	err := row.Scan(&id, &wsID, &b.CreatedBy, &b.Name, &b.Kind, &b.SubjectType,
		&subjectID, &layout, &b.IsPinned, &b.Position, &createdAt, &updatedAt)
	if err != nil {
		return b, err
	}
	b.ID = id.String()
	b.WorkspaceID = wsID.String()
	b.SubjectID = uuidPtrToStrPtr(subjectID)
	b.Layout = json.RawMessage("[]")
	if len(layout) > 0 {
		b.Layout = json.RawMessage(layout)
	}
	b.CreatedAt = createdAt.Format(time.RFC3339)
	b.UpdatedAt = updatedAt.Format(time.RFC3339)
	return b, nil
}

// getBoardScoped loads one board gated by workspace membership (the workspace
// filter IS the access check: a board in another workspace reads as 404).
func (h *BoardHandler) getBoardScoped(c *gin.Context, boardID, wsID uuid.UUID) (boardResponse, error) {
	row := h.pool.QueryRow(c.Request.Context(), `
		SELECT `+boardCols+`
		FROM   boards
		WHERE  id = $1 AND workspace_id = $2
	`, boardID, wsID)
	return scanBoard(row)
}

// ListBoards returns the workspace's boards, optionally filtered to pinned.
// GET /api/boards?pinned=true
func (h *BoardHandler) ListBoards(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.boardWorkspaceScope(c, user.ID)
	if !ok {
		return
	}

	query := `
		SELECT ` + boardCols + `
		FROM   boards
		WHERE  workspace_id = $1`
	switch c.Query("pinned") {
	case "true", "1":
		query += ` AND is_pinned = TRUE`
	case "false", "0":
		query += ` AND is_pinned = FALSE`
	}
	query += ` ORDER BY position ASC, created_at ASC`

	rows, err := h.pool.Query(c.Request.Context(), query, wsID)
	if err != nil {
		if respondBoardsNotMigrated(c, err) {
			return
		}
		RespondInternalErr(c, "list boards", err)
		return
	}
	defer rows.Close()

	boards := make([]boardResponse, 0)
	for rows.Next() {
		b, err := scanBoard(rows)
		if err != nil {
			RespondInternalErr(c, "scan board", err)
			return
		}
		boards = append(boards, b)
	}
	if err := rows.Err(); err != nil {
		if respondBoardsNotMigrated(c, err) {
			return
		}
		RespondInternalErr(c, "iterate boards", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"boards": boards, "count": len(boards)})
}

// boardInput is the create/update payload.
type boardInput struct {
	Name        *string         `json:"name"`
	Kind        *string         `json:"kind"`
	SubjectType *string         `json:"subject_type"`
	SubjectID   *string         `json:"subject_id"`
	Layout      json.RawMessage `json:"layout"`
	IsPinned    *bool           `json:"is_pinned"`
	Position    *int            `json:"position"`
}

// CreateBoard inserts a board into the active workspace, created_by = caller.
// POST /api/boards  body: {name, kind, subject_type, subject_id, layout}
func (h *BoardHandler) CreateBoard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.boardWorkspaceScope(c, user.ID)
	if !ok {
		return
	}

	var in boardInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}
	if in.Name == nil || strings.TrimSpace(*in.Name) == "" {
		RespondBadRequestErr(c, "name is required")
		return
	}
	kind := "board"
	if in.Kind != nil && *in.Kind != "" {
		kind = *in.Kind
	}
	layout := json.RawMessage("[]")
	if len(in.Layout) > 0 && string(in.Layout) != "null" {
		if !json.Valid(in.Layout) {
			RespondBadRequestErr(c, "layout must be valid JSON")
			return
		}
		layout = in.Layout
	}

	row := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO boards
			(workspace_id, created_by, name, kind, subject_type, subject_id, layout)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+boardCols+`
	`, wsID, user.ID, strings.TrimSpace(*in.Name), kind, in.SubjectType,
		nullableUUID(in.SubjectID), []byte(layout))

	b, err := scanBoard(row)
	if err != nil {
		if respondBoardsNotMigrated(c, err) {
			return
		}
		RespondInternalErr(c, "create board", err)
		return
	}
	c.JSON(http.StatusCreated, b)
}

// GetBoard returns a single board definition.
// GET /api/boards/:id
func (h *BoardHandler) GetBoard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid board id")
		return
	}
	wsID, ok := h.boardWorkspaceScope(c, user.ID)
	if !ok {
		return
	}

	b, err := h.getBoardScoped(c, boardID, wsID)
	if err != nil {
		if respondBoardsNotMigrated(c, err) {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			RespondNotFoundErr(c, "board")
			return
		}
		RespondInternalErr(c, "get board", err)
		return
	}
	c.JSON(http.StatusOK, b)
}

// UpdateBoard edits name/layout/is_pinned/position of a workspace board.
// PUT /api/boards/:id  body: {name?, layout?, is_pinned?, position?}
func (h *BoardHandler) UpdateBoard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid board id")
		return
	}
	wsID, ok := h.boardWorkspaceScope(c, user.ID)
	if !ok {
		return
	}

	var in boardInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}
	if len(in.Layout) > 0 && string(in.Layout) != "null" && !json.Valid(in.Layout) {
		RespondBadRequestErr(c, "layout must be valid JSON")
		return
	}

	row := h.pool.QueryRow(c.Request.Context(), `
		UPDATE boards SET
			name       = COALESCE($3, name),
			layout     = COALESCE($4, layout),
			is_pinned  = COALESCE($5, is_pinned),
			position   = COALESCE($6, position),
			updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2
		RETURNING `+boardCols+`
	`, boardID, wsID, in.Name, nullableRawJSON(in.Layout), in.IsPinned, in.Position)

	b, err := scanBoard(row)
	if err != nil {
		if respondBoardsNotMigrated(c, err) {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			RespondNotFoundErr(c, "board")
			return
		}
		RespondInternalErr(c, "update board", err)
		return
	}
	c.JSON(http.StatusOK, b)
}

// DeleteBoard removes a board from the workspace.
// DELETE /api/boards/:id
func (h *BoardHandler) DeleteBoard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid board id")
		return
	}
	wsID, ok := h.boardWorkspaceScope(c, user.ID)
	if !ok {
		return
	}

	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM boards WHERE id = $1 AND workspace_id = $2`, boardID, wsID)
	if err != nil {
		if respondBoardsNotMigrated(c, err) {
			return
		}
		RespondInternalErr(c, "delete board", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "board")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "board deleted"})
}

// PinBoard toggles sidebar pinning for a board.
// POST /api/boards/:id/pin  body: {"pinned": true}
func (h *BoardHandler) PinBoard(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid board id")
		return
	}
	wsID, ok := h.boardWorkspaceScope(c, user.ID)
	if !ok {
		return
	}

	var body struct {
		Pinned *bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Pinned == nil {
		RespondBadRequestErr(c, "pinned (boolean) is required")
		return
	}

	row := h.pool.QueryRow(c.Request.Context(), `
		UPDATE boards SET is_pinned = $3, updated_at = NOW()
		WHERE id = $1 AND workspace_id = $2
		RETURNING `+boardCols+`
	`, boardID, wsID, *body.Pinned)

	b, err := scanBoard(row)
	if err != nil {
		if respondBoardsNotMigrated(c, err) {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			RespondNotFoundErr(c, "board")
			return
		}
		RespondInternalErr(c, "pin board", err)
		return
	}
	c.JSON(http.StatusOK, b)
}

// GetBoardData resolves the board's layout server-side: one section per
// layout entry, each a slice of the caller's module data. Board data is
// USER-scoped (the module data model is user_id-scoped today - projects,
// tasks, deals and clients all belong to a user, not a workspace), so every
// query below filters by the caller's user_id.
// GET /api/boards/:id/data
func (h *BoardHandler) GetBoardData(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid board id")
		return
	}
	wsID, ok := h.boardWorkspaceScope(c, user.ID)
	if !ok {
		return
	}

	board, err := h.getBoardScoped(c, boardID, wsID)
	if err != nil {
		if respondBoardsNotMigrated(c, err) {
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			RespondNotFoundErr(c, "board")
			return
		}
		RespondInternalErr(c, "get board", err)
		return
	}

	// A layout that fails to decode as an entry array resolves to zero
	// sections rather than an error - the definition endpoint still exposes
	// the raw layout for the builder UI to repair.
	entries := make([]boardLayoutEntry, 0)
	_ = json.Unmarshal(board.Layout, &entries)

	sections := make([]gin.H, 0, len(entries))
	for _, entry := range entries {
		// Optional client context filter, shared by all view resolvers.
		var clientID *uuid.UUID
		if entry.Filters.ClientID != "" {
			id, err := uuid.Parse(entry.Filters.ClientID)
			if err != nil {
				sections = append(sections, gin.H{
					"view": entry.View, "items": []gin.H{}, "count": 0,
					"error": "invalid client_id filter",
				})
				continue
			}
			clientID = &id
		}

		var (
			items    []gin.H
			viewErr  error
			resolved = true
		)
		switch entry.View {
		case "projects":
			items, viewErr = h.boardViewProjects(c, user.ID, clientID)
		case "tasks":
			items, viewErr = h.boardViewTasks(c, user.ID, clientID)
		case "team":
			items, viewErr = h.boardViewTeam(c, user.ID, clientID)
		case "deals":
			items, viewErr = h.boardViewDeals(c, user.ID, clientID)
		case "clients":
			items, viewErr = h.boardViewClients(c, user.ID)
		default:
			resolved = false
		}
		if !resolved {
			sections = append(sections, gin.H{
				"view": entry.View, "items": []gin.H{}, "count": 0,
				"error": "unsupported view",
			})
			continue
		}
		if viewErr != nil {
			RespondInternalErr(c, "resolve board view "+entry.View, viewErr)
			return
		}
		if items == nil {
			items = []gin.H{}
		}
		sections = append(sections, gin.H{
			"view": entry.View, "items": items, "count": len(items),
		})
	}

	c.JSON(http.StatusOK, gin.H{"board": board, "sections": sections})
}

// boardViewProjects: the caller's projects, optionally filtered to a client.
func (h *BoardHandler) boardViewProjects(c *gin.Context, userID string, clientID *uuid.UUID) ([]gin.H, error) {
	query := `
		SELECT id, name, description, status::text, priority::text,
		       client_id, client_name, start_date, due_date, created_at
		FROM   projects
		WHERE  user_id = $1`
	args := []interface{}{userID}
	if clientID != nil {
		query += ` AND client_id = $2`
		args = append(args, *clientID)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id                 uuid.UUID
			name               string
			description        *string
			status, priority   *string
			cliID              *uuid.UUID
			clientName         *string
			startDate, dueDate *time.Time
			createdAt          *time.Time
		)
		if err := rows.Scan(&id, &name, &description, &status, &priority,
			&cliID, &clientName, &startDate, &dueDate, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, gin.H{
			"id":          id.String(),
			"name":        name,
			"description": description,
			"status":      status,
			"priority":    priority,
			"client_id":   uuidPtrToStrPtr(cliID),
			"client_name": clientName,
			"start_date":  boardTimeStr(startDate),
			"due_date":    boardTimeStr(dueDate),
			"created_at":  boardTimeStr(createdAt),
		})
	}
	return items, rows.Err()
}

// boardViewTasks: the caller's tasks. With a client filter, tasks attach to a
// client transitively through their project (tasks have no client_id column);
// orphan tasks (no project) are only visible in the unfiltered view.
func (h *BoardHandler) boardViewTasks(c *gin.Context, userID string, clientID *uuid.UUID) ([]gin.H, error) {
	query := `
		SELECT t.id, t.title, t.status::text, t.priority::text, t.due_date,
		       t.project_id, p.name, t.assignee_id, t.created_at
		FROM   tasks t
		LEFT   JOIN projects p ON p.id = t.project_id
		WHERE  t.user_id = $1`
	args := []interface{}{userID}
	if clientID != nil {
		query = strings.Replace(query, "LEFT   JOIN", "JOIN  ", 1)
		query += ` AND p.client_id = $2`
		args = append(args, *clientID)
	}
	query += ` ORDER BY t.position ASC, t.created_at ASC`

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id               uuid.UUID
			title            string
			status, priority *string
			dueDate          *time.Time
			projectID        *uuid.UUID
			projectName      *string
			assigneeID       *uuid.UUID
			createdAt        *time.Time
		)
		if err := rows.Scan(&id, &title, &status, &priority, &dueDate,
			&projectID, &projectName, &assigneeID, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, gin.H{
			"id":           id.String(),
			"title":        title,
			"status":       status,
			"priority":     priority,
			"due_date":     boardTimeStr(dueDate),
			"project_id":   uuidPtrToStrPtr(projectID),
			"project_name": projectName,
			"assignee_id":  uuidPtrToStrPtr(assigneeID),
			"created_at":   boardTimeStr(createdAt),
		})
	}
	return items, rows.Err()
}

// boardViewTeam: distinct team members assigned to the caller's projects via
// project_members (team_members have no client link; "who works for client X"
// is the join through project_members -> projects.client_id).
func (h *BoardHandler) boardViewTeam(c *gin.Context, userID string, clientID *uuid.UUID) ([]gin.H, error) {
	query := `
		SELECT DISTINCT tm.id, tm.name, tm.email, tm.role, tm.avatar_url, tm.status::text
		FROM   team_members tm
		JOIN   project_members pm ON pm.team_member_id = tm.id
		JOIN   projects p         ON p.id = pm.project_id
		WHERE  p.user_id = $1`
	args := []interface{}{userID}
	if clientID != nil {
		query += ` AND p.client_id = $2`
		args = append(args, *clientID)
	}
	query += ` ORDER BY tm.name ASC`

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id          uuid.UUID
			name, email string
			role        string
			avatarURL   *string
			status      *string
		)
		if err := rows.Scan(&id, &name, &email, &role, &avatarURL, &status); err != nil {
			return nil, err
		}
		items = append(items, gin.H{
			"id":         id.String(),
			"name":       name,
			"email":      email,
			"role":       role,
			"avatar_url": avatarURL,
			"status":     status,
		})
	}
	return items, rows.Err()
}

// boardViewDeals: the caller's deals, optionally filtered to a client.
func (h *BoardHandler) boardViewDeals(c *gin.Context, userID string, clientID *uuid.UUID) ([]gin.H, error) {
	query := `
		SELECT d.id, d.name, d.amount::float8, d.currency, d.status, d.priority,
		       d.probability, d.expected_close_date, d.client_id, s.name, d.created_at
		FROM   deals d
		LEFT   JOIN pipeline_stages s ON s.id = d.stage_id
		WHERE  d.user_id = $1`
	args := []interface{}{userID}
	if clientID != nil {
		query += ` AND d.client_id = $2`
		args = append(args, *clientID)
	}
	query += ` ORDER BY d.created_at DESC`

	rows, err := h.pool.Query(c.Request.Context(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id                uuid.UUID
			name              string
			amount            *float64
			currency          *string
			status, priority  *string
			probability       *int32
			expectedCloseDate *time.Time
			cliID             *uuid.UUID
			stageName         *string
			createdAt         *time.Time
		)
		if err := rows.Scan(&id, &name, &amount, &currency, &status, &priority,
			&probability, &expectedCloseDate, &cliID, &stageName, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, gin.H{
			"id":                  id.String(),
			"name":                name,
			"amount":              amount,
			"currency":            currency,
			"status":              status,
			"priority":            priority,
			"probability":         probability,
			"expected_close_date": boardDateStr(expectedCloseDate),
			"client_id":           uuidPtrToStrPtr(cliID),
			"stage_name":          stageName,
			"created_at":          boardTimeStr(createdAt),
		})
	}
	return items, rows.Err()
}

// boardViewClients: the caller's clients list (no client_id filter - a
// clients view IS the client list).
func (h *BoardHandler) boardViewClients(c *gin.Context, userID string) ([]gin.H, error) {
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, name, type::text, status::text, email, phone, industry, created_at
		FROM   clients
		WHERE  user_id = $1
		ORDER  BY name ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id           uuid.UUID
			name         string
			ctype        *string
			status       *string
			email, phone *string
			industry     *string
			createdAt    *time.Time
		)
		if err := rows.Scan(&id, &name, &ctype, &status, &email, &phone, &industry, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, gin.H{
			"id":         id.String(),
			"name":       name,
			"type":       ctype,
			"status":     status,
			"email":      email,
			"phone":      phone,
			"industry":   industry,
			"created_at": boardTimeStr(createdAt),
		})
	}
	return items, rows.Err()
}

// boardTimeStr formats an optional timestamp as RFC3339 (nil stays nil).
func boardTimeStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// boardDateStr formats an optional DATE column as YYYY-MM-DD (nil stays nil).
func boardDateStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}
