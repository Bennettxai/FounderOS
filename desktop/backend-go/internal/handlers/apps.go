package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// AppsHandler manages workspace-owned URL apps that can be launched inside
// BusinessOS. These are separate from canonical modules and generated app infra.
type AppsHandler struct {
	pool *pgxpool.Pool
}

func NewAppsHandler(pool *pgxpool.Pool) *AppsHandler {
	return &AppsHandler{pool: pool}
}

type workspaceApp struct {
	ID            string    `json:"id"`
	Source        string    `json:"source"`
	SourceID      string    `json:"source_id"`
	CatalogAppID  *string   `json:"catalog_app_id"`
	Name          string    `json:"name"`
	AppType       string    `json:"app_type"`
	Provider      string    `json:"provider"`
	URL           string    `json:"url"`
	LaunchMode    string    `json:"launch_mode"`
	Status        string    `json:"status"`
	Icon          string    `json:"icon"`
	LogoURL       string    `json:"logo_url"`
	Color         string    `json:"color"`
	Category      string    `json:"category"`
	Notes         string    `json:"notes"`
	ShowOnDesktop bool      `json:"show_on_desktop"`
	ShowInDock    bool      `json:"show_in_dock"`
	PositionIndex int       `json:"position_index"`
	URLClass      string    `json:"url_class"`
	ReadOnly      bool      `json:"read_only"`
	CreatedBy     *string   `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type workspaceAppInput struct {
	Name          string `json:"name"`
	AppType       string `json:"app_type"`
	Provider      string `json:"provider"`
	URL           string `json:"url"`
	LaunchMode    string `json:"launch_mode"`
	Status        string `json:"status"`
	Icon          string `json:"icon"`
	LogoURL       string `json:"logo_url"`
	Color         string `json:"color"`
	Category      string `json:"category"`
	Notes         string `json:"notes"`
	ShowOnDesktop *bool  `json:"show_on_desktop"`
	ShowInDock    *bool  `json:"show_in_dock"`
	PositionIndex int    `json:"position_index"`
	URLClass      string `json:"url_class"`
}

type catalogApp struct {
	ID            string    `json:"id"`
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	AppType       string    `json:"app_type"`
	Provider      string    `json:"provider"`
	URL           string    `json:"url"`
	LaunchMode    string    `json:"launch_mode"`
	Icon          string    `json:"icon"`
	LogoURL       string    `json:"logo_url"`
	Color         string    `json:"color"`
	Category      string    `json:"category"`
	Notes         string    `json:"notes"`
	URLClass      string    `json:"url_class"`
	Status        string    `json:"status"`
	IsFeatured    bool      `json:"is_featured"`
	PositionIndex int       `json:"position_index"`
	Installed     bool      `json:"installed"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func normalizeAppType(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "mini_app":
		return "mini_app"
	case "internal_app":
		return "internal_app"
	case "client_app":
		return "client_app"
	case "embedded_tool":
		return "embedded_tool"
	default:
		return "web_app"
	}
}

func normalizeAppProvider(v string) string {
	value := strings.ToLower(strings.TrimSpace(v))
	if value == "" || len(value) > 60 {
		return "custom"
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "custom"
	}
	return value
}

func normalizeLaunchMode(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "browser":
		return "browser"
	case "external":
		return "external"
	default:
		return "iframe"
	}
}

func normalizeAppStatus(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "draft":
		return "draft"
	case "archived":
		return "archived"
	default:
		return "active"
	}
}

func normalizeURLClass(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "temporary_preview":
		return "temporary_preview"
	case "always_on_preview":
		return "always_on_preview"
	case "durable_deployment":
		return "durable_deployment"
	case "custom_domain":
		return "custom_domain"
	default:
		return "stable_sandbox_embed"
	}
}

func normalizeAppURL(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	if parsed.Host == "" {
		return "", false
	}
	return parsed.String(), true
}

func boolValue(v *bool, fallback bool) bool {
	if v == nil {
		return fallback
	}
	return *v
}

