-- Migration 109: Gmail Pub/Sub watch tracking.
--
-- Stores per-user Gmail watch state (the equivalent of a Graph
-- subscription, but using Google's Cloud Pub/Sub push model). Required
-- by the renewal scheduler and the webhook receiver:
--
--   - history_id pins the baseline for incremental fetches via
--     gmail.users.history.list. Every notification carries a new
--     historyId; we list the diff since the stored value, then bump.
--   - expires_at drives the renewal scheduler. Google caps watches at
--     7 days; we ask for the max and renew when <24h remains.
--   - topic_name records which Cloud Pub/Sub topic Google publishes to,
--     so swapping topics doesn't strand orphan watches.
--
-- Webhook receiver: internal/handlers/comms_webhooks_gmail.go
-- Service:          internal/integrations/google/gmail_watch.go

BEGIN;

CREATE TABLE IF NOT EXISTS google_gmail_watches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL UNIQUE,

    -- Google Pub/Sub config
    topic_name VARCHAR(500) NOT NULL,

    -- Pinned by every successful history.list call
    history_id BIGINT NOT NULL,

    -- Lifecycle (Google caps at 7d; we renew when <24h)
    expires_at TIMESTAMPTZ NOT NULL,
    last_renewed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gmail_watches_expiring
    ON google_gmail_watches(expires_at);

COMMIT;
