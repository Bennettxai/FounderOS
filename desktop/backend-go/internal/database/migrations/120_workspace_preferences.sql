-- Workspace preferences: the business-level settings primitive.
-- These are fundamental settings every module reads (time zone, date/time
-- format, week start, working hours), NOT calendar-specific. Calendar-specific
-- config (default view, notifications, member colors) lives in the `calendar`
-- JSONB so the calendar module can configure a feature without a new table.
CREATE TABLE IF NOT EXISTS workspace_preferences (
    workspace_id          UUID PRIMARY KEY,
    timezone              TEXT     NOT NULL DEFAULT 'America/New_York',
    date_format           TEXT     NOT NULL DEFAULT 'MM/DD/YYYY',
    time_format           TEXT     NOT NULL DEFAULT '12h',   -- 12h | 24h
    week_start            SMALLINT NOT NULL DEFAULT 0,        -- 0=Sunday, 1=Monday
    working_hours_start   SMALLINT NOT NULL DEFAULT 9,
    working_hours_end     SMALLINT NOT NULL DEFAULT 17,
    default_event_minutes SMALLINT NOT NULL DEFAULT 60,
    language              TEXT     NOT NULL DEFAULT 'en-US',
    calendar              JSONB    NOT NULL DEFAULT '{}'::jsonb,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