func isMissingAppSchema(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "42P01" || pgErr.Code == "42703"
	}
	return false
}

func normalizeOptionalAppURL(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	return normalizeAppURL(value)
}

func cleanAppInput(in workspaceAppInput) (workspaceAppInput, string) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return in, "name is required"
	}
	appURL, ok := normalizeAppURL(in.URL)
	if !ok {
		return in, "valid http or https url is required"
	}
	logoURL, ok := normalizeOptionalAppURL(in.LogoURL)
	if !ok {
		return in, "valid http or https logo url is required"
	}
	icon := strings.TrimSpace(in.Icon)
	if icon == "" {
		icon = "layout-grid"
	}
	color := strings.TrimSpace(in.Color)
	if color == "" {
		color = "#111827"
	}
	category := strings.TrimSpace(in.Category)
	if category == "" {
		category = "general"
	}
	in.Name = name
	in.URL = appURL
	in.LogoURL = logoURL
	in.AppType = normalizeAppType(in.AppType)
	in.Provider = normalizeAppProvider(in.Provider)
	in.LaunchMode = normalizeLaunchMode(in.LaunchMode)
	in.Status = normalizeAppStatus(in.Status)
	in.URLClass = normalizeURLClass(in.URLClass)
	in.Icon = icon
	in.Color = color
	in.Category = category
	in.Notes = strings.TrimSpace(in.Notes)
	return in, ""
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *AppsHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

