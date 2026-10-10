package handlers

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

const desktopSpaceMaxConfigBytes = 256 * 1024

type DesktopSpacesHandler struct {
	pool *pgxpool.Pool
}

func NewDesktopSpacesHandler(pool *pgxpool.Pool) *DesktopSpacesHandler {
	return &DesktopSpacesHandler{pool: pool}
}

type desktopSpaceRecord struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Config      json.RawMessage `json:"config"`
	CreatedBy   *string         `json:"created_by"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type desktopSpaceInput struct {
	ID     string          `json:"id"`
	Name   string          `json:"name"`
	Kind   string          `json:"kind"`
	Config json.RawMessage `json:"config"`
}

func (h *DesktopSpacesHandler) workspaceFromPath(c *gin.Context, userID string) (uuid.UUID, bool) {
	wsID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, false
	}
	var ok bool
	err = h.pool.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND status='active')`,
		wsID, userID).Scan(&ok)
	return wsID, err == nil && ok
}

func normalizeDesktopSpaceKind(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "personal":
		return "personal"
	case "team":
		return "team"
	case "workspace":
		return "workspace"
	default:
		return ""
	}
}

func desktopSpaceIDFromConfig(config json.RawMessage) (uuid.UUID, bool) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(config, &payload); err != nil {
		return uuid.Nil, false
	}
	var id string
	if raw, ok := payload["id"]; ok && json.Unmarshal(raw, &id) == nil {
		parsed, err := uuid.Parse(strings.TrimSpace(id))
		return parsed, err == nil
	}
	return uuid.Nil, false
}

func validateDesktopSpaceConfig(config json.RawMessage, id uuid.UUID, name string, kind string) (json.RawMessage, error) {
	if len(config) == 0 {
		config = json.RawMessage(`{}`)
	}
	if len(config) > desktopSpaceMaxConfigBytes {
		return nil, errDesktopSpaceConfig("desktop space config is too large")
	}
	if !json.Valid(config) {
		return nil, errDesktopSpaceConfig("invalid desktop space config")
	}
	var payload map[string]any
	if err := json.Unmarshal(config, &payload); err != nil || payload == nil {
		return nil, errDesktopSpaceConfig("desktop space config must be a JSON object")
	}
	payload["id"] = id.String()
	payload["name"] = name
	payload["kind"] = kind
	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, errDesktopSpaceConfig("invalid desktop space config")
	}
	return normalized, nil
}

type errDesktopSpaceConfig string

func (e errDesktopSpaceConfig) Error() string {
	return string(e)
}

func scanDesktopSpace(row pgx.Row) (desktopSpaceRecord, error) {
	var rec desktopSpaceRecord
	err := row.Scan(
		&rec.ID,
		&rec.WorkspaceID,
		&rec.Name,
		&rec.Kind,
		&rec.Config,
		&rec.CreatedBy,
		&rec.CreatedAt,
		&rec.UpdatedAt,
	)
	return rec, err
}

func (h *DesktopSpacesHandler) List(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromPath(c, user.ID)
	if !ok {
		c.JSON(403, gin.H{"error": "workspace access denied"})
		return
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id::text, workspace_id::text, name, kind, config, created_by, created_at, updated_at
		FROM workspace_desktop_spaces
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, name ASC
	`, wsID)
	if err != nil {
		RespondInternalErr(c, "list desktop spaces", err)
		return
	}
	defer rows.Close()

	spaces := make([]desktopSpaceRecord, 0)
	for rows.Next() {
		rec, scanErr := scanDesktopSpace(rows)
		if scanErr != nil {
			RespondInternalErr(c, "scan desktop space", scanErr)
			return
		}
		spaces = append(spaces, rec)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "list desktop spaces", err)
		return
	}
	c.JSON(200, gin.H{"spaces": spaces, "count": len(spaces)})
}

func (h *DesktopSpacesHandler) Save(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromPath(c, user.ID)
	if !ok {
		c.JSON(403, gin.H{"error": "workspace access denied"})
		return
	}

	var input desktopSpaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondBadRequestErr(c, "invalid desktop space payload")
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 160 {
		RespondBadRequestErr(c, "invalid desktop space name")
		return
	}
	kind := normalizeDesktopSpaceKind(input.Kind)
	if kind == "" {
		RespondBadRequestErr(c, "invalid desktop space kind")
		return
	}

	var id uuid.UUID
	var err error
	if strings.TrimSpace(input.ID) == "" {
		if configID, ok := desktopSpaceIDFromConfig(input.Config); ok {
			id = configID
		} else {
			id = uuid.New()
		}
	} else {
		id, err = uuid.Parse(input.ID)
		if err != nil {
			RespondBadRequestErr(c, "invalid desktop space id")
			return
		}
	}
	config, err := validateDesktopSpaceConfig(input.Config, id, name, kind)
	if err != nil {
		RespondBadRequestErr(c, err.Error())
		return
	}

	rec, err := scanDesktopSpace(h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO workspace_desktop_spaces (id, workspace_id, name, kind, config, created_by)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
		    kind = EXCLUDED.kind,
		    config = EXCLUDED.config,
		    updated_at = NOW()
		WHERE workspace_desktop_spaces.workspace_id = EXCLUDED.workspace_id
		RETURNING id::text, workspace_id::text, name, kind, config, created_by, created_at, updated_at
	`, id, wsID, name, kind, config, user.ID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(403, gin.H{"error": "workspace access denied"})
			return
		}
		RespondInternalErr(c, "save desktop space", err)
		return
	}
	globalDesktopPresenceHub.broadcastDesktopSpaceUpdated(wsID.String(), id.String(), rec.UpdatedAt.Format(time.RFC3339Nano))
	c.JSON(200, rec)
}

func (h *DesktopSpacesHandler) Delete(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromPath(c, user.ID)
	if !ok {
		c.JSON(403, gin.H{"error": "workspace access denied"})
		return
	}
	id, err := uuid.Parse(c.Param("desktopSpaceId"))
	if err != nil {
		RespondBadRequestErr(c, "invalid desktop space id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM workspace_desktop_spaces WHERE id=$1 AND workspace_id=$2`,
		id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete desktop space", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "desktop space")
		return
	}
	globalDesktopPresenceHub.broadcastDesktopSpaceDeleted(wsID.String(), id.String())
	c.JSON(200, gin.H{"ok": true})
}
