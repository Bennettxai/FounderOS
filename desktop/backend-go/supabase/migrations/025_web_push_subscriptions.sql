-- Web Push subscriptions for browser notifications (Web Push API)
-- This is separate from push_devices which is for mobile (FCM/APNs)
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL UNIQUE,  -- Web Push endpoint URL
    p256dh TEXT NOT NULL,           -- Public key for encryption
    auth TEXT NOT NULL,             -- Auth secret
    user_agent TEXT,                -- Browser/device info
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_push_subscriptions_user ON push_subscriptions(user_id);
CREATE INDEX idx_push_subscriptions_endpoint ON push_subscriptions(endpoint);
