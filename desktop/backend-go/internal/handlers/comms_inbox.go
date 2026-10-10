package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/services"
)

// ============================================================================
// COMMS INBOX HANDLER — unified Gmail + Outlook inbox
// ============================================================================
//
// Surfaces a single normalized email shape across providers so the email
// tab can render either through the same components. Frontend contract
// lives in frontend/src/lib/api/comms/types.ts (UnifiedEmail).
//
// Wave-2 scope:
//   GET /api/comms/inbox        — unified email list across gmail+outlook
//   GET /api/comms/contacts/search — merged contact suggestions for compose
//
// Known Wave-2 limitations (documented in PR for follow-up):
//   - Outlook folder filtering is best-effort: we don't yet sync mail
//     folders, so we can't map folder=inbox → Microsoft folder GUID. For
//     now, every folder filter returns all non-trash Outlook rows. Real
//     mapping requires a one-shot mailFolders sync (Wave 3).
//   - Outlook attachments stay empty []. Schema has the column; the
//     existing outlook_mail.go sync path doesn't populate it.
//   - Gmail-specific is_starred / is_important / labels surface as
//     Outlook flag_status / importance / categories. The mapping is
//     documented in the row builders below.

// commsHandler owns the /api/comms/* routes. Constructed once in
// registerIntegrationRoutes alongside the integration routes.
//
// broadcaster is the per-user SSE fan-out shared with every webhook
// handler that publishes realtime events; see comms_stream.go for the
// event contract. Pass nil to disable realtime — the typed publishers
// no-op gracefully so feature gating doesn't require an extra check.
type commsHandler struct {
	pool        *pgxpool.Pool
	broadcaster *services.SSEBroadcaster
	whatsapp    *whatsappStore
	engineSync  *services.EngineSync
}

func newCommsHandler(pool *pgxpool.Pool, broadcaster *services.SSEBroadcaster, engineSync *services.EngineSync) *commsHandler {
	return &commsHandler{pool: pool, broadcaster: broadcaster, whatsapp: newWhatsAppStore(), engineSync: engineSync}
}

// RegisterRoutes wires every /api/comms/* endpoint.
func (h *commsHandler) RegisterRoutes(api *gin.RouterGroup, auth gin.HandlerFunc) {
	comms := api.Group("/comms")
	comms.Use(auth)
	{
		comms.GET("/inbox", h.GetInbox)
		comms.GET("/contacts/search", h.SearchContacts)
		// Realtime SSE — see comms_stream.go.
		comms.GET("/stream", h.Stream)
		comms.GET("/routes", h.ListRoutes)
		comms.PUT("/routes", h.UpsertRoute)
		comms.DELETE("/routes", h.DeleteRoute)
		comms.POST("/routes/sync", h.SyncRoutes)
		// Channels + messages — see comms_channels.go.
		h.registerChannelRoutes(comms)
	}
}

// ----------------------------------------------------------------------------
// Unified email shape — must match frontend/src/lib/api/comms/types.ts
// (UnifiedEmail = Omit<Email, "provider"> & { provider: "gmail" | "outlook" }).
// ----------------------------------------------------------------------------

type unifiedEmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type unifiedAttachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

type unifiedEmail struct {
	ID          string                `json:"id"`
	UserID      string                `json:"user_id"`
	Provider    string                `json:"provider"` // "gmail" | "outlook"
	ExternalID  string                `json:"external_id"`
	ThreadID    string                `json:"thread_id,omitempty"`
	Subject     string                `json:"subject"`
	Snippet     string                `json:"snippet"`
	FromEmail   string                `json:"from_email"`
	FromName    string                `json:"from_name,omitempty"`
	ToEmails    []unifiedEmailAddress `json:"to_emails"`
	CcEmails    []unifiedEmailAddress `json:"cc_emails,omitempty"`
	BccEmails   []unifiedEmailAddress `json:"bcc_emails,omitempty"`
	ReplyTo     string                `json:"reply_to,omitempty"`
	BodyText    string                `json:"body_text,omitempty"`
	BodyHTML    string                `json:"body_html,omitempty"`
	Attachments []unifiedAttachment   `json:"attachments,omitempty"`
	IsRead      bool                  `json:"is_read"`
	IsStarred   bool                  `json:"is_starred"`
	IsImportant bool                  `json:"is_important"`
	IsDraft     bool                  `json:"is_draft"`
	IsSent      bool                  `json:"is_sent"`
	IsArchived  bool                  `json:"is_archived"`
	IsTrash     bool                  `json:"is_trash"`
	Labels      []string              `json:"labels"`
	Date        time.Time             `json:"date"`
	ReceivedAt  *time.Time            `json:"received_at,omitempty"`
}