func scanWorkspaceApp(row pgx.Row) (workspaceApp, error) {
	var app workspaceApp
	var id uuid.UUID
	var catalogID *uuid.UUID
	err := row.Scan(
		&id,
		&catalogID,
		&app.Name,
		&app.AppType,
		&app.Provider,
		&app.URL,
		&app.LaunchMode,
		&app.Status,
		&app.Icon,
		&app.LogoURL,
		&app.Color,
		&app.Category,
		&app.Notes,
		&app.ShowOnDesktop,
		&app.ShowInDock,
		&app.PositionIndex,
		&app.URLClass,
		&app.CreatedBy,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	app.ID = id.String()
	if catalogID != nil {
		value := catalogID.String()
		app.CatalogAppID = &value
	}
	app.Source = "manual"
	app.SourceID = app.ID
	app.ReadOnly = false
	return app, err
}

func scanCatalogApp(row pgx.Row) (catalogApp, error) {
	var app catalogApp
	var id uuid.UUID
	err := row.Scan(
		&id,
		&app.Slug,
		&app.Name,
		&app.AppType,
		&app.Provider,
		&app.URL,
		&app.LaunchMode,
		&app.Icon,
		&app.LogoURL,
		&app.Color,
		&app.Category,
		&app.Notes,
		&app.URLClass,
		&app.Status,
		&app.IsFeatured,
		&app.PositionIndex,
		&app.Installed,
		&app.CreatedAt,
		&app.UpdatedAt,
	)
	app.ID = id.String()
	return app, err
}

func (h *AppsHandler) listGeneratedApps(c *gin.Context, wsID uuid.UUID, q string, desktopOnly bool) ([]workspaceApp, error) {
	var exists bool
	if err := h.pool.QueryRow(c.Request.Context(), `SELECT to_regclass('public.user_generated_apps') IS NOT NULL`).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return []workspaceApp{}, nil
	}

	args := []any{wsID}
	where := []string{"workspace_id = $1", "is_visible = TRUE"}
	if q != "" {
		args = append(args, q)
		where = append(where, "(app_name ILIKE '%'||$2||'%' OR custom_config::text ILIKE '%'||$2||'%')")
	}
	if desktopOnly {
		where = append(where, "is_pinned = TRUE")
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id,
		       app_name,
		       COALESCE(custom_config->>'url', custom_config->>'deployment_url', custom_config->>'app_url', custom_config->>'preview_url', '') AS url,
		       COALESCE(custom_config->>'logo_url', custom_config->>'icon_url', custom_icon, '') AS logo_url,
		       COALESCE(custom_config->>'category', 'generated') AS category,
		       COALESCE(custom_config->>'description', '') AS notes,
		       is_pinned,
		       is_pinned,
		       COALESCE(position_index, 0),
		       generated_at,
		       updated_at
		FROM user_generated_apps
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY is_pinned DESC, position_index ASC NULLS LAST, generated_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	apps := make([]workspaceApp, 0)
	for rows.Next() {
		var id uuid.UUID
		var app workspaceApp
		if err := rows.Scan(
			&id,
			&app.Name,
			&app.URL,
			&app.LogoURL,
			&app.Category,
			&app.Notes,
			&app.ShowOnDesktop,
			&app.ShowInDock,
			&app.PositionIndex,
			&app.CreatedAt,
			&app.UpdatedAt,
		); err != nil {
			return nil, err
		}
		app.ID = "generated:" + id.String()
		app.Source = "generated"
		app.SourceID = id.String()
		app.AppType = "mini_app"
		app.Provider = "miosa"
		app.LaunchMode = "iframe"
		app.Status = "active"
		app.Icon = "app-window"
		app.Color = "#111827"
		app.URLClass = "durable_deployment"
		app.ReadOnly = true
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return apps, nil
}

// ListApps returns the workspace's URL apps, optionally filtered by q.
// GET /api/v1/apps
func (h *AppsHandler) ListApps(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	desktopOnly := c.Query("desktop") == "true"
	args := []any{wsID}
	where := []string{"workspace_id = $1"}
	if q != "" {
		args = append(args, q)
		where = append(where, "(name ILIKE '%'||$2||'%' OR provider ILIKE '%'||$2||'%' OR url ILIKE '%'||$2||'%' OR category ILIKE '%'||$2||'%' OR notes ILIKE '%'||$2||'%')")
	}
	if desktopOnly {
		where = append(where, "show_on_desktop = TRUE")
	}

	rows, err := h.pool.Query(c.Request.Context(), `
			SELECT id, catalog_app_id, name, app_type, provider, url, launch_mode, status, icon, logo_url, color, category, notes,
			       show_on_desktop, show_in_dock, position_index, url_class, created_by, created_at, updated_at
		FROM workspace_apps
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY position_index ASC, name ASC
	`, args...)
	if err != nil {
		if isMissingAppSchema(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "apps schema not migrated"})
			return
		}
		RespondInternalErr(c, "list apps", err)
		return
	}
	defer rows.Close()

	apps := make([]workspaceApp, 0)
	for rows.Next() {
		app, err := scanWorkspaceApp(rows)
		if err != nil {
			RespondInternalErr(c, "scan app", err)
			return
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate apps", err)
		return
	}

	generatedApps, err := h.listGeneratedApps(c, wsID, q, desktopOnly)
	if err != nil {
		if isMissingAppSchema(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "apps schema not migrated"})
			return
		}
		RespondInternalErr(c, "list generated apps", err)
		return
	}
	apps = append(apps, generatedApps...)

	c.JSON(http.StatusOK, gin.H{"apps": apps, "count": len(apps)})
}

// CreateApp adds a workspace URL app. POST /api/v1/apps
func (h *AppsHandler) CreateApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	var in workspaceAppInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}
	cleaned, msg := cleanAppInput(in)
	if msg != "" {
		RespondBadRequestErr(c, msg)
		return
	}

	app, err := scanWorkspaceApp(h.pool.QueryRow(c.Request.Context(), `
			INSERT INTO workspace_apps (
				workspace_id, name, app_type, provider, url, launch_mode, status, icon, logo_url, color, category,
				notes, show_on_desktop, show_in_dock, position_index, url_class, created_by
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			RETURNING id, catalog_app_id, name, app_type, provider, url, launch_mode, status, icon, logo_url, color, category, notes,
			          show_on_desktop, show_in_dock, position_index, url_class, created_by, created_at, updated_at
		`, wsID, cleaned.Name, cleaned.AppType, cleaned.Provider, cleaned.URL, cleaned.LaunchMode, cleaned.Status, cleaned.Icon,
		cleaned.LogoURL, cleaned.Color, cleaned.Category, cleaned.Notes, boolValue(cleaned.ShowOnDesktop, true), boolValue(cleaned.ShowInDock, true),
		cleaned.PositionIndex, cleaned.URLClass, user.ID))
	if err != nil {
		if isMissingAppSchema(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "apps schema not migrated"})
			return
		}
		RespondInternalErr(c, "create app", err)
		return
	}
	c.JSON(http.StatusCreated, app)
}

