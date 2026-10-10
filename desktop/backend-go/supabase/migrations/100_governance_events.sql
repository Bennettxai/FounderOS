-- Governance Events audit trail
-- Beer's VSM algedonic channel and governance event logging

CREATE TABLE IF NOT EXISTS governance_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL DEFAULT 'algedonic',   -- 'algedonic', 'setpoint_adjustment', 'policy_change'
    source TEXT NOT NULL,                            -- Component that fired the event
    severity TEXT NOT NULL DEFAULT 'INFO',           -- CRITICAL, HIGH, MEDIUM, LOW, INFO
    description TEXT NOT NULL DEFAULT '',             -- Human-readable description
    metadata JSONB DEFAULT '{}',                     -- Event-specific data
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_governance_events_type ON governance_events (event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_governance_events_severity ON governance_events (severity, created_at DESC);

COMMENT ON TABLE governance_events IS 'Audit trail for governance decisions, algedonic signals, and policy changes';