type unifiedInboxResponse struct {
	Emails  []unifiedEmail `json:"emails"`
	Total   int            `json:"total"`
	HasMore bool           `json:"has_more"`
}

// ----------------------------------------------------------------------------
// Handler
// ----------------------------------------------------------------------------

// GetInbox returns the merged Gmail+Outlook inbox.
//
// Query params:
//
//	providers — comma-list, "gmail,outlook" (default: both)
//	folder    — inbox|sent|drafts|starred|archive|trash (default: inbox)
//	limit     — page size, default 50
//	offset    — page offset, default 0
//	q         — free-text search across subject + body + sender (optional)
//
// Response: { emails: UnifiedEmail[], total: int, has_more: bool }.
//
// Returns 200 with empty list when neither provider has data — frontend's
// probe-and-cache logic in inbox.ts treats 404 as "endpoint absent" and
// falls back to provider-specific calls; we MUST 200 to be picked up.
func (h *commsHandler) GetInbox(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	providers := parseProviders(c.Query("providers"))
	folder := strings.ToLower(c.DefaultQuery("folder", "inbox"))
	limit := parseIntDefault(c.Query("limit"), 50, 200)
	offset := parseIntDefault(c.Query("offset"), 0, 0)
	q := strings.TrimSpace(c.Query("q"))

	ctx := c.Request.Context()
	workspaceID := strings.TrimSpace(c.GetHeader("X-Workspace-ID"))
	var routeIndex communicationRouteIndex
	if workspaceID != "" {
		if _, err := uuid.Parse(workspaceID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid X-Workspace-ID header"})
			return
		}
		var member bool
		if err := h.pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM workspace_members
			WHERE workspace_id=$1::uuid AND user_id=$2 AND status='active')
		`, workspaceID, userID).Scan(&member); err != nil || !member {
			c.JSON(http.StatusForbidden, gin.H{"error": "active workspace membership required"})
			return
		}
		routeIndex = loadCommunicationRouteIndex(ctx, h.pool, userID)
	}
	var rows []unifiedEmail

	// We over-fetch (limit+offset combined cap) per provider so we can
	// merge-sort by date, then page. Cap per-provider fetch to keep the
	// hot path bounded — at limit=50/offset=200 we still pull ≤ 250 from
	// each side.
	perProviderCap := limit + offset
	if perProviderCap < 50 {
		perProviderCap = 50
	}

	if providers["gmail"] {
		gmailRows, err := h.fetchGmailRows(ctx, userID, folder, q, perProviderCap)
		if err != nil {
			slog.Info("comms inbox: gmail fetch failed", "error", err)
		} else {
			rows = append(rows, gmailRows...)
		}
	}

	if providers["outlook"] {
		outlookRows, err := h.fetchOutlookRows(ctx, userID, folder, q, perProviderCap)
		if err != nil {
			slog.Info("comms inbox: outlook fetch failed", "error", err)
		} else {
			rows = append(rows, outlookRows...)
		}
	}

	if workspaceID != "" {
		rows = filterUnifiedEmailsForWorkspace(rows, routeIndex, workspaceID)
	}

	// Sort merged set by date desc.
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].Date.After(rows[j].Date)
	})

	total := len(rows)
	end := offset + limit
	if offset > total {
		offset = total
	}
	if end > total {
		end = total
	}
	pageRows := rows[offset:end]

	c.JSON(http.StatusOK, unifiedInboxResponse{
		Emails:  pageRows,
		Total:   total,
		HasMore: end < total,
	})
}

func filterUnifiedEmailsForWorkspace(rows []unifiedEmail, routes communicationRouteIndex, workspaceID string) []unifiedEmail {
	filtered := make([]unifiedEmail, 0, len(rows))
	for _, email := range rows {
		conversationID := email.ThreadID
		if conversationID == "" {
			conversationID = email.ExternalID
		}
		route, ok := routes.resolve(email.Provider, conversationID)
		if ok && route.WorkspaceID == workspaceID {
			filtered = append(filtered, email)
		}
	}
	return filtered
}

// ----------------------------------------------------------------------------
// Gmail row builder — direct mapping from `emails` table.
// ----------------------------------------------------------------------------

func (h *commsHandler) fetchGmailRows(ctx context.Context, userID, folder, q string, fetchLimit int) ([]unifiedEmail, error) {
	folderClause, folderArgs := gmailFolderClause(folder)
	searchClause, searchArgs := gmailSearchClause(q, len(folderArgs)+2)

	args := []interface{}{userID}
	args = append(args, folderArgs...)
	args = append(args, searchArgs...)
	args = append(args, fetchLimit)

	limitPlaceholder := "$" + strconv.Itoa(len(args))

	query := `
		SELECT id, user_id, external_id, thread_id,
			subject, snippet, from_email, from_name,
			to_emails, cc_emails, reply_to,
			body_text, body_html, attachments,
			is_read, is_starred, is_important, is_draft, is_sent, is_archived, is_trash,
			labels, date, received_at
		FROM emails
		WHERE user_id = $1 AND provider = 'gmail'` +
		folderClause + searchClause +
		` ORDER BY date DESC NULLS LAST LIMIT ` + limitPlaceholder

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []unifiedEmail
	for rows.Next() {
		var (
			e          unifiedEmail
			threadID   *string
			snippet    *string
			fromName   *string
			replyTo    *string
			bodyText   *string
			bodyHTML   *string
			toRaw      []byte
			ccRaw      []byte
			attachRaw  []byte
			labels     []string
			date       *time.Time
			receivedAt *time.Time
		)
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.ExternalID, &threadID,
			&e.Subject, &snippet, &e.FromEmail, &fromName,
			&toRaw, &ccRaw, &replyTo,
			&bodyText, &bodyHTML, &attachRaw,
			&e.IsRead, &e.IsStarred, &e.IsImportant, &e.IsDraft, &e.IsSent, &e.IsArchived, &e.IsTrash,
			&labels, &date, &receivedAt,
		); err != nil {
			return nil, err
		}
		e.Provider = "gmail"
		if threadID != nil {
			e.ThreadID = *threadID
		}
		if snippet != nil {
			e.Snippet = *snippet
		}
		if fromName != nil {
			e.FromName = *fromName
		}
		if replyTo != nil {
			e.ReplyTo = *replyTo
		}
		if bodyText != nil {
			e.BodyText = *bodyText
		}
		if bodyHTML != nil {
			e.BodyHTML = *bodyHTML
		}
		e.ToEmails = decodeAddresses(toRaw)
		e.CcEmails = decodeAddresses(ccRaw)
		e.Attachments = decodeAttachments(attachRaw)
		e.Labels = labels
		if date != nil {
			e.Date = *date
		}
		e.ReceivedAt = receivedAt
		out = append(out, e)
	}
	return out, rows.Err()
}

// ----------------------------------------------------------------------------
// Outlook row builder — maps microsoft_mail_messages → unified shape.
// ----------------------------------------------------------------------------

func (h *commsHandler) fetchOutlookRows(ctx context.Context, userID, folder, q string, fetchLimit int) ([]unifiedEmail, error) {
	// Outlook folder mapping is best-effort (see file-header limitations).
	// Trash is the only folder we can detect today — when the user explicitly
	// asks for trash we filter by !is_draft semantics; for everything else we
	// surface every non-trash row and let client-side filters do the rest.
	folderClause := ""
	switch folder {
	case "drafts":
		folderClause = " AND is_draft = TRUE"
	case "sent":
		// folder_id semantics not synced; surface nothing for sent in Wave 2
		// rather than misleading rows. Frontend will show empty state.
		return nil, nil
	}

	searchClause, searchArgs := outlookSearchClause(q, 2)
	args := []interface{}{userID}
	args = append(args, searchArgs...)
	args = append(args, fetchLimit)
	limitPlaceholder := "$" + strconv.Itoa(len(args))

	query := `
		SELECT id, user_id, message_id, conversation_id,
			subject, body_preview, body_content, body_content_type, importance,
			from_email, from_name, to_recipients, cc_recipients,
			is_read, is_draft, has_attachments, folder_id, categories, flag_status,
			received_datetime, sent_datetime
		FROM microsoft_mail_messages
		WHERE user_id = $1` + folderClause + searchClause +
		` ORDER BY received_datetime DESC NULLS LAST LIMIT ` + limitPlaceholder

	rows, err := h.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []unifiedEmail
	for rows.Next() {
		var (
			e               unifiedEmail
			conversationID  *string
			bodyPreview     *string
			bodyContent     *string
			bodyContentType *string
			importance      *string
			fromName        *string
			toRaw           []byte
			ccRaw           []byte
			folderID        *string
			categoriesRaw   []byte
			flagStatus      *string
			receivedAt      *time.Time
			sentAt          *time.Time
			hasAttachments  bool
		)
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.ExternalID, &conversationID,
			&e.Subject, &bodyPreview, &bodyContent, &bodyContentType, &importance,
			&e.FromEmail, &fromName, &toRaw, &ccRaw,
			&e.IsRead, &e.IsDraft, &hasAttachments, &folderID, &categoriesRaw, &flagStatus,
			&receivedAt, &sentAt,
		); err != nil {
			return nil, err
		}

		e.Provider = "outlook"
		if conversationID != nil {
			e.ThreadID = *conversationID
		}
		if bodyPreview != nil {
			e.Snippet = *bodyPreview
		}
		if fromName != nil {
			e.FromName = *fromName
		}
		// body_content carries either text or html depending on content_type.
		if bodyContent != nil && *bodyContent != "" {
			ct := ""
			if bodyContentType != nil {
				ct = strings.ToLower(*bodyContentType)
			}
			if ct == "html" {
				e.BodyHTML = *bodyContent
			} else {
				e.BodyText = *bodyContent
			}
		}
		e.ToEmails = decodeAddresses(toRaw)
		e.CcEmails = decodeAddresses(ccRaw)
		e.Labels = decodeStringArray(categoriesRaw)
		// Map Outlook semantics into the unified Gmail-shaped flags.
		if flagStatus != nil && *flagStatus == "flagged" {
			e.IsStarred = true
		}
		if importance != nil && *importance == "high" {
			e.IsImportant = true
		}
		// Surface the attachments-present indicator without manufacturing
		// fake attachment metadata. UI uses len(attachments) > 0; we
		// preserve the bool by making attachments a zero-cost placeholder.
		if hasAttachments {
			e.Attachments = []unifiedAttachment{{Filename: "(attachment)"}}
		}
		if receivedAt != nil {
			e.Date = *receivedAt
			e.ReceivedAt = receivedAt
		} else if sentAt != nil {
			e.Date = *sentAt
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// parseProviders accepts "gmail,outlook" or "" (= both). Returns a set.
func parseProviders(raw string) map[string]bool {
	out := map[string]bool{}
	if raw == "" {
		out["gmail"] = true
		out["outlook"] = true
		return out
	}
	for _, p := range strings.Split(raw, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "gmail" || p == "outlook" {
			out[p] = true
		}
	}
	if len(out) == 0 {
		// Unknown providers — degrade to both rather than empty.
		out["gmail"] = true
		out["outlook"] = true
	}
	return out
}

// parseIntDefault parses an integer, falling back to def. minClamp clamps the
// minimum (use 0 to disable). Negative inputs always return def.
func parseIntDefault(raw string, def, minClamp int) int {
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return def
	}
	if n < minClamp {
		return minClamp
	}
	return n
}

// gmailFolderClause translates the unified folder name into a SQL fragment
// against the `emails` table. Mirrors gmail_read.go:174-187 to keep folder
// semantics aligned across the unified inbox and the legacy single-provider
// list endpoint.
func gmailFolderClause(folder string) (string, []interface{}) {
	switch folder {
	case "inbox":
		return " AND is_archived = FALSE AND is_trash = FALSE AND is_draft = FALSE", nil
	case "sent":
		return " AND is_sent = TRUE", nil
	case "drafts":
		return " AND is_draft = TRUE", nil
	case "starred":
		return " AND is_starred = TRUE", nil
	case "archive":
		return " AND is_archived = TRUE", nil
	case "trash":
		return " AND is_trash = TRUE", nil
	default:
		return "", nil
	}
}

// gmailSearchClause builds an ILIKE filter across subject/body/sender. The
// startIndex is the next $N positional parameter to use.
func gmailSearchClause(q string, startIndex int) (string, []interface{}) {
	if q == "" {
		return "", nil
	}
	pat := "%" + q + "%"
	clause := " AND (subject ILIKE $" + strconv.Itoa(startIndex) +
		" OR snippet ILIKE $" + strconv.Itoa(startIndex) +
		" OR body_text ILIKE $" + strconv.Itoa(startIndex) +
		" OR from_email ILIKE $" + strconv.Itoa(startIndex) +
		" OR from_name ILIKE $" + strconv.Itoa(startIndex) + ")"
	return clause, []interface{}{pat}
}

// outlookSearchClause builds an ILIKE filter against microsoft_mail_messages.
func outlookSearchClause(q string, startIndex int) (string, []interface{}) {
	if q == "" {
		return "", nil
	}
	pat := "%" + q + "%"
	clause := " AND (subject ILIKE $" + strconv.Itoa(startIndex) +
		" OR body_preview ILIKE $" + strconv.Itoa(startIndex) +
		" OR body_content ILIKE $" + strconv.Itoa(startIndex) +
		" OR from_email ILIKE $" + strconv.Itoa(startIndex) +
		" OR from_name ILIKE $" + strconv.Itoa(startIndex) + ")"
	return clause, []interface{}{pat}
}

// decodeAddresses parses a JSONB array of {email,name} into the unified
// shape. Missing/invalid → empty slice.
func decodeAddresses(raw []byte) []unifiedEmailAddress {
	if len(raw) == 0 {
		return nil
	}
	var addrs []unifiedEmailAddress
	if err := json.Unmarshal(raw, &addrs); err == nil && len(addrs) > 0 {
		return addrs
	}
	// Fallback for plain string arrays ["a@b.com", ...]
	var strs []string
	if err := json.Unmarshal(raw, &strs); err == nil {
		out := make([]unifiedEmailAddress, 0, len(strs))
		for _, s := range strs {
			out = append(out, unifiedEmailAddress{Email: s})
		}
		return out
	}
	return nil
}

// decodeAttachments parses a JSONB array of attachment metadata.
func decodeAttachments(raw []byte) []unifiedAttachment {
	if len(raw) == 0 {
		return nil
	}
	var atts []unifiedAttachment
	if err := json.Unmarshal(raw, &atts); err == nil {
		return atts
	}
	return nil
}

// decodeStringArray parses a JSONB string array (e.g. Outlook categories).
func decodeStringArray(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err == nil {
		return out
	}
	return nil
}