// UpdateApp edits a workspace URL app. PUT /api/v1/apps/:id
func (h *AppsHandler) UpdateApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid app id")
		return
	}
	var in workspaceAppInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}
	cleaned, msg := cleanAppInput(in)
	if msg != "" {
		RespondBadRequestErr(c, msg)
		return
	}

	app, err := scanWorkspaceApp(h.pool.QueryRow(c.Request.Context(), `
		UPDATE workspace_apps
		SET name = $3,
			    app_type = $4,
			    provider = $5,
			    url = $6,
			    launch_mode = $7,
			    status = $8,
			    icon = $9,
			    logo_url = $10,
			    color = $11,
			    category = $12,
			    notes = $13,
			    show_on_desktop = $14,
			    show_in_dock = $15,
			    position_index = $16,
			    url_class = $17,
			    updated_at = NOW()
			WHERE id = $1 AND workspace_id = $2
				RETURNING id, catalog_app_id, name, app_type, provider, url, launch_mode, status, icon, logo_url, color, category, notes,
				          show_on_desktop, show_in_dock, position_index, url_class, created_by, created_at, updated_at
		`, id, wsID, cleaned.Name, cleaned.AppType, cleaned.Provider, cleaned.URL, cleaned.LaunchMode, cleaned.Status, cleaned.Icon,
		cleaned.LogoURL, cleaned.Color, cleaned.Category, cleaned.Notes, boolValue(cleaned.ShowOnDesktop, true), boolValue(cleaned.ShowInDock, true),
		cleaned.PositionIndex, cleaned.URLClass))
	if err != nil {
		if isMissingAppSchema(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "apps schema not migrated"})
			return
		}
		RespondNotFoundErr(c, "app")
		return
	}
	c.JSON(http.StatusOK, app)
}

// DeleteApp removes a workspace URL app. DELETE /api/v1/apps/:id
func (h *AppsHandler) DeleteApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid app id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM workspace_apps WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete app", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "app")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "app deleted"})
}

