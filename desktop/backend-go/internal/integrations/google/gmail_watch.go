package google

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"google.golang.org/api/gmail/v1"
)

// ============================================================================
// GMAIL PUB/SUB WATCH
// ============================================================================
//
// Lifecycle of a Gmail push subscription:
//
//   1. StartWatch  — call gmail.users.watch(topicName); store the
//                    returned historyId + expiration.
//   2. Notification webhook arrives at /api/integrations/google_gmail/webhook
//      with payload {emailAddress, historyId} (base64 JSON inside Pub/Sub
//      envelope). HandleHistoryEvent does:
//        - users.history.list(startHistoryId=stored_history_id)
//        - For every messageAdded: users.messages.get + saveEmail
//          (which fires OnEmailSaved → engine sync via Wave 1 hook)
//        - Bump stored history_id to the newest historyId returned.
//   3. RenewalLoop runs every 12h, re-issues StartWatch on watches
//      expiring within 24h. Google caps at 7 days.
//   4. StopWatch tears down on disconnect.
//
// Pub/Sub topic + push subscription provisioning lives in Roberto's GCP
// console; we only consume the topic name here.

// WatchTTL is how long Google's gmail.users.watch keeps the watch
// alive before requiring renewal. Google's hard cap is 7 days; we
// renew when <24h remains so transient outages don't strand users.
const WatchTTL = 7 * 24 * time.Hour

// ----------------------------------------------------------------------------
// Service methods (live on GmailService — same struct that owns Sync etc.)
// ----------------------------------------------------------------------------

