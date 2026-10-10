-- Fix user_id type mismatch in thinking system tables
-- The auth system uses TEXT user IDs (like "ZVtQRaictVbO9lN0p-csSA"), not UUIDs

-- Fix thinking_traces table
ALTER TABLE thinking_traces
ALTER COLUMN user_id TYPE TEXT USING user_id::TEXT;

-- Make message_id nullable since thinking can happen before message is created
ALTER TABLE thinking_traces
ALTER COLUMN message_id DROP NOT NULL;

-- Fix reasoning_templates table
ALTER TABLE reasoning_templates
ALTER COLUMN user_id TYPE TEXT USING user_id::TEXT;