// ListCatalog returns platform catalog apps with workspace install state.
// GET /api/v1/apps/catalog
func (h *AppsHandler) ListCatalog(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	args := []any{wsID}
	where := []string{"ac.status = 'active'"}
	if q != "" {
		args = append(args, q)
		where = append(where, "(ac.name ILIKE '%'||$2||'%' OR ac.provider ILIKE '%'||$2||'%' OR ac.category ILIKE '%'||$2||'%' OR ac.notes ILIKE '%'||$2||'%')")
	}

	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT ac.id, ac.slug, ac.name, ac.app_type, ac.provider, ac.url, ac.launch_mode,
		       ac.icon, ac.logo_url, ac.color, ac.category, ac.notes, ac.url_class, ac.status,
		       ac.is_featured, ac.position_index,
		       EXISTS (
		           SELECT 1 FROM workspace_apps wa
		           WHERE wa.workspace_id = $1
		             AND (wa.catalog_app_id = ac.id OR (wa.url <> '' AND wa.url = ac.url))
		       ) AS installed,
		       ac.created_at, ac.updated_at
		FROM app_catalog ac
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY ac.is_featured DESC, ac.position_index ASC, ac.name ASC
	`, args...)
	if err != nil {
		if isMissingAppSchema(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "apps schema not migrated"})
			return
		}
		RespondInternalErr(c, "list app catalog", err)
		return
	}
	defer rows.Close()

	apps := make([]catalogApp, 0)
	for rows.Next() {
		app, err := scanCatalogApp(rows)
		if err != nil {
			RespondInternalErr(c, "scan app catalog", err)
			return
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate app catalog", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"apps": apps, "count": len(apps)})
}

// InstallCatalogApp copies a catalog app into the current workspace.
// POST /api/v1/apps/catalog/:id/install
func (h *AppsHandler) InstallCatalogApp(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	wsID, ok := h.workspaceFromHeader(c, user.ID)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "select a workspace first"})
		return
	}
	catalogID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondBadRequestErr(c, "invalid catalog app id")
		return
	}

	existing, err := scanWorkspaceApp(h.pool.QueryRow(c.Request.Context(), `
		SELECT wa.id, wa.catalog_app_id, wa.name, wa.app_type, wa.provider, wa.url, wa.launch_mode, wa.status, wa.icon, wa.logo_url, wa.color, wa.category, wa.notes,
		       wa.show_on_desktop, wa.show_in_dock, wa.position_index, wa.url_class, wa.created_by, wa.created_at, wa.updated_at
		FROM workspace_apps wa
		JOIN app_catalog ac ON ac.id = $2
		WHERE wa.workspace_id = $1
		  AND (wa.catalog_app_id = ac.id OR (wa.url <> '' AND wa.url = ac.url))
		LIMIT 1
	`, wsID, catalogID))
	if err == nil {
		updated, updateErr := scanWorkspaceApp(h.pool.QueryRow(c.Request.Context(), `
			UPDATE workspace_apps wa
			SET catalog_app_id = ac.id,
			    name = ac.name,
			    app_type = ac.app_type,
			    provider = ac.provider,
			    url = ac.url,
			    launch_mode = ac.launch_mode,
			    status = 'active',
			    icon = ac.icon,
			    logo_url = ac.logo_url,
			    color = ac.color,
			    category = ac.category,
			    notes = ac.notes,
			    show_on_desktop = TRUE,
			    show_in_dock = TRUE,
			    url_class = ac.url_class,
			    updated_at = NOW()
			FROM app_catalog ac
			WHERE wa.id = $1
			  AND wa.workspace_id = $2
			  AND ac.id = $3
			RETURNING wa.id, wa.catalog_app_id, wa.name, wa.app_type, wa.provider, wa.url, wa.launch_mode, wa.status, wa.icon, wa.logo_url, wa.color, wa.category, wa.notes,
			          wa.show_on_desktop, wa.show_in_dock, wa.position_index, wa.url_class, wa.created_by, wa.created_at, wa.updated_at
		`, existing.ID, wsID, catalogID))
		if updateErr != nil {
			RespondInternalErr(c, "refresh installed app", updateErr)
			return
		}
		c.JSON(http.StatusOK, updated)
		return
	}
	if err != pgx.ErrNoRows {
		if isMissingAppSchema(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "apps schema not migrated"})
			return
		}
		RespondInternalErr(c, "check installed app", err)
		return
	}

	app, err := scanWorkspaceApp(h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO workspace_apps (
			workspace_id, catalog_app_id, name, app_type, provider, url, launch_mode, status,
			icon, logo_url, color, category, notes, show_on_desktop, show_in_dock,
			position_index, url_class, created_by
		)
		SELECT $1, id, name, app_type, provider, url, launch_mode, 'active',
		       icon, logo_url, color, category, notes, TRUE, TRUE,
		       COALESCE((SELECT MAX(position_index) + 1 FROM workspace_apps WHERE workspace_id = $1), 0),
		       url_class, $3
		FROM app_catalog
		WHERE id = $2 AND status = 'active'
		RETURNING id, catalog_app_id, name, app_type, provider, url, launch_mode, status, icon, logo_url, color, category, notes,
		          show_on_desktop, show_in_dock, position_index, url_class, created_by, created_at, updated_at
	`, wsID, catalogID, user.ID))
	if err != nil {
		if isMissingAppSchema(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "apps schema not migrated"})
			return
		}
		RespondNotFoundErr(c, "catalog app")
		return
	}
	c.JSON(http.StatusCreated, app)
}