// StartWatch issues gmail.users.watch and persists the returned baseline
// history id + expiration. Idempotent: re-calling refreshes the row.
//
// topicName is the Cloud Pub/Sub topic (e.g.
// "projects/my-proj/topics/gmail-push"). The user's Gmail account must
// have already granted IAM permission to that topic — that's a console
// step Roberto handles per deployment.
func (s *GmailService) StartWatch(ctx context.Context, userID, topicName string) error {
	if topicName == "" {
		return fmt.Errorf("topic name required")
	}

	srv, err := s.GetGmailAPI(ctx, userID)
	if err != nil {
		return fmt.Errorf("get gmail api: %w", err)
	}

	resp, err := srv.Users.Watch("me", &gmail.WatchRequest{
		TopicName: topicName,
		// LabelFilterBehavior: include INBOX so we hear about incoming.
		// Default is "include" which matches all labels — fine.
		LabelIds: []string{"INBOX"},
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gmail watch: %w", err)
	}

	// Google returns expiration as Unix milliseconds; convert and clamp
	// to our local TTL (we don't trust the server beyond 7 days).
	expiresAt := time.UnixMilli(resp.Expiration)
	if expiresAt.After(time.Now().Add(WatchTTL + time.Hour)) {
		expiresAt = time.Now().Add(WatchTTL)
	}

	_, err = s.provider.Pool().Exec(ctx, `
		INSERT INTO google_gmail_watches (
			user_id, topic_name, history_id, expires_at, last_renewed_at
		) VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			topic_name = EXCLUDED.topic_name,
			history_id = EXCLUDED.history_id,
			expires_at = EXCLUDED.expires_at,
			last_renewed_at = NOW(),
			updated_at = NOW()
	`, userID, topicName, int64(resp.HistoryId), expiresAt)
	if err != nil {
		return fmt.Errorf("save watch: %w", err)
	}

	slog.Info("gmail watch started",
		"user_id", userID, "history_id", resp.HistoryId, "expires_at", expiresAt)
	return nil
}

// StopWatch tears down the user's watch and removes the local row.
// Best-effort — we always delete the local row so a stuck Google-side
// stop doesn't block disconnection.
func (s *GmailService) StopWatch(ctx context.Context, userID string) error {
	srv, err := s.GetGmailAPI(ctx, userID)
	if err == nil {
		_ = srv.Users.Stop("me").Context(ctx).Do()
	}
	_, _ = s.provider.Pool().Exec(ctx, `
		DELETE FROM google_gmail_watches WHERE user_id = $1
	`, userID)
	return nil
}

// HandleHistoryEvent processes a Pub/Sub notification: lists every
// message added since the stored historyId, fetches each, and saves
// it (which fires OnEmailSaved → engine sync). Returns the count of
// new messages persisted.
//
// Notification's historyId is informational; we always list FROM the
// stored value so missed notifications are caught up by the next one.
func (s *GmailService) HandleHistoryEvent(ctx context.Context, userID string) (int, error) {
	var stored int64
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT history_id FROM google_gmail_watches WHERE user_id = $1
	`, userID).Scan(&stored)
	if err != nil {
		return 0, fmt.Errorf("no active watch for user: %w", err)
	}

	srv, err := s.GetGmailAPI(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("get gmail api: %w", err)
	}

	hist, err := srv.Users.History.List("me").
		StartHistoryId(uint64(stored)).
		HistoryTypes("messageAdded").
		Context(ctx).Do()
	if err != nil {
		return 0, fmt.Errorf("history list: %w", err)
	}

	saved := 0
	var newest int64 = stored
	for _, h := range hist.History {
		if int64(h.Id) > newest {
			newest = int64(h.Id)
		}
		for _, ma := range h.MessagesAdded {
			if ma == nil || ma.Message == nil {
				continue
			}
			full, ferr := srv.Users.Messages.Get("me", ma.Message.Id).Format("full").Context(ctx).Do()
			if ferr != nil {
				slog.Info("gmail watch: get message failed", "id", ma.Message.Id, "error", ferr)
				continue
			}
			if serr := s.saveEmail(ctx, userID, full); serr != nil {
				slog.Info("gmail watch: save failed", "id", ma.Message.Id, "error", serr)
				continue
			}
			saved++
		}
	}

	// Bump the stored history_id forward so the next notification
	// processes the diff after this one. Use the higher of newest seen
	// or hist.HistoryId returned by the API (the latter is the new
	// watermark even if no events were in the window).
	if int64(hist.HistoryId) > newest {
		newest = int64(hist.HistoryId)
	}
	if newest > stored {
		_, _ = s.provider.Pool().Exec(ctx, `
			UPDATE google_gmail_watches
			SET history_id = $1, updated_at = NOW()
			WHERE user_id = $2
		`, newest, userID)
	}

	return saved, nil
}

// LookupUserByEmail finds the BO user_id whose Gmail account matches
// the given email. Used by the webhook handler: notifications carry
// {emailAddress} but no BO identity.
func (s *GmailService) LookupUserByEmail(ctx context.Context, email string) (string, error) {
	var userID string
	err := s.provider.Pool().QueryRow(ctx, `
		SELECT user_id FROM google_oauth_tokens WHERE google_email = $1
	`, email).Scan(&userID)
	return userID, err
}

// ----------------------------------------------------------------------------
// Renewal loop
// ----------------------------------------------------------------------------

// WatchRow is the persisted view the renewer reads.
type WatchRow struct {
	UserID    string
	TopicName string
	ExpiresAt time.Time
}

// ListExpiringWatches returns watches whose expires_at is within
// `within`. Used by RenewalLoop.
func (s *GmailService) ListExpiringWatches(ctx context.Context, within time.Duration) ([]WatchRow, error) {
	rows, err := s.provider.Pool().Query(ctx, `
		SELECT user_id, topic_name, expires_at
		FROM google_gmail_watches
		WHERE expires_at <= NOW() + $1::interval
		ORDER BY expires_at
	`, strconv.Itoa(int(within.Seconds()))+" seconds")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WatchRow
	for rows.Next() {
		var w WatchRow
		if err := rows.Scan(&w.UserID, &w.TopicName, &w.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// WatchRenewalLoop runs forever. Hourly tick — re-issues watch on any
// row expiring within 24h. Goroutine-safe; honors ctx for shutdown.
func (s *GmailService) WatchRenewalLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	s.renewExpiringWatches(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.renewExpiringWatches(ctx)
		}
	}
}

func (s *GmailService) renewExpiringWatches(ctx context.Context) {
	expiring, err := s.ListExpiringWatches(ctx, 24*time.Hour)
	if err != nil {
		slog.Info("gmail watches: list expiring failed", "error", err)
		return
	}
	for _, w := range expiring {
		// Re-issue StartWatch — same code path that creates/refreshes
		// the watch from scratch. Idempotent on the row.
		if err := s.StartWatch(ctx, w.UserID, w.TopicName); err != nil {
			slog.Info("gmail watches: renew failed",
				"user_id", w.UserID, "error", err)
		}
	}
	if len(expiring) > 0 {
		slog.Info("gmail watches: renewal pass complete", "count", len(expiring))
	}
}
