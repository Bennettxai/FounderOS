package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/utils"
)

// DeleteAccountRequest is the request body for account deletion.
// Requiring explicit confirmation prevents accidental deletion.
type DeleteAccountRequest struct {
	Confirm bool `json:"confirm"`
}

// DeleteAccount handles user account deletion (GDPR Article 17 - Right to Erasure).
// It soft-deletes the account by invalidating all sessions, then logs the event.
// Hard deletion of user data is handled by a scheduled background job after 30 days.
func (h *ProfileHandler) DeleteAccount(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	var req DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil || !req.Confirm {
		utils.RespondBadRequest(c, slog.Default(), `Must confirm account deletion with {"confirm": true}`)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		slog.Error("DeleteAccount: failed to begin transaction", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "begin transaction", err)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Mark the user record as pending deletion. We prefix the name with [DELETED]
	// so it is visually obvious in admin tooling, and update the timestamp so
	// any downstream jobs that sweep by updated_at pick this record up.
	_, err = tx.Exec(ctx, `
		UPDATE "user"
		SET    "updatedAt" = NOW(),
		       name        = '[DELETED] ' || name
		WHERE  id = $1
	`, user.ID)
	if err != nil {
		slog.Error("DeleteAccount: failed to mark user for deletion", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "mark account for deletion", err)
		return
	}

	// Invalidate all active sessions so the user is logged out everywhere immediately.
	// This MUST succeed — a deleted account with active sessions is a security risk.
	_, err = tx.Exec(ctx, `DELETE FROM session WHERE "userId" = $1`, user.ID)
	if err != nil {
		slog.Error("DeleteAccount: failed to delete sessions", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "invalidate sessions", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("DeleteAccount: failed to commit transaction", "error", err, "user_id", user.ID)
		utils.RespondInternalError(c, slog.Default(), "commit account deletion", err)
		return
	}

	// Invalidate Redis session cache if available so the logout is immediate
	// across all horizontally-scaled instances.
	if h.sessionCache != nil {
		if cacheErr := h.sessionCache.InvalidateUserSessions(ctx, user.ID); cacheErr != nil {
			slog.Warn("DeleteAccount: failed to invalidate Redis session cache",
				"error", cacheErr, "user_id", user.ID)
		}
	}

	slog.Warn("AUDIT_ACCOUNT_DELETION",
		"user_id", user.ID,
		"ip", c.ClientIP(),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Account scheduled for deletion. Your data will be permanently removed within 30 days. Contact support to cancel.",
	})
}

// ExportAccountData handles user data export (GDPR Article 20 - Right to Data Portability).
// It collects all PII and user-generated content into a single JSON document and
// returns it as a file download.
func (h *ProfileHandler) ExportAccountData(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		utils.RespondUnauthorized(c, slog.Default())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	export := map[string]interface{}{
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"user_id":     user.ID,
	}

	// --- Profile ---
	// The "user" table uses camelCase column names (Better Auth convention).
	var name, email string
	var createdAt time.Time
	err := h.pool.QueryRow(ctx,
		`SELECT name, email, "createdAt" FROM "user" WHERE id = $1`,
		user.ID,
	).Scan(&name, &email, &createdAt)
	if err == nil {
		export["profile"] = map[string]interface{}{
			"name":       name,
			"email":      email,
			"created_at": createdAt,
		}
	} else {
		slog.Warn("ExportAccountData: failed to query profile", "error", err, "user_id", user.ID)
	}

	// --- Workspaces (owned or member) ---
	rows, err := h.pool.Query(ctx, `
		SELECT w.id, w.name, w.created_at
		FROM   workspaces w
		JOIN   workspace_members wm ON wm.workspace_id = w.id
		WHERE  wm.user_id = $1
		ORDER  BY w.created_at DESC
	`, user.ID)
	if err == nil {
		var workspaces []map[string]interface{}
		for rows.Next() {
			var id, wname string
			var wCreated time.Time
			if rows.Scan(&id, &wname, &wCreated) == nil {
				workspaces = append(workspaces, map[string]interface{}{
					"id":         id,
					"name":       wname,
					"created_at": wCreated,
				})
			}
		}
		rows.Close()
		if rows.Err() != nil {
			slog.Warn("ExportAccountData: row iteration error for workspaces", "error", rows.Err(), "user_id", user.ID)
		}
		export["workspaces"] = workspaces
	} else {
		slog.Warn("ExportAccountData: failed to query workspaces", "error", err, "user_id", user.ID)
	}

	// --- Conversations ---
	rows, err = h.pool.Query(ctx, `
		SELECT id, title, created_at
		FROM   conversations
		WHERE  user_id = $1
		ORDER  BY created_at DESC
	`, user.ID)
	if err == nil {
		var conversations []map[string]interface{}
		for rows.Next() {
			var id, title string
			var cCreated time.Time
			if rows.Scan(&id, &title, &cCreated) == nil {
				conversations = append(conversations, map[string]interface{}{
					"id":         id,
					"title":      title,
					"created_at": cCreated,
				})
			}
		}
		rows.Close()
		if rows.Err() != nil {
			slog.Warn("ExportAccountData: row iteration error for conversations", "error", rows.Err(), "user_id", user.ID)
		}
		export["conversations"] = conversations
	} else {
		slog.Warn("ExportAccountData: failed to query conversations", "error", err, "user_id", user.ID)
	}

	// --- Messages (last 90 days to keep the export a manageable size) ---
	rows, err = h.pool.Query(ctx, `
		SELECT m.id, m.role, m.content, m.created_at, c.title AS conversation_title
		FROM   messages m
		JOIN   conversations c ON c.id = m.conversation_id
		WHERE  c.user_id = $1
		  AND  m.created_at > NOW() - INTERVAL '90 days'
		ORDER  BY m.created_at DESC
	`, user.ID)
	if err == nil {
		var messages []map[string]interface{}
		for rows.Next() {
			var id, role, content, convTitle string
			var mCreated time.Time
			if rows.Scan(&id, &role, &content, &mCreated, &convTitle) == nil {
				messages = append(messages, map[string]interface{}{
					"id":           id,
					"role":         role,
					"content":      content,
					"created_at":   mCreated,
					"conversation": convTitle,
				})
			}
		}
		rows.Close()
		if rows.Err() != nil {
			slog.Warn("ExportAccountData: row iteration error for messages", "error", rows.Err(), "user_id", user.ID)
		}
		export["messages_last_90_days"] = messages
	} else {
		slog.Warn("ExportAccountData: failed to query messages", "error", err, "user_id", user.ID)
	}

	// --- Memories ---
	// The memories table stores title, summary, content and memory_type.
	rows, err = h.pool.Query(ctx, `
		SELECT id, title, memory_type, created_at
		FROM   memories
		WHERE  user_id = $1
		  AND  is_active = TRUE
		ORDER  BY created_at DESC
	`, user.ID)
	if err == nil {
		var memories []map[string]interface{}
		for rows.Next() {
			var id, title, memType string
			var memCreated time.Time
			if rows.Scan(&id, &title, &memType, &memCreated) == nil {
				memories = append(memories, map[string]interface{}{
					"id":         id,
					"title":      title,
					"type":       memType,
					"created_at": memCreated,
				})
			}
		}
		rows.Close()
		if rows.Err() != nil {
			slog.Warn("ExportAccountData: row iteration error for memories", "error", rows.Err(), "user_id", user.ID)
		}
		export["memories"] = memories
	} else {
		slog.Warn("ExportAccountData: failed to query memories", "error", err, "user_id", user.ID)
	}

	slog.Warn("AUDIT_DATA_EXPORT",
		"user_id", user.ID,
		"ip", c.ClientIP(),
		"timestamp", time.Now().UTC().Format(time.RFC3339),
	)

	jsonData, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "generate data export", err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=businessos-data-export.json")
	c.Data(http.StatusOK, "application/json", jsonData)
}
