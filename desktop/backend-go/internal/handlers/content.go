package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/middleware"
)

// ContentHandler manages the per-workspace content pipeline: the posts, reels,
// newsletters, podcasts, threads and articles the business is producing, tracked
// through their lifecycle (idea -> draft -> scheduled -> published).
// Workspace-scoped via X-Workspace-ID.
type ContentHandler struct {
	pool *pgxpool.Pool
}

func NewContentHandler(pool *pgxpool.Pool) *ContentHandler {
	return &ContentHandler{pool: pool}
}

type contentItem struct {
	ID                  string    `json:"id"`
	Title               string    `json:"title"`
	ContentType         string    `json:"content_type"`
	Status              string    `json:"status"`
	Hook                string    `json:"hook"`
	Body                string    `json:"body"`
	Caption             string    `json:"caption"`
	CTA                 string    `json:"cta"`
	Channel             string    `json:"channel"`
	Link                string    `json:"link"`
	Category            string    `json:"category"`
	Theme               string    `json:"theme"`
	Client              string    `json:"client"`
	Campaign            string    `json:"campaign"`
	Owner               string    `json:"owner"`
	Editor              string    `json:"editor"`
	Priority            string    `json:"priority"`
	DueDate             string    `json:"due_date"`
	PublishDate         string    `json:"publish_date"`
	AssetLink           string    `json:"asset_link"`
	ReviewLink          string    `json:"review_link"`
	RevisionNotes       string    `json:"revision_notes"`
	Notes               string    `json:"notes"`
	Views               int       `json:"views"`
	Reach               int       `json:"reach"`
	Likes               int       `json:"likes"`
	Comments            int       `json:"comments"`
	Saves               int       `json:"saves"`
	Shares              int       `json:"shares"`
	Reposts             int       `json:"reposts"`
	Follows             int       `json:"follows"`
	ProfileActivity     int       `json:"profile_activity"`
	AccountsEngaged     int       `json:"accounts_engaged"`
	AvgWatchTimeSeconds float64   `json:"avg_watch_time_seconds"`
	RetentionRate       float64   `json:"retention_rate"`
	AnalyticsNotes      string    `json:"analytics_notes"`
	CreatedBy           *string   `json:"created_by"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// workspaceFromHeader resolves X-Workspace-ID and confirms active membership.
func (h *ContentHandler) workspaceFromHeader(c *gin.Context, userID string) (uuid.UUID, bool) {
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

// ListItems returns the workspace's content, optionally filtered by ?q=.
// GET /api/v1/content
func (h *ContentHandler) ListItems(c *gin.Context) {
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
	var rows interface {
		Next() bool
		Scan(...any) error
		Err() error
		Close()
	}
	var err error
	if q != "" {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, title, content_type, status, hook, body, caption, cta, channel, link,
			       category, theme, client, campaign, owner, editor, priority, due_date, publish_date, asset_link, review_link, revision_notes, notes,
			       views, reach, likes, comments, saves, shares, reposts, follows, profile_activity, accounts_engaged, avg_watch_time_seconds, retention_rate, analytics_notes,
			       created_by, created_at, updated_at
			FROM   content_items
			WHERE  workspace_id = $1
			AND   (title ILIKE '%'||$2||'%' OR hook ILIKE '%'||$2||'%' OR body ILIKE '%'||$2||'%' OR caption ILIKE '%'||$2||'%' OR cta ILIKE '%'||$2||'%' OR channel ILIKE '%'||$2||'%' OR link ILIKE '%'||$2||'%'
			       OR category ILIKE '%'||$2||'%' OR theme ILIKE '%'||$2||'%' OR client ILIKE '%'||$2||'%' OR campaign ILIKE '%'||$2||'%' OR owner ILIKE '%'||$2||'%' OR editor ILIKE '%'||$2||'%' OR revision_notes ILIKE '%'||$2||'%' OR analytics_notes ILIKE '%'||$2||'%')
			ORDER  BY updated_at DESC
		`, wsID, q)
	} else {
		rows, err = h.pool.Query(c.Request.Context(), `
			SELECT id, title, content_type, status, hook, body, caption, cta, channel, link,
			       category, theme, client, campaign, owner, editor, priority, due_date, publish_date, asset_link, review_link, revision_notes, notes,
			       views, reach, likes, comments, saves, shares, reposts, follows, profile_activity, accounts_engaged, avg_watch_time_seconds, retention_rate, analytics_notes,
			       created_by, created_at, updated_at
			FROM   content_items
			WHERE  workspace_id = $1
			ORDER  BY updated_at DESC
		`, wsID)
	}
	if err != nil {
		RespondInternalErr(c, "list content items", err)
		return
	}
	defer rows.Close()

	items := make([]contentItem, 0)
	for rows.Next() {
		var it contentItem
		var id uuid.UUID
		if err := rows.Scan(
			&id, &it.Title, &it.ContentType, &it.Status, &it.Hook, &it.Body, &it.Caption, &it.CTA, &it.Channel, &it.Link,
			&it.Category, &it.Theme, &it.Client, &it.Campaign, &it.Owner, &it.Editor, &it.Priority, &it.DueDate, &it.PublishDate, &it.AssetLink, &it.ReviewLink, &it.RevisionNotes, &it.Notes,
			&it.Views, &it.Reach, &it.Likes, &it.Comments, &it.Saves, &it.Shares, &it.Reposts, &it.Follows, &it.ProfileActivity, &it.AccountsEngaged, &it.AvgWatchTimeSeconds, &it.RetentionRate, &it.AnalyticsNotes,
			&it.CreatedBy, &it.CreatedAt, &it.UpdatedAt,
		); err != nil {
			RespondInternalErr(c, "scan content item", err)
			return
		}
		it.ID = id.String()
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		RespondInternalErr(c, "iterate content items", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items)})
}

