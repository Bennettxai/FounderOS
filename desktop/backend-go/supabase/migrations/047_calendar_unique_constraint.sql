-- Migration 029: Add unique constraint to calendar_events for Google event deduplication
-- Required for ON CONFLICT upsert to work properly during calendar sync

-- Add unique constraint for (user_id, google_event_id) if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'calendar_events_user_id_google_event_id_key'
    ) THEN
        ALTER TABLE calendar_events
        ADD CONSTRAINT calendar_events_user_id_google_event_id_key
        UNIQUE (user_id, google_event_id);
    END IF;
END $$;
