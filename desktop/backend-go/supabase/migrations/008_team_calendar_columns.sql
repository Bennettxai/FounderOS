-- Migration: Add share_calendar and calendar_user_id columns to team_members
-- These columns are required for calendar sharing functionality

-- Add share_calendar column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'team_members' AND column_name = 'share_calendar'
    ) THEN
        ALTER TABLE team_members ADD COLUMN share_calendar BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

-- Add calendar_user_id column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'team_members' AND column_name = 'calendar_user_id'
    ) THEN
        ALTER TABLE team_members ADD COLUMN calendar_user_id VARCHAR(255);
    END IF;
END $$;