type contentInput struct {
	Title               string  `json:"title"`
	ContentType         string  `json:"content_type"`
	Status              string  `json:"status"`
	Hook                string  `json:"hook"`
	Body                string  `json:"body"`
	Caption             string  `json:"caption"`
	CTA                 string  `json:"cta"`
	Channel             string  `json:"channel"`
	Link                string  `json:"link"`
	Category            string  `json:"category"`
	Theme               string  `json:"theme"`
	Client              string  `json:"client"`
	Campaign            string  `json:"campaign"`
	Owner               string  `json:"owner"`
	Editor              string  `json:"editor"`
	Priority            string  `json:"priority"`
	DueDate             string  `json:"due_date"`
	PublishDate         string  `json:"publish_date"`
	AssetLink           string  `json:"asset_link"`
	ReviewLink          string  `json:"review_link"`
	RevisionNotes       string  `json:"revision_notes"`
	Notes               string  `json:"notes"`
	Views               int     `json:"views"`
	Reach               int     `json:"reach"`
	Likes               int     `json:"likes"`
	Comments            int     `json:"comments"`
	Saves               int     `json:"saves"`
	Shares              int     `json:"shares"`
	Reposts             int     `json:"reposts"`
	Follows             int     `json:"follows"`
	ProfileActivity     int     `json:"profile_activity"`
	AccountsEngaged     int     `json:"accounts_engaged"`
	AvgWatchTimeSeconds float64 `json:"avg_watch_time_seconds"`
	RetentionRate       float64 `json:"retention_rate"`
	AnalyticsNotes      string  `json:"analytics_notes"`
}

// normContentType clamps to the allowed content formats, defaulting to "post".
// These are the fundamental content primitives: scripts, copy, carousels, reels,
// posts, newsletters, images, and long-form video/articles.
func normContentType(v string) string {
	switch strings.TrimSpace(v) {
	case "script", "copywriting", "carousel", "image", "video",
		"post", "reel", "story", "newsletter", "podcast", "thread", "article", "other":
		return strings.TrimSpace(v)
	default:
		return "post"
	}
}

// normStatus clamps to the allowed set, defaulting to "idea".
func normStatus(v string) string {
	switch strings.TrimSpace(v) {
	case "scripting_priority":
		return "scripting"
	case "revisions":
		return "to_edit"
	case "idea", "scripting", "to_film", "to_edit",
		"client_review", "approved", "to_post", "posted", "archive",
		"draft", "scheduled", "published":
		return strings.TrimSpace(v)
	default:
		return "idea"
	}
}

