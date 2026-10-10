-- Preserve provider presentation metadata without widening calendar_events.
-- Several generated queries intentionally select that table's stable shape.
CREATE TABLE IF NOT EXISTS calendar_event_colors (
    event_id UUID PRIMARY KEY REFERENCES calendar_events(id) ON DELETE CASCADE,
    color_id VARCHAR(50),
    color_hex VARCHAR(7)
        CHECK (color_hex IS NULL OR color_hex ~ '^#[0-9A-Fa-f]{6}$'),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
