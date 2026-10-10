-- Migration: Add meeting_recordings table for voice feature
-- This migration adds support for meeting recording with speaker diarization

-- Create meeting_recordings table
CREATE TABLE IF NOT EXISTS meeting_recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    calendar_event_id UUID REFERENCES calendar_events(id) ON DELETE SET NULL,
    duration FLOAT DEFAULT 0,
    audio_path TEXT,
    transcript TEXT,
    segments JSONB DEFAULT '[]'::jsonb,  -- Speaker segments with timestamps
    summary TEXT,
    action_items TEXT[] DEFAULT ARRAY[]::TEXT[],
    participants TEXT[] DEFAULT ARRAY[]::TEXT[],
    status VARCHAR(50) DEFAULT 'recording',  -- recording, processing, completed, failed
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for meeting_recordings
CREATE INDEX IF NOT EXISTS idx_meeting_recordings_user_id ON meeting_recordings(user_id);
CREATE INDEX IF NOT EXISTS idx_meeting_recordings_status ON meeting_recordings(status);
CREATE INDEX IF NOT EXISTS idx_meeting_recordings_calendar_event ON meeting_recordings(calendar_event_id);
CREATE INDEX IF NOT EXISTS idx_meeting_recordings_created_at ON meeting_recordings(created_at DESC);

-- Add columns to voice_notes if they don't exist
DO $$ 
BEGIN
    -- Add title column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'voice_notes' AND column_name = 'title') THEN
        ALTER TABLE voice_notes ADD COLUMN title VARCHAR(500);
    END IF;
    
    -- Add audio_path column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'voice_notes' AND column_name = 'audio_path') THEN
        ALTER TABLE voice_notes ADD COLUMN audio_path TEXT;
    END IF;
    
    -- Add tags column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'voice_notes' AND column_name = 'tags') THEN
        ALTER TABLE voice_notes ADD COLUMN tags TEXT[] DEFAULT ARRAY[]::TEXT[];
    END IF;
    
    -- Add updated_at column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'voice_notes' AND column_name = 'updated_at') THEN
        ALTER TABLE voice_notes ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
    END IF;
END $$;

-- Create voice_commands_log table for analytics
CREATE TABLE IF NOT EXISTS voice_commands_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    raw_text TEXT NOT NULL,
    intent VARCHAR(100),
    confidence FLOAT,
    entities JSONB DEFAULT '{}'::jsonb,
    executed BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_voice_commands_user_id ON voice_commands_log(user_id);
CREATE INDEX IF NOT EXISTS idx_voice_commands_intent ON voice_commands_log(intent);

-- Comment on tables
COMMENT ON TABLE meeting_recordings IS 'Stores meeting recordings with transcription and speaker diarization';
COMMENT ON TABLE voice_commands_log IS 'Logs voice commands for analytics and improvement';
COMMENT ON COLUMN meeting_recordings.segments IS 'JSON array of {speaker, start_time, end_time, text} objects';