func normPriority(v string) string {
	switch strings.TrimSpace(v) {
	case "low", "normal", "high", "urgent":
		return strings.TrimSpace(v)
	default:
		return "normal"
	}
}

func nonNegativeInt(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func nonNegativeFloat(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

// CreateItem adds a content item. POST /api/v1/content
func (h *ContentHandler) CreateItem(c *gin.Context) {
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
	var in contentInput
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.Title) == "" {
		RespondBadRequestErr(c, "title is required")
		return
	}

	var it contentItem
	var id uuid.UUID
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO content_items (
			workspace_id, title, content_type, status, hook, body, caption, cta, channel, link,
			category, theme, client, campaign, owner, editor, priority, due_date, publish_date, asset_link, review_link, revision_notes, notes,
			views, reach, likes, comments, saves, shares, reposts, follows, profile_activity, accounts_engaged, avg_watch_time_seconds, retention_rate, analytics_notes,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37)
		RETURNING id, title, content_type, status, hook, body, caption, cta, channel, link,
		          category, theme, client, campaign, owner, editor, priority, due_date, publish_date, asset_link, review_link, revision_notes, notes,
		          views, reach, likes, comments, saves, shares, reposts, follows, profile_activity, accounts_engaged, avg_watch_time_seconds, retention_rate, analytics_notes,
		          created_by, created_at, updated_at
	`, wsID, strings.TrimSpace(in.Title), normContentType(in.ContentType), normStatus(in.Status), in.Hook, in.Body, in.Caption, in.CTA, in.Channel, in.Link,
		strings.TrimSpace(in.Category), strings.TrimSpace(in.Theme), strings.TrimSpace(in.Client), strings.TrimSpace(in.Campaign), strings.TrimSpace(in.Owner), strings.TrimSpace(in.Editor),
		normPriority(in.Priority), strings.TrimSpace(in.DueDate), strings.TrimSpace(in.PublishDate), strings.TrimSpace(in.AssetLink), strings.TrimSpace(in.ReviewLink), strings.TrimSpace(in.RevisionNotes), strings.TrimSpace(in.Notes),
		nonNegativeInt(in.Views), nonNegativeInt(in.Reach), nonNegativeInt(in.Likes), nonNegativeInt(in.Comments), nonNegativeInt(in.Saves), nonNegativeInt(in.Shares), nonNegativeInt(in.Reposts), nonNegativeInt(in.Follows), nonNegativeInt(in.ProfileActivity), nonNegativeInt(in.AccountsEngaged), nonNegativeFloat(in.AvgWatchTimeSeconds), nonNegativeFloat(in.RetentionRate), strings.TrimSpace(in.AnalyticsNotes), user.ID).
		Scan(
			&id, &it.Title, &it.ContentType, &it.Status, &it.Hook, &it.Body, &it.Caption, &it.CTA, &it.Channel, &it.Link,
			&it.Category, &it.Theme, &it.Client, &it.Campaign, &it.Owner, &it.Editor, &it.Priority, &it.DueDate, &it.PublishDate, &it.AssetLink, &it.ReviewLink, &it.RevisionNotes, &it.Notes,
			&it.Views, &it.Reach, &it.Likes, &it.Comments, &it.Saves, &it.Shares, &it.Reposts, &it.Follows, &it.ProfileActivity, &it.AccountsEngaged, &it.AvgWatchTimeSeconds, &it.RetentionRate, &it.AnalyticsNotes,
			&it.CreatedBy, &it.CreatedAt, &it.UpdatedAt,
		)
	if err != nil {
		RespondInternalErr(c, "create content item", err)
		return
	}
	it.ID = id.String()
	c.JSON(http.StatusCreated, it)
}

// UpdateItem edits a content item. PUT /api/v1/content/:id
func (h *ContentHandler) UpdateItem(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid item id")
		return
	}
	var in contentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		RespondBadRequestErr(c, "invalid request body")
		return
	}

	var it contentItem
	var rid uuid.UUID
	err = h.pool.QueryRow(c.Request.Context(), `
		UPDATE content_items
		SET    title        = COALESCE(NULLIF($3,''), title),
		       content_type = $4,
		       status       = $5,
		       hook         = $6,
		       body         = $7,
		       caption      = $8,
		       cta          = $9,
		       channel      = $10,
		       link         = $11,
		       category     = $12,
		       theme        = $13,
		       client       = $14,
		       campaign     = $15,
		       owner        = $16,
		       editor       = $17,
		       priority     = $18,
		       due_date     = $19,
		       publish_date = $20,
		       asset_link   = $21,
		       review_link  = $22,
		       revision_notes = $23,
		       notes        = $24,
		       views        = $25,
		       reach        = $26,
		       likes        = $27,
		       comments     = $28,
		       saves        = $29,
		       shares       = $30,
		       reposts      = $31,
		       follows      = $32,
		       profile_activity = $33,
		       accounts_engaged = $34,
		       avg_watch_time_seconds = $35,
		       retention_rate = $36,
		       analytics_notes = $37,
		       updated_at   = NOW()
		WHERE  id = $1 AND workspace_id = $2
		RETURNING id, title, content_type, status, hook, body, caption, cta, channel, link,
		          category, theme, client, campaign, owner, editor, priority, due_date, publish_date, asset_link, review_link, revision_notes, notes,
		          views, reach, likes, comments, saves, shares, reposts, follows, profile_activity, accounts_engaged, avg_watch_time_seconds, retention_rate, analytics_notes,
		          created_by, created_at, updated_at
	`, id, wsID, strings.TrimSpace(in.Title), normContentType(in.ContentType), normStatus(in.Status), in.Hook, in.Body, in.Caption, in.CTA, in.Channel, in.Link,
		strings.TrimSpace(in.Category), strings.TrimSpace(in.Theme), strings.TrimSpace(in.Client), strings.TrimSpace(in.Campaign), strings.TrimSpace(in.Owner), strings.TrimSpace(in.Editor),
		normPriority(in.Priority), strings.TrimSpace(in.DueDate), strings.TrimSpace(in.PublishDate), strings.TrimSpace(in.AssetLink), strings.TrimSpace(in.ReviewLink), strings.TrimSpace(in.RevisionNotes), strings.TrimSpace(in.Notes),
		nonNegativeInt(in.Views), nonNegativeInt(in.Reach), nonNegativeInt(in.Likes), nonNegativeInt(in.Comments), nonNegativeInt(in.Saves), nonNegativeInt(in.Shares), nonNegativeInt(in.Reposts), nonNegativeInt(in.Follows), nonNegativeInt(in.ProfileActivity), nonNegativeInt(in.AccountsEngaged), nonNegativeFloat(in.AvgWatchTimeSeconds), nonNegativeFloat(in.RetentionRate), strings.TrimSpace(in.AnalyticsNotes)).
		Scan(
			&rid, &it.Title, &it.ContentType, &it.Status, &it.Hook, &it.Body, &it.Caption, &it.CTA, &it.Channel, &it.Link,
			&it.Category, &it.Theme, &it.Client, &it.Campaign, &it.Owner, &it.Editor, &it.Priority, &it.DueDate, &it.PublishDate, &it.AssetLink, &it.ReviewLink, &it.RevisionNotes, &it.Notes,
			&it.Views, &it.Reach, &it.Likes, &it.Comments, &it.Saves, &it.Shares, &it.Reposts, &it.Follows, &it.ProfileActivity, &it.AccountsEngaged, &it.AvgWatchTimeSeconds, &it.RetentionRate, &it.AnalyticsNotes,
			&it.CreatedBy, &it.CreatedAt, &it.UpdatedAt,
		)
	if err != nil {
		RespondNotFoundErr(c, "item")
		return
	}
	it.ID = rid.String()
	c.JSON(http.StatusOK, it)
}

// DeleteItem removes a content item. DELETE /api/v1/content/:id
func (h *ContentHandler) DeleteItem(c *gin.Context) {
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
		RespondBadRequestErr(c, "invalid item id")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`DELETE FROM content_items WHERE id = $1 AND workspace_id = $2`, id, wsID)
	if err != nil {
		RespondInternalErr(c, "delete content item", err)
		return
	}
	if tag.RowsAffected() == 0 {
		RespondNotFoundErr(c, "item")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "item deleted"})
}
