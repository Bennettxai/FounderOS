-- Migration 108: Microsoft Graph subscription tracking.
--
-- Stores the active Graph subscriptions per user so we know what to
-- renew before they expire (Graph caps message subscriptions at ~71
-- hours; the renewal scheduler runs every hour and refreshes anything
-- expiring within 2h).
--
-- See docs/COMMUNICATIONS_ENGINE_SYNC.md and Wave 3 chunk 3.
-- Webhook receiver: internal/handlers/comms_webhooks_microsoft.go
-- Service:          internal/integrations/microsoft/subscriptions.go

BEGIN;

CREATE TABLE IF NOT EXISTS microsoft_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,

    -- Graph identifiers
    subscription_id VARCHAR(255) NOT NULL,
    resource VARCHAR(500) NOT NULL,
    -- "mail" | "teams_messages" — discriminator picked up by the webhook
    -- dispatcher to decide whether this subscription's notifications go
    -- through OutlookService or TeamsService. Kept denormalized off the
    -- resource string so the dispatcher doesn't re-parse paths on every
    -- notification.
    resource_kind VARCHAR(50) NOT NULL,
    change_type VARCHAR(100),

    -- Webhook config returned to Graph at create time
    notification_url TEXT,
    -- Random secret echoed back by Graph in every notification.
    -- We compare incoming notifications' clientState against this to
    -- prove the notification really came from a subscription we own.
    client_state VARCHAR(255),

    -- Lifecycle
    expires_at TIMESTAMPTZ,
    last_renewed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(user_id, subscription_id)
);

CREATE INDEX IF NOT EXISTS idx_ms_subs_user ON microsoft_subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_ms_subs_expiring
    ON microsoft_subscriptions(expires_at)
    WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ms_subs_subscription
    ON microsoft_subscriptions(subscription_id);

COMMIT;
