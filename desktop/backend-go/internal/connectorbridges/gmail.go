// Package services — gmail_connector.go
//
// Adapts the existing google.GmailService to the
// adapters.GmailFetcher interface so the Phase 7 connector adapter can run
// real syncs against the user's mailbox without any new OAuth code path.
//
// The connector adapter doesn't import this package directly; bootstrap
// wires the fetcher in via adapters.SetGmailFetcher.
package connectorbridges

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rhl/businessos-backend/internal/integrations/google"
	gmailapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GmailFetcher implements adapters.GmailFetcher on top of the existing
// google.GmailService. The ID supplied to ListIDs / Get is the BusinessOS
// user.id — the same key google.Provider.GetTokenSource reads from
// google_oauth_tokens. We deliberately do not look up the actual email
// address here; the Gmail API accepts the literal "me" user-id when
// authenticated via OAuth, and that's what we use.
type GmailFetcher struct {
	svc *google.GmailService
}

// NewGmailFetcher returns a fetcher backed by the given service.
// Callers (bootstrap) build the service from a Provider already wired to
// the OAuth config.
func NewGmailFetcher(svc *google.GmailService) *GmailFetcher {
	return &GmailFetcher{svc: svc}
}

// ListIDs returns up to `max` message ids in the user's inbox+sent label
// set, plus the next-page token for incremental sync. An empty cursor means
// "start from newest"; an empty next means "caught up".
//
// Filters: `in:inbox OR in:sent` matches the existing SyncEmails query so
// the two paths surface the same superset of messages.
func (f *GmailFetcher) ListIDs(ctx context.Context, userID, cursor string, max int) ([]string, string, error) {
	api, err := f.api(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	req := api.Users.Messages.List("me").MaxResults(int64(max)).Q("in:inbox OR in:sent")
	if cursor != "" {
		req = req.PageToken(cursor)
	}
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, "", fmt.Errorf("gmail list: %w", err)
	}
	ids := make([]string, len(resp.Messages))
	for i, m := range resp.Messages {
		ids[i] = m.Id
	}
	return ids, resp.NextPageToken, nil
}

// Get pulls one full message and converts it to the loose map shape the
// adapter's Transform expects. We round-trip via JSON because the gmail-go
// client's JSON tags happen to match the field names Transform reads
// (id, payload.headers[].name/value, snippet, internalDate) — saves a
// hand-written mapper that would have to track every Gmail schema change.
func (f *GmailFetcher) Get(ctx context.Context, userID, msgID string) (map[string]any, error) {
	api, err := f.api(ctx, userID)
	if err != nil {
		return nil, err
	}
	msg, err := api.Users.Messages.Get("me", msgID).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("gmail get %s: %w", msgID, err)
	}
	encoded, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("gmail marshal %s: %w", msgID, err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil, fmt.Errorf("gmail unmarshal %s: %w", msgID, err)
	}
	return out, nil
}

// api builds a Gmail API client from the user's OAuth token source. We
// don't go through GmailService.GetGmailAPI to avoid a circular service
// dependency at construction time — we rely on the same TokenSource.
func (f *GmailFetcher) api(ctx context.Context, userID string) (*gmailapi.Service, error) {
	if f.svc == nil {
		return nil, fmt.Errorf("gmail fetcher: service not wired")
	}
	return f.svc.GetGmailAPI(ctx, userID)
}

// Compile-time assertion that GmailFetcher matches the adapter's
// expected method set. We can't import adapters.GmailFetcher here without
// pulling the adapter package into services (which is fine, no cycle), so
// the assertion lives in the bootstrap call site instead. Keeping a clean
// signature comment here as documentation of the contract:
//
//	ListIDs(ctx, userID, cursor string, max int) ([]string, string, error)
//	Get(ctx, userID, msgID string) (map[string]any, error)
//
// See internal/optimal/connectors/adapters/gmail.go GmailFetcher.

// Discard the unused option import warning. We may need it for finer-grained
// HTTP client control (timeouts, retries) once production traffic shapes show.
var _ = option.WithTokenSource
